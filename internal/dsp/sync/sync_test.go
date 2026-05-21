package sync

import (
	"math"
	"testing"
)

func TestMuellerMullerRecoversSymbolCount(t *testing.T) {
	// Synthesize a 4-PAM signal at 8 sps with raised-cosine pulses peaking
	// at the *start* of each symbol period so that boundary sampling lands
	// on a non-zero gradient.
	const sps = 8
	const nSym = 300
	src := make([]float32, nSym*sps)
	want := make([]float32, nSym)
	for i := 0; i < nSym; i++ {
		v := float32([]int{-3, -1, 1, 3}[(i*5+3)%4])
		want[i] = v
		for k := 0; k < sps; k++ {
			alpha := 0.5 + 0.5*math.Cos(math.Pi*float64(k)/float64(sps-1))
			src[i*sps+k] = v * float32(alpha)
		}
	}
	m := NewMuellerMuller(float64(sps), 0.02)
	got := m.Process(nil, src)
	if len(got) < nSym-3 || len(got) > nSym+3 {
		t.Fatalf("symbol count = %d, want %d ± 3", len(got), nSym)
	}

	// After warmup, sliced polarity should match expected better than chance.
	// Find the best alignment offset; the loop's first output corresponds
	// to whatever symbol falls at its sub-sample lock point, which depends
	// on the init phase.
	const warm = 20
	bestRate := 0.0
	for shift := -2; shift <= 2; shift++ {
		matches, total := 0, 0
		for i := warm; i < len(got); i++ {
			j := i + shift
			if j < 0 || j >= len(want) {
				continue
			}
			total++
			if (got[i] > 0) == (want[j] > 0) {
				matches++
			}
		}
		if total > 0 {
			rate := float64(matches) / float64(total)
			if rate > bestRate {
				bestRate = rate
			}
		}
	}
	if bestRate < 0.95 {
		t.Errorf("best polarity match rate = %.2f, want > 0.95", bestRate)
	}
}

func TestMuellerMullerChunkInvariance(t *testing.T) {
	// The recovered symbol stream must not depend on how the input is
	// chunked. Issue #275: the loop skipped src[0] on every Process
	// call, losing one sample of clock phase per chunk, so a stream fed
	// in RTL-realistic small chunks drifted the C4FM symbol timing.
	const sps = 8
	src := make([]float32, 4000)
	for i := range src {
		sym := float32([]int{-3, -1, 1, 3}[(i/sps*7+1)%4])
		theta := math.Pi * float64(i%sps) / float64(sps)
		src[i] = sym * float32(0.5+0.5*math.Cos(theta))
	}
	oneShot := NewMuellerMuller(sps, 0.05).Process(nil, src)

	m := NewMuellerMuller(sps, 0.05)
	var chunked []float32
	for i := 0; i < len(src); i += 37 { // an awkward, non-sps chunk size
		end := i + 37
		if end > len(src) {
			end = len(src)
		}
		chunked = append(chunked, m.Process(nil, src[i:end])...)
	}
	if len(chunked) != len(oneShot) {
		t.Fatalf("chunked symbol count = %d, one-shot = %d", len(chunked), len(oneShot))
	}
	for i := range oneShot {
		if chunked[i] != oneShot[i] {
			t.Fatalf("symbol %d: chunked = %v, one-shot = %v — Mueller-Müller "+
				"output depends on chunk size (#275)", i, chunked[i], oneShot[i])
		}
	}
}

func TestCorrelatorFindsPattern(t *testing.T) {
	pattern := []float32{1, -1, 1, 1, -1}
	noise := []float32{0, 0, 0.1, -0.1, 0}
	stream := append([]float32{}, noise...)
	stream = append(stream, pattern...)
	stream = append(stream, noise...)
	stream = append(stream, pattern...)

	threshold := float32(0)
	for _, p := range pattern {
		threshold += p * p
	}
	threshold *= 0.9 // 90% of perfect-match correlation

	c := NewCorrelator(pattern, threshold)
	hits, _ := c.Process(nil, stream, 0)
	if len(hits) < 2 {
		t.Fatalf("hits = %v, want >= 2 matches", hits)
	}
	// The correlator emits a hit at the index of the *last* pattern sample.
	expected := []int{len(noise) + len(pattern) - 1, len(noise) + len(pattern) + len(noise) + len(pattern) - 1}
	for _, want := range expected {
		found := false
		for _, h := range hits {
			if h == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing match at index %d (got %v)", want, hits)
		}
	}
}
