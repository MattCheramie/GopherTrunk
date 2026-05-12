package edacs

import (
	"log/slog"
	"strings"
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

// LockedFrequencyHz / LockedNAC make LockState satisfy
// trunking.LockedPayload so the cchunt supervisor's state machine
// recognises EDACS lock events alongside the protocol-neutral P25 /
// DMR / NXDN / TETRA payloads. EDACS doesn't have a P25-style NAC;
// SystemID is the closest per-site identifier and gets plumbed
// into the NAC slot. Without these methods, the supervisor's
// type-assertion on cc.locked silently drops the event and
// /api/v1/scanner never surfaces state=locked.
func (s LockState) LockedFrequencyHz() uint32 { return s.FrequencyHz }
func (s LockState) LockedNAC() uint16         { return s.SystemID }

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

	// proc is the cross-call bit / sync state the Process adapter
	// uses (see process.go). Lazily constructed on the first
	// Process call.
	proc *processState

	mu               sync.Mutex
	locked           bool
	last             LockState
	strictValidation bool
	bchMode          BCHMode
}

// BCHMode selects how the Process adapter interprets the incoming
// bit stream:
//
//   - BCHOff (default): the adapter slices 40-bit pre-stripped
//     information windows and treats them as CCWs. Works on
//     synthesized test fixtures whose codewords are not BCH-
//     protected.
//
//   - BCHOn: the adapter slices 40-bit on-wire BCH(40,28,2)
//     codewords (info at bits 12..39 high; 12-bit BCH parity at
//     bits 0..11 low), runs the
//     internal/radio/framing/bch_edacs.go primitive to
//     validate + correct up to 2 bit errors per codeword, then
//     re-encodes the corrected info into a 40-bit wire word
//     that the existing CCWFromBits parser consumes.
//
// Under BCHOn the effective CCW model carries:
//
//	Command (4 bits, positions 36..39 of the codeword)
//	Status  (4 bits, positions 32..35)
//	Address (16 bits, positions 16..31)
//	LCN     (4 bits, positions 12..15 — only the high 4 bits
//	         of the existing 5-bit Codeword.LCN field;
//	         the low bit is BCH parity)
//
// The existing `Aux` field at codeword bits 0..10 is BCH parity
// under BCHOn — not data. Callers that depend on the legacy
// 5-bit LCN range or the Aux payload should keep using BCHOff.
type BCHMode uint8

const (
	BCHOff BCHMode = iota
	BCHOn
)

// SetBCHMode toggles the BCH(40, 28, 2) FEC layer on the Process
// adapter. See BCHMode for the trade-offs. The mode applies to
// every subsequent Process call; the Ingest entry point is
// unaffected (callers that pre-parse CCWs don't go through this
// adapter).
func (c *ControlChannel) SetBCHMode(mode BCHMode) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bchMode = mode
}

// BCHMode returns the configured BCHMode.
func (c *ControlChannel) BCHMode() BCHMode {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.bchMode
}

// ParseBCHMode maps a config / user-facing string into a BCHMode.
// Recognised values (case-insensitive): "" / "off" / "false" / "0"
// → BCHOff (legacy pre-stripped 40-bit CCW path); "on" / "true" /
// "1" → BCHOn (40-bit on-wire BCH(40, 28, 2) decode + 1/2-bit
// correction). Unknown strings return BCHOff with `ok = false`.
func ParseBCHMode(s string) (BCHMode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "off", "false", "0":
		return BCHOff, true
	case "on", "true", "1":
		return BCHOn, true
	default:
		return BCHOff, false
	}
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

// SetStrictValidation toggles the strict frame-validity filter on
// the Ingest path. When enabled, CCWs whose Command field falls
// outside the recognised opcode set (see Command.IsKnown) are
// silently dropped. This is "soft FEC" — it doesn't correct bit
// errors, but it rejects most random-noise codewords by relying on
// the protocol-level invariant that real CCWs only ever carry one
// of the documented opcodes. Useful when the upstream FEC layer
// isn't yet implemented.
func (c *ControlChannel) SetStrictValidation(strict bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.strictValidation = strict
}

// Ingest hands a single decoded CCW to the state machine. Real
// captures arrive via an upstream GMSK demod + FEC; tests publish
// CCWs directly.
func (c *ControlChannel) Ingest(w CCW) {
	if w.IsIdle() {
		return
	}
	c.mu.Lock()
	strict := c.strictValidation
	c.mu.Unlock()
	if strict && !w.Command.IsKnown() {
		// Drop CCWs whose Command is outside the recognised set.
		// Almost certainly a misaligned codeword or noise.
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
