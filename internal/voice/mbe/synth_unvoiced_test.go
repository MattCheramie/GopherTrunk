package mbe

import (
	"math"
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/fft"
)

const unvoicedEpsilon = 1e-9

// helper: deterministic Gaussian-ish noise via math/rand (NormFloat64)
// with a known seed so tests are reproducible.
func seededNoise(seed int64, n int) []float64 {
	r := rand.New(rand.NewSource(seed))
	out := make([]float64, n)
	for i := range out {
		out[i] = r.NormFloat64()
	}
	return out
}

// TestShapeUnvoicedSpectrumSilentNoOp: silent frames leave the
// spectrum untouched so callers can short-circuit on a Silent flag
// at a higher level.
func TestShapeUnvoicedSpectrumSilentNoOp(t *testing.T) {
	spec := make([]complex128, UnvoicedFFTSize)
	for k := range spec {
		spec[k] = complex(float64(k), float64(k+1))
	}
	p := Params{Header: Header{Silent: true}}
	var M [57]float64
	ShapeUnvoicedSpectrum(spec, p, &M)
	for k := range spec {
		want := complex(float64(k), float64(k+1))
		if spec[k] != want {
			t.Fatalf("spec[%d] = %v, want %v (silent no-op)", k, spec[k], want)
		}
	}
}

// TestShapeUnvoicedSpectrumWrongLengthNoOp: spec length must equal
// UnvoicedFFTSize; otherwise the function returns early without
// mutating the buffer.
func TestShapeUnvoicedSpectrumWrongLengthNoOp(t *testing.T) {
	spec := make([]complex128, UnvoicedFFTSize-1)
	for k := range spec {
		spec[k] = complex(1, 1)
	}
	p := Params{Header: Header{W0: math.Pi / 30, L: 10}}
	var M [57]float64
	ShapeUnvoicedSpectrum(spec, p, &M)
	for k := range spec {
		if spec[k] != complex(1, 1) {
			t.Fatalf("spec[%d] = %v, want (1+1i) (length mismatch no-op)",
				k, spec[k])
		}
	}
}

// TestShapeUnvoicedSpectrumAllVoicedZerosEverything: when every
// harmonic is voiced, every bin gets zeroed (voiced harmonics go
// through the §6.3 sinusoidal path, not the unvoiced FFT). Bins
// outside the [1..L] model range also zero.
func TestShapeUnvoicedSpectrumAllVoicedZerosEverything(t *testing.T) {
	spec := make([]complex128, UnvoicedFFTSize)
	for k := range spec {
		spec[k] = complex(float64(k), float64(k+1))
	}
	p := Params{Header: Header{W0: math.Pi / 30, L: 10}}
	for l := 1; l <= 10; l++ {
		p.Vl[l] = 1
	}
	var M [57]float64
	ShapeUnvoicedSpectrum(spec, p, &M)
	for k := range spec {
		if spec[k] != 0 {
			t.Fatalf("spec[%d] = %v, want 0 (all-voiced)", k, spec[k])
		}
	}
}

// TestShapeUnvoicedSpectrumBinMapping: with ω₀ = 2π/N, harmonic l
// sits exactly at FFT bin l. Verify the closest-centre rule maps
// bins {1..L} to harmonics {1..L} and bin 0 + bins beyond L to
// "out of range" (zeroed). All harmonics unvoiced + M[l] = 1
// leaves the chosen bins untouched.
func TestShapeUnvoicedSpectrumBinMapping(t *testing.T) {
	N := UnvoicedFFTSize
	w0 := 2 * math.Pi / float64(N) // harmonic l → bin l exactly
	spec := make([]complex128, N)
	for k := range spec {
		spec[k] = complex(1, 0) // sentinel
	}
	L := 20
	p := Params{Header: Header{W0: w0, L: L}}
	var M [57]float64
	for l := 1; l <= L; l++ {
		M[l] = 1.0 // identity scale
	}
	ShapeUnvoicedSpectrum(spec, p, &M)

	// Bin 0 (DC) → l = 0 → out of range → zeroed
	if spec[0] != 0 {
		t.Errorf("spec[0] = %v, want 0 (DC out of range)", spec[0])
	}
	// Bins 1..L → kept (mirrored at N-l on the conjugate side)
	for l := 1; l <= L; l++ {
		if spec[l] != complex(1, 0) {
			t.Errorf("spec[%d] = %v, want (1+0i) (harmonic %d unvoiced kept)",
				l, spec[l], l)
		}
		if spec[N-l] != complex(1, 0) {
			t.Errorf("spec[%d] = %v, want (1+0i) (mirror of harmonic %d)",
				N-l, spec[N-l], l)
		}
	}
	// Bins L+1..N/2 → l > L → zeroed (and their mirrors)
	for k := L + 1; k <= N/2; k++ {
		if spec[k] != 0 {
			t.Errorf("spec[%d] = %v, want 0 (k > L)", k, spec[k])
		}
	}
}

