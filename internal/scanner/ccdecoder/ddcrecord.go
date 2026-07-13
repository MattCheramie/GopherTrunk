package ccdecoder

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/baseband"
)

// ddcRecorder tees the decoder's post-DDC narrowband IQ (the exact
// channelized complex stream the active pipeline decodes) to a two-channel
// 16-bit baseband WAV. It is the live half of the "record the digital
// down-converter output" tooling: instead of a fat wideband capture at the
// full SDR rate, the operator gets a small file at the pipeline rate (144 kHz
// for TETRA, ~48 kHz for the C4FM family) of exactly the signal that failed
// to decode — small enough to share and replay with `replay -format wav`.
//
// It is owned by the pump goroutine (all calls happen under Decoder.mu), so
// it needs no locking of its own. A new file is opened lazily on the first
// chunk and rolled whenever the system name or the pipeline rate changes (a
// retune to a different protocol/rate), so each recording holds one
// contiguous channelized stream at one rate.
type ddcRecorder struct {
	dir string
	log *slog.Logger

	w      *baseband.IQWriter
	path   string
	system string
	rate   uint32
}

// newDDCRecorder returns a recorder writing into dir, or nil if dir is empty
// (the zero-cost disabled case).
func newDDCRecorder(dir string, log *slog.Logger) *ddcRecorder {
	if strings.TrimSpace(dir) == "" {
		return nil
	}
	if log == nil {
		log = slog.Default()
	}
	return &ddcRecorder{dir: dir, log: log}
}

// write appends one channelized chunk, opening or rolling the WAV as needed.
// system is the active system name (for the filename) and rateHz is the DDC
// output rate. A write/open failure is logged once and disables further
// recording rather than disrupting decode.
func (r *ddcRecorder) write(system string, rateHz float64, samples []complex64) {
	if r == nil || len(samples) == 0 {
		return
	}
	rate := uint32(rateHz + 0.5)
	if rate == 0 {
		return
	}
	if r.w == nil || r.system != system || r.rate != rate {
		r.roll(system, rate)
		if r.w == nil {
			return // open failed; roll already logged and left us disabled
		}
	}
	if err := r.w.Write(samples); err != nil {
		r.log.Warn("ccdecoder: DDC recording write failed; recording stopped",
			"path", r.path, "err", err)
		r.closeCurrent()
		r.dir = "" // disable further rolls after a write error
	}
}

// roll closes the current file (if any) and opens a fresh one for the given
// system + rate.
func (r *ddcRecorder) roll(system string, rate uint32) {
	r.closeCurrent()
	if r.dir == "" {
		return
	}
	if err := os.MkdirAll(r.dir, 0o755); err != nil {
		r.log.Warn("ccdecoder: cannot create DDC recording dir; recording disabled",
			"dir", r.dir, "err", err)
		r.dir = ""
		return
	}
	name := fmt.Sprintf("%s_%s_%dhz.wav", ddcRecSanitize(system),
		time.Now().UTC().Format("20060102T150405Z"), rate)
	path := filepath.Join(r.dir, name)
	w, err := baseband.NewIQWriter(path, rate)
	if err != nil {
		r.log.Warn("ccdecoder: cannot open DDC recording; recording disabled",
			"path", path, "err", err)
		r.dir = ""
		return
	}
	r.w = w
	r.path = path
	r.system = system
	r.rate = rate
	r.log.Info("ccdecoder: DDC recording started", "path", path, "rate_hz", rate)
}

// closeCurrent closes the open WAV (if any) and clears the handle.
func (r *ddcRecorder) closeCurrent() {
	if r.w == nil {
		return
	}
	if err := r.w.Close(); err != nil {
		r.log.Warn("ccdecoder: close DDC recording", "path", r.path, "err", err)
	} else {
		r.log.Info("ccdecoder: DDC recording stopped",
			"path", r.path, "bytes", r.w.BytesWritten())
	}
	r.w = nil
}

// close finalizes the recording; safe to call when disabled or already closed.
func (r *ddcRecorder) close() {
	if r == nil {
		return
	}
	r.closeCurrent()
}

// ddcRecSanitize strips path-hostile characters from a system name for use in
// a filename, mirroring the baseband recorder's serial sanitizer.
func ddcRecSanitize(s string) string {
	s = strings.TrimSpace(s)
	out := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '-' || r == '_':
			return r
		default:
			return '_'
		}
	}, s)
	if out == "" {
		return "system"
	}
	return out
}
