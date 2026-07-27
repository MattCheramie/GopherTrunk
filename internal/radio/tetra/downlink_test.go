package tetra

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestProcessDecodesGrantFromNCDB is the regression for TETRA voice grants never
// decoding on real air (the receiver fed raw SCH/F bits to the L3 parser and
// missed the MAC layer + used the wrong burst geometry). It lays a real captured
// MAC-RESOURCE grant PDU (carrier 2716 / timeslot 1 / SSI 1005356, from the
// 467.913 MHz SCBS capture) into a correctly-shaped Normal Continuous Downlink
// Burst — BKN1 before the training sequence, BKN2 after, AACH either side — and
// asserts Process publishes the grant. Before the NCDB + MAC wiring this
// produced no grant.
func TestProcessDecodesGrantFromNCDB(t *testing.T) {
	const colour = 0x1234 // scrambler seed for the round-trip; content is in the PDU

	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys", FrequencyHz: 467_913_000})
	cc.SetChannelCoding(ChannelCodingOn)
	cc.SetColourCode(colour)

	// The exact MAC-RESOURCE SCH/F type-1 payload recovered from the capture.
	info := hexToBits(t, "20760f572c3c83d538c90a1fe305009e0f92813c83d538c91e1fe301304a1eae5880")[:268]
	type5 := EncodeSCHF(info, colour)
	dibits := TetraBitsToDibits(type5) // 216 dibits = BKN1(108) + BKN2(108)
	if len(dibits) != 216 {
		t.Fatalf("SCH/F type-5 → %d dibits, want 216", len(dibits))
	}
	bkn1, bkn2 := dibits[:108], dibits[108:]

	// Assemble the NCDB: [pad][BKN1][AACH-h1][NTS][AACH-h2][BKN2][tail]. The AACH
	// halves are left as fill (the decoder falls back to trying all rotations
	// when the AACH does not decode).
	var stream []uint8
	stream = append(stream, make([]uint8, 24)...) // priming / BKN1 look-back headroom
	stream = append(stream, bkn1...)
	stream = append(stream, make([]uint8, ndbAACH1Len)...)
	stream = append(stream, NormalSyncDibits()...)
	stream = append(stream, make([]uint8, ndbAACH2Len)...)
	stream = append(stream, bkn2...)
	stream = append(stream, make([]uint8, 8)...) // look-ahead tail

	cc.Process(stream, 0)

	var grant *trunking.Grant
	for drained := false; !drained; {
		select {
		case ev := <-sub.C:
			if g, ok := ev.Payload.(trunking.Grant); ok && ev.Kind == events.KindGrant {
				gg := g
				grant = &gg
			}
		default:
			drained = true
		}
	}
	if grant == nil {
		t.Fatal("Process published no grant for a real MAC-RESOURCE grant in an NCDB")
	}
	if grant.ChannelNum != 2716 {
		t.Errorf("grant carrier = %d, want 2716", grant.ChannelNum)
	}
	if grant.GroupID != 1005356 {
		t.Errorf("grant group/dest SSI = %d, want 1005356", grant.GroupID)
	}
	if grant.Timeslot != 2 { // chan-alloc timeslot 1 (0-based) → 1-based 2
		t.Errorf("grant timeslot = %d, want 2", grant.Timeslot)
	}
	// The grant is addressed by SSI+usage (address type 6), so it carries the
	// call's downlink usage marker — the reliable key the voice chain uses to
	// demultiplex concurrent same-carrier calls (the channel-allocation timeslot
	// field does not map to the physical slot on real air). It must survive the
	// MAC → VoiceGrant → trunking.Grant plumbing so the voice chain can match it
	// against each burst's AACH marker.
	if grant.TETRAUsageMarker != 15 {
		t.Errorf("grant usage marker = %d, want 15 (from the addrSSIUsage grant)", grant.TETRAUsageMarker)
	}
}

// TestCarrierFrequencyDerivation checks that a grant carrier number resolves to
// Hz relative to the cell's own carrier (learned from SYSINFO) at 25 kHz
// spacing, with no configured band plan.
func TestCarrierFrequencyDerivation(t *testing.T) {
	cc := New(Options{Log: slog.Default(), FrequencyHz: 467_913_000})
	if _, ok := cc.carrierFrequency(2716); ok {
		t.Error("carrierFrequency resolved before the main carrier was learned")
	}
	cc.learnMainCarrier(2716)
	for carrier, want := range map[uint16]uint32{
		2716: 467_913_000, // same carrier ⇒ the control frequency
		2717: 467_938_000, // +25 kHz
		2715: 467_888_000, // −25 kHz
	} {
		hz, ok := cc.carrierFrequency(carrier)
		if !ok || hz != want {
			t.Errorf("carrier %d → %d (ok=%v), want %d", carrier, hz, ok, want)
		}
	}
}

// TestGrantPublishesResolvedFrequency drives the MAC ingest end to end: a real
// SYSINFO broadcast teaches the cell's carrier + frequency band + offset, then a
// real MAC-RESOURCE grant publishes with the traffic frequency resolved to the
// cell's own carrier. The SYSINFO carries frequency band 4 (400 MHz base), main
// carrier 2716 and offset field 3 (+12.5 kHz), so the *true* downlink carrier is
// 400 MHz + 2716×25 kHz + 12.5 kHz = 467.9125 MHz — 500 Hz below the nominal
// 467.913 the device was tuned to. Resolving the offset field (rather than
// echoing the tuned frequency) is the fix under test.
func TestGrantPublishesResolvedFrequency(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys", FrequencyHz: 467_913_000})
	cc.ingestMAC(hexToBits(t, "8a9c4c0e928eec8bd0c0041cffffd700"))                                     // SYSINFO → carrier 2716
	cc.ingestMAC(hexToBits(t, "20760f572c3c83d538c90a1fe305009e0f92813c83d538c91e1fe301304a1eae5880")) // grant

	var grant *trunking.Grant
	for drained := false; !drained; {
		select {
		case ev := <-sub.C:
			if g, ok := ev.Payload.(trunking.Grant); ok && ev.Kind == events.KindGrant {
				gg := g
				grant = &gg
			}
		default:
			drained = true
		}
	}
	if grant == nil {
		t.Fatal("no grant published")
	}
	if grant.FrequencyHz != 467_912_500 {
		t.Errorf("grant frequency = %d, want 467912500 (carrier 2716 + band 4 + offset +12.5 kHz)", grant.FrequencyHz)
	}
}
