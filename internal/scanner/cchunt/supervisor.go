package cchunt

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// Tuner is the subset of sdr.Device the supervisor needs. Decoupled
// from sdr.Device so tests can substitute a fake without spinning up
// a real driver.
type Tuner interface {
	SetCenterFreq(hz uint32) error
}

// IQHealthProvider is the optional hook the supervisor queries when a
// hunt round fails, so the cchunt.failed event/log can explain why
// rather than just reporting the symptom. internal/scanner/ccdecoder's
// Decoder satisfies it. Decoupled via an interface so the supervisor
// doesn't import the decoder (avoids a cycle) and tests can substitute
// a fake or omit it entirely.
type IQHealthProvider interface {
	IQHealth(system string) (trunking.HuntDiagnostics, bool)
}

// CCHealthProvider is the optional capability that reports a locked control
// channel's decode-quality bucket and carrier offset for the live signal-quality
// indicator. The ccdecoder Decoder satisfies it, so it is type-asserted from the
// same handle set via SetIQHealthProvider — no separate wiring needed. Decoupled
// via an interface so the supervisor doesn't import the decoder.
type CCHealthProvider interface {
	DecodeHealth(system string) (quality string, offsetHz int32, ok bool)
	// SignalDbFS reports the last-window mean channel power (dBFS) for a locked,
	// active system — the raw front-end level for antenna/LNA aiming, distinct
	// from the DecodeHealth error-rate bucket. ok is false when the system is not
	// the active locked pipeline or no IQ-power window has been observed yet.
	SignalDbFS(system string) (dbfs float64, ok bool)
}

// noIQDiagnosis is published when the decoder reports no observations
// for a failed system — the strongest single hint that the control SDR
// isn't delivering IQ at all (USB/driver binding, a dead handle, or
// nothing tuned).
const noIQDiagnosis = "no IQ observed from the control SDR during the hunt — confirm the dongle is streaming (`sdr list --probe`), the driver is bound (`sdr doctor`), and the antenna is connected"

