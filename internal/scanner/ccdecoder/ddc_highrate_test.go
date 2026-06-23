package ccdecoder

import (
	"math"
	"math/rand"
	"testing"
)

// Regression harness for issue #771. The reporter sees a P25 control channel
// at −812.5 kHz lock cleanly when replayed from a 2.5 MS/s capture but fail to
// lock (AGC stuck ~10–20× above target, never converging) when replayed from a
// 10 MS/s capture of the same site — in pure offline replay, with no live
// daemon or USB in the loop. `gophertrunk replay -tune-hz` runs through this
// package's Downconverter (NewDownconverterWithOffset), so any rate-dependent
// level/alias defect would live here. These tests drive the same path at both
// rates and assert the extracted channel is equivalent, so a real defect fails
// first and a fix keeps them green.

// c4fmSymbols returns n deterministic P25 C4FM symbols drawn from the 4-level
// alphabet {−3,−1,+1,+3}. A fixed seed keeps the two-rate comparison apples to
// apples: both rates modulate the identical symbol stream.
func c4fmSymbols(n int, seed int64) []int {
	r := rand.New(rand.NewSource(seed))
	levels := []int{-3, -1, 1, 3}
	out := make([]int, n)
	for i := range out {
		out[i] = levels[r.Intn(len(levels))]
	}
	return out
}

// c4fmChannel synthesises a continuous-phase 4800-baud C4FM carrier at
// carrierHz offset, sampled at rateHz, from the supplied symbol stream. The
// instantaneous deviation is sym·600 Hz, i.e. the standard P25 ±1800 / ±600 Hz.
// Symbol boundaries are placed in absolute time so the same symbols line up
// across sample rates.
func c4fmChannel(symbols []int, carrierHz, rateHz, amp float64) []complex64 {
	const baud = 4800.0
	nSamp := int(float64(len(symbols)) / baud * rateHz)
	out := make([]complex64, nSamp)
	fmPhase := 0.0
	carrierW := 2 * math.Pi * carrierHz / rateHz
	for i := 0; i < nSamp; i++ {
		symIdx := int(float64(i) * baud / rateHz)
		if symIdx >= len(symbols) {
			symIdx = len(symbols) - 1
		}
		fmPhase += 2 * math.Pi * float64(symbols[symIdx]) * 600.0 / rateHz
		ph := fmPhase + carrierW*float64(i)
		out[i] = complex(float32(amp*math.Cos(ph)), float32(amp*math.Sin(ph)))
	}
	return out
}

// fmDiscRMS returns the RMS of an amplitude-normalised FM discriminator
// (angle of x[n]·conj(x[n−1])). For a C4FM channel this is the symbol-domain
// signal the receiver's matched filter and AGC operate on, so it is the level
// the AGC tries to drive to its target. A skip drops the filter startup
// transient.
func fmDiscRMS(x []complex64, skip int) float64 {
	if len(x) <= skip+1 {
		return 0
	}
	var sum float64
	cnt := 0
	for i := skip + 1; i < len(x); i++ {
		d := complex128(x[i]) * cmplxConj(complex128(x[i-1]))
		a := math.Atan2(imag(d), real(d))
		sum += a * a
		cnt++
	}
	if cnt == 0 {
		return 0
	}
	return math.Sqrt(sum / float64(cnt))
}

func cmplxConj(c complex128) complex128 { return complex(real(c), -imag(c)) }

