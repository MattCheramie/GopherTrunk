package composer

import (
	"encoding/hex"

	"github.com/MattCheramie/GopherTrunk/internal/radio/p25"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
)

// p25ADPState is one P25 Phase 1 voice call's in-process ADP (RC4, ALGID
// 0xAA) decryption state (issue #1187): the Encryption Sync the LDU2s
// announce, the operator key resolved for its key ID, and the Message
// Indicator schedule across superframes. It runs on the chain's own
// goroutine (the receiver's LDU sink) — no locking.
//
// Two things make it more than "XOR the keystream":
//
//   - The ES in an LDU2 names the MI of the NEXT superframe, and successive
//     MIs follow the 64-bit LFSR (phase1.AdvanceMI / RewindMI, capture-
//     pinned). So on the FIRST ES of a call the current superframe's MI is
//     RewindMI(es.MI) — the HDU that would have carried it is not delivered
//     by the LDU assembler — and an LDU2 whose ES fails RS still advances
//     the chain by prediction.
//   - The first superframe's LDU1 arrives BEFORE the ES that lets its MI be
//     known, so with a key resolver configured the state holds the first
//     superframe's frames (at most one LDU1 + one LDU2, 360 ms) until that
//     first ES decodes, then descrambles and flushes them in order. A clear
//     call, or an ES that never decodes, flushes them untouched at the same
//     point; nothing is held past the first LDU2 or the end of the call.
//
// It never guesses a key (no key ⇒ frames stay ciphertext, exactly as
// before), never touches the cryptolab capture (which re-extracts the
// ciphertext from the LDU bits), and counts rather than fails on errors.
type p25ADPState struct {
	c      *Composer
	serial string
	system string

	haveES bool
	algID  uint8
	keyID  uint16
	key    []byte
	keyFor uint16 // key ID the key was resolved for
	keyTry bool   // a resolution was attempted for keyFor

	cur      [9]byte // MI of the superframe in progress
	haveCur  bool
	next     [9]byte // MI the last ES announced for the following superframe
	haveNext bool
	ks       []byte // keystream for cur
	ksMI     [9]byte
	haveKS   bool

	pending  []p25HeldFrame
	holding  bool // still before the first ES: frames are held, not written
	ldu2Seen bool

	// counters for the end-of-call summary
	descrambled, held, flushedHeld int
	noKey, noMI, unsupported       int
	rewound, predicted, mismatch   int
}

// p25HeldFrame is one voice frame held until the first ES decodes.
type p25HeldFrame struct {
	duid     phase1.DUID
	subframe int
	frame    []byte
	errs     int
}

// p25FrameWriter hands one (possibly descrambled) 11-byte IMBE frame and
// its FEC corrected-bit count to the recorder.
type p25FrameWriter func(frame []byte, errs int)

func newP25ADPState(c *Composer, serial, system string) *p25ADPState {
	return &p25ADPState{c: c, serial: serial, system: system, holding: true}
}

// onES is called for every LDU2 BEFORE its voice frames are handled, with
// that LDU2's Encryption Sync when it decoded (ok) — RS-uncorrectable or
// missing blocks arrive as ok=false. It establishes the current superframe's
// MI on the first decode, records the next one, and flushes any held
// first-superframe frames now that their fate is known.
func (s *p25ADPState) onES(es phase1.EncryptionSync, ok bool, write p25FrameWriter) {
	s.ldu2Seen = true
	if ok && es.Encrypted() && p25.AlgorithmKnown(es.AlgorithmID) {
		if !s.haveES || es.AlgorithmID != s.algID || es.KeyID != s.keyID {
			s.haveES, s.algID, s.keyID = true, es.AlgorithmID, es.KeyID
			s.resolveKey()
		}
		if s.algID == p25.AlgorithmADP {
			if !s.haveCur {
				s.cur, s.haveCur = phase1.RewindMI(es.MessageIndicator), true
				s.rewound++
			} else if s.haveNext && s.next != es.MessageIndicator {
				// The previous ES (or the LFSR prediction) disagreed with
				// this one; the ES on air wins for the next superframe.
				s.mismatch++
			}
			s.next, s.haveNext = es.MessageIndicator, true
		}
	}
	s.flushHeld(write)
}

// resolveKey looks the current ES's key up once per (algorithm, key ID).
func (s *p25ADPState) resolveKey() {
	s.key = nil
	if s.algID != p25.AlgorithmADP {
		s.unsupported++
		s.c.log.Debug("composer: p25p1 encryption algorithm not decrypted in-process",
			"serial", s.serial, "system", s.system, "alg", p25.FormatAlgorithm(s.algID), "key_id", s.keyID)
		return
	}
	if s.keyTry && s.keyFor == s.keyID {
		return
	}
	s.keyTry, s.keyFor = true, s.keyID
	if s.c.keyResolver == nil {
		return
	}
	key, found := s.c.keyResolver(s.system, "rc4", s.keyID)
	if !found {
		s.c.log.Debug("composer: p25p1 adp call has no configured key",
			"serial", s.serial, "system", s.system, "key_id", s.keyID)
		return
	}
	if len(key) != phase1.ADPKeyBytes {
		s.c.log.Warn("composer: p25p1 adp key has the wrong length — not applied",
			"serial", s.serial, "system", s.system, "key_id", s.keyID, "bytes", len(key), "want", phase1.ADPKeyBytes)
		return
	}
	s.key = key
	s.c.log.Info("composer: p25p1 adp key resolved — decrypting in-process",
		"serial", s.serial, "system", s.system, "key_id", s.keyID)
}

