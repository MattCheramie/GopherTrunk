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
	"github.com/MattCheramie/GopherTrunk/internal/voice/mbe"
)

// VoiceEnhanceable is the optional interface a software vocoder
// implements to accept the opt-in output enhancement chain
// (recordings.enhance). The recorder type-asserts the vocoder it builds
// for each call and, when the assertion succeeds, installs the current
// enhancement config so the chain applies to BOTH the recorded WAV and
// the live decoded-PCM fan-out (they share this one decode). Vocoders
// that don't implement it (e.g. a future hardware backend) simply decode
// faithfully. The imbe and ambe2 decoders implement it.
type VoiceEnhanceable interface {
	SetVoiceEnhancer(cfg mbe.EnhancerConfig)
}

// StartupSquelchable is the optional interface a software vocoder implements
// to accept the call-startup acquisition squelch — muting the receiver's
// acquisition "startup scratch" until real speech is confirmed. The recorder
// enables it on every decoded call so recordings and the live fan-out both
// open cleanly. Vocoders that don't implement it decode unchanged. The imbe
// (P25 Phase 1) decoder implements it.
type StartupSquelchable interface {
	EnableStartupSquelch()
}

// Recorder writes per-call audio + raw-frame files. It subscribes to
// events.KindCallStart and events.KindCallEnd from the trunking engine,
// opens a WAV (and optional raw-frame sidecar) for each new call, and
// closes them on call end. The demod-pipeline composer pushes PCM
// samples in via WritePCM (analog protocols) and raw vocoder frames
// in via WriteRawFrame (digital protocols), keyed by device serial.
//
// Default layout under OutDir:
//
//	<OutDir>/<system>/<talkgroup-or-decimal-id>/<YYYYMMDD>_<HHMMSS>_<tg>.wav
//	<OutDir>/<system>/individual/<dest-id>/<YYYYMMDD>_<HHMMSS>_<dest>.wav
//
// The basename and the directory tree are both operator-configurable via
// {token} templates (RecorderOptions.FilenameTemplate / PathTemplate; see
// expandNameTemplate). The default basename drops the timezone offset, source,
// and timeslot the earlier scheme carried — all of which remain in the .json
// sidecar — so a template need not repeat them; two calls that would share a
// basename (concurrent DMR TS1/TS2 on one talkgroup and second) get a -2 suffix
// (ensureUniqueBase). The .raw and .json sidecars share the .wav basename. When
// per-transmission recording is enabled, each over rolls to a new file with a
// fresh timestamp.
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
	writeCallJSON      bool
	normalize          NormalizeConfig
	vocoderForProtocol map[string]string

	// callNum is a monotonic per-recorder call counter stamped into each
	// .json sidecar's call_num (the trunk-recorder sequential call number).
	callNum atomic.Uint64

	// decodeOnly runs the recorder as a live decoder that writes NO files:
	// every call still builds its per-protocol vocoder and fans decoded PCM
	// to decodedTap (the web/gRPC stream), but no WAV or .raw is opened. Set
	// when OutDir is empty, so live browser audio works even when the operator
	// has not configured a recordings directory. When false the recorder
	// behaves exactly as before (decode + persist).
	decodeOnly bool

	// enhance is the current voice enhancement config, read when a new
	// call's vocoder is built (buildSession). Guarded by enhanceMu so the
	// settings endpoint can swap it at runtime via SetVoiceEnhance — the
	// change then takes effect on the NEXT call (each call constructs a
	// fresh vocoder), which is the live-edit granularity for this feature.
	enhanceMu sync.RWMutex
	enhance   mbe.EnhancerConfig

	mu       sync.Mutex
	sessions map[string]*recordingSession // by device serial
	// callTalkers accumulates every distinct talker heard during a call, keyed by
	// device serial and spanning the call's whole life — unlike a recordingSession's
	// srcs, which resets on each per-transmission segment roll. In transmission
	// grouping each rolled WAV legitimately holds one talker, but the trunk-recorder
	// .json is call metadata, so every transmission's sidecar srcList should list all
	// the call's talkers (the operator's "second srcID is missed"). Seeded on a new
	// call (handleStart), appended on a talker change (backfillSessionGrant), read at
	// finalize, and cleared when the call ends. Guarded by mu.
	callTalkers map[string][]srcEvent

	// drainCoordinated is set (via EnableDrainCoordination) when a composer that
	// signals NotifyDrainComplete is wired to this recorder. When true, handleEnd
	// does NOT finalize a call immediately on KindCallEnd; it defers until the
	// composer signals its voice chain has drained (or a safety timeout fires),
	// so the transmission-tail frames the chain writes during teardown land in
	// the recording instead of on an already-deleted session. When false the
	// recorder finalizes on CallEnd exactly as before (standalone use, tests).
	drainCoordinated bool
	// pendingFinalize holds calls awaiting drain coordination, keyed by device
	// serial. A call finalizes once BOTH its KindCallEnd (handleEnd) and the
	// composer's NotifyDrainComplete have arrived — in either order — or the
	// safety timer fires. Guarded by mu.
	pendingFinalize map[string]*pendingFinalize

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

	// rawTap, when set, receives the verbatim raw vocoder frames the
	// recorder also writes to its .raw sidecar, so the gRPC audio
	// publisher's include_raw subscribers can stream un-decoded IMBE /
	// AMBE bytes. Wired once at daemon construction via SetRawFrameSink
	// before Run starts, so the hot path reads it without locking. nil =
	// no raw fan-out (analog-only callers, tests).
	rawTap RawFrameSink

	// recordDisabled gates new sessions at runtime. Toggled from
	// the API by operators who want to stop laying down WAVs
	// without restarting the daemon. In-flight sessions are NOT
	// truncated on disable — they finish naturally on CallEnd so
	// the head of a call isn't lost when the operator flips the
	// switch mid-conversation.
	recordDisabled atomic.Bool

	// displayLoc is the timezone the recording-filename timestamp renders
	// in, matching the location the rest of the app uses for human-facing
	// timestamps (display.timezone). Never nil after NewRecorder; defaults
	// to time.UTC when the caller leaves it unset (the prior behaviour).
	displayLoc *time.Location

	// filenameTmpl / pathTmpl are the operator's optional {token} templates for
	// the recording basename and directory tree (recordings.filename_template /
	// path_template). Empty selects the built-in defaults. See expandNameTemplate.
	filenameTmpl string
	pathTmpl     string

	// dedup configures cross-site duplicate-recording suppression; dedupSeen is
	// the recently-recorded (talkgroup, source) → (system, time) index it keys
	// on, guarded by dedupMu. Only consulted for file-writing recorders (a
	// decode-only recorder writes nothing, so there is no duplicate to suppress).
	dedup     DedupConfig
	dedupMu   sync.Mutex
	dedupSeen map[dedupKey]dedupEntry
}

// DedupConfig configures cross-site duplicate-recording suppression. When
// Enabled, a call whose (talkgroup, source) was already recorded from a
// DIFFERENT system within Window is skipped — one copy of a call heard on
// several networked/simulcast sites instead of one per site.
type DedupConfig struct {
	Enabled bool
	Window  time.Duration
}

type dedupKey struct{ group, source uint32 }

type dedupEntry struct {
	system string
	at     time.Time
}

// dedupMaxKeys bounds the recently-recorded index so a long-running busy
// multi-system deployment cannot grow it without bound; stale keys are pruned
// once it is exceeded.
const dedupMaxKeys = 4096

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

