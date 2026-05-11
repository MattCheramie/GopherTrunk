package motorola

import (
	"log/slog"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// LockState is the payload of cc.locked / cc.lost events emitted by
// the Motorola control-channel state machine.
type LockState struct {
	FrequencyHz uint32
	SystemID    uint16
}

// ControlChannel ingests OSWs from a single SmartZone control channel,
// emits cc.locked the first time it sees a System-ID announcement on
// a freshly-tuned device, and republishes voice grants as
// events.KindGrant carrying a `trunking.Grant` payload.
//
// The state machine is intentionally minimal: it watches for system
// identification (so a stale device → frequency mapping doesn't fire
// false locks) and for the two voice-grant opcodes. Adjacent-site
// reactions, neighbour-list maintenance, and roaming live in the
// engine layer once the higher-level state machine wants them.
type ControlChannel struct {
	bus        *events.Bus
	log        *slog.Logger
	systemName string
	freqHz     uint32
	resolver   Resolver
	now        func() time.Time

	// proc is the cross-call bit / sync state the Process adapter
	// uses (see process.go). Lazily constructed on the first
	// Process call.
	proc *processState

	mu     sync.Mutex
	locked bool
	last   LockState
}

// Options configure a ControlChannel.
type Options struct {
	Bus        *events.Bus
	Log        *slog.Logger
	SystemName string
	// FrequencyHz is the control-channel frequency this state machine
	// is bound to. Carried in cc.locked / cc.lost payloads.
	FrequencyHz uint32
	// Resolver maps grant LCNs to voice-channel frequencies. Required
	// when grant events should carry an actual Hz; optional otherwise.
	Resolver Resolver
	Now      func() time.Time
}

// New constructs a ControlChannel.
func New(opts Options) *ControlChannel {
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &ControlChannel{
		bus:        opts.Bus,
		log:        log,
		systemName: opts.SystemName,
		freqHz:     opts.FrequencyHz,
		resolver:   opts.Resolver,
		now:        now,
	}
}

// Ingest hands a single decoded OSW to the state machine. Real
// captures arrive via an upstream MSK demod + BCH decoder; tests
// publish OSWs directly.
func (c *ControlChannel) Ingest(o OSW) {
	if o.IsIdle() {
		return
	}

	if sys, ok := o.AsSystemID(); ok {
		c.maybeLock(LockState{FrequencyHz: c.freqHz, SystemID: sys.ID})
		return
	}
	if grant, ok := o.AsGroupVoiceChannelGrant(); ok {
		c.publishGrant(grant)
		return
	}
}

func (c *ControlChannel) publishGrant(g GroupVoiceChannelGrant) {
	if c.bus == nil {
		return
	}
	freq := uint32(0)
	if c.resolver != nil {
		if hz, err := c.resolver.Frequency(g.LCN); err == nil {
			freq = hz
		} else {
			c.log.Debug("motorola: band-plan resolution failed", "lcn", g.LCN, "err", err)
		}
	}
	c.bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System:        c.systemName,
			Protocol:      "motorola",
			GroupID:       uint32(g.GroupAddress),
			FrequencyHz:   freq,
			ChannelID:     0, // Motorola LCNs carry no separate band-ID; we
			// represent the LCN in ChannelNumber so downstream
			// consumers that don't yet have a band plan still see
			// something useful.
			ChannelNum: g.LCN,
			At:         c.now(),
		},
	})
	c.log.Debug("motorola: grant",
		"system", c.systemName, "tg", g.GroupAddress,
		"lcn", g.LCN, "freq_hz", freq)
}

func (c *ControlChannel) maybeLock(s LockState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.locked && c.last == s {
		return
	}
	c.locked = true
	c.last = s
	c.bus.Publish(events.Event{Kind: events.KindCCLocked, Payload: s})
	c.log.Info("motorola cc locked",
		"freq", s.FrequencyHz, "sys", s.SystemID, "system", c.systemName)
}

// MarkLost publishes cc.lost and resets the locked flag. The trunking
// engine's hunter calls this when the control channel goes silent.
func (c *ControlChannel) MarkLost() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.locked {
		return
	}
	c.locked = false
	c.bus.Publish(events.Event{Kind: events.KindCCLost, Payload: c.last})
}
