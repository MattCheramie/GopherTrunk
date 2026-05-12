package phase2

import (
	"log/slog"
	"strings"
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
	trellisMode      TrellisMode
}

// TrellisMode selects how the Process adapter interprets the MAC
// PDU dibit window inside the Phase 2 traffic channel.
//
//   - TrellisOff (default): the adapter reads 72 dibits = 144 raw
//     information bits straight off the wire, parses 18 bytes as
//     a MAC PDU. Works on test fixtures + clean synthesized
//     streams whose MAC bits aren't trellis-coded; matches the
//     legacy adapter behaviour.
//
//   - TrellisOn: the adapter collects 146 channel dibits (72 info
//     + 1 finisher transition × 2 channel dibits per transition),
//     runs them through the TIA-102 Annex A 4-state ½-rate
//     trellis Viterbi decoder in
//     internal/radio/framing/p25_trellis.go, and parses the
//     recovered 72 info dibits = 18 bytes as a MAC PDU. The
//     trellis tables are identical to the ones P25 Phase 1 uses
//     for TSBKs (TIA-102.BAAA-A Annex A); TIA-102.BBAB inherits
//     them for Phase 2.
//
// The Reed-Solomon outer layer + the per-burst block interleaver
// that the Phase 2 spec wraps around the trellis-coded MAC PDU
// are documented follow-ups; TrellisOn handles bare-bones
// trellis coding only.
type TrellisMode uint8

const (
	TrellisOff TrellisMode = iota
	TrellisOn
)

// SetTrellisMode toggles the 4-state ½-rate trellis FEC layer on
// the MAC PDU dibit window. See TrellisMode for the trade-offs.
// The mode applies to every subsequent Process call; the Ingest
// entry point is unaffected (callers that pre-parse MAC PDUs
// don't go through this adapter).
func (c *ControlChannel) SetTrellisMode(mode TrellisMode) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.trellisMode = mode
}

// TrellisMode returns the current TrellisMode. Mirrors the Set*
// family so callers (and tests) can introspect the configured
// mode without poking at unexported state.
func (c *ControlChannel) TrellisMode() TrellisMode {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.trellisMode
}

// ParseTrellisMode maps a config / user-facing string into a
// TrellisMode. Recognised values (case-insensitive): "" / "off" /
// "false" / "0" → TrellisOff (the legacy 72-dibit raw-MAC-PDU
// path); "on" / "true" / "1" → TrellisOn (146 channel dibits run
// through the 4-state ½-rate trellis decoder). Unknown strings
// return TrellisOff with `ok = false` so callers can surface the
// misconfiguration.
func ParseTrellisMode(s string) (TrellisMode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "off", "false", "0":
		return TrellisOff, true
	case "on", "true", "1":
		return TrellisOn, true
	default:
		return TrellisOff, false
	}
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

// LockedFrequencyHz / LockedNAC make LockState satisfy
// trunking.LockedPayload so the cchunt supervisor's state machine
// recognises P25 Phase 2 lock events alongside the protocol-neutral
// P25 Phase 1 / DMR / NXDN / TETRA payloads. Phase 2's MAC PDU
// header doesn't carry a NAC equivalent (the NAC lives one layer
// up in the Phase 2 superframe), so LockedNAC returns 0; the
// supervisor uses it only as a cache key on retune, so 0 is
// harmless. Without these methods, the supervisor's type-assertion
// on cc.locked silently drops the event and /api/v1/scanner never
// surfaces state=locked.
func (s LockState) LockedFrequencyHz() uint32 { return s.FrequencyHz }
func (s LockState) LockedNAC() uint16         { return 0 }

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
