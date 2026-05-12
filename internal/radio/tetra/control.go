package tetra

import (
	"log/slog"
	"strings"
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

// LockedFrequencyHz / LockedNAC make LockState satisfy
// trunking.LockedPayload so the cchunt supervisor's state machine
// recognises TETRA lock events alongside the protocol-neutral P25 /
// DMR / NXDN payloads. TETRA doesn't have a P25-style NAC; the
// LocationArea is the closest per-cell identifier and gets plumbed
// into the NAC slot. Without these methods, the supervisor's
// type-assertion on cc.locked silently drops the event and
// /api/v1/scanner never surfaces state=locked.
func (s LockState) LockedFrequencyHz() uint32 { return s.FrequencyHz }
func (s LockState) LockedNAC() uint16         { return s.LocationArea }

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
	channelCoding    ChannelCodingMode
	channelType      ChannelType
	colourCode       uint32
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

// ChannelCodingMode selects how the Process adapter interprets the
// incoming dibit stream:
//
//   - ChannelCodingOff (default): the adapter slices a fixed 48-
//     dibit window after each normal-training-sequence sync and
//     parses the bits straight as a PDU. Works on synthesized
//     test fixtures where the type-2 / type-5 layers aren't
//     present; matches the legacy adapter behaviour.
//
//   - ChannelCodingOn: the adapter slices the channel-appropriate
//     number of dibits per the configured ChannelType, runs the
//     full type-5 → type-1 decode chain (descramble + deinterleave
//     + depuncture + Viterbi + CRC-16 verify + tail strip) per
//     ETSI EN 300 392-2 §8.3.1 using the per-channel helpers in
//     channel_coding.go, then parses the recovered info bits as a
//     PDU. Frames whose CRC fails are silently dropped.
//
// Use SetColourCode to seed the scrambler and SetExpectedChannel
// to tell the adapter which logical channel lives in each burst.
type ChannelCodingMode uint8

const (
	ChannelCodingOff ChannelCodingMode = iota
	ChannelCodingOn
)

// ChannelType identifies which TETRA logical channel the Process
// adapter is currently decoding under ChannelCodingOn. The
// connector (or higher-layer caller) sets this per burst /
// per-slot via SetExpectedChannel.
type ChannelType uint8

const (
	// ChannelSCHHD covers SCH/HD, BNCH and STCH — they share the
	// same coding chain per §8.3.1.4.1. 216 type-5 bits / 108
	// dibits per burst, recovering 124 info bits.
	ChannelSCHHD ChannelType = iota
	// ChannelSCHF — full-slot signaling channel. 432 type-5 bits
	// / 216 dibits, recovering 268 info bits.
	ChannelSCHF
	// ChannelSCHHU — half-slot signaling on the uplink. 168
	// type-5 bits / 84 dibits, recovering 92 info bits.
	ChannelSCHHU
	// ChannelBSCH — broadcast synchronisation channel. 120 type-5
	// bits / 60 dibits, recovering 60 info bits. Colour code
	// is implicitly 0 for BSCH regardless of SetColourCode.
	ChannelBSCH
	// ChannelAACH — access-assignment channel (slot header).
	// 30 type-5 bits / 15 dibits, recovering 14 info bits.
	// AACH skips RCPC + interleaving, just RM + scramble.
	ChannelAACH
)

// ParseChannelType maps a config / user-facing string into a
// ChannelType. Recognised values (case-insensitive, "/" optional):
// "sch/hd" | "schhd" | "sch_hd", "sch/f" | "schf", "sch/hu" |
// "schhu", "bsch", "aach". An empty string returns ChannelSCHHD —
// the default ChannelCodingOn channel — and `ok = true` so config
// callers can leave the field blank. Unknown strings return
// ChannelSCHHD with `ok = false` so callers can surface the
// misconfiguration.
func ParseChannelType(s string) (ChannelType, bool) {
	switch strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(s, "/", ""), "_", "")) {
	case "":
		return ChannelSCHHD, true
	case "schhd", "bnch", "stch":
		return ChannelSCHHD, true
	case "schf":
		return ChannelSCHF, true
	case "schhu":
		return ChannelSCHHU, true
	case "bsch":
		return ChannelBSCH, true
	case "aach":
		return ChannelAACH, true
	default:
		return ChannelSCHHD, false
	}
}

// SetChannelCoding toggles the full EN 300 392-2 §8.3.1 channel
// coding chain on the Process adapter. See ChannelCodingMode for
// the trade-offs.
func (c *ControlChannel) SetChannelCoding(mode ChannelCodingMode) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.channelCoding = mode
}

// SetExpectedChannel tells the Process adapter which TETRA logical
// channel lives in each burst window. Only consulted when
// ChannelCodingMode is ChannelCodingOn; ignored otherwise. The
// default channel under ChannelCodingOn is ChannelSCHHD (the most
// common signaling carrier for cc.locked / Grant events).
func (c *ControlChannel) SetExpectedChannel(ch ChannelType) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.channelType = ch
}

// SetColourCode sets the 30-bit extended colour code the scrambler
// uses under ChannelCodingOn (low 30 bits of colourCode hold
// e(1)..e(30)). BSCH ignores this and uses 0 per §8.2.5.2.
func (c *ControlChannel) SetColourCode(colourCode uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.colourCode = colourCode & 0x3FFFFFFF
}

// ChannelCoding returns the current ChannelCodingMode. Mirrors the
// Set* family so callers (and tests) can introspect the configured
// mode without poking at unexported state.
func (c *ControlChannel) ChannelCoding() ChannelCodingMode {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.channelCoding
}

// ExpectedChannel returns the ChannelType the Process adapter
// currently expects under ChannelCodingOn.
func (c *ControlChannel) ExpectedChannel() ChannelType {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.channelType
}

// ColourCode returns the configured 30-bit extended colour code.
func (c *ControlChannel) ColourCode() uint32 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.colourCode
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
