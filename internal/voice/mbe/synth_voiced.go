package mbe

import "math"

// IMBE 4400 voiced harmonic generator — TIA-102.BABA §6.3.
//
// Step 4a recovered log2(Ml) from cross-frame prediction; step 4b
// converted that to linear Ml. This step turns those amplitudes
// into the voiced portion of the 8 kHz output PCM by summing one
// sinusoid per voiced harmonic, with phase + amplitude continuity
// across frame boundaries.
//
// The §6.3 model for one frame of length N = SamplesPerFrame = 160:
//
//	a(n) = (1 − n/N) · M_prev[l] + (n/N) · M_curr[l]                   (eq. 88)
//	θ(n) = θ_prev[l] + n · l · ω₀_prev + l · (ω₀_curr − ω₀_prev) · n²/(2N)
//	s_v(n) = Σ_{l: voiced this or last frame} a(n) · cos(θ(n))         (eq. 89)
//
// M_prev[l] is the prev-frame linear amplitude (0 if the harmonic
// was unvoiced or the stream just started); M_curr[l] is the
// current-frame linear amplitude (0 if currently unvoiced). This
// dual-frame iteration gives a clean fade-in on unvoiced→voiced
// transitions and a fade-out on voiced→unvoiced transitions
// without click artifacts.
//
// On the first frame (s.PrevW0 == 0) the prev-ω₀ collapses to the
// curr-ω₀ so the quadratic phase term is zero — the harmonic
// starts at its target frequency, no synthetic frequency sweep.
//
// Algorithmic reference: TIA-102.BABA §6.3 + szechyjs/mbelib's
// voiced-synthesis loop (ISC-licensed; attribution preserved at
// the bottom of tables.go).

// SynthVoiced fills dst[0..N−1] (N = SamplesPerFrame = 160) with the
// voiced contribution for the current frame, summed across every
// harmonic that's voiced this frame OR was voiced last frame
// (the latter for fade-out continuity).
//
// dst must have length >= SamplesPerFrame; the function adds to
// existing values rather than overwriting so the caller can sum
// the unvoiced step's output (4d) into the same buffer. The
// function does not allocate.
//
// SynthVoiced does not mutate s — callers invoke
// (s).UpdateVoicedState afterward to roll phase + amplitude
// memory forward for the next frame.
//
// Silence + zero-L frames short-circuit (dst left as the caller
// initialized it). The caller is expected to have called
// (s).Reset() first on a Silent frame so the next non-silent
// frame starts from a clean state.
func SynthVoiced(s *SynthState, p Params, M *[57]float64, dst []float64) {
	if p.Silent || p.L == 0 {
		return
	}
	if len(dst) < SamplesPerFrame {
		return
	}

	// Effective prev ω₀: a fresh-stream first frame uses the curr
	// ω₀ so the quadratic phase term collapses to zero.
	prevW0 := s.PrevW0
	if prevW0 == 0 {
		prevW0 = p.W0
	}

	Lmax := p.L
	if s.PrevL > Lmax {
		Lmax = s.PrevL
	}
	if Lmax > 56 {
		Lmax = 56
	}

	const N = SamplesPerFrame
	const invN = 1.0 / float64(N)
	const invDoubleN = 1.0 / (2 * float64(N))

	for l := 1; l <= Lmax; l++ {
		prevAmp := s.PrevMl[l]
		var currAmp float64
		if l <= p.L && p.Vl[l] == 1 {
			currAmp = M[l]
		}
		if prevAmp == 0 && currAmp == 0 {
			continue
		}

		lf := float64(l)
		thetaBase := s.PrevPhase[l]
		// Linear phase coefficient (n^1) and quadratic (n^2):
		//   θ(n) = θ₀ + a·n + b·n²
		// a = l · ω_prev, b = l · (ω_curr − ω_prev) / (2N)
		a := lf * prevW0
		b := lf * (p.W0 - prevW0) * invDoubleN
		dAmp := currAmp - prevAmp
		for n := 0; n < N; n++ {
			nf := float64(n)
			amp := prevAmp + dAmp*(nf*invN)
			phase := thetaBase + a*nf + b*nf*nf
			dst[n] += amp * math.Cos(phase)
		}
	}
}

