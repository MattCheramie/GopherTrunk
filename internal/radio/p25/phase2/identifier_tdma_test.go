package phase2

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// tdmaIdenFields packs the 64-bit IDEN_UP_TDMA field group shared by the
// Phase 1 TSBK (0x33) and both Phase 2 MAC forms: iden(4) | channel
// type(4) | tx-offset sign(1) + magnitude(13) | spacing(10) | base/5 (32).
// Built from the references' bit positions, independent of any GT
// assembler: SDRTrunk FrequencyBandUpdateTDMAAbbreviated (iden at octet 2,
// sign bit 16, offset 17-29, spacing 30-39, base octet 6) and
// FrequencyBandUpdateTDMAExtended (the same group one octet later, then
// WACN at octet 11 and system at bit 100); OP25 tk_p25.py op 0xf3 reads
// iden/type from msg[2], offset msg[3:5]>>2, spacing msg[4:6]&0x3ff,
// base msg[6:10].
func tdmaIdenFields(iden, chanType uint8, spacing10 uint16, base uint32) []byte {
	v := uint64(iden&0xF)<<60 | uint64(chanType&0xF)<<56 |
		uint64(1)<<55 | // tx-offset sign bit, magnitude 0
		uint64(spacing10&0x3FF)<<32 | uint64(base/5)
	out := make([]byte, 8)
	for i := range out {
		out[i] = byte(v >> (56 - 8*uint(i)))
	}
	return out
}

// tdmaIdenExtendedPDU is a literal opcode-0xF3 MAC PDU: opcode, the
// reserved octet OP25 skips, the 8-byte field group, then WACN(20) +
// System(12).
func tdmaIdenExtendedPDU(iden, chanType uint8, spacing10 uint16, base uint32, wacn uint32, sys uint16) []byte {
	b := append([]byte{0xF3, 0x00}, tdmaIdenFields(iden, chanType, spacing10, base)...)
	return append(b,
		byte(wacn>>12), byte(wacn>>4), byte(wacn<<4)|byte(sys>>8&0x0F), byte(sys))
}

// TestPhase2TDMAIdentifierUpdateResolvesGrantToCarrier is the #1273
// Phase 2 regression. A Phase 2 control channel announces its TDMA
// channel IDs with IDEN_UP_TDMA (MAC opcode 0x73 abbreviated / 0xF3
// extended), and a grant's channel number counts logical channels
// (carrier × slots + slot). GT parsed only the FDMA form (0x7D), so a site
// that announces its voice channels in a TDMA form left every grant at
// frequency 0 — and had it parsed one, BandPlan.Frequency ignored the slot
// count and would have put channel n on carrier n instead of n/slots.
func TestPhase2TDMAIdentifierUpdateResolvesGrantToCarrier(t *testing.T) {
	const base = 851_000_000 // spacing 12.5 kHz = 100 × 125 Hz
	for _, tc := range []struct {
		name   string
		pdu    []byte
		num    uint16
		wantHz uint32
		wantTS uint8
	}{
		{"0xF3 extended 2-slot, slot 2 of carrier 5",
			tdmaIdenExtendedPDU(7, 0x3, 100, base, 0xBEE00, 0x1CE), 11, base + 5*12_500, 2},
		{"0x73 abbreviated 2-slot, slot 1 of carrier 5",
			append([]byte{0x73}, tdmaIdenFields(7, 0x3, 100, base)...), 10, base + 5*12_500, 1},
		{"0x73 abbreviated 4-slot, slot 4 of carrier 2",
			append([]byte{0x73}, tdmaIdenFields(7, 0x4, 100, base)...), 11, base + 2*12_500, 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bus := events.NewBus(8)
			defer bus.Close()
			sub := bus.Subscribe()
			defer sub.Close()

			cc := New(Options{Bus: bus, SystemName: "p2", FrequencyHz: 851_012_500})
			p, err := ParseMACPDU(tc.pdu)
			if err != nil {
				t.Fatal(err)
			}
			cc.Ingest(p)
			cc.Ingest(grantPDU(42, 0x000123, 7, tc.num))

			var grant *trunking.Grant
			deadline := time.After(time.Second)
			for grant == nil {
				select {
				case ev := <-sub.C:
					if ev.Kind == events.KindGrant {
						g := ev.Payload.(trunking.Grant)
						grant = &g
					}
				case <-deadline:
					t.Fatal("no grant event published")
				}
			}
			if grant.FrequencyHz != tc.wantHz {
				t.Errorf("grant FrequencyHz = %d, want %d", grant.FrequencyHz, tc.wantHz)
			}
			if grant.Timeslot != tc.wantTS {
				t.Errorf("grant Timeslot = %d, want %d", grant.Timeslot, tc.wantTS)
			}
		})
	}
}

// TestPhase2TDMAIdentifierUpdateSurvivesStrictValidation: strict mode (and
// the RSCorrect accept gate) drops a PDU whose opcode is not IsKnown, so the
// TDMA forms must be on that list or a strict control channel never learns
// its band plan.
func TestPhase2TDMAIdentifierUpdateSurvivesStrictValidation(t *testing.T) {
	for _, op := range []Opcode{OpIdentifierUpdateTDMAAbbreviated, OpIdentifierUpdateTDMAExtended} {
		if !op.IsKnown() {
			t.Errorf("%v.IsKnown() = false", op)
		}
	}
	bus := events.NewBus(8)
	defer bus.Close()
	cc := New(Options{Bus: bus, SystemName: "p2", FrequencyHz: 851_012_500})
	cc.SetStrictValidation(true)
	p, err := ParseMACPDU(tdmaIdenExtendedPDU(7, 0x3, 100, 851_000_000, 0xBEE00, 0x1CE))
	if err != nil {
		t.Fatal(err)
	}
	cc.Ingest(p)
	if hz := cc.resolveFreq(7, 11); hz != 851_062_500 {
		t.Errorf("strict control channel resolved (7,11) to %d, want 851062500", hz)
	}
}

// TestPhase2TDMAFormWithFDMAChannelTypeIsNotDivided: channel types 0-2 are
// FDMA (SDRTrunk ChannelType TYPE_0..2, OP25 slots_per_carrier 1) even
// when announced in a TDMA-form PDU, so the channel number is the carrier.
func TestPhase2TDMAFormWithFDMAChannelTypeIsNotDivided(t *testing.T) {
	u, ok := ParseIdentifierUpdateTDMA(tdmaIdenFields(7, 0x1, 100, 851_000_000))
	if !ok {
		t.Fatal("ParseIdentifierUpdateTDMA !ok on an 8-byte field")
	}
	if u.AccessTDMA || u.SlotsPerCarrier > 1 {
		t.Errorf("channel type 1 parsed as TDMA: %+v", u)
	}
	var bp BandPlan
	bp.Apply(u)
	if hz, _ := bp.Frequency(7, 11); hz != 851_000_000+11*12_500 {
		t.Errorf("Frequency(7,11) = %d, want %d", hz, 851_000_000+11*12_500)
	}
	if ts := bp.Timeslot(7, 11); ts != 0 {
		t.Errorf("Timeslot(7,11) = %d, want 0", ts)
	}
	if _, ok := (MACPDU{Opcode: OpIdentifierUpdateTDMAExtended, Payload: make([]byte, 8)}).AsIdentifierUpdate(); ok {
		t.Error("short 0xF3 payload parsed as ok")
	}
}
