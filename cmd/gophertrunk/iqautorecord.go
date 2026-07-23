package main

import (
	"context"
	"fmt"
	"log/slog"
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

	// capture actuates a capture; defaults to captureIQToFile over broker.
	capture captureFunc
	// now is the clock (injectable for tests).
	now func() time.Time

	mu       sync.Mutex
	baseCtx  context.Context
	seen     map[string]time.Time // observed-call key -> last grant time
	lastAuto time.Time            // last automatic (non-manual) trigger, for cooldown
	inFlight int
}

// newIQAutoRecorder builds an auto-recorder for the control SDR's broker.
// Returns nil when disabled, so the daemon can skip wiring it entirely. cfg is
// assumed already validated (config.Validate); format/cooldown fall back to
// their documented defaults.
func newIQAutoRecorder(cfg config.BasebandAutoRecordConfig, system, protocol, serial string, broker *iqtap.Broker, log *slog.Logger) *iqAutoRecorder {
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
		log:      log.With("component", "iq-autorecord", "serial", serial),
		now:      time.Now,
		baseCtx:  context.Background(),
		seen:     make(map[string]time.Time),
	}
	a.capture = a.defaultCapture
	return a
}

// autoRecordDefaultCooldownFallback mirrors config.autoRecordDefaultCooldown
// for the (validated-away) error path in the constructor.
const autoRecordDefaultCooldownFallback = 10 * time.Second

// defaultCapture writes the capture through the shared captureIQToFile helper.
func (a *iqAutoRecorder) defaultCapture(ctx context.Context, path string, format siglab.SampleFormat, seconds int) (int64, uint64, error) {
	if a.broker == nil {
		return 0, 0, fmt.Errorf("iq-autorecord: no broker")
	}
	samples, _, drops, err := captureIQToFile(ctx, a.broker, path, format, seconds)
	return samples, drops, err
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
		"on_encrypted", a.cfg.OnEncrypted, "on_emergency", a.cfg.OnEmergency)
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
	if a.broker != nil {
		centerHz = a.broker.CenterHz()
		rateHz = a.broker.SampleRateHz()
	}
	if system == "" {
		system = a.system
	}
	if protocol == "" {
		protocol = a.protocol
	}
	path := filepath.Join(a.dir, autoRecordFilename(system, reason, at, centerHz, rateHz, a.format))

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