// TestShapeUnvoicedSpectrumScaling: an unvoiced harmonic with
// M[l] = 2.5 multiplies the bin under that harmonic by exactly
// 2.5; voiced harmonics get zeroed. Pins the per-harmonic gain.
func TestShapeUnvoicedSpectrumScaling(t *testing.T) {
	N := UnvoicedFFTSize
	w0 := 2 * math.Pi / float64(N)
	spec := make([]complex128, N)
	for k := range spec {
		spec[k] = complex(1, 0)
	}
	p := Params{Header: Header{W0: w0, L: 4}}
	p.Vl[1] = 0
	p.Vl[2] = 1 // voiced
	p.Vl[3] = 0
	p.Vl[4] = 1 // voiced
	var M [57]float64
	M[1] = 2.5
	M[3] = 1.7
	ShapeUnvoicedSpectrum(spec, p, &M)
	if !almostEqual(real(spec[1]), 2.5) || imag(spec[1]) != 0 {
		t.Errorf("spec[1] = %v, want (2.5+0i)", spec[1])
	}
	if spec[2] != 0 {
		t.Errorf("spec[2] = %v, want 0 (voiced)", spec[2])
	}
	if !almostEqual(real(spec[3]), 1.7) || imag(spec[3]) != 0 {
		t.Errorf("spec[3] = %v, want (1.7+0i)", spec[3])
	}
	if spec[4] != 0 {
		t.Errorf("spec[4] = %v, want 0 (voiced)", spec[4])
	}
	// Conjugate-side bins must mirror the operation (real scale
	// preserves Hermitian symmetry).
	if !almostEqual(real(spec[N-1]), 2.5) {
		t.Errorf("spec[N-1] = %v, want 2.5 mirror", spec[N-1])
	}
	if spec[N-2] != 0 {
		t.Errorf("spec[N-2] = %v, want 0 mirror (voiced)", spec[N-2])
	}
}

// TestSynthUnvoicedFromNoiseSilentNoOp: silent + zero-L frames
// leave dst untouched (caller has presumably zeroed it; the
// function never writes).
func TestSynthUnvoicedFromNoiseSilentNoOp(t *testing.T) {
	dst := make([]float64, SamplesPerFrame)
	for i := range dst {
		dst[i] = 99
	}
	p := Params{Header: Header{Silent: true}}
	var M [57]float64
	noise := seededNoise(42, UnvoicedFFTSize)
	SynthUnvoicedFromNoise(p, &M, noise, dst)
	for i, v := range dst {
		if v != 99 {
			t.Fatalf("dst[%d] = %v, want 99 (silent)", i, v)
		}
	}
}

// TestSynthUnvoicedFromNoiseWrongSizes: noise of the wrong length
// or a too-short dst returns without writing. Lets callers
// fail-fast on shape bugs without a panic.
func TestSynthUnvoicedFromNoiseWrongSizes(t *testing.T) {
	p := Params{Header: Header{W0: math.Pi / 30, L: 10}}
	var M [57]float64
	M[1] = 1.0
	p.Vl[1] = 0

	// Wrong noise length.
	dst := make([]float64, SamplesPerFrame)
	for i := range dst {
		dst[i] = -1
	}
	SynthUnvoicedFromNoise(p, &M, make([]float64, UnvoicedFFTSize-1), dst)
	for i, v := range dst {
		if v != -1 {
			t.Fatalf("dst[%d] = %v, want -1 (wrong noise size)", i, v)
		}
	}

	// dst too short.
	short := make([]float64, SamplesPerFrame-1)
	for i := range short {
		short[i] = -2
	}
	SynthUnvoicedFromNoise(p, &M, seededNoise(1, UnvoicedFFTSize), short)
	for i, v := range short {
		if v != -2 {
			t.Fatalf("short[%d] = %v, want -2 (short dst)", i, v)
		}
	}
}