// Options configure a new Supervisor.
type Options struct {
	Bus     *events.Bus
	Log     *slog.Logger
	Tuner   Tuner
	Cache   *trunking.Cache
	Systems []trunking.System
	// Dwell is forwarded to each Hunter.
	Dwell time.Duration
	// InitialBackoff is the sleep after the first hunt round that
	// exhausts without a lock. Doubles per consecutive failure, up to
	// MaxBackoff.
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

// Supervisor multiplexes per-system trunking.Hunter runs over one
// shared control SDR. Run blocks until ctx cancels and never returns
// an error other than ctx.Err() — individual hunt failures are
// reported on the bus and via Snapshot.
type Supervisor struct {
	bus    *events.Bus
	log    *slog.Logger
	tuner  Tuner
	cache  *trunking.Cache
	dwell  time.Duration
	initBO time.Duration
	maxBO  time.Duration

	mu     sync.RWMutex
	states map[string]*systemRuntime
	order  []string // stable iteration order for round-robin + Snapshot
	// iqHealth is the optional decoder hook queried on hunt failure to
	// enrich the cchunt.failed event/log. Set once before Run via
	// SetIQHealthProvider; guarded by mu. Nil disables enrichment.
	iqHealth IQHealthProvider
}

// systemRuntime is the per-system mutable state the supervisor owns.
// All access goes through Supervisor.mu.
type systemRuntime struct {
	sys      trunking.System
	state    HuntState
	heldByOp bool
	// Last hunt progress (overwritten on every retune).
	progress trunking.HuntProgress
	// Last successful lock (carried forward across failures so the
	// TUI can still show "was locked on X minutes ago").
	lockedFreqHz uint32
	lockedAt     time.Time
	nac          uint16
	// Failure state.
	lastFailedAt  time.Time
	backoffWindow time.Duration
	// Last grant observed (any grant whose system matches this one).
	lastGrantAt time.Time
	// retuneCh is non-nil while a hunt is in flight; closing it
	// asks the in-flight Hunter to cancel its dwell and return
	// early so the supervisor can re-hunt immediately. Only one
	// retune may be in flight per system at a time.
	retuneCh chan struct{}
}

// New constructs a Supervisor. Returns an error if opts.Bus or
// opts.Tuner are missing. opts.Systems may be empty — the supervisor
// then idles harmlessly.
func New(opts Options) (*Supervisor, error) {
	if opts.Bus == nil {
		return nil, errors.New("cchunt: events.Bus is required")
	}
	if opts.Tuner == nil {
		return nil, errors.New("cchunt: Tuner is required")
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	dwell := opts.Dwell
	if dwell <= 0 {
		dwell = 3 * time.Second
	}
	initBO := opts.InitialBackoff
	if initBO <= 0 {
		initBO = 5 * time.Second
	}
	maxBO := opts.MaxBackoff
	if maxBO <= 0 {
		maxBO = 60 * time.Second
	}
	if maxBO < initBO {
		maxBO = initBO
	}
	s := &Supervisor{
		bus:    opts.Bus,
		log:    log,
		tuner:  opts.Tuner,
		cache:  opts.Cache,
		dwell:  dwell,
		initBO: initBO,
		maxBO:  maxBO,
		states: make(map[string]*systemRuntime, len(opts.Systems)),
	}
	for _, sys := range opts.Systems {
		s.states[sys.Name] = &systemRuntime{
			sys:           sys,
			state:         StateIdle,
			backoffWindow: initBO,
		}
		s.order = append(s.order, sys.Name)
	}
	// Stable round-robin order matches config-file order.
	sort.SliceStable(s.order, func(i, j int) bool { return false })
	return s, nil
}

// Run blocks until ctx cancels. It walks the configured systems in
// round-robin order and hunts each one, sleeping per the per-system
// backoff window on failure. Operator hold short-circuits the round
// (the held system is skipped; the supervisor advances to the next).
func (s *Supervisor) Run(ctx context.Context) error {
	// Subscribe once for the whole supervisor lifetime so we can
	// observe cc.locked / cc.lost / grant events across all systems.
	sub := s.bus.Subscribe()
	defer sub.Close()

	// A separate goroutine drains the subscription into our state
	// machine so the main hunt loop doesn't have to multiplex
	// between hunting and listening on the same channel.
	go s.listen(ctx, sub)

	if len(s.order) == 0 {
		<-ctx.Done()
		return ctx.Err()
	}

	cursor := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		name := s.order[cursor%len(s.order)]
		cursor++

		rt := s.runtime(name)
		if rt == nil {
			continue
		}

		// Skip held systems — they own their current state until the
		// operator resumes them.
		if s.isHeld(name) {
			s.waitOrSleep(ctx, 500*time.Millisecond)
			continue
		}
		// Skip systems still in backoff.
		if remaining := s.backoffRemaining(name); remaining > 0 {
			s.waitOrSleep(ctx, minDur(remaining, 500*time.Millisecond))
			continue
		}
		// Park an already-locked system instead of re-hunting it. A cc.locked
		// edge (recordLock → StateLocked) can land AFTER a hunt round already
		// failed and backed off — e.g. a TETRA control channel whose cold
		// acquisition (BSCH sync + colour code) took longer than the dwell.
		// Without this the loop re-hunts a system it already knows is locked,
		// and because an edge-triggered decoder (TETRA) never re-emits cc.locked
		// on the same locked pipeline, every such re-hunt exhausts the dwell and
		// reports "candidates exhausted" while the decoder keeps decoding calls.
		// parkUntilUnlocked returns as soon as the state leaves StateLocked,
		// which the protocol's lock-loss watchdog (cc.lost) drives on real loss.
		if s.isLocked(name) {
			s.parkUntilUnlocked(ctx, name)
			continue
		}

		s.startRound(name)
		hunter, err := trunking.NewHunter(trunking.HunterOptions{
			System: rt.sys,
			Tuner:  s.currentTuner(),
			Bus:    s.bus,
			Cache:  s.cache,
			Log:    s.log,
			Dwell:  s.dwell,
		})
		if err != nil {
			s.log.Warn("cchunt: hunter construct failed", "system", name, "err", err)
			s.markFailed(name)
			continue
		}

		// Cancel the hunt if the operator forces a retune or holds
		// us; the supervisor advances on either signal.
		hctx, hcancel := context.WithCancel(ctx)
		s.armRetune(name, hcancel)
		_, herr := hunter.Hunt(hctx)
		s.disarmRetune(name)
		hcancel()

		if herr == nil {
			// Lock succeeded — the listen() goroutine will turn the
			// matching cc.locked event into a StateLocked transition.
			s.parkUntilUnlocked(ctx, name)
			continue
		}
		if errors.Is(herr, context.Canceled) || errors.Is(herr, context.DeadlineExceeded) {
			// Either the operator forced a retune (we want to loop
			// immediately) or the daemon is shutting down.
			if err := ctx.Err(); err != nil {
				return err
			}
			continue
		}
		// A conventional / direct-mode system (CampsWhenIdle) has no
		// continuous control channel, so a hunt round that ends without a
		// lock means the channel is idle — not a failure. Camp on it and
		// re-dwell promptly instead of the WARN + KindHuntFailed +
		// exponential backoff a trunked no-lock triggers. Issue #1036.
		if rt.sys.Protocol.CampsWhenIdle() {
			s.markCamped(name)
			continue
		}
		s.markFailed(name)
	}
}

// listen drains events from the bus and updates per-system state.
func (s *Supervisor) listen(ctx context.Context, sub *events.Subscription) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-sub.C:
			if !ok {
				return
			}
			switch ev.Kind {
			case events.KindCCLocked:
				lp, ok := ev.Payload.(trunking.LockedPayload)
				if !ok {
					continue
				}
				s.recordLock(lp.LockedFrequencyHz(), lp.LockedNAC())
			case events.KindCCLost:
				lp, _ := ev.Payload.(trunking.LockedPayload)
				freq := uint32(0)
				if lp != nil {
					freq = lp.LockedFrequencyHz()
				}
				s.recordLost(freq)
			case events.KindGrant:
				g, ok := ev.Payload.(trunking.Grant)
				if !ok {
					continue
				}
				s.recordGrant(g.System, ev.Timestamp)
			case events.KindHuntProgress:
				p, ok := ev.Payload.(trunking.HuntProgress)
				if !ok {
					continue
				}
				s.recordProgress(p)
			}
		}
	}
}

