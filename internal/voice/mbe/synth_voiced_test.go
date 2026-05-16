package mbe

import (
	"math"
	"testing"
)

const synthEpsilon = 1e-9

// TestSynthVoicedFreshStreamUnvoiced: a fresh-state frame with no
// voiced harmonics produces a fully-silent voiced contribution.
// Pins the "no spurious carrier on the first frame" invariant.
func TestSynthVoicedFreshStreamUnvoiced(t *testing.T) {
	var s SynthState
	p := Params{Header: Header{W0: math.Pi / 30, L: 10}}
	// All Vl[l] = 0 (default). M is irrelevant when unvoiced.
	var M [57]float64
	dst := make([]float64, SamplesPerFrame)
	SynthVoiced(&s, p, &M, dst)
	for n, v := range dst {
		if v != 0 {
			t.Errorf("dst[%d] = %v, want 0 (no voiced harmonics)", n, v)
		}
	}
}

// TestSynthVoicedFreshStreamSingleHarmonic: with prev_amp = 0,
// curr_amp = 1, a single voiced harmonic l = 1 at constant ω₀
// produces a ramping cos(l·ω₀·n) with linear amp tilt 0 → 1 across
// the frame. Pins the fade-in path on a fresh stream.
func TestSynthVoicedFreshStreamSingleHarmonic(t *testing.T) {
	var s SynthState
	w0 := math.Pi / 30
	p := Params{Header: Header{W0: w0, L: 1}}
	p.Vl[1] = 1
	var M [57]float64
	M[1] = 1.0
	dst := make([]float64, SamplesPerFrame)
	SynthVoiced(&s, p, &M, dst)

	N := float64(SamplesPerFrame)
	for n := 0; n < SamplesPerFrame; n++ {
		nf := float64(n)
		amp := nf / N
		phase := w0 * nf // prevW0 == w0 (fresh-stream collapse), quadratic term zero
		want := amp * math.Cos(phase)
		if math.Abs(dst[n]-want) > synthEpsilon {
			t.Fatalf("dst[%d] = %v, want %v (fade-in cos at l=1)", n, dst[n], want)
		}
	}
}

// TestSynthVoicedSteadyState: when prev_amp = curr_amp = 1 and ω₀
// is steady, the output is a pure cos(l·ω₀·n) sinusoid (the linear
// amp tilt collapses to a constant 1). Pins the steady-state path.
func TestSynthVoicedSteadyState(t *testing.T) {
	w0 := math.Pi / 30
	s := SynthState{
		PrevW0:    w0,
		PrevL:     1,
		PrevPhase: [57]float64{},
		PrevMl:    [57]float64{},
	}
	s.PrevMl[1] = 1.0
	p := Params{Header: Header{W0: w0, L: 1}}
	p.Vl[1] = 1
	var M [57]float64
	M[1] = 1.0
	dst := make([]float64, SamplesPerFrame)
	SynthVoiced(&s, p, &M, dst)

	for n := 0; n < SamplesPerFrame; n++ {
		want := math.Cos(w0 * float64(n))
		if math.Abs(dst[n]-want) > synthEpsilon {
			t.Fatalf("dst[%d] = %v, want %v (steady cos)", n, dst[n], want)
		}
	}
}

// TestSynthVoicedFadeOut: a harmonic that was voiced last frame but
// is unvoiced now fades from prev_amp → 0 across the frame. The
// curr_amp = 0 case must still synthesize so the fade-out is
// audible (no click on voiced→unvoiced transitions).
func TestSynthVoicedFadeOut(t *testing.T) {
	w0 := math.Pi / 30
	s := SynthState{PrevW0: w0, PrevL: 1}
	s.PrevMl[1] = 1.0
	p := Params{Header: Header{W0: w0, L: 1}}
	// Vl[1] = 0 — curr unvoiced.
	var M [57]float64
	dst := make([]float64, SamplesPerFrame)
	SynthVoiced(&s, p, &M, dst)

	N := float64(SamplesPerFrame)
	for n := 0; n < SamplesPerFrame; n++ {
		nf := float64(n)
		amp := 1.0 - nf/N // ramp 1 → ~0
		want := amp * math.Cos(w0*nf)
		if math.Abs(dst[n]-want) > synthEpsilon {
			t.Fatalf("dst[%d] = %v, want %v (fade-out)", n, dst[n], want)
		}
	}
}

