package dpmr

import (
	"log/slog"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// LockState is the payload of cc.locked / cc.lost events emitted by
// the dPMR Mode 3 control-channel state machine.
type LockState struct {
	FrequencyHz uint32
	SystemID    uint32 // first StandingServiceStatus' DestID, when seen
}

// LockedFrequencyHz / LockedNAC make LockState satisfy
// trunking.LockedPayload so the cchunt supervisor's state machine
// recognises dPMR lock events alongside the protocol-neutral P25 /
// DMR / NXDN / TETRA payloads. dPMR doesn't have a P25-style NAC;
// the low 16 bits of SystemID are the closest per-cell identifier
// and get plumbed into the NAC slot. Without these methods, the
// supervisor's type-assertion on cc.locked silently drops the event
// and /api/v1/scanner never surfaces state=locked.
func (s LockState) LockedFrequencyHz() uint32 { return s.FrequencyHz }
func (s LockState) LockedNAC() uint16         { return uint16(s.SystemID) }

// ControlChannel ingests CSBKs from a single dPMR Mode 3 control
// channel, emits cc.locked the first time a valid StandingServiceStatus
// (or any non-idle CSBK) arrives on a freshly-tuned device, and
// republishes voice grants as events.KindGrant carrying a
// `trunking.Grant` payload with `Protocol = "dpmr"`. Same shape as the
// Motorola / EDACS / LTR / MPT 1327 / P25 Phase 2 control channels.
type ControlChannel struct {
	bus        *events.Bus
	log        *slog.Logger
	systemName string
	freqHz     uint32
	resolver   Resolver
	now        func() time.Time

	// proc is the cross-call dibit / sync state the Process
	// adapter uses (see process.go). Lazily constructed on the
	// first Process call so tests that drive Ingest directly
	// don't pay the construction cost.
	proc *processState

	mu               sync.Mutex
	locked           bool
	last             LockState
	strictValidation bool
}

// SetStrictValidation toggles the strict frame-validity filter on the
// Ingest path. When enabled, CSBKs whose 5-bit MessageType is not in
// the documented ETSI TS 102 658 §6.5.2 set are silently dropped at
// Ingest time. The Process adapter already filters at the framing
// layer; strict-mode tightens it further so PDUs from a
// misaligned-but-passing window still drop out.
func (c *ControlChannel) SetStrictValidation(strict bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.strictValidation = strict
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

// Ingest hands a single decoded CSBK to the state machine. Real
// captures arrive via an upstream 4FSK demod + FEC; tests publish
// CSBKs directly.
func (c *ControlChannel) Ingest(b CSBK) {
	c.mu.Lock()
	strict := c.strictValidation
	c.mu.Unlock()
	if strict && !b.Type.IsKnown() {
		return
	}
	if b.IsIdle() {
		return
	}
	if sb, ok := b.AsSiteBroadcast(); ok {
		c.maybeLock(LockState{FrequencyHz: c.freqHz, SystemID: sb.SystemID})
		return
	}
	if g, ok := b.AsVoiceGrant(); ok {
		// Even if we haven't seen a SiteBroadcast yet, a voice grant on
		// the CC is enough to declare the channel locked.
		c.maybeLock(LockState{FrequencyHz: c.freqHz})
		c.publishGrant(g)
	}
}

func (c *ControlChannel) publishGrant(g VoiceGrant) {
	if c.bus == nil {
		return
	}
	freq := uint32(0)
	if c.resolver != nil {
		if hz, err := c.resolver.Frequency(g.Channel); err == nil {
			freq = hz
		} else {
			c.log.Debug("dpmr: band-plan resolution failed",
				"channel", g.Channel, "err", err)
		}
	}
	c.bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System:      c.systemName,
			Protocol:    "dpmr",
			GroupID:     g.DestID,
			SourceID:    g.SourceID,
			FrequencyHz: freq,
			ChannelNum:  g.Channel,
			Encrypted:   g.Encrypted,
			Emergency:   g.Emergency,
			At:          c.now(),
		},
	})
	c.log.Debug("dpmr: grant",
		"system", c.systemName,
		"src", g.SourceID, "dst", g.DestID,
		"channel", g.Channel, "freq_hz", freq,
		"group", g.Group, "enc", g.Encrypted, "emer", g.Emergency)
}

func (c *ControlChannel) maybeLock(s LockState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.locked && c.last == s {
		return
	}
	// Preserve a previously-learned SystemID if the new state has none.
	if c.locked && s.SystemID == 0 && c.last.SystemID != 0 {
		s.SystemID = c.last.SystemID
		if c.last == s {
			return
		}
	}
	c.locked = true
	c.last = s
	c.bus.Publish(events.Event{Kind: events.KindCCLocked, Payload: s})
	c.log.Info("dpmr cc locked",
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
