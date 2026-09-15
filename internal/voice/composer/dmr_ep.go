package composer

import (
	"encoding/hex"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/MattCheramie/GopherTrunk/internal/voice/cryptocap"
)

// KeyResolver returns the raw key bytes an operator configured for a
// system's (algorithm, key ID), or false when none is configured. The daemon
// builds one from trunking.systems[].encryption_keys.
type KeyResolver func(system, algorithm string, keyID uint16) ([]byte, bool)

// dmrEPState is one DMR voice call's Enhanced Privacy state (issue #1187):
// what the Privacy Indicator header announced, the key resolved for it, and
// the Message Indicator chain across superframes. It is driven from the
// chain's dibit sink (onPIHeader, noteLC) and per decoded superframe
// (superframe), on the chain's own goroutine — no locking.
//
// Three things it never does: guess a key (no key ⇒ frames stay ciphertext
// and the .raw sidecar is what it always was), act on an embedded IV of a
// call with no other evidence of encryption unless it decoded perfectly
// clean (dmrvoice.ExtractEmbeddedIV's ~1% false-verify rate on vocoder
// bits), or disturb the voice path on any error (counted, logged at debug).
type dmrEPState struct {
	c       *Composer
	serial  string
	system  string
	groupID uint32

	encrypted  bool // grant flag, LC service option, PI header or clean embedded IV
	haveHeader bool
	header     dmr.PIHeader
	key        []byte
	tracker    dmrvoice.EPTracker

	// Instruments for the decode-quality log / the next field report.
	headers        int
	superframes    int
	ivVerified     int
	descrambled    int
	silence        int
	captured       int
	captureSkipped int
	noMI           int
	noKey          int
	loggedEmbedded bool
	loggedNoKey    bool
}

func (c *Composer) newDMREPState(serial, system string, groupID uint32, grantEncrypted bool) *dmrEPState {
	return &dmrEPState{c: c, serial: serial, system: system, groupID: groupID, encrypted: grantEncrypted}
}

// noteLC folds the embedded Full LC's privacy service option in: it is the
// only encryption signal a Tier III grant-less call carries before its PI
// header decodes.
func (s *dmrEPState) noteLC(encrypted bool) {
	if encrypted {
		s.encrypted = true
	}
}

// onPIHeader records a decoded Privacy Indicator header: publishes the
// algorithm / key ID / MI as a call.encryption event (the engine and the
// recorder backfill the call), seeds the MI chain, and resolves the key.
func (s *dmrEPState) onPIHeader(h dmr.PIHeader) {
	s.headers++
	s.encrypted = true
	changed := !s.haveHeader || h.AlgID != s.header.AlgID || h.FID != s.header.FID || h.KeyID != s.header.KeyID
	s.header, s.haveHeader = h, true
	// A radio repeats the PI header at keyup; each copy names the MI of the
	// superframe that follows, so re-seeding is always right.
	s.tracker.SetHeaderMI(h.MI32())
	if !changed {
		s.c.log.Debug("composer: dmr PI header repeated",
			"serial", s.serial, "mi", hex.EncodeToString(h.MI[:]))
		return
	}
	s.key = nil
	if s.c.keyResolver != nil && h.IsRC4() {
		if k, ok := s.c.keyResolver(s.system, "rc4", uint16(h.KeyID)); ok && len(k) > 0 {
			s.key = k
		}
	}
	s.c.log.Info("composer: dmr PI header — call is encrypted",
		"system", s.system, "serial", s.serial, "group", s.groupID,
		"algorithm", h.AlgName(), "alg_id", h.AlgID, "fid", h.FID, "key_id", h.KeyID,
		"mi", hex.EncodeToString(h.MI[:]), "dst", h.DstAddr,
		"key_configured", s.key != nil, "raw", hex.EncodeToString(h.Raw[:]))
	if s.key == nil && !s.loggedNoKey {
		s.loggedNoKey = true
		if h.IsRC4() {
			s.c.log.Info("composer: dmr enhanced privacy — no encryption_keys entry for this key id; recording ciphertext (docs/dmr-encryption.md)",
				"system", s.system, "serial", s.serial, "key_id", h.KeyID)
		} else {
			s.c.log.Info("composer: dmr privacy algorithm has no in-process decoder; recording ciphertext",
				"system", s.system, "serial", s.serial, "algorithm", h.AlgName())
		}
	}
	if s.c.bus == nil {
		return
	}
	var mi [9]byte
	copy(mi[:], h.MI[:])
	s.c.bus.Publish(events.Event{
		Kind: events.KindCallEncryption,
		Payload: trunking.CallEncryption{
			DeviceSerial:     s.serial,
			Protocol:         "dmr",
			GroupID:          s.groupID,
			AlgorithmID:      h.AlgID,
			KeyID:            uint16(h.KeyID),
			MessageIndicator: mi,
			At:               time.Now(),
		},
	})
}

