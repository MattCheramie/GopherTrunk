package phase2

import (
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// pn44SeedFromNSB computes the PN44 descrambler seed from the
// (WACN, SystemID, ColorCode) triple carried in a Network Status
// Broadcast - Update MAC PDU per TIA-102.BBAC-1 §7.2.5 equation (5).
func pn44SeedFromNSB(nsb NetworkStatusBroadcast) uint64 {
	return framing.PN44SeedFromIdentity(nsb.WACN, nsb.SystemID, nsb.ColorCode)
}

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

	// aliasAsm reassembles multi-fragment talker-alias MAC PDUs into
	// a radio's display name. Self-synchronised (its own mutex).
	aliasAsm *TalkerAliasAssembler

	mu               sync.Mutex
	locked           bool
	strictValidation bool
	trellisMode      TrellisMode
	rsMode           RSMode
	interleaveMode   InterleaveMode
	scramblerMode    ScramblerMode
	scramblerSeed    uint64
	scramblerOffset  int
	// bandPlan accumulates IdentifierUpdate MAC PDUs so publishGrant
	// can resolve a voice grant's (ChannelID, ChannelNumber) into a
	// downlink frequency. Guarded by mu.
	bandPlan BandPlan
	// lastEncSync holds the most recently ingested Encryption Sync
	// (Algorithm ID + Key ID); publishGrant attaches it to an
	// encrypted grant so encrypted calls surface which algorithm/key
	// they use. Guarded by mu.
	lastEncSync EncryptionSync
	hasEncSync  bool
}

// TrellisMode selects how the Process adapter interprets the MAC
// PDU dibit window inside the Phase 2 traffic channel.
//
//   - TrellisOff: the adapter reads 72 dibits = 144 raw information
//     bits straight off the wire, parses 18 bytes as a MAC PDU.
//     Useful only on synthesized streams whose MAC bits aren't
//     trellis-coded; explicit opt-out for operators feeding
//     pre-stripped capture files.
//
//   - TrellisOn (default): the adapter collects 146 channel dibits (72 info
//
//   - 1 finisher transition × 2 channel dibits per transition),
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
// TrellisMode. Recognised values (case-insensitive): "" → TrellisOn
// (the new default — 146 channel dibits run through the 4-state
// ½-rate trellis decoder); "off" / "false" / "0" → TrellisOff (legacy
// 72-dibit raw-MAC-PDU path, explicit opt-out for pre-stripped
// fixtures); "on" / "true" / "1" → TrellisOn. Unknown strings return
// TrellisOn with `ok = false` so callers can surface the
// misconfiguration.
func ParseTrellisMode(s string) (TrellisMode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return TrellisOn, true
	case "off", "false", "0":
		return TrellisOff, true
	case "on", "true", "1":
		return TrellisOn, true
	default:
		return TrellisOn, false
	}
}

// RSMode selects whether the Process adapter applies the outer
// Reed-Solomon verification layer per TIA-102.BAAA-A §5.9 on top
// of the trellis-decoded MAC PDU.
//
//   - RSOff (default): the trellis-decoded 144-bit MAC PDU is parsed
//     straight into the state machine. Matches every shipped capture
//     fixture in the test suite and the historical decoder output.
//
//   - RSOn: the trellis-decoded 144-bit MAC PDU is treated as 24
//     hex symbols and verified with the RS(24, 16, 9) outer code
//     (8-symbol parity, t = 4 corrections of detection). MAC PDUs
//     whose syndromes are non-zero are dropped at the framing layer
//     before reaching the state machine.
//
// RSOn is currently the only opt-in setting that exercises the
// outer RS layer; the per-burst block interleaver schedule defined
// in TIA-102.BBAC (MAC Layer) is documented as a follow-up because
// the spec text was not available at implementation time. The
// framing primitives (EncodeRS24_*, VerifyRS24_*) are spec-correct
// per TIA-102.BAAA-A §5.9 and round-trip through unit tests.
type RSMode uint8

const (
	RSOff RSMode = iota
	RSOn
)