// frames handles one voice LDU's FEC-decoded frames: descrambles them in
// place when the superframe's MI and a key are known, holds them while the
// first ES is still outstanding, and otherwise writes them through as they
// are (ciphertext, or clear voice on an unencrypted call).
func (s *p25ADPState) frames(duid phase1.DUID, fs *[phase1.LDUVoiceSubframeCount][]byte, errs [phase1.LDUVoiceSubframeCount]int, write p25FrameWriter) {
	if s.holding {
		for i, f := range fs {
			if f == nil {
				continue
			}
			s.pending = append(s.pending, p25HeldFrame{duid: duid, subframe: i, frame: f, errs: errs[i]})
			s.held++
		}
		// Never hold more than one superframe: an LDU2 that never decodes
		// an ES flushes via onES, but guard the LDU1-only case too.
		if len(s.pending) > 2*phase1.LDUVoiceSubframeCount {
			s.flushHeld(write)
		}
		return
	}
	s.descramble(duid, fs)
	for i, f := range fs {
		if f == nil {
			continue
		}
		write(f, errs[i])
	}
}

// descramble applies the current superframe's keystream to fs in place
// when there is one to apply; counts the reason when there is not.
func (s *p25ADPState) descramble(duid phase1.DUID, fs *[phase1.LDUVoiceSubframeCount][]byte) {
	if !s.haveES || s.algID != p25.AlgorithmADP {
		return
	}
	if s.key == nil {
		s.noKey++
		return
	}
	if !s.haveCur {
		s.noMI++
		return
	}
	if !s.haveKS || s.ksMI != s.cur {
		ks, err := phase1.ADPSuperframeKeystream(s.key, s.cur)
		if err != nil {
			s.c.log.Debug("composer: p25p1 adp keystream failed", "serial", s.serial, "err", err)
			return
		}
		s.ks, s.ksMI, s.haveKS = ks, s.cur, true
	}
	n, err := phase1.ADPDescrambleVoiceFrames(s.ks, duid, fs)
	if err != nil {
		s.c.log.Debug("composer: p25p1 adp descramble failed", "serial", s.serial, "err", err)
		return
	}
	s.descrambled += n
}

// endLDU2 is called after an LDU2's frames are handled: the superframe is
// over, so the chain moves to the MI its ES announced — or, when that ES
// did not decode, to the LFSR prediction.
func (s *p25ADPState) endLDU2() {
	if !s.haveCur {
		return
	}
	if s.haveNext {
		s.cur = s.next
	} else {
		s.cur = phase1.AdvanceMI(s.cur)
		s.predicted++
	}
	s.haveNext = false
}

// flushHeld writes the frames held before the first ES, descrambling each
// LDU's worth with the now-known MI when there is a key. Idempotent.
func (s *p25ADPState) flushHeld(write p25FrameWriter) {
	if !s.holding {
		return
	}
	s.holding = false
	// Group the held frames back into per-LDU arrays so the keystream
	// offsets line up, then write in arrival order.
	var (
		group [phase1.LDUVoiceSubframeCount][]byte
		gerrs [phase1.LDUVoiceSubframeCount]int
		gduid phase1.DUID
		open  bool
	)
	emit := func() {
		if !open {
			return
		}
		s.descramble(gduid, &group)
		for i, f := range group {
			if f == nil {
				continue
			}
			write(f, gerrs[i])
			s.flushedHeld++
		}
		group = [phase1.LDUVoiceSubframeCount][]byte{}
		gerrs = [phase1.LDUVoiceSubframeCount]int{}
		open = false
	}
	lastSub := -1
	for _, h := range s.pending {
		if open && (h.duid != gduid || h.subframe <= lastSub) {
			emit()
		}
		if !open {
			gduid, open = h.duid, true
		}
		group[h.subframe], gerrs[h.subframe] = h.frame, h.errs
		lastSub = h.subframe
	}
	emit()
	s.pending = nil
}

// finish flushes anything still held (a call that ended before its first
// LDU2) and logs the call's summary. Info when something was decrypted,
// Debug otherwise.
func (s *p25ADPState) finish(write p25FrameWriter) {
	s.flushHeld(write)
	if !s.haveES {
		return
	}
	logf := s.c.log.Debug
	if s.descrambled > 0 {
		logf = s.c.log.Info
	}
	logf("composer: p25p1 adp",
		"serial", s.serial, "system", s.system,
		"alg", p25.FormatAlgorithm(s.algID), "key_id", s.keyID, "key", s.key != nil,
		"descrambled_frames", s.descrambled, "held_frames", s.held, "flushed_held", s.flushedHeld,
		"no_key", s.noKey, "no_mi", s.noMI, "unsupported", s.unsupported,
		"mi_rewound", s.rewound, "mi_predicted", s.predicted, "mi_mismatches", s.mismatch,
		"last_mi", hex.EncodeToString(s.cur[:]))
}
