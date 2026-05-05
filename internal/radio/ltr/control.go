package ltr

import (
	"log/slog"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// LockState is the payload of cc.locked / cc.lost events emitted by
// the LTR per-repeater state machine. LTR has no central control
// channel, so "locked" here means "we're receiving valid status
// words from the repeater on this frequency".
type LockState struct {
	FrequencyHz uint32
	Area        uint8
	Repeater    uint8 // home repeater number from the first valid status word
}

// ControlChannel ingests Status words from a single LTR repeater,
// emits cc.locked the first time a valid status arrives on a
// freshly-tuned device, and republishes any active call as a
// `trunking.Grant` event with `Protocol = "ltr"`.
//
// The state machine is deliberately minimal: each ingested status
// either confirms the lock, signals an active call (group flag set
// + non-zero group ID), or is silently dropped (idle frame, or a
// status word for a different area than we last locked to).
type ControlChannel struct {
	bus        *events.Bus
	log        *slog.Logger
	systemName string
	freqHz     uint32
	resolver   Resolver
	now        func() time.Time
	// expectedArea: when set, ignore status words for any other area
	// (lets a single physical channel host multiple LTR systems).
	expectedArea uint8
	areaSet      bool

	mu     sync.Mutex
	locked bool
	last   LockState
	// activeGroup tracks the group currently announced as active so
	// repeated grant frames during one call don't republish a new
	// grant on every status word.
	activeGroup uint16
}

// Options configure a ControlChannel.
type Options struct {
	Bus         *events.Bus
	Log         *slog.Logger
	SystemName  string
	FrequencyHz uint32
	Resolver    Resolver
	// Area, when non-zero, restricts the state machine to status
	// words whose Area field matches. Zero accepts every area.
	Area uint8
	Now  func() time.Time
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
		bus:          opts.Bus,
		log:          log,
		systemName:   opts.SystemName,
		freqHz:       opts.FrequencyHz,
		resolver:     opts.Resolver,
		now:          now,
		expectedArea: opts.Area,
		areaSet:      opts.Area != 0,
	}
}

// Ingest hands one Status word to the state machine. Real captures
// arrive from an upstream sub-audible 300-baud demod; tests publish
// status words directly.
func (c *ControlChannel) Ingest(s Status) {
	if !s.Sync {
		// Malformed frame; drop.
		return
	}
	if c.areaSet && s.Area != c.expectedArea {
		// A different LTR system is sharing the air; ignore.
		return
	}

	c.maybeLock(LockState{
		FrequencyHz: c.freqHz,
		Area:        s.Area,
		Repeater:    s.Home,
	})

	if !s.IsActive() {
		c.mu.Lock()
		c.activeGroup = 0
		c.mu.Unlock()
		return
	}
	c.mu.Lock()
	if c.activeGroup == s.GroupID {
		c.mu.Unlock()
		return
	}
	c.activeGroup = s.GroupID
	c.mu.Unlock()
	c.publishGrant(s)
}

func (c *ControlChannel) publishGrant(s Status) {
	if c.bus == nil {
		return
	}
	freq := uint32(0)
	if c.resolver != nil {
		if hz, err := c.resolver.Frequency(s.Channel); err == nil {
			freq = hz
		} else {
			c.log.Debug("ltr: band-plan resolution failed",
				"channel", s.Channel, "err", err)
		}
	}
	c.bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System:      c.systemName,
			Protocol:    "ltr",
			GroupID:     uint32(s.GroupID),
			FrequencyHz: freq,
			ChannelNum:  uint16(s.Channel),
			At:          c.now(),
		},
	})
	c.log.Debug("ltr: grant",
		"system", c.systemName, "tg", s.GroupID,
		"channel", s.Channel, "home", s.Home, "freq_hz", freq)
}

func (c *ControlChannel) maybeLock(st LockState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.locked && c.last == st {
		return
	}
	c.locked = true
	c.last = st
	c.bus.Publish(events.Event{Kind: events.KindCCLocked, Payload: st})
	c.log.Info("ltr cc locked",
		"freq", st.FrequencyHz, "area", st.Area, "repeater", st.Repeater,
		"system", c.systemName)
}

// MarkLost publishes cc.lost and resets the locked flag. The trunking
// engine's hunter calls this when status words stop arriving.
func (c *ControlChannel) MarkLost() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.locked {
		return
	}
	c.locked = false
	c.activeGroup = 0
	c.bus.Publish(events.Event{Kind: events.KindCCLost, Payload: c.last})
}