// SetRSMode toggles the outer Reed-Solomon verification layer on
// the trellis-decoded MAC PDU window. See RSMode for the trade-offs.
// The mode applies to every subsequent Process call; the Ingest
// entry point is unaffected (callers that pre-parse MAC PDUs
// don't go through this adapter).
func (c *ControlChannel) SetRSMode(mode RSMode) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rsMode = mode
}

// RSMode returns the current RSMode.
func (c *ControlChannel) RSMode() RSMode {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.rsMode
}

// ParseRSMode maps a config / user-facing string into an RSMode.
// Recognised values (case-insensitive): "" / "off" / "false" / "0"
// → RSOff (the default — outer RS verification is off; matches the
// historical decoder behaviour); "on" / "true" / "1" → RSOn (outer
// RS(24, 16, 9) verification on top of trellis-decoded MAC PDU).
// Unknown strings return RSOff with `ok = false` so callers can
// surface the misconfiguration.
func ParseRSMode(s string) (RSMode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "off", "false", "0":
		return RSOff, true
	case "on", "true", "1":
		return RSOn, true
	default:
		return RSOff, false
	}
}

// InterleaveMode selects whether the Process adapter applies the
// TIA-102.BBAC per-burst block deinterleaver to the collected MAC-burst
// dibits before trellis decoding.
//
//   - InterleaveOff (default): the MAC-burst dibits go straight to the
//     trellis decoder. Matches every shipped capture fixture (which
//     synthesize the burst without interleaving) and the historical
//     decoder output.
//
//   - InterleaveOn: the MAC-burst dibits are run through
//     framing.DeinterleaveMACBurst first, undoing the block interleaver
//     a real Phase 2 transmitter applies between trellis coding and the
//     channel.
type InterleaveMode uint8

const (
	InterleaveOff InterleaveMode = iota
	InterleaveOn
)

// SetInterleaveMode toggles the per-burst block deinterleaver. The mode
// applies to every subsequent Process call.
func (c *ControlChannel) SetInterleaveMode(mode InterleaveMode) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.interleaveMode = mode
}

// InterleaveMode returns the current InterleaveMode.
func (c *ControlChannel) InterleaveMode() InterleaveMode {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.interleaveMode
}

// ParseInterleaveMode maps a config / user-facing string into an
// InterleaveMode. Recognised values (case-insensitive): "" / "off" /
// "false" / "0" → InterleaveOff (the default); "on" / "true" / "1" →
// InterleaveOn. Unknown strings return InterleaveOff with ok = false.
func ParseInterleaveMode(s string) (InterleaveMode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "off", "false", "0":
		return InterleaveOff, true
	case "on", "true", "1":
		return InterleaveOn, true
	default:
		return InterleaveOff, false
	}
}

// ScramblerMode selects whether the Process adapter applies the
// PN44 descrambler per TIA-102.BBAC-1 §7.2.5 to the trellis-decoded
// MAC PDU.
//
//   - ScramblerOff (default): the trellis-decoded 144-bit MAC PDU
//     is parsed straight into the state machine. Matches every
//     shipped capture fixture in the test suite (the fixtures
//     synthesize MAC PDUs without applying scrambling) and the
//     historical decoder output.
//
//   - ScramblerOn: the trellis-decoded 144-bit MAC PDU is XORed
//     with the leading 144 bits of the PN44 scrambling sequence
//     derived from the configured (WACN_ID, System_ID, Color_Code)
//     seed before ParseMACPDU runs. SetScramblerSeed must be
//     called first so the LFSR has a real seed to clock from; a
//     zero seed maps to (2^44 - 1) per spec, which is unlikely to
//     produce useful decoding against on-air traffic.
//
//     ScramblerOn descrambles from the per-burst slot offset
//     installed via SetScramblerOffset (default 0). Operators
//     with superframe tracking can update the offset before each
//     burst arrives; operators without superframe tracking should
//     use ScramblerProbe instead.
//
//   - ScramblerProbe: the Process adapter walks each of the 12
//     spec-defined slot offsets in
//     framing.PN44SlotOffsetsOutbound (Figure 7-5), descrambles
//     with each, and accepts the first candidate whose outer
//     RS(24, 16, 9) syndromes are zero. This is the blind-probe
//     form for receivers without superframe synchronization — it
//     locks onto any of the 12 slots without external timing
//     context.
//
//     ScramblerProbe requires RSMode to be RSOn — without RS
//     verification there is no way to tell which of the 12
//     descrambled candidates is the true PDU. When RSMode is
//     RSOff, ScramblerProbe degrades silently to ScramblerOn
//     behaviour (the leading offset wins).
type ScramblerMode uint8