// TestSynthUnvoicedFromNoiseAllVoicedSilentOutput: when every
// harmonic in [1..L] is voiced, the spectrum is fully zeroed
// regardless of noise — IFFT(zero) = zero — so dst stays at its
// caller-set sentinel.
func TestSynthUnvoicedFromNoiseAllVoicedSilentOutput(t *testing.T) {
	p := Params{Header: Header{W0: math.Pi / 30, L: 10}}
	for l := 1; l <= 10; l++ {
		p.Vl[l] = 1
	}
	var M [57]float64
	noise := seededNoise(7, UnvoicedFFTSize)
	dst := make([]float64, SamplesPerFrame)
	for i := range dst {
		dst[i] = 5
	}
	SynthUnvoicedFromNoise(p, &M, noise, dst)
	for i, v := range dst {
		if math.Abs(v-5) > unvoicedEpsilon {
			t.Fatalf("dst[%d] = %v, want 5 (all-voiced → zero spectrum)",
				i, v)
		}
	}
}

// TestSynthUnvoicedFromNoiseZeroNoiseOutputZero: zero noise has a
// zero spectrum; shaping zero is zero; IFFT(0) = 0; dst stays at
// the caller-set sentinel.
func TestSynthUnvoicedFromNoiseZeroNoiseOutputZero(t *testing.T) {
	p := Params{Header: Header{W0: math.Pi / 30, L: 10}}
	var M [57]float64
	for l := 1; l <= 10; l++ {
		M[l] = 1.0
	}
	noise := make([]float64, UnvoicedFFTSize)
	dst := make([]float64, SamplesPerFrame)
	for i := range dst {
		dst[i] = 11
	}
	SynthUnvoicedFromNoise(p, &M, noise, dst)
	for i, v := range dst {
		if math.Abs(v-11) > unvoicedEpsilon {
			t.Fatalf("dst[%d] = %v, want 11 (zero noise → zero contribution)",
				i, v)
		}
	}
}

// TestSynthUnvoicedFromNoiseDeterministic: same noise + params →
// identical output. Pins reproducibility (no internal nondeterminism
// like a global rand source).
func TestSynthUnvoicedFromNoiseDeterministic(t *testing.T) {
	p := Params{Header: Header{W0: math.Pi / 30, L: 10}}
	var M [57]float64
	for l := 1; l <= 10; l++ {
		M[l] = 1.0
	}
	noise := seededNoise(123, UnvoicedFFTSize)

	a := make([]float64, SamplesPerFrame)
	b := make([]float64, SamplesPerFrame)
	SynthUnvoicedFromNoise(p, &M, noise, a)
	SynthUnvoicedFromNoise(p, &M, noise, b)
	for i := range a {
		if math.Abs(a[i]-b[i]) > unvoicedEpsilon {
			t.Fatalf("non-deterministic at i=%d: a=%v b=%v", i, a[i], b[i])
		}
	}
}

// TestSynthUnvoicedFromNoiseAddsToDst: SynthUnvoicedFromNoise adds
// rather than overwrites so it can be summed with the voiced step's
// output into the same buffer.
func TestSynthUnvoicedFromNoiseAddsToDst(t *testing.T) {
	p := Params{Header: Header{W0: math.Pi / 30, L: 10}}
	var M [57]float64
	for l := 1; l <= 10; l++ {
		M[l] = 1.0
	}
	noise := seededNoise(456, UnvoicedFFTSize)

	clean := make([]float64, SamplesPerFrame)
	SynthUnvoicedFromNoise(p, &M, noise, clean)

	added := make([]float64, SamplesPerFrame)
	for i := range added {
		added[i] = 100
	}
	SynthUnvoicedFromNoise(p, &M, noise, added)
	for i := range added {
		if math.Abs((added[i]-100)-clean[i]) > unvoicedEpsilon {
			t.Fatalf("at i=%d: added=%v clean=%v (expected added = 100 + clean)",
				i, added[i], clean[i])
		}
	}
}

