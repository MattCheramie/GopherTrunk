package main

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

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/iqtap"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// ddcVoiceTap is the subset of the control-channel decoder the DDC-tap capture
// consumes: the post-DDC channelised IQ fan-out (the same stream the same-carrier
// voice taps read) plus its pipeline rate and centre. *ccdecoder.Decoder
// satisfies it; tests supply a fake. Used only when auto_record's tap is "ddc".
type ddcVoiceTap interface {
	SubscribeVoiceIQ() (<-chan []complex64, func() uint64)
	PipelineRateHz() float64
	CenterFreqHz() uint32
}

// autoRecordConcurrencyWindow is how long a granted call is counted as
// "active" for the on_concurrent_calls trigger. The control channel repeats a
// call's grant well inside this window (typically every few hundred ms while
// the call is up), so a live call keeps its slot; a call that stops being
// granted ages out. This mirrors, at a coarse grain, the engine's own
// observed-call tracking — and, crucially, it counts calls the control channel
// announced whether or not a voice tuner is following them, which is the
// concurrency the operator sees in the "Active calls" view (and the case that
// matters for the same-carrier TETRA tap, where only one call is ever bound).
const autoRecordConcurrencyWindow = 4 * time.Second

// autoRecordMaxInFlight bounds simultaneous captures so a burst of triggers
// can't spawn unbounded goroutines/files. Excess triggers are dropped with a
// logged warning (never silently).
const autoRecordMaxInFlight = 2

// captureFunc actuates one raw-IQ capture to path. Swappable in tests so the
// trigger logic can be exercised without an SDR/broker.
type captureFunc func(ctx context.Context, path string, format siglab.SampleFormat, seconds int) (samples int64, drops uint64, err error)

// iqAutoRecorder subscribes to the event bus and captures a short slice of the
// control SDR's raw IQ whenever a configured classification fires: N+
// concurrent calls, a grant that found no free voice device
// (events.KindGrantUnserved), an encrypted/emergency grant, or a manual API
// trigger. Captures are self-describing (a `.metadata.json` sidecar) so they
// drop straight into `gophertrunk replay` / siglab. This is the event-driven
// debugging hook the operator asked for.
//
// Classification (which grants matter) is decoupled from actuation (writing the
// IQ), mirroring the voice Recorder: the engine already publishes the events;
// this type only decides and writes.
type iqAutoRecorder struct {
	cfg      config.BasebandAutoRecordConfig
	system   string // control system name, used for the manual trigger / fallback
	protocol string // control system protocol, used for capture metadata
	broker   *iqtap.Broker
	format   siglab.SampleFormat
	cooldown time.Duration
	dir      string
	seconds  int
	log      *slog.Logger

	// ddc lazily resolves the control decoder for the "ddc" tap (nil when the
	// decoder is not yet built / not applicable). Nil for the wideband tap.
	ddc func() ddcVoiceTap

	// capture actuates a capture; defaults to captureIQToFile over broker.
	capture captureFunc
	// now is the clock (injectable for tests).
	now func() time.Time

	mu       sync.Mutex
	baseCtx  context.Context
	seen     map[string]time.Time // observed-call key -> last grant time
	lastAuto time.Time            // last automatic (non-manual) trigger, for cooldown
	inFlight int
	reserved map[string]struct{} // capture paths handed out, so concurrent/same-second captures never collide
}