// TestSynthVoicedPhaseContinuity: synth two frames back-to-back
// with the same ω₀ + voicing + amplitude, confirm the phase
// at the start of frame 2 equals what θ(N) would have been on
// frame 1 (no phase discontinuity at the boundary).
func TestSynthVoicedPhaseContinuity(t *testing.T) {
	var s SynthState
	w0 := math.Pi / 17 // not a clean divisor of 2π so phase wrap is non-trivial
	p := Params{Header: Header{W0: w0, L: 1}}
	p.Vl[1] = 1
	var M [57]float64
	M[1] = 1.0

	dst1 := make([]float64, SamplesPerFrame)
	SynthVoiced(&s, p, &M, dst1)
	s.PrevW0 = p.W0
	s.PrevL = p.L
	s.UpdateVoicedState(p, &M)

	dst2 := make([]float64, SamplesPerFrame)
	SynthVoiced(&s, p, &M, dst2)

	// Frame 1's last sample at n=N-1 has amp=(N-1)/N and phase=w0·(N-1).
	// Frame 2's first sample at n=0 has amp=prevAmp=1 and phase=θ_prev.
	// θ_prev = N · w0 (closed-form integral). So expected dst2[0]:
	wantStart := math.Cos(w0 * float64(SamplesPerFrame))
	if math.Abs(dst2[0]-wantStart) > synthEpsilon {
		t.Errorf("frame 2 dst[0] = %v, want %v (phase continuity)",
			dst2[0], wantStart)
	}

	// More generally, frame 2 sample n is cos(w0 · (N + n)) at amp 1.
	for n := 0; n < SamplesPerFrame; n++ {
		want := math.Cos(w0 * float64(SamplesPerFrame+n))
		if math.Abs(dst2[n]-want) > 1e-7 {
			t.Fatalf("frame 2 dst[%d] = %v, want %v", n, dst2[n], want)
		}
	}
}

// TestSynthVoicedMultiHarmonicSum: two voiced harmonics produce the
// linear sum of their individual sinusoids. Pins the additive
// contribution + the harmonic frequency relationship (l · ω₀).
func TestSynthVoicedMultiHarmonicSum(t *testing.T) {
	w0 := math.Pi / 30
	s := SynthState{PrevW0: w0, PrevL: 2}
	s.PrevMl[1] = 1.0
	s.PrevMl[2] = 0.5
	p := Params{Header: Header{W0: w0, L: 2}}
	p.Vl[1] = 1
	p.Vl[2] = 1
	var M [57]float64
	M[1] = 1.0
	M[2] = 0.5
	dst := make([]float64, SamplesPerFrame)
	SynthVoiced(&s, p, &M, dst)

	for n := 0; n < SamplesPerFrame; n++ {
		nf := float64(n)
		want := 1.0*math.Cos(w0*nf) + 0.5*math.Cos(2*w0*nf)
		if math.Abs(dst[n]-want) > synthEpsilon {
			t.Fatalf("dst[%d] = %v, want %v (sum of two cosines)", n, dst[n], want)
		}
	}
}

// TestSynthVoicedQuadraticPhase: when ω₀ changes between frames,
// the phase tilts quadratically across the frame. Verify at
// n = N/2 the phase equals θ_prev + a·N/2 + b·(N/2)².
func TestSynthVoicedQuadraticPhase(t *testing.T) {
	wPrev := math.Pi / 30
	wCurr := math.Pi / 15 // doubled
	s := SynthState{PrevW0: wPrev, PrevL: 1}
	s.PrevMl[1] = 1.0
	p := Params{Header: Header{W0: wCurr, L: 1}}
	p.Vl[1] = 1
	var M [57]float64
	M[1] = 1.0
	dst := make([]float64, SamplesPerFrame)
	SynthVoiced(&s, p, &M, dst)

	N := float64(SamplesPerFrame)
	a := wPrev
	b := (wCurr - wPrev) / (2 * N)
	for _, n := range []int{0, 40, 80, 120, 159} {
		nf := float64(n)
		phase := a*nf + b*nf*nf
		want := math.Cos(phase) // amp = 1 (steady prev=curr=1)
		if math.Abs(dst[n]-want) > synthEpsilon {
			t.Errorf("dst[%d] = %v, want %v (quadratic phase)", n, dst[n], want)
		}
	}
}