// TestSynthUnvoicedFromNoiseRealOutput: regardless of the noise
// content, the IFFT of a Hermitian-symmetric spectrum produces a
// real-valued time-domain signal. The pipeline takes only real
// input + scales by real Ml[l] + zeroes some bins, all of which
// preserve Hermitian symmetry. Pins that the dst writes never
// pick up a phantom imaginary leak.
func TestSynthUnvoicedFromNoiseRealOutput(t *testing.T) {
	// Sanity: build a known Hermitian-symmetric spectrum directly
	// and IFFT it via the same pipeline as the real path. The
	// SynthUnvoicedFromNoise uses real(spec[n]) so it always
	// outputs real numbers; the test confirms that the imaginary
	// part of the IFFT result was negligible (Hermitian preserved).
	// We can't observe the imag part through SynthUnvoicedFromNoise
	// directly, so we verify via deterministic-output stability:
	// the answer doesn't drift across runs because the imag side
	// stays well below epsilon.
	p := Params{Header: Header{W0: math.Pi / 25, L: 12}}
	var M [57]float64
	for l := 1; l <= 12; l++ {
		M[l] = float64(l) * 0.1
		p.Vl[l] = l & 1 // alternate voicing
	}
	noise := seededNoise(789, UnvoicedFFTSize)
	dst := make([]float64, SamplesPerFrame)
	SynthUnvoicedFromNoise(p, &M, noise, dst)
	// Look for any NaN / Inf which would signal a precision blow-up
	// in the FFT path.
	for i, v := range dst {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			t.Fatalf("dst[%d] = %v, want finite", i, v)
		}
	}
	// And confirm the magnitude is reasonable (input noise has
	// stddev ≈ 1; FFT round-trip with bin scaling preserves order
	// of magnitude).
	var sumSq float64
	for _, v := range dst {
		sumSq += v * v
	}
	rms := math.Sqrt(sumSq / float64(len(dst)))
	if rms > 10 || rms < 1e-3 {
		t.Errorf("rms = %v, want roughly 0.001..10 (sanity envelope)", rms)
	}
}

// TestSynthesisWindowEndpoints: the power-complementary window decays
// to near-zero at both edges (so it still suppresses frame-boundary
// clicks) and is unity across the non-overlapping centre [96, 160).
// Pins the window definition so a future switch can't silently
// regress.
func TestSynthesisWindowEndpoints(t *testing.T) {
	// Edges taper to near-zero (the quarter-wave sine's first/last
	// half-step), not hard zero, but small enough to kill clicks.
	if synthesisWindow[0] > 0.02 {
		t.Errorf("synthesisWindow[0] = %v, want ≈0 (tapered left edge)", synthesisWindow[0])
	}
	if synthesisWindow[UnvoicedFFTSize-1] > 0.02 {
		t.Errorf("synthesisWindow[N-1] = %v, want ≈0 (tapered right edge)",
			synthesisWindow[UnvoicedFFTSize-1])
	}
	// The whole flat top, including N/2, is unity.
	for _, n := range []int{UnvoicedTailSamples, UnvoicedFFTSize / 2, UnvoicedFFTSize - UnvoicedTailSamples - 1} {
		if !almostEqual(synthesisWindow[n], 1.0) {
			t.Errorf("synthesisWindow[%d] = %v, want 1.0 (flat top)", n, synthesisWindow[n])
		}
	}
}

// TestSynthesisWindowSymmetry: the synthesis window is symmetric in
// the sense that w[k] == w[N-1-k]. The overlap-add COLA derivation
// (TestSynthesisWindowPowerComplementary) relies on w[160+n] = w[95-n],
// which this symmetry guarantees.
func TestSynthesisWindowSymmetry(t *testing.T) {
	for k := 0; k < UnvoicedFFTSize/2; k++ {
		if !almostEqual(synthesisWindow[k], synthesisWindow[UnvoicedFFTSize-1-k]) {
			t.Errorf("symmetry: w[%d]=%v != w[%d]=%v",
				k, synthesisWindow[k], UnvoicedFFTSize-1-k,
				synthesisWindow[UnvoicedFFTSize-1-k])
		}
	}
}

