package composer

import (
	"fmt"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// ConventionalKeySystem is the pseudo system name a conventional scan-list
// channel's encryption_keys are registered under in the KeyResolver (the
// daemon builds one per channel frequency), so a scanner call — whose grant
// names the scanner, not a trunking system — still resolves the channel's
// keys. Issue #1261.
func ConventionalKeySystem(frequencyHz uint32) string {
	return fmt.Sprintf("conventional:%d", frequencyHz)
}

// resolveKey looks a (algorithm, key ID) up for a call: the grant's system
// first, then the conventional channel the call is on. groups are the call's
// talkgroup candidates in preference order (the grant's group, then the one
// the call's own signalling names); each is tried before the unscoped entry
// for the key ID, so a talkgroup-scoped key wins (issue #1261). A zero group
// is skipped.
func (c *Composer) resolveKey(system string, frequencyHz uint32, algorithm string, keyID uint16, groups ...uint32) ([]byte, bool) {
	if c.keyResolver == nil {
		return nil, false
	}
	systems := []string{system}
	if frequencyHz != 0 {
		systems = append(systems, ConventionalKeySystem(frequencyHz))
	}
	for _, sys := range systems {
		for _, g := range groups {
			if g == 0 {
				continue
			}
			if k, ok := c.keyResolver(sys, algorithm, keyID, g); ok && len(k) > 0 {
				return k, true
			}
		}
		if k, ok := c.keyResolver(sys, algorithm, keyID, 0); ok && len(k) > 0 {
			return k, true
		}
	}
	return nil, false
}

// nxdnCipherState is one NXDN voice call's cipher bookkeeping (issue
// #1261): it owns the nxdn.VoiceCipherTracker that follows the SACCH
// superframe, and on each decoded VCALL logs the call's parties and cipher,
// publishes call.encryption for a scrambled call, and resolves the
// operator's "scrambler" key for the VCALL's key ID. Driven from the
// traffic channel's frame callback, on the chain's goroutine — no locking.
type nxdnCipherState struct {
	c           *Composer
	serial      string
	system      string
	frequencyHz uint32
	groupID     uint32
	tracker     nxdn.VoiceCipherTracker

	keyConfigured bool
	loggedNoKey   bool
}

func (c *Composer) newNXDNCipherState(serial, system string, frequencyHz uint32, groupID uint32) *nxdnCipherState {
	s := &nxdnCipherState{c: c, serial: serial, system: system, frequencyHz: frequencyHz, groupID: groupID}
	// Voice is held until the first VCALL names the cipher only when the
	// operator holds keys at all; with none the path is byte-identical to a
	// chain without the tracker (the VCALL is still decoded for the log and
	// the call.encryption event).
	s.tracker.Hold = c.keyResolver != nil
	s.tracker.OnVCALL = s.onVCALL
	return s
}

// onVCALL runs on the first VCALL superframe of the call and on any change.
func (s *nxdnCipherState) onVCALL(v nxdn.VCALLMessage) {
	s.tracker.ClearKey()
	s.keyConfigured = false
	if v.Cipher == nxdn.CipherScrambler {
		// The grant's group first (a trunked call's talkgroup), then the
		// VCALL's destination — on a conventional channel the grant's group
		// is synthetic and the VCALL names the real talkgroup.
		if k, ok := s.c.resolveKey(s.system, s.frequencyHz, "scrambler", uint16(v.KeyID), s.groupID, uint32(v.Destination)); ok && len(k) == 2 {
			s.tracker.SetKey(uint16(k[0])<<8 | uint16(k[1]))
			s.keyConfigured = true
		}
	}
	if !v.Encrypted() {
		s.c.log.Debug("composer: nxdn VCALL", "serial", s.serial, "system", s.system,
			"src", v.Source, "dst", v.Destination, "call_type", v.CallType, "cipher", v.Cipher.String())
		return
	}
	s.c.log.Info("composer: nxdn VCALL — call is encrypted",
		"system", s.system, "serial", s.serial, "group", s.groupID,
		"src", v.Source, "dst", v.Destination, "call_type", v.CallType,
		"cipher", v.Cipher.String(), "cipher_type", uint8(v.Cipher), "key_id", v.KeyID,
		"key_configured", s.keyConfigured)
	if !s.keyConfigured && !s.loggedNoKey {
		s.loggedNoKey = true
		switch v.Cipher {
		case nxdn.CipherScrambler:
			s.c.log.Info("composer: nxdn scrambler — no encryption_keys entry (algorithm: scrambler) for this key id; recording ciphertext (docs/dmr-encryption.md)",
				"system", s.system, "serial", s.serial, "key_id", v.KeyID)
		default:
			s.c.log.Info("composer: nxdn cipher has no in-process decoder; recording ciphertext",
				"system", s.system, "serial", s.serial, "cipher", v.Cipher.String())
		}
	}
	if s.c.bus == nil {
		return
	}
	s.c.bus.Publish(events.Event{
		Kind: events.KindCallEncryption,
		Payload: trunking.CallEncryption{
			DeviceSerial: s.serial,
			Protocol:     "nxdn",
			GroupID:      s.groupID,
			AlgorithmID:  uint8(v.Cipher),
			KeyID:        uint16(v.KeyID),
			At:           time.Now(),
		},
	})
}

// logFields are the cipher counters for the chain's ended line.
func (s *nxdnCipherState) logFields() []any {
	t := &s.tracker
	out := []any{"sacch_ok", t.SACCHOK, "sacch_fail", t.SACCHFail, "sacch_superframes", t.Superframes}
	if !t.HaveVCALL {
		return append(out, "vcall", "none")
	}
	out = append(out, "src", t.VCALL.Source, "dst", t.VCALL.Destination, "cipher", t.VCALL.Cipher.String())
	if t.VCALL.Encrypted() {
		out = append(out, "key_id", t.VCALL.KeyID, "key_configured", s.keyConfigured,
			"descrambled_frames", t.Descrambled, "ciphertext_frames", t.Ciphertext, "unaligned_frames", t.Unaligned,
			"pre_vcall_frames", t.Unknown)
	}
	return out
}
