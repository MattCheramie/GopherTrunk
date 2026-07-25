package tetra

import (
	"bytes"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// buildNCDB lays out one Normal Continuous Downlink Burst into a dibit
// stream: BKN1 and BKN2 at the §9.4.4.3.2 offsets around the normal
// training sequence leading at absolute index L. Positions outside the
// blocks/TS are filler.
func buildNCDB(stream []uint8, L int, bkn1, bkn2 []uint8) {
	nts := NormalSyncDibits()
	copy(stream[L:], nts)
	copy(stream[L+ndbBKN1Start:], bkn1)
	copy(stream[L+ndbBKN2Start:], bkn2)
}

func rampDibits(n int, seed uint8) []uint8 {
	out := make([]uint8, n)
	for i := range out {
		out[i] = (uint8(i) + seed) & 3
	}
	return out
}

func wantFrame(bkn1, bkn2 []uint8) []byte {
	d := append(append([]uint8{}, bkn1...), bkn2...)
	return framing.PackBitsMSB(TetraDibitsToBits(d))
}

func TestTrafficExtractorSingleBurst(t *testing.T) {
	bkn1 := rampDibits(ndbBlockDibits, 0)
	bkn2 := rampDibits(ndbBlockDibits, 2)
	stream := make([]uint8, 600)
	L := 200
	buildNCDB(stream, L, bkn1, bkn2)

	var frames [][]byte
	te := NewTrafficExtractor(0, func(f []byte, _, _ uint8) {
		frames = append(frames, append([]byte(nil), f...))
	})
	te.Process(stream, 0)

	if len(frames) != 1 {
		t.Fatalf("emitted %d frames, want 1", len(frames))
	}
	if len(frames[0]) != TrafficFrameBytes {
		t.Errorf("frame is %d bytes, want %d", len(frames[0]), TrafficFrameBytes)
	}
	if !bytes.Equal(frames[0], wantFrame(bkn1, bkn2)) {
		t.Errorf("frame payload mismatch")
	}
}

// TestTrafficExtractorDescrambles pins the voice-path descramble: with a
// non-zero colour code the emitted frame is the raw type-5 bits descrambled
// with the cell's extended colour code — the input the TCH/S channel decoder
// expects — not the still-scrambled bits.
func TestTrafficExtractorDescrambles(t *testing.T) {
	bkn1 := rampDibits(ndbBlockDibits, 0)
	bkn2 := rampDibits(ndbBlockDibits, 2)
	stream := make([]uint8, 600)
	buildNCDB(stream, 200, bkn1, bkn2)

	const colour = 262144876 // the reporter's cell (467.913 MHz)
	var got []byte
	te := NewTrafficExtractor(colour, func(f []byte, _, _ uint8) { got = append([]byte(nil), f...) })
	te.Process(stream, 0)

	rawBits := TetraDibitsToBits(append(append([]uint8{}, bkn1...), bkn2...))
	want := framing.PackBitsMSB(framing.DescrambleTetra(rawBits, colour))
	if !bytes.Equal(got, want) {
		t.Errorf("emitted frame is not descrambled with the colour code")
	}
	if bytes.Equal(got, wantFrame(bkn1, bkn2)) {
		t.Errorf("descramble not applied — frame equals the still-scrambled bits")
	}
}

func TestTrafficExtractorChunkedAcrossCalls(t *testing.T) {
	bkn1 := rampDibits(ndbBlockDibits, 1)
	bkn2 := rampDibits(ndbBlockDibits, 3)
	stream := make([]uint8, 600)
	L := 300
	buildNCDB(stream, L, bkn1, bkn2)

	var frames [][]byte
	te := NewTrafficExtractor(0, func(f []byte, _, _ uint8) {
		frames = append(frames, append([]byte(nil), f...))
	})
	// Feed in small chunks so the training-sequence match, BKN1 look-back
	// and BKN2 look-ahead land in different Process calls.
	const chunk = 37
	for i := 0; i < len(stream); i += chunk {
		e := i + chunk
		if e > len(stream) {
			e = len(stream)
		}
		te.Process(stream[i:e], i)
	}
	if len(frames) != 1 {
		t.Fatalf("emitted %d frames across chunks, want 1", len(frames))
	}
	if !bytes.Equal(frames[0], wantFrame(bkn1, bkn2)) {
		t.Errorf("chunked frame payload mismatch")
	}
}

func TestTrafficExtractorMultipleSlots(t *testing.T) {
	// Two consecutive timeslots (255 dibits apart) both carrying an NCDB.
	stream := make([]uint8, 1000)
	b1a, b2a := rampDibits(ndbBlockDibits, 0), rampDibits(ndbBlockDibits, 1)
	b1b, b2b := rampDibits(ndbBlockDibits, 2), rampDibits(ndbBlockDibits, 3)
	buildNCDB(stream, 200, b1a, b2a)
	buildNCDB(stream, 200+255, b1b, b2b)

	var frames [][]byte
	te := NewTrafficExtractor(0, func(f []byte, _, _ uint8) {
		frames = append(frames, append([]byte(nil), f...))
	})
	te.Process(stream, 0)
	if len(frames) != 2 {
		t.Fatalf("emitted %d frames, want 2", len(frames))
	}
	if !bytes.Equal(frames[0], wantFrame(b1a, b2a)) || !bytes.Equal(frames[1], wantFrame(b1b, b2b)) {
		t.Errorf("multi-slot frame payloads mismatch")
	}
}

// TestTrafficExtractorSlotTagging pins the SB-anchored slot numbering: a
// synchronisation burst (STS) anchors slot 1 (TN1), and each subsequent NCDB is
// tagged with its TDMA timeslot by rounding its 255-dibit offset from the
// anchor. The small (+30-dibit) offsets prove the rounding absorbs the intra-
// slot NTS↔STS displacement (< half a slot).
func TestTrafficExtractorSlotTagging(t *testing.T) {
	const sbL = 200
	stream := make([]uint8, 1500)
	for i := range stream {
		stream[i] = uint8((i*7 + 1) % 4) // filler
	}
	// Synchronisation burst: STS leading at sbL anchors slot 1.
	copy(stream[sbL:], SyncTrainingDibits())

	// NCDBs at slots 2, 3, 4, then slot 1 of the next frame — each offset +30
	// dibits from the exact grid to exercise the rounding tolerance.
	type ndb struct {
		L        int
		wantSlot uint8
	}
	// The SB's STS anchors the grid one NDB-slot before its frame's TN1 traffic
	// (ndbSBSlotShift), so the first NDB after the SB is TN1, then 2, 3, 4.
	ndbs := []ndb{
		{sbL + 1*255 + 30, 1},
		{sbL + 2*255 + 30, 2},
		{sbL + 3*255 + 30, 3},
		{sbL + 4*255 + 30, 4},
	}
	for i, n := range ndbs {
		buildNCDB(stream, n.L, rampDibits(ndbBlockDibits, uint8(i)), rampDibits(ndbBlockDibits, uint8(i+1)))
	}

	var slots []uint8
	te := NewTrafficExtractor(0, func(_ []byte, slot, _ uint8) { slots = append(slots, slot) })
	te.Process(stream, 0)

	if len(slots) != len(ndbs) {
		t.Fatalf("emitted %d frames, want %d (slots=%v)", len(slots), len(ndbs), slots)
	}
	for i, n := range ndbs {
		if slots[i] != n.wantSlot {
			t.Errorf("burst %d (L=%d): slot=%d, want %d", i, n.L, slots[i], n.wantSlot)
		}
	}
}

// TestTrafficExtractorSlotUnanchored checks that without a synchronisation burst
// the slot is reported as 0 (unknown) — the single-call / traffic-only-carrier
// fallback.
func TestTrafficExtractorSlotUnanchored(t *testing.T) {
	stream := make([]uint8, 600)
	buildNCDB(stream, 300, rampDibits(ndbBlockDibits, 0), rampDibits(ndbBlockDibits, 1))
	var slots []uint8
	te := NewTrafficExtractor(0, func(_ []byte, slot, _ uint8) { slots = append(slots, slot) })
	te.Process(stream, 0)
	if len(slots) != 1 || slots[0] != 0 {
		t.Fatalf("unanchored slots=%v, want [0]", slots)
	}
}

func TestTrafficExtractorNoFalseEmitOnNoise(t *testing.T) {
	// A stream with no training sequence must emit nothing.
	stream := make([]uint8, 2000)
	for i := range stream {
		stream[i] = uint8((i*7 + 1) % 4) // deterministic non-sync filler
	}
	n := 0
	te := NewTrafficExtractor(0, func(f []byte, _, _ uint8) { n++ })
	te.Process(stream, 0)
	// A rare chance correlation within tolerance is possible; assert it does
	// not spuriously fire on structured filler.
	if n > 0 {
		t.Errorf("emitted %d frames on noise, want 0", n)
	}
}