// superframe runs once per decoded voice superframe with its FEC-decoded
// 49-bit payloads (nil where a frame failed FEC). It resolves the
// superframe's Message Indicator, captures the ciphertext for the cryptolab
// bridge when one is configured, and descrambles IN PLACE when a key is
// known — so the frames the caller then hands the recorder are clear.
func (s *dmrEPState) superframe(sf dmrvoice.VoiceSuperframe, infos [][]byte) {
	iv, corrected, ivOK := dmrvoice.ExtractEmbeddedIV(sf.Frames)
	if !s.encrypted {
		// No header, no flag: only a perfectly clean embedded IV (three exact
		// Golay codewords + CRC-4, ~2^-40 by chance) may declare the call
		// encrypted on its own — that is the late-entry case.
		if !ivOK || corrected != 0 {
			return
		}
		s.encrypted = true
	}
	s.superframes++
	if ivOK {
		s.ivVerified++
		if !s.haveHeader && !s.loggedEmbedded {
			s.loggedEmbedded = true
			s.c.log.Info("composer: dmr enhanced privacy — Message Indicator recovered from the voice superframe (late entry, no PI header seen)",
				"system", s.system, "serial", s.serial, "group", s.groupID,
				"mi", hexMI(iv), "key_configured", s.key != nil)
		}
	}
	mi, ok := s.tracker.Next(iv, ivOK, corrected)
	if !ok {
		s.noMI++
		return
	}
	if s.c.cryptoSink != nil {
		s.capture(mi, infos)
	}
	if s.key == nil {
		s.noKey++
		return
	}
	n, sil, err := dmrvoice.DescrambleSuperframe(s.key, mi, infos)
	if err != nil {
		s.c.log.Debug("composer: dmr enhanced privacy descramble failed", "serial", s.serial, "err", err)
		return
	}
	s.descrambled += n
	s.silence += sil
}

// capture hands the superframe's ciphertext (the 18 FEC-decoded frames,
// packed 7 bytes each, as they are on air — silence frames in clear) and
// its MI to the cryptolab crypto-frame bridge. A superframe with a frame
// that failed FEC is skipped: a placeholder would put wrong bytes at a
// keystream position the offline analysis trusts.
func (s *dmrEPState) capture(mi uint32, infos [][]byte) {
	ct := make([]byte, 0, dmrvoice.EPSuperframeBytes)
	for _, info := range infos {
		if info == nil {
			s.captureSkipped++
			return
		}
		ct = append(ct, packBits(info)...)
	}
	s.captured++
	s.c.cryptoSink.WriteCryptoFrame(cryptocap.Frame{
		System:   s.system,
		Protocol: "dmr",
		Serial:   s.serial,
		TG:       s.groupID,
		AlgID:    s.header.AlgID,
		KeyID:    uint16(s.header.KeyID),
		MI:       []byte{byte(mi >> 24), byte(mi >> 16), byte(mi >> 8), byte(mi)},
		CT:       ct,
		At:       time.Now(),
	})
}

// logQuality appends the Enhanced Privacy counters to the chain's rolling
// decode-quality log; silent for a clear call.
func (s *dmrEPState) logQuality(final bool) {
	if !s.encrypted {
		return
	}
	s.c.log.Debug("composer: dmr enhanced privacy",
		"serial", s.serial, "final", final,
		"pi_headers", s.headers, "key_configured", s.key != nil,
		"superframes", s.superframes, "iv_verified", s.ivVerified,
		"mi_predicted", s.tracker.Predicted, "mi_mismatches", s.tracker.Mismatches,
		"descrambled_frames", s.descrambled, "silence_frames", s.silence,
		"no_mi", s.noMI, "no_key", s.noKey,
		"captured", s.captured, "capture_skipped", s.captureSkipped)
}

func hexMI(mi uint32) string {
	return hex.EncodeToString([]byte{byte(mi >> 24), byte(mi >> 16), byte(mi >> 8), byte(mi)})
}