// UpdateVoicedState rolls the per-harmonic synthesis memory forward
// by the closed-form average-frequency phase increment
//
//	Δθ_l = N · l · (ω_prev + ω_curr) / 2
//
// (the integral of the linear-frequency tilt over n=[0..N], so the
// next frame's θ_prev[l] equals what θ(n=N) would have been on this
// frame), wraps it into [0, 2π), and stores the current-frame
// linear amplitudes in PrevMl (zeroed for harmonics that are
// unvoiced this frame, so the next frame's fade-in math sees a
// clean zero baseline).
//
// Silence + zero-L frames are a no-op so callers can invoke this
// unconditionally on the synthesis path. This is the coherent-phase
// variant (every harmonic stays phase-locked frame to frame) — see
// UpdateVoicedStateDispersed for the §6.3 phase-regeneration path that
// the IMBE decoder uses to avoid the robotic "buzz" of fully coherent
// synthesis.
func (s *SynthState) UpdateVoicedState(p Params, M *[57]float64) {
	s.updateVoicedState(p, M, nil)
}

// UpdateVoicedStateDispersed is UpdateVoicedState plus TIA-102.BABA
// §6.3 voiced-phase regeneration. Fully phase-coherent voiced synthesis
// re-aligns every harmonic once per pitch period, which radiates a
// buzzy impulse train — the "robotic" artifact that makes a from-scratch
// MBE decoder sound markedly worse than the reference imbe_vocoder. The
// reference disperses the upper harmonics' phase
// (`if (i > num_harms_max/4) ph_mem[i] += rand()`); this does the same:
// each upper harmonic (l > L/4) accumulates a per-frame random phase
// step phaseDisp[l] on top of the coherent average-frequency increment,
// so the upper harmonics drift out of strict harmonic lock and the
// excitation regains the natural phase spread of voiced speech. Low
// harmonics (l ≤ L/4) stay coherent so pitch + formant structure are
// preserved.
//
// phaseDisp[l] is the random phase step (radians) for harmonic l. The
// caller supplies it (from a seeded source) so output stays
// deterministic and unit tests can pin it — mirroring how the unvoiced
// path takes caller-supplied noise. Indices l ≤ L/4 are ignored; a nil
// phaseDisp is identical to UpdateVoicedState.
func (s *SynthState) UpdateVoicedStateDispersed(p Params, M *[57]float64, phaseDisp *[57]float64) {
	s.updateVoicedState(p, M, phaseDisp)
}

func (s *SynthState) updateVoicedState(p Params, M *[57]float64, phaseDisp *[57]float64) {
	if p.Silent || p.L == 0 {
		return
	}
	prevW0 := s.PrevW0
	if prevW0 == 0 {
		prevW0 = p.W0
	}

	Lmax := p.L
	if s.PrevL > Lmax {
		Lmax = s.PrevL
	}
	if Lmax > 56 {
		Lmax = 56
	}

	const halfN = float64(SamplesPerFrame) * 0.5
	const twoPi = 2 * math.Pi

	var newPrevPhase, newPrevMl [57]float64
	for l := 1; l <= Lmax; l++ {
		lf := float64(l)
		delta := lf * (prevW0 + p.W0) * halfN
		ph := s.PrevPhase[l] + delta
		// §6.3 upper-harmonic phase regeneration: l > L/4 accumulates a
		// random phase step so the upper harmonics drift out of strict
		// harmonic lock (de-buzz). Low harmonics stay coherent.
		if phaseDisp != nil && 4*l > p.L {
			ph += phaseDisp[l]
		}
		ph = math.Mod(ph, twoPi)
		if ph < 0 {
			ph += twoPi
		}
		newPrevPhase[l] = ph
		if l <= p.L && p.Vl[l] == 1 {
			newPrevMl[l] = M[l]
		}
	}
	s.PrevPhase = newPrevPhase
	s.PrevMl = newPrevMl
}
