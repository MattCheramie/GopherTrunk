package mbe

import (
	"math"
	"math/rand"
	"testing"
)

// TestVoicedPhaseReferenceMorePeriodicThanRandomWalk is a confound-free
// A/B of the pre-fix random-walk §6.3 dispersion (full [−π,π) accumulated
// into the phase memory every frame) vs the shipped reference model
// (coherent phase memory + numUv/L-scaled per-frame offset) on a sustained
// female-pitch vowel. No unvoiced noise is involved, so any difference is
// purely the voiced-phase model. Measures waveform periodicity (normalized
// autocorrelation at the pitch lag); higher = more harmonic/intelligible.
//
// Regression guard: the random walk decorrelated the upper harmonics into
// noise (the "female voices barely intelligible" report); the reference
// model keeps mostly-voiced frames coherent. If a future change reverts to
// unconditional/accumulated dispersion this test fails.
func TestVoicedPhaseReferenceMorePeriodicThanRandomWalk(t *testing.T) {
	const sr = 8000.0
	const f0 = 220.0 // female fundamental
	w0 := 2 * math.Pi * f0 / sr
	L := int(math.Pi / w0) // ~18 harmonics
	if L > 56 {
		L = 56
	}
	// Formant-shaped amplitudes, all voiced (numUv small for real female
	// speech; here fully voiced is the clearest case — NEW => ~0 dispersion).
	var M [57]float64
	p := Params{Header: Header{W0: w0, L: L}}
	for l := 1; l <= L; l++ {
		p.Vl[l] = 1
		f := float64(l) * f0
		M[l] = math.Exp(-math.Pow((f-700)/700, 2)) + 0.6*math.Exp(-math.Pow((f-1800)/900, 2)) + 0.05
	}

	const frames = 60
	old := synthSeries(p, &M, frames, true)  // old: full random-walk
	neu := synthSeries(p, &M, frames, false) // new: reference model

	lag := int(math.Round(float64(sr) / float64(f0)))
	po := periodicity(old, lag)
	pn := periodicity(neu, lag)
	t.Logf("sustained female vowel F0=%.0f L=%d (fully voiced):", f0, L)
	t.Logf("  OLD random-walk dispersion: pitch-lag periodicity = %.3f", po)
	t.Logf("  NEW reference model:        pitch-lag periodicity = %.3f", pn)
	t.Logf("  -> NEW is %.1f%% more periodic/harmonic (less robotic-noise on upper harmonics)", (pn/po-1)*100)
	if pn <= po {
		t.Errorf("expected NEW (reference) more periodic than OLD random-walk: new=%.3f old=%.3f", pn, po)
	}
}

// synthSeries renders `frames` steady frames and returns the concatenated
// voiced waveform. oldModel=true reproduces the pre-fix behavior (full
// [-π,π) random offset accumulated into the phase memory each frame);
// oldModel=false uses the shipped SynthVoicedDispersed + coherent
// UpdateVoicedState with the numUv/L-scaled offset.
func synthSeries(p Params, M *[57]float64, frames int, oldModel bool) []float64 {
	var s SynthState
	rng := rand.New(rand.NewSource(42))
	numUv := p.L
	for l := 1; l <= p.L; l++ {
		if p.Vl[l] == 1 {
			numUv--
		}
	}
	scale := float64(numUv) / float64(p.L)
	var out []float64
	for f := 0; f < frames; f++ {
		dst := make([]float64, SamplesPerFrame)
		var disp [57]float64
		for l := 1; l <= 56; l++ {
			if oldModel {
				disp[l] = (rng.Float64()*2 - 1) * math.Pi // full scale
			} else {
				disp[l] = (rng.Float64()*2 - 1) * math.Pi * scale // numUv/L scaled
			}
		}
		if oldModel {
			// Pre-fix: coherent synthesis start phase, dispersion accumulated
			// into the memory (random walk).
			SynthVoiced(&s, p, M, dst)
			oldAccumulateDispersed(&s, p, M, &disp)
		} else {
			SynthVoicedDispersed(&s, p, M, dst, &disp)
			s.UpdateVoicedState(p, M)
		}
		s.PrevW0 = p.W0
		s.PrevL = p.L
		out = append(out, dst...)
	}
	return out
}

// oldAccumulateDispersed replicates the removed UpdateVoicedStateDispersed:
// it folds a full-scale random offset into PrevPhase for voiced upper
// harmonics, so the phase random-walks frame to frame.
func oldAccumulateDispersed(s *SynthState, p Params, M *[57]float64, disp *[57]float64) {
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
	var np, nm [57]float64
	for l := 1; l <= Lmax; l++ {
		lf := float64(l)
		ph := s.PrevPhase[l] + lf*(prevW0+p.W0)*halfN
		voiced := l <= p.L && p.Vl[l] == 1
		if voiced && 4*l > p.L {
			ph += disp[l]
		}
		ph = math.Mod(ph, twoPi)
		if ph < 0 {
			ph += twoPi
		}
		np[l] = ph
		if voiced {
			nm[l] = M[l]
		}
	}
	s.PrevPhase = np
	s.PrevMl = nm
}

func periodicity(d []float64, lag int) float64 {
	// skip the first few frames (transient), use steady tail
	if len(d) < lag*4 {
		return 0
	}
	d = d[len(d)/3:]
	var num, e0 float64
	for i := 0; i+lag < len(d); i++ {
		num += d[i] * d[i+lag]
	}
	for i := 0; i < len(d); i++ {
		e0 += d[i] * d[i]
	}
	if e0 == 0 {
		return 0
	}
	return num / e0
}
