package spectrum

import (
	"github.com/MattCheramie/GopherTrunk/internal/dsp/fft"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/window"
)

// Spectrogram computes the short-time Fourier transform of iq as a sequence of
// FFT-shifted dBFS frames — the time-frequency plane behind a waterfall plot
// (the scipy.signal.stft / matplotlib specgram analog). It slides a Hann window
// of length fftSize by hop samples, computing each frame's power spectrum via
// PowerDB, so each returned row uses the same dBFS normalization and FFT-shift
// (DC in the middle) as AverageDB and the streaming Producer.
//
// fftSize must be a power of two; hop <= 0 defaults to fftSize/2 (50% overlap).
// The result is frames×fftSize: rows are time steps, columns are frequency bins
// (column 0 = centerFreq - sampleRate/2). Returns nil when iq holds fewer than
// fftSize samples.
func Spectrogram(iq []complex64, fftSize, hop int) [][]float32 {
	if fftSize <= 0 || len(iq) < fftSize {
		return nil
	}
	if hop <= 0 {
		hop = fftSize / 2
	}
	if hop < 1 {
		hop = 1
	}
	plan := fft.New(fftSize)
	win := window.Hann(fftSize)
	var winSum float64
	for _, w := range win {
		winSum += w * w
	}
	bufIn := make([]complex128, fftSize)
	bufOut := make([]complex128, fftSize)

	var rows [][]float32
	for off := 0; off+fftSize <= len(iq); off += hop {
		row := make([]float32, fftSize)
		PowerDB(plan, win, winSum, iq[off:off+fftSize], bufIn, bufOut, row)
		rows = append(rows, row)
	}
	return rows
}
