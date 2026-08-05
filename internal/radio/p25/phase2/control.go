package phase2

import (
	"encoding/hex"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25"
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
	// softDecision, when set, is stamped onto every published Phase 2
	// voice grant (P25Phase2Decode.SoftDecision) so the voice composer /
	// sigfollow build their traffic-channel receiver with the
	// soft-decision demod path (issue #915). The CC's own receiver is
	// unaffected — this field only travels to the traffic-channel decode.
	softDecision bool
	// equalizer, when set, is stamped onto every published Phase 2 voice
	// grant (P25Phase2Decode.Equalizer) so the voice composer / sigfollow
	// build the traffic-channel receiver with the blind CMA equalizer
	// (issue #915). Default false leaves the symbol stream untouched.
	equalizer bool
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
	// Camped-site identity accumulated from the control channel's
	// status broadcasts: siteRFSS / siteSite come from the RFSS Status
	// Broadcast (opcode 0xFA), netWACN / netSysID / nac from the
	// Network Status Broadcast (the Color Code equals the Phase 1 NAC
	// per spec). Stamped onto grant / affiliation / registration events
	// so consumers can label them by serving site (issue #698). The
	// Phase 2 MAC has no NetworkModel — a simple last-seen cache (like
	// bandPlan / lastEncSync above) is sufficient. Guarded by mu.
	siteRFSS uint8
	siteSite uint8
	netWACN  uint32
	netSysID uint16
	nac      uint16

	// macDecoded counts MAC PDUs that reached Ingest, i.e. those that
	// cleared trellis + RS + CRC in DecodeSuperframeMACPDUs. It is the
	// Phase 2 decode-activity signal the wideband engine's diagnostics
	// gate reads via DecodedFrames; bumped from the IQ-pump goroutine but
	// exported as an atomic so other goroutines can sample it lock-free.
	macDecoded atomic.Uint64

	// macCensus counts, per MAC opcode, every PDU that reached Ingest.
	// It backs the control-channel opcode census (issue #915): the
	// per-grant DEBUG line only fires for opcodes GT already parses as a
	// grant, so it can't reveal a RID-bearing grant arriving under an
	// opcode GT drops or mis-maps — the leading suspect for the calls
	// whose source RID never populates. The census has no such blind
	// spot: it inventories the raw opcode set with a one-shot byte sample
	// so a source-less-call log pins the remaining gap on a missing/
	// mis-mapped opcode (decode-side) vs. an association gap. Guarded by mu.
	macCensus map[Opcode]uint64
}

// DecodedFrames reports the cumulative count of MAC PDUs that cleared
// FEC + CRC on this channel. It is the protocol-agnostic decode-activity
// counter the wideband engine polls to gate per-channel power logging.
func (c *ControlChannel) DecodedFrames() uint64 { return c.macDecoded.Load() }

// MACCensus returns a snapshot of the per-opcode PDU counts observed on
// this control channel since start. Used by tests and available for the
// daemon/metrics to surface the opcode inventory that disambiguates the
// #915 remaining-gap fork (missing/mis-mapped grant opcode vs association).
func (c *ControlChannel) MACCensus() map[Opcode]uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[Opcode]uint64, len(c.macCensus))
	for op, n := range c.macCensus {
		out[op] = n
	}
	return out
}

