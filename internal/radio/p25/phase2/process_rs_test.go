package phase2

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// TestParseRSMode covers the user-facing config-string → RSMode
// mapping. Empty / off / false / 0 → RSOff (the default); on / true /
// 1 → RSOn; everything else returns RSOff with ok = false.
func TestParseRSMode(t *testing.T) {
	cases := []struct {
		in   string
		want RSMode
		ok   bool
	}{
		{"", RSOff, true},
		{"off", RSOff, true},
		{"OFF", RSOff, true},
		{"false", RSOff, true},
		{"0", RSOff, true},
		{"on", RSOn, true},
		{"On", RSOn, true},
		{"true", RSOn, true},
		{"1", RSOn, true},
		{" on ", RSOn, true},
		{"yes", RSOff, false},
		{"rs", RSOff, false},
	}
	for _, c := range cases {
		got, ok := ParseRSMode(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("ParseRSMode(%q) = (%d, %v); want (%d, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

// TestSetRSModeDefault verifies the ControlChannel boots with RSOff
// (no outer RS verification on the MAC PDU). SetRSMode round-trips
// through RSMode().
func TestSetRSModeDefault(t *testing.T) {
	cc := New(Options{
		Bus:         events.NewBus(8),
		SystemName:  "P25P2",
		FrequencyHz: 851_000_000,
	})
	if got := cc.RSMode(); got != RSOff {
		t.Errorf("New() RSMode = %d, want %d (RSOff)", got, RSOff)
	}
	cc.SetRSMode(RSOn)
	if got := cc.RSMode(); got != RSOn {
		t.Errorf("SetRSMode(RSOn) did not take effect; RSMode = %d", got)
	}
	cc.SetRSMode(RSOff)
	if got := cc.RSMode(); got != RSOff {
		t.Errorf("SetRSMode(RSOff) did not take effect; RSMode = %d", got)
	}
}

// TestVerifyMACPDURSAcceptsEncodedCodeword feeds an 18-byte payload
// that has been packed from a valid 24-symbol RS(24, 16, 9) codeword
// through verifyMACPDURS and asserts true. The reverse — flipping
// any bit — asserts false. This exercises the bit-packing path the
// Process adapter uses when RSMode == RSOn.
func TestVerifyMACPDURSAcceptsEncodedCodeword(t *testing.T) {
	info := [16]byte{0o31, 0o12, 0o42, 0o77, 0o00, 0o63, 0o25, 0o14,
		0o55, 0o22, 0o44, 0o70, 0o03, 0o01, 0o66, 0o57}
	cw := framing.EncodeRS24_16(info)
	// Pack the 24 hex symbols (6 bits each) into 18 bytes MSB-first.
	var pdu [18]byte
	bitIdx := 0
	for _, sym := range cw {
		for b := 5; b >= 0; b-- {
			if (sym>>uint(b))&1 == 1 {
				pdu[bitIdx>>3] |= 1 << uint(7-(bitIdx&7))
			}
			bitIdx++
		}
	}
	if !verifyMACPDURS(pdu[:]) {
		t.Errorf("verifyMACPDURS rejected a freshly-encoded codeword")
	}
	// Flip the most-significant bit of the 5th symbol (bit 24 of the
	// 144-bit stream) and confirm rejection.
	bad := pdu
	bad[3] ^= 0x01
	if verifyMACPDURS(bad[:]) {
		t.Errorf("verifyMACPDURS accepted a single-bit-flipped codeword")
	}
}

// TestVerifyMACPDURSRejectsWrongLength confirms the helper rejects
// any input that isn't exactly 18 bytes.
func TestVerifyMACPDURSRejectsWrongLength(t *testing.T) {
	if verifyMACPDURS(make([]byte, 17)) {
		t.Errorf("verifyMACPDURS accepted 17-byte input")
	}
	if verifyMACPDURS(make([]byte, 19)) {
		t.Errorf("verifyMACPDURS accepted 19-byte input")
	}
	if verifyMACPDURS(nil) {
		t.Errorf("verifyMACPDURS accepted nil input")
	}
}
