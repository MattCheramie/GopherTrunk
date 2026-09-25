package conventional

import (
	"math"
	"math/bits"
	"math/rand"
	"strconv"
	"testing"
)

// refDCSWord builds the 23-bit DCS word for a 9-bit code from the DCS
// parity equations as published (onfreq.com/syntorx/dcs.html; the same
// equations SDRangel's NFMModDCS::setDCS transmits). Bit i of the result
// is the i-th bit ON AIR: C0..C8 (the code, least significant bit
// first), the fixed 0 0 1, then P0..P10. It is written independently of
// dcsCodewordFromOctal on purpose — the old encoder and the old test
// synthesizer shared one wrong layout, so every test passed while no
// real radio could ever match (issue #1184).
func refDCSWord(c uint32) uint32 {
	par := func(v uint32) uint32 { return uint32(bits.OnesCount32(v) & 1) }
	p := [11]uint32{
		par(c & 0b10011111),    // P0  = C0+C1+C2+C3+C4+C7
		1 ^ par(c&0b100111110), // P1  = NOT(C1+C2+C3+C4+C5+C8)
		par(c & 0b11100011),    // P2  = C0+C1+C5+C6+C7
		1 ^ par(c&0b111000110), // P3  = NOT(C1+C2+C6+C7+C8)
		1 ^ par(c&0b100010011), // P4  = NOT(C0+C1+C4+C8)
		1 ^ par(c&0b10111001),  // P5  = NOT(C0+C3+C4+C5+C7)
		par(c & 0b111101101),   // P6  = C0+C2+C3+C5+C6+C7+C8
		par(c & 0b111011010),   // P7  = C1+C3+C4+C6+C7+C8
		par(c & 0b110110100),   // P8  = C2+C4+C5+C7+C8
		1 ^ par(c&0b101101000), // P9  = NOT(C3+C5+C6+C8)
		1 ^ par(c&0b1001111),   // P10 = NOT(C0+C1+C2+C3+C6)
	}
	w := c&0x1FF | 1<<11
	for i, b := range p {
		w |= b << uint(12+i)
	}
	return w
}

// TestDCSCodewordMatchesPublishedParity pins every one of the 512 codes
// against the published parity equations, and code 023 against its
// literal on-air word 0x763813.
func TestDCSCodewordMatchesPublishedParity(t *testing.T) {
	if got := dcsCodewordOrPanic(t, "023"); got != 0x763813 {
		t.Errorf("DCS 023 codeword = %#06x, want 0x763813", got)
	}
	for c := uint32(0); c < 512; c++ {
		code := strconv.FormatUint(uint64(c), 8)
		for len(code) < 3 {
			code = "0" + code
		}
		if got, want := dcsCodewordOrPanic(t, code), refDCSWord(c); got != want {
			t.Errorf("DCS %s codeword = %#06x, want %#06x", code, got, want)
		}
	}
}

// TestDCSCodewordAliasesMatchReferenceTables checks two properties of the
// real code that only the correct construction has, taken from the
// published DCS tables (SDRangel dcscodes.cpp): a code's word read from a
// different start bit is another code (023 ≡ 340 ≡ 766), and the
// complement of a code's word is another code (023 normal ≡ 047
// inverted).
func TestDCSCodewordAliasesMatchReferenceTables(t *testing.T) {
	rotationOf := func(a, b uint32, invert bool) bool {
		for _, r := range dcsRotations(a) {
			if invert {
				r = ^r & dcsCodewordMask
			}
			if r == b {
				return true
			}
		}
		return false
	}
	aliases := [][2]string{{"023", "340"}, {"023", "766"}, {"025", "025"}, {"054", "405"}, {"054", "675"}}
	for _, a := range aliases {
		if !rotationOf(dcsCodewordOrPanic(t, a[0]), dcsCodewordOrPanic(t, a[1]), false) {
			t.Errorf("DCS %s and %s should be rotations of one word", a[0], a[1])
		}
	}
	flips := [][2]string{{"023", "047"}, {"754", "116"}, {"627", "031"}}
	for _, f := range flips {
		if !rotationOf(dcsCodewordOrPanic(t, f[0]), dcsCodewordOrPanic(t, f[1]), true) {
			t.Errorf("inverted DCS %s should read as %s", f[0], f[1])
		}
	}
}

