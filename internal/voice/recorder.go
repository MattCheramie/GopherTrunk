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
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// Recorder writes per-call audio + raw-frame files. It subscribes to
// events.KindCallStart and events.KindCallEnd from the trunking engine,
// opens a WAV (and optional raw-frame sidecar) for each new call, and
// closes them on call end. The demod-pipeline composer pushes PCM
// samples in via WritePCM (analog protocols) and raw vocoder frames
// in via WriteRawFrame (digital protocols), keyed by device serial.
//
// Layout under OutDir (Trunk-Recorder-style):
//
//   <OutDir>/<system>/<talkgroup-or-decimal-id>/<UTC-RFC3339>_src<src>.wav
//   <OutDir>/<system>/<talkgroup-or-decimal-id>/<UTC-RFC3339>_src<src>.raw
//
// The raw sidecar is appended once per WriteRawFrame call. It is
// intentionally a flat concatenation of frames so users can BYO decoder
// (external libmbe, DVSI hardware, etc.) without parsing surrounding
// metadata.
//
// Per-call vocoder: when Grant.Protocol matches an entry in the
// configured VocoderForProtocol map, the recorder instantiates a
// fresh Vocoder from voice.DefaultRegistry on CallStart and decodes
// each WriteRawFrame call through it, writing the resulting PCM into
// the WAV. This makes captures of P25 / DMR / NXDN voice produce
// playable WAVs alongside the optional raw sidecar — out-of-band
// decode via `gophertrunk decode` remains available for operators
// who want bit-exact mbelib / DSD-FME output.
//
// EDACS ProVoice grants (Grant.ProVoice == true) always force a `.raw`
// sidecar even when WriteRaw is false. The ProVoice vocoder is patent
// + trade-secret encumbered so we cannot ship a built-in decoder;
// the sidecar lets researchers feed frames into an external decoder.
type Recorder struct {
	bus                *events.Bus
	log                *slog.Logger
	outDir             string
	sampleRate         uint32
	writeRaw           bool
	vocoderForProtocol map[string]string

	mu        sync.Mutex
	sessions  map[string]*recordingSession // by device serial
	sub       *events.Subscription
	runDone   chan struct{}
	closeOnce sync.Once

	// recordDisabled gates new sessions at runtime. Toggled from
	// the API by operators who want to stop laying down WAVs
	// without restarting the daemon. In-flight sessions are NOT
	// truncated on disable — they finish naturally on CallEnd so
	// the head of a call isn't lost when the operator flips the
	// switch mid-conversation.
	recordDisabled atomic.Bool
}

// RecorderOptions configure a new Recorder.
type RecorderOptions struct {
	Bus        *events.Bus
	Log        *slog.Logger
	OutDir     string
	SampleRate uint32 // 8000 typical
	WriteRaw   bool   // emit a .raw sidecar alongside each .wav

	// VocoderForProtocol maps a Grant.Protocol value to a vocoder
	// registry name used to decode raw frames into PCM that's
	// written to the call's WAV. nil means "use the package
	// defaults" (DefaultVocoderForProtocol). Pass an explicit empty
	// (non-nil) map to disable auto-decode entirely; the .raw
	// sidecar then becomes the only path for digital voice.
	//
	// Protocols not in the map produce no decoded audio — typically
	// analog protocols (motorola, edacs, ltr, mpt1327) where the
	// composer's FM chain feeds WritePCM directly, and ProVoice
	// where no in-binary decoder is available.
	VocoderForProtocol map[string]string
}

// DefaultVocoderForProtocol returns the Protocol → vocoder-name
// mapping NewRecorder uses when RecorderOptions.VocoderForProtocol
// is nil. The keys match the strings the radio decoders set on
// Grant.Protocol; the values match factory names registered into
// voice.DefaultRegistry by the imbe / ambe2 package init()s.
//
// Callers wanting to override one entry should start with a copy of
// this map (DefaultVocoderForProtocol() returns a fresh map per
// call) and mutate from there — RecorderOptions.VocoderForProtocol
// is taken as-is, no merging.
func DefaultVocoderForProtocol() map[string]string {
	return map[string]string{
		"p25":        "imbe",  // P25 Phase 1 — IMBE 4400
		"p25-phase2": "ambe2", // P25 Phase 2 — AMBE+2 2400
		"dmr-tier2":  "ambe2", // DMR Tier II conventional
		"dmr-tier3":  "ambe2", // DMR Tier III trunked
		"nxdn":       "ambe2",
		"dpmr":       "ambe2", // dPMR Mode 3 (digital)
		"tetra":      "ambe2", // TETRA voice
	}
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
	vocoderMap := opts.VocoderForProtocol
	if vocoderMap == nil {
		vocoderMap = DefaultVocoderForProtocol()
	}
	r := &Recorder{
		bus:                opts.Bus,
		log:                opts.Log,
		outDir:             opts.OutDir,
		sampleRate:         opts.SampleRate,
		writeRaw:           opts.WriteRaw,
		vocoderForProtocol: vocoderMap,
		sessions:           make(map[string]*recordingSession),
		runDone:            make(chan struct{}),
	}
	r.sub = opts.Bus.Subscribe()
	return r, nil
}

