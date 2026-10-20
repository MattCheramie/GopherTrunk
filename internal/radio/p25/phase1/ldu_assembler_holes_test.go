package phase1

import "testing"

// holeyLDUStream builds an over of LDUs (alternating LDU1/LDU2) and deletes
// cut[i] dibits from the middle of LDU i, the way an IQ hole upstream of the
// receiver does: the next frame sync then arrives early.
func holeyLDUStream(t *testing.T, cuts []int) ([]uint8, []DUID) {
	t.Helper()
	const nac = 0x293
	var stream []uint8
	var want []DUID
	stream = append(stream, onAirFrameDibits(t, nac, DUIDHeader, 792)...)
	for i, cut := range cuts {
		duid := DUIDLogicalLink1
		if i%2 == 1 {
			duid = DUIDLogicalLink2
		}
		ldu := onAirLDUDibits(t, nac, duid)
		mid := 500
		ldu = append(ldu[:mid:mid], ldu[mid+cut:]...)
		stream = append(stream, ldu...)
		want = append(want, duid)
	}
	stream = append(stream, onAirFrameDibits(t, nac, DUIDTerminator, 144)...)
	want = append(want, DUIDTerminator)
	stream = append(stream, dibitsFor(LDUDibitCount, 1, 3, 0, 2)...)
	return stream, want
}

// TestLDUAssemblerKeepsFrameAfterIQHole pins the #1242 IQ-hole signature:
// frame syncs 790-850 symbols apart instead of 864. The assembler used to take
// a fixed 864-dibit window after every sync, so a short LDU's window swallowed
// the NEXT frame's sync and that frame was lost as well; with holes in a third
// of the LDUs, two thirds of the over never reached the vocoder, and nothing
// in the log said why. The frame after a hole must be kept, and the hole
// counted.
func TestLDUAssemblerKeepsFrameAfterIQHole(t *testing.T) {
	// 9 LDUs (1.6 s); holes of 14..74 dibits in three of them, the range the
	// reporter's captures measured.
	cuts := []int{0, 40, 0, 0, 74, 0, 14, 0, 0}
	stream, want := holeyLDUStream(t, cuts)

	for _, chunk := range []int{len(stream), 1, 37} {
		var got []DUID
		a := NewLDUAssembler(func(ldu []byte) {
			d, err := LDUDuid(ldu)
			if err != nil {
				t.Fatalf("chunk %d: emitted window with undecodable NID: %v", chunk, err)
			}
			got = append(got, d)
		}, 0)
		for i := 0; i < len(stream); i += chunk {
			a.Process(stream[i:min(i+chunk, len(stream))])
		}
		if len(got) != len(want) {
			t.Fatalf("chunk %d: emitted %d frames %v, want %d %v", chunk, len(got), got, len(want), want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("chunk %d: emitted %v, want %v", chunk, got, want)
			}
		}
		n, missing := a.FrameSlips()
		if n != 3 || missing != 40+74+14 {
			t.Fatalf("chunk %d: FrameSlips = %d frames / %d dibits, want 3 / %d", chunk, n, missing, 40+74+14)
		}
	}
}

// TestLDUAssemblerNoSlipsOnCleanStream: a gap-free over reports no slips, so
// the counter can be read as evidence.
func TestLDUAssemblerNoSlipsOnCleanStream(t *testing.T) {
	stream, want := holeyLDUStream(t, make([]int, 9))
	var got int
	a := NewLDUAssembler(func([]byte) { got++ }, 0)
	a.Process(stream)
	if got != len(want) {
		t.Fatalf("emitted %d frames, want %d", got, len(want))
	}
	if n, missing := a.FrameSlips(); n != 0 || missing != 0 {
		t.Fatalf("FrameSlips = %d / %d on a clean stream, want 0 / 0", n, missing)
	}
}
