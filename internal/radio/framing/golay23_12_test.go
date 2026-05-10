package framing

import "testing"

func TestGolay23_12RoundTripCleanChannel(t *testing.T) {
	// Encode every 12-bit data value, decode it back unchanged with
	// metric 0 — no off-by-one in the "drop the overall-parity bit"
	// shim.
	for data := uint16(0); data < 1<<12; data++ {
		cw := GolayEncode23_12(data)
		got, errs := GolayDecode23_12(cw)
		if got != data {
			t.Errorf("data=%03X: got %03X (errs=%d)", data, got, errs)
		}
		if errs != 0 {
			t.Errorf("data=%03X: errs=%d, want 0", data, errs)
		}
	}
}

func TestGolay23_12CorrectsSingleBitErrors(t *testing.T) {
	// (23, 12, 7) — distance 7, corrects up to 3 errors. A
	// representative payload + every possible single-bit flip in the
	// 23-bit codeword should still recover the original data.
	data := uint16(0xABC)
	clean := GolayEncode23_12(data)
	for bit := 0; bit < 23; bit++ {
		corrupt := clean ^ (1 << uint(bit))
		got, errs := GolayDecode23_12(corrupt)
		if got != data {
			t.Errorf("bit %d flip: got %03X want %03X (errs=%d)", bit, got, data, errs)
		}
		if errs <= 0 {
			t.Errorf("bit %d flip: errs=%d, want > 0", bit, errs)
		}
	}
}

func TestGolay23_12CorrectsTripleBitErrors(t *testing.T) {
	// Every triple-bit error inside the 23-bit codeword should still
	// decode (correction radius t = 3 = ⌊(d-1)/2⌋ for d=7). Sample a
	// handful so the test stays under a second.
	data := uint16(0x555)
	clean := GolayEncode23_12(data)
	for _, trip := range [][3]int{{0, 1, 2}, {3, 11, 22}, {5, 12, 18}, {0, 11, 22}} {
		corrupt := clean ^ (1 << uint(trip[0])) ^ (1 << uint(trip[1])) ^ (1 << uint(trip[2]))
		got, errs := GolayDecode23_12(corrupt)
		if got != data {
			t.Errorf("triple %v: got %03X want %03X (errs=%d)", trip, got, data, errs)
		}
		if errs <= 0 || errs > 3 {
			t.Errorf("triple %v: errs=%d, want 1..3", trip, errs)
		}
	}
}

func TestGolay23_12HighWeightErrorsTripUncorrectableFlag(t *testing.T) {
	// Beyond the correction radius (t = 3), the decoder can
	// mis-decode to a different valid codeword without flagging it
	// — that's standard for any minimum-distance decoder, not a
	// property unique to this shim. The honest invariant we *can*
	// pin is: when the corrupted word is far from every valid
	// codeword (e.g., random-ish bits across the codeword), at
	// least one of the two append-parity branches reports
	// uncorrectable. Sample a high-Hamming-weight error pattern
	// and verify the survivor branch reports errs == -1.
	data := uint16(0xDEA)
	clean := GolayEncode23_12(data)
	// Eleven flipped bits scattered across the codeword — well
	// past the 3-error correction radius and unlikely to land
	// inside a neighbour codeword's t-ball.
	corrupt := clean ^ 0b101010_10101010_10101010
	_, errs := GolayDecode23_12(corrupt)
	if errs != -1 {
		t.Errorf("11-bit error reported errs=%d, want -1 (uncorrectable)", errs)
	}
}

func TestGolay23_12IgnoresBitsAbove23(t *testing.T) {
	// Stale high bits in the input must not influence decode — the
	// implementation masks input to 23 bits.
	data := uint16(0x123)
	cw := GolayEncode23_12(data)
	got, _ := GolayDecode23_12(cw | 0xFF000000)
	if got != data {
		t.Errorf("got %03X want %03X (high-bit mask leaked)", got, data)
	}
}