// newIQAutoRecorder builds an auto-recorder for the control SDR's broker.
// Returns nil when disabled, so the daemon can skip wiring it entirely. cfg is
// assumed already validated (config.Validate); format/cooldown fall back to
// their documented defaults.
func newIQAutoRecorder(cfg config.BasebandAutoRecordConfig, system, protocol, serial string, broker *iqtap.Broker, ddc func() ddcVoiceTap, log *slog.Logger) *iqAutoRecorder {
	if !cfg.Enabled {
		return nil
	}
	if log == nil {
		log = slog.Default()
	}
	// Empty format defaults to cs16 (compact, replay-ready) rather than
	// ParseSampleFormat's u8 default — the operator's captures are cs16.
	formatStr := strings.TrimSpace(cfg.Format)
	if formatStr == "" {
		formatStr = "cs16"
	}
	format, err := siglab.ParseSampleFormat(formatStr)
	if err != nil {
		// Validation already rejected a bad format; treat any residual error
		// as cs16 rather than failing daemon start.
		format = siglab.FormatS16
	}
	cooldown, err := cfg.CooldownDuration()
	if err != nil {
		cooldown = autoRecordDefaultCooldownFallback
	}
	a := &iqAutoRecorder{
		cfg:      cfg,
		system:   system,
		protocol: protocol,
		broker:   broker,
		format:   format,
		cooldown: cooldown,
		dir:      cfg.Dir,
		seconds:  cfg.Seconds,
		ddc:      ddc,
		log:      log.With("component", "iq-autorecord", "serial", serial),
		now:      time.Now,
		baseCtx:  context.Background(),
		seen:     make(map[string]time.Time),
		reserved: make(map[string]struct{}),
	}
	a.capture = a.defaultCapture
	return a
}

// autoRecordDefaultCooldownFallback mirrors config.autoRecordDefaultCooldown
// for the (validated-away) error path in the constructor.
const autoRecordDefaultCooldownFallback = 10 * time.Second

// inFlightCount returns the number of captures currently running. Used by
// tests to wait until the recorder is idle before tearing down temp dirs (a
// capture goroutine writes its metadata sidecar after the capture returns).
func (a *iqAutoRecorder) inFlightCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.inFlight
}

// defaultCapture writes the capture: the narrowband post-DDC channel stream when
// tap: ddc, otherwise the wideband broker via the shared captureIQToFile helper.
func (a *iqAutoRecorder) defaultCapture(ctx context.Context, path string, format siglab.SampleFormat, seconds int) (int64, uint64, error) {
	if a.cfg.TapDDC() {
		var tap ddcVoiceTap
		if a.ddc != nil {
			tap = a.ddc()
		}
		return captureDDCToFile(ctx, tap, path, format, seconds)
	}
	if a.broker == nil {
		return 0, 0, fmt.Errorf("iq-autorecord: no broker")
	}
	samples, _, drops, err := captureIQToFile(ctx, a.broker, path, format, seconds)
	return samples, drops, err
}

// captureDDCToFile records `seconds` of the control decoder's post-DDC channelised
// IQ (the pipeline-rate stream: 144 kHz for TETRA, ~48 kHz for the C4FM family) to
// path in the chosen format, reusing siglab.CaptureWriter so the bytes are
// identical to a wideband capture at that rate. Orders of magnitude smaller than
// the wideband broker capture, and directly replayable. ctx cancels early.
func captureDDCToFile(ctx context.Context, tap ddcVoiceTap, path string, format siglab.SampleFormat, seconds int) (samples int64, drops uint64, err error) {
	if tap == nil {
		return 0, 0, fmt.Errorf("iq-autorecord: no control decoder for ddc tap")
	}
	f, err := os.Create(path)
	if err != nil {
		return 0, 0, fmt.Errorf("iq-autorecord: create %s: %w", path, err)
	}
	ch, unsub := tap.SubscribeVoiceIQ()
	enc := siglab.NewCaptureWriter(f, format)

	// The DDC fan-out only broadcasts while the control pipeline is locked/active;
	// a timer arm ends the capture after the requested duration even if the stream
	// stalls (mirrors captureIQToFile's safety timer).
	timer := time.NewTimer(time.Duration(seconds) * time.Second)
	defer timer.Stop()
	deadline := time.Now().Add(time.Duration(seconds) * time.Second)

	streamErr := func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			case chunk, ok := <-ch:
				if !ok {
					return errors.New("iq-autorecord: voice tap closed before capture finished")
				}
				if werr := enc.Write(chunk); werr != nil {
					return fmt.Errorf("write: %w", werr)
				}
				samples += int64(len(chunk))
				if time.Now().After(deadline) {
					return nil
				}
			}
		}
	}()

	// Unsubscribe and read the fan-out's drop count for this capture: dropped IQ
	// chunks are time gaps in the grab that break downstream decode, exactly like
	// the wideband path's subscriber drops. runCapture surfaces the count (and the
	// fan-out logs its own warning at unsubscribe).
	drops = unsub()

	closeErr := f.Close()
	if streamErr == nil && closeErr != nil && !errors.Is(closeErr, os.ErrClosed) {
		streamErr = closeErr
	}
	return samples, drops, streamErr
}