const (
	ScramblerOff ScramblerMode = iota
	ScramblerOn
	ScramblerProbe
)

// SetScramblerMode toggles the PN44 descrambler. See ScramblerMode
// for the trade-offs.
func (c *ControlChannel) SetScramblerMode(mode ScramblerMode) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.scramblerMode = mode
}

// ScramblerMode returns the current ScramblerMode.
func (c *ControlChannel) ScramblerMode() ScramblerMode {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.scramblerMode
}

// SetScramblerSeed installs the 44-bit PN44 seed the descrambler
// uses when ScramblerMode is ScramblerOn. Typical callers derive
// the seed via framing.PN44SeedFromIdentity(WACN, SysID, CC) from
// the values published in the system's Network Status Broadcast
// MAC message.
func (c *ControlChannel) SetScramblerSeed(seed uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.scramblerSeed = seed
}

// ScramblerSeed returns the currently-configured 44-bit PN44 seed.
func (c *ControlChannel) ScramblerSeed() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.scramblerSeed
}

// SetScramblerOffset positions the PN44 sequence at the supplied
// offset (in bits) before XORing with the MAC PDU under
// ScramblerOn. Operators with superframe tracking update the
// offset before each burst arrives (see Figure 7-5 of the spec
// for the slot offsets); operators without superframe tracking
// should use ScramblerProbe so the Process adapter walks the
// offsets automatically.
//
// Negative or out-of-range offsets are folded into the 4320-bit
// superframe period at apply time.
func (c *ControlChannel) SetScramblerOffset(offset int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.scramblerOffset = offset
}

// ScramblerOffset returns the currently-configured PN44 offset.
func (c *ControlChannel) ScramblerOffset() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.scramblerOffset
}

// ParseScramblerMode maps a config / user-facing string into a
// ScramblerMode. Recognised values (case-insensitive): "" / "off" /
// "false" / "0" → ScramblerOff (the default — no PN44 descrambling);
// "on" / "true" / "1" → ScramblerOn (XOR the trellis-decoded MAC
// PDU bits with the PN44 sequence starting at the configured
// per-burst offset); "probe" / "auto" → ScramblerProbe (try each
// of the 12 spec-defined slot offsets and accept the first that
// passes RS verification).
//
// ScramblerProbe is only meaningful when RSMode is RSOn — without
// RS verification there's no way to tell which offset produced the
// real PDU; the connector emits a warning if probe is selected
// without RSOn and degrades to ScramblerOn behaviour.
//
// Unknown strings return ScramblerOff with `ok = false`.
func ParseScramblerMode(s string) (ScramblerMode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "off", "false", "0":
		return ScramblerOff, true
	case "on", "true", "1":
		return ScramblerOn, true
	case "probe", "auto":
		return ScramblerProbe, true
	default:
		return ScramblerOff, false
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
		aliasAsm:   NewTalkerAliasAssembler(now),
	}
}