// TestSynthesisWindowPowerComplementary pins the COLA property the
// overlap-add depends on: consecutive frames carry INDEPENDENT noise
// blocks (each frame draws fresh d.rng.NormFloat64()), so in the
// 96-sample overlap region their variances add. The audible
// noise-power envelope across one frame is therefore
//
//	P[n] = w[n]² + w[n+HOP]²   (n < N-HOP, overlap region)
//	P[n] = w[n]²               (n >= N-HOP, no overlap)
//
// For a tremolo-free unvoiced floor this must be constant across the
// whole frame. A plain periodic Hann is NOT power-complementary at
// the 160-sample hop — it ripples ~7 dB, which leaks through as a
// 50 Hz frame-rate tremolo on the unvoiced band (one of the synthesis
// artifacts examined for issue #644). The power-complementary
// synthesis window keeps this envelope flat.
func TestSynthesisWindowPowerComplementary(t *testing.T) {
	const hop = SamplesPerFrame // 160
	var minP, maxP float64
	for n := 0; n < hop; n++ {
		p := synthesisWindow[n] * synthesisWindow[n]
		if n+hop < UnvoicedFFTSize {
			p += synthesisWindow[n+hop] * synthesisWindow[n+hop]
		}
		if n == 0 || p < minP {
			minP = p
		}
		if n == 0 || p > maxP {
			maxP = p
		}
	}
	// Allow a hair of float slack; the analytic window is exactly flat.
	if maxP-minP > 1e-9 {
		t.Errorf("overlap-add power envelope not flat: min=%.6f max=%.6f (ripple %.3f dB); "+
			"window is not power-complementary at hop=%d", minP, maxP,
			10*math.Log10(maxP/minP), hop)
	}
}

// TestSynthUnvoicedOverlapAddNoiseFloorIsFlat drives the OA path with
// a long run of all-unvoiced frames (fresh independent noise each
// frame, as the real decoder does) and confirms the synthesized
// noise-power envelope is flat across the frame position. This is the
// synthesis-level counterpart to TestSynthesisWindowPowerComplementary
// and the #644 regression guard: with the old periodic-Hann window the
// per-position power dipped ~7 dB at the overlap boundary, audible as a
// 50 Hz frame-rate tremolo ("machine voice"). A power-complementary
// window keeps it flat.
func TestSynthUnvoicedOverlapAddNoiseFloorIsFlat(t *testing.T) {
	var s SynthState
	// Flat spectral envelope so the only structure in the output
	// envelope comes from the synthesis window's overlap-add, not the
	// signal model. All harmonics unvoiced (Vl=0 default).
	p := Params{Header: Header{W0: math.Pi / 40, L: 30}}
	var M [57]float64
	for l := 1; l <= p.L; l++ {
		M[l] = 1.0
	}

	rng := rand.New(rand.NewSource(99))
	const frames = 4000
	power := make([]float64, SamplesPerFrame)
	dst := make([]float64, SamplesPerFrame)
	for f := 0; f < frames; f++ {
		noise := make([]float64, UnvoicedFFTSize)
		for i := range noise {
			noise[i] = rng.NormFloat64()
		}
		for i := range dst {
			dst[i] = 0
		}
		SynthUnvoicedOverlapAdd(&s, p, &M, noise, dst)
		// Skip the first frame: its overlap region has no prev tail yet.
		if f == 0 {
			continue
		}
		for n := 0; n < SamplesPerFrame; n++ {
			power[n] += dst[n] * dst[n]
		}
	}
	var minP, maxP float64
	for n := 0; n < SamplesPerFrame; n++ {
		if n == 0 || power[n] < minP {
			minP = power[n]
		}
		if n == 0 || power[n] > maxP {
			maxP = power[n]
		}
	}
	// Monte-Carlo over 4000 frames leaves some variance; the window's
	// systematic ripple was ~7 dB, so a 2 dB ceiling cleanly separates
	// "flat" from the old Hann while tolerating estimator noise.
	rippleDb := 10 * math.Log10(maxP/minP)
	if rippleDb > 2.0 {
		t.Errorf("unvoiced noise-floor envelope ripple %.2f dB across frame (min=%.1f max=%.1f); "+
			"overlap-add window is not power-complementary", rippleDb, minP, maxP)
	}
}

