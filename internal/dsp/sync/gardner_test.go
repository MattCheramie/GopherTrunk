package sync

import (
	"math"
	"math/cmplx"
	"testing"
)

// TestGardnerRecoversAlignedQPSK: synthesize a clean QPSK signal at
// 8 sps with sample boundaries aligned to symbol time. Gardner
// should recover the original symbol count and lock the polarity.
func TestGardnerRecoversAlignedQPSK(t *testing.T) {
	const sps = 8
	const nSym = 400
	// QPSK constellation: ±1 ± 1j scaled by 1/sqrt(2).
	const scale = 0.7071067811865475
	syms := make([]complex64, nSym)
	for i := 0; i < nSym; i++ {
		s := i % 4
		r := float32(scale)
		switch s {
		case 0:
			syms[i] = complex(r, r)
		case 1:
			syms[i] = complex(-r, r)
		case 2:
			syms[i] = complex(-r, -r)
		case 3:
			syms[i] = complex(r, -r)
		}
	}
	// Upsample with raised-cosine pulses peaking at the *start*
	// of each symbol period so boundary sampling lands on a
	// non-zero gradient (same approach as the MuellerMuller
	// test). With peaks at i*sps, Gardner's default fire-time
	// at sample-index multiples of sps aligns with the symbol
	// peaks.
	src := make([]complex64, nSym*sps)
	for i := 0; i < nSym; i++ {
		for k := 0; k < sps; k++ {
			alpha := 0.5 + 0.5*math.Cos(math.Pi*float64(k)/float64(sps-1))
			src[i*sps+k] = syms[i] * complex(float32(alpha), 0)
		}
	}
	g := NewGardner(float64(sps), 0.03)
	got := g.Process(nil, src)
	if len(got) < nSym-3 || len(got) > nSym+3 {
		t.Fatalf("recovered symbol count = %d, want %d ± 3", len(got), nSym)
	}
	// After warmup, find the best polarity alignment and require a
	// high match rate.
	const warm = 30
	bestRate := 0.0
	for shift := -2; shift <= 2; shift++ {
		matches, total := 0, 0
		for i := warm; i < len(got); i++ {
			j := i + shift
			if j < 0 || j >= nSym {
				continue
			}
			total++
			// Quadrant comparison.
			gotQuad := quadrant(got[i])
			wantQuad := quadrant(syms[j])
			if gotQuad == wantQuad {
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
	if bestRate < 0.85 {
		t.Errorf("best quadrant match rate = %.2f, want > 0.85", bestRate)
	}
}

// TestGardnerLocksOnPhaseOffset: insert a fractional-sample phase
// offset by shifting the input by 0.4 samples. Gardner should pull
// the symbol clock toward the correct phase within a reasonable
// warmup window.
func TestGardnerLocksOnPhaseOffset(t *testing.T) {
	const sps = 8
	const nSym = 500
	const scale = 0.7071067811865475
	syms := make([]complex64, nSym)
	for i := 0; i < nSym; i++ {
		s := (i * 3) % 4
		r := float32(scale)
		switch s {
		case 0:
			syms[i] = complex(r, r)
		case 1:
			syms[i] = complex(-r, r)
		case 2:
			syms[i] = complex(-r, -r)
		case 3:
			syms[i] = complex(r, -r)
		}
	}
	// Shift the upsampled signal by inserting a partial-symbol
	// lead-in pad so the symbol clock doesn't align with sample 0.
	spsF := float64(sps)
	pad := int(spsF * 1.4)
	src := make([]complex64, nSym*sps+pad)
	for i := 0; i < nSym; i++ {
		for k := 0; k < sps; k++ {
			frac := float32(1.0 - math.Abs(float64(k)-float64(sps/2))/float64(sps/2))
			src[pad+i*sps+k] = syms[i] * complex(frac, 0)
		}
	}
	g := NewGardner(float64(sps), 0.05)
	got := g.Process(nil, src)
	// Allow a tighter range now since the input has a known shift.
	if len(got) < nSym-5 || len(got) > nSym+5 {
		t.Fatalf("recovered symbol count = %d (want ~%d)", len(got), nSym)
	}
	// Quadrant match rate over the second half (after lock) should
	// be high.
	const warm = 50
	bestRate := 0.0
	for shift := -3; shift <= 3; shift++ {
		matches, total := 0, 0
		for i := warm; i < len(got); i++ {
			j := i + shift
			if j < 0 || j >= nSym {
				continue
			}
			total++
			if quadrant(got[i]) == quadrant(syms[j]) {
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
	if bestRate < 0.80 {
		t.Errorf("best quadrant match rate after phase shift = %.2f, want > 0.80", bestRate)
	}
}

// TestGardnerProcessChunkedStreamMatchesContiguous: split the input
// across multiple Process calls and confirm the recovered symbol
// stream matches what a single contiguous call would produce. The
// stashed-tail logic is what makes this work.
func TestGardnerProcessChunkedStreamMatchesContiguous(t *testing.T) {
	const sps = 8
	const nSym = 200
	const scale = 0.7071067811865475
	src := make([]complex64, nSym*sps)
	for i := 0; i < nSym; i++ {
		var sym complex64
		s := i % 4
		r := float32(scale)
		switch s {
		case 0:
			sym = complex(r, r)
		case 1:
			sym = complex(-r, r)
		case 2:
			sym = complex(-r, -r)
		case 3:
			sym = complex(r, -r)
		}
		for k := 0; k < sps; k++ {
			frac := float32(1.0 - math.Abs(float64(k)-float64(sps/2))/float64(sps/2))
			src[i*sps+k] = sym * complex(frac, 0)
		}
	}

	// Contiguous reference.
	gRef := NewGardner(float64(sps), 0.03)
	ref := gRef.Process(nil, src)

	// Chunked: split at 1/3 and 2/3 points.
	gChunk := NewGardner(float64(sps), 0.03)
	a := len(src) / 3
	b := 2 * len(src) / 3
	out := gChunk.Process(nil, src[:a])
	out = append(out, gChunk.Process(nil, src[a:b])...)
	out = append(out, gChunk.Process(nil, src[b:])...)

	// Lengths within ±2 of the contiguous version.
	if abs(len(out)-len(ref)) > 2 {
		t.Errorf("chunked symbol count = %d, contiguous = %d (diff > 2)", len(out), len(ref))
	}
	// Quadrant agreement over the overlap.
	match := 0
	n := len(ref)
	if len(out) < n {
		n = len(out)
	}
	for i := 5; i < n; i++ {
		if quadrant(out[i]) == quadrant(ref[i]) {
			match++
		}
	}
	if n > 5 && float64(match)/float64(n-5) < 0.95 {
		t.Errorf("chunked vs contiguous quadrant agreement = %d/%d", match, n-5)
	}
}

func TestGardnerResetClearsState(t *testing.T) {
	g := NewGardner(8, 0.03)
	src := make([]complex64, 100)
	for i := range src {
		src[i] = complex64(cmplx.Rect(1, float64(i)*0.1))
	}
	g.Process(nil, src)
	if !g.have {
		t.Errorf("post-Process have = false, want true")
	}
	g.Reset()
	if g.have {
		t.Errorf("post-Reset have = true, want false")
	}
	if g.mu != g.sps {
		t.Errorf("post-Reset mu = %v, want %v", g.mu, g.sps)
	}
	if len(g.stashed) != 0 {
		t.Errorf("post-Reset stashed len = %d, want 0", len(g.stashed))
	}
}

func TestNewGardnerRejectsBadSps(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("NewGardner(1, ...) did not panic")
		}
	}()
	NewGardner(1, 0.02)
}

func TestNewGardnerDefaultsGain(t *testing.T) {
	g := NewGardner(8, 0)
	if g.gain != 0.02 {
		t.Errorf("default gain = %v, want 0.02", g.gain)
	}
	g = NewGardner(8, -1)
	if g.gain != 0.02 {
		t.Errorf("negative-gain default = %v, want 0.02", g.gain)
	}
}

func quadrant(c complex64) int {
	if real(c) >= 0 && imag(c) >= 0 {
		return 0
	}
	if real(c) < 0 && imag(c) >= 0 {
		return 1
	}
	if real(c) < 0 && imag(c) < 0 {
		return 2
	}
	return 3
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
