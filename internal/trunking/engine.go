package trunking

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// Engine is the central trunking state machine. It subscribes to
// events.KindGrant, looks up the talkgroup, dispatches to the voice pool
// (preempting lower-priority active calls when necessary), and emits
// events.KindCallStart / events.KindCallEnd.
//
// The engine deliberately knows nothing about the demod pipeline — it
// just tunes Voice devices and publishes structured events. Downstream
// consumers (the voice composer + recorder, the SQLite call log)
// subscribe to the CallStart / CallEnd events to do their work.
type Engine struct {
	bus        *events.Bus
	log        *slog.Logger
	pool       *VoicePool
	talkgroups *TalkgroupDB
	timeout    time.Duration
	now        func() time.Time
	sub        *events.Subscription
	closeOnce  sync.Once

	mu    sync.Mutex
	calls map[string]*ActiveCall // by device serial; mirror of pool.active for fast access
}

// EngineOptions configure a new Engine.
type EngineOptions struct {
	Bus        *events.Bus
	Log        *slog.Logger
	VoicePool  *VoicePool
	Talkgroups *TalkgroupDB
	// CallTimeout is how long a call can run without a Touch before the
	// watchdog ends it as EndReasonTimeout. Default 30 s.
	CallTimeout time.Duration
	// Now is injectable for tests; defaults to time.Now.
	Now func() time.Time
}

// NewEngine validates opts and returns a ready-to-Run engine.
func NewEngine(opts EngineOptions) (*Engine, error) {
	if opts.Bus == nil {
		return nil, errors.New("trunking/engine: events.Bus is required")
	}
	if opts.VoicePool == nil {
		return nil, errors.New("trunking/engine: VoicePool is required")
	}
	if opts.Talkgroups == nil {
		opts.Talkgroups = NewTalkgroupDB()
	}
	if opts.Log == nil {
		opts.Log = slog.Default()
	}
	if opts.CallTimeout <= 0 {
		opts.CallTimeout = 30 * time.Second
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	e := &Engine{
		bus:        opts.Bus,
		log:        opts.Log,
		pool:       opts.VoicePool,
		talkgroups: opts.Talkgroups,
		timeout:    opts.CallTimeout,
		now:        opts.Now,
		calls:      make(map[string]*ActiveCall),
	}
	// Subscribe at construction time so callers can publish grants
	// before Run starts without losing them.
	e.sub = opts.Bus.Subscribe()
	return e, nil
}

// Close releases the engine's subscription. Safe to call concurrently
// with Run; idempotent on repeat calls. Subscription.Close is itself
// idempotent so we don't need to nil the field — that nil-write was
// previously a race with Run's read of e.sub.C.
func (e *Engine) Close() {
	e.closeOnce.Do(func() {
		e.sub.Close()
	})
}

// Run drains grant events from the bus and runs the watchdog until ctx
// cancels. Returns ctx.Err(). Run does NOT close the engine's
// subscription; call Close when you're done with the engine.
func (e *Engine) Run(ctx context.Context) error {
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			e.shutdown()
			return ctx.Err()
		case ev, ok := <-e.sub.C:
			if !ok {
				return nil
			}
			if ev.Kind != events.KindGrant {
				continue
			}
			g, ok := ev.Payload.(Grant)
			if !ok {
				continue
			}
			e.HandleGrant(g)
		case <-tick.C:
			e.runWatchdog()
		}
	}
}

// HandleGrant is the engine's grant-dispatch entrypoint. It is exported
// so tests can drive it directly without a running event loop.
func (e *Engine) HandleGrant(g Grant) {
	if g.At.IsZero() {
		g.At = e.now()
	}
	if g.FrequencyHz == 0 {
		e.log.Warn("dropping grant with zero frequency", "grant", g.String())
		return
	}
	tg := e.talkgroups.Lookup(g.GroupID)
	if tg != nil && tg.Lockout && !g.Emergency {
		e.log.Info("grant locked out", "grant", g.String(), "tg", tg.AlphaTag)
		return
	}

	// 1) Free device available? Allocate.
	if free := e.pool.FindFree(); free != nil {
		e.startCall(free, g, tg)
		return
	}
	// 2) All busy. Look at the lowest-priority active call.
	victim := e.pool.LowestPriorityActive()
	if victim == nil {
		// Shouldn't happen: pool was full but has no actives. Drop.
		e.log.Warn("voice pool full but no actives", "grant", g.String())
		return
	}
	if !CanPreempt(victim.Grant, victim.Talkgroup, g, tg) {
		e.log.Info("no voice device available for grant", "grant", g.String())
		return
	}
	// 3) Preempt: end victim, allocate freed device.
	e.endCall(victim, EndReasonPreempted)
	e.startCall(victim.Device, g, tg)
}

// EndCall is the explicit external signal that a call has ended (e.g.
// the protocol decoder saw a channel-release announcement, or an upstream
// test wants to release the device). reason is published in the CallEnd
// event payload.
func (e *Engine) EndCall(deviceSerial string, reason EndReason) bool {
	e.mu.Lock()
	ac, ok := e.calls[deviceSerial]
	e.mu.Unlock()
	if !ok {
		return false
	}
	e.endCall(ac, reason)
	return true
}

// Touch refreshes the LastHeardAt timestamp on the active call bound to
// deviceSerial. The protocol decoder calls this every time it sees voice
// activity on the followed frequency so the watchdog doesn't time it out.
func (e *Engine) Touch(deviceSerial string) {
	e.pool.Touch(deviceSerial, e.now())
}

// ActiveCalls returns a snapshot of every active call.
func (e *Engine) ActiveCalls() []*ActiveCall { return e.pool.Active() }

func (e *Engine) startCall(d *VoiceDevice, g Grant, tg *TalkGroup) {
	ac, err := e.pool.Bind(d, g, tg, e.now())
	if err != nil {
		e.log.Warn("voice bind failed", "err", err, "grant", g.String())
		return
	}
	e.mu.Lock()
	e.calls[d.Serial] = ac
	e.mu.Unlock()
	e.bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: CallStart{
			Grant:        g,
			Talkgroup:    tg,
			DeviceSerial: d.Serial,
			StartedAt:    ac.StartedAt,
		},
	})
	e.log.Info("call started",
		"device", d.Serial,
		"grant", g.String(),
		"priority", EffectivePriority(g, tg))
}

func (e *Engine) endCall(ac *ActiveCall, reason EndReason) {
	released := e.pool.Release(ac.Device.Serial)
	if released == nil {
		return // already released elsewhere
	}
	e.mu.Lock()
	delete(e.calls, ac.Device.Serial)
	e.mu.Unlock()
	e.bus.Publish(events.Event{
		Kind: events.KindCallEnd,
		Payload: CallEnd{
			Grant:        released.Grant,
			Talkgroup:    released.Talkgroup,
			DeviceSerial: ac.Device.Serial,
			StartedAt:    released.StartedAt,
			EndedAt:      e.now(),
			Reason:       reason,
		},
	})
	e.log.Info("call ended",
		"device", ac.Device.Serial,
		"grant", released.Grant.String(),
		"reason", reason.String())
}

func (e *Engine) runWatchdog() {
	now := e.now()
	cutoff := now.Add(-e.timeout)
	for _, ac := range e.pool.Active() {
		if ac.LastHeardAt.Before(cutoff) {
			e.endCall(ac, EndReasonTimeout)
		}
	}
}

func (e *Engine) shutdown() {
	for _, ac := range e.pool.Active() {
		e.endCall(ac, EndReasonNormal)
	}
}
