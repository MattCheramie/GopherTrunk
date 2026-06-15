package survey

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// loadIQWav reads a canonical baseband RIFF/WAVE file (2×16-bit signed PCM =
// interleaved I/Q) into []complex64. It is a small test-local reader so the
// survey package stays free of an internal/sdr/baseband import (which would
// invert the package dependency direction). It mirrors the canonical reader in
// internal/sdr/baseband/wav.go.
func loadIQWav(path string) ([]complex64, uint32, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	hdr := make([]byte, 12)
	if _, err = io.ReadFull(f, hdr); err != nil {
		return nil, 0, err
	}
	if string(hdr[0:4]) != "RIFF" || string(hdr[8:12]) != "WAVE" {
		return nil, 0, errors.New("not a RIFF/WAVE file")
	}
	var (
		rate    uint32
		chans   uint16
		bits    uint16
		gotFmt  bool
		chunkHd = make([]byte, 8)
	)
	for {
		if _, err = io.ReadFull(f, chunkHd); err != nil {
			return nil, 0, errors.New("WAV ended before a data chunk")
		}
		id := string(chunkHd[0:4])
		size := binary.LittleEndian.Uint32(chunkHd[4:8])
		switch id {
		case "fmt ":
			buf := make([]byte, size)
			if _, err = io.ReadFull(f, buf); err != nil {
				return nil, 0, err
			}
			if len(buf) < 16 {
				return nil, 0, errors.New("short fmt chunk")
			}
			chans = binary.LittleEndian.Uint16(buf[2:4])
			rate = binary.LittleEndian.Uint32(buf[4:8])
			bits = binary.LittleEndian.Uint16(buf[14:16])
			gotFmt = true
		case "data":
			if !gotFmt {
				return nil, 0, errors.New("data chunk before fmt chunk")
			}
			if chans != 2 || bits != 16 {
				return nil, 0, errors.New("IQ WAV must be 2-channel 16-bit")
			}
			raw := make([]byte, size)
			n, _ := io.ReadFull(f, raw)
			raw = raw[:n-n%4]
			iq := make([]complex64, len(raw)/4)
			for i := range iq {
				ii := int16(binary.LittleEndian.Uint16(raw[i*4 : i*4+2]))
				qq := int16(binary.LittleEndian.Uint16(raw[i*4+2 : i*4+4]))
				iq[i] = complex(float32(ii)/32768, float32(qq)/32768)
			}
			return iq, rate, nil
		default:
			skip := int64(size)
			if size%2 == 1 {
				skip++
			}
			if _, err = f.Seek(skip, io.SeekCurrent); err != nil {
				return nil, 0, err
			}
		}
	}
}

// tetraCapturePath is the in-tree, git-tracked on-air TETRA downlink capture.
func tetraCapturePath() string {
	return filepath.Join("..", "..", "samples", "tetra", "TETRA IQ.wav")
}

// loadTetraCapture loads the committed capture or skips the test.
func loadTetraCapture(t *testing.T) ([]complex64, uint32) {
	t.Helper()
	iq, rate, err := loadIQWav(tetraCapturePath())
	if err != nil {
		t.Skipf("TETRA capture unavailable (%v)", err)
	}
	if len(iq) < minClassifySamples {
		t.Skipf("TETRA capture too short: %d samples", len(iq))
	}
	return iq, rate
}

