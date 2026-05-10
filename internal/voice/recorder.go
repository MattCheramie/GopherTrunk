package voice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// Recorder writes per-call audio + raw-frame files. It subscribes to
// events.KindCallStart and events.KindCallEnd from the trunking engine,
// opens a WAV (and optional raw-frame sidecar) for each new call, and
// closes them on call end. A future demod-pipeline composer will push
// PCM samples in via WritePCM and raw vocoder frames in via
// WriteRawFrame, keyed by device serial.
//
// Layout under OutDir (Trunk-Recorder-style):
//
//   <OutDir>/<system>/<talkgroup-or-decimal-id>/<UTC-RFC3339>_src<src>.wav
//   <OutDir>/<system>/<talkgroup-or-decimal-id>/<UTC-RFC3339>_src<src>.raw
//
// The raw sidecar is appended once per WriteRawFrame call. It is
// intentionally a flat concatenation of frames so users can BYO decoder
// (mbelib, DVSI, etc.) without parsing surrounding metadata.
//
// EDACS ProVoice grants (Grant.ProVoice == true) always force a `.raw`
// sidecar even when WriteRaw is false. The vocoder is patent +
// trade-secret encumbered so we cannot ship a built-in decoder; the
// sidecar lets researchers feed frames into an external decoder.
type Recorder struct {
	bus        *events.Bus
	log        *slog.Logger
	outDir     string
	sampleRate uint32
	writeRaw   bool

	mu        sync.Mutex
	sessions  map[string]*recordingSession // by device serial
	sub       *events.Subscription
	runDone   chan struct{}
	closeOnce sync.Once
}

// RecorderOptions configure a new Recorder.
type RecorderOptions struct {
	Bus        *events.Bus
	Log        *slog.Logger
	OutDir     string
	SampleRate uint32 // 8000 typical
	WriteRaw   bool   // emit a .raw sidecar alongside each .wav
}

// NewRecorder validates options and returns a recorder ready to Run.
// Like the engine, the recorder subscribes to the bus at construction
// so that CallStart events published before Run starts are not lost.
func NewRecorder(opts RecorderOptions) (*Recorder, error) {
	if opts.Bus == nil {
		return nil, errors.New("voice/recorder: events.Bus is required")
	}
	if opts.OutDir == "" {
		return nil, errors.New("voice/recorder: OutDir is required")
	}
	if opts.SampleRate == 0 {
		opts.SampleRate = pcmHzDefault
	}
	if opts.Log == nil {
		opts.Log = slog.Default()
	}
	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return nil, fmt.Errorf("voice/recorder: mkdir: %w", err)
	}
	r := &Recorder{
		bus:        opts.Bus,
		log:        opts.Log,
		outDir:     opts.OutDir,
		sampleRate: opts.SampleRate,
		writeRaw:   opts.WriteRaw,
		sessions:   make(map[string]*recordingSession),
		runDone:    make(chan struct{}),
	}
	r.sub = opts.Bus.Subscribe()
	return r, nil
}

// SessionCount returns the number of currently-open recording sessions.
// Useful in tests; takes the internal lock so it is race-free.
func (r *Recorder) SessionCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.sessions)
}

// HasSession reports whether a session exists for deviceSerial.
func (r *Recorder) HasSession(deviceSerial string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.sessions[deviceSerial]
	return ok
}

// Close releases the bus subscription, waits for Run (if running) to
// exit, then closes any outstanding sessions. Safe to call multiple
// times; second and later calls are no-ops.
func (r *Recorder) Close() error {
	var firstErr error
	r.closeOnce.Do(func() {
		// Subscription.Close is idempotent and signals Run to exit on its
		// next select.
		r.sub.Close()
		// If Run was started, wait for it to drain.
		select {
		case <-r.runDone:
		case <-time.After(time.Second):
		}
		r.mu.Lock()
		defer r.mu.Unlock()
		for serial, s := range r.sessions {
			if err := s.close(); err != nil && firstErr == nil {
				firstErr = err
			}
			delete(r.sessions, serial)
		}
	})
	return firstErr
}

// Run drains CallStart/CallEnd events until ctx cancels.
func (r *Recorder) Run(ctx context.Context) error {
	defer close(r.runDone)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-r.sub.C:
			if !ok {
				return nil
			}
			switch ev.Kind {
			case events.KindCallStart:
				if cs, ok := ev.Payload.(trunking.CallStart); ok {
					r.handleStart(cs)
				}
			case events.KindCallEnd:
				if ce, ok := ev.Payload.(trunking.CallEnd); ok {
					r.handleEnd(ce)
				}
			}
		}
	}
}