// DecodedPCMCallSink is the call-aware extension of DecodedPCMSink: it carries
// the CallID the decoded PCM belongs to so the live fan-out can fence a stale
// frame from a previous call on a reused voice-tap serial — the same
// cross-call audio-bleed guard sessionForWrite applies to the WAV. Wideband
// voice taps reuse a fixed pool of serials, so without this the live stream
// labels (and leaks) an old call's draining audio with the next call's
// talkgroup. A sink implementing this is preferred over plain WritePCM in
// WriteRawFrame's fan-out; sinks that don't implement it still get WritePCM.
type DecodedPCMCallSink interface {
	WritePCMForCall(deviceSerial string, callID uint64, samples []int16) error
}

// RawFrameSink receives the verbatim, un-decoded vocoder frames (IMBE /
// AMBE+2) the recorder also lays down as its .raw sidecar, so a live consumer
// — the gRPC audio publisher's include_raw subscribers — can stream the raw
// bytes without the recorder remaining the only place they exist. It carries
// the vocoder name that produced the session so the wire frame can label its
// codec, and unlike the decoded tap it fires even for protocols with no
// in-process decoder (ProVoice, encrypted): the raw bytes exist even when
// decoded PCM does not.
type RawFrameSink interface {
	WriteRawFrame(deviceSerial, vocoder string, frame []byte) error
}

// RawFrameCallSink is the call-aware extension of RawFrameSink: it carries the
// CallID the raw frame belongs to so the live fan-out applies the same
// cross-call fence the decoded tap does when a voice-tap serial is reused. A
// sink implementing it is preferred over plain WriteRawFrame in the raw-tap
// fan-out; sinks that don't still get WriteRawFrame.
type RawFrameCallSink interface {
	WriteRawFrameForCall(deviceSerial string, callID uint64, vocoder string, frame []byte) error
}

// SetDecodedPCMSink wires the live-audio tap that receives PCM decoded
// from digital vocoder frames. Call once during daemon construction
// before Run/any calls start; it is not safe to change concurrently
// with WriteRawFrame.
func (r *Recorder) SetDecodedPCMSink(s DecodedPCMSink) {
	r.decodedTap = s
}

// SetRawFrameSink wires the live raw-frame tap that receives the verbatim
// un-decoded vocoder frames (the gRPC audio publisher). Call once during
// daemon construction before Run/any calls start; it is not safe to change
// concurrently with WriteRawFrame.
func (r *Recorder) SetRawFrameSink(s RawFrameSink) {
	r.rawTap = s
}

// RecorderOptions configure a new Recorder.
type RecorderOptions struct {
	Bus        *events.Bus
	Log        *slog.Logger
	OutDir     string
	SampleRate uint32 // 8000 typical
	WriteRaw   bool   // emit a .raw sidecar alongside each .wav
	// WriteCallJSON emits a trunk-recorder-compatible <basename>.json metadata
	// sidecar next to each recording (per WAV / segment).
	WriteCallJSON bool

	// SkipEncrypted, when true, makes the recorder refuse to write files
	// for calls flagged encrypted. A grant that already signals encryption
	// never opens a session; a call whose encryption is only discovered
	// mid-stream (P25 Phase 1 LDU2 Encryption Sync, or a Phase 2 compressed
	// grant resolved in-call) has its in-progress WAV/raw files closed and
	// deleted, and no CallComplete is published so downstream upload feeds
	// never see the partial.
	SkipEncrypted bool

	// Normalize configures optional per-call EBU R128 / BS.1770 loudness
	// normalization. When Enabled, a finished WAV is measured and rewritten
	// in place to TargetLUFS (true-peak limited) before CallComplete is
	// published, so every downstream consumer (web playback, MP3 encode,
	// uploads) reads the normalized audio. Off by default.
	Normalize NormalizeConfig

	// Enhance configures the optional, opt-in "sound-good" voice
	// enhancement chain (band-limit + warmth shelf + louder AGC +
	// optional compression) applied to decoded digital voice. When
	// Enabled it shapes both the recorded WAV and the live decoded-PCM
	// fan-out, since both come from the recorder's single per-call
	// vocoder. Disabled by default (faithful, byte-identical output).
	Enhance mbe.EnhancerConfig

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

	// DisplayLoc is the timezone the recording-filename timestamp renders
	// in (display.timezone, via config.DisplayConfig.Location()). nil falls
	// back to time.UTC (the prior behaviour). Threaded so WAV filenames
	// match the local wall-clock the rest of the UI/logs already show,
	// instead of always UTC.
	DisplayLoc *time.Location

	// Dedup opts into cross-site duplicate-recording suppression (off by
	// default). See DedupConfig / Recorder.dedupSuppress.
	Dedup DedupConfig

	// FilenameTemplate customises the recording basename (no extension) via
	// {token} substitution; empty selects the default "{date}_{time}_{tg}".
	// PathTemplate customises the per-call directory tree under OutDir; empty
	// keeps the <system>/<talkgroup>/ (+ individual/) layout. See
	// config.RecordingsConfig and expandNameTemplate for the token set.
	FilenameTemplate string
	PathTemplate     string
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
		// NXDN VCH voice is AMBE+2 "Enhanced Half Rate" (3600x2450), the
		// same codebook DMR uses — and the composer's NXDN voice chain
		// FEC-decodes 49-bit frames with the 3600x2450 C0/C1 Golay +
		// descramble (internal/radio/nxdn/voice_ambe.go), so it renders
		// through the 3600x2450 decoder, not the 3600x2400 "ambe2".
		// (Unverified on air — the 2450-vs-2400 codebook is one of the
		// things an NXDN voice capture confirms; see voice_ambe.go.)
		"nxdn":  "ambe2-dmr",   // NXDN VCH — AMBE+2 3600x2450 (EHR)
		"dpmr":  "ambe2",       // dPMR Mode 3 (digital)
		"tetra": "tetra-acelp", // TETRA full-rate voice — clean-room ACELP (EN 300 395-2)
	}
}

// dmrVoiceProtocol reports whether protocol is a DMR voice protocol.
// DMR voice calls always get a .raw sidecar (alongside the decoded
// WAV) so the on-air AMBE frames remain available for out-of-band
// tools even though the in-process vocoder now renders them.
func dmrVoiceProtocol(protocol string) bool {
	return protocol == "dmr-tier1" || protocol == "dmr-tier2" || protocol == "dmr-tier3"
}

// tetraVoiceProtocol reports whether a call is TETRA voice. The composer's
// TETRA chain now TCH/S-decodes each traffic burst and emits post-FEC 137-bit
// speech frames, which the clean-room ACELP vocoder ("tetra-acelp") renders to
// PCM — the same post-FEC-frames-in-`.raw` + decoded-WAV shape as DMR. The
// `.raw` sidecar is always written so the speech frames remain available for
// out-of-band tools.
func tetraVoiceProtocol(protocol string) bool {
	return protocol == "tetra"
}