// Run drains the event subscription until ctx is cancelled. sub is an
// events.Bus subscription (its C channel yields events.Event).
func (a *iqAutoRecorder) Run(ctx context.Context, sub *events.Subscription) error {
	a.mu.Lock()
	a.baseCtx = ctx
	a.mu.Unlock()
	a.log.Info("iq-autorecord: started",
		"dir", a.dir, "format", a.format.String(), "seconds", a.seconds,
		"cooldown", a.cooldown, "on_concurrent_calls", a.cfg.OnConcurrentCalls,
		"on_no_voice_device", a.cfg.OnNoVoiceDevice,
		"on_encrypted", a.cfg.OnEncrypted, "on_emergency", a.cfg.OnEmergency,
		"on_cc_sync_loss", a.cfg.OnCCSyncLoss)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-sub.C:
			if !ok {
				return nil
			}
			switch ev.Kind {
			case events.KindGrant:
				if g, ok := ev.Payload.(trunking.Grant); ok {
					a.onGrant(g)
				}
			case events.KindGrantUnserved:
				if g, ok := ev.Payload.(trunking.Grant); ok {
					a.onGrantUnserved(g)
				}
			case events.KindCCLost:
				a.onCCLost(ev.Payload)
			}
		}
	}
}

// observedKey identifies a logical call the way the engine does:
// (system, talkgroup, timeslot). Frequency is deliberately excluded — a call's
// frequency can change mid-call.
func observedKey(g trunking.Grant) string {
	return fmt.Sprintf("%s|%d|%d", g.System, g.GroupID, g.Timeslot)
}

// onGrant tracks concurrency and evaluates the per-grant + concurrency triggers.
func (a *iqAutoRecorder) onGrant(g trunking.Grant) {
	now := a.now()
	concurrent := a.trackConcurrency(g, now)
	if reason := a.classify(g, concurrent); reason != "" {
		a.maybeFire(reason, g.System, g.Protocol, false)
	}
}

// onGrantUnserved fires the no-voice-device trigger.
func (a *iqAutoRecorder) onGrantUnserved(g trunking.Grant) {
	if a.cfg.OnNoVoiceDevice {
		a.maybeFire("no_voice_device", g.System, g.Protocol, false)
	}
}

// onCCLost fires the CC-sync-loss trigger. events.KindCCLost is published only
// after a genuine lock (every protocol's MarkLost early-returns unless locked),
// so this is exclusively a "was locked, now lost" edge — never a hunt that never
// locked. The capture is forward-looking, so it records the re-acquisition after
// the drop: the raw IQ needed to debug sync-loss and slow warm-up-lock episodes,
// where the carrier is still present but GT fails to re-lock. The event payload
// carries the frequency but no system name, so the capture is labelled with the
// primary control system (a.system), like the manual trigger.
func (a *iqAutoRecorder) onCCLost(payload any) {
	if !a.cfg.OnCCSyncLoss {
		return
	}
	var freqHz uint32
	if lp, ok := payload.(trunking.LockedPayload); ok {
		freqHz = lp.LockedFrequencyHz()
	}
	a.log.Info("iq-autorecord: control-channel sync loss — capturing re-acquisition IQ",
		"system", a.system, "freq_hz", freqHz)
	a.maybeFire("cc_sync_loss", a.system, a.protocol, false)
}

