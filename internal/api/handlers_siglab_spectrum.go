package api

import (
	"net/http"
	"strconv"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/spectrum"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// iqPointsToComplex converts the JSON-friendly IQPoint taps to complex64 for
// the DSP primitives.
func iqPointsToComplex(pts []siglab.IQPoint) []complex64 {
	out := make([]complex64, len(pts))
	for i, p := range pts {
		out[i] = complex(p.I, p.Q)
	}
	return out
}

// queryPow2 returns a power-of-two query int (>= 8), or def when absent/invalid.
func queryPow2(r *http.Request, key string, def int) int {
	if s := r.URL.Query().Get(key); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 8 && v&(v-1) == 0 {
			return v
		}
	}
	return def
}

// handleSiglabJobPSD answers GET /api/v1/siglab/jobs/{id}/psd with the
// Welch-averaged power spectrum (dBFS, FFT-shifted: bin 0 = -rate/2) of the
// job's decimated IQ. Computed server-side via spectrum.AverageDB so the web UI
// renders the PSD without an in-browser FFT library.
func (s *Server) handleSiglabJobPSD(w http.ResponseWriter, r *http.Request) {
	taps, ok := s.siglabJobTaps(w, r)
	if !ok {
		return
	}
	fftSize := queryPow2(r, "fft", 1024)
	frame := spectrum.AverageDB(iqPointsToComplex(taps.DecimatedIQ), 0, taps.DecimatedRateHz, fftSize)
	writeJSON(w, http.StatusOK, map[string]any{
		"sample_rate_hz": taps.DecimatedRateHz,
		"fft_size":       fftSize,
		"bins":           frame.Bins,
	})
}

// handleSiglabJobSpectrogram answers GET /api/v1/siglab/jobs/{id}/spectrogram
// with the STFT (time-frequency dBFS plane) of the job's decimated IQ, computed
// server-side via spectrum.Spectrogram.
func (s *Server) handleSiglabJobSpectrogram(w http.ResponseWriter, r *http.Request) {
	taps, ok := s.siglabJobTaps(w, r)
	if !ok {
		return
	}
	fftSize := queryPow2(r, "fft", 256)
	hop := fftSize / 2
	if hs := r.URL.Query().Get("hop"); hs != "" {
		if v, err := strconv.Atoi(hs); err == nil && v > 0 {
			hop = v
		}
	}
	z := spectrum.Spectrogram(iqPointsToComplex(taps.DecimatedIQ), fftSize, hop)
	writeJSON(w, http.StatusOK, map[string]any{
		"sample_rate_hz": taps.DecimatedRateHz,
		"fft_size":       fftSize,
		"hop":            hop,
		"frames":         len(z),
		"z":              z,
	})
}

// siglabJobTaps fetches a job's captured IQ taps, writing the appropriate error
// response and returning ok=false when the job or its IQ is absent.
func (s *Server) siglabJobTaps(w http.ResponseWriter, r *http.Request) (*siglab.IQTaps, bool) {
	job, ok := s.siglab.getJob(r.PathValue("id"))
	if !ok {
		s.writeError(w, http.StatusNotFound, "siglab: job not found")
		return nil, false
	}
	job.mu.Lock()
	taps := job.iqTaps
	job.mu.Unlock()
	if taps == nil || len(taps.DecimatedIQ) == 0 {
		s.writeError(w, http.StatusNotFound, "siglab: no IQ captured (run with capture_iq=true)")
		return nil, false
	}
	return taps, true
}
