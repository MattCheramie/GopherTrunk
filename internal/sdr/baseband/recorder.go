package baseband

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// RecordingDevice wraps an sdr.Device and tees its IQ stream to a
// wideband WAV recording while passing every sample through to the
// normal consumer unchanged. A fresh WAV is opened each time StreamIQ
// is called and closed when that stream ends.
type RecordingDevice struct {
	inner sdr.Device
	dir   string
	log   *slog.Logger

	mu   sync.Mutex
	rate uint32
}

// NewRecordingDevice wraps inner so its IQ is recorded into dir.
func NewRecordingDevice(inner sdr.Device, dir string, log *slog.Logger) *RecordingDevice {
	if log == nil {
		log = slog.Default()
	}
	return &RecordingDevice{inner: inner, dir: dir, log: log}
}

// Inner returns the wrapped device.
func (r *RecordingDevice) Inner() sdr.Device { return r.inner }

func (r *RecordingDevice) Info() sdr.Info                { return r.inner.Info() }
func (r *RecordingDevice) SetCenterFreq(hz uint32) error { return r.inner.SetCenterFreq(hz) }
func (r *RecordingDevice) SetGain(tenthDB int) error     { return r.inner.SetGain(tenthDB) }
func (r *RecordingDevice) SetPPM(ppm int) error          { return r.inner.SetPPM(ppm) }
func (r *RecordingDevice) SetBiasTee(enable bool) error  { return r.inner.SetBiasTee(enable) }
func (r *RecordingDevice) Close() error                  { return r.inner.Close() }

// SetSampleRate records the rate (for the WAV header) and forwards it.
func (r *RecordingDevice) SetSampleRate(hz uint32) error {
	r.mu.Lock()
	r.rate = hz
	r.mu.Unlock()
	return r.inner.SetSampleRate(hz)
}

// StreamIQ starts the wrapped device's stream, opens a recording WAV,
// and returns a channel that forwards every chunk while writing it to
// disk. A recording-open failure is logged and degrades gracefully —
// the inner stream is returned unrecorded rather than failing.
func (r *RecordingDevice) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	in, err := r.inner.StreamIQ(ctx)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	rate := r.rate
	r.mu.Unlock()
	if rate == 0 {
		rate = sdr.DefaultSampleRateHz
	}

	if err := os.MkdirAll(r.dir, 0o755); err != nil {
		r.log.Warn("baseband: cannot create recording dir; streaming unrecorded",
			"dir", r.dir, "err", err)
		return in, nil
	}
	path := filepath.Join(r.dir, recordingName(r.inner.Info().Serial))
	w, err := NewIQWriter(path, rate)
	if err != nil {
		r.log.Warn("baseband: cannot open recording; streaming unrecorded",
			"path", path, "err", err)
		return in, nil
	}
	r.log.Info("baseband: recording started", "path", path, "rate_hz", rate)

	out := make(chan []complex64, 8)
	go func() {
		defer close(out)
		defer func() {
			if err := w.Close(); err != nil {
				r.log.Warn("baseband: close recording", "path", path, "err", err)
			} else {
				r.log.Info("baseband: recording stopped",
					"path", path, "bytes", w.BytesWritten())
			}
		}()
		for chunk := range in {
			if err := w.Write(chunk); err != nil {
				r.log.Warn("baseband: recording write failed; continuing unrecorded",
					"path", path, "err", err)
			}
			select {
			case out <- chunk:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

// recordingName builds a per-stream filename: serial + UTC timestamp.
func recordingName(serial string) string {
	s := sanitize(serial)
	if s == "" {
		s = "sdr"
	}
	return s + "_" + time.Now().UTC().Format("20060102T150405Z") + ".wav"
}

// sanitize strips path-hostile characters from a serial.
func sanitize(s string) string {
	s = strings.TrimSpace(s)
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '-' || r == '_':
			return r
		default:
			return '_'
		}
	}, s)
}
