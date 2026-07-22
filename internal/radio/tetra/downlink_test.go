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
}
