package imbe

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/fft"
)

// IMBE 4400 unvoiced excitation — TIA-102.BABA §6.4.
//
// Voiced harmonics get the deterministic sinusoidal synthesis from
// step 4c (synth_voiced.go). Unvoiced harmonics get a noise-band
// excitation: white noise is FFT'd, bins under voiced harmonics
// (and bins outside the [1..L] model range) are zeroed, bins under
// unvoiced harmonics are multiplied by the per-harmonic amplitude
// Ml[l], and the result is IFFT'd back to the time domain.
//
// This file ships the spectrum-shaping kernel + the noise-driven
// pipeline. The caller passes pre-generated noise (rather than a
// noise source on SynthState) so unit tests stay deterministic;
// the high-level Decode() wiring that lands in step 4e + the
// post-merge Decode plumbing will pull noise from a seeded
// rand.Source attached to a per-call decoder.
//
// Algorithmic reference: TIA-102.BABA §6.4 + szechyjs/mbelib's
// unvoiced-synthesis loop (ISC-licensed; attribution preserved at
// the bottom of tables.go).

// UnvoicedFFTSize is the FFT length for the §6.4 noise spectrum.
// IMBE specifies 256 — long enough to give each harmonic band
// several bins for its noise excitation and short enough that the
// FFT is cheap. The 96-sample overlap (UnvoicedFFTSize − N) is
// the §6.4 overlap-add region — see SynthUnvoicedOverlapAdd.
const UnvoicedFFTSize = 256

// UnvoicedTailSamples is the number of windowed-IFFT samples
// carried over to the next frame for the §6.4 overlap-add. The
// 256-sample frame at 160-sample stride leaves 96 samples of
// overlap.
const UnvoicedTailSamples = UnvoicedFFTSize - SamplesPerFrame

// synthesisWindow is the §6.4 unvoiced overlap-add window — a
// 256-sample periodic Hann window. Multiplying the IFFT output by
// this window before overlap-add eliminates the click artifacts
// that would appear at frame boundaries if the IFFT were truncated
// to 160 samples without windowing.
//
// COLA caveat: the periodic Hann at 160-sample stride does not
// exactly satisfy the Constant-Overlap-Add condition. Sum of the
// two contributing windows (curr + prev) at output sample n in
// [0, 96) ranges roughly between 0.85 and 1.0, introducing a
// small (≲1.5 dB) amplitude modulation across the frame audible
// as a slight tremolo on broadband noise. The §6.2 spectral
// enhancement + spec-derived gain calibration polish PRs
// (roadmap step 5b/5c) revisit this — Hann is the simplest
// window that's strictly better than no window.
var synthesisWindow [UnvoicedFFTSize]float64

func init() {
	for n := 0; n < UnvoicedFFTSize; n++ {
		synthesisWindow[n] = 0.5 - 0.5*math.Cos(2*math.Pi*float64(n)/float64(UnvoicedFFTSize))
	}
}

// ShapeUnvoicedSpectrum modifies spec in place: each FFT bin gets
// classified by the harmonic it falls under, then either zeroed
// (voiced harmonic, harmonic out of [1..L]) or scaled by Ml[l]
// (unvoiced harmonic). The mapping uses the bin's effective
// frequency (mirroring k > N/2 back through the conjugate
// symmetry) so the shape preserves real-valued output: scaling
// each (k, N−k) pair by the same real Ml[l] keeps the spectrum
// Hermitian-symmetric.
//
// The bin → harmonic mapping is l = round(2π·k_eff / (N · ω₀)) —
// IMBE's "closest harmonic centre" rule. Bins between l and l+1
// land in whichever harmonic's band they're closer to; the model
// has no per-bin partition between adjacent harmonics, just the
// nearest-centre snap.
//
// Silent + zero-L frames leave spec untouched (caller handles
// silence at a higher level).
func ShapeUnvoicedSpectrum(spec []complex128, p Params, M *[57]float64) {
	if p.Silent || p.L == 0 {
		return
	}
	if len(spec) != UnvoicedFFTSize {
		return
	}
	const twoPi = 2 * math.Pi
	N := UnvoicedFFTSize
	for k := 0; k < N; k++ {
		kEff := k
		if k > N/2 {
			kEff = N - k
		}
		f := twoPi * float64(kEff) / float64(N)
		l := int(math.Round(f / p.W0))
		if l < 1 || l > p.L || p.Vl[l] == 1 {
			spec[k] = 0
			continue
		}
		spec[k] *= complex(M[l], 0)
	}
}