// TestSynthVoicedAddsToExistingDst: the function adds rather than
// overwrites so the unvoiced step (4d) can sum into the same buffer.
func TestSynthVoicedAddsToExistingDst(t *testing.T) {
	var s SynthState
	w0 := math.Pi / 30
	p := Params{Header: Header{W0: w0, L: 1}}
	p.Vl[1] = 1
	var M [57]float64
	M[1] = 1.0
	dst := make([]float64, SamplesPerFrame)
	for n := range dst {
		dst[n] = 7 // sentinel
	}
	SynthVoiced(&s, p, &M, dst)
	// dst[0] = 7 + (0/N) · cos(0) = 7 + 0 = 7 (fade-in starts at zero amp).
	if math.Abs(dst[0]-7) > synthEpsilon {
		t.Errorf("dst[0] = %v, want 7 (sum into sentinel, amp=0 at n=0)", dst[0])
	}
	// dst[N-1] = 7 + (159/N) · cos(w0·159).
	N := float64(SamplesPerFrame)
	wantTail := 7.0 + (float64(SamplesPerFrame-1)/N)*math.Cos(w0*float64(SamplesPerFrame-1))
	if math.Abs(dst[SamplesPerFrame-1]-wantTail) > synthEpsilon {
		t.Errorf("dst[N-1] = %v, want %v (sum into sentinel)",
			dst[SamplesPerFrame-1], wantTail)
	}
}

// TestSynthVoicedSilentFrameLeavesDstUntouched: silent frames
// short-circuit; dst keeps its prior contents. Caller is expected
// to Reset() s on the silence indicator.
func TestSynthVoicedSilentFrameLeavesDstUntouched(t *testing.T) {
	var s SynthState
	p := Params{Header: Header{Silent: true}}
	var M [57]float64
	dst := make([]float64, SamplesPerFrame)
	for n := range dst {
		dst[n] = 42
	}
	SynthVoiced(&s, p, &M, dst)
	for n, v := range dst {
		if v != 42 {
			t.Errorf("dst[%d] = %v, want 42 (silent short-circuit)", n, v)
		}
	}
}

// TestSynthVoicedShortDstNoPanic: passing dst shorter than
// SamplesPerFrame returns early without writing or panicking. Lets
// callers fail-fast in unit tests without crashing the daemon.
func TestSynthVoicedShortDstNoPanic(t *testing.T) {
	var s SynthState
	p := Params{Header: Header{W0: math.Pi / 30, L: 1}}
	p.Vl[1] = 1
	var M [57]float64
	M[1] = 1.0
	dst := make([]float64, 100)
	for n := range dst {
		dst[n] = -1
	}
	SynthVoiced(&s, p, &M, dst)
	for n, v := range dst {
		if v != -1 {
			t.Errorf("dst[%d] = %v, want -1 (short-buffer no-op)", n, v)
		}
	}
}

// TestUpdateVoicedStateAdvancesPhase: the phase increment is the
// closed-form integral N · l · (ω_prev + ω_curr) / 2. Pins the
// algebra without depending on SynthVoiced.
func TestUpdateVoicedStateAdvancesPhase(t *testing.T) {
	wPrev := 0.1
	wCurr := 0.2
	s := SynthState{PrevW0: wPrev, PrevL: 2, PrevPhase: [57]float64{0, 1.0, 0.5}}
	p := Params{Header: Header{W0: wCurr, L: 2}}
	p.Vl[1] = 1
	p.Vl[2] = 1
	var M [57]float64
	M[1] = 0.7
	M[2] = 0.3
	s.UpdateVoicedState(p, &M)

	N := float64(SamplesPerFrame)
	twoPi := 2 * math.Pi
	for l := 1; l <= 2; l++ {
		oldPhase := []float64{0, 1.0, 0.5}[l]
		delta := float64(l) * (wPrev + wCurr) * N / 2
		want := math.Mod(oldPhase+delta, twoPi)
		if want < 0 {
			want += twoPi
		}
		if math.Abs(s.PrevPhase[l]-want) > synthEpsilon {
			t.Errorf("PrevPhase[%d] = %v, want %v", l, s.PrevPhase[l], want)
		}
	}
}

