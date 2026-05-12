package nxdn

import (
	"log/slog"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// LockState is the payload of cc.locked / cc.lost events emitted by the
// NXDN control-channel state machine.
type LockState struct {
	FrequencyHz uint32
	BaudRate    BaudRate
	SiteID      uint16
	SystemID    uint16
}

// LockedFrequencyHz / LockedNAC make LockState satisfy
// trunking.LockedPayload so the cchunt supervisor's state
// machine recognises NXDN lock events alongside the protocol-
// neutral P25 / DMR / TETRA payloads it already handles.
//
// NXDN doesn't have a P25-style NAC; the SiteID is the closest
// per-cell identifier (it scopes the lock to a specific repeater
// + site combination), so it's plumbed into the "NAC" slot.
// Downstream consumers that key off LockedNAC see a stable
// per-site value across re-locks.
func (s LockState) LockedFrequencyHz() uint32 { return s.FrequencyHz }
func (s LockState) LockedNAC() uint16         { return s.SiteID }

// ControlChannel ingests parsed NXDN frames whose LICH indicates RCCH
// (control channel), reads the CAC payload, and emits cc.locked when it
// observes a SITE_INFO or CCH announcement message that confirms the
// frequency. Mirrors the P25 phase1 / DMR tier3 control channels.
//
// The full RCCH state machine (Aloha tracking, channel-grant follow,
// neighbor list reaction) lives in the trunking engine, which
// subscribes to the events this package emits.
type ControlChannel struct {
	bus    *events.Bus
	log    *slog.Logger
	freqHz uint32
	rate   BaudRate
	locked bool
	last   LockState

	// proc is the cross-call dibit / sync state the Process
	// adapter uses (see process.go). Lazily constructed on the
	// first Process call.
	proc *processState

	// viterbiMode controls whether Process treats the CAC region
	// of the Info field as raw on-wire bits (ViterbiOff, default)
	// or runs the 144 dibits through the K=5 ½-rate Viterbi
	// primitive before parsing (ViterbiOn). Set via
	// SetViterbiMode.
	viterbiMode ViterbiMode
}

// ViterbiMode selects how the Process adapter interprets the CAC
// region inside the 144-dibit Info field.
//
//   - ViterbiOff (default): the adapter reads 44 dibits = 88 raw
//     information bits straight off the wire. Works on test
//     fixtures + clean synthesized streams whose CAC bits aren't
//     channel-coded; it does NOT match the on-air NXDN encoding,
//     so live-air CAC frames typically fail their CRC and the
//     adapter silently drops them.
//
//   - ViterbiOn: the adapter collects the first 92 dibits of the
//     Info field (= 184 wire bits), runs them through the K=5
//     ½-rate Viterbi primitive in internal/radio/framing, and
//     parses the 88 leading info bits as a CAC. This matches a
//     bare-bones NXDN CAC FEC layer (88 CAC bits + 4 tail zeros
//     → K=5 R=½), as used by MMDVMHost / DSDcc test fixtures.
//     Inner per-protocol interleaver + puncture are still
//     deferred.
//
//   - ViterbiSpec: the adapter slices the full 150-dibit CAC slot
//     (= 300 channel bits) and runs the spec-correct chain per
//     NXDN-TS-1-A rev 1.3 §4.5.1.1 (CAC outbound): deinterleave
//     25×12 → depuncture 50/350 → K=5 R=½ Viterbi → 16-bit CRC
//     verify → strip tail. Recovers the 155-bit info block
//     (8 SR + 144 L3 Data + 3 Null); the first 88 bits of the L3
//     are then run through ParseCAC for compatibility with the
//     existing RCCH parser. This is the path that lights up on
//     live captures.
type ViterbiMode uint8

const (
	ViterbiOff ViterbiMode = iota
	ViterbiOn
	ViterbiSpec
)

// SetViterbiMode toggles the K=5 ½-rate Viterbi FEC layer on the
// CAC region of the Info field. See ViterbiMode for the trade-
// offs. The mode applies to every subsequent Process call; the
// IngestFrame entry point is unaffected (callers that pre-parse
// frames don't go through this adapter).
func (c *ControlChannel) SetViterbiMode(mode ViterbiMode) {
	c.viterbiMode = mode
}

// ViterbiMode returns the configured ViterbiMode. Mirrors the
// Set* family so callers (and tests) can introspect the configured
// mode without poking at unexported state.
func (c *ControlChannel) ViterbiMode() ViterbiMode {
	return c.viterbiMode
}

// ParseViterbiMode maps a config / user-facing string into a
// ViterbiMode. Recognised values (case-insensitive): "" / "off" /
// "false" / "0" → ViterbiOff (legacy 44-dibit raw-CAC path);
// "on" / "true" / "1" → ViterbiOn (92 dibits via the simplified
// K=5 Viterbi path used by older MMDVMHost / DSDcc fixtures);
// "spec" → ViterbiSpec (150 dibits via the full NXDN-TS-1-A
// §4.5.1.1 outbound CAC chain — deinterleave + depuncture + K=5
// Viterbi + 16-bit CRC + tail strip — the path that works on
// live captures). Unknown strings return ViterbiOff with
// `ok = false`.
func ParseViterbiMode(s string) (ViterbiMode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "off", "false", "0":
		return ViterbiOff, true
	case "on", "true", "1":
		return ViterbiOn, true
	case "spec":
		return ViterbiSpec, true
	default:
		return ViterbiOff, false
	}
}

func NewControlChannel(bus *events.Bus, log *slog.Logger, freqHz uint32, rate BaudRate) *ControlChannel {
	if log == nil {
		log = slog.Default()
	}
	return &ControlChannel{bus: bus, log: log, freqHz: freqHz, rate: rate}
}

// IngestFrame consumes one decoded NXDN frame. The caller must have
// already extracted and parsed the LICH and CAC fields.
func (c *ControlChannel) IngestFrame(lich LICH, cac *CACMessage) {
	if !lich.ParityOK || lich.RFCh != RFChControl {
		return
	}
	if cac == nil {
		return
	}
	switch cac.Type {
	case RCCHSITEINFO:
		s := ParseSiteInfo(cac.Payload)
		c.maybeLock(LockState{FrequencyHz: c.freqHz, BaudRate: c.rate, SiteID: s.SiteID, SystemID: s.SystemID})
	case RCCHCCH:
		c.maybeLock(LockState{FrequencyHz: c.freqHz, BaudRate: c.rate})
	}
}

func (c *ControlChannel) maybeLock(s LockState) {
	if !c.locked || c.last != s {
		c.locked = true
		c.last = s
		c.bus.Publish(events.Event{Kind: events.KindCCLocked, Payload: s})
		c.log.Info("nxdn cc locked", "freq", s.FrequencyHz, "rate", s.BaudRate.String(), "site", s.SiteID, "sys", s.SystemID)
	}
}

// MarkLost publishes cc.lost and resets the locked flag.
func (c *ControlChannel) MarkLost() {
	if !c.locked {
		return
	}
	c.locked = false
	c.bus.Publish(events.Event{Kind: events.KindCCLost, Payload: c.last})
}
