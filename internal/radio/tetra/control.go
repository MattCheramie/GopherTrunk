package tetra

import (
	"context"
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

	// fecObserver, when non-nil, receives the Viterbi/FEC correction
	// depth (the number of channel bits the §8.3.1 decode chain
	// corrected) each time a signalling burst is recovered CRC-clean.
	// It is the hook the metrics layer wires the
	// gophertrunk_tetra_viterbi_corrections histogram onto; nil ⇒ no
	// computation and zero overhead (the production default unless
	// metrics.detailed_fec is enabled).
	fecObserver func(channel string, corrections int)

	// proc is the cross-call dibit / sync state the Process
	// adapter uses (see process.go). Lazily constructed on the
	// first Process call.
	proc *processState

	// debug is set once in New when the logger is at debug level. It gates
	// the decode-health counter accumulation (bumpStat) so a production
	// (info-level) decode carries no counter overhead — same nil-guard
	// spirit as fecObserver above.
	debug bool
	// statsMu guards stats independently of mu: the counters are bumped
	// from the decode path (dispatchSlice / decodeSB / publishGrant), some
	// of which run outside mu or call methods that take mu, so a separate
	// lock avoids any re-entrancy. DrainStats reads and resets under it.
	statsMu sync.Mutex
	stats   Stats

	mu               sync.Mutex
	locked           bool
	last             LockState
	strictValidation bool
	channelCoding    ChannelCodingMode
	channelType      ChannelType
	colourCode       uint32
	colourLearned    bool
	// mainCarrier is the cell's own carrier number, learned from the
	// broadcast SYSINFO. With the tuned control-channel frequency it lets a
	// grant's carrier number resolve to Hz relative to this carrier, without a
	// configured band plan (see publishGrant).
	mainCarrier    uint16
	mainCarrierSet bool

	// pendingSoft holds the per-symbol complex differential (soft
	// information) for the next Process call, stashed by StashSoft
	// immediately before the matching dibit chunk arrives (see
	// receiver SoftSink). Process consumes it via takeStashSoft.
	pendingSoft     []complex64
	pendingSoftBase int
	pendingSoftSet  bool
}

// StashSoft records the soft per-symbol differentials for the dibit
// chunk that arrives next at Process (same baseIdx). The receiver's
// SoftSink calls this just before DibitSink → Process. diffs is copied
// since the caller reuses its buffer.
func (c *ControlChannel) StashSoft(diffs []complex64, baseIdx int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cap(c.pendingSoft) < len(diffs) {
		c.pendingSoft = make([]complex64, len(diffs))
	} else {
		c.pendingSoft = c.pendingSoft[:len(diffs)]
	}
	copy(c.pendingSoft, diffs)
	c.pendingSoftBase = baseIdx
	c.pendingSoftSet = true
}

// takeStashSoft returns the stashed soft differentials if they match the
// chunk at baseIdx with length n, then clears the stash. Returns nil
// when no matching soft data was stashed (hard-only path).
func (c *ControlChannel) takeStashSoft(baseIdx, n int) []complex64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.pendingSoftSet || c.pendingSoftBase != baseIdx || len(c.pendingSoft) != n {
		c.pendingSoftSet = false
		return nil
	}
	c.pendingSoftSet = false
	return c.pendingSoft
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
//
//   - depuncture + Viterbi + CRC-16 verify + tail strip) per
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

// ParseChannelCoding maps a config / user-facing string into a
// ChannelCodingMode. Recognised values (case-insensitive): "" →
// ChannelCodingOn (the new default — full §8.3.1 channel-coding
// chain); "off" / "false" / "0" → ChannelCodingOff (legacy 48-dibit
// raw-PDU path, explicit opt-out for pre-stripped fixtures);
// "on" / "true" / "1" → ChannelCodingOn. Unknown strings return
// ChannelCodingOn with `ok = false` so callers can surface the
// misconfiguration.
func ParseChannelCoding(s string) (ChannelCodingMode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return ChannelCodingOn, true
	case "off", "false", "0":
		return ChannelCodingOff, true
	case "on", "true", "1":
		return ChannelCodingOn, true
	default:
		return ChannelCodingOn, false
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
	c.colourLearned = true
}

