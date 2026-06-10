package voice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
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
//	<OutDir>/<system>/<talkgroup-or-decimal-id>/<UTC-RFC3339>_freq<Hz>_src<src>.wav
//	<OutDir>/<system>/<talkgroup-or-decimal-id>/<UTC-RFC3339>_freq<Hz>_src<src>.raw
//
// (The _freq<Hz> tag carries the RF voice-channel frequency; it is
// omitted when the grant frequency is unknown. A _ts<slot> tag is
// appended for slotted protocols. When per-transmission recording is
// enabled, each over rolls to a new file with a fresh timestamp.)
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
	skipEncrypted      bool
	vocoderForProtocol map[string]string

	mu        sync.Mutex
	sessions  map[string]*recordingSession // by device serial
	sub       *events.Subscription
	runDone   chan struct{}
	closeOnce sync.Once

	// decodedTap, when set, receives the PCM decoded from digital
	// vocoder frames in WriteRawFrame — the live consumers (web
	// stream, host player, tone-out) that only implement WritePCM and
	// so never see raw frames. Wired once at daemon construction via
	// SetDecodedPCMSink before Run starts, so the hot path reads it
	// without locking. nil = no live fan-out (analog-only callers,
	// tests).
	decodedTap DecodedPCMSink

	// recordDisabled gates new sessions at runtime. Toggled from
	// the API by operators who want to stop laying down WAVs
	// without restarting the daemon. In-flight sessions are NOT
	// truncated on disable — they finish naturally on CallEnd so
	// the head of a call isn't lost when the operator flips the
	// switch mid-conversation.
	recordDisabled atomic.Bool
}

// DecodedPCMSink receives PCM the recorder decodes from digital
// vocoder frames, so live consumers (web stream, host player,
// tone-out) hear digital calls — not just analog ones. The composer's
// digital chains emit only raw vocoder frames, which fan out solely to
// the recorder (the lone decoder); without this tap that decoded audio
// never reaches the WritePCM-only live sinks. Mirrors composer.PCMSink's
// WritePCM but is declared here to avoid a voice → composer import cycle.
type DecodedPCMSink interface {
	WritePCM(deviceSerial string, samples []int16) error
}

// SetDecodedPCMSink wires the live-audio tap that receives PCM decoded
// from digital vocoder frames. Call once during daemon construction
// before Run/any calls start; it is not safe to change concurrently
// with WriteRawFrame.
func (r *Recorder) SetDecodedPCMSink(s DecodedPCMSink) {
	r.decodedTap = s
}

// RecorderOptions configure a new Recorder.
type RecorderOptions struct {
	Bus        *events.Bus
	Log        *slog.Logger
	OutDir     string
	SampleRate uint32 // 8000 typical
	WriteRaw   bool   // emit a .raw sidecar alongside each .wav

	// SkipEncrypted, when true, makes the recorder refuse to write files
	// for calls flagged encrypted. A grant that already signals encryption
	// never opens a session; a call whose encryption is only discovered
	// mid-stream (P25 Phase 1 LDU2 Encryption Sync, or a Phase 2 compressed
	// grant resolved in-call) has its in-progress WAV/raw files closed and
	// deleted, and no CallComplete is published so downstream upload feeds
	// never see the partial.
	SkipEncrypted bool

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
	// DMR maps to "ambe2-dmr" — the AMBE+2 3600x2450 variant DMR uses,
	// distinct from the 3600x2400 "ambe2" used by P25 Phase 2 / NXDN.
	// The composer's DMR voice chain emits FEC-decoded 49-bit frames,
	// which that decoder renders to PCM.
	return map[string]string{
		"p25":        "imbe",      // P25 Phase 1 — IMBE 4400
		"p25-phase2": "ambe2",     // P25 Phase 2 — AMBE+2 3600x2400
		"dmr-tier1":  "ambe2-dmr", // DMR Tier I direct-mode — AMBE+2 3600x2450
		"dmr-tier2":  "ambe2-dmr", // DMR Tier II — AMBE+2 3600x2450
		"dmr-tier3":  "ambe2-dmr", // DMR Tier III — AMBE+2 3600x2450
		"nxdn":       "ambe2",
		"dpmr":       "ambe2", // dPMR Mode 3 (digital)
		"tetra":      "ambe2", // TETRA voice
	}
}