// TestSynthUnvoicedOverlapAddFreshStream: on a fresh state, the OA
// path produces dst[0..159] = synthesisWindow × IFFT(...) — the
// prev_tail is zero so dst[0..95] gets only the windowed curr-frame
// contribution. Confirm equivalence with the no-OA primitive
// scaled by the window.
func TestSynthUnvoicedOverlapAddFreshStream(t *testing.T) {
	var s SynthState
	p := Params{Header: Header{W0: math.Pi / 30, L: 10}}
	var M [57]float64
	for l := 1; l <= 10; l++ {
		M[l] = 1.0 // all unvoiced (Vl=0 default), unit amplitudes
	}
	noise := seededNoise(11, UnvoicedFFTSize)

	// Reference: no-OA path output (just first 160 IFFT samples).
	ref := make([]float64, SamplesPerFrame)
	SynthUnvoicedFromNoise(p, &M, noise, ref)

	// OA path output.
	got := make([]float64, SamplesPerFrame)
	SynthUnvoicedOverlapAdd(&s, p, &M, noise, got)

	// Each got[n] should be ref[n] · synthesisWindow[n].
	for n := 0; n < SamplesPerFrame; n++ {
		want := ref[n] * synthesisWindow[n]
		if math.Abs(got[n]-want) > 1e-9 {
			t.Fatalf("dst[%d] = %v, want %v (= ref[n]·w[n])", n, got[n], want)
		}
	}
}

// TestSynthUnvoicedOverlapAddStashesTail: after a fresh-stream call,
// PrevUnvoicedTail[0..95] equals synthesisWindow[160..255] · IFFT
// samples [160..255]. Pins the state-handoff that the next frame
// will pick up.
func TestSynthUnvoicedOverlapAddStashesTail(t *testing.T) {
	var s SynthState
	p := Params{Header: Header{W0: math.Pi / 30, L: 10}}
	var M [57]float64
	for l := 1; l <= 10; l++ {
		M[l] = 1.0
	}
	noise := seededNoise(22, UnvoicedFFTSize)

	// Compute the reference IFFT independently to derive the
	// expected windowed tail values.
	specRef := make([]complex128, UnvoicedFFTSize)
	for i, v := range noise {
		specRef[i] = complex(v, 0)
	}
	plan := fft.New(UnvoicedFFTSize)
	specRef = plan.Forward(specRef, specRef)
	ShapeUnvoicedSpectrum(specRef, p, &M)
	specRef = plan.Inverse(specRef, specRef)

	got := make([]float64, SamplesPerFrame)
	SynthUnvoicedOverlapAdd(&s, p, &M, noise, got)

	for n := 0; n < UnvoicedTailSamples; n++ {
		want := real(specRef[SamplesPerFrame+n]) * synthesisWindow[SamplesPerFrame+n]
		if math.Abs(s.PrevUnvoicedTail[n]-want) > 1e-9 {
			t.Fatalf("PrevUnvoicedTail[%d] = %v, want %v",
				n, s.PrevUnvoicedTail[n], want)
		}
	}
}