// LearnColourCode records a colour code recovered at runtime from a
// decoded BSCH SYNC PDU (see process.go). It overrides the configured
// colour code only when none was configured (or it changed), so an
// operator-set colour code still wins on the first burst but a cold
// receiver auto-acquires the cell's scrambling code and every
// subsequent BNCH/SCH burst descrambles. Returns true if the stored
// value changed.
func (c *ControlChannel) LearnColourCode(ext uint32) bool {
	ext &= 0x3FFFFFFF
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.colourLearned && c.colourCode == ext {
		return false
	}
	// A configured (non-zero) colour code is authoritative; don't let a
	// marginal BSCH decode clobber it, but do fill an unset one.
	if c.colourLearned && c.colourCode != 0 {
		return false
	}
	c.colourCode = ext
	c.colourLearned = true
	// colour_code is the ETSI 6-bit colour code (low 6 bits); colour_ext is the
	// full 30-bit extended value that also seeds the scrambler.
	c.log.Info("tetra cc learned colour code from BSCH",
		"colour_code", ext&0x3F, "colour_ext", ext, "system", c.systemName)
	return true
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

// TopologyConfig is a snapshot of the TETRA single-cell identity for the hunt
// layer: the MCC/MNC/Location Area learned from MLE-SYSINFO plus the colour
// code. Neighbor-cell broadcasts are not accumulated, so identity only.
type TopologyConfig struct {
	MCC          uint16
	MNC          uint16
	LocationArea uint16
	ColourCode   uint32
}

// Topology returns the accumulated single-cell identity. Safe for concurrent
// use (reads the lock state + colour code under the control-channel mutex).
func (c *ControlChannel) Topology() TopologyConfig {
	c.mu.Lock()
	defer c.mu.Unlock()
	return TopologyConfig{
		MCC:          c.last.MCC,
		MNC:          c.last.MNC,
		LocationArea: c.last.LocationArea,
		ColourCode:   c.colourCode,
	}
}

// Stats is a snapshot of TETRA decode-health counters accumulated since the
// last DrainStats call. It is debug-only telemetry: the counters only move
// when the control channel's logger is at debug level (see New), and feed the
// throttled "tetra: decode status" console line the connector emits. All
// fields are cumulative counts over the drain window except Dibits, which is
// the symbol count used to estimate the effective baud.
type Stats struct {
	Dibits   int64 // symbols (dibits) processed — effective-baud numerator
	SBBursts int64 // synchronisation-burst candidates entering the SB decoder
	BSCHOK   int64 // BSCH blocks recovered CRC-clean
	BSCHFail int64 // SB candidates whose BSCH did not decode under any rotation
	SysInfo  int64 // BNCH SYSINFO PDUs decoded off the synchronisation burst
	SCHPDUs  int64 // signalling PDUs parsed off the normal-sync path
	Grants   int64 // voice grants published
}

// addStat adds n to a decode-health counter when debug telemetry is on. A
// no-op (no lock, no write) otherwise, so production decode is unaffected.
// field must point into c.stats.
func (c *ControlChannel) addStat(field *int64, n int64) {
	if !c.debug {
		return
	}
	c.statsMu.Lock()
	*field += n
	c.statsMu.Unlock()
}

// DrainStats returns the decode-health counters accumulated since the last
// call and resets them to zero. Stays all-zero unless the logger is at debug
// level (see New / addStat). Safe for concurrent use.
func (c *ControlChannel) DrainStats() Stats {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	s := c.stats
	c.stats = Stats{}
	return s
}

// Locked reports whether the control channel has declared lock. Read-only;
// mirrors the Topology accessor so the connector can surface lock state in the
// periodic decode-status line without poking unexported fields.
func (c *ControlChannel) Locked() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.locked
}

// Options configure a ControlChannel.
type Options struct {
	Bus         *events.Bus
	Log         *slog.Logger
	SystemName  string
	FrequencyHz uint32
	Resolver    Resolver
	Now         func() time.Time

	// FECObserver, when non-nil, is invoked with the per-burst FEC
	// correction depth (channel bits the §8.3.1 chain corrected) every
	// time a BSCH or BNCH/SCH-HD burst recovers CRC-clean. Wired by the
	// connector to the metrics layer only when metrics.detailed_fec is
	// enabled. nil ⇒ zero overhead.
	FECObserver func(channel string, corrections int)
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
		bus:         opts.Bus,
		log:         log,
		systemName:  opts.SystemName,
		freqHz:      opts.FrequencyHz,
		resolver:    opts.Resolver,
		now:         now,
		fecObserver: opts.FECObserver,
		debug:       log.Enabled(context.Background(), slog.LevelDebug),
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

// tetraChannelSpacingHz is the TETRA carrier spacing (25 kHz), used to derive a
// grant carrier's frequency relative to the cell's own carrier.
const tetraChannelSpacingHz = 25_000

// carrierFrequency derives the Hz of a TETRA carrier number relative to this
// cell's own carrier (learned from SYSINFO) at 25 kHz spacing. Returns false
// until both the cell's carrier and the tuned control frequency are known — so
// an offline replay with no centre frequency simply reports 0.
func (c *ControlChannel) carrierFrequency(carrier uint16) (uint32, bool) {
	c.mu.Lock()
	mc, set := c.mainCarrier, c.mainCarrierSet
	c.mu.Unlock()
	if !set || c.freqHz == 0 {
		return 0, false
	}
	hz := int64(c.freqHz) + (int64(carrier)-int64(mc))*tetraChannelSpacingHz
	if hz <= 0 {
		return 0, false
	}
	return uint32(hz), true
}

// learnMainCarrier records the cell's own carrier number from a SYSINFO
// broadcast, so grant carrier numbers can resolve to Hz relative to it.
func (c *ControlChannel) learnMainCarrier(carrier uint16) {
	c.mu.Lock()
	c.mainCarrier = carrier
	c.mainCarrierSet = true
	c.mu.Unlock()
}

func (c *ControlChannel) publishGrant(g VoiceGrant) {
	if c.bus == nil {
		return
	}
	c.mu.Lock()
	colourExt := c.colourCode
	c.mu.Unlock()
	freq := uint32(0)
	if c.resolver != nil {
		if hz, err := c.resolver.Frequency(g.CarrierNumber); err == nil {
			freq = hz
		} else {
			c.log.Debug("tetra: band-plan resolution failed",
				"carrier", g.CarrierNumber, "err", err)
		}
	} else if hz, ok := c.carrierFrequency(g.CarrierNumber); ok {
		// No configured band plan: derive the grant frequency relative to this
		// cell's own carrier (learned from SYSINFO) at 25 kHz TETRA spacing.
		// Exact for a same-carrier SCBS (carrier == mainCarrier ⇒ the CC freq).
		freq = hz
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
			// TETRA carries a 0-based (0..3) timeslot on the grant; the
			// trunking engine's Timeslot is 1-based with 0 reserved for
			// "not applicable", so map 0..3 → 1..4. The voice tap uses it
			// to pick the granted slot out of the 4-slot TDMA frame.
			Timeslot:       g.Timeslot + 1,
			Encrypted:      g.Encrypted,
			Emergency:      g.Emergency,
			TETRAColourExt: colourExt,
			At:             c.now(),
		},
	})
	c.addStat(&c.stats.Grants, 1)
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