// censusObserve records p's opcode in the running census and logs a
// one-shot sample (opcode + name + known-flag + payload hex) the first
// time each distinct opcode is seen, plus a full opcode:count summary at a
// coarse cadence. Pure telemetry — it changes no decode or dispatch
// behaviour. See macCensus for why this exists (issue #915).
func (c *ControlChannel) censusObserve(p MACPDU) {
	c.mu.Lock()
	if c.macCensus == nil {
		c.macCensus = make(map[Opcode]uint64)
	}
	first := c.macCensus[p.Opcode] == 0
	c.macCensus[p.Opcode]++
	c.mu.Unlock()

	if first {
		c.log.Debug("p25/phase2 cc mac census: opcode seen",
			"system", c.systemName, "freq", c.freqHz,
			"opcode", fmt.Sprintf("0x%02X", uint8(p.Opcode)),
			"name", p.Opcode.String(), "known", p.Opcode.IsKnown(),
			"len", len(p.Payload), "payload_hex", hex.EncodeToString(p.Payload))
	}
	// Rolling frequency table, throttled to keep the log quiet. The
	// distribution (which opcodes dominate, and whether a RID-bearing
	// grant opcode is present at all) is what characterises a system.
	if n := c.macDecoded.Load(); n != 0 && n%censusSummaryEvery == 0 {
		c.log.Debug("p25/phase2 cc mac census: summary",
			"system", c.systemName, "freq", c.freqHz,
			"pdus", n, "opcodes", c.censusSummary())
	}
}

// censusSummary renders the per-opcode counts as a stable, opcode-sorted
// "0xNN(name)=count" list for a single log field.
func (c *ControlChannel) censusSummary() string {
	c.mu.Lock()
	ops := make([]Opcode, 0, len(c.macCensus))
	for op := range c.macCensus {
		ops = append(ops, op)
	}
	counts := make(map[Opcode]uint64, len(c.macCensus))
	for op, n := range c.macCensus {
		counts[op] = n
	}
	c.mu.Unlock()
	sort.Slice(ops, func(i, j int) bool { return ops[i] < ops[j] })
	var b strings.Builder
	for i, op := range ops {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "0x%02X(%s)=%d", uint8(op), op.String(), counts[op])
	}
	return b.String()
}

// censusSummaryEvery is how many Ingested MAC PDUs elapse between rolling
// census-summary log lines. Coarse enough to stay quiet on a busy CC.
const censusSummaryEvery = 2048

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
//     (8-symbol parity). MAC PDUs whose syndromes are non-zero are
//     dropped at the framing layer before reaching the state machine.
//     This is detection-only: a PDU with any residual symbol error is
//     rejected, not repaired.
//
//   - RSCorrect: the same RS(24, 16, 9) outer code, but run as a
//     bounded-distance *error corrector* (Berlekamp-Massey + Chien +
//     Forney, framing.DecodeRS24_16) that repairs up to t = 4 symbol
//     errors before the PDU is parsed, instead of dropping it. This
//     recovers the weak-frame case that leaves issue #915's
//     ground-truth replay at 0 recovered source RIDs: a MAC PDU that
//     framed and descrambled at the right phase but carries a handful
//     of symbol errors from marginal-SNR demod. Because t = 4
//     correction admits ~6e-4 of random windows (versus 2^-48 for the
//     verify-only gate), an accepted-after-correction PDU is
//     additionally required to carry a recognised opcode
//     (Opcode.IsKnown) so the ScramblerProbe offset sweep and the
//     weak-frame path cannot inject a bogus source RID (issue
//     #915 / #924). RSCorrect strictly supersets RSOn's accepts (a
//     clean codeword corrects with zero errors).
//
// The framing primitives (EncodeRS24_*, VerifyRS24_*, DecodeRS24_*)
// are spec-correct per TIA-102.BAAA-A §5.9 and round-trip through unit
// tests.
type RSMode uint8

const (
	RSOff RSMode = iota
	RSOn
	RSCorrect
)

