package tier3

import (
	"log/slog"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// LockState is the payload of cc.locked / cc.lost events emitted by the
// DMR Tier III control-channel state machine.
type LockState struct {
	FrequencyHz uint32
	ColorCode   uint8
	SystemID    uint16
}

// LockedFrequencyHz / LockedNAC implement trunking.LockedPayload so
// the cc-hunter can consume Tier III lock events without importing
// this package.
func (s LockState) LockedFrequencyHz() uint32 { return s.FrequencyHz }
func (s LockState) LockedNAC() uint16         { return s.SystemID }

// ControlChannel ingests detected DMR bursts whose Slot Type
// identifies a CSBK, runs BPTC(196,96) decode + CRC, and dispatches
// each opcode:
//
//   - OpAloha announces the trunked system → CCLocked events fan out
//     the first time we see one. OpBcast (C_BCAST) carries the site's
//     Gen_Site_Params and Adjacent_Site announcements, folded into the
//     topology model.
//   - OpTVGrant / OpPVGrant carry voice grants. The LCN is resolved
//     through the supplied band plan; on success a trunking.Grant is
//     published with Protocol = "dmr-tier3". A grant whose LCN has no
//     entry in the band plan publishes a `decode.error` with
//     stage="no-bandplan" so operators can spot configuration gaps.
//
// Every CSBK is logged at debug (so the control stream is visible the
// way dsd-neo shows it); opcodes without a dedicated handler (ACK,
// Ahoy, Move, Preamble, …) are otherwise ignored.
type ControlChannel struct {
	bus              *events.Bus
	log              *slog.Logger
	systemName       string
	freqHz           uint32
	resolver         Resolver
	now              func() time.Time
	interleavedVoice bool
	locked           bool
	last             LockState

	// topo accumulates the system topology (identity + adjacent sites) for the
	// hunt/discovery layer; read via Topology().
	topo topologyModel

	// restChannel tracks the LCN a Capacity Plus system currently
	// advertises as its rest (control) channel. Zero until a
	// Motorola vendor system-info CSBK has been seen.
	restChannel uint8

	// proc is the cross-call dibit / sync state the Process adapter
	// uses (see process.go). Lazily constructed on the first
	// Process call so tests that drive IngestBurst directly don't
	// pay the construction cost.
	proc *processState
}

// Options configure a ControlChannel.
type Options struct {
	Bus         *events.Bus
	Log         *slog.Logger
	SystemName  string
	FrequencyHz uint32
	Resolver    Resolver
	Now         func() time.Time
	// InterleavedVoice tags published voice grants with
	// Grant.DMRInterleavedVoice so the composer uses the 2-slot
	// interleaved decoder. Mirrors System.DMRInterleavedVoice.
	InterleavedVoice bool
}

// New constructs a ControlChannel from Options.
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
		bus:              opts.Bus,
		log:              log,
		systemName:       opts.SystemName,
		freqHz:           opts.FrequencyHz,
		resolver:         opts.Resolver,
		now:              now,
		interleavedVoice: opts.InterleavedVoice,
	}
}

// NewControlChannel keeps the legacy positional constructor working —
// existing tests + callers that don't yet care about grant emission
// don't need to migrate.
func NewControlChannel(bus *events.Bus, log *slog.Logger, freqHz uint32) *ControlChannel {
	return New(Options{Bus: bus, Log: log, FrequencyHz: freqHz})
}

// IngestBurst hands one DMR burst to the state machine. The burst's
// slot type must already be parsed by the caller; the 20-bit
// Hamming(20,8) over the slot type lives in dmr/slottype.go.
func (c *ControlChannel) IngestBurst(b *dmr.Burst, slot dmr.SlotType) {
	if slot.DataType != dmr.DTCSBK {
		return
	}
	bits, errs := framing.DecodeBPTC196_96(b.PayloadBits())
	if errs < 0 {
		c.log.Debug("dmr/tier3: BPTC uncorrectable")
		return
	}
	csbk, err := ParseCSBK(InfoBitsToBytes(bits))
	if err != nil {
		c.log.Debug("dmr/tier3: CSBK CRC failed")
		return
	}
	c.handleCSBK(slot.ColorCode, csbk)
}

func (c *ControlChannel) handleCSBK(cc uint8, csbk CSBK) {
	// Dispatch on FID before opcode: a vendor CSBK is routed to the
	// vendor handler so its opcode is not misread against the
	// standard ETSI opcode table.
	if vendor := VendorFromFID(csbk.FID); vendor != VendorStandard {
		c.handleVendorCSBK(vendor, cc, csbk)
		return
	}
	// Log every standard CSBK so the control-channel stream is visible
	// in the debug log the way a reference decoder (dsd-neo) shows it —
	// the dominant Aloha beacon is otherwise consumed silently once the
	// CC is locked, which made GopherTrunk look idle.
	c.log.Debug("dmr/tier3: csbk", "opcode", csbk.Opcode, "fid", csbk.FID, "cc", cc)
	switch csbk.Opcode {
	case OpAloha:
		sysID := ParseAloha(csbk.Payload).SystemID
		c.topo.applyIdentity(sysID, cc)
		c.maybeLock(LockState{FrequencyHz: c.freqHz, ColorCode: cc, SystemID: sysID})
	case OpBcast:
		c.handleBroadcast(cc, ParseBroadcast(csbk.Payload))
	case OpTVGrant:
		c.publishTVGrant(cc, ParseTVGrant(csbk.Payload))
	case OpPVGrant:
		c.publishPVGrant(cc, ParsePVGrant(csbk.Payload))
	}
}

