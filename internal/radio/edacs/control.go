package edacs

import (
	"log/slog"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// LockState is the payload of cc.locked / cc.lost events emitted by
// the EDACS control-channel state machine.
type LockState struct {
	FrequencyHz uint32
	SystemID    uint16
}

// ControlChannel ingests CCWs from a single EDACS control channel,
// emits cc.locked the first time it sees a System-ID announcement on
// a freshly-tuned device, and republishes voice grants as
// events.KindGrant carrying a `trunking.Grant` payload. Mirrors the
// shape of internal/radio/motorola/control.go.
type ControlChannel struct {
	bus        *events.Bus
	log        *slog.Logger
	systemName string
	freqHz     uint32
	resolver   Resolver
	now        func() time.Time

	mu     sync.Mutex
	locked bool
	last   LockState
}

// Options configure a ControlChannel.
type Options struct {
	Bus         *events.Bus
	Log         *slog.Logger
	SystemName  string
	FrequencyHz uint32
	Resolver    Resolver
	Now         func() time.Time
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

// Ingest hands a single decoded CCW to the state machine. Real
// captures arrive via an upstream GMSK demod + FEC; tests publish
// CCWs directly.
func (c *ControlChannel) Ingest(w CCW) {
	if w.IsIdle() {
		return
	}
	if sys, ok := w.AsSystemID(); ok {
		c.maybeLock(LockState{FrequencyHz: c.freqHz, SystemID: sys.ID})
		return
	}
	if grant, ok := w.AsGroupVoiceGrant(); ok {
		c.publishGrant(grant)
		return
	}
}

func (c *ControlChannel) publishGrant(g GroupVoiceGrant) {
	if c.bus == nil {
		return
	}
	freq := uint32(0)
	if c.resolver != nil {
		if hz, err := c.resolver.Frequency(g.LCN); err == nil {
			freq = hz
		} else {
			c.log.Debug("edacs: band-plan resolution failed", "lcn", g.LCN, "err", err)
		}
	}
	c.bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System:      c.systemName,
			Protocol:    "edacs",
			GroupID:     uint32(g.GroupAddress),
			FrequencyHz: freq,
			ChannelNum:  uint16(g.LCN),
			Encrypted:   g.Encrypted,
			Emergency:   g.Emergency,
			ProVoice:    g.ProVoice,
			At:          c.now(),
		},
	})
	c.log.Debug("edacs: grant",
		"system", c.systemName, "tg", g.GroupAddress,
		"lcn", g.LCN, "freq_hz", freq,
		"provoice", g.ProVoice, "enc", g.Encrypted, "emer", g.Emergency)
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
	c.log.Info("edacs cc locked",
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
