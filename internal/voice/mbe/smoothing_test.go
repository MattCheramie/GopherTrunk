package mbe

import (
	"math"
	"testing"
)

// TestSmootherCleanIsInert: at zero corrected bits the error rate stays
// clean and Smooth leaves amplitudes + voicing untouched.
func TestSmootherCleanIsInert(t *testing.T) {
	var s Smoother
	p := Params{Header: Header{W0: math.Pi / 30, L: 16}}
	var M [MaxL + 1]float64
	for l := 1; l <= p.L; l++ {
		M[l] = float64(l) // arbitrary
		p.Vl[l] = 0       // all unvoiced
	}
	want := M
	for i := 0; i < 50; i++ {
		if mute := s.UpdateErrorRate(0); mute {
			t.Fatalf("frame %d: muted on clean channel", i)
		}
		s.Smooth(&p, &M, 0)
	}
	if M != want {
		t.Error("Smooth modified amplitudes on a clean channel")
	}
	for l := 1; l <= p.L; l++ {
		if p.Vl[l] != 0 {
			t.Errorf("Smooth forced voicing on clean channel at l=%d", l)
		}
	}
}

// TestSmootherMutesOnHighErrorRate: a sustained high corrected-bit count
// drives the error rate past the mute threshold.
func TestSmootherMutesOnHighErrorRate(t *testing.T) {
	var s Smoother
	muted := false
	for i := 0; i < 200; i++ {
		if s.UpdateErrorRate(15) { // max FEC corrections, sustained
			muted = true
			break
		}
	}
	if !muted {
		t.Errorf("error rate never crossed mute threshold %.4f (got %.4f)",
			SmoothMuteErrorRate, s.ErrorRate())
	}
	// And a clean stream never mutes.
	var s2 Smoother
	for i := 0; i < 1000; i++ {
		if s2.UpdateErrorRate(0) {
			t.Fatal("clean stream muted")
		}
	}
}

// TestSmootherCapsAmplitudeSpike: once the channel is noisy, a frame whose
// energy spikes far above the running local energy is pulled down.
func TestSmootherCapsAmplitudeSpike(t *testing.T) {
	var s Smoother
	p := Params{Header: Header{W0: math.Pi / 30, L: 16}}
	// Establish a local-energy baseline over several modest noisy frames.
	for i := 0; i < 20; i++ {
		var M [MaxL + 1]float64
		for l := 1; l <= p.L; l++ {
			M[l] = 100
		}
		s.UpdateErrorRate(8) // noisy enough to leave the clean regime
		s.Smooth(&p, &M, 8)
	}
	// Now a huge spike frame.
	var spike [MaxL + 1]float64
	for l := 1; l <= p.L; l++ {
		spike[l] = 100000
	}
	before := FrameEnergy(&spike, p.L)
	s.UpdateErrorRate(8)
	s.Smooth(&p, &spike, 8)
	after := FrameEnergy(&spike, p.L)
	if after >= before {
		t.Errorf("spike not capped: energy before=%.0f after=%.0f", before, after)
	}
}

func TestSmootherReset(t *testing.T) {
	var s Smoother
	s.UpdateErrorRate(10)
	s.Reset()
	if s.ErrorRate() != 0 || s.localEnergy != 0 {
		t.Errorf("Reset left state er=%v le=%v", s.ErrorRate(), s.localEnergy)
	}
}