// WritePCM appends 16-bit PCM samples for the named device serial. If no
// session is open for that device the samples are dropped (the demod
// pipeline can race ahead of the CallStart event).
func (r *Recorder) WritePCM(deviceSerial string, samples []int16) error {
	r.mu.Lock()
	s, ok := r.sessions[deviceSerial]
	r.mu.Unlock()
	if !ok {
		return nil
	}
	return s.wav.WriteSamples(samples)
}

// WriteRawFrame appends a raw vocoder frame to the per-call sidecar.
// The session decides whether a sidecar exists: it is opened either when
// WriteRaw is globally enabled or when the call's grant is flagged
// ProVoice. Frames for a session without a sidecar are dropped silently.
func (r *Recorder) WriteRawFrame(deviceSerial string, frame []byte) error {
	r.mu.Lock()
	s, ok := r.sessions[deviceSerial]
	r.mu.Unlock()
	if !ok || s.raw == nil {
		return nil
	}
	_, err := s.raw.Write(frame)
	return err
}

func (r *Recorder) handleStart(cs trunking.CallStart) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, busy := r.sessions[cs.DeviceSerial]; busy {
		// Engine should have ended the prior call first, but be defensive.
		r.log.Warn("recorder: device already has session, replacing",
			"device", cs.DeviceSerial)
		_ = r.sessions[cs.DeviceSerial].close()
		delete(r.sessions, cs.DeviceSerial)
	}
	dir := r.directoryFor(cs)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		r.log.Error("recorder: mkdir", "dir", dir, "err", err)
		return
	}
	base := r.basenameFor(cs)
	wavPath := filepath.Join(dir, base+".wav")
	wav, err := NewWavFile(wavPath, r.sampleRate)
	if err != nil {
		r.log.Error("recorder: open wav", "path", wavPath, "err", err)
		return
	}
	s := &recordingSession{wav: wav, wavPath: wavPath, startedAt: cs.StartedAt}
	// ProVoice grants always get a sidecar — the vocoder isn't decodable
	// in-process, so the .raw file is the only way to capture the call.
	if r.writeRaw || cs.Grant.ProVoice {
		rawPath := filepath.Join(dir, base+".raw")
		raw, err := os.Create(rawPath)
		if err != nil {
			r.log.Error("recorder: open raw", "path", rawPath, "err", err)
		} else {
			s.raw = raw
			s.rawPath = rawPath
		}
	}
	r.sessions[cs.DeviceSerial] = s
	r.log.Info("recorder: call started",
		"device", cs.DeviceSerial, "wav", wavPath,
		"tg", cs.Grant.GroupID, "provoice", cs.Grant.ProVoice)
}

func (r *Recorder) handleEnd(ce trunking.CallEnd) {
	r.mu.Lock()
	s, ok := r.sessions[ce.DeviceSerial]
	if ok {
		delete(r.sessions, ce.DeviceSerial)
	}
	r.mu.Unlock()
	if !ok {
		return
	}
	if err := s.close(); err != nil {
		r.log.Error("recorder: close session", "err", err)
	}
	r.log.Info("recorder: call ended",
		"device", ce.DeviceSerial,
		"wav", s.wavPath,
		"duration", ce.Duration().Round(time.Millisecond),
		"reason", ce.Reason)
}

func (r *Recorder) directoryFor(cs trunking.CallStart) string {
	system := sanitize(cs.Grant.System)
	if system == "" {
		system = "unknown-system"
	}
	tgDir := fmt.Sprintf("%d", cs.Grant.GroupID)
	if cs.Talkgroup != nil && cs.Talkgroup.AlphaTag != "" {
		tgDir = sanitize(cs.Talkgroup.AlphaTag)
	}
	return filepath.Join(r.outDir, system, tgDir)
}

func (r *Recorder) basenameFor(cs trunking.CallStart) string {
	t := cs.StartedAt.UTC()
	if t.IsZero() {
		t = time.Now().UTC()
	}
	stamp := t.Format("20060102T150405Z")
	return fmt.Sprintf("%s_src%d", stamp, cs.Grant.SourceID)
}

// sanitize strips characters that are awkward in file paths across OSes.
func sanitize(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	mapper := func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-' || r == '_' || r == '.':
			return r
		default:
			return '_'
		}
	}
	return strings.Map(mapper, s)
}

type recordingSession struct {
	wav       *WavWriter
	wavPath   string
	raw       *os.File
	rawPath   string
	startedAt time.Time
}

func (s *recordingSession) close() error {
	var firstErr error
	if err := s.wav.Close(); err != nil {
		firstErr = err
	}
	if s.raw != nil {
		if err := s.raw.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
