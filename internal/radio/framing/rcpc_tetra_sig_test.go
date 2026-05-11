package framing

import "testing"

func TestRCPCTetraSigMotherRoundTripCleanChannel(t *testing.T) {
	info := []byte{
		1, 0, 1, 1, 0, 1, 0, 0, 1, 1, 1, 0, 0, 0, 1, 1,
		1, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 1, 0, 0, 1, 1,
	}
	stages := len(info) + 4
	in := make([]byte, stages)
	copy(in, info)
	channel := EncodeRCPCTetraSigMother(in)
	if len(channel) != 4*stages {
		t.Fatalf("encoded length = %d, want %d", len(channel), 4*stages)
	}
	out, metric := DecodeRCPCTetraSigMother(channel, stages)
	if metric != 0 {
		t.Errorf("clean-channel metric = %d, want 0", metric)
	}
	for i, want := range in {
		if out[i] != want {
			t.Errorf("bit %d: got %d, want %d", i, out[i], want)
		}
	}
}

func TestRCPCTetraSigMotherCorrectsSingleBitError(t *testing.T) {
	info := []byte{1, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0, 1, 1, 0, 0, 1}
	stages := len(info) + 4
	in := make([]byte, stages)
	copy(in, info)
	channel := EncodeRCPCTetraSigMother(in)
	channel[15] ^= 1
	out, metric := DecodeRCPCTetraSigMother(channel, stages)
	if metric == 0 {
		t.Error("expected non-zero metric on a corrupted channel")
	}
	for i, want := range in {
		if out[i] != want {
			t.Errorf("bit %d: got %d, want %d", i, out[i], want)
		}
	}
}

func TestRCPCTetraSigEncoderInitialStateZero(t *testing.T) {
	in := []byte{1, 0, 0, 0, 0, 0, 0, 0, 0}
	out := EncodeRCPCTetraSigMother(in)
	// First tuple: input=1, all state bits 0 → g1=1, g2=1, g3=1, g4=1
	if out[0] != 1 || out[1] != 1 || out[2] != 1 || out[3] != 1 {
		t.Errorf("first tuple = (%d, %d, %d, %d), want (1, 1, 1, 1)", out[0], out[1], out[2], out[3])
	}
}

func TestRCPCTetraSigEncoderMatchesGeneratorPolys(t *testing.T) {
	// Second tuple: input=0, d1=1, others 0.
	//   g1 = 0 ^ 1 ^ 0 = 1
	//   g2 = 0 ^ 0 ^ 0 ^ 0 = 0
	//   g3 = 0 ^ 1 ^ 0 ^ 0 = 1
	//   g4 = 0 ^ 1 ^ 0 ^ 0 = 1
	in := []byte{1, 0, 0, 0, 0, 0, 0, 0, 0}
	out := EncodeRCPCTetraSigMother(in)
	if out[4] != 1 || out[5] != 0 || out[6] != 1 || out[7] != 1 {
		t.Errorf("second tuple = (%d, %d, %d, %d), want (1, 0, 1, 1)", out[4], out[5], out[6], out[7])
	}
}

// TestRCPCTetraSigRate23RoundTrip exercises the SCH/HD / BSCH /
// SCH/F path: 8 input bits + 4 tail → 12 stages → 48 mother bits →
// puncture to 18 channel bits (rate 2/3).
func TestRCPCTetraSigRate23RoundTrip(t *testing.T) {
	info := []byte{1, 0, 1, 1, 0, 1, 0, 0}
	stages := len(info) + 4
	in := make([]byte, stages)
	copy(in, info)
	mother := EncodeRCPCTetraSigMother(in)
	if len(mother) != 48 {
		t.Fatalf("mother length = %d, want 48", len(mother))
	}
	const k3 = 18
	punctured := PunctureRCPCTetraSig(mother, RCPCTetraSigPuncture23, k3, nil)
	depunctured := DepunctureRCPCTetraSig(punctured, RCPCTetraSigPuncture23, len(mother), nil)
	out, _ := DecodeRCPCTetraSigMother(depunctured, stages)
	for i, want := range in {
		if out[i] != want {
			t.Errorf("bit %d: got %d, want %d (rate 2/3 round-trip)", i, out[i], want)
		}
	}
}