// TestDownconverterC4FMLevelRateInvariant is the core #771 check: the same
// C4FM control channel, replayed through the production down-converter at
// 2.5 MS/s vs 10 MS/s, must reach the receiver at the same symbol-domain level.
// The AGC target is fixed for the 48 kHz channel rate, so a level that differs
// by an order of magnitude between rates is exactly the reported failure.
func TestDownconverterC4FMLevelRateInvariant(t *testing.T) {
	const (
		offsetHz = -812_500.0 // Mt Anakie, the reporter's strong tap
		amp      = 0.3        // well below full-scale; no clipping
	)
	symbols := c4fmSymbols(4800, 0x771) // ~1 s of symbols

	// 2.5 MS/s: no shared decimation; the proven-good path.
	lo := NewDownconverterWithOffset(2_500_000, DDCTargetRateHz, offsetHz).
		Process(nil, c4fmChannel(symbols, offsetHz, 2_500_000, amp))
	loLvl := fmDiscRMS(lo, 64)

	// 10 MS/s: the reported-failing path. Identical symbols, identical carrier.
	hi := NewDownconverterWithOffset(10_000_000, DDCTargetRateHz, offsetHz).
		Process(nil, c4fmChannel(symbols, offsetHz, 10_000_000, amp))
	hiLvl := fmDiscRMS(hi, 64)

	if loLvl == 0 || hiLvl == 0 {
		t.Fatalf("empty discriminator output (lo=%v hi=%v)", loLvl, hiLvl)
	}
	ratio := hiLvl / loLvl
	if ratio < 0.8 || ratio > 1.25 {
		t.Errorf("C4FM symbol level not rate-invariant: 2.5 MS/s=%.5f 10 MS/s=%.5f (ratio %.2f, want ~1.0). "+
			"A ratio near the reported 10–20× points at a level/gain defect in the high-rate replay path.",
			loLvl, hiLvl, ratio)
	}
}

// TestDownconverterRejectsWidebandNeighbours stresses the 10 MS/s path the way
// a real wideband capture does: the −812.5 kHz channel of interest plus strong
// off-channel carriers (a neighbour site, an adjacent service) and a center
// DC/LO spike, all of which the analog front end admits at 10 MS/s but not at
// 2.5 MS/s. None of that out-of-band energy may fold into the ±24 kHz output
// and inflate the level the AGC sees. The channel of interest must still arrive
// at essentially the same level as when it is alone.
func TestDownconverterRejectsWidebandNeighbours(t *testing.T) {
	const (
		rate     = 10_000_000.0
		offsetHz = -812_500.0
		amp      = 0.3
	)
	symbols := c4fmSymbols(4800, 0x771)

	clean := c4fmChannel(symbols, offsetHz, rate, amp)

	// Add wideband neighbours far stronger than the target, plus a DC spike.
	withNeighbours := append([]complex64(nil), clean...)
	addTone(withNeighbours, +937_500, rate, 1.0) // Mt Blackwood neighbour
	addTone(withNeighbours, +200_000, rate, 1.0) // close-in interferer
	addTone(withNeighbours, -1_500_000, rate, 1.0)
	addTone(withNeighbours, +3_000_000, rate, 1.0)
	addTone(withNeighbours, 0, rate, 1.5) // center DC/LO spike

	cleanLvl := fmDiscRMS(
		NewDownconverterWithOffset(rate, DDCTargetRateHz, offsetHz).Process(nil, clean), 64)
	stressLvl := fmDiscRMS(
		NewDownconverterWithOffset(rate, DDCTargetRateHz, offsetHz).Process(nil, withNeighbours), 64)

	if cleanLvl == 0 {
		t.Fatalf("empty discriminator output for clean channel")
	}
	ratio := stressLvl / cleanLvl
	if ratio < 0.8 || ratio > 1.25 {
		t.Errorf("wideband neighbours leaked into the channel: clean=%.5f with-neighbours=%.5f (ratio %.2f, want ~1.0). "+
			"Inadequate anti-alias rejection at high M would raise the level/noise the AGC sees.",
			cleanLvl, stressLvl, ratio)
	}
}

// addTone superimposes a complex exponential at freqHz onto buf in place.
func addTone(buf []complex64, freqHz, rateHz, amp float64) {
	w := 2 * math.Pi * freqHz / rateHz
	for i := range buf {
		buf[i] += complex(
			float32(amp*math.Cos(w*float64(i))),
			float32(amp*math.Sin(w*float64(i))),
		)
	}
}