// dmrVoiceProtocol reports whether protocol is a DMR voice protocol.
// DMR voice calls always get a .raw sidecar (alongside the decoded
// WAV) so the on-air AMBE frames remain available for out-of-band
// tools even though the in-process vocoder now renders them.
func dmrVoiceProtocol(protocol string) bool {
	return protocol == "dmr-tier1" || protocol == "dmr-tier2" || protocol == "dmr-tier3"
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
		skipEncrypted:      opts.SkipEncrypted,
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
			case events.KindCallSegment:
				if seg, ok := ev.Payload.(trunking.CallSegment); ok {
					r.handleSegment(seg)
				}
			case events.KindCallEncryption:
				// In-call Encryption Sync recovered (P25 Phase 1 LDU2).
				// AlgorithmID 0x80 is CLEAR; anything else is encrypted.
				if ce, ok := ev.Payload.(trunking.CallEncryption); ok {
					r.handleEncryptionUpdate(ce.DeviceSerial, ce.AlgorithmID != algorithmClear)
				}
			case events.KindCallSourceUpdate:
				// In-call source/encryption resolved on the traffic channel
				// (e.g. a P25 Phase 2 compressed grant). Carries an explicit
				// encrypted flag.
				if su, ok := ev.Payload.(trunking.CallSourceUpdate); ok {
					r.handleEncryptionUpdate(su.DeviceSerial, su.Encrypted)
				}
			}
		}
	}
}

// WritePCM appends 16-bit PCM samples for the named device serial. If no
// session is open for that device the samples are dropped (the demod
// pipeline can race ahead of the CallStart event).
func (r *Recorder) WritePCM(deviceSerial string, samples []int16) error {
	s := r.sessionForWrite(deviceSerial)
	if s == nil {
		return nil
	}
	return s.wav.WriteSamples(samples)
}

// sessionForWrite returns the live session for serial, lazily opening
// the next per-transmission segment file when the session is dormant
// (parked after a KindCallSegment roll). Returns nil when no session is
// open for the device.
func (r *Recorder) sessionForWrite(serial string) *recordingSession {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[serial]
	if !ok {
		return nil
	}
	if s.wav == nil {
		ns := r.buildSession(s.cs, time.Now().UTC())
		if ns == nil {
			return nil
		}
		r.sessions[serial] = ns
		s = ns
	}
	return s
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
	return r.writeRawFrame(deviceSerial, frame, 0, false)
}

// WriteRawFrameWithErrors is WriteRawFrame plus the channel FEC
// corrected-bit count for the frame. When the session's vocoder implements
// voice.ErrorAware (the pure-Go IMBE decoder), the count is handed to it
// before Decode so adaptive smoothing can estimate the channel error rate.
// Callers without an error count use WriteRawFrame.
func (r *Recorder) WriteRawFrameWithErrors(deviceSerial string, frame []byte, correctedBits int) error {
	return r.writeRawFrame(deviceSerial, frame, correctedBits, true)
}