// NewRecorder validates options and returns a recorder ready to Run.
// Like the engine, the recorder subscribes to the bus at construction
// so that CallStart events published before Run starts are not lost.
func NewRecorder(opts RecorderOptions) (*Recorder, error) {
	if opts.Bus == nil {
		return nil, errors.New("voice/recorder: events.Bus is required")
	}
	// An empty OutDir runs the recorder in decode-only mode: it still decodes
	// each call and feeds the live-audio tap, it just never writes files. This
	// is how live browser audio works without a configured recordings dir.
	decodeOnly := opts.OutDir == ""
	if opts.SampleRate == 0 {
		opts.SampleRate = pcmHzDefault
	}
	if opts.Log == nil {
		opts.Log = slog.Default()
	}
	if !decodeOnly {
		if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
			return nil, fmt.Errorf("voice/recorder: mkdir: %w", err)
		}
	}
	vocoderMap := opts.VocoderForProtocol
	if vocoderMap == nil {
		vocoderMap = DefaultVocoderForProtocol()
	}
	if opts.Normalize.Enabled {
		opts.Normalize = opts.Normalize.withDefaults()
	}
	if opts.Enhance.Enabled {
		opts.Enhance = opts.Enhance.WithDefaults()
	}
	loc := opts.DisplayLoc
	if loc == nil {
		loc = time.UTC
	}
	r := &Recorder{
		bus:                opts.Bus,
		log:                opts.Log,
		outDir:             opts.OutDir,
		sampleRate:         opts.SampleRate,
		writeRaw:           opts.WriteRaw,
		skipEncrypted:      opts.SkipEncrypted,
		writeCallJSON:      opts.WriteCallJSON,
		normalize:          opts.Normalize,
		enhance:            opts.Enhance,
		vocoderForProtocol: vocoderMap,
		displayLoc:         loc,
		filenameTmpl:       opts.FilenameTemplate,
		pathTmpl:           opts.PathTemplate,
		decodeOnly:         decodeOnly,
		dedup:              opts.Dedup,
		dedupSeen:          make(map[dedupKey]dedupEntry),
		sessions:           make(map[string]*recordingSession),
		callTalkers:        make(map[string][]srcEvent),
		pendingFinalize:    make(map[string]*pendingFinalize),
		runDone:            make(chan struct{}),
	}
	r.sub = opts.Bus.Subscribe()
	return r, nil
}

// dedupSuppress reports whether this CallStart is a cross-site duplicate of a
// call already recorded within the window from a DIFFERENT system, and records
// this call's (talkgroup, source) fingerprint for future checks. Networked /
// simulcast sites carry the same call under the same talkgroup and calling-radio
// RID, so only the first system to key it up records; later copies are skipped.
// A re-key on the SAME system is never suppressed (it is a new over). No-op
// (returns false) when dedup is disabled. now is passed in for testability.
func (r *Recorder) dedupSuppress(cs trunking.CallStart, now time.Time) bool {
	if !r.dedup.Enabled {
		return false
	}
	key := dedupKey{group: cs.Grant.GroupID, source: cs.Grant.SourceID}
	r.dedupMu.Lock()
	defer r.dedupMu.Unlock()
	if prev, ok := r.dedupSeen[key]; ok && now.Sub(prev.at) <= r.dedup.Window && prev.system != cs.Grant.System {
		// The same call is already being recorded from another system. Refresh
		// the window so a long call keeps suppressing late copies, keep the
		// original recording system, and skip this one.
		prev.at = now
		r.dedupSeen[key] = prev
		return true
	}
	r.dedupSeen[key] = dedupEntry{system: cs.Grant.System, at: now}
	if len(r.dedupSeen) > dedupMaxKeys {
		for k, e := range r.dedupSeen {
			if now.Sub(e.at) > r.dedup.Window {
				delete(r.dedupSeen, k)
			}
		}
	}
	return false
}

// VoiceEnhance returns the current voice enhancement config (defaults
// already backfilled when enabled). Read under the lock so a concurrent
// SetVoiceEnhance never tears the struct.
func (r *Recorder) VoiceEnhance() mbe.EnhancerConfig {
	r.enhanceMu.RLock()
	defer r.enhanceMu.RUnlock()
	return r.enhance
}

// SetVoiceEnhance swaps the voice enhancement config at runtime. The
// change applies to the NEXT call (each call builds a fresh vocoder that
// reads VoiceEnhance); in-flight calls keep the config they started with.
// This backs the live "enhance on/off" toggle in PATCH /api/v1/settings.
func (r *Recorder) SetVoiceEnhance(cfg mbe.EnhancerConfig) {
	if cfg.Enabled {
		cfg = cfg.WithDefaults()
	}
	r.enhanceMu.Lock()
	r.enhance = cfg
	r.enhanceMu.Unlock()
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
	// On teardown, finalize any calls still deferred for drain coordination so a
	// pending recording is flushed to disk rather than left open until its safety
	// timer fires (or lost if the process exits first).
	defer r.flushPendingFinalize()
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
					enc := ce.AlgorithmID != algorithmClear
					r.backfillSessionGrant(ce.DeviceSerial, enc, ce.AlgorithmID, ce.KeyID, 0)
					r.handleEncryptionUpdate(ce.DeviceSerial, enc)
				}
			case events.KindCallSourceUpdate:
				// In-call source/encryption resolved on the traffic channel
				// (e.g. a P25 Phase 2 compressed grant). Carries an explicit
				// encrypted flag.
				if su, ok := ev.Payload.(trunking.CallSourceUpdate); ok {
					r.backfillSessionGrant(su.DeviceSerial, su.Encrypted, 0, 0, su.SourceID)
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
	s := r.sessionForWrite(deviceSerial, 0)
	if s == nil {
		return nil
	}
	// Decode-only sessions never open a WAV. Analog PCM still reaches the live
	// stream via the composer's direct audioPub fanout, so there is nothing to
	// do here but drop it.
	if s.wav == nil {
		return nil
	}
	return s.wav.WriteSamples(samples)
}