// Enabled reports whether the outer RS(24, 16, 9) layer is exercised at
// all (verify or correct) — i.e. anything but RSOff. The ScramblerProbe
// offset sweep is only safe when RS is enabled, because it relies on the
// RS gate to reject a wrong descramble phase.
func (m RSMode) Enabled() bool { return m == RSOn || m == RSCorrect }

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
// RS(24, 16, 9) verification on top of trellis-decoded MAC PDU);
// "correct" / "fix" / "ecc" → RSCorrect (bounded-distance error
// correction of up to t = 4 symbol errors, the weak-frame recovery
// path for issue #915). Unknown strings return RSOff with `ok = false`
// so callers can surface the misconfiguration.
func ParseRSMode(s string) (RSMode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "off", "false", "0":
		return RSOff, true
	case "on", "true", "1":
		return RSOn, true
	case "correct", "fix", "ecc":
		return RSCorrect, true
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
//   - ScramblerOff: the trellis-decoded 144-bit MAC PDU is parsed
//     straight into the state machine. Matches every shipped capture
//     fixture in the test suite (the fixtures synthesize MAC PDUs
//     without applying scrambling) and the historical decoder output.
//     Available as an explicit opt-out; live on-air traffic is always
//     scrambled, so the config default (ParseScramblerMode) is
//     ScramblerOn.
//
//   - ScramblerOn (config default): the CODED MAC channel bits are XORed
//     with the PN44 scrambling sequence — derived from the configured
//     (WACN_ID, System_ID, NAC) seed — BEFORE deinterleave/trellis, per
//     TIA-102.BBAC-1 §7.2.5 (the scrambler wraps the burst "between
//     demodulation and FEC"). SetScramblerSeed must be called first so
//     the LFSR has a real seed to clock from; a zero seed maps to
//     (2^44 - 1) per spec, which is unlikely to produce useful decoding
//     against on-air traffic. Descrambling in the channel domain is the
//     issue-#915 fix: the pre-fix code XORed the post-trellis info bits,
//     a domain the trellis code cannot commute with, so RS(24,16,9)
//     could never converge on a scrambled burst (mac_rs_valid=0).
//
//     ScramblerOn descrambles from the channel-bit offset installed via
//     SetScramblerOffset (default 0); the superframe-structured path
//     supplies each sub-frame's slotChannelPN44Offset automatically.
//     Operators without superframe tracking should use ScramblerProbe.
//
//   - ScramblerProbe: descrambles the channel bits at each candidate
//     offset in the continuous 4320-bit superframe sequence and accepts
//     the first whose outer RS(24, 16, 9) syndromes are zero. This is the
//     self-aligning form for a receiver whose slot-to-sequence phase is
//     not yet pinned; the RS gate makes the sweep safe (a wrong offset
//     satisfies the syndromes with probability ≈2^-48, so garbage is
//     never accepted).
//
//     ScramblerProbe requires RSMode to be enabled (RSOn or RSCorrect)
//     — without the RS gate there is no way to tell which descrambled
//     candidate is the true PDU. Under RSCorrect the sweep additionally
//     repairs up to t = 4 symbol errors per candidate and gates the
//     accept on a recognised opcode. When RSMode is RSOff, ScramblerProbe
//     degrades silently to ScramblerOn behaviour at the configured offset.
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

// SetSoftDecision toggles whether published Phase 2 voice grants request
// the soft-decision traffic-channel demod path (issue #915). It does not
// change the CC's own receiver; it only sets the flag the composer /
// sigfollow read off the grant to build a soft-decision receiver.
func (c *ControlChannel) SetSoftDecision(on bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.softDecision = on
}

// SetEqualizer toggles whether published Phase 2 voice grants request the
// blind CMA adaptive equalizer on the traffic-channel demod path (issue
// #915). Like SetSoftDecision it does not change the CC's own receiver; it
// only sets the flag the composer / sigfollow read off the grant.
func (c *ControlChannel) SetEqualizer(on bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.equalizer = on
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
// ScramblerMode. Recognised values (case-insensitive): "" →
// ScramblerOn (the default — every on-air P25 Phase 2 MAC PDU is
// PN44 scrambled per TIA-102.BBAC-1 §7.2.5, and the production MAC
// paths supply the correct per-slot offset from superframe sync, so
// descrambling decodes live traffic out of the box); "off" /
// "false" / "0" → ScramblerOff (opt-out for the synthesized,
// unscrambled test fixtures); "on" / "true" / "1" → ScramblerOn
// (XOR the trellis-decoded MAC PDU bits with the PN44 sequence
// starting at the configured per-burst offset); "probe" / "auto" →
// ScramblerProbe (try each of the 12 spec-defined slot offsets and
// accept the first that passes RS verification).
//
// ScramblerProbe is only meaningful when RSMode is enabled (RSOn or
// RSCorrect) — without the RS gate there's no way to tell which offset
// produced the real PDU; the connector emits a warning if probe is
// selected without an RS mode and degrades to ScramblerOn behaviour.
//
// Unknown strings return ScramblerOn with `ok = false` so callers
// can surface the misconfiguration while still defaulting to the
// live-decode behaviour.
func ParseScramblerMode(s string) (ScramblerMode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return ScramblerOn, true
	case "off", "false", "0":
		return ScramblerOff, true
	case "on", "true", "1":
		return ScramblerOn, true
	case "probe", "auto":
		return ScramblerProbe, true
	default:
		return ScramblerOn, false
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
	c.macDecoded.Add(1)
	c.censusObserve(p)
	c.mu.Lock()
	strict := c.strictValidation
	c.mu.Unlock()
	if strict && !p.Opcode.IsKnown() {
		return
	}
	if nsb, ok := p.AsNetworkStatusBroadcast(); ok {
		c.installSeedFromNSB(nsb)
		// Capture the network identity so site / grant / affiliation /
		// registration events can carry WACN + System ID + NAC. The
		// Color Code equals the Phase 1 NAC per TIA-102.BBAB.
		c.mu.Lock()
		c.netWACN = nsb.WACN
		c.netSysID = nsb.SystemID
		c.nac = nsb.ColorCode
		c.mu.Unlock()
		// The NSB is the sole carrier of WACN + System ID on Phase 2 —
		// publish immediately so they reach the SiteTracker/API the instant
		// they land, rather than waiting for an RFSS Status Broadcast to
		// flush them (mirrors the Phase 1 NSB handler). Called outside the
		// lock: publishSiteUpdate takes c.mu itself.
		c.publishSiteUpdate()
	}
	if r, ok := p.AsRFSSStatusBroadcast(); ok {
		c.mu.Lock()
		c.siteRFSS = r.RFSS
		c.siteSite = r.Site
		c.mu.Unlock()
		c.publishSiteUpdate()
	}
	if u, ok := p.AsIdentifierUpdate(); ok {
		c.mu.Lock()
		c.bandPlan.Apply(u)
		c.mu.Unlock()
	}
	if es, ok := p.AsEncryptionSync(); ok && p25.AlgorithmKnown(es.AlgorithmID) {
		// Validity gate (#924): a bit-errored MAC Encryption Sync decodes to an
		// Algorithm ID outside the TIA-102 registry (the field-observed smear
		// across 0x00-0xFF, one Key ID per call). Refusing to store an out-of-set
		// sync keeps a single mis-decode from poisoning lastEncSync and being
		// attached to every subsequent encrypted grant in publishGrant — the
		// same gate the voice composer applies before publishing CallEncryption.
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

// siteIdentity returns the camped site's RFSS / Site / NAC accumulated
// from the control channel's status broadcasts, for stamping onto grant
// / affiliation / registration events (issue #698). RFSS and Site stay
// zero until an RFSS Status Broadcast has been decoded; NAC stays zero
// until a Network Status Broadcast lands.
func (c *ControlChannel) siteIdentity() (rfss, site uint8, nac uint16) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.siteRFSS, c.siteSite, c.nac
}

// publishSiteUpdate emits a KindSiteUpdate naming the site this Phase 2
// control channel is camped on, joining the decoded RFSS/Site identity
// to the tuned frequency. The SiteTracker accumulates these into GET
// /api/v1/sites (issue #698), so Phase 2 (TDMA) control channels now
// surface their sites the same way Phase 1 does. Skipped until the site
// has actually been identified.
func (c *ControlChannel) publishSiteUpdate() {
	if c.bus == nil {
		return
	}
	c.mu.Lock()
	rfss, site, wacn, sysid := c.siteRFSS, c.siteSite, c.netWACN, c.netSysID
	c.mu.Unlock()
	// Publish once any identity scalar is known. WACN/System ID come only
	// from the NSB and RFSS/Site only from the RFSS Status Broadcast, so an
	// NSB-only or RFSS-only site must still surface (mirrors Phase 1's gate).
	if rfss == 0 && site == 0 && wacn == 0 && sysid == 0 {
		return
	}
	c.bus.Publish(events.Event{
		Kind: events.KindSiteUpdate,
		Payload: trunking.SiteUpdate{
			System:           c.systemName,
			RFSSID:           rfss,
			SiteID:           site,
			ControlChannelHz: c.freqHz,
			WACN:             wacn,
			SystemID:         sysid,
			At:               c.now(),
		},
	})
}

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
	// Snapshot the per-channel FEC config so the voice composer can run
	// the same MAC dispatch on MAC subframes that interleave with voice
	// on the traffic channel — that path is how talker-alias fragments
	// reach the receiver on Phase 2 systems that do not emit them on
	// the CC (see internal/voice/composer/p25p2_voice.go).
	c.mu.Lock()
	dec := trunking.P25Phase2Decode{
		Trellis:      uint8(c.trellisMode),
		RS:           uint8(c.rsMode),
		Interleave:   uint8(c.interleaveMode),
		Scrambler:    uint8(c.scramblerMode),
		Seed:         c.scramblerSeed,
		SoftDecision: c.softDecision,
		Equalizer:    c.equalizer,
	}
	c.mu.Unlock()
	rfss, site, nac := c.siteIdentity()
	c.bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System:          c.systemName,
			Protocol:        "p25-phase2",
			GroupID:         groupID,
			SourceID:        g.SourceID,
			FrequencyHz:     freq,
			ChannelID:       g.ChannelID,
			ChannelNum:      g.ChannelNumber,
			RFSSID:          rfss,
			SiteID:          site,
			NAC:             nac,
			Encrypted:       so.Encrypted(),
			Emergency:       so.Emergency(),
			Priority:        so.Priority(),
			AlgorithmID:     algID,
			KeyID:           keyID,
			P25Phase2Decode: dec,
			At:              c.now(),
		},
	})
	args := []any{
		"system", c.systemName,
		"opcode", op, "tg", groupID,
		"src", g.SourceID,
		"channel_id", g.ChannelID, "channel_num", g.ChannelNumber,
		"freq_hz", freq, "enc", so.Encrypted(), "emer", so.Emergency(),
	}
	if so.Encrypted() && (algID != 0 || keyID != 0) {
		args = append(args, "alg", p25.FormatAlgorithm(algID), "key", keyID)
	}
	c.log.Debug("p25/phase2 grant", args...)
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
	rfss, site, nac := c.siteIdentity()
	c.bus.Publish(events.Event{
		Kind: events.KindAffiliation,
		Payload: trunking.Affiliation{
			System:            c.systemName,
			Protocol:          "p25-phase2",
			SourceID:          g.TargetID,
			GroupID:           uint32(g.GroupAddress),
			AnnouncementGroup: uint32(g.AnnouncementGroup),
			Response:          trunking.AffiliationResponse(g.Response),
			RFSSID:            rfss,
			SiteID:            site,
			NAC:               nac,
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
	rfss, site, nac := c.siteIdentity()
	c.bus.Publish(events.Event{
		Kind: events.KindUnitRegistration,
		Payload: trunking.UnitRegistration{
			System:   c.systemName,
			Protocol: "p25-phase2",
			SourceID: u.SourceID,
			WACN:     u.WACN,
			SystemID: u.SystemID,
			Response: trunking.RegistrationResponse(u.Response),
			RFSSID:   rfss,
			SiteID:   site,
			NAC:      nac,
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