// --- runtime accessors (lock-guarded) ---

func (s *Supervisor) runtime(name string) *systemRuntime {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.states[name]
}

func (s *Supervisor) startRound(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rt := s.states[name]
	if rt == nil {
		return
	}
	// A camped conventional / direct-mode system stays camped across its
	// re-dwells: sitting on the frequency waiting for the next transmission IS
	// the camp, not a fresh acquisition attempt. Overwriting it here left
	// StateCamped set only for the few microseconds between markCamped
	// returning and the next round starting, against a whole dwell of
	// StateHunting — about a 0.1% duty cycle. Three things fell out of that:
	// the TUI's "camped" badge (panels/scanner.go) never rendered, markCamped's
	// transitioned guard logged every round instead of once (~25 INFO lines a
	// second on a quiet channel), and TestSupervisorCampsIdleConventionalDMR
	// was left racing a transient it usually lost. Issue #1036.
	if rt.state == StateCamped {
		return
	}
	rt.state = StateHunting
}

// SetIQHealthProvider wires an optional IQ-health source the supervisor
// queries on hunt failure to enrich the cchunt.failed event/log. Call
// before Run (the daemon does, right after building the ccdecoder).
// Safe to call with nil to clear it.
func (s *Supervisor) SetIQHealthProvider(p IQHealthProvider) {
	s.mu.Lock()
	s.iqHealth = p
	s.mu.Unlock()
}

// markCamped records that a conventional / direct-mode system's hunt
// round ended without a lock. Unlike markFailed this is not an error
// condition — the channel is simply idle — so it keeps the backoff
// window at its floor (the next round re-dwells promptly, camping on the
// frequency to catch the next transmission) and publishes no
// KindHuntFailed event. A single Info log fires only on the transition
// into camped, so a quiet channel doesn't spam the log every round.
// Issue #1036.
func (s *Supervisor) markCamped(name string) {
	s.mu.Lock()
	rt := s.states[name]
	if rt == nil {
		s.mu.Unlock()
		return
	}
	transitioned := rt.state != StateCamped
	rt.state = StateCamped
	rt.backoffWindow = s.initBO // never grow — camp, don't back off
	s.mu.Unlock()

	if transitioned {
		s.log.Info("cchunt: camped on conventional channel — idle, waiting for traffic",
			"system", name)
	}
}

