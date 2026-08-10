package receiver

import (
	"math"
	"math/cmplx"
	"testing"
)

// TestDCBlockRemovesConstantOffset feeds a pure tone riding on a large static
// DC offset — the zero-IF-spur scenario the block exists for — and checks the
// output has the DC stripped while the tone amplitude survives. A constant IQ
// bias like this is what collapses the FM-discriminator eye upstream of the
// slicer (see dcblock.go); the block must remove it without eating the signal.
func TestDCBlockRemovesConstantOffset(t *testing.T) {
	const (
		n    = 200_000 // long enough for the ~1.2 Hz-corner high-pass to settle
		fs   = 78125.0
		tone = 1800.0 // a C4FM outer-symbol tone; well above the block's corner
		amp  = 0.16    // representative signal level (mean|x| of a real capture)
		dc   = 0.10    // static DC spur — >0.05 zeroes real P25 voice decode
	)
	src := make([]complex64, n)
	for i := range src {
		ph := 2 * math.Pi * tone * float64(i) / fs
		src[i] = complex64(complex(amp*math.Cos(ph), amp*math.Sin(ph))) + complex(dc, 0)
	}

	var d dcBlock
	out := d.process(nil, src)

	// Measure over the settled tail only (skip the first 20k samples).
	const skip = 20_000
	var meanOut complex128
	var rmsOut float64
	for i := skip; i < n; i++ {
		meanOut += complex128(out[i])
		re, im := float64(real(out[i])), float64(imag(out[i]))
		rmsOut += re*re + im*im
	}
	cnt := float64(n - skip)
	meanOut /= complex(cnt, 0)
	rmsOut = math.Sqrt(rmsOut / cnt)

	// DC must be gone (well below the tone amplitude).
	if got := cmplx.Abs(meanOut); got > amp*0.05 {
		t.Errorf("residual DC after block = %.4f, want ≤ %.4f (5%% of tone amp)", got, amp*0.05)
	}
	// The tone (a pure sinusoid, RMS = amp) must be preserved within a few %.
	if math.Abs(rmsOut-amp) > amp*0.05 {
		t.Errorf("tone RMS after block = %.4f, want %.4f ±5%% (passband must be untouched)", rmsOut, amp)
	}
}

// TestEnableDCBlockInitialisedForBothDemodModes guards the regression where the
// DC blocker was initialised only inside New's C4FM branch. Process applies the
// blocker before the demod-mode split, and the voice composer requests it for
// every P25 Phase 1 voice chain — including an LSM/simulcast site that resolves
// to DemodCQPSK. If the init is mode-gated, a CQPSK voice receiver silently
// drops the block it asked for. Both modes must end up with r.dcBlk != nil.
func TestEnableDCBlockInitialisedForBothDemodModes(t *testing.T) {
	for _, mode := range []DemodMode{DemodC4FM, DemodCQPSK} {
		r := New(Options{
			SampleRateHz:  48000,
			DeviationHz:   1800.0,
			DemodMode:     mode,
			EnableDCBlock: true,
			DibitSink:     func([]uint8, int) {},
		})
		if r.dcBlk == nil {
			t.Errorf("DemodMode %v: EnableDCBlock set but r.dcBlk is nil (block silently dropped)", mode)
		}
	}
}

// TestDCBlockReset clears the filter memory so a re-sync starts clean.
func TestDCBlockReset(t *testing.T) {
	var d dcBlock
	d.process(nil, []complex64{complex(0.5, 0.5), complex(0.5, 0.5)})
	if d.prevX == 0 && d.prevY == 0 {
		t.Fatal("state not advanced by process")
	}
	d.Reset()
	if d.prevX != 0 || d.prevY != 0 {
		t.Errorf("Reset left state prevX=%v prevY=%v, want 0", d.prevX, d.prevY)
	}
}
