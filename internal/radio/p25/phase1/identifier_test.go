package phase1

import (
	"errors"
	"testing"
)

func TestIdentifierUpdateRoundTrip700MHz(t *testing.T) {
	// Realistic 700/800 MHz Phase 1 fixture: ChannelID 1, base 851.0
	// MHz, 12.5 kHz spacing + bandwidth, -39 MHz transmit offset.
	in := IdentifierUpdate{
		ChannelID:   1,
		BandwidthHz: 12_500,
		SpacingHz:   12_500,
		TxOffsetHz:  -39_000_000,
		BaseHz:      851_000_000,
	}
	out := ParseIdentifierUpdate(AssembleIdentifierUpdate(in))
	if out != in {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", out, in)
	}
}

func TestIdentifierUpdateRoundTripPositiveOffset(t *testing.T) {
	// VHF-style fixture with a positive offset to exercise the sign
	// bit. Spacing is 6.25 kHz; bandwidth differs from spacing.
	in := IdentifierUpdate{
		ChannelID:   5,
		BandwidthHz: 6_250,
		SpacingHz:   6_250,
		TxOffsetHz:  5_000_000,
		BaseHz:      154_000_000,
	}
	out := ParseIdentifierUpdate(AssembleIdentifierUpdate(in))
	if out != in {
		t.Errorf("round-trip mismatch: got %+v want %+v", out, in)
	}
}

func TestIdentifierUpdateVUHFRoundTripNegativeOffset(t *testing.T) {
	// Realistic UHF P25 site fixture: ChannelID 1, base 460.0 MHz,
	// 12.5 kHz spacing + bandwidth, -5 MHz transmit offset.
	in := IdentifierUpdate{
		ChannelID:   1,
		BandwidthHz: 12_500,
		SpacingHz:   12_500,
		TxOffsetHz:  -5_000_000,
		BaseHz:      460_000_000,
	}
	out := ParseIdentifierUpdateVUHF(AssembleIdentifierUpdateVUHF(in))
	if out != in {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", out, in)
	}
}

func TestIdentifierUpdateVUHFRoundTripPositiveOffset(t *testing.T) {
	// 6.25 kHz narrowband VHF fixture with a positive offset to exercise
	// the sign bit.
	in := IdentifierUpdate{
		ChannelID:   3,
		BandwidthHz: 6_250,
		SpacingHz:   6_250,
		TxOffsetHz:  5_000_000,
		BaseHz:      154_000_000,
	}
	out := ParseIdentifierUpdateVUHF(AssembleIdentifierUpdateVUHF(in))
	if out != in {
		t.Errorf("round-trip mismatch: got %+v want %+v", out, in)
	}
}

// TestIdentifierUpdateTDMARoundTripMtAnakieID10 mirrors the VUHF
// round-trip with the Mt Anakie id=10 fixture from issue #345 (base
// 467.5125 MHz, 6.25 kHz step, -10 MHz tx offset). The on-air log that
// fixture came from did not record the channel-type code, so the
// fixture states it explicitly as a 2-slot Phase 2 carrier.
func TestIdentifierUpdateTDMARoundTripMtAnakieID10(t *testing.T) {
	in := IdentifierUpdate{
		ChannelID:       10,
		BandwidthHz:     12_500, // channel type 3 → TDMA 12.5 kHz, 2 slots
		SpacingHz:       6_250,
		TxOffsetHz:      -10_000_000,
		BaseHz:          467_512_500,
		AccessTDMA:      true,
		SlotsPerCarrier: 2,
	}
	out := ParseIdentifierUpdateTDMA(AssembleIdentifierUpdateTDMA(in))
	if out != in {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", out, in)
	}
}

func TestIdentifierUpdateTDMARoundTripPositiveOffset(t *testing.T) {
	// 25 kHz TDMA-4 fixture with a positive offset to exercise the
	// sign bit and the 4-slot channel-type code.
	in := IdentifierUpdate{
		ChannelID:       5,
		BandwidthHz:     25_000, // channel type 4 → TDMA 25 kHz, 4 slots
		SpacingHz:       12_500,
		TxOffsetHz:      5_000_000,
		BaseHz:          420_000_000,
		AccessTDMA:      true,
		SlotsPerCarrier: 4,
	}
	out := ParseIdentifierUpdateTDMA(AssembleIdentifierUpdateTDMA(in))
	if out != in {
		t.Errorf("round-trip mismatch: got %+v want %+v", out, in)
	}
}

// TestTDMAChannelTypeTable pins the 0x33 channel-type table to the two
// independent decoders that agree on it: SDRTrunk ChannelType TYPE_0..5
// (access, bandwidth, slots) and OP25 tk_p25.py slots_per_carrier
// [1,1,1,2,4,2,...]. Issue #1273.
func TestTDMAChannelTypeTable(t *testing.T) {
	for _, tc := range []struct {
		code  uint8
		tdma  bool
		bwHz  uint32
		slots uint8
	}{
		{0x0, false, 12_500, 1},
		{0x1, false, 12_500, 1},
		{0x2, false, 6_250, 1},
		{0x3, true, 12_500, 2},
		{0x4, true, 25_000, 4},
		{0x5, true, 12_500, 2},
		{0xF, true, 0, 2}, // reserved: OP25 treats it as 2 slots
	} {
		got := tdmaChannelType(tc.code)
		if got.TDMA != tc.tdma || got.BandwidthHz != tc.bwHz || got.SlotsPerCarrier != tc.slots {
			t.Errorf("tdmaChannelType(0x%X) = %+v, want tdma=%v bw=%d slots=%d",
				tc.code, got, tc.tdma, tc.bwHz, tc.slots)
		}
	}
}