// synthDCSOnAir FM-modulates a DCS word as a transmitter does: bit 0
// first, NRZ at 134.4 baud, with a carrier offset, complex white noise at
// snrDb over the full sample rate, and a random start bit + sub-bit
// phase. invert flips the NRZ polarity.
func synthDCSOnAir(word uint32, invert bool, sampleHz, devHz, offsetHz, snrDb, seconds float64, seed int64) []complex64 {
	rng := rand.New(rand.NewSource(seed))
	sigma := math.Sqrt(math.Pow(10, -snrDb/10) / 2)
	spb := sampleHz / 134.4
	start := rng.Float64() * 23 * spb
	n := int(seconds * sampleHz)
	out := make([]complex64, n)
	phase := 0.0
	for i := range out {
		bit := int((float64(i)+start)/spb) % 23
		m := -1.0
		if (word>>uint(bit))&1 == 1 {
			m = 1
		}
		if invert {
			m = -m
		}
		phase += 2 * math.Pi * (devHz*m + offsetHz) / sampleHz
		out[i] = complex(
			float32(math.Cos(phase)+rng.NormFloat64()*sigma),
			float32(math.Sin(phase)+rng.NormFloat64()*sigma))
	}
	return out
}

// dcsPresentAfter feeds x in chunks and reports whether the detector is
// present on every chunk from warmup seconds on.
func dcsPresentAfter(d *DCSDetector, x []complex64, sampleHz, warmup float64) (always, ever bool) {
	const chunk = 16384
	always = true
	for s := 0; s+chunk <= len(x); s += chunk {
		got := d.Process(x[s : s+chunk])
		ever = ever || got
		if float64(s) >= warmup*sampleHz && !got {
			always = false
		}
	}
	return always, ever
}

// TestDCSDetectsOnAirWordAtSDRRate is the #1184 DCS regression: "DCS
// squelch does not detect any code, normal or inverted". A narrowband
// radio (~350 Hz of DCS deviation), 600 Hz off frequency, at the RTL-SDR
// rate the scanner feeds the detector.
func TestDCSDetectsOnAirWordAtSDRRate(t *testing.T) {
	const rate = 2_400_000
	for i, code := range []string{"023", "754", "411"} {
		c, _ := strconv.ParseUint(code, 8, 32)
		for _, invert := range []bool{false, true} {
			pol := DCSPolarityNormal
			if invert {
				pol = DCSPolarityInverted
			}
			d := NewDCSDetector(DCSConfig{SampleHz: rate, Code: code, Polarity: pol})
			x := synthDCSOnAir(refDCSWord(uint32(c)), invert, rate, 350, 600, 25, 1.5, int64(i))
			if always, _ := dcsPresentAfter(d, x, rate, dcsMinDwell.Seconds()); !always {
				t.Errorf("DCS %s (inverted=%v) not continuously detected after %v", code, invert, dcsMinDwell)
			}
		}
	}
}

// A different code, or a carrier without DCS, must never open the gate.
func TestDCSRejectsOtherCodeAndBareCarrierAtSDRRate(t *testing.T) {
	const rate = 2_400_000
	d := NewDCSDetector(DCSConfig{SampleHz: rate, Code: "023"})
	x := synthDCSOnAir(refDCSWord(0754), false, rate, 350, 600, 25, 1.5, 7)
	if _, ever := dcsPresentAfter(d, x, rate, 0); ever {
		t.Error("DCS 023 gate opened on a DCS 754 transmission")
	}
	d = NewDCSDetector(DCSConfig{SampleHz: rate, Code: "023"})
	x = genNoisyCTCSS(100, 0, rate, rate*3/2, 25, 8) // bare carrier + noise
	if _, ever := dcsPresentAfter(d, x, rate, 0); ever {
		t.Error("DCS 023 gate opened on a carrier with no DCS")
	}
}

// With both polarities accepted the detector reports which one matched
// (the #1184 instrument that pinned N = false, I = true on air).
func TestDCSReportsMatchedPolarity(t *testing.T) {
	const rate = 240_000
	for _, invert := range []bool{false, true} {
		d := NewDCSDetector(DCSConfig{SampleHz: rate, Code: "023", Polarity: DCSPolarityBoth})
		x := synthDCSOnAir(refDCSWord(0o023), invert, rate, 350, 0, 25, 1.5, 3)
		if always, _ := dcsPresentAfter(d, x, rate, dcsMinDwell.Seconds()); !always {
			t.Fatalf("inverted=%v: not detected", invert)
		}
		if d.Inverted() != invert {
			t.Errorf("Inverted() = %v, want %v", d.Inverted(), invert)
		}
	}
}
