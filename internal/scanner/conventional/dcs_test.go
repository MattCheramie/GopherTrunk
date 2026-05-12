package conventional

import (
	"math"
	"math/bits"
	"testing"
)

// synthesizeDCSIQ generates an FM-modulated IQ stream that cycles
// the supplied 23-bit codeword as an NRZ stream at 134.4 baud. The
// deviation knob controls the FM index for the sub-audible tone; a
// few hundred Hz of peak deviation is typical on commercial radios.
func synthesizeDCSIQ(codeword uint32, sampleHz float64, devHz float64, nFrames int) []complex64 {
	const bitRate = 134.4
	samplesPerBit := sampleHz / bitRate
	totalSamples := int(float64(nFrames*23) * samplesPerBit)
	out := make([]complex64, totalSamples)
	phase := 0.0
	dt := 1.0 / sampleHz
	for i := range totalSamples {
		// Which bit are we in?
		bitIdx := int(float64(i)/samplesPerBit) % 23
		// Bits are emitted MSB-first to match the encoder's
		// "data in high bits" layout.
		bit := (codeword >> (22 - bitIdx)) & 1
		var modAmp float64
		if bit == 1 {
			modAmp = 1
		} else {
			modAmp = -1
		}
		// FM phase advance: integrate the modulating signal.
		phase += 2 * math.Pi * devHz * modAmp * dt
		out[i] = complex(float32(math.Cos(phase)), float32(math.Sin(phase)))
	}
	return out
}

func TestDCSCodewordFromOctal(t *testing.T) {
	// Round-trip a sample of codes through the encoder + the
	// framing-package decoder and confirm the 12 info bits match
	// the constructed pattern.
	cases := []struct {
		code     string
		wantBits uint16 // 9-bit code (high) | "100" (low)
	}{
		{"023", 0b000_010_011_100},
		{"754", 0b111_101_100_100},
		{"000", 0b000_000_000_100},
		{"777", 0b111_111_111_100},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			cw, err := dcsCodewordFromOctal(tc.code)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			// The framing package's encoder layout is [data | parity].
			// Bits 22..11 hold the data, 10..0 hold the 11 parity bits.
			data := uint16((cw >> 11) & 0xFFF)
			if data != tc.wantBits {
				t.Errorf("data bits = %012b, want %012b", data, tc.wantBits)
			}
		})
	}
}

func TestDCSCodewordRejectsBadInput(t *testing.T) {
	cases := []string{"", "12", "1234", "089", "abc"}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			if _, err := dcsCodewordFromOctal(c); err == nil {
				t.Errorf("expected error for %q", c)
			}
		})
	}
}

func TestDCSRotations(t *testing.T) {
	cw, err := dcsCodewordFromOctal("023")
	if err != nil {
		t.Fatal(err)
	}
	rots := dcsRotations(cw)
	if len(rots) != 46 {
		t.Errorf("expected 46 rotations (23 × 2 polarities), got %d", len(rots))
	}
	// The first entry must be the original codeword.
	if rots[0] != (cw & dcsCodewordMask) {
		t.Errorf("rotation[0] = %x, want %x", rots[0], cw)
	}
	// The second entry must be the bit-inverse polarity.
	if rots[1] != ((^cw) & dcsCodewordMask) {
		t.Errorf("rotation[1] = %x, want %x", rots[1], (^cw)&dcsCodewordMask)
	}
	// Every rotation should differ from the original by some non-
	// trivial amount (Golay codewords are not all-same-bit).
	for i := 2; i < len(rots); i += 2 {
		if rots[i] == rots[0] {
			t.Errorf("rotation %d duplicates rotation 0", i)
		}
	}
}

func TestDCSDetector_MatchesConfiguredCode(t *testing.T) {
	d := NewDCSDetector(DCSConfig{SampleHz: 48_000, Code: "023"})
	if d == nil {
		t.Fatal("constructor returned nil for valid config")
	}
	iq := synthesizeDCSIQ(dcsCodewordOrPanic(t, "023"), 48_000, 400, 4)
	if !d.Process(iq) {
		t.Error("detector failed to match configured DCS code")
	}
}

func TestDCSDetector_RejectsDifferentCode(t *testing.T) {
	d := NewDCSDetector(DCSConfig{SampleHz: 48_000, Code: "023"})
	// Transmit "754" but configure detection for "023". The two
	// codewords share no rotation, so detection must stay false.
	iq := synthesizeDCSIQ(dcsCodewordOrPanic(t, "754"), 48_000, 400, 4)
	if d.Process(iq) {
		t.Error("detector matched a different DCS code")
	}
}

func TestDCSDetector_RejectsSilence(t *testing.T) {
	d := NewDCSDetector(DCSConfig{SampleHz: 48_000, Code: "023"})
	iq := make([]complex64, 48_000)
	for i := range iq {
		iq[i] = complex(1, 0)
	}
	if d.Process(iq) {
		t.Error("detector matched on a silent carrier")
	}
}

func TestDCSDetector_MatchesInvertedPolarity(t *testing.T) {
	d := NewDCSDetector(DCSConfig{SampleHz: 48_000, Code: "023"})
	cw, _ := dcsCodewordFromOctal("023")
	inverted := (^cw) & dcsCodewordMask
	iq := synthesizeDCSIQ(inverted, 48_000, 400, 4)
	if !d.Process(iq) {
		t.Error("detector failed to match inverted-polarity DCS")
	}
}

func TestDCSDetector_ResetClearsState(t *testing.T) {
	d := NewDCSDetector(DCSConfig{SampleHz: 48_000, Code: "023"})
	iq := synthesizeDCSIQ(dcsCodewordOrPanic(t, "023"), 48_000, 400, 4)
	d.Process(iq)
	if !d.Present() {
		t.Fatal("test setup: detector never matched")
	}
	d.Reset()
	if d.Present() {
		t.Error("Present remained true after Reset")
	}
}

func TestDCSDetector_NilSafe(t *testing.T) {
	var d *DCSDetector
	if d.Process(nil) {
		t.Error("nil detector should not report matched")
	}
	d.Reset() // must not panic
}

func TestDCSDetector_BadConfigReturnsNil(t *testing.T) {
	cases := []DCSConfig{
		{},
		{SampleHz: 48_000},
		{SampleHz: 48_000, Code: ""},
		{SampleHz: 48_000, Code: "12"},
		{SampleHz: 48_000, Code: "089"},
		{SampleHz: 0, Code: "023"},
	}
	for i, c := range cases {
		if NewDCSDetector(c) != nil {
			t.Errorf("case %d: expected nil for %+v", i, c)
		}
	}
}

func TestDCSDetector_ToleratesSingleBitError(t *testing.T) {
	d := NewDCSDetector(DCSConfig{SampleHz: 48_000, Code: "023"})
	cw, _ := dcsCodewordFromOctal("023")
	// Flip a single bit in the cycled codeword. With distance
	// threshold of 2 (default), this should still match.
	corrupt := cw ^ (1 << 5)
	iq := synthesizeDCSIQ(corrupt, 48_000, 400, 4)
	if !d.Process(iq) {
		t.Error("detector failed to match a single-bit-corrupted DCS code")
	}
	// Sanity check: the corruption was within the threshold.
	if bits.OnesCount32(cw^corrupt) > d.distanceThreshold {
		t.Fatal("test setup: corruption exceeds threshold; tighten test")
	}
}

func dcsCodewordOrPanic(t *testing.T, code string) uint32 {
	t.Helper()
	cw, err := dcsCodewordFromOctal(code)
	if err != nil {
		t.Fatal(err)
	}
	return cw
}
