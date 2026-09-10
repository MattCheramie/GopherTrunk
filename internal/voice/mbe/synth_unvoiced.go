package mbe

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
// 256-sample power-complementary (tapered-cosine / Tukey) window.
// Multiplying the IFFT output by this window before overlap-add
// eliminates the click artifacts that would appear at frame
// boundaries if the IFFT were truncated to 160 samples without
// windowing.
//
// Why not a plain Hann: consecutive frames carry INDEPENDENT noise
// blocks, so in the UnvoicedTailSamples (= 96) overlap region their
// variances add. The audible noise-power envelope across a frame is
//
//	P[n] = w[n]² + w[n+160]²   for n < 96  (overlap region)
//	P[n] = w[n]²               for n ≥ 96  (no overlap)
//
// A periodic Hann at the 160-sample hop makes P[n] ripple ~7 dB,
// which leaks through as a 50 Hz frame-rate amplitude modulation — a
// buzzy tremolo on the unvoiced band (most audible on noisy /
// fricative-heavy speech), one of the synthesis artifacts examined
// while chasing the issue #644 "machine voice" report. The fix is a
// window that is power-complementary at this hop: a flat
// top over the non-overlapping centre [96, 160) and a quarter-wave
// sine taper of length 96 on each edge. With a symmetric window
// w[160+n] = w[95−n], and a sine taper gives w[n]² + w[95−n]² = 1,
// so P[n] ≡ 1 across the whole frame (verified flat to <1e-9 by
// TestSynthesisWindowPowerComplementary). The taper still decays to
// ≈0 at the edges, so it suppresses boundary clicks exactly as the
// Hann did, while passing more fricative/aspiration energy through
// the flat top (the Hann under-weighted the unvoiced band).
var synthesisWindow [UnvoicedFFTSize]float64

func init() {
	// taper = the overlap length; the centre [taper, N-taper) is unity.
	const taper = UnvoicedTailSamples // 96
	for n := 0; n < UnvoicedFFTSize; n++ {
		switch {
		case n < taper:
			synthesisWindow[n] = math.Sin(0.5 * math.Pi * (float64(n) + 0.5) / float64(taper))
		case n >= UnvoicedFFTSize-taper:
			synthesisWindow[n] = math.Sin(0.5 * math.Pi * (float64(UnvoicedFFTSize-n) - 0.5) / float64(taper))
		default:
			synthesisWindow[n] = 1.0
		}
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
	ShapeUnvoicedSpectrumGain(spec, p, M, LegacyUnvoicedGain)
}

// LegacyUnvoicedGain selects the pre-calibration unvoiced band level: each
// noise bin scaled by Ml alone. With unit-variance noise, an unnormalised
// forward FFT and a 1/N inverse, that puts (2·bins/N)·Ml² of power in the
// band — for a 125 Hz male voice (≈4 bins per harmonic side) ~12 dB BELOW
// the Ml²/2 a voiced harmonic of the same amplitude carries, and ~19 dB
// below mbelib. It is kept so the raw decoders' unit-test goldens hold.
const LegacyUnvoicedGain = -1

// DefaultUnvoicedGain is the unvoiced band power, relative to the Ml²/2 of
// a voiced harmonic of the same amplitude, that matches mbelib (the
// reference every DSD-family decoder plays through). mbelib synthesises an
// unvoiced harmonic as uvquality=3 random-phase cosines at (l−⅓, l, l+⅓)·ω₀,
// each of amplitude uvsine·qfactor·Ml = (1.3591409·e)·(ln 3/3)·Ml ≈ 1.353·Ml
// (mbe_synthesizeSpeechf), i.e. 3·(1.353·Ml)²/2 ≈ 2.745·Ml² per band —
// 5.49× (+7.4 dB) the equal-power level. Measured on the 10 Sep calibration
// pairs (same .imb/.amb frames decoded by both): GopherTrunk's UNVOICED
// harmonics sat 6 dB (IMBE) to 13–15 dB (AMBE+2 2450) below dsd-neo's while
// the VOICED ones matched above 1.5 kHz — the "no fricatives / crushed
// highs" half of the "sounds awful" report, distinct from the low-frequency
// tilt (see EnhancerConfig.TiltHz). 1 is the equal-power spec reading.
const DefaultUnvoicedGain = 5.49

// ShapeUnvoicedSpectrumGain is ShapeUnvoicedSpectrum with the unvoiced band
// level calibrated: gain ≥ 0 scales each unvoiced harmonic's noise band so
// its expected time-domain power is gain·Ml²/2 (the power of a voiced
// harmonic of amplitude Ml, times gain), independent of the pitch — the
// number of FFT bins a band spans (ω₀·N/2π per side, so a low-pitched voice
// spreads the same Ml over fewer bins) is folded into the per-bin scale so
// the band total is what the encoder measured. gain < 0 (LegacyUnvoicedGain)
// reproduces the uncalibrated per-bin Ml scaling byte-for-byte.
func ShapeUnvoicedSpectrumGain(spec []complex128, p Params, M *[57]float64, gain float64) {
	if p.Silent || p.L == 0 {
		return
	}
	if len(spec) != UnvoicedFFTSize {
		return
	}
	const twoPi = 2 * math.Pi
	N := UnvoicedFFTSize
	// Bins per harmonic (one side), from the same nearest-centre mapping the
	// shaping loop uses, so the per-bin scale exactly compensates the count.
	var bins [57]int
	if gain >= 0 {
		for k := 0; k <= N/2; k++ {
			f := twoPi * float64(k) / float64(N)
			l := int(math.Round(f / p.W0))
			if l >= 1 && l <= p.L {
				bins[l]++
			}
		}
	}
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
		scale := M[l]
		if gain >= 0 && bins[l] > 0 {
			// E[y²] = (2·bins/N)·(M·s)² for unit-variance noise, unnormalised
			// forward FFT and 1/N inverse; solve for s so E[y²] = gain·M²/2.
			scale *= math.Sqrt(gain * float64(N) / (4 * float64(bins[l])))
		}
		spec[k] *= complex(scale, 0)
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
	SynthUnvoicedOverlapAddGain(s, p, M, noise, dst, LegacyUnvoicedGain)
}

// SynthUnvoicedOverlapAddGain is SynthUnvoicedOverlapAdd with the unvoiced
// band level set by gain (see ShapeUnvoicedSpectrumGain / DefaultUnvoicedGain).
func SynthUnvoicedOverlapAddGain(s *SynthState, p Params, M *[57]float64, noise []float64, dst []float64, gain float64) {
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
	ShapeUnvoicedSpectrumGain(spec, p, M, gain)
	spec = plan.Inverse(spec, spec)

	for n := 0; n < SamplesPerFrame; n++ {
		dst[n] += real(spec[n]) * synthesisWindow[n]
	}
	for n := 0; n < UnvoicedTailSamples; n++ {
		s.PrevUnvoicedTail[n] = real(spec[SamplesPerFrame+n]) * synthesisWindow[SamplesPerFrame+n]
	}
}
