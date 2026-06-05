package equalizer

import (
	"math"
	"testing"
)

// TestFSECentreSpikePassthrough verifies the freshly-constructed FSE is the
// identity: with the centre-spike taps and no adaptation pressure, the on-time
// sample passes through (after the fixed group delay) unchanged.
func TestFSECentreSpikePassthrough(t *testing.T) {
	f := NewFSE(6, 0, 1.0, 0) // step 0 → taps never move
	// Feed on-time samples; mids are zero. Output of symbol n appears after
	// the centre-tap latency (center/2 symbols). Drive enough symbols to flush.
	want := []complex64{complex(1, 0), complex(0, 1), complex(-1, 0), complex(0, -1)}
	var got []complex64
	for i := 0; i < 16; i++ {
		on := want[i%len(want)]
		y, _ := f.Process(0, on)
		got = append(got, y)
	}
	// Once flushed, y must equal the on-time sample delayed by center/2 symbols
	// (center taps span; each symbol pushes 2 samples). Find the constant delay.
	delay := -1
	for d := 0; d < 8; d++ {
		ok := true
		for i := d; i < len(got); i++ {
			exp := want[(i-d)%len(want)]
			if cmplxAbs(got[i]-exp) > 1e-5 {
				ok = false
				break
			}
		}
		if ok {
			delay = d
			break
		}
	}
	if delay < 0 {
		t.Fatalf("centre-spike FSE is not a pure delayed passthrough; got=%v", got)
	}
}

// TestFSEOpensFractionalISI checks the FSE adapts to undo a fractional (T/2)
// channel a symbol-spaced equalizer could not: a half-symbol-delayed echo
// applied at 2 sps. After adaptation the recovered symbol modulus settles near
// the target (the CMA error proxy goes toward zero).
func TestFSEOpensFractionalISI(t *testing.T) {
	f := NewFSE(8, 0.01, 1.0, 1e-4)
	// Unit-modulus QPSK symbols, each represented by [mid, on] T/2 samples.
	// Channel: on-time = sym, mid = 0.6*previous-sym (a half-symbol echo).
	syms := []complex64{complex(1, 0), complex(0, 1), complex(-1, 0), complex(0, -1)}
	rng := uint32(12345)
	next := func() complex64 {
		rng = rng*1664525 + 1013904223
		return syms[(rng>>16)&3]
	}
	var prev complex64
	var lastErr float32
	for i := 0; i < 8000; i++ {
		s := next()
		mid := complex(0.6*real(prev), 0.6*imag(prev))
		_, e := f.Process(mid, s)
		lastErr = e
		prev = s
	}
	if math.Abs(float64(lastErr)) > 0.15 {
		t.Errorf("FSE did not open the fractional-ISI channel: final |err|=%.4f, want <= 0.15", lastErr)
	}
}

func cmplxAbs(c complex64) float64 {
	return math.Hypot(float64(real(c)), float64(imag(c)))
}