// handleBroadcast folds a C_BCAST announcement into the topology model.
// Gen_Site_Params contributes the camped site's identity + RFSS/Site;
// Adjacent_Site adds a neighbour. Locking stays Aloha-driven: the
// Gen_Site_Params identity field differs from the Aloha raw identity, so
// driving maybeLock from both would churn the lock state every burst.
func (c *ControlChannel) handleBroadcast(cc uint8, b BroadcastAnnouncement) {
	switch b.Type {
	case AnncGenSiteParms:
		gs := ParseGenSiteParams(b.Payload)
		c.topo.applyIdentity(gs.SystemID, cc)
		c.topo.applySiteParams(gs.RFSSID, gs.SiteID)
	case AnncAdjacentSite:
		c.topo.applyAdjacent(ParseAdjacentSite(b.Payload))
	}
}

// publishGrant resolves an LCN to a downlink frequency and publishes a
// voice grant on the bus. Shared by the standard and vendor CSBK
// paths. Returns the resolved frequency and false when the LCN has no
// band-plan entry (resolveLCN has already published the decode error).
//
// slot is the CSBK's 0-based timeslot bit (0 = TS1, 1 = TS2); it is
// mapped to trunking.Grant's 1-based Timeslot (1 = TS1, 2 = TS2) so a
// DMR call's slot becomes part of the engine's (frequency, timeslot)
// call identity — both slots of a 12.5 kHz carrier carry independent
// calls.
func (c *ControlChannel) publishGrant(cc, lcn, slot uint8, group, source uint32, serviceOptions uint8) (uint32, bool) {
	freq, ok := c.resolveLCN(lcn)
	if !ok {
		return 0, false
	}
	c.bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System:              c.systemName,
			Protocol:            "dmr-tier3",
			GroupID:             group,
			SourceID:            source,
			FrequencyHz:         freq,
			ChannelID:           cc,
			ChannelNum:          uint16(lcn),
			Timeslot:            slot + 1,
			DMRInterleavedVoice: c.interleavedVoice,
			Encrypted:           serviceOptions&0x40 != 0,
			Emergency:           serviceOptions&0x80 != 0,
			At:                  c.now(),
		},
	})
	return freq, true
}

func (c *ControlChannel) publishTVGrant(cc uint8, g TVGrant) {
	freq, ok := c.publishGrant(cc, g.LCN, g.Timeslot, g.GroupAddress, g.SourceID, g.ServiceOptions)
	if !ok {
		return
	}
	c.log.Debug("dmr/tier3: tv-grant",
		"system", c.systemName, "cc", cc, "tg", g.GroupAddress, "src", g.SourceID,
		"lcn", g.LCN, "ts", g.Timeslot, "freq_hz", freq)
}

func (c *ControlChannel) publishPVGrant(cc uint8, g PVGrant) {
	freq, ok := c.publishGrant(cc, g.LCN, g.Timeslot, g.DestinationID, g.SourceID, g.ServiceOptions)
	if !ok {
		return
	}
	c.log.Debug("dmr/tier3: pv-grant",
		"system", c.systemName, "cc", cc, "dst", g.DestinationID, "src", g.SourceID,
		"lcn", g.LCN, "ts", g.Timeslot, "freq_hz", freq)
}

func (c *ControlChannel) resolveLCN(lcn uint8) (uint32, bool) {
	if c.resolver == nil {
		c.log.Debug("dmr/tier3: grant dropped, no band-plan resolver configured", "lcn", lcn)
		c.bus.Publish(events.Event{
			Kind:    events.KindDecodeError,
			Payload: events.DecodeError{Protocol: "dmr-tier3", Stage: events.StageNoBandPlan},
		})
		return 0, false
	}
	freq, err := c.resolver.Frequency(lcn)
	if err != nil {
		c.log.Debug("dmr/tier3: band-plan miss", "lcn", lcn, "err", err)
		c.bus.Publish(events.Event{
			Kind:    events.KindDecodeError,
			Payload: events.DecodeError{Protocol: "dmr-tier3", Stage: events.StageNoBandPlan},
		})
		return 0, false
	}
	return freq, true
}

func (c *ControlChannel) maybeLock(s LockState) {
	if !c.locked || c.last != s {
		c.locked = true
		c.last = s
		c.bus.Publish(events.Event{Kind: events.KindCCLocked, Payload: s})
		c.log.Info("dmr cc locked", "freq", s.FrequencyHz, "cc", s.ColorCode, "sysid", s.SystemID)
	}
}

// MarkLost publishes cc.lost and resets the locked flag. Wired up by the
// engine's watchdog.
func (c *ControlChannel) MarkLost() {
	if !c.locked {
		return
	}
	c.locked = false
	c.bus.Publish(events.Event{Kind: events.KindCCLost, Payload: c.last})
}

// Topology returns a snapshot of the system topology accumulated from the
// site's Aloha / System-Info / Adjacent-Site CSBKs. Used by the signal-lab /
// hunt layers to document a discovered DMR Tier III system.
func (c *ControlChannel) Topology() TopologyConfig { return c.topo.snapshot() }
