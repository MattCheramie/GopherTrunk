package spectrum

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/fft"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/window"
)

// AverageDB builds a Welch-averaged dBFS spectrum over the whole of iq,
// returning a Frame ready for peak detection or display. It is the offline,
// one-shot analog of the streaming Producer: rather than emitting frames at a
// fixed rate it walks iq in non-overlapping n-sample windows, computes the
// dBFS power spectrum of each via PowerDB, and averages the per-bin dB values
// across all windows (scipy.signal.welch with a Hann window and zero overlap).
//
// n must be a power of two. When iq holds fewer than n samples a single
// zero-padded window is taken so a short capture still yields a frame. The
// returned Frame's CenterHz/SampleRate carry centerHz and the rounded
// sampleRateHz so callers can map bins back to absolute frequency.
func AverageDB(iq []complex64, centerHz uint32, sampleRateHz float64, n int) Frame {
	plan := fft.New(n)
	win := window.Hann(n)
	var winSum float64
	for _, w := range win {
		winSum += w * w
	}

	acc := make([]float64, n)
	bufIn := make([]complex128, n)
	bufOut := make([]complex128, n)
	dst := make([]float32, n)
	frames := 0
	for off := 0; off+n <= len(iq); off += n {
		PowerDB(plan, win, winSum, iq[off:off+n], bufIn, bufOut, dst)
		for i, v := range dst {
			acc[i] += float64(v)
		}
		frames++
	}

	bins := make([]float32, n)
	if frames == 0 {
		// Fewer than one FFT window: single padded pass so a short carrier
		// still produces a frame.
		pad := make([]complex64, n)
		copy(pad, iq)
		PowerDB(plan, win, winSum, pad, bufIn, bufOut, dst)
		copy(bins, dst)
	} else {
		for i := range bins {
			bins[i] = float32(acc[i] / float64(frames))
		}
	}
	return Frame{CenterHz: centerHz, SampleRate: uint32(math.Round(sampleRateHz)), Bins: bins}
}