func (s *Supervisor) markFailed(name string) {
	s.mu.Lock()
	rt := s.states[name]
	if rt == nil {
		s.mu.Unlock()
		return
	}
	now := time.Now()
	rt.state = StateFailed
	rt.lastFailedAt = now
	if rt.backoffWindow <= 0 {
		rt.backoffWindow = s.initBO
	}
	wait := rt.backoffWindow
	// Schedule next double.
	rt.backoffWindow *= 2
	if rt.backoffWindow > s.maxBO {
		rt.backoffWindow = s.maxBO
	}
	provider := s.iqHealth
	s.mu.Unlock()

	// Capture the control SDR's IQ health *outside* s.mu — the provider
	// (the ccdecoder) takes its own lock, and we must not hold ours
	// across a foreign call.
	diag := diagnoseFailure(provider, name)

	// A WARN here is the channel operators actually paste in bug
	// reports; the structured fields + one-line diagnosis turn a
	// "constantly getting cchunt.failed" report into a self-triaging
	// log entry. Fires once per failed hunt round, then backs off.
	s.log.Warn("cchunt: hunt failed — no control-channel lock",
		"system", name,
		"backoff_ms", int(wait/time.Millisecond),
		"iq_observed", diag.IQObserved,
		"iq_samples", diag.IQSamples,
		"iq_power_dbfs", diag.IQPowerDbFS,
		"iq_clip_ratio", diag.IQClipRatio,
		"iq_dc_ratio_db", diag.IQDCRatioDb,
		"pipeline_active", diag.PipelineActive,
		"diagnosis", diag.Diagnosis)

	s.bus.Publish(events.Event{
		Kind: events.KindHuntFailed,
		Payload: trunking.HuntFailed{
			System:      name,
			At:          now,
			BackoffMs:   int(wait / time.Millisecond),
			Diagnostics: diag,
		},
	})
}

// diagnoseFailure queries the IQ-health provider (if wired) for a
// failed system. When the provider has no observations for it, that
// absence is itself the diagnosis (noIQDiagnosis). With no provider at
// all (unit tests) the zero value is returned and Diagnosis stays empty.
func diagnoseFailure(p IQHealthProvider, name string) trunking.HuntDiagnostics {
	if p == nil {
		return trunking.HuntDiagnostics{}
	}
	if d, ok := p.IQHealth(name); ok {
		return d
	}
	return trunking.HuntDiagnostics{Diagnosis: noIQDiagnosis}
}

func (s *Supervisor) backoffRemaining(name string) time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rt := s.states[name]
	if rt == nil || rt.state != StateFailed {
		return 0
	}
	// Next attempt is allowed after lastFailedAt + the *previously*
	// recorded wait, which we stashed in backoffWindow before
	// doubling. To keep the math simple we don't track it
	// separately; instead use the current window (already doubled)
	// halved — works for round 2+; round 1 uses initBO.
	wait := rt.backoffWindow / 2
	if wait < s.initBO {
		wait = s.initBO
	}
	elapsed := time.Since(rt.lastFailedAt)
	if elapsed >= wait {
		return 0
	}
	return wait - elapsed
}

func (s *Supervisor) recordLock(freqHz uint32, nac uint16) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rt := range s.states {
		for _, cc := range rt.sys.ControlChannels {
			if cc == freqHz {
				rt.state = StateLocked
				rt.lockedFreqHz = freqHz
				rt.lockedAt = time.Now()
				rt.nac = nac
				rt.backoffWindow = s.initBO // reset on success
				return
			}
		}
	}
}

func (s *Supervisor) recordLost(freqHz uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rt := range s.states {
		if rt.lockedFreqHz == freqHz || (freqHz == 0 && rt.state == StateLocked) {
			rt.state = StateHunting
			rt.lockedFreqHz = 0
		}
	}
}

func (s *Supervisor) recordProgress(p trunking.HuntProgress) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rt := s.states[p.System]; rt != nil {
		rt.progress = p
	}
}

