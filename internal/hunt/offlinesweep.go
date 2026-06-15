package hunt

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/fft"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/spectrum"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/window"
)

// offlineCarrier is a carrier detected in a recorded wideband capture: its
// offset from the capture's centre (Hz, signed) and how far it stands above the
// noise floor. Offsets are signed because an offline capture has no SDR to
// retune — a control channel can sit either side of the recorded centre.
type offlineCarrier struct {
	OffsetHz int64
	SNRDb    float32
}

// offlineSweepMaxFrames bounds how many FFT frames are averaged for the
// spectrum estimate — enough to smooth the noise floor without scanning a long
// capture end to end.
const offlineSweepMaxFrames = 64

// detectOfflineCarriers finds every carrier in a recorded wideband IQ buffer by
// averaging a windowed power spectrum and running the same peak detector the
// live sweeper uses (DetectPeaks) — but on an in-memory buffer, with no SDR
// retune. It returns the carriers as signed offsets from DC so the caller can
// frequency-shift each to baseband. This is what lets an offline survey find a
// control channel that is off-centre (and not the loudest carrier) in a single
// wideband grab, instead of treating the whole capture as one channel.
func detectOfflineCarriers(iq []complex64, rate uint32, fftSize int, peakOpts PeakOptions) []offlineCarrier {
	if fftSize <= 0 {
		fftSize = 4096
	}
	if len(iq) < fftSize || rate == 0 {
		return nil
	}
	win := window.Hann(fftSize)
	var winSum float64
	for _, w := range win {
		winSum += w * w
	}
	plan := fft.New(fftSize)
	bufIn := make([]complex128, fftSize)
	bufOut := make([]complex128, fftSize)
	bins := make([]float32, fftSize)
	acc := make([]float64, fftSize)

	frames := 0
	for off := 0; off+fftSize <= len(iq) && frames < offlineSweepMaxFrames; off += fftSize {
		spectrum.PowerDB(plan, win, winSum, iq[off:off+fftSize], bufIn, bufOut, bins)
		for i, v := range bins {
			acc[i] += math.Pow(10, float64(v)/10) // average linear power, not dB
		}
		frames++
	}
	if frames == 0 {
		return nil
	}
	avg := make([]float32, fftSize)
	for i := range acc {
		avg[i] = float32(10 * math.Log10(acc[i]/float64(frames)))
	}

	// DetectPeaks maps bins to absolute Hz as centre + binOffset and returns a
	// uint32 frequency, which would underflow for a carrier below a near-zero
	// centre. Use a synthetic centre of one full sample rate so every carrier in
	// ±rate/2 maps to a positive frequency, then recover the signed offset.
	frame := spectrum.Frame{CenterHz: rate, SampleRate: rate, Bins: avg}
	peaks := DetectPeaks(frame, peakOpts)
	out := make([]offlineCarrier, 0, len(peaks))
	for _, p := range peaks {
		out = append(out, offlineCarrier{OffsetHz: int64(p.FreqHz) - int64(rate), SNRDb: p.SNRDb})
	}
	return out
}

// offlineNeighborHz returns the smallest absolute distance from carriers[i] to
// any other detected carrier, or 0 when it stands alone — the spacing used to
// bound the occupied-bandwidth walk so a dense channel grid doesn't bridge
// neighbours into one span (mirrors nearestNeighborHz for the swept path).
func offlineNeighborHz(carriers []offlineCarrier, i int) uint32 {
	if i < 0 || i >= len(carriers) {
		return 0
	}
	var best int64
	for j, c := range carriers {
		if j == i {
			continue
		}
		d := c.OffsetHz - carriers[i].OffsetHz
		if d < 0 {
			d = -d
		}
		if best == 0 || d < best {
			best = d
		}
	}
	return uint32(best)
}