// TriggerManual fires a capture on operator request, bypassing the cooldown.
// Returns the reason label used ("manual"); the actual write happens
// asynchronously.
func (a *iqAutoRecorder) TriggerManual() {
	a.maybeFire("manual", a.system, a.protocol, true)
}

// trackConcurrency records this grant's call as active and returns the number
// of distinct calls active within autoRecordConcurrencyWindow.
func (a *iqAutoRecorder) trackConcurrency(g trunking.Grant, now time.Time) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.seen[observedKey(g)] = now
	cutoff := now.Add(-autoRecordConcurrencyWindow)
	for k, t := range a.seen {
		if t.Before(cutoff) {
			delete(a.seen, k)
		}
	}
	return len(a.seen)
}

// classify returns the trigger reason for a grant given the current concurrency,
// or "" for no trigger. Pure given its inputs — no IO, no cooldown. Per-grant
// conditions (emergency, encrypted) take precedence over the concurrency
// condition so the filename reflects the most specific cause.
func (a *iqAutoRecorder) classify(g trunking.Grant, concurrent int) string {
	switch {
	case a.cfg.OnEmergency && g.Emergency:
		return "emergency"
	case a.cfg.OnEncrypted && g.Encrypted:
		return "encrypted"
	case a.cfg.OnConcurrentCalls > 0 && concurrent >= a.cfg.OnConcurrentCalls:
		return "concurrent"
	default:
		return ""
	}
}

// maybeFire launches a capture unless throttled (cooldown, non-manual) or the
// in-flight cap is reached. Automatic triggers respect the cooldown; the manual
// trigger bypasses it. Dropped triggers are logged, never silent.
func (a *iqAutoRecorder) maybeFire(reason, system, protocol string, manual bool) {
	now := a.now()
	a.mu.Lock()
	if !manual && now.Sub(a.lastAuto) < a.cooldown {
		a.mu.Unlock()
		a.log.Debug("iq-autorecord: trigger throttled by cooldown", "reason", reason,
			"since_last", now.Sub(a.lastAuto), "cooldown", a.cooldown)
		return
	}
	if a.inFlight >= autoRecordMaxInFlight {
		a.mu.Unlock()
		a.log.Warn("iq-autorecord: dropping trigger — capture already in flight (raise storage throughput or lengthen cooldown)",
			"reason", reason, "in_flight", autoRecordMaxInFlight)
		return
	}
	if !manual {
		a.lastAuto = now
	}
	a.inFlight++
	ctx := a.baseCtx
	a.mu.Unlock()

	go a.runCapture(ctx, reason, system, protocol, now)
}

// runCapture actuates one capture and writes its metadata sidecar.
func (a *iqAutoRecorder) runCapture(ctx context.Context, reason, system, protocol string, at time.Time) {
	defer func() {
		a.mu.Lock()
		a.inFlight--
		a.mu.Unlock()
	}()

	var centerHz, rateHz uint32
	if a.cfg.TapDDC() {
		// DDC tap: centre + rate come from the control decoder's channelised
		// pipeline (≈144 kHz for TETRA), not the wideband SDR.
		if a.ddc != nil {
			if tap := a.ddc(); tap != nil {
				centerHz = tap.CenterFreqHz()
				rateHz = uint32(tap.PipelineRateHz() + 0.5)
			}
		}
	} else if a.broker != nil {
		centerHz = a.broker.CenterHz()
		rateHz = a.broker.SampleRateHz()
	}
	if system == "" {
		system = a.system
	}
	if protocol == "" {
		protocol = a.protocol
	}
	// Ensure the capture directory exists — a missing dir was the operator's
	// "no such file or directory" capture failure. Cheap and idempotent.
	if err := os.MkdirAll(a.dir, 0o755); err != nil {
		a.log.Warn("iq-autorecord: could not create capture dir", "reason", reason, "dir", a.dir, "err", err)
		return
	}
	path := a.reserveCapturePath(autoRecordFilename(system, reason, at, centerHz, rateHz, a.format))

	a.log.Info("iq-autorecord: capturing", "reason", reason, "path", path,
		"seconds", a.seconds, "center_hz", centerHz, "rate_hz", rateHz)
	samples, drops, err := a.capture(ctx, path, a.format, a.seconds)
	if err != nil {
		a.log.Warn("iq-autorecord: capture failed", "reason", reason, "path", path, "err", err)
		return
	}
	// Metadata sidecar so the file replays / loads into siglab directly.
	meta := &siglab.Metadata{
		Protocol:     protocol,
		Source:       fmt.Sprintf("gophertrunk auto_record (%s)", reason),
		SampleRateHz: float64(rateHz),
		CenterFreqHz: centerHz,
		Format:       a.format.String(),
	}
	metaPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".metadata.json"
	if err := siglab.WriteMetadata(metaPath, meta); err != nil {
		a.log.Warn("iq-autorecord: metadata write failed", "path", metaPath, "err", err)
	}
	a.log.Info("iq-autorecord: captured", "reason", reason, "path", path,
		"samples", samples, "drops", drops)
}