func (s *Supervisor) recordGrant(system string, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rt := s.states[system]; rt != nil {
		rt.lastGrantAt = at
	}
}

// currentTuner returns the supervisor's active control tuner under
// s.mu. Read on every hunt round so SwapTuner can replace the handle
// in-place after a USB reacquisition without racing the hunt loop.
func (s *Supervisor) currentTuner() Tuner {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tuner
}

// SwapTuner replaces the supervisor's control tuner. Newly-spawned
// hunters use the fresh handle; in-flight hunters keep the previous
// reference until they finish. Any armed retune channels are closed
// so the loop drops out of its current dwell and re-hunts immediately
// with the new tuner — the closed channel triggers the per-system
// hctx cancel via the goroutine armRetune spawned. Returns an error
// only when t is nil.
//
// Used by the daemon's USB reacquisition path so a freshly re-opened
// SDR replaces a disconnected one without a process restart. See
// issue #345.
func (s *Supervisor) SwapTuner(t Tuner) error {
	if t == nil {
		return errors.New("cchunt: SwapTuner requires a non-nil Tuner")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tuner = t
	for _, rt := range s.states {
		if rt == nil || rt.retuneCh == nil {
			continue
		}
		select {
		case <-rt.retuneCh:
		default:
			close(rt.retuneCh)
		}
		rt.retuneCh = nil
	}
	s.log.Info("cchunt: control tuner swapped (reacquired)")
	return nil
}

func (s *Supervisor) armRetune(name string, cancel context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rt := s.states[name]; rt != nil {
		ch := make(chan struct{})
		rt.retuneCh = ch
		go func() {
			<-ch
			cancel()
		}()
	}
}

func (s *Supervisor) disarmRetune(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rt := s.states[name]; rt != nil && rt.retuneCh != nil {
		// Closing here is benign if the consumer already fired the
		// cancel; the goroutine guards against double-close because
		// we replace retuneCh with nil after this.
		select {
		case <-rt.retuneCh:
		default:
			close(rt.retuneCh)
		}
		rt.retuneCh = nil
	}
}

func (s *Supervisor) parkUntilUnlocked(ctx context.Context, name string) {
	// Wait until the system transitions out of StateLocked (the
	// listen() goroutine flips it on cc.lost) OR until ctx cancels.
	t := time.NewTicker(250 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.mu.RLock()
			rt := s.states[name]
			locked := rt != nil && rt.state == StateLocked && !rt.heldByOp
			s.mu.RUnlock()
			if !locked {
				return
			}
		}
	}
}

func (s *Supervisor) waitOrSleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

func (s *Supervisor) isHeld(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rt := s.states[name]
	return rt != nil && rt.heldByOp
}

// isLocked reports whether the system is already in StateLocked (a cc.locked
// edge was observed), so the Run loop parks it instead of re-hunting.
func (s *Supervisor) isLocked(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rt := s.states[name]
	return rt != nil && rt.state == StateLocked && !rt.heldByOp
}

// --- operator mutation surface ---

// PauseAll holds every configured system and bails any in-flight hunt, so the
// supervisor stops retuning the control SDR. It is used by the live hunt's
// "borrow the control SDR" path (when no spare SDR is available) to quiesce
// cchunt before sweeping, then ResumeAll restores normal scanning. Idempotent.
func (s *Supervisor) PauseAll() {
	s.mu.Lock()
	var toCancel []chan struct{}
	for _, rt := range s.states {
		if rt == nil {
			continue
		}
		rt.heldByOp = true
		rt.state = StateHeld
		if rt.retuneCh != nil {
			toCancel = append(toCancel, rt.retuneCh)
			rt.retuneCh = nil
		}
	}
	s.mu.Unlock()
	// Cancel in-flight hunts outside the lock so they bail promptly; the next
	// Run round then skips every (held) system and stops touching the tuner.
	for _, ch := range toCancel {
		select {
		case <-ch:
		default:
			close(ch)
		}
	}
}

// ResumeAll undoes PauseAll, returning every system to the round-robin with a
// fresh backoff. Idempotent.
func (s *Supervisor) ResumeAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rt := range s.states {
		if rt == nil {
			continue
		}
		rt.heldByOp = false
		rt.state = StateIdle
		rt.backoffWindow = s.initBO
	}
}