// SynthUnvoicedFromNoise runs the full §6.4 unvoiced-excitation
// pipeline on a caller-supplied length-UnvoicedFFTSize noise
// buffer, *without* the overlap-add synthesis window:
//
//  1. interpret noise[0..255] as a real time-domain signal,
//  2. forward-FFT to a 256-point complex spectrum,
//  3. ShapeUnvoicedSpectrum (zero voiced + out-of-range bins,
//     scale unvoiced bins by Ml[l]),
//  4. inverse-FFT back to a real time-domain signal,
//  5. accumulate the first SamplesPerFrame samples into dst.
//
// dst must be >= SamplesPerFrame; the function adds rather than
// overwrites so callers can sum it with the voiced-step output
// (step 4c, SynthVoiced) into the same buffer. Allocates one
// 256-complex spectrum + one fft.Plan per call.
//
// Noise input contract: the caller is responsible for noise
// statistics (Gaussian / unit-variance / seeded for tests). The
// IFFT result inherits whatever statistics the noise carried;
// callers wanting exact §6.4 RMS preservation should normalize
// upstream. This split keeps SynthUnvoicedFromNoise a pure
// function of its inputs (testable without a rand.Source).
//
// Silent + zero-L frames leave dst untouched. dst shorter than
// SamplesPerFrame, or noise of the wrong length, also leave
// dst untouched (caller short-circuits cleanly without a panic).
//
// Production callers should prefer SynthUnvoicedOverlapAdd, which
// applies the §6.4 synthesis window + threads the 96-sample tail
// through SynthState so frame boundaries are click-free.
// SynthUnvoicedFromNoise is retained as a stateless primitive for
// the spectrum-shaping unit tests.
func SynthUnvoicedFromNoise(p Params, M *[57]float64, noise []float64, dst []float64) {
	if p.Silent || p.L == 0 {
		return
	}
	if len(noise) != UnvoicedFFTSize || len(dst) < SamplesPerFrame {
		return
	}
	plan := fft.New(UnvoicedFFTSize)
	spec := make([]complex128, UnvoicedFFTSize)
	for i, v := range noise {
		spec[i] = complex(v, 0)
	}
	spec = plan.Forward(spec, spec)
	ShapeUnvoicedSpectrum(spec, p, M)
	spec = plan.Inverse(spec, spec)
	for n := 0; n < SamplesPerFrame; n++ {
		dst[n] += real(spec[n])
	}
}

// SynthUnvoicedOverlapAdd is the production §6.4 unvoiced-excitation
// path with the synthesis window + overlap-add threaded through
// SynthState.PrevUnvoicedTail. For each frame:
//
//  1. emit prev_tail[0..95] into dst[0..95] — the overlap region
//     where the previous frame's windowed IFFT extends into this
//     frame's output range;
//  2. forward-FFT(noise) → ShapeUnvoicedSpectrum → inverse-FFT;
//  3. multiply by the 256-sample synthesis window;
//  4. accumulate windowed[0..159] into dst[0..159] (this becomes
//     the curr-frame contribution that's audible immediately);
//  5. stash windowed[160..255] in s.PrevUnvoicedTail for the next
//     frame's overlap region.
//
// Silent + zero-L frames still emit the prev_tail into dst[0..95]
// (so a non-silent → silent transition fades the previous unvoiced
// content cleanly instead of truncating it), then clear the tail
// so the next non-silent frame starts from a clean baseline.
//
// dst must be >= SamplesPerFrame. dst shorter than SamplesPerFrame
// or noise of the wrong length leave dst + state untouched.
func SynthUnvoicedOverlapAdd(s *SynthState, p Params, M *[57]float64, noise []float64, dst []float64) {
	if len(dst) < SamplesPerFrame {
		return
	}
	// Always fade the prev tail into the overlap region so silence
	// transitions don't truncate audible content.
	for n := 0; n < UnvoicedTailSamples; n++ {
		dst[n] += s.PrevUnvoicedTail[n]
	}
	if p.Silent || p.L == 0 {
		s.PrevUnvoicedTail = [UnvoicedTailSamples]float64{}
		return
	}
	if len(noise) != UnvoicedFFTSize {
		return
	}

	plan := fft.New(UnvoicedFFTSize)
	spec := make([]complex128, UnvoicedFFTSize)
	for i, v := range noise {
		spec[i] = complex(v, 0)
	}
	spec = plan.Forward(spec, spec)
	ShapeUnvoicedSpectrum(spec, p, M)
	spec = plan.Inverse(spec, spec)

	for n := 0; n < SamplesPerFrame; n++ {
		dst[n] += real(spec[n]) * synthesisWindow[n]
	}
	for n := 0; n < UnvoicedTailSamples; n++ {
		s.PrevUnvoicedTail[n] = real(spec[SamplesPerFrame+n]) * synthesisWindow[SamplesPerFrame+n]
	}
}