// tdmaIdenPayload builds a literal opcode-0x33 payload from OP25's bit
// positions (tk_p25.py iden_up_tdma: iden >>76, channel_type >>72,
// toff >>58, spac >>48, f1 >>16 of the 96-bit TSBK; the 64-bit payload
// is TSBK bits 16..79). Independent of AssembleIdentifierUpdateTDMA so a
// shared packing mistake cannot hide.
func tdmaIdenPayload(iden, chanType uint8, toff14, spac10 uint16, f1 uint32) [8]byte {
	v := uint64(iden&0xF)<<60 | uint64(chanType&0xF)<<56 |
		uint64(toff14&0x3FFF)<<42 | uint64(spac10&0x3FF)<<32 | uint64(f1)
	var p [8]byte
	for i := 0; i < 8; i++ {
		p[i] = byte(v >> (56 - 8*uint(i)))
	}
	return p
}

// TestBandPlanTDMAGrantResolvesToCarrierNotLogicalChannel is the #1273
// regression. A TDMA grant's channel number counts logical channels
// (carrier × slots + slot): SDRTrunk FrequencyBandUpdateTDMA resolves
// base + spacing × floor(n / slots), OP25 channel_id_to_frequency
// base + step × int(n / tdma), and both take the slot from the
// remainder. The old resolver used base + n × spacing, putting every
// grant on a 2-slot carrier twice as far from the base as its carrier.
func TestBandPlanTDMAGrantResolvesToCarrierNotLogicalChannel(t *testing.T) {
	const base = 161_000_000
	// spacing 12.5 kHz = 100 × 125 Hz; toff sign bit set, magnitude 0.
	for _, tc := range []struct {
		name     string
		chanType uint8
		num      uint16
		wantHz   uint32
		wantTS   uint8
	}{
		{"2-slot slot 1 of carrier 5", 0x3, 10, base + 5*12_500, 1},
		{"2-slot slot 2 of carrier 5", 0x3, 11, base + 5*12_500, 2},
		{"4-slot slot 4 of carrier 2", 0x4, 11, base + 2*12_500, 4},
		{"FDMA type under 0x33 is not divided", 0x1, 11, base + 11*12_500, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := ParseIdentifierUpdateTDMA(tdmaIdenPayload(7, tc.chanType, 1<<13, 100, base/5))
			var bp BandPlan
			bp.Apply(u)
			hz, err := bp.Frequency(7, tc.num)
			if err != nil {
				t.Fatal(err)
			}
			if hz != tc.wantHz {
				t.Errorf("Frequency(7, %d) = %d, want %d", tc.num, hz, tc.wantHz)
			}
			if ts := bp.Timeslot(7, tc.num); ts != tc.wantTS {
				t.Errorf("Timeslot(7, %d) = %d, want %d", tc.num, ts, tc.wantTS)
			}
			if bp.IsTDMA(7) != (tc.wantTS != 0) {
				t.Errorf("IsTDMA(7) = %v, want %v", bp.IsTDMA(7), tc.wantTS != 0)
			}
		})
	}
}

func TestIdentifierUpdateVUHFUnknownBandwidthCodeMapsToZero(t *testing.T) {
	// On-air payload with BW code 0 (reserved per TIA-102.AABF Table 16).
	// Parser must surface BandwidthHz=0 rather than fabricating a value.
	var p [8]byte
	p[0] = 0x10 // ChannelID=1, BW code 0
	p[3] = 0x64 // STEP=100 → 12.5 kHz
	// FREQ=92_000_000 → 460 MHz at ×5 Hz
	freq5 := uint32(460_000_000 / 5)
	p[4] = byte(freq5 >> 24)
	p[5] = byte(freq5 >> 16)
	p[6] = byte(freq5 >> 8)
	p[7] = byte(freq5)

	out := ParseIdentifierUpdateVUHF(p)
	if out.BandwidthHz != 0 {
		t.Errorf("BandwidthHz = %d, want 0 for reserved BW code", out.BandwidthHz)
	}
	if out.SpacingHz != 12_500 {
		t.Errorf("SpacingHz = %d, want 12_500", out.SpacingHz)
	}
	if out.BaseHz != 460_000_000 {
		t.Errorf("BaseHz = %d, want 460_000_000", out.BaseHz)
	}
}