// reserveCapturePath turns a capture name into a collision-free path under
// a.dir. autoRecordFilename is only unique to the second, and autoRecordMaxInFlight
// permits concurrent captures by design, so two triggers in the same second (or
// the same nanosecond) would otherwise build the same name and the second
// os.Create would truncate/race the first. On a name that is already handed out
// this session (a.reserved) or already present on disk (surviving a restart), a
// -2, -3, … counter is inserted before the extension until the path is free; the
// winner is recorded so a second concurrent caller skips past it. Held under a.mu
// so the two runCapture goroutines serialise. Entries are never released: once a
// file is written its name must never be reused, and the disk check would catch
// it anyway; growth is one small string per capture (cooldown-throttled).
func (a *iqAutoRecorder) reserveCapturePath(name string) string {
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	a.mu.Lock()
	defer a.mu.Unlock()
	for n := 1; ; n++ {
		candidate := name
		if n > 1 {
			candidate = fmt.Sprintf("%s-%d%s", stem, n, ext)
		}
		path := filepath.Join(a.dir, candidate)
		if _, taken := a.reserved[path]; taken {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			continue // already on disk (e.g. from a previous run)
		}
		a.reserved[path] = struct{}{}
		return path
	}
}

// autoRecordFilename builds a self-describing capture name:
//
//	<system>_<UTC 20060102T150405Z>_<reason>_<freqHz>_<rateHz>hz.<ext>
//
// The UTC timestamp in the name is the capture start — it makes "which capture
// is which" obvious from the filename alone (the operator's siglab request), and
// mirrors the DDC recorder's naming scheme.
func autoRecordFilename(system, reason string, at time.Time, centerHz, rateHz uint32, format siglab.SampleFormat) string {
	ts := at.UTC().Format("20060102T150405Z")
	name := fmt.Sprintf("%s_%s_%s_%d_%dhz.%s",
		autoRecSanitize(system), ts, autoRecSanitize(reason), centerHz, rateHz, format.String())
	return name
}

// autoRecordTrigger adapts *iqAutoRecorder to the api.AutoRecordProvider
// interface so the manual API trigger can fire a capture.
type autoRecordTrigger struct{ rec *iqAutoRecorder }

func (t autoRecordTrigger) Trigger() {
	if t.rec != nil {
		t.rec.TriggerManual()
	}
}

// primaryControlSystem returns the first configured trunking system's name and
// protocol, used as the fallback labels for manual auto-record captures (which
// carry no grant). Returns empty strings when no system is configured.
func primaryControlSystem(cfg config.Config) (name, protocol string) {
	if len(cfg.Trunking.Systems) == 0 {
		return "", ""
	}
	s := cfg.Trunking.Systems[0]
	return s.Name, s.Protocol
}

// autoRecSanitize keeps a filename component to a safe, path-free set.
func autoRecSanitize(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}
