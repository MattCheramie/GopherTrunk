package framing

import "testing"

func TestViterbiK5RoundTripCleanChannel(t *testing.T) {
	// Random-ish 32-bit information word followed by 4 zero tail
	// bits to flush the encoder. Encode → decode should recover the
	// original bits exactly with metric 0.
	info := []byte{
		1, 0, 1, 1, 0, 1, 0, 0, 1, 1, 1, 0, 0, 0, 1, 1,
		1, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 1, 0, 0, 1, 1,
	}
	stages := len(info) + 4 // tail
	in := make([]byte, stages)
	copy(in, info)

	channel := EncodeK5(in)
	if len(channel) != 2*stages {
		t.Fatalf("EncodeK5 len = %d, want %d", len(channel), 2*stages)
	}
	out, metric := ViterbiK5(channel, stages)
	if metric != 0 {
		t.Errorf("clean-channel metric = %d, want 0", metric)
	}
	for i := 0; i < len(in); i++ {
		if out[i] != in[i] {
			t.Errorf("bit %d: got %d, want %d", i, out[i], in[i])
		}
	}
}

func TestViterbiK5CorrectsSingleBitError(t *testing.T) {
	// One bit error per few stages is well within the K=5 ½-rate
	// code's correction radius. The Viterbi survivor should still
	// recover the original info.
	info := []byte{1, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0, 1}
	stages := len(info) + 4
	in := make([]byte, stages)
	copy(in, info)
	channel := EncodeK5(in)
	channel[7] ^= 1 // flip one channel bit

	out, metric := ViterbiK5(channel, stages)
	if metric == 0 {
		t.Error("expected non-zero metric on a corrupted channel")
	}
	for i := 0; i < len(in); i++ {
		if out[i] != in[i] {
			t.Errorf("bit %d: got %d, want %d (single-bit error not corrected)", i, out[i], in[i])
		}
	}
}

func TestViterbiK5DepunctureMarkSkipsCost(t *testing.T) {
	// Stub a "punctured" position with DepunctureMark. The decoder
	// should treat it as no-info (zero cost contribution) so the
	// surviving path's metric equals the number of remaining bit
	// disagreements only.
	info := []byte{1, 1, 0, 0, 1, 0, 1, 0}
	stages := len(info) + 4
	in := make([]byte, stages)
	copy(in, info)
	channel := EncodeK5(in)
	want := byte(channel[3]) // remember it
	channel[3] = DepunctureMark
	out, metric := ViterbiK5(channel, stages)
	// The surviving path should still be the original input since
	// the un-punctured bits constrain it tightly.
	for i := 0; i < len(in); i++ {
		if out[i] != in[i] {
			t.Errorf("bit %d: got %d, want %d", i, out[i], in[i])
		}
	}
	// Metric must not have charged anything for the masked slot.
	if metric != 0 {
		t.Errorf("metric = %d, want 0 (punctured slot should be free)", metric)
	}
	// Sanity: the slot really was 0 or 1 (i.e. we picked a valid
	// channel index and replacing it actually mattered).
	if want != 0 && want != 1 {
		t.Fatalf("test setup bug: channel[3] was %d, want 0 or 1", want)
	}
}

func TestEncodeK5InitialStateZero(t *testing.T) {
	// Encoder starts with d1..d4 = 0; the first input bit alone
	// determines the first (g1, g2) pair: g1 = g2 = bit. A single 1
	// followed by zeros (incl. tail) produces the expected impulse
	// response of the polynomials.
	in := []byte{1, 0, 0, 0, 0, 0, 0, 0, 0}
	out := EncodeK5(in)
	// First pair: g1 = 1, g2 = 1.
	if out[0] != 1 || out[1] != 1 {
		t.Errorf("first pair = (%d,%d), want (1,1)", out[0], out[1])
	}
}