// Ingest hands one decoded MAC PDU to the state machine. Real
// captures arrive from an upstream H-DQPSK demod + TDMA superframe
// sync + Trellis FEC; tests publish PDUs directly.
//
// Network Status Broadcast - Update PDUs (opcode 0xFB) auto-update
// the descrambler seed from their (WACN, SystemID, ColorCode)
// payload per TIA-102.BBAC-1 §7.2.5 equation (5). Operators don't
// need to hand-configure the seed any more once the first NSB-
// Update lands on the bus — the connector still installs an
// initial seed from per-system config for descrambling to work
// against the very first PDUs.
func (c *ControlChannel) Ingest(p MACPDU) {
	c.mu.Lock()
	strict := c.strictValidation
	c.mu.Unlock()
	if strict && !p.Opcode.IsKnown() {
		return
	}
	if nsb, ok := p.AsNetworkStatusBroadcast(); ok {
		c.installSeedFromNSB(nsb)
	}
	if u, ok := p.AsIdentifierUpdate(); ok {
		c.mu.Lock()
		c.bandPlan.Apply(u)
		c.mu.Unlock()
	}
	if es, ok := p.AsEncryptionSync(); ok {
		c.mu.Lock()
		c.lastEncSync = es
		c.hasEncSync = true
		c.mu.Unlock()
	}
	if pg, ok := p.AsMotorolaPatchGroup(); ok {
		members := make([]uint32, len(pg.Patched))
		for i, m := range pg.Patched {
			members[i] = uint32(m)
		}
		c.publishPatch(uint32(pg.SuperGroup), members, "motorola", true)
	}
	if hr, ok := p.AsHarrisRegroup(); ok {
		c.publishPatch(uint32(hr.RegroupGroup), nil, "harris", true)
	}
	if super, ok := p.AsMotorolaPatchDelete(); ok {
		c.publishPatch(super, nil, "motorola", false)
	}
	if g, ok := p.AsGroupAffiliationResponse(); ok {
		c.publishAffiliation(g)
	}
	if u, ok := p.AsUnitRegistrationResponse(); ok {
		c.publishUnitRegistration(u)
	}
	if f, ok := p.AsTalkerAliasFragment(); ok {
		if alias, src, complete := c.aliasAsm.Add(f); complete {
			c.publishTalkerAlias(src, alias)
		}
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
		c.publishGrant(g, p.Opcode, uint32(g.GroupAddress))
	}
	if u, ok := p.AsUnitToUnitVoiceChannelGrant(); ok {
		// A unit-to-unit (private) call is still a voice grant the
		// engine must tune; map the 24-bit target unit into GroupID so
		// the recorder files it under the destination.
		c.publishGrant(GroupVoiceChannelGrant{
			ServiceOptions: u.ServiceOptions,
			ChannelID:      u.ChannelID,
			ChannelNumber:  u.ChannelNumber,
			SourceID:       u.SourceID,
		}, p.Opcode, u.TargetID)
	}
}

// installSeedFromNSB recomputes the PN44 descrambler seed from a
// freshly-parsed Network Status Broadcast - Update payload and
// updates ScramblerSeed if the new value differs from the one
// already configured. Operators with a hand-configured seed see no
// change unless the NSB-derived seed actually differs (rare in
// practice; the NSB values are stable for the lifetime of a system).
func (c *ControlChannel) installSeedFromNSB(nsb NetworkStatusBroadcast) {
	newSeed := pn44SeedFromNSB(nsb)
	c.mu.Lock()
	prev := c.scramblerSeed
	c.scramblerSeed = newSeed
	c.mu.Unlock()
	if prev != newSeed {
		c.log.Debug("p25/phase2 scrambler seed updated from NSB",
			"wacn", nsb.WACN, "sysid", nsb.SystemID, "cc", nsb.ColorCode,
			"seed", newSeed, "system", c.systemName)
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

func (c *ControlChannel) publishGrant(g GroupVoiceChannelGrant, op Opcode, groupID uint32) {
	if c.bus == nil {
		return
	}
	// Resolve the grant's channel through the band plan. Resolution is
	// best-effort: a grant that arrives before the site's first
	// IdentifierUpdate is still published (with FrequencyHz left 0, as
	// before band-plan support landed) so the event surface is
	// unchanged; the engine drops a zero-frequency grant on its own.
	freq := c.resolveFreq(g.ChannelID, g.ChannelNumber)
	// Decode the SVC_OPTIONS byte for the emergency + protected
	// (encryption) indicators. When the call is protected and an
	// Encryption Sync has been seen, attach its Algorithm ID / Key ID
	// so the recorder + API can surface which crypto the call uses.
	so := ServiceOptions(g.ServiceOptions)
	var algID uint8
	var keyID uint16
	if so.Encrypted() {
		c.mu.Lock()
		if c.hasEncSync {
			algID = c.lastEncSync.AlgorithmID
			keyID = c.lastEncSync.KeyID
		}
		c.mu.Unlock()
	}
	c.bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System:      c.systemName,
			Protocol:    "p25-phase2",
			GroupID:     groupID,
			SourceID:    g.SourceID,
			FrequencyHz: freq,
			ChannelID:   g.ChannelID,
			ChannelNum:  g.ChannelNumber,
			Encrypted:   so.Encrypted(),
			Emergency:   so.Emergency(),
			AlgorithmID: algID,
			KeyID:       keyID,
			At:          c.now(),
		},
	})
	c.log.Debug("p25/phase2 grant",
		"system", c.systemName,
		"opcode", op, "tg", groupID,
		"src", g.SourceID,
		"channel_id", g.ChannelID, "channel_num", g.ChannelNumber,
		"freq_hz", freq, "enc", so.Encrypted(), "emer", so.Emergency())
}

