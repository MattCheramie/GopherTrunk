package sync

import (
	"math"
	"math/cmplx"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
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

// TestGardnerConvergesOnEyeCentre runs random RRC-shaped QPSK (the matched
// filter's output, as every Gardner caller feeds it) from several starting
// phases and requires the loop to settle on the eye centre. The loop used to
// apply its error with the wrong sign: the centre was an unstable point and
// it walked to the transition zero crossing from every start, half a symbol
// off, closing the eye after a few thousand symbols. That is what wrecked
// every P25 Phase 2 call a few seconds in (8 sps at 48 kHz). The old tests
// stayed green because their triangle pulse and quadrant check survive a
// half-symbol error.
func TestGardnerConvergesOnEyeCentre(t *testing.T) {
	const sps = 8
	const n = 40000
	x := uint32(7)
	up := make([]complex64, n*sps)
	for i := 0; i < n; i++ {
		x = x*1664525 + 1013904223
		ph := float64(x>>30)*math.Pi/2 + math.Pi/4
		up[i*sps] = complex(float32(math.Cos(ph)), float32(math.Sin(ph)))
	}
	taps := filter.RootRaisedCosine(sps, 8, 0.2)
	rx := filter.NewFIR(taps).Process(nil, filter.NewFIR(taps).Process(nil, up))
	for start := 0; start < sps; start += 2 {
		g := NewGardner(sps, 0.005)
		var out []complex64
		src := rx[start:]
		for off := 0; off < len(src); off += 4096 {
			out = append(out, g.Process(nil, src[off:min(off+4096, len(src))])...)
		}
		// At the eye centre every symbol sits on a QPSK point: |re| and
		// |im| cluster tightly about their mean. At the zero crossing
		// they spread across the whole range.
		tail := out[len(out)-5000:]
		var m, v float64
		for _, s := range tail {
			m += math.Abs(float64(real(s))) + math.Abs(float64(imag(s)))
		}
		m /= 2 * float64(len(tail))
		for _, s := range tail {
			a, b := math.Abs(float64(real(s)))-m, math.Abs(float64(imag(s)))-m
			v += a*a + b*b
		}
		if spread := math.Sqrt(v/float64(2*len(tail))) / m; spread > 0.1 {
			t.Errorf("start %d: relative spread %.3f after %d symbols, want < 0.1 (loop off the eye centre, mu %.2f)", start, spread, len(out), g.Mu())
		}
	}
}