// TestRCPCTetraSigRate13RoundTrip exercises the rate-1/3 stronger-
// protection path: 8 input bits + 4 tail → 12 stages → 48 mother
// bits → puncture to 36 channel bits.
func TestRCPCTetraSigRate13RoundTrip(t *testing.T) {
	info := []byte{1, 1, 0, 0, 1, 0, 1, 1}
	stages := len(info) + 4
	in := make([]byte, stages)
	copy(in, info)
	mother := EncodeRCPCTetraSigMother(in)
	const k3 = 36
	punctured := PunctureRCPCTetraSig(mother, RCPCTetraSigPuncture13, k3, nil)
	depunctured := DepunctureRCPCTetraSig(punctured, RCPCTetraSigPuncture13, len(mother), nil)
	out, _ := DecodeRCPCTetraSigMother(depunctured, stages)
	for i, want := range in {
		if out[i] != want {
			t.Errorf("bit %d: got %d, want %d (rate 1/3 round-trip)", i, out[i], want)
		}
	}
}

func TestRCPCTetraSigRate23CorrectsSingleError(t *testing.T) {
	info := []byte{1, 0, 1, 1, 0, 1, 0, 0}
	stages := len(info) + 4
	in := make([]byte, stages)
	copy(in, info)
	mother := EncodeRCPCTetraSigMother(in)
	const k3 = 18
	punctured := PunctureRCPCTetraSig(mother, RCPCTetraSigPuncture23, k3, nil)
	punctured[7] ^= 1
	depunctured := DepunctureRCPCTetraSig(punctured, RCPCTetraSigPuncture23, len(mother), nil)
	out, metric := DecodeRCPCTetraSigMother(depunctured, stages)
	if metric == 0 {
		t.Error("expected non-zero metric on a corrupted punctured channel")
	}
	for i, want := range in {
		if out[i] != want {
			t.Errorf("bit %d: got %d, want %d (rate 2/3 + 1 error)", i, out[i], want)
		}
	}
}

// TestRCPCTetraSigIndexShifts: sanity check the spec's special-rate
// index shifts produce monotone-increasing i values within their
// declared output ranges.
func TestRCPCTetraSigIndexShifts(t *testing.T) {
	// Rate 292/432: j = 1..432, i = j + (j-1) div 65
	last := 0
	for j := 1; j <= 432; j++ {
		i := RCPCTetraSigIndexShift292_432(j)
		if i <= last {
			t.Errorf("292/432: i(%d) = %d is not strictly greater than i(%d) = %d", j, i, j-1, last)
		}
		last = i
	}
	// Rate 148/432
	last = 0
	for j := 1; j <= 432; j++ {
		i := RCPCTetraSigIndexShift148_432(j)
		if i <= last {
			t.Errorf("148/432: i(%d) = %d is not strictly greater than i(%d) = %d", j, i, j-1, last)
		}
		last = i
	}
}

func TestRCPCTetraSigPunctureScheduleSanity(t *testing.T) {
	cases := []struct {
		name    string
		pattern []int
	}{
		{"rate 2/3", RCPCTetraSigPuncture23},
		{"rate 1/3", RCPCTetraSigPuncture13},
	}
	for _, tc := range cases {
		for i, p := range tc.pattern {
			if p < 1 || p > 8 {
				t.Errorf("%s: pattern[%d] = %d is outside 1..8 (Period)", tc.name, i, p)
			}
			if i > 0 && p <= tc.pattern[i-1] {
				t.Errorf("%s: pattern[%d] = %d is not strictly greater than pattern[%d] = %d",
					tc.name, i, p, i-1, tc.pattern[i-1])
			}
		}
	}
}