func (r *Recorder) writeRawFrame(deviceSerial string, frame []byte, correctedBits int, haveErrs bool) error {
	s := r.sessionForWrite(deviceSerial)
	if s == nil {
		return nil
	}
	if s.raw != nil {
		if _, err := s.raw.Write(frame); err != nil {
			return err
		}
	}
	if s.vocoder != nil {
		if haveErrs {
			if ea, ok := s.vocoder.(ErrorAware); ok {
				ea.SetFrameErrors(correctedBits)
			}
		}
		samples, err := s.vocoder.Decode(frame)
		if err != nil {
			r.log.Warn("recorder: vocoder decode failed; dropping frame from PCM",
				"device", deviceSerial, "vocoder", s.vocoder.Name(), "err", err)
			return nil
		}
		if err := s.wav.WriteSamples(samples); err != nil {
			return err
		}
		// Fan the freshly-decoded PCM to the live consumers (web
		// stream, host player, tone-out). For digital protocols this
		// is the only point where PCM exists — the composer never
		// calls WritePCM for them — so without this the live audio
		// path is silent while recordings play fine (issue #598). Runs
		// outside r.mu (sessionForWrite released it), so the tap is
		// free to take its own locks.
		if r.decodedTap != nil {
			_ = r.decodedTap.WritePCM(deviceSerial, samples)
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
	if cs.Talkgroup != nil && !cs.Talkgroup.Record {
		// Talkgroup is flagged record=false — follow and play the
		// call live, but write no WAV/raw files for it.
		return
	}
	if r.skipEncrypted && cs.Grant.Encrypted {
		// Operator opted out of recording encrypted calls and the grant
		// already signals encryption — never open a file. Calls whose
		// encryption only surfaces mid-stream are handled by
		// handleEncryptionUpdate.
		r.log.Debug("recorder: skipping encrypted call",
			"device", cs.DeviceSerial, "tg", cs.Grant.GroupID,
			"alg", cs.Grant.AlgorithmID, "key", cs.Grant.KeyID)
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
	s := r.buildSession(cs, cs.StartedAt)
	if s == nil {
		return
	}
	r.sessions[cs.DeviceSerial] = s
	r.log.Info("recorder: call started",
		"device", cs.DeviceSerial, "wav", s.wavPath,
		"tg", cs.Grant.GroupID, "provoice", cs.Grant.ProVoice,
		"vocoder", s.vocoderName)
}

// algorithmClear is the encryption Algorithm ID a clear (unencrypted)
// call advertises; anything else means the call is encrypted. Mirrors
// p25.AlgorithmClear, kept local to avoid a radio-package import.
const algorithmClear uint8 = 0x80

// handleEncryptionUpdate aborts an in-flight recording when SkipEncrypted
// is set and a call is discovered to be encrypted mid-stream — e.g. a P25
// Phase 1 LDU2 Encryption Sync or a Phase 2 compressed grant whose
// encryption flag only resolves on the traffic channel. The open WAV/raw
// files are closed and removed and the session dropped without publishing
// a CallComplete, so the partial never reaches the upload feeds. No-op
// when SkipEncrypted is off, the call is clear, or no session is open.
func (r *Recorder) handleEncryptionUpdate(deviceSerial string, encrypted bool) {
	if !r.skipEncrypted || !encrypted {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[deviceSerial]
	if !ok {
		return
	}
	delete(r.sessions, deviceSerial)
	// A dormant post-segment session (parked between overs) has no open
	// files; only close when a WAV is actually open, mirroring handleEnd.
	if s.wav == nil {
		return
	}
	if err := s.close(); err != nil {
		r.log.Warn("recorder: closing aborted encrypted session",
			"device", deviceSerial, "err", err)
	}
	for _, p := range []string{s.wavPath, s.rawPath} {
		if p == "" {
			continue
		}
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			r.log.Warn("recorder: removing aborted encrypted file",
				"path", p, "err", err)
		}
	}
	r.log.Info("recorder: aborted mid-call encrypted recording",
		"device", deviceSerial, "wav", s.wavPath)
}

// buildSession opens the WAV (+ optional .raw sidecar + vocoder) for a
// call, naming the files from cs but timestamped at startedAt (which
// differs from cs.StartedAt for a per-transmission segment roll).
// Returns nil on a fatal open error (already logged). The caller holds
// r.mu and registers the returned session.
func (r *Recorder) buildSession(cs trunking.CallStart, startedAt time.Time) *recordingSession {
	dir := r.directoryFor(cs)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		r.log.Error("recorder: mkdir", "dir", dir, "err", err)
		return nil
	}
	nameCS := cs
	nameCS.StartedAt = startedAt
	base := r.basenameFor(nameCS)
	s := &recordingSession{startedAt: startedAt, cs: cs}
	// Instantiate a vocoder for the protocol if one is mapped. This must
	// happen before the WAV is opened so the header rate can track the
	// vocoder's native output rate (below). Construction failure (unknown
	// registry name) logs a warning and proceeds with no auto-decode —
	// the sidecar (if any) is still the safety net.
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
	// Pick the WAV header rate. Vocoder output is always 8 kHz (see the
	// Vocoder interface contract) and WriteRawFrame appends those samples
	// to the WAV without resampling, so a decoded call's header MUST be
	// 8 kHz regardless of recordings.sample_rate — otherwise clean audio
	// plays back at the wrong speed (garbled). recordings.sample_rate
	// only applies to analog/NBFM calls fed via WritePCM. Issue #356
	// follow-up.
	s.sampleRate = r.sampleRate
	if s.vocoder != nil {
		s.sampleRate = pcmHzDefault
		if r.sampleRate != pcmHzDefault {
			r.log.Warn("recorder: forcing WAV rate to vocoder-native 8000; recordings.sample_rate applies to analog/NBFM only",
				"device", cs.DeviceSerial, "protocol", cs.Grant.Protocol,
				"recordings_sample_rate", r.sampleRate)
		}
	}
	wavPath := filepath.Join(dir, base+".wav")
	wav, err := NewWavFile(wavPath, s.sampleRate)
	if err != nil {
		r.log.Error("recorder: open wav", "path", wavPath, "err", err)
		if s.vocoder != nil {
			_ = s.vocoder.Close()
		}
		return nil
	}
	s.wav = wav
	s.wavPath = wavPath
	// ProVoice and DMR voice grants always get a sidecar — neither has
	// an in-process vocoder, so the .raw file is the only capture of
	// the call.
	if r.writeRaw || cs.Grant.ProVoice || dmrVoiceProtocol(cs.Grant.Protocol) {
		rawPath := filepath.Join(dir, base+".raw")
		raw, err := os.Create(rawPath)
		if err != nil {
			r.log.Error("recorder: open raw", "path", rawPath, "err", err)
		} else {
			s.raw = raw
			s.rawPath = rawPath
		}
	}
	return s
}

// handleSegment finalizes the current recording at a per-transmission
// boundary and parks the session as dormant so the next over opens a
// fresh file. Published by the composer only in "transmission" grouping.
func (r *Recorder) handleSegment(seg trunking.CallSegment) {
	r.mu.Lock()
	s, ok := r.sessions[seg.DeviceSerial]
	if !ok || s.wav == nil {
		r.mu.Unlock()
		return // no active file to roll (already dormant or unknown)
	}
	cc := r.finalizeLocked(s, seg.DeviceSerial, seg.At, trunking.EndReasonNormal)
	// Park a dormant session: keeps the call's identity so the next write
	// opens the next segment file, without creating an empty trailing
	// file if no further audio arrives before the call ends.
	r.sessions[seg.DeviceSerial] = &recordingSession{cs: s.cs}
	r.mu.Unlock()
	if cc != nil {
		r.bus.Publish(events.Event{Kind: events.KindCallComplete, Payload: *cc})
	}
}

// finalizeLocked closes a session's files and returns a CallComplete to
// publish when audio was written; an empty file (no PCM) is removed and
// nil returned. Caller holds r.mu and publishes after releasing it.
func (r *Recorder) finalizeLocked(s *recordingSession, serial string, endedAt time.Time, reason trunking.EndReason) *trunking.CallComplete {
	dataBytes := s.wav.DataBytes()
	r.logVoiceStats(serial, s)
	if err := s.close(); err != nil {
		r.log.Error("recorder: close session", "err", err)
	}
	// No PCM decoded into the WAV — nothing to stream. The files are left
	// in place: a digital call (ProVoice / DMR / pre-vocoder) keeps its
	// .raw sidecar as the only capture even when the WAV is empty.
	// Per-transmission segment rolls never reach here empty because the
	// next file is opened lazily on the first write.
	if dataBytes == 0 {
		return nil
	}
	return &trunking.CallComplete{
		Grant:        s.cs.Grant,
		Talkgroup:    s.cs.Talkgroup,
		DeviceSerial: serial,
		StartedAt:    s.startedAt,
		EndedAt:      endedAt,
		Reason:       reason,
		AudioPath:    s.wavPath,
		SampleRate:   s.sampleRate,
	}
}

// voiceClipWarnPct is the output-clipping rate above which the per-call
// voice summary escalates to WARN: a non-trivial fraction of samples
// hitting the limiter means the vocoder gain is too hot (the signature of
// the over-driven AGC that flattens speech into a robotic, clipped tone).
const voiceClipWarnPct = 0.5

// logVoiceStats emits the per-call audio-quality summary for a session
// whose vocoder tracks VoiceStats (currently the pure-Go IMBE decoder).
// It is the audio-layer complement to the composer's FEC-layer "decode
// quality" line: pitch, harmonic count, AGC gain, and amplitude health
// (peak / clip% / crest). Logged at DEBUG; escalated to WARN when the
// output is clipping. A vocoder that doesn't implement StatProvider, or a
// call with no frames, produces nothing.
func (r *Recorder) logVoiceStats(serial string, s *recordingSession) {
	sp, ok := s.vocoder.(StatProvider)
	if !ok {
		return
	}
	vs, have := sp.VoiceStats()
	if !have || vs.Frames == 0 {
		return
	}
	args := []any{
		"device", serial, "wav", s.wavPath, "vocoder", s.vocoderName,
		"frames", vs.Frames, "voiced", vs.Voiced, "unvoiced", vs.Unvoiced,
		"silent", vs.Silent, "idle_muted", vs.IdleMuted,
		"bad", vs.Bad, "repeated", vs.Repeated,
		"mean_f0_hz", round1(vs.MeanF0Hz),
		"f0_range_hz", fmt.Sprintf("%.0f-%.0f", vs.MinF0Hz, vs.MaxF0Hz),
		"mean_l", round1(vs.MeanL),
		"voiced_frac", round2(vs.MeanVoicedFrac),
		"mean_gain", round2(vs.MeanAGCGain),
		"gain_range", fmt.Sprintf("%.1f-%.1f", vs.MinAGCGain, vs.MaxAGCGain),
		"peak_preclip", round1(vs.MaxPreClipPeak),
		"rms", round1(vs.OutputRMS),
		"crest", round2(vs.CrestFactor),
		"clip_pct", round2(vs.ClipPct()),
	}
	if vs.ClipPct() > voiceClipWarnPct {
		r.log.Warn("recorder: voice audio quality — output clipping (vocoder gain too hot)", args...)
		return
	}
	r.log.Debug("recorder: voice audio quality", args...)
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
func round2(v float64) float64 { return math.Round(v*100) / 100 }

func (r *Recorder) handleEnd(ce trunking.CallEnd) {
	r.mu.Lock()
	s, ok := r.sessions[ce.DeviceSerial]
	if ok {
		delete(r.sessions, ce.DeviceSerial)
	}
	if !ok || s.wav == nil {
		// No session, or a dormant post-segment session whose next over
		// never arrived — nothing to finalize.
		r.mu.Unlock()
		return
	}
	wavPath := s.wavPath
	// Announce the finished WAV so the outbound-streaming subsystem can
	// upload it. Skip (and delete) calls that captured no PCM.
	cc := r.finalizeLocked(s, ce.DeviceSerial, ce.EndedAt, ce.Reason)
	r.mu.Unlock()
	r.log.Info("recorder: call ended",
		"device", ce.DeviceSerial,
		"wav", wavPath,
		"duration", ce.Duration().Round(time.Millisecond),
		"reason", ce.Reason)
	if cc != nil {
		r.bus.Publish(events.Event{Kind: events.KindCallComplete, Payload: *cc})
	}
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
	// Tag the RF voice-channel frequency (Hz) into the name. Voice
	// frequencies are shared across talkgroups on a trunked system, so
	// having the frequency on each file makes it easy to tell which
	// physical channel a recording came from. Omitted when unknown (0).
	base := stamp
	if cs.Grant.FrequencyHz != 0 {
		base = fmt.Sprintf("%s_freq%d", base, cs.Grant.FrequencyHz)
	}
	base = fmt.Sprintf("%s_src%d", base, cs.Grant.SourceID)
	// A DMR Tier III carrier runs two concurrent calls (TS1 + TS2); when
	// they share a talkgroup (same directory) and a start second, the
	// stamp+src basename would collide. Tag the slot so each slot's WAV
	// is distinct on disk and self-labelling. Omitted for non-slotted
	// protocols (Timeslot 0).
	if cs.Grant.Timeslot != 0 {
		base = fmt.Sprintf("%s_ts%d", base, cs.Grant.Timeslot)
	}
	return base
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
	sampleRate  uint32
	startedAt   time.Time
	// cs is the originating CallStart, retained so a per-transmission
	// segment roll (KindCallSegment) can open the next file with the same
	// grant/talkgroup under a new timestamp. A session with wav == nil is
	// "dormant" — finalized at a segment boundary and waiting to open its
	// next file lazily on the first write, so an over with no following
	// audio never leaves an empty trailing file.
	cs trunking.CallStart
}

func (s *recordingSession) close() error {
	var firstErr error
	if s.vocoder != nil {
		if err := s.vocoder.Close(); err != nil {
			firstErr = err
		}
	}
	// A dormant post-segment session has wav == nil (see the recordingSession
	// doc comment) — guard so finalizing it never dereferences a nil writer.
	if s.wav != nil {
		if err := s.wav.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if s.raw != nil {
		if err := s.raw.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