// TestClassifyTetraRealAir guards the issue #648 family directly against the
// committed on-air TETRA capture: a real π/4-DQPSK control carrier must classify
// as a digital modulation, never analog nbfm (which would keep it from reaching
// the siglab identify the colour-code recovery in #648 lives in). The capture is
// 48 kHz IQ — equal to the survey channel rate — so it feeds straight into the
// classifier with no decimation. The carrier's 18 kHz symbol line is the
// load-bearing feature; assert the classifier finds it and routes digital.
func TestClassifyTetraRealAir(t *testing.T) {
	iq, rate := loadTetraCapture(t)

	cls := Classify(iq, float64(rate))
	t.Logf("class=%s conf=%.2f bw=%d baud=%.0f prom=%.1f kurt=%.2f mod=%d",
		cls.Class, cls.Confidence, cls.Features.OccupiedBwHz, cls.Features.BaudHz,
		cls.Features.BaudProminence, cls.Features.IFKurtosis, cls.Features.IFModality)

	if cls.Class == ClassNBFM {
		t.Fatalf("TETRA capture classified as nbfm; want a digital class (issue #648)")
	}
	if !IsDigital(cls.Class) {
		t.Fatalf("Classify() = %q, want a digital class", cls.Class)
	}
	if cls.Features.BaudHz < 17_000 || cls.Features.BaudHz > 19_000 {
		t.Errorf("baud line = %.0f Hz, want TETRA's ~18 kHz", cls.Features.BaudHz)
	}
}

// TestClassifyTetraRealAirNoisy is the regression-margin guard: it confirms the
// 18 kHz symbol line stays well clear of the digital-prominence gate even after
// the capture is degraded to field-realistic SNRs and sliced to a survey-length
// window. This pins the conclusion that the baud-line detection is robust on this
// signal (the misclassification risk flagged in #648 is not a baud-line-floor
// sensitivity problem on a present carrier), so a future change that weakens it
// trips here.
func TestClassifyTetraRealAirNoisy(t *testing.T) {
	iq, rate := loadTetraCapture(t)

	// One survey-window slice (~0.68 s); baudLine itself caps its FFT at 16384
	// discriminator samples, so this covers the windowed path.
	if len(iq) > 32768 {
		iq = iq[:32768]
	}
	for _, snrDb := range []float64{20, 15, 12, 10} {
		noisy := addNoiseToSNR(iq, snrDb, 1)
		cls := Classify(noisy, float64(rate))
		t.Logf("snr~%.0fdB class=%s prom=%.1f baud=%.0f", snrDb, cls.Class,
			cls.Features.BaudProminence, cls.Features.BaudHz)
		if cls.Class == ClassNBFM || !IsDigital(cls.Class) {
			t.Errorf("snr~%.0fdB: class=%q, want a digital class", snrDb, cls.Class)
		}
		if cls.Features.BaudProminence < DefaultClassifyConfig().DigitalProminence {
			t.Errorf("snr~%.0fdB: baud prominence %.1f below the digital gate %.1f",
				snrDb, cls.Features.BaudProminence, DefaultClassifyConfig().DigitalProminence)
		}
	}
}

// addNoiseToSNR adds complex-Gaussian noise scaled to bring iq to approximately
// targetSNRDb, returning a new slice. Deterministic given seed.
func addNoiseToSNR(iq []complex64, targetSNRDb float64, seed int64) []complex64 {
	var sigP float64
	for _, s := range iq {
		sigP += float64(real(s)*real(s) + imag(s)*imag(s))
	}
	sigP /= float64(len(iq))
	noiseP := sigP / math.Pow(10, targetSNRDb/10)
	sigma := math.Sqrt(noiseP / 2)
	rng := newRng(seed)
	out := make([]complex64, len(iq))
	for i, s := range iq {
		out[i] = s + complex(float32(rng.normal()*sigma), float32(rng.normal()*sigma))
	}
	return out
}

// rngState is a tiny deterministic LCG + Box-Muller gaussian so the noisy guard
// is reproducible without depending on math/rand's global stream.
type rngState struct{ s uint64 }

func newRng(seed int64) *rngState { return &rngState{s: uint64(seed)*2862933555777941757 + 1} }

func (r *rngState) f64() float64 {
	r.s = r.s*6364136223846793005 + 1442695040888963407
	return float64(r.s>>11) / float64(uint64(1)<<53)
}

func (r *rngState) normal() float64 {
	u1 := r.f64()
	if u1 < 1e-12 {
		u1 = 1e-12
	}
	u2 := r.f64()
	return math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
}