// TestUpdateVoicedStateZeroesUnvoiced: harmonics that are unvoiced
// this frame get PrevMl = 0 so the next frame's fade-in has a clean
// zero baseline.
func TestUpdateVoicedStateZeroesUnvoiced(t *testing.T) {
	s := SynthState{PrevW0: 0.1, PrevL: 3}
	s.PrevMl[1] = 9
	s.PrevMl[2] = 9
	s.PrevMl[3] = 9
	p := Params{Header: Header{W0: 0.1, L: 3}}
	p.Vl[1] = 1 // voiced
	p.Vl[2] = 0 // unvoiced
	p.Vl[3] = 1 // voiced
	var M [57]float64
	M[1] = 0.5
	M[2] = 0.7 // ignored because unvoiced
	M[3] = 0.9
	s.UpdateVoicedState(p, &M)
	if !almostEqual(s.PrevMl[1], 0.5) {
		t.Errorf("PrevMl[1] = %v, want 0.5", s.PrevMl[1])
	}
	if s.PrevMl[2] != 0 {
		t.Errorf("PrevMl[2] = %v, want 0 (unvoiced zeroed)", s.PrevMl[2])
	}
	if !almostEqual(s.PrevMl[3], 0.9) {
		t.Errorf("PrevMl[3] = %v, want 0.9", s.PrevMl[3])
	}
}

// TestUpdateVoicedStateSilentNoOp: silent frames don't roll state.
func TestUpdateVoicedStateSilentNoOp(t *testing.T) {
	s := SynthState{PrevPhase: [57]float64{0, 1.5}, PrevMl: [57]float64{0, 0.7}}
	p := Params{Header: Header{Silent: true}}
	var M [57]float64
	s.UpdateVoicedState(p, &M)
	if s.PrevPhase[1] != 1.5 || s.PrevMl[1] != 0.7 {
		t.Errorf("Silent frame mutated state: phase=%v ml=%v",
			s.PrevPhase[1], s.PrevMl[1])
	}
}

// TestResetClearsVoicedFields confirms Reset() also clears the new
// PrevMl + PrevPhase fields added in this PR (regression guard
// against the SynthState extension drifting from Reset).
func TestResetClearsVoicedFields(t *testing.T) {
	s := SynthState{PrevW0: 0.1, PrevL: 5}
	for l := 1; l <= 5; l++ {
		s.PrevLog2Ml[l] = float64(l)
		s.PrevMl[l] = float64(l)
		s.PrevPhase[l] = float64(l) * 0.1
	}
	s.Reset()
	for l := 0; l <= 56; l++ {
		if s.PrevMl[l] != 0 {
			t.Errorf("Reset: PrevMl[%d] = %v, want 0", l, s.PrevMl[l])
		}
		if s.PrevPhase[l] != 0 {
			t.Errorf("Reset: PrevPhase[%d] = %v, want 0", l, s.PrevPhase[l])
		}
	}
}

// TestSynthVoicedZeroAmpSkipsHarmonic: a voiced harmonic with M[l]
// = 0 and prev_amp = 0 contributes nothing to dst. Pins the
// short-circuit optimization (skips the inner loop entirely).
func TestSynthVoicedZeroAmpSkipsHarmonic(t *testing.T) {
	var s SynthState
	p := Params{Header: Header{W0: math.Pi / 30, L: 3}}
	p.Vl[1] = 1
	p.Vl[2] = 0
	p.Vl[3] = 1
	var M [57]float64
	M[1] = 0 // zero amp on a voiced harmonic — degenerate but possible
	M[3] = 0 // ditto
	dst := make([]float64, SamplesPerFrame)
	SynthVoiced(&s, p, &M, dst)
	for n, v := range dst {
		if v != 0 {
			t.Fatalf("dst[%d] = %v, want 0 (all amps zero)", n, v)
		}
	}
}