// publishPatch publishes an events.KindPatch for a vendor patch /
// dynamic-regroup MAC PDU so the engine can attribute later grants on
// the super-group to its member talkgroups. add=false cancels a patch.
func (c *ControlChannel) publishPatch(superGroup uint32, members []uint32, vendor string, add bool) {
	if c.bus == nil {
		return
	}
	c.bus.Publish(events.Event{
		Kind: events.KindPatch,
		Payload: trunking.Patch{
			System:     c.systemName,
			Protocol:   "p25-phase2",
			SuperGroup: superGroup,
			Members:    members,
			Vendor:     vendor,
			Add:        add,
			At:         c.now(),
		},
	})
	c.log.Debug("p25/phase2 patch",
		"system", c.systemName, "vendor", vendor,
		"super", superGroup, "members", members, "add", add)
}

// publishAffiliation publishes an events.KindAffiliation for a Group
// Affiliation Response MAC PDU — the Phase 2 counterpart of the Phase 1
// control channel's opcode-0x28 handling.
func (c *ControlChannel) publishAffiliation(g GroupAffiliationResponse) {
	if c.bus == nil {
		return
	}
	c.bus.Publish(events.Event{
		Kind: events.KindAffiliation,
		Payload: trunking.Affiliation{
			System:            c.systemName,
			Protocol:          "p25-phase2",
			SourceID:          g.TargetID,
			GroupID:           uint32(g.GroupAddress),
			AnnouncementGroup: uint32(g.AnnouncementGroup),
			Response:          trunking.AffiliationResponse(g.Response),
			At:                c.now(),
		},
	})
}

// publishUnitRegistration publishes an events.KindUnitRegistration for a
// Unit Registration Response MAC PDU.
func (c *ControlChannel) publishUnitRegistration(u UnitRegistrationResponse) {
	if c.bus == nil {
		return
	}
	c.bus.Publish(events.Event{
		Kind: events.KindUnitRegistration,
		Payload: trunking.UnitRegistration{
			System:   c.systemName,
			Protocol: "p25-phase2",
			SourceID: u.SourceID,
			WACN:     u.WACN,
			SystemID: u.SystemID,
			Response: trunking.RegistrationResponse(u.Response),
			At:       c.now(),
		},
	})
}

// publishTalkerAlias publishes an events.KindTalkerAlias once a radio's
// display name has been fully reassembled from its fragment MAC PDUs.
func (c *ControlChannel) publishTalkerAlias(sourceID uint32, alias string) {
	if c.bus == nil {
		return
	}
	c.bus.Publish(events.Event{
		Kind: events.KindTalkerAlias,
		Payload: trunking.TalkerAlias{
			System:   c.systemName,
			Protocol: "p25-phase2",
			SourceID: sourceID,
			Alias:    alias,
			At:       c.now(),
		},
	})
	c.log.Debug("p25/phase2 talker alias",
		"system", c.systemName, "src", sourceID, "alias", alias)
}

// resolveFreq looks the (channelID, channelNumber) pair up in the band
// plan, returning 0 (and logging) when no IdentifierUpdate has defined
// the channel's slot yet.
func (c *ControlChannel) resolveFreq(channelID uint8, channelNumber uint16) uint32 {
	c.mu.Lock()
	f, err := c.bandPlan.Frequency(channelID, channelNumber)
	c.mu.Unlock()
	if err != nil {
		c.log.Debug("p25/phase2 grant before identifier update",
			"id", channelID, "num", channelNumber, "err", err)
		return 0
	}
	return f
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
