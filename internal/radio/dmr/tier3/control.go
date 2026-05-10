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
func (s LockState) LockedNAC() uint16          { return s.SystemID }

// ControlChannel ingests detected DMR bursts whose Slot Type
// identifies a CSBK, runs BPTC(196,96) decode + CRC, and dispatches
// each opcode:
//
//   - OpAloha / OpSysInfo announce the trunked system → CCLocked
//     events fan out the first time we see one.
//   - OpTVGrant / OpPVGrant carry voice grants. The LCN is resolved
//     through the supplied band plan; on success a trunking.Grant is
//     published with Protocol = "dmr-tier3". A grant whose LCN has no
//     entry in the band plan publishes a `decode.error` with
//     stage="no-bandplan" so operators can spot configuration gaps.
//
// Every other opcode (preamble, ACK, Ahoy, neighbor lists, …) is
// logged at debug and ignored — they're noise from the hunter's
// perspective.
type ControlChannel struct {
	bus        *events.Bus
	log        *slog.Logger
	systemName string
	freqHz     uint32
	resolver   Resolver
	now        func() time.Time
	locked     bool
	last       LockState
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
		bus:        opts.Bus,
		log:        log,
		systemName: opts.SystemName,
		freqHz:     opts.FrequencyHz,
		resolver:   opts.Resolver,
		now:        now,
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
	switch csbk.Opcode {
	case OpAloha:
		c.maybeLock(LockState{FrequencyHz: c.freqHz, ColorCode: cc, SystemID: ParseAloha(csbk.Payload).SystemID})
	case OpSysInfo:
		si := ParseSystemInfoBroadcast(csbk.Payload)
		c.maybeLock(LockState{FrequencyHz: c.freqHz, ColorCode: cc, SystemID: si.SystemID})
	case OpTVGrant:
		c.publishTVGrant(cc, ParseTVGrant(csbk.Payload))
	case OpPVGrant:
		c.publishPVGrant(cc, ParsePVGrant(csbk.Payload))
	default:
		c.log.Debug("dmr/tier3: csbk", "opcode", csbk.Opcode, "cc", cc)
	}
}

func (c *ControlChannel) publishTVGrant(cc uint8, g TVGrant) {
	freq, ok := c.resolveLCN(g.LCN)
	if !ok {
		return
	}
	emergency := g.ServiceOptions&0x80 != 0
	encrypted := g.ServiceOptions&0x40 != 0
	c.bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System:      c.systemName,
			Protocol:    "dmr-tier3",
			GroupID:     g.GroupAddress,
			SourceID:    g.SourceID,
			FrequencyHz: freq,
			ChannelID:   cc,
			ChannelNum:  uint16(g.LCN),
			Encrypted:   encrypted,
			Emergency:   emergency,
			At:          c.now(),
		},
	})
	c.log.Debug("dmr/tier3: tv-grant",
		"system", c.systemName, "cc", cc, "tg", g.GroupAddress, "src", g.SourceID,
		"lcn", g.LCN, "ts", g.Timeslot, "freq_hz", freq,
		"enc", encrypted, "emer", emergency)
}

func (c *ControlChannel) publishPVGrant(cc uint8, g PVGrant) {
	freq, ok := c.resolveLCN(g.LCN)
	if !ok {
		return
	}
	emergency := g.ServiceOptions&0x80 != 0
	encrypted := g.ServiceOptions&0x40 != 0
	c.bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System:      c.systemName,
			Protocol:    "dmr-tier3",
			GroupID:     g.DestinationID,
			SourceID:    g.SourceID,
			FrequencyHz: freq,
			ChannelID:   cc,
			ChannelNum:  uint16(g.LCN),
			Encrypted:   encrypted,
			Emergency:   emergency,
			At:          c.now(),
		},
	})
	c.log.Debug("dmr/tier3: pv-grant",
		"system", c.systemName, "cc", cc, "dst", g.DestinationID, "src", g.SourceID,
		"lcn", g.LCN, "ts", g.Timeslot, "freq_hz", freq,
		"enc", encrypted, "emer", emergency)
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
