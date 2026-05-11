package phase2

import (
	"log/slog"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// ControlChannel ingests P25 Phase 2 MAC PDUs from a single Phase 2
// traffic channel and republishes voice grants as
// events.KindGrant. Phase 2 doesn't have a dedicated control
// channel — late-grant signalling rides MAC slots interleaved with
// voice — so the state machine treats every MAC PDU as a
// potential grant carrier.
//
// Mirrors the shape of internal/radio/p25/phase1/control.go: the
// engine-facing surface is identical (cc.locked / cc.lost / grant
// events), with `trunking.Grant.Protocol = "p25-phase2"` so the
// engine + recorder + composer don't need to know the difference.
type ControlChannel struct {
	bus        *events.Bus
	log        *slog.Logger
	systemName string
	freqHz     uint32
	now        func() time.Time

	// proc is the cross-call dibit / sync state the Process
	// adapter uses (see process.go). Lazily constructed on the
	// first Process call.
	proc *processState

	mu               sync.Mutex
	locked           bool
	strictValidation bool
}

// SetStrictValidation toggles the strict frame-validity filter on the
// Ingest path. When enabled, MAC PDUs whose 8-bit Opcode is not in
// the documented TIA-102.AABF / BBAB set are silently dropped at
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
		now:        now,
	}
}

// Ingest hands one decoded MAC PDU to the state machine. Real
// captures arrive from an upstream H-DQPSK demod + TDMA superframe
// sync + Trellis FEC; tests publish PDUs directly.
func (c *ControlChannel) Ingest(p MACPDU) {
	c.mu.Lock()
	strict := c.strictValidation
	c.mu.Unlock()
	if strict && !p.Opcode.IsKnown() {
		return
	}
	if p.IsIdle() {
		return
	}
	if !c.locked {
		c.mu.Lock()
		if !c.locked {
			c.locked = true
			c.bus.Publish(events.Event{
				Kind: events.KindCCLocked,
				Payload: LockState{
					FrequencyHz: c.freqHz,
				},
			})
			c.log.Info("p25/phase2 cc locked",
				"freq", c.freqHz, "system", c.systemName)
		}
		c.mu.Unlock()
	}
	if g, ok := p.AsGroupVoiceChannelGrant(); ok {
		c.publishGrant(g, p.Opcode)
	}
}

// LockState is the payload of cc.locked / cc.lost events emitted
// by the Phase 2 state machine.
type LockState struct {
	FrequencyHz uint32
}

func (c *ControlChannel) publishGrant(g GroupVoiceChannelGrant, op Opcode) {
	if c.bus == nil {
		return
	}
	c.bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System:     c.systemName,
			Protocol:   "p25-phase2",
			GroupID:    uint32(g.GroupAddress),
			SourceID:   g.SourceID,
			ChannelID:  g.ChannelID,
			ChannelNum: g.ChannelNumber,
			At:         c.now(),
		},
	})
	c.log.Debug("p25/phase2 grant",
		"system", c.systemName,
		"opcode", op, "tg", g.GroupAddress,
		"src", g.SourceID,
		"channel_id", g.ChannelID, "channel_num", g.ChannelNumber)
}

// MarkLost publishes cc.lost and resets the locked flag. The
// engine's hunter calls this when no MAC PDU has arrived for the
// configured timeout.
func (c *ControlChannel) MarkLost() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.locked {
		return
	}
	c.locked = false
	c.bus.Publish(events.Event{Kind: events.KindCCLost, Payload: LockState{FrequencyHz: c.freqHz}})
}