// sessionForWrite returns the live session for serial, lazily opening
// the next per-transmission segment file when the session is dormant
// (parked after a KindCallSegment roll). Returns nil when no session is
// open for the device.
//
// callID fences a stale frame from the call that previously held this
// serial: when it is non-zero and the open session's CallID is non-zero
// and they differ, the frame is rejected (returns nil) WITHOUT reopening
// a dormant session, so a mismatched frame can't even spawn an empty
// file. A zero on either side matches (un-stamped / synthetic calls),
// preserving prior behaviour for callers that pass 0.
func (r *Recorder) sessionForWrite(serial string, callID uint64) *recordingSession {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[serial]
	if !ok {
		return nil
	}
	if callID != 0 && s.callID != 0 && callID != s.callID {
		return nil
	}
	// Decode-only: no files are ever opened. The session already carries its
	// per-protocol vocoder (buildSession), so WriteRawFrame can decode and fan
	// PCM to the live tap; hand it back with wav == nil. Segment rolls never
	// park a decode-only session (handleSegment bails on wav == nil), so the
	// dormant-rebuild path below is unreachable here.
	if r.decodeOnly {
		return s
	}
	if s.wav == nil {
		// A dormant post-segment park carries only cs+callID with no paths
		// yet, so prepare a fresh session for it under a new timestamp. A
		// session prepared by handleStart already has its paths/vocoder and
		// just needs its files opened on this first write.
		if s.wavPath == "" {
			ns := r.buildSession(s.cs, time.Now().UTC())
			if ns == nil {
				return nil
			}
			r.sessions[serial] = ns
			s = ns
		}
		if err := r.openSessionFiles(s); err != nil {
			// The WAV couldn't be created — drop the session so we don't
			// retry on every frame. Nothing partial lands on disk.
			delete(r.sessions, serial)
			return nil
		}
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
	return r.writeRawFrame(deviceSerial, 0, frame, 0, false)
}

// WriteRawFrameForCall is WriteRawFrameWithErrors plus the call identity
// (Grant.CallID) the frame was decoded for. A frame whose callID doesn't
// match the open session is dropped, fencing the cross-call audio-bleed
// window when a voice-tap serial is reused for the next call before the
// previous chain has fully drained. Digital voice chains that know their
// call's CallID call this in preference to WriteRawFrameWithErrors.
func (r *Recorder) WriteRawFrameForCall(deviceSerial string, callID uint64, frame []byte, correctedBits int) error {
	return r.writeRawFrame(deviceSerial, callID, frame, correctedBits, true)
}

// WriteRawFrameWithErrors is WriteRawFrame plus the channel FEC
// corrected-bit count for the frame. When the session's vocoder implements
// voice.ErrorAware (the pure-Go IMBE decoder), the count is handed to it
// before Decode so adaptive smoothing can estimate the channel error rate.
// Callers without an error count use WriteRawFrame.
func (r *Recorder) WriteRawFrameWithErrors(deviceSerial string, frame []byte, correctedBits int) error {
	return r.writeRawFrame(deviceSerial, 0, frame, correctedBits, true)
}

func (r *Recorder) writeRawFrame(deviceSerial string, callID uint64, frame []byte, correctedBits int, haveErrs bool) error {
	s := r.sessionForWrite(deviceSerial, callID)
	if s == nil {
		return nil
	}
	s.rawFrames++
	if s.raw != nil {
		if _, err := s.raw.Write(frame); err != nil {
			return err
		}
	}
	// Fan the verbatim raw frame to the live raw tap (the gRPC audio
	// publisher's include_raw subscribers) before any decode. Fires for
	// every protocol with an open session — including ProVoice / encrypted
	// calls that have no in-process decoder — since the raw bytes exist even
	// when decoded PCM doesn't. Hands the session's CallID to a call-aware
	// sink so the live raw stream is fenced by the same identity the WAV and
	// decoded tap use, closing the cross-call bleed window on a reused tap
	// serial (the frame already passed sessionForWrite's CallID check, so
	// s.callID is authoritative for this audio).
	if r.rawTap != nil {
		if cs, ok := r.rawTap.(RawFrameCallSink); ok {
			_ = cs.WriteRawFrameForCall(deviceSerial, s.callID, s.vocoderName, frame)
		} else {
			_ = r.rawTap.WriteRawFrame(deviceSerial, s.vocoderName, frame)
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
		if n := len(samples); n > 0 {
			s.lastSample = samples[n-1]
		}
		// Decode-only sessions have no WAV — skip the file write but still fan
		// the decoded PCM to the live tap below so browser audio works without
		// a recordings directory.
		if s.wav != nil {
			if err := s.wav.WriteSamples(samples); err != nil {
				return err
			}
		}
		// Fan the freshly-decoded PCM to the live consumers (web
		// stream, host player, tone-out). For digital protocols this
		// is the only point where PCM exists — the composer never
		// calls WritePCM for them — so without this the live audio
		// path is silent while recordings play fine (issue #598). Runs
		// outside r.mu (sessionForWrite released it), so the tap is
		// free to take its own locks. Hand the session's CallID to a
		// call-aware sink so the live stream is fenced by the same identity
		// the WAV used — closing the cross-call bleed window when a tap
		// serial is reused (the samples already passed sessionForWrite's
		// CallID check, so s.callID is authoritative for this audio).
		if r.decodedTap != nil {
			if cs, ok := r.decodedTap.(DecodedPCMCallSink); ok {
				_ = cs.WritePCMForCall(deviceSerial, s.callID, samples)
			} else {
				_ = r.decodedTap.WritePCM(deviceSerial, samples)
			}
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
	if !r.decodeOnly && cs.Talkgroup != nil && !cs.Talkgroup.Record {
		// Talkgroup is flagged record=false — follow and play the
		// call live, but write no WAV/raw files for it. A decode-only
		// recorder writes nothing regardless, so it must NOT drop the
		// call here or the talkgroup would be silent live too.
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
	if !r.decodeOnly && r.dedupSuppress(cs, time.Now()) {
		// Cross-site duplicate: this exact call (talkgroup + source) is already
		// being recorded from another monitored system. Skip the file write so a
		// call heard on several networked/simulcast sites is saved once. Live
		// follow/monitoring is unaffected (this gate is recording-only).
		r.log.Info("recorder: skipping cross-site duplicate recording",
			"device", cs.DeviceSerial, "system", cs.Grant.System,
			"tg", cs.Grant.GroupID, "src", cs.Grant.SourceID)
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
	// Reset+seed the call-level talker list for this new call (the previous call
	// on this serial, if any, is over). Later talkers accrue in backfillSessionGrant.
	if cs.Grant.SourceID != 0 {
		r.callTalkers[cs.DeviceSerial] = []srcEvent{{src: cs.Grant.SourceID, at: cs.StartedAt, emergency: cs.Grant.Emergency}}
	} else {
		delete(r.callTalkers, cs.DeviceSerial)
	}
	r.log.Info("recorder: call started",
		"device", cs.DeviceSerial, "wav", s.wavPath,
		"tg", cs.Grant.GroupID, "provoice", cs.Grant.ProVoice,
		"vocoder", s.vocoderName)
}

// algorithmClear is the encryption Algorithm ID a clear (unencrypted)
// call advertises; anything else means the call is encrypted. Mirrors
// p25.AlgorithmClear, kept local to avoid a radio-package import.
const algorithmClear uint8 = 0x80

// backfillSessionGrant mirrors onto the recorder's stored grant the encryption
// and source facts the engine recovers mid-call — a P25 Phase 1 LDU2 Encryption
// Sync or a Phase 2 traffic-channel GROUP_VOICE_CHANNEL_USER PDU — and backfills
// onto the live ActiveCall. Without this the CallComplete the recorder later
// publishes (built from the session's grant, see finalizeLocked) carries the
// grant-time snapshot, so a call whose encryption only resolves on the traffic
// channel broadcasts as encrypted=false even though the call log — fed from the
// engine-backfilled CallEnd grant — correctly shows it encrypted (issue #897).
// Encryption is sticky-true and never cleared here; alg/key are recorded only
// for an encrypted call; a zero source is ignored so a later clear-source update
// can't erase a known RID. No-op when no session is open for the device. Caller
// must not hold r.mu.
func (r *Recorder) backfillSessionGrant(deviceSerial string, encrypted bool, algID uint8, keyID uint16, sourceID uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[deviceSerial]
	if !ok {
		return
	}
	if encrypted {
		s.cs.Grant.Encrypted = true
		if algID != 0 {
			s.cs.Grant.AlgorithmID = algID
		}
		if keyID != 0 {
			s.cs.Grant.KeyID = keyID
		}
	}
	if sourceID != 0 {
		s.cs.Grant.SourceID = sourceID
		now := time.Now().UTC()
		// Record a talker change for the .json srcList: append when the source
		// differs from the last one recorded for this file (a genuine talker
		// change, not a repeat backfill of the same source).
		if n := len(s.srcs); n == 0 || s.srcs[n-1].src != sourceID {
			s.srcs = append(s.srcs, srcEvent{src: sourceID, at: now, emergency: s.cs.Grant.Emergency})
		}
		// Also accrue onto the call-level list (spans segment rolls) so every
		// transmission's sidecar can list all the call's talkers.
		if t := r.callTalkers[deviceSerial]; len(t) == 0 || t[len(t)-1].src != sourceID {
			r.callTalkers[deviceSerial] = append(t, srcEvent{src: sourceID, at: now, emergency: s.cs.Grant.Emergency})
		}
	}
}

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
	delete(r.callTalkers, deviceSerial) // call aborted — drop its talker history
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

// buildSession prepares a recording session for a call: it makes the output
// directory, instantiates the vocoder, picks the WAV sample rate, and
// resolves the .wav / .raw paths (named from cs but timestamped at startedAt,
// which differs from cs.StartedAt for a per-transmission segment roll). It
// deliberately does NOT open the files — that happens lazily on the first
// write (openSessionFiles) so a call with no audio leaves nothing on disk.
// Returns nil on a fatal error (already logged). The caller holds r.mu and
// registers the returned session.
func (r *Recorder) buildSession(cs trunking.CallStart, startedAt time.Time) *recordingSession {
	// The talkgroup directory is NOT created here — it is made lazily in
	// openSessionFiles, just before the first file is opened. A grant that is
	// followed but never yields audio (a dead-key, an immediately-aborted
	// encrypted call, or a voice tap left off-frequency while a hunt borrows
	// the SDR) therefore leaves no empty <system>/<talkgroup> folder behind.
	dir := r.directoryFor(cs)
	nameCS := cs
	nameCS.StartedAt = startedAt
	base := r.ensureUniqueBase(dir, r.basenameFor(nameCS))
	s := &recordingSession{startedAt: startedAt, cs: cs, callID: cs.Grant.CallID}
	// Seed the trunk-recorder freqList / srcList accumulators. The frequency is
	// always known; the source may not be yet (a call that opens before the
	// calling party resolves), in which case the first talker is recorded when
	// backfillSessionGrant fills it in.
	s.freqs = []freqEvent{{freq: cs.Grant.FrequencyHz, at: startedAt}}
	if cs.Grant.SourceID != 0 {
		s.srcs = []srcEvent{{src: cs.Grant.SourceID, at: startedAt, emergency: cs.Grant.Emergency}}
	}
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
			// Install the opt-in enhancement chain if this vocoder
			// supports it and enhancement is currently enabled. Reading
			// the config here (per call) is what makes runtime toggles via
			// SetVoiceEnhance take effect on the next call.
			if enh, ok := v.(VoiceEnhanceable); ok {
				enh.SetVoiceEnhancer(r.VoiceEnhance())
			}
			// Suppress the receiver-acquisition "startup scratch" on the
			// recording/live path (opt-in on the vocoder; off in the raw
			// decoder so unit tests are unaffected).
			if sq, ok := v.(StartupSquelchable); ok {
				sq.EnableStartupSquelch()
			}
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
	s.wavPath = filepath.Join(dir, base+".wav")
	// ProVoice and DMR voice grants always get a sidecar — neither has
	// an in-process vocoder, so the .raw file is the only capture of
	// the call.
	if r.writeRaw || cs.Grant.ProVoice || dmrVoiceProtocol(cs.Grant.Protocol) || tetraVoiceProtocol(cs.Grant.Protocol) {
		s.rawPath = filepath.Join(dir, base+".raw")
		s.rawWanted = true
	}
	// The on-disk files are NOT opened here — they are created lazily on the
	// first write (openSessionFiles). A grant that never yields audio (a
	// dead-key, an immediately-aborted encrypted call, or a tap that never
	// emits a frame) therefore leaves nothing on disk, instead of a 44-byte
	// header-only .wav and a 0-byte .raw.
	return s
}

// openSessionFiles opens a prepared session's WAV (and .raw sidecar, when
// wanted) if they aren't open yet. Called from the write path on the first
// sample/frame so empty calls never touch disk. Idempotent: a no-op once the
// WAV is open. On WAV-open failure the vocoder is closed and an error is
// returned so the caller drops the frame.
func (r *Recorder) openSessionFiles(s *recordingSession) error {
	if s.wav != nil {
		return nil
	}
	// Create the talkgroup directory lazily, on the first write, so a call
	// that never yields audio leaves no empty folder (see buildSession).
	if dir := filepath.Dir(s.wavPath); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			r.log.Error("recorder: mkdir", "dir", dir, "err", err)
			return err
		}
	}
	wav, err := NewWavFile(s.wavPath, s.sampleRate)
	if err != nil {
		r.log.Error("recorder: open wav", "path", s.wavPath, "err", err)
		if s.vocoder != nil {
			_ = s.vocoder.Close()
			s.vocoder = nil
		}
		return err
	}
	s.wav = wav
	if s.rawWanted && s.raw == nil {
		raw, err := os.Create(s.rawPath)
		if err != nil {
			r.log.Error("recorder: open raw", "path", s.rawPath, "err", err)
		} else {
			s.raw = raw
		}
	}
	return nil
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
	cc, meta := r.finalizeLocked(s, seg.DeviceSerial, seg.At, trunking.EndReasonNormal)
	// Park a dormant session: keeps the call's identity so the next write
	// opens the next segment file, without creating an empty trailing
	// file if no further audio arrives before the call ends.
	r.sessions[seg.DeviceSerial] = &recordingSession{cs: s.cs, callID: s.callID}
	r.mu.Unlock()
	if cc != nil {
		r.normalizeIfEnabled(cc.AudioPath)
		r.writeCallMetaFor(cc.AudioPath, meta)
		r.bus.Publish(events.Event{Kind: events.KindCallComplete, Payload: *cc})
	}
}

// writeCallMetaFor writes the trunk-recorder .json sidecar next to a finished
// recording, off the recorder lock (like normalizeIfEnabled). A nil meta (JSON
// disabled or nothing to write) is a no-op; a failure is logged and the
// recording is kept.
func (r *Recorder) writeCallMetaFor(wavPath string, meta *callMeta) {
	if meta == nil {
		return
	}
	if err := writeCallMeta(wavPath, meta); err != nil {
		r.log.Warn("recorder: writing call .json sidecar failed", "wav", wavPath, "err", err)
	}
}

// normalizeIfEnabled applies per-call loudness normalization to a finished
// WAV when configured. It runs after r.mu is released (whole-file I/O must
// not block other sessions) and before CallComplete is published, so the
// MP3/upload path sees normalized audio. A failure is logged and the
// original recording is kept — normalization never drops a call.
func (r *Recorder) normalizeIfEnabled(wavPath string) {
	if !r.normalize.Enabled || wavPath == "" {
		return
	}
	if err := normalizeWAVFile(wavPath, r.normalize); err != nil {
		r.log.Warn("recorder: loudness normalize failed", "wav", wavPath, "err", err)
	}
}

// finalizeLocked closes a session's files and returns a CallComplete to
// publish when audio was written; an empty file (no PCM) is removed and
// nil returned. Caller holds r.mu and publishes after releasing it.
// finalizeLocked returns the CallComplete to publish and, when a JSON sidecar
// is enabled and audio was written, the trunk-recorder metadata to write for it
// (both nil when the recording is discarded). The caller writes the JSON and
// publishes after releasing r.mu.
func (r *Recorder) finalizeLocked(s *recordingSession, serial string, endedAt time.Time, reason trunking.EndReason) (*trunking.CallComplete, *callMeta) {
	// Callers only reach here with an open WAV (handleEnd/handleSegment guard on
	// s.wav != nil); a decode-only session has no file to finalize. Guard anyway
	// so a future caller can't panic on the nil writer.
	if s.wav == nil {
		_ = s.close()
		return nil, nil
	}
	dataBytes := s.wav.DataBytes()
	vs, haveStats := r.voiceStatsFor(s)
	r.logVoiceStats(serial, s)
	// Audio-vs-wall-clock: a decoded (digital) call whose WAV runs much shorter
	// than the call's wall-clock span lost voice frames upstream — talkgroup
	// gating or undecoded windows — rather than being a genuinely short over.
	// Surfacing both makes the "16 s call, 9 s recording" symptom self-evident
	// in the log without diffing the WAV against the call record. Digital only
	// (analog FM writes continuous PCM, so audio ≈ wall by construction). Logged
	// at DEBUG: a multi-over call legitimately drops the silent gaps between
	// overs, so this is an investigation aid, not a routine warning.
	if s.vocoder != nil && dataBytes > 0 && s.sampleRate > 0 {
		audioSec := float64(dataBytes) / (float64(s.sampleRate) * 2)
		wallSec := endedAt.Sub(s.startedAt).Seconds()
		if wallSec > 1 && audioSec < 0.75*wallSec {
			// audio_pct is the frame yield: how much of the call span actually
			// decoded to audio. A very low value (e.g. 0.1 s / 5.4 s = ~2%) with
			// a small frames count is the fingerprint of upstream frame loss —
			// on TETRA, few TCH/S bursts passing the class-2 CRC (a marginal
			// same-carrier signal without soft-decision coding gain) or bursts
			// mis-routed away from the call by the usage-marker demux — rather
			// than a genuinely short over. (IQ starvation of the tap is a distinct
			// cause, surfaced separately by the ccdecoder "dropped IQ to a lagging
			// voice consumer" warning; a zero there rules it out.) Surfacing
			// frames + pct makes this diagnosable from this one line.
			r.log.Debug("recorder: recording shorter than call span",
				"device", serial, "wav", s.wavPath, "vocoder", s.vocoderName,
				"audio_seconds", round1(audioSec), "wall_seconds", round1(wallSec),
				"frames", s.rawFrames, "audio_pct", round1(100*audioSec/wallSec))
		}
	}
	// Smooth an abrupt end-of-call cut on decoded digital voice. A short
	// transmission that stops mid-waveform leaves a non-zero final sample
	// that clicks/scratches on playback — the "trailing" artifact most
	// audible on short overs, where the tail is a big fraction of the call.
	// Ramp the last decoded sample down to zero on both the WAV and the live
	// decoded fan-out, mirroring the analog FM chain's squelch-tail fade.
	// Digital only (s.vocoder != nil): analog already faded in the composer,
	// and a silent/dead-key call has lastSample == 0 so fadeTail is a no-op.
	if s.vocoder != nil && s.wav != nil && dataBytes > 0 {
		n := int(s.sampleRate) / 100 // ~10 ms
		if n < 8 {
			n = 8
		}
		if tail := fadeTail(s.lastSample, n); tail != nil {
			if err := s.wav.WriteSamples(tail); err != nil {
				r.log.Warn("recorder: writing call-tail fade failed",
					"device", serial, "err", err)
			}
			if r.decodedTap != nil {
				if cs, ok := r.decodedTap.(DecodedPCMCallSink); ok {
					_ = cs.WritePCMForCall(serial, s.callID, tail)
				} else {
					_ = r.decodedTap.WritePCM(serial, tail)
				}
			}
		}
	}
	if err := s.close(); err != nil {
		r.log.Error("recorder: close session", "err", err)
	}
	// A vocoder-decoded call that produced no real speech (every frame was
	// idle/silent/bad — voiced+unvoiced == 0) is a dead-key/idle carrier:
	// the WAV holds only muted silence, so it's worthless and, in
	// per-transmission mode, the dominant source of tiny-file spam. Remove
	// both outputs and publish no CallComplete. Scoped to StatProvider
	// vocoders (the pure-Go IMBE decoder) — analog and ProVoice/DMR can't be
	// classified here and keep the dataBytes behaviour below.
	if haveStats && vs.Voiced+vs.Unvoiced == 0 {
		r.removeSessionFiles(s)
		return nil, nil
	}
	// No PCM decoded into the WAV — nothing to stream. The files are left
	// in place: a digital call (ProVoice / DMR / pre-vocoder) keeps its
	// .raw sidecar as the only capture even when the WAV is empty.
	// Per-transmission segment rolls never reach here empty because the
	// next file is opened lazily on the first write.
	if dataBytes == 0 {
		return nil, nil
	}
	cc := &trunking.CallComplete{
		Grant:        s.cs.Grant,
		Talkgroup:    s.cs.Talkgroup,
		DeviceSerial: serial,
		StartedAt:    s.startedAt,
		EndedAt:      endedAt,
		Reason:       reason,
		AudioPath:    s.wavPath,
		SampleRate:   s.sampleRate,
	}
	var meta *callMeta
	if r.writeCallJSON {
		// Prefer the call-level talker list (all talkers of the whole call, across
		// transmission rolls) so a rolled segment's sidecar still lists every
		// srcID; fall back to this file's own srcs when none was accumulated.
		srcs := s.srcs
		if ct := r.callTalkers[serial]; len(ct) > 0 {
			srcs = ct
		}
		meta = buildCallMeta(s.cs, s.startedAt, endedAt, int(r.callNum.Add(1)), s.vocoder != nil, srcs, s.freqs)
	}
	return cc, meta
}

// removeSessionFiles deletes a session's WAV and .raw sidecar from disk.
// Used to discard a recording that captured no useful audio (a dead-key /
// idle-only call) so it doesn't spam the recordings tree. The session must
// already be closed. Mirrors the removal in handleEncryptionUpdate.
func (r *Recorder) removeSessionFiles(s *recordingSession) {
	for _, p := range []string{s.wavPath, s.rawPath} {
		if p == "" {
			continue
		}
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			r.log.Warn("recorder: removing empty recording", "path", p, "err", err)
		}
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
	vs, ok := r.voiceStatsFor(s)
	if !ok {
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
	// No decodable speech: every frame was idle/silent/bad (the call
	// produced no voiced OR unvoiced audio). Escalate to WARN with the b_0
	// range + first raw frame so an operator can tell a genuine dead-key
	// (b_0 varying within the idle corner [0,7]) from empty/zero frames
	// reaching the vocoder — a mistuned tap or extraction bug (b_0 pinned
	// at 0). The recording itself is suppressed by finalizeLocked.
	if vs.Voiced+vs.Unvoiced == 0 {
		r.log.Warn("recorder: no decodable speech — likely dead-key/idle carrier or mistuned tap",
			append(args,
				"min_b0", vs.MinB0, "max_b0", vs.MaxB0,
				"first_frame_hex", vs.FirstFrameHex)...)
		return
	}
	if vs.ClipPct() > voiceClipWarnPct {
		r.log.Warn("recorder: voice audio quality — output clipping (vocoder gain too hot)", args...)
		return
	}
	r.log.Debug("recorder: voice audio quality", args...)
}

// voiceStatsFor returns the session vocoder's per-call VoiceStats when the
// vocoder implements StatProvider and has decoded at least one frame. It is
// the shared accessor for the per-call quality log and the empty-recording
// suppression in finalizeLocked.
func (r *Recorder) voiceStatsFor(s *recordingSession) (VoiceStats, bool) {
	sp, ok := s.vocoder.(StatProvider)
	if !ok {
		return VoiceStats{}, false
	}
	vs, have := sp.VoiceStats()
	if !have || vs.Frames == 0 {
		return VoiceStats{}, false
	}
	return vs, true
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
func round2(v float64) float64 { return math.Round(v*100) / 100 }

// drainFinalizeTimeout bounds how long a drain-coordinated call waits for the
// composer's NotifyDrainComplete after its KindCallEnd before finalizing anyway.
// It is a safety backstop for the rare case where the composer never signals —
// e.g. the event bus dropped the composer's copy of KindCallEnd under
// backpressure (Bus.Publish drops for slow subscribers), so the composer never
// tears down that chain. The normal path finalizes immediately on the signal
// (drains are milliseconds), so this timeout only ever fires on a lost signal;
// it is generous so it never pre-empts a legitimately in-progress drain (which
// would re-introduce the tail-drop it exists to prevent).
const drainFinalizeTimeout = 3 * time.Second

// pendingFinalize tracks one call deferred for drain coordination. It finalizes
// when both haveCE (KindCallEnd seen) and drained (NotifyDrainComplete seen) are
// true, or when timer fires. Guarded by Recorder.mu.
type pendingFinalize struct {
	ce      trunking.CallEnd
	haveCE  bool
	drained bool
	timer   *time.Timer
}

// EnableDrainCoordination switches the recorder into drain-coordinated finalize:
// handleEnd defers a call's finalize until the composer signals the call's voice
// chain has drained (NotifyDrainComplete), closing the race where the recorder
// deleted the session before the chain wrote the transmission tail. Called once
// by the composer at construction when it is wired to this recorder. Idempotent.
func (r *Recorder) EnableDrainCoordination() {
	r.mu.Lock()
	r.drainCoordinated = true
	r.mu.Unlock()
}

// NotifyDrainComplete tells the recorder that the composer has finished draining
// the named call's voice chain (all tail frames written). If the call's
// KindCallEnd has already been seen, the call finalizes now; otherwise the drain
// is recorded and finalize happens when handleEnd runs. A serial with no open
// session and no pending CallEnd is ignored (nothing to finalize).
func (r *Recorder) NotifyDrainComplete(deviceSerial string) {
	r.mu.Lock()
	if !r.drainCoordinated {
		r.mu.Unlock()
		return
	}
	st := r.pendingFinalize[deviceSerial]
	if st == nil {
		// Drain signal arrived before handleEnd. Only track it if a session is
		// still open (a CallEnd for it is therefore still coming); otherwise
		// there is nothing to finalize and tracking it would leak.
		if _, hasSession := r.sessions[deviceSerial]; !hasSession {
			r.mu.Unlock()
			return
		}
		st = &pendingFinalize{}
		r.pendingFinalize[deviceSerial] = st
	}
	st.drained = true
	if !st.haveCE {
		r.mu.Unlock()
		return // wait for handleEnd to supply the CallEnd
	}
	ce := st.ce
	if st.timer != nil {
		st.timer.Stop()
	}
	delete(r.pendingFinalize, deviceSerial)
	r.mu.Unlock()
	r.finalizeCall(ce)
}

// finalizeOnDrainTimeout is the safety backstop: it finalizes a deferred call
// whose drain signal never arrived within drainFinalizeTimeout.
func (r *Recorder) finalizeOnDrainTimeout(deviceSerial string) {
	r.mu.Lock()
	st := r.pendingFinalize[deviceSerial]
	if st == nil {
		r.mu.Unlock()
		return // already finalized by NotifyDrainComplete
	}
	ce := st.ce
	delete(r.pendingFinalize, deviceSerial)
	r.mu.Unlock()
	r.log.Warn("recorder: drain signal missing; finalizing call on timeout",
		"device", deviceSerial, "timeout", drainFinalizeTimeout)
	r.finalizeCall(ce)
}

func (r *Recorder) handleEnd(ce trunking.CallEnd) {
	if !r.drainCoordinated {
		r.finalizeCall(ce)
		return
	}
	// Drain-coordinated: keep the session alive until the composer signals the
	// call's voice chain has drained, so the tail frames it writes during
	// teardown are not dropped onto a deleted session. Finalize when both this
	// CallEnd and the drain signal have arrived, or when the safety timer fires.
	serial := ce.DeviceSerial
	r.mu.Lock()
	if _, hasSession := r.sessions[serial]; !hasSession {
		// No session to keep alive (skipped/dead call): finalizeCall is a cheap
		// no-op cleanup — run it now rather than wait for a drain signal.
		r.mu.Unlock()
		r.finalizeCall(ce)
		return
	}
	st := r.pendingFinalize[serial]
	if st == nil {
		st = &pendingFinalize{}
		r.pendingFinalize[serial] = st
	}
	st.ce = ce
	st.haveCE = true
	if st.drained {
		if st.timer != nil {
			st.timer.Stop()
		}
		delete(r.pendingFinalize, serial)
		r.mu.Unlock()
		r.finalizeCall(ce)
		return
	}
	if st.timer == nil {
		st.timer = time.AfterFunc(drainFinalizeTimeout, func() { r.finalizeOnDrainTimeout(serial) })
	}
	r.mu.Unlock()
}

// flushPendingFinalize finalizes every call still awaiting drain coordination.
// Called on Run teardown so nothing is left dangling.
func (r *Recorder) flushPendingFinalize() {
	r.mu.Lock()
	pend := r.pendingFinalize
	r.pendingFinalize = make(map[string]*pendingFinalize)
	ends := make([]trunking.CallEnd, 0, len(pend))
	for _, st := range pend {
		if st.timer != nil {
			st.timer.Stop()
		}
		if st.haveCE {
			ends = append(ends, st.ce)
		}
	}
	r.mu.Unlock()
	for _, ce := range ends {
		r.finalizeCall(ce)
	}
}

// finalizeCall closes and finalizes the recording for a completed call. It is
// the terminal step for both the immediate (uncoordinated) and drain-coordinated
// paths.
func (r *Recorder) finalizeCall(ce trunking.CallEnd) {
	r.mu.Lock()
	s, ok := r.sessions[ce.DeviceSerial]
	if ok {
		delete(r.sessions, ce.DeviceSerial)
	}
	if !ok {
		r.mu.Unlock()
		return
	}
	if s.wav == nil {
		// No open WAV: a decode-only session (never writes files), a dormant
		// post-segment park whose next over never arrived, or a dead-key call
		// that built a vocoder but decoded no frame. finalizeLocked assumes an
		// open WAV and there's nothing to publish, but the session may still
		// hold a live vocoder — close it so it isn't leaked.
		_ = s.close()
		delete(r.callTalkers, ce.DeviceSerial) // call over — drop its talker history
		r.mu.Unlock()
		return
	}
	wavPath := s.wavPath
	// Announce the finished WAV so the outbound-streaming subsystem can
	// upload it. Skip (and delete) calls that captured no PCM.
	cc, meta := r.finalizeLocked(s, ce.DeviceSerial, ce.EndedAt, ce.Reason)
	// The call is over — drop its talker history now that finalizeLocked has read
	// it for the closing sidecar (still under r.mu, before the unlock below).
	delete(r.callTalkers, ce.DeviceSerial)
	r.mu.Unlock()
	r.log.Info("recorder: call ended",
		"device", ce.DeviceSerial,
		"wav", wavPath,
		"duration", ce.Duration().Round(time.Millisecond),
		"reason", ce.Reason)
	if cc != nil {
		r.normalizeIfEnabled(cc.AudioPath)
		r.writeCallMetaFor(cc.AudioPath, meta)
		r.bus.Publish(events.Event{Kind: events.KindCallComplete, Payload: *cc})
	}
}

// defaultFilenameTemplate is the basename used when recordings.filename_template
// is unset: local date + time + talkgroup/dest id, e.g. 20260810_230041_1005479.
// It deliberately omits the timezone offset, source, and timeslot the older
// scheme put in every name — all of which remain in the .json sidecar — and
// relies on the -N collision suffix (ensureUniqueBase) for the rare concurrent
// same-second, same-talkgroup case (DMR Tier III TS1/TS2).
const defaultFilenameTemplate = "{date}_{time}_{tg}"

// expandNameTemplate substitutes the {token} set into a recording filename or
// path template, rendering time tokens in the display timezone. The token set is
// mirrored in config.namingTemplateTokens (validated at load) — keep them in
// sync. Path separators in the result are the caller's concern: sanitizeBasename
// strips them for a filename; sanitizePathTemplate keeps them for a directory.
func (r *Recorder) expandNameTemplate(tmpl string, cs trunking.CallStart, t time.Time) string {
	loc := r.displayLoc
	if loc == nil {
		loc = time.UTC
	}
	if t.IsZero() {
		t = time.Now()
	}
	lt := t.In(loc)
	tgID := fmt.Sprintf("%d", cs.Grant.GroupID)
	alpha := tgID
	if cs.Talkgroup != nil && cs.Talkgroup.AlphaTag != "" {
		alpha = sanitize(cs.Talkgroup.AlphaTag)
	}
	system := sanitize(cs.Grant.System)
	if system == "" {
		system = "unknown-system"
	}
	return strings.NewReplacer(
		"{datetime}", lt.Format("20060102T150405"),
		"{date}", lt.Format("20060102"),
		"{time}", lt.Format("150405"),
		"{year}", lt.Format("2006"),
		"{month}", lt.Format("01"),
		"{day}", lt.Format("02"),
		"{tg}", tgID,
		"{alpha}", alpha,
		"{freq}", fmt.Sprintf("%d", cs.Grant.FrequencyHz),
		"{src}", fmt.Sprintf("%d", cs.Grant.SourceID),
		"{ts}", fmt.Sprintf("%d", cs.Grant.Timeslot),
		"{proto}", sanitize(cs.Grant.Protocol),
		"{system}", system,
		"{callid}", fmt.Sprintf("%d", cs.Grant.CallID),
	).Replace(tmpl)
}

func (r *Recorder) directoryFor(cs trunking.CallStart) string {
	// Operator-supplied directory tree (recordings.path_template) is used verbatim
	// for every call — the individual/group split is left to the template author
	// (the flag is still in the .json). Each path segment is sanitised so a token
	// value can't inject "..", an empty segment, or a control character.
	if r.pathTmpl != "" {
		rel := r.expandNameTemplate(r.pathTmpl, cs, cs.StartedAt)
		parts := []string{r.outDir}
		for _, seg := range strings.Split(rel, "/") {
			if s := sanitize(seg); s != "" {
				parts = append(parts, s)
			}
		}
		return filepath.Join(parts...)
	}
	system := sanitize(cs.Grant.System)
	if system == "" {
		system = "unknown-system"
	}
	// A unit-to-unit / private (individual) call: GroupID is a target radio
	// address, not a talkgroup. File it under an "individual" subtree keyed by the
	// target SSI so these recordings never masquerade as phantom talkgroups — a
	// radio-ID-named directory alongside real talkgroups that the WebUI talkgroup
	// list never shows (the leaked "fake TG" recordings the operator reported).
	if cs.Grant.Individual {
		return filepath.Join(r.outDir, system, "individual", fmt.Sprintf("%d", cs.Grant.GroupID))
	}
	tgDir := fmt.Sprintf("%d", cs.Grant.GroupID)
	if cs.Talkgroup != nil && cs.Talkgroup.AlphaTag != "" {
		tgDir = sanitize(cs.Talkgroup.AlphaTag)
	}
	return filepath.Join(r.outDir, system, tgDir)
}

func (r *Recorder) basenameFor(cs trunking.CallStart) string {
	tmpl := r.filenameTmpl
	if tmpl == "" {
		tmpl = defaultFilenameTemplate
	}
	base := sanitize(r.expandNameTemplate(tmpl, cs, cs.StartedAt))
	if base == "" {
		// A template that renders empty (e.g. all-unknown tokens) would create
		// "<dir>/.wav"; fall back to the start-second stamp so a file is still named.
		t := cs.StartedAt
		if t.IsZero() {
			t = time.Now()
		}
		loc := r.displayLoc
		if loc == nil {
			loc = time.UTC
		}
		base = t.In(loc).Format("20060102_150405")
	}
	return base
}

// ensureUniqueBase returns base, or base with a -2/-3/… suffix, such that
// <dir>/<base>.wav collides with neither an on-disk file nor an active session.
// The default scheme drops the source/timeslot that used to disambiguate, so two
// concurrent same-talkgroup, same-second calls (DMR Tier III TS1/TS2) would
// otherwise land on one path and overwrite each other. Caller holds r.mu; the
// active-session scan is race-free because grant processing is single-goroutine
// and each session is registered before the next is built.
func (r *Recorder) ensureUniqueBase(dir, base string) string {
	candidate := base
	for n := 2; ; n++ {
		if !r.wavPathClaimed(filepath.Join(dir, candidate+".wav")) {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", base, n)
	}
}

// wavPathClaimed reports whether a WAV path already exists on disk or is held by
// an active recording session.
func (r *Recorder) wavPathClaimed(wav string) bool {
	if _, err := os.Stat(wav); err == nil {
		return true
	}
	for _, s := range r.sessions {
		if s != nil && s.wavPath == wav {
			return true
		}
	}
	return false
}

// fadeTail returns an n-sample linear ramp from `from` down toward zero,
// used to smooth an abrupt end-of-call cut on decoded digital voice (the
// digital counterpart of the analog composer's squelch-tail fade). The
// first sample equals `from` so the ramp continues seamlessly from the
// last decoded sample. Returns nil when there is nothing to ramp (already
// at silence, or a non-positive length).
func fadeTail(from int16, n int) []int16 {
	if from == 0 || n <= 0 {
		return nil
	}
	out := make([]int16, n)
	start := float64(from)
	for i := range out {
		out[i] = int16(start * (1.0 - float64(i)/float64(n)))
	}
	return out
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
	// lastSample is the most recent decoded PCM sample written for this
	// session. On close a short fade from it down to zero is appended to
	// smooth an abrupt end-of-call cut (finalizeLocked) — the digital
	// equivalent of the analog composer's squelch-tail fade. Zero (the
	// default / trailing silence) means no fade is needed.
	lastSample int16
	// rawWanted records whether this call should get a .raw sidecar once
	// its files are opened. The decision is made when the session is
	// prepared (buildSession) but the file itself is created lazily on the
	// first frame (openSessionFiles), so a call with no audio never leaves a
	// 0-byte .raw behind.
	rawWanted bool
	// rawFrames counts vocoder frames delivered to this session (each ~one
	// speech frame, e.g. 30 ms for TETRA ACELP). Surfaced in the
	// "recording shorter than call span" diagnostic so the frame yield of a
	// too-short digital recording is visible without cross-referencing the
	// composer's follow-ended log.
	rawFrames int
	startedAt time.Time
	// callID is the Grant.CallID of the call this session records. Frames
	// arriving via WriteRawFrameForCall with a different non-zero callID are
	// dropped (see sessionForWrite) so a reused tap serial can't bleed the
	// previous call's audio into this one. Preserved across per-transmission
	// segment rolls (the dormant session carries the same cs).
	callID uint64
	// cs is the originating CallStart, retained so a per-transmission
	// segment roll (KindCallSegment) can open the next file with the same
	// grant/talkgroup under a new timestamp. A session with wav == nil is
	// "dormant" — finalized at a segment boundary and waiting to open its
	// next file lazily on the first write, so an over with no following
	// audio never leaves an empty trailing file.
	cs trunking.CallStart
	// srcs / freqs accumulate the talker and frequency observations for this
	// file, seeded at buildSession and appended on talker changes
	// (backfillSessionGrant). buildCallMeta turns them into the trunk-recorder
	// srcList / freqList in the .json sidecar at finalize.
	srcs  []srcEvent
	freqs []freqEvent
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
