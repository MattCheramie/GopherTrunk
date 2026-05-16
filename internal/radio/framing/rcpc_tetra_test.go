package framing

import "testing"

// TestRCPCTetraMotherRoundTripCleanChannel: encode a random-ish
// information bit sequence + 4 tail zeros, decode the mother-code
// output, recover the original bits exactly with metric 0.
func TestRCPCTetraMotherRoundTripCleanChannel(t *testing.T) {
	info := []byte{
		1, 0, 1, 1, 0, 1, 0, 0, 1, 1, 1, 0, 0, 0, 1, 1,
		1, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 1, 0, 0, 1, 1,
	}
	stages := len(info) + 4 // K-1 = 4 tail bits
	in := make([]byte, stages)
	copy(in, info)
	channel := EncodeRCPCTetraMother(in)
	if len(channel) != 3*stages {
		t.Fatalf("encoded length = %d, want %d", len(channel), 3*stages)
	}
	out, metric := DecodeRCPCTetraMother(channel, stages)
	if metric != 0 {
		t.Errorf("clean-channel metric = %d, want 0", metric)
	}
	for i, want := range in {
		if out[i] != want {
			t.Errorf("bit %d: got %d, want %d", i, out[i], want)
		}
	}
}

// TestRCPCTetraMotherCorrectsSingleBitError: flip a single channel
// bit in a clean encoding; the Viterbi survivor must still recover
// the original info exactly. K=5 1/3-rate has free distance 12 so
// it corrects many single-bit errors easily.
func TestRCPCTetraMotherCorrectsSingleBitError(t *testing.T) {
	info := []byte{1, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0, 1, 1, 0, 0, 1}
	stages := len(info) + 4
	in := make([]byte, stages)
	copy(in, info)
	channel := EncodeRCPCTetraMother(in)
	channel[11] ^= 1 // flip one channel bit
	out, metric := DecodeRCPCTetraMother(channel, stages)
	if metric == 0 {
		t.Error("expected non-zero metric on a corrupted channel")
	}
	for i, want := range in {
		if out[i] != want {
			t.Errorf("bit %d: got %d, want %d", i, out[i], want)
		}
	}
}

// TestRCPCTetraEncoderInitialStateZero: with all-zero state and a
// single 1 followed by zeros, the first triple is (1, 1, 1) — the
// impulse response of the three generators at lag 0.
func TestRCPCTetraEncoderInitialStateZero(t *testing.T) {
	in := []byte{1, 0, 0, 0, 0, 0, 0, 0, 0}
	out := EncodeRCPCTetraMother(in)
	if out[0] != 1 || out[1] != 1 || out[2] != 1 {
		t.Errorf("first triple = (%d, %d, %d), want (1, 1, 1)", out[0], out[1], out[2])
	}
}

// TestRCPCTetraEncoderMatchesGeneratorPolys: the second triple (lag
// 1, i.e. input = 0, d1 = 1, others zero) must equal the polynomial
// pattern at index 1:
//
//	g1 = 0 ^ 1 ^ 0 ^ 0 ^ 0 = 1
//	g2 = 0 ^ 1 ^ 0 ^ 0 = 1
//	g3 = 0 ^ 0 ^ 0 = 0
func TestRCPCTetraEncoderMatchesGeneratorPolys(t *testing.T) {
	in := []byte{1, 0, 0, 0, 0, 0, 0, 0, 0}
	out := EncodeRCPCTetraMother(in)
	if out[3] != 1 || out[4] != 1 || out[5] != 0 {
		t.Errorf("second triple = (%d, %d, %d), want (1, 1, 0)", out[3], out[4], out[5])
	}
}

// TestRCPCTetraPunctureRate23RoundTrip: rate-8/12 (= 2/3)
// puncturing scheme used for class-1 bits in the normal speech
// channel. Encode 8 info bits + 4 tail bits, mother-code produces
// 36 bits; puncture to 12 bits (since 8 input bits at rate 2/3
// give 12 output bits, here we test 12 mother bits → 8 punctured
// bits scaled by the 2 input bits per period). Round-trip back
// through depuncture + Viterbi must recover the original info.
func TestRCPCTetraPunctureRate23RoundTrip(t *testing.T) {
	// 8 information bits + 4 tail bits = 12 stages.
	info := []byte{1, 0, 1, 1, 0, 1, 0, 0}
	stages := len(info) + 4
	in := make([]byte, stages)
	copy(in, info)
	mother := EncodeRCPCTetraMother(in)
	if len(mother) != 36 {
		t.Fatalf("mother length = %d, want 36", len(mother))
	}
	// Rate 2/3: 12 input → 18 output. (We round to whole periods.)
	// k3 must be a multiple of t=3 for clean tiling, and the
	// punctured length must not exceed the mother length scaled by
	// the rate. For 12 stages and rate 2/3 we get 18 output bits.
	const k3 = 18
	punctured := PunctureRCPCTetra(mother, RCPCTetraPeriod23, RCPCTetraPuncture23, k3)
	if len(punctured) != k3 {
		t.Fatalf("punctured length = %d, want %d", len(punctured), k3)
	}
	depunctured := DepunctureRCPCTetra(punctured, RCPCTetraPeriod23, RCPCTetraPuncture23, len(mother))
	out, _ := DecodeRCPCTetraMother(depunctured, stages)
	for i, want := range in {
		if out[i] != want {
			t.Errorf("bit %d: got %d, want %d (rate 2/3 round-trip)", i, out[i], want)
		}
	}
}

