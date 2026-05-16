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
		{"probe", ScramblerProbe, true},
		{"PROBE", ScramblerProbe, true},
		{"auto", ScramblerProbe, true},
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

// TestSetScramblerOffsetRoundTrip verifies the offset getter / setter
// round-trip pair.
func TestSetScramblerOffsetRoundTrip(t *testing.T) {
	cc := New(Options{
		Bus:         events.NewBus(8),
		SystemName:  "P25P2",
		FrequencyHz: 851_000_000,
	})
	if got := cc.ScramblerOffset(); got != 0 {
		t.Errorf("New() ScramblerOffset = %d, want 0", got)
	}
	cc.SetScramblerOffset(1440)
	if got := cc.ScramblerOffset(); got != 1440 {
		t.Errorf("SetScramblerOffset(1440) did not take effect; got %d", got)
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
// info-bit window. The blind-probe variant
// (TestProcessDescramblerProbeFindsOffset below) exercises the
// per-burst superframe-offset search.
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
	cc.tryIngestMACPDU(infoDibits, TrellisOff, RSOff, ScramblerOn, seed, 0)

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

// TestProcessDescramblerProbeFindsOffset exercises ScramblerProbe
// against a fixture pre-scrambled at slot offset #5 from
// framing.PN44SlotOffsetsOutbound. The probe walks all 12 spec-
// defined slot offsets and must lock onto the right one purely from
// the RS(24, 16, 9) syndrome check.
//
// The fixture wraps a known-shape MAC PDU in an RS(24, 16, 9)
// codeword first so the syndrome check tells the probe which
// candidate offset is the true PDU. Without the RS layer there's
// no way to disambiguate the 12 candidates, which is why
// ScramblerProbe degrades silently when RSMode != RSOn.
func TestProcessDescramblerProbeFindsOffset(t *testing.T) {
	const (
		seed       = uint64(0xABCDE0123)
		slotIndex  = 5
		opGrpGrant = byte(0x40)
	)
	offset := framing.PN44SlotOffsetsOutbound[slotIndex]

	// Build a MAC PDU whose first 16 hex symbols (96 bits = 12
	// bytes) carry the payload, then RS-encode to fill the trailing
	// 8 hex symbols (48 bits = 6 bytes) with parity.
	var info [16]byte
	info[0] = opGrpGrant >> 2
	info[1] = (opGrpGrant&0x3)<<4 | 0x09>>4
	info[2] = (0x09 & 0xF) << 2
	cw := framing.EncodeRS24_16(info)
	// Pack 24 hex symbols (6 bits each) into 18 bytes MSB-first.
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
	// Confirm the un-scrambled, un-offset RS codeword passes
	// verification; otherwise the probe assertion below tells us
	// nothing.
	if !verifyMACPDURS(pdu[:]) {
		t.Fatalf("hand-built RS codeword does not verify; check encoder layout")
	}

	// Scramble the 144 bits at the chosen slot offset.
	var bits [144]byte
	for i := 0; i < 18; i++ {
		for j := 0; j < 8; j++ {
			bits[i*8+j] = (pdu[i] >> uint(7-j)) & 1
		}
	}
	scr := framing.NewPN44Scrambler(seed)
	scr.Advance(offset)
	scr.Apply(bits[:])

	// Reshape into 72 dibits the way Process feeds tryIngestMACPDU
	// under TrellisOff.
	infoDibits := make([]uint8, 72)
	for i := 0; i < 72; i++ {
		b1 := bits[i*2]
		b0 := bits[i*2+1]
		infoDibits[i] = (b1 << 1) | b0
	}

	bus := events.NewBus(8)
	sub := bus.Subscribe()
	defer sub.Close()
	cc := New(Options{
		Bus:         bus,
		SystemName:  "P25P2",
		FrequencyHz: 851_000_000,
	})
	// Probe mode requires RSOn — without RS verification it
	// degrades to ScramblerOn (offset 0) and won't find a match
	// for a fixture scrambled at offset != 0.
	cc.SetRSMode(RSOn)
	cc.SetScramblerMode(ScramblerProbe)
	cc.SetScramblerSeed(seed)
	cc.tryIngestMACPDU(infoDibits, TrellisOff, RSOn, ScramblerProbe, seed, 0)

	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLocked && ev.Kind != events.KindGrant {
			t.Errorf("expected cc.locked or grant; got %v", ev.Kind)
		}
	default:
		t.Fatal("probe-mode descrambler did not lock onto the correct slot offset")
	}
}

// TestProcessDescramblerProbeRejectsWrongSeed confirms the probe
// returns silently (no false-positive Ingest) when no candidate
// offset produces an RS-verifying PDU. A seed that doesn't match
// the scrambling seed should fail all 12 candidate offsets.
func TestProcessDescramblerProbeRejectsWrongSeed(t *testing.T) {
	const (
		seed          = uint64(0xABCDE0123)
		differentSeed = uint64(0x1234567890A)
		slotIndex     = 3
	)
	offset := framing.PN44SlotOffsetsOutbound[slotIndex]

	// Build any RS-encoded MAC PDU.
	var info [16]byte
	info[0] = 0x10
	info[1] = 0x09
	cw := framing.EncodeRS24_16(info)
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
	var bits [144]byte
	for i := 0; i < 18; i++ {
		for j := 0; j < 8; j++ {
			bits[i*8+j] = (pdu[i] >> uint(7-j)) & 1
		}
	}
	scr := framing.NewPN44Scrambler(seed)
	scr.Advance(offset)
	scr.Apply(bits[:])
	infoDibits := make([]uint8, 72)
	for i := 0; i < 72; i++ {
		infoDibits[i] = (bits[i*2] << 1) | bits[i*2+1]
	}

	bus := events.NewBus(8)
	sub := bus.Subscribe()
	defer sub.Close()
	cc := New(Options{
		Bus:         bus,
		SystemName:  "P25P2",
		FrequencyHz: 851_000_000,
	})
	cc.SetRSMode(RSOn)
	cc.SetScramblerMode(ScramblerProbe)
	cc.tryIngestMACPDU(infoDibits, TrellisOff, RSOn, ScramblerProbe, differentSeed, 0)

	select {
	case ev := <-sub.C:
		t.Errorf("probe with wrong seed produced an event: %v", ev.Kind)
	default:
		// Expected: no event lands because none of the 12 offsets
		// produce an RS-verifying PDU under the wrong seed.
	}
}
