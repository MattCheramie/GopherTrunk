package dstar

import (
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// LockState is the payload of cc.locked / cc.lost events emitted by
// the D-STAR repeater state machine.
type LockState struct {
	FrequencyHz uint32
	Repeater    string // RPT2 callsign of the first valid header (trimmed)
}

// LockedFrequencyHz / LockedNAC make LockState satisfy
// trunking.LockedPayload so the cchunt supervisor's state machine
// recognises D-STAR lock events alongside the other protocols.
// D-STAR doesn't have a P25-style NAC; the repeater callsign is the
// closest per-site identifier — its first 2 ASCII bytes are packed
// into the NAC slot so downstream consumers that key off LockedNAC
// see a stable per-site value across re-locks.
func (s LockState) LockedFrequencyHz() uint32 { return s.FrequencyHz }
func (s LockState) LockedNAC() uint16 {
	r := s.Repeater
	var nac uint16
	if len(r) > 0 {
		nac = uint16(r[0]) << 8
	}
	if len(r) > 1 {
		nac |= uint16(r[1])
	}
	return nac
}

// ControlChannel ingests D-STAR PCH headers from a single repeater
// frequency, emits cc.locked the first time a valid header arrives on
// a freshly-tuned device, and republishes group transmissions as
// events.KindGrant carrying a `trunking.Grant` payload with
// `Protocol = "dstar"`. D-STAR is conventional (each repeater is its
// own channel and there's no separate control channel allocating
// traffic frequencies), so a "grant" here just records that a new
// transmission has started and points the engine at the same
// frequency.
type ControlChannel struct {
	bus        *events.Bus
	log        *slog.Logger
	systemName string
	freqHz     uint32
	now        func() time.Time

	// proc is the cross-call bit / sync state the Process adapter
	// uses (see process.go). Lazily constructed on the first
	// Process call.
	proc *processState

	mu      sync.Mutex
	locked  bool
	last    LockState
	fecMode FECMode
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

// Ingest hands one decoded PCH header to the state machine. Real
// captures arrive via an upstream GMSK demod + convolutional /
// scrambler / deinterleaver; tests publish headers directly.
func (c *ControlChannel) Ingest(h Header) {
	if h.IsData() {
		// Pure data transmissions don't generate voice grants for the
		// engine to follow.
		return
	}
	c.maybeLock(LockState{
		FrequencyHz: c.freqHz,
		Repeater:    strings.TrimSpace(h.RPT2),
	})
	if h.IsGroupCall() {
		c.publishGrant(h)
	}
}

func (c *ControlChannel) publishGrant(h Header) {
	if c.bus == nil {
		return
	}
	// D-STAR doesn't have a numeric talkgroup — UR carries the routing
	// callsign. Hash the trimmed UR field into GroupID so two distinct
	// UR routings produce distinct GroupIDs in the call log without us
	// having to surface the string separately.
	ur := strings.TrimSpace(h.UR)
	src := strings.TrimSpace(h.MY1)
	c.bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System:      c.systemName,
			Protocol:    "dstar",
			GroupID:     hashCallsign(ur),
			SourceID:    hashCallsign(src),
			FrequencyHz: c.freqHz,
			ChannelNum:  0,
			Emergency:   h.IsEmergency(),
			At:          c.now(),
		},
	})
	c.log.Debug("dstar: grant",
		"system", c.systemName,
		"src", src, "ur", ur, "rpt1", strings.TrimSpace(h.RPT1),
		"rpt2", strings.TrimSpace(h.RPT2),
		"emer", h.IsEmergency())
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
	c.log.Info("dstar cc locked",
		"freq", s.FrequencyHz, "repeater", s.Repeater,
		"system", c.systemName)
}

// MarkLost publishes cc.lost and resets the locked flag. The trunking
// engine's hunter calls this when the repeater goes silent.
func (c *ControlChannel) MarkLost() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.locked {
		return
	}
	c.locked = false
	c.bus.Publish(events.Event{Kind: events.KindCCLost, Payload: c.last})
}

// hashCallsign maps a callsign string into a 32-bit identifier by
// packing up to 4 ASCII characters into the low / high halves. Two
// distinct callsigns hash to distinct IDs (no collisions for the
// 4-char-significant-prefix family); this keeps trunking.Grant typed
// the same way as the other protocols without us having to widen the
// schema.
func hashCallsign(s string) uint32 {
	var v uint32
	for i := 0; i < len(s) && i < 4; i++ {
		v = (v << 8) | uint32(s[i])
	}
	return v
}
