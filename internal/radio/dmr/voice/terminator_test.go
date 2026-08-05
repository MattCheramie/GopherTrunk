package voice

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// buildTerminatorBurstDibits builds a full on-air Terminator-with-LC burst
// carrying flc: BPTC(196,96) over the RS(12,9) terminator-seeded FLC, the
// slot-type Hamming codeword for DTTerminatorWithLC, and a BS-Data sync.
// Mirrors the Tier II process test's voice-LC-header builder.
func buildTerminatorBurstDibits(t *testing.T, flc dmr.FLC, colorCode uint8) []uint8 {
	t.Helper()
	flcBytes := dmr.AssembleFLC(flc)
	var data [9]byte
	copy(data[:], flcBytes)
	cw := framing.EncodeRS12_9(data)
	for i := 0; i < 3; i++ {
		cw[9+i] ^= framing.RS129SeedTerminatorLC[i]
	}
	bits := make([]byte, 96)
	for i := 0; i < 96; i++ {
		bits[i] = (cw[i>>3] >> uint(7-(i&7))) & 1
	}
	payloadDibits := framing.BitsToDibits(framing.EncodeBPTC196_96(bits))
	slotDibits := framing.BitsToDibits(dmr.AssembleSlotType(dmr.SlotType{ColorCode: colorCode, DataType: dmr.DTTerminatorWithLC}))

	burst := make([]uint8, 0, dmr.BurstDibits)
	burst = append(burst, payloadDibits[:dmr.HalfPayloadDibits]...)
	burst = append(burst, slotDibits[:dmr.SlotTypeDibits]...)
	burst = append(burst, dmr.BSData.Dibits[:]...)
	burst = append(burst, slotDibits[dmr.SlotTypeDibits:]...)
	burst = append(burst, payloadDibits[dmr.HalfPayloadDibits:]...)
	return burst
}

func TestTerminatorDetectorFindsTerminator(t *testing.T) {
	flc := dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 0x000064, SrcAddr: 0x100200}
	burst := buildTerminatorBurstDibits(t, flc, 5)
	stream := append(make([]uint8, 60), burst...) // lead-in filler
	stream = append(stream, make([]uint8, 40)...) // trailing so the burst is fully buffered

	d := NewTerminatorDetector()
	got := d.Process(stream, 0)
	if len(got) != 1 {
		t.Fatalf("expected 1 terminator, got %d", len(got))
	}
	if dest, ok := got[0].AsGroupVoiceUser(); !ok || dest.GroupAddress != 0x64 {
		t.Fatalf("recovered FLC destination wrong: %+v", got[0])
	}
}

func TestTerminatorDetectorIgnoresNonTerminator(t *testing.T) {
	// A voice-cadence stream (no data-sync terminator) yields nothing.
	stream, _ := buildStream(t, 1, 40)
	d := NewTerminatorDetector()
	if got := d.Process(stream, 0); len(got) != 0 {
		t.Fatalf("expected no terminator in a voice stream, got %d", len(got))
	}
}

func TestTerminatorDetectorRejectsCorruptedPayload(t *testing.T) {
	// A terminator burst whose payload is corrupted past FEC correction must
	// not be reported — the RS/BPTC gate is what keeps a garbled mid-call
	// data burst from forging an end-of-call.
	flc := dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 0x64, SrcAddr: 0x100200}
	burst := buildTerminatorBurstDibits(t, flc, 5)
	// Corrupt many payload dibits (well beyond BPTC/RS correction capacity).
	for i := 0; i < dmr.HalfPayloadDibits; i++ {
		burst[i] ^= 0x3
	}
	stream := append(make([]uint8, 60), burst...)
	stream = append(stream, make([]uint8, 40)...)
	d := NewTerminatorDetector()
	if got := d.Process(stream, 0); len(got) != 0 {
		t.Fatalf("reported a terminator from a corrupted burst (got %d)", len(got))
	}
}
