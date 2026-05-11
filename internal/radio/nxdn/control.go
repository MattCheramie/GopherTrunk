package nxdn

import (
	"log/slog"

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
//     ½-rate Viterbi primitive in internal/radio/framing (the
//     same code-polynomial pair used by MMDVMHost / DSDcc /
//     op25), and parses the 88 leading info bits as a CAC. This
//     matches the bare-bones NXDN CAC FEC layer (88 CAC bits +
//     4 tail zeros → K=5 R=½). Inner per-protocol interleaver +
//     puncture (spec-shape-dependent and not in the public
//     references) are still deferred.
type ViterbiMode uint8

const (
	ViterbiOff ViterbiMode = iota
	ViterbiOn
)

// SetViterbiMode toggles the K=5 ½-rate Viterbi FEC layer on the
// CAC region of the Info field. See ViterbiMode for the trade-
// offs. The mode applies to every subsequent Process call; the
// IngestFrame entry point is unaffected (callers that pre-parse
// frames don't go through this adapter).
func (c *ControlChannel) SetViterbiMode(mode ViterbiMode) {
	c.viterbiMode = mode
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