// Hold pins the supervisor on the named system's current state — no
// further retunes happen until Resume. Returns false if the system
// isn't configured.
func (s *Supervisor) Hold(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	rt := s.states[name]
	if rt == nil {
		return false
	}
	rt.heldByOp = true
	// Remember the held variant of whatever state we were in so the
	// Snapshot still surfaces "locked + held" or "failed + held"
	// rather than a generic "held" with no context.
	rt.state = StateHeld
	return true
}

// Resume undoes Hold. The next Run iteration picks the system back up
// in the next round-robin slot.
func (s *Supervisor) Resume(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	rt := s.states[name]
	if rt == nil {
		return false
	}
	rt.heldByOp = false
	rt.state = StateIdle
	rt.backoffWindow = s.initBO
	return true
}

// ForceRetune asks the in-flight Hunter (if any) for the named system
// to bail out so the next round picks it up immediately. If no hunt
// is in flight (e.g. the system is in backoff) it clears the backoff
// so the next round restarts the hunt.
func (s *Supervisor) ForceRetune(name string) bool {
	s.mu.Lock()
	rt := s.states[name]
	if rt == nil {
		s.mu.Unlock()
		return false
	}
	rt.lastFailedAt = time.Time{} // clear backoff
	rt.backoffWindow = s.initBO
	rt.state = StateIdle
	ch := rt.retuneCh
	rt.retuneCh = nil
	s.mu.Unlock()
	if ch != nil {
		select {
		case <-ch:
		default:
			close(ch)
		}
	}
	return true
}

// Snapshot returns a copy of every system's current status. Safe to
// call concurrently with Run.
func (s *Supervisor) Snapshot() []SystemStatus {
	s.mu.RLock()
	out := make([]SystemStatus, 0, len(s.order))
	for _, name := range s.order {
		rt := s.states[name]
		if rt == nil {
			continue
		}
		st := SystemStatus{
			Name:            rt.sys.Name,
			Protocol:        rt.sys.Protocol.String(),
			State:           rt.state,
			AttemptedFreqHz: rt.progress.AttemptedFreqHz,
			AttemptIndex:    rt.progress.AttemptIndex,
			TotalCandidates: rt.progress.TotalCandidates,
			LockedFreqHz:    rt.lockedFreqHz,
			LockedAt:        rt.lockedAt,
			NAC:             rt.nac,
			LastFailedAt:    rt.lastFailedAt,
			LastGrantAt:     rt.lastGrantAt,
		}
		if rt.state == StateFailed {
			// Surface the wait window so the TUI can render
			// "retry in 5 s".
			rem := s.backoffRemainingLocked(rt)
			st.BackoffMs = int(rem / time.Millisecond)
		}
		out = append(out, st)
	}
	provider := s.iqHealth
	s.mu.RUnlock()

	// Enrich with live control-channel decode health, queried OUTSIDE s.mu — the
	// decoder takes its own locks, the same rule markFailed follows. DecodeHealth
	// self-gates on the active/locked system, so a hunting or non-active system
	// simply reports nothing.
	if h, ok := provider.(CCHealthProvider); ok {
		for i := range out {
			if q, off, ok := h.DecodeHealth(out[i].Name); ok {
				out[i].DecodeQuality = q
				out[i].CarrierOffsetHz = off
				out[i].HasDecodeHealth = true
			}
			// Signal level is reported independently of the decode-quality gate:
			// a freshly-locked system has a dBFS reading before enough frames
			// have decoded to bucket its error rate.
			if dbfs, ok := h.SignalDbFS(out[i].Name); ok {
				out[i].SignalDbFS = dbfs
				out[i].HasSignal = true
			}
		}
	}
	return out
}

func (s *Supervisor) backoffRemainingLocked(rt *systemRuntime) time.Duration {
	if rt.state != StateFailed {
		return 0
	}
	wait := rt.backoffWindow / 2
	if wait < s.initBO {
		wait = s.initBO
	}
	elapsed := time.Since(rt.lastFailedAt)
	if elapsed >= wait {
		return 0
	}
	return wait - elapsed
}

func minDur(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