func TestBandPlanResolvesKnownChannel(t *testing.T) {
	bp := &BandPlan{}
	bp.Apply(IdentifierUpdate{
		ChannelID: 1,
		SpacingHz: 12_500,
		BaseHz:    851_000_000,
	})

	// Channel 0 → base. Channel 16 → base + 16 × 12.5 kHz = 851.2 MHz.
	if hz, err := bp.Frequency(1, 0); err != nil || hz != 851_000_000 {
		t.Errorf("Frequency(1,0) = %d, %v; want 851_000_000, nil", hz, err)
	}
	if hz, err := bp.Frequency(1, 16); err != nil || hz != 851_200_000 {
		t.Errorf("Frequency(1,16) = %d, %v; want 851_200_000, nil", hz, err)
	}
}

func TestBandPlanUnknownChannelID(t *testing.T) {
	bp := &BandPlan{}
	_, err := bp.Frequency(7, 0)
	if !errors.Is(err, ErrUnknownChannelID) {
		t.Errorf("Frequency on empty plan: err = %v, want ErrUnknownChannelID", err)
	}
	if bp.Known(7) {
		t.Error("Known(7) on empty plan should be false")
	}
}

func TestBandPlanReplacesSlotOnNewIdentifierUpdate(t *testing.T) {
	bp := &BandPlan{}
	bp.Apply(IdentifierUpdate{ChannelID: 2, SpacingHz: 12_500, BaseHz: 851_000_000})
	bp.Apply(IdentifierUpdate{ChannelID: 2, SpacingHz: 25_000, BaseHz: 852_000_000})

	hz, err := bp.Frequency(2, 4)
	if err != nil {
		t.Fatalf("Frequency: %v", err)
	}
	if hz != 852_100_000 {
		t.Errorf("hz = %d, want 852_100_000 (uses replaced slot)", hz)
	}
}

func TestBandPlanRejectsOverflow(t *testing.T) {
	bp := &BandPlan{}
	bp.Apply(IdentifierUpdate{ChannelID: 0, SpacingHz: 1_000_000, BaseHz: 4_000_000_000})
	if _, err := bp.Frequency(0, 1000); err == nil {
		t.Error("expected overflow error for >4.29 GHz resolved frequency")
	}
}

// TestParseIdentifierUpdateTDMASetsAccessFlag confirms the TDMA
// variant with a TDMA channel type flips AccessTDMA so the Phase 1
// control channel can route grants on the channel ID into the Phase 2
// voice chain (issue #376).
// FDMA variants (the 0x3D and 0x34 forms) must leave it false.
func TestParseIdentifierUpdateTDMASetsAccessFlag(t *testing.T) {
	tdma := tdmaIdenPayload(3, 0x3, 1<<13, 100, 851_000_000/5) // channel type 3
	if u := ParseIdentifierUpdateTDMA(tdma); !u.AccessTDMA {
		t.Errorf("TDMA: AccessTDMA = false, want true")
	}
	// An FDMA channel type carried in the 0x33 form is still FDMA
	// (SDRTrunk ChannelType TYPE_0..2; #1273).
	fdmaType := tdmaIdenPayload(3, 0x1, 1<<13, 100, 851_000_000/5)
	if u := ParseIdentifierUpdateTDMA(fdmaType); u.AccessTDMA {
		t.Errorf("0x33 with FDMA channel type 1: AccessTDMA = true, want false")
	}

	fdma := AssembleIdentifierUpdate(IdentifierUpdate{
		ChannelID: 4, SpacingHz: 12_500, BaseHz: 851_000_000,
	})
	if u := ParseIdentifierUpdate(fdma); u.AccessTDMA {
		t.Errorf("FDMA 0x3D: AccessTDMA = true, want false")
	}

	vuhf := AssembleIdentifierUpdateVUHF(IdentifierUpdate{
		ChannelID: 5, SpacingHz: 6_250, BaseHz: 460_000_000,
	})
	if u := ParseIdentifierUpdateVUHF(vuhf); u.AccessTDMA {
		t.Errorf("VUHF 0x34: AccessTDMA = true, want false")
	}
}

// TestBandPlanIsTDMA round-trips a TDMA IdentifierUpdate through Apply
// and confirms IsTDMA reports true only for channel IDs advertised via
// opcode 0x33. Unknown IDs return false (no false positives), and FDMA
// slots return false even when they are Known.
func TestBandPlanIsTDMA(t *testing.T) {
	bp := &BandPlan{}
	bp.Apply(IdentifierUpdate{
		ChannelID: 1, SpacingHz: 12_500, BaseHz: 851_000_000, AccessTDMA: true,
	})
	bp.Apply(IdentifierUpdate{
		ChannelID: 2, SpacingHz: 12_500, BaseHz: 851_500_000, // AccessTDMA defaults to false
	})

	if !bp.IsTDMA(1) {
		t.Error("IsTDMA(1) = false, want true (TDMA slot)")
	}
	if bp.IsTDMA(2) {
		t.Error("IsTDMA(2) = true, want false (FDMA slot)")
	}
	if bp.IsTDMA(7) {
		t.Error("IsTDMA(7) = true on unknown ID, want false")
	}
	if bp.IsTDMA(99) {
		t.Error("IsTDMA(99) = true on out-of-range ID, want false")
	}
}
