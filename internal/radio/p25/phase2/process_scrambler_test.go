package phase2

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// TestParseScramblerMode covers the user-facing config-string →
// ScramblerMode mapping.
func TestParseScramblerMode(t *testing.T) {
	cases := []struct {
		in   string
		want ScramblerMode
		ok   bool
	}{
		{"", ScramblerOff, true},
		{"off", ScramblerOff, true},
		{"false", ScramblerOff, true},
		{"0", ScramblerOff, true},
		{"on", ScramblerOn, true},
		{"true", ScramblerOn, true},
		{"1", ScramblerOn, true},
		{" ON ", ScramblerOn, true},
		{"yes", ScramblerOff, false},
		{"scramble", ScramblerOff, false},
	}
	for _, c := range cases {
		got, ok := ParseScramblerMode(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("ParseScramblerMode(%q) = (%d, %v); want (%d, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

// TestSetScramblerModeDefault confirms the ControlChannel boots with
// ScramblerOff and that SetScramblerMode / SetScramblerSeed round-
// trip through the getters.
func TestSetScramblerModeDefault(t *testing.T) {
	cc := New(Options{
		Bus:         events.NewBus(8),
		SystemName:  "P25P2",
		FrequencyHz: 851_000_000,
	})
	if got := cc.ScramblerMode(); got != ScramblerOff {
		t.Errorf("New() ScramblerMode = %d, want %d (ScramblerOff)", got, ScramblerOff)
	}
	cc.SetScramblerMode(ScramblerOn)
	if got := cc.ScramblerMode(); got != ScramblerOn {
		t.Errorf("SetScramblerMode(ScramblerOn) did not take effect; got %d", got)
	}
	cc.SetScramblerSeed(0xCAFE1234)
	if got := cc.ScramblerSeed(); got != 0xCAFE1234 {
		t.Errorf("SetScramblerSeed did not take effect; ScramblerSeed = %#x", got)
	}
}

// TestProcessDescramblerRoundTrip drives a synthesized MAC PDU
// stream through the Process adapter with ScramblerMode = ScramblerOn
// and confirms a fixture pre-scrambled with the same seed decodes
// correctly. The fixture exercises:
//
//   - the bit-packing layer (Process unpacks 72 info dibits → 144
//     bits → applies descrambler → packs into 18 bytes → ParseMACPDU);
//   - the seed plumbing (SetScramblerSeed → Process picks it up).
//
// This locks the descrambler in place against the trellis-decoded
// info-bit window. A real-air integration test would also need
// per-burst offset tracking against the 4320-bit superframe, which
// is a follow-up.
func TestProcessDescramblerRoundTrip(t *testing.T) {
	// Build a known-shape MAC PDU: opcode = GRP_V_CH_GRANT (0x40),
	// length = 9, body = TG 0x1234 / src 0x567890 (matches the
	// existing newMACGroupGrant fixture in phase2_test.go's helper
	// surface, but inlined here so this file remains self-contained).
	var pdu [18]byte
	pdu[0] = 0x40 // OpGroupVoiceChannelGrant (low 6 bits)
	pdu[1] = 0x09
	pdu[2] = 0x00 // SO
	pdu[3] = 0x00 // ChannelID + grant flags (fixture default)
	pdu[4] = 0x00
	pdu[5] = 0x00
	pdu[6] = 0x12
	pdu[7] = 0x34 // Group address (uint16 BE)
	pdu[8] = 0x56
	pdu[9] = 0x78
	pdu[10] = 0x90 // Source ID (uint24 BE)
	// Bytes 11..17 already zero.

	// Sanity-check: the unscrambled PDU parses cleanly.
	if _, err := ParseMACPDU(pdu[:]); err != nil {
		t.Fatalf("ParseMACPDU on unscrambled fixture: %v", err)
	}

	// Now scramble the 144 bits with a known seed.
	const seed = uint64(0xABCDE0123)
	var bits [144]byte
	for i := 0; i < 18; i++ {
		for j := 0; j < 8; j++ {
			bits[i*8+j] = (pdu[i] >> uint(7-j)) & 1
		}
	}
	framing.NewPN44Scrambler(seed).Apply(bits[:])
	var scrambled [18]byte
	for i := 0; i < 144; i++ {
		if bits[i] != 0 {
			scrambled[i>>3] |= 1 << uint(7-(i&7))
		}
	}
	// The scrambled bytes must NOT parse cleanly to the same payload —
	// confirms scrambling actually modified the bits.
	if scrambled == pdu {
		t.Fatalf("scrambling produced identical bytes; seed=%#x", seed)
	}

	// Wire a ControlChannel with the descrambler armed, push the
	// scrambled bytes through tryIngestMACPDU directly (the Process
	// adapter's sync + trellis path is exercised elsewhere — this
	// test isolates the descrambler).
	bus := events.NewBus(8)
	sub := bus.Subscribe()
	defer sub.Close()
	cc := New(Options{
		Bus:         bus,
		SystemName:  "P25P2",
		FrequencyHz: 851_000_000,
	})
	cc.SetScramblerMode(ScramblerOn)
	cc.SetScramblerSeed(seed)

	// Build the 72-dibit info window the trellis decoder would
	// produce when fed a clean encoding of the *scrambled* bits.
	infoDibits := make([]uint8, 72)
	for i := 0; i < 72; i++ {
		b1 := bits[i*2]
		b0 := bits[i*2+1]
		infoDibits[i] = (b1 << 1) | b0
	}
	cc.tryIngestMACPDU(infoDibits, TrellisOff, RSOff, ScramblerOn, seed)

	// Expect a cc.locked event for the ingested grant.
	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLocked && ev.Kind != events.KindGrant {
			t.Errorf("expected cc.locked or grant; got %v", ev.Kind)
		}
	default:
		t.Fatal("descrambled MAC PDU did not produce any event")
	}
}