// SetRecordingEnabled toggles the recorder's runtime "create new
// sessions" gate. When enabled is false, subsequent CallStart events
// do NOT open .wav / .raw files; in-flight sessions are left alone
// so the head of a mid-call disable isn't lost on disk. Default
// (after NewRecorder) is enabled = true.
func (r *Recorder) SetRecordingEnabled(enabled bool) {
	r.recordDisabled.Store(!enabled)
}

// RecordingEnabled reports the current gate state.
func (r *Recorder) RecordingEnabled() bool {
	return !r.recordDisabled.Load()
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

// WriteRawFrame consumes a raw vocoder frame for the named device
// serial. Two outputs are produced when applicable:
//
//   - The .raw sidecar (when one was opened — see handleStart). The
//     frame bytes are appended verbatim so external decoders can
//     consume the file with no surrounding metadata.
//   - The .wav (when a vocoder was instantiated for the call's
//     Grant.Protocol). The frame is decoded into PCM and the
//     samples are appended to the WAV. A per-frame Decode error is
//     logged and the frame is dropped from PCM but still written to
//     the sidecar.
//
// Frames for a session without either output (no sidecar, no
// vocoder) are dropped silently.
func (r *Recorder) WriteRawFrame(deviceSerial string, frame []byte) error {
	r.mu.Lock()
	s, ok := r.sessions[deviceSerial]
	r.mu.Unlock()
	if !ok {
		return nil
	}
	if s.raw != nil {
		if _, err := s.raw.Write(frame); err != nil {
			return err
		}
	}
	if s.vocoder != nil {
		samples, err := s.vocoder.Decode(frame)
		if err != nil {
			r.log.Warn("recorder: vocoder decode failed; dropping frame from PCM",
				"device", deviceSerial, "vocoder", s.vocoder.Name(), "err", err)
			return nil
		}
		if err := s.wav.WriteSamples(samples); err != nil {
			return err
		}
	}
	return nil
}

func (r *Recorder) handleStart(cs trunking.CallStart) {
	if r.recordDisabled.Load() {
		// Operator has flipped off recording at runtime. Drop the
		// CallStart silently so no files land on disk for this call.
		// In-flight sessions started before the disable continue to
		// completion via handleEnd.
		return
	}
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
	// Instantiate a vocoder for the protocol if one is mapped.
	// Construction failure (unknown registry name) logs a warning and
	// proceeds with no auto-decode — the sidecar (if any) is still the
	// safety net.
	if name, ok := r.vocoderForProtocol[cs.Grant.Protocol]; ok && name != "" {
		v, err := DefaultRegistry.New(name)
		if err != nil {
			r.log.Warn("recorder: cannot instantiate vocoder; auto-decode disabled for this call",
				"device", cs.DeviceSerial, "protocol", cs.Grant.Protocol,
				"vocoder", name, "err", err)
		} else {
			s.vocoder = v
			s.vocoderName = name
		}
	}
	r.sessions[cs.DeviceSerial] = s
	r.log.Info("recorder: call started",
		"device", cs.DeviceSerial, "wav", wavPath,
		"tg", cs.Grant.GroupID, "provoice", cs.Grant.ProVoice,
		"vocoder", s.vocoderName)
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
	wav         *WavWriter
	wavPath     string
	raw         *os.File
	rawPath     string
	vocoder     Vocoder
	vocoderName string
	startedAt   time.Time
}

func (s *recordingSession) close() error {
	var firstErr error
	if s.vocoder != nil {
		if err := s.vocoder.Close(); err != nil {
			firstErr = err
		}
	}
	if err := s.wav.Close(); err != nil && firstErr == nil {
		firstErr = err
	}
	if s.raw != nil {
		if err := s.raw.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