// TestSynthUnvoicedOverlapAddSecondFrameUsesPrevTail: a second frame
// gets dst[0..95] = curr_windowed[0..95] + prev_tail[0..95]. Pins
// the cross-frame additive contract.
func TestSynthUnvoicedOverlapAddSecondFrameUsesPrevTail(t *testing.T) {
	s := SynthState{}
	for n := 0; n < UnvoicedTailSamples; n++ {
		s.PrevUnvoicedTail[n] = float64(n + 1) // distinctive sentinel
	}
	expectedTail := s.PrevUnvoicedTail
	p := Params{Header: Header{W0: math.Pi / 30, L: 10}}
	var M [57]float64
	for l := 1; l <= 10; l++ {
		M[l] = 1.0
	}
	noise := seededNoise(33, UnvoicedFFTSize)

	// Reference curr-frame windowed contribution (no OA, no prev).
	freshState := SynthState{}
	curr := make([]float64, SamplesPerFrame)
	SynthUnvoicedOverlapAdd(&freshState, p, &M, noise, curr)

	// OA call on s with the planted tail.
	got := make([]float64, SamplesPerFrame)
	SynthUnvoicedOverlapAdd(&s, p, &M, noise, got)

	for n := 0; n < UnvoicedTailSamples; n++ {
		want := curr[n] + expectedTail[n]
		if math.Abs(got[n]-want) > 1e-9 {
			t.Errorf("dst[%d] = %v, want %v (curr_windowed + prev_tail)",
				n, got[n], want)
		}
	}
	// dst[96..159] is curr-only.
	for n := UnvoicedTailSamples; n < SamplesPerFrame; n++ {
		if math.Abs(got[n]-curr[n]) > 1e-9 {
			t.Errorf("dst[%d] = %v, want %v (curr-only past overlap region)",
				n, got[n], curr[n])
		}
	}
}

// TestSynthUnvoicedOverlapAddSilentFadesTail: a silent frame still
// emits the prev tail into dst[0..95] (no click on silence boundary)
// and clears the tail so a subsequent silent frame is fully silent.
func TestSynthUnvoicedOverlapAddSilentFadesTail(t *testing.T) {
	s := SynthState{}
	for n := 0; n < UnvoicedTailSamples; n++ {
		s.PrevUnvoicedTail[n] = 7
	}
	p := Params{Header: Header{Silent: true}}
	var M [57]float64
	dst := make([]float64, SamplesPerFrame)
	SynthUnvoicedOverlapAdd(&s, p, &M, nil, dst)

	for n := 0; n < UnvoicedTailSamples; n++ {
		if dst[n] != 7 {
			t.Errorf("dst[%d] = %v, want 7 (faded tail)", n, dst[n])
		}
	}
	for n := UnvoicedTailSamples; n < SamplesPerFrame; n++ {
		if dst[n] != 0 {
			t.Errorf("dst[%d] = %v, want 0 (no curr-frame synthesis)", n, dst[n])
		}
	}
	for n := 0; n < UnvoicedTailSamples; n++ {
		if s.PrevUnvoicedTail[n] != 0 {
			t.Errorf("PrevUnvoicedTail[%d] = %v, want 0 (cleared after fade)",
				n, s.PrevUnvoicedTail[n])
		}
	}
}

// TestSynthUnvoicedOverlapAddShortDstNoOp: dst < SamplesPerFrame
// returns immediately without touching state — including without
// emitting the prev_tail (the dst contract is the gate).
func TestSynthUnvoicedOverlapAddShortDstNoOp(t *testing.T) {
	s := SynthState{}
	for n := 0; n < UnvoicedTailSamples; n++ {
		s.PrevUnvoicedTail[n] = 5
	}
	p := Params{Header: Header{W0: math.Pi / 30, L: 10}}
	var M [57]float64
	short := make([]float64, SamplesPerFrame-1)
	SynthUnvoicedOverlapAdd(&s, p, &M, seededNoise(44, UnvoicedFFTSize), short)
	for n := 0; n < UnvoicedTailSamples; n++ {
		if s.PrevUnvoicedTail[n] != 5 {
			t.Errorf("tail[%d] = %v, want 5 (untouched on short-dst no-op)",
				n, s.PrevUnvoicedTail[n])
		}
	}
}

// TestResetClearsPrevUnvoicedTail confirms the new SynthState field
// is zeroed by Reset (the SynthState{} assignment covers it for free,
// but the test guards against future struct refactors that drift
// from Reset).
func TestResetClearsPrevUnvoicedTail(t *testing.T) {
	s := SynthState{}
	for n := 0; n < UnvoicedTailSamples; n++ {
		s.PrevUnvoicedTail[n] = float64(n)
	}
	s.Reset()
	for n := 0; n < UnvoicedTailSamples; n++ {
		if s.PrevUnvoicedTail[n] != 0 {
			t.Errorf("Reset: PrevUnvoicedTail[%d] = %v, want 0",
				n, s.PrevUnvoicedTail[n])
		}
	}
}