// TestRCPCTetraPunctureRate818RoundTrip: rate-8/18 scheme for
// class-2 bits in normal speech. 8 input bits → 24 mother bits →
// 18 punctured bits.
func TestRCPCTetraPunctureRate818RoundTrip(t *testing.T) {
	info := []byte{1, 0, 1, 0, 1, 1, 0, 1}
	stages := len(info) + 4
	in := make([]byte, stages)
	copy(in, info)
	mother := EncodeRCPCTetraMother(in)
	if len(mother) != 36 {
		t.Fatalf("mother length = %d, want 36", len(mother))
	}
	const k3 = 27 // 12 stages * 9/(12/3) hmm; just take a multiple of t=9 that fits
	punctured := PunctureRCPCTetra(mother, RCPCTetraPeriod818, RCPCTetraPuncture818, k3)
	depunctured := DepunctureRCPCTetra(punctured, RCPCTetraPeriod818, RCPCTetraPuncture818, len(mother))
	out, _ := DecodeRCPCTetraMother(depunctured, stages)
	for i, want := range in {
		if out[i] != want {
			t.Errorf("bit %d: got %d, want %d (rate 8/18 round-trip)", i, out[i], want)
		}
	}
}

// TestRCPCTetraPunctureRate817RoundTrip: rate-8/17 scheme used in
// frame-stealing mode.
func TestRCPCTetraPunctureRate817RoundTrip(t *testing.T) {
	info := []byte{1, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 1}
	stages := len(info) + 4
	in := make([]byte, stages)
	copy(in, info)
	mother := EncodeRCPCTetraMother(in)
	// Use k3 = 34 (2 full periods of 17 each, well within the
	// 48-bit mother for 16-stage input).
	const k3 = 34
	punctured := PunctureRCPCTetra(mother, RCPCTetraPeriod817, RCPCTetraPuncture817, k3)
	depunctured := DepunctureRCPCTetra(punctured, RCPCTetraPeriod817, RCPCTetraPuncture817, len(mother))
	out, _ := DecodeRCPCTetraMother(depunctured, stages)
	for i, want := range in {
		if out[i] != want {
			t.Errorf("bit %d: got %d, want %d (rate 8/17 round-trip)", i, out[i], want)
		}
	}
}

// TestRCPCTetraPunctureCorrectsSingleErrorOnRate23: with the
// rate-2/3 puncture in place, flipping one received bit still
// recovers the original info (K=5 1/3-rate has plenty of free
// distance to correct single errors).
func TestRCPCTetraPunctureCorrectsSingleErrorOnRate23(t *testing.T) {
	info := []byte{1, 0, 1, 1, 0, 1, 0, 0}
	stages := len(info) + 4
	in := make([]byte, stages)
	copy(in, info)
	mother := EncodeRCPCTetraMother(in)
	const k3 = 18
	punctured := PunctureRCPCTetra(mother, RCPCTetraPeriod23, RCPCTetraPuncture23, k3)
	// Flip one bit in the punctured stream.
	punctured[5] ^= 1
	depunctured := DepunctureRCPCTetra(punctured, RCPCTetraPeriod23, RCPCTetraPuncture23, len(mother))
	out, metric := DecodeRCPCTetraMother(depunctured, stages)
	if metric == 0 {
		t.Error("expected non-zero metric on a corrupted punctured channel")
	}
	for i, want := range in {
		if out[i] != want {
			t.Errorf("bit %d: got %d, want %d (rate 2/3 + 1 error)", i, out[i], want)
		}
	}
}

// TestRCPCTetraPunctureScheduleSanity: the puncture maps must list
// strictly-increasing 1-indexed positions all <= period.
func TestRCPCTetraPunctureScheduleSanity(t *testing.T) {
	cases := []struct {
		name    string
		pattern []int
		period  int
	}{
		{"rate 2/3", RCPCTetraPuncture23, RCPCTetraPeriod23},
		{"rate 8/18", RCPCTetraPuncture818, RCPCTetraPeriod818},
		{"rate 8/17", RCPCTetraPuncture817, RCPCTetraPeriod817},
	}
	for _, tc := range cases {
		for i, p := range tc.pattern {
			if p < 1 || p > tc.period {
				t.Errorf("%s: pattern[%d] = %d is outside 1..%d", tc.name, i, p, tc.period)
			}
			if i > 0 && p <= tc.pattern[i-1] {
				t.Errorf("%s: pattern[%d] = %d is not strictly greater than pattern[%d] = %d",
					tc.name, i, p, i-1, tc.pattern[i-1])
			}
		}
	}
}
