package tetra

import (
	"log/slog"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// LockState is the payload of cc.locked / cc.lost events emitted by
// the TETRA TMO control-channel state machine.
type LockState struct {
	FrequencyHz  uint32
	MCC          uint16 // first MLE-SYSINFO MCC, when seen
	MNC          uint16 // first MLE-SYSINFO MNC, when seen
	LocationArea uint16
}

// ControlChannel ingests TETRA Layer-3 PDUs from a single control
// channel, emits cc.locked the first time a valid MLE-SYSINFO (or
// any non-idle CMCE PDU) arrives on a freshly-tuned device, and
// republishes voice grants as events.KindGrant carrying a
// `trunking.Grant` payload with `Protocol = "tetra"`. Same shape as
// the other trunked-protocol control channels.
type ControlChannel struct {
	bus        *events.Bus
	log        *slog.Logger
	systemName string
	freqHz     uint32
	resolver   Resolver
	now        func() time.Time

	// proc is the cross-call dibit / sync state the Process
	// adapter uses (see process.go). Lazily constructed on the
	// first Process call.
	proc *processState

	mu               sync.Mutex
	locked           bool
	last             LockState
	strictValidation bool
}

// SetStrictValidation toggles the strict frame-validity filter on the
// Ingest path. When enabled, PDUs whose (Discriminator, Type) pair is
// not in the documented ETSI EN 300 392-2 set are silently dropped at
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

// Ingest hands a single decoded PDU to the state machine. Real
// captures arrive via an upstream π/4-DQPSK demod + RCPC/RM FEC;
// tests publish PDUs directly.
func (c *ControlChannel) Ingest(p PDU) {
	c.mu.Lock()
	strict := c.strictValidation
	c.mu.Unlock()
	if strict && !p.IsKnown() {
		return
	}
	if p.IsIdle() {
		return
	}
	if sb, ok := p.AsSystemBroadcast(); ok {
		c.maybeLock(LockState{
			FrequencyHz:  c.freqHz,
			MCC:          sb.MCC,
			MNC:          sb.MNC,
			LocationArea: sb.LocationArea,
		})
		return
	}
	if g, ok := p.AsVoiceGrant(); ok {
		// Even without a prior SYSINFO, a voice grant on the CC is
		// enough to declare the channel locked.
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
		if hz, err := c.resolver.Frequency(g.CarrierNumber); err == nil {
			freq = hz
		} else {
			c.log.Debug("tetra: band-plan resolution failed",
				"carrier", g.CarrierNumber, "err", err)
		}
	}
	c.bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System:      c.systemName,
			Protocol:    "tetra",
			GroupID:     g.DestSSI,
			SourceID:    g.SourceSSI,
			FrequencyHz: freq,
			ChannelNum:  g.CarrierNumber,
			Encrypted:   g.Encrypted,
			Emergency:   g.Emergency,
			At:          c.now(),
		},
	})
	c.log.Debug("tetra: grant",
		"system", c.systemName,
		"src", g.SourceSSI, "dst", g.DestSSI,
		"carrier", g.CarrierNumber, "slot", g.Timeslot, "freq_hz", freq,
		"group", g.Group, "enc", g.Encrypted, "emer", g.Emergency)
}

func (c *ControlChannel) maybeLock(s LockState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.locked && c.last == s {
		return
	}
	// Preserve previously-learned MCC/MNC/LA if the new state has none.
	if c.locked && s.MCC == 0 && c.last.MCC != 0 {
		s.MCC = c.last.MCC
		s.MNC = c.last.MNC
		s.LocationArea = c.last.LocationArea
		if c.last == s {
			return
		}
	}
	c.locked = true
	c.last = s
	c.bus.Publish(events.Event{Kind: events.KindCCLocked, Payload: s})
	c.log.Info("tetra cc locked",
		"freq", s.FrequencyHz, "mcc", s.MCC, "mnc", s.MNC,
		"la", s.LocationArea, "system", c.systemName)
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
