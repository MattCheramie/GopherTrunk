package phase1

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// ControlChannel consumes a stream of P25 Phase 1 dibits (already
// symbol-time-recovered and mapped via SymbolToDibit) and emits
// trunking events onto an events.Bus.
//
// Pipeline: dibit window → FSW detect → NID parse (BCH(63,16,11) +
// even-parity check) → if DUID is TSDU and the buffer holds enough
// dibits, deinterleave + Viterbi-decode the next 98-dibit TSBK block,
// validate the CRC trailer, and dispatch on the parsed opcode.
//
//   - OpIdentifierUpdate (0x3D) and OpIdentifierUpdateVUHF (0x34)
//     populate the band-plan slot for their Channel ID; 0x3D carries
//     the 700/800/900 MHz packing, 0x34 carries the VHF/UHF packing.
//   - OpGroupVoiceChannelGrant (0x00) parses the channel/group/source
//     payload, looks up the frequency in the band plan, and publishes
//     a trunking.Grant with Protocol="p25" on the bus.
//
// CCLocked / CCLost events fan out on the first corrected NID with a
// TSDU DUID. Uncorrectable NIDs and TSBK CRC failures publish
// KindDecodeError; a grant whose Channel ID has no IdentifierUpdate
// yet publishes KindDecodeError with stage="no-bandplan" so the
// metric counter surfaces the gap.
type ControlChannel struct {
	bus        *events.Bus
	log        *slog.Logger
	det        *SyncDetector
	fswTol     int // FSW correlator tolerance det was built with (≥ fswStrictTolerance)
	systemName string
	freqHz     uint32
	bandPlan   *BandPlan
	now        func() time.Time
	// carrierOffsetHz, when non-nil, reports the demod's current locked carrier
	// offset in Hz (Options.CarrierOffsetHz); publishSiteUpdate snapshots it onto
	// the site-update event so a large adjacent-channel offset is visible on
	// /api/v1/sites (issue #815). Nil leaves the published field zero.
	carrierOffsetHz func() float64
	locked          bool
	lastNAC         uint16
	// lastSiteLog dedupes the concise site-configuration log line so an
	// established site logs once and re-logs only when its identity / primary
	// control channel materially changes (see logSiteIdentity).
	lastSiteLog siteLogKey
	// lastNoHitsAt throttles the "no FSW hits" debug log so the chunk-rate
	// emission doesn't flood at debug level. See Process for the rationale.
	lastNoHitsAt time.Time

	// buf accumulates dibits across Process calls so a frame whose
	// FSW + NID + TSBK straddles IQ-chunk boundaries is still
	// assembled; bufBase is the absolute dibit index of buf[0].
	// pending holds FSW hits whose NID + TSBK has not been fully
	// buffered yet. See Process — this is the fix for issue #275,
	// where a live SDR's small IQ chunks delivered far fewer than a
	// frame's worth of dibits per call.
	buf     []uint8
	bufBase int
	pending []pendingHit

	// aliasAsm reassembles multi-fragment vendor talker-alias TSBKs
	// into a radio's display name. Self-synchronised (its own mutex).
	aliasAsm *TalkerAliasAssembler

	// netModel accumulates the site's status-broadcast TSBKs into a
	// queryable system-topology snapshot. Self-synchronised.
	netModel NetworkModel

	// pendingGrants buffers voice grants whose channel ID had no
	// IdentifierUpdate at arrival. publishVoiceGrant adds on a
	// BandPlan miss; dispatchTSBK drains after BandPlan.Apply lands a
	// new slot. See pending_grants.go.
	pendingGrants pendingGrants

	// rotations is the dibit-alphabet rotation set the NID-alignment
	// search probes. Nil/empty means RotationsAll (the legacy
	// four-rotation behaviour). The ccdecoder pipeline restricts it
	// to RotationsC4FM on the C4FM demod path so the search cannot
	// converge on a non-physical rot 1 / rot 3 miscorrection — issue
	// #275, where the post-#321 retest converged on rot=3 on a C4FM
	// site.
	rotations RotationSet

	// pduSink mirrors Options.PDUSink — the optional data-PDU dissection feed.
	pduSink func(SignalingBlock)

	// nidSearchSpan is the per-instance grid radius used by searchNID
	// in place of the package-level NIDSearchSpan constant — a
	// bisect knob for issue #275 retests where the closest-miss
	// keeps pegging at the boundary even after the ±6 default. The
	// replay subcommand exposes it via -nid-search-span so an
	// operator can re-run an IQ capture at ±12 or ±36 to rule out
	// alignment-distance as the dominant failure mode. Zero in
	// Options falls back to NIDSearchSpan, so production wiring is
	// unaffected.
	nidSearchSpan int

	// p25Phase1DemodMode is the system-level operator-set demod-mode
	// string (e.g. "cqpsk" / "c4fm"). Stamped onto every published
	// trunking.Grant so the voice composer can route the voice IQ
	// through the matching symbol-recovery path; without this a
	// CQPSK/LSM site's voice grants would land in a hardcoded C4FM
	// voice receiver and never decode (issue #356 follow-up). Empty
	// string preserves the C4FM default in the voice chain.
	p25Phase1DemodMode string

	// p25Phase2Trellis / p25Phase2RS / p25Phase2Interleave /
	// p25Phase2Scrambler carry the operator-configured Phase 2 FEC
	// modes the CC stamps onto any grant whose channel ID was
	// advertised as TDMA via opcode 0x33. Without this, a Phase 1 CC
	// could not give the voice composer enough information to run
	// the Phase 2 MAC dispatch path on the traffic channel, so
	// MMR-class hybrid sites silently lost in-call source ID +
	// alias + encryption-sync metadata (issue #376).
	p25Phase2Trellis    uint8
	p25Phase2RS         uint8
	p25Phase2Interleave uint8
	p25Phase2Scrambler  uint8
	p25Phase2Equalizer  bool
	// p25Phase2SoftDecision is stamped onto Phase 2 TDMA voice grants so
	// the voice composer builds a soft-decision traffic-channel receiver
	// (issue #915). Default false keeps the hard slicer.
	p25Phase2SoftDecision bool

	// stats is the per-frame outcome counter exposed via Stats().
	// Atomic so the daemon's API / diagnostic goroutines can read it
	// concurrently with Process. Issue #402: gives replay and any
	// future operator dashboard a per-run "of the frames the FSW
	// correlator delivered, how many cleared each acceptance gate"
	// breakdown — the steady-state shape that distinguishes "demod
	// is broken" from "demod is fine but one frame in N is corrupt".
	stats CCStats

	// diagSeen counts, per diagnostic key, how many times the key has
	// been observed, so one-shot census lines fire on the first sight
	// and payload samples are capped — letting a field tester see what
	// a site emits (including the raw bytes of opcodes we don't decode)
	// without the per-frame flood that keeps the per-opcode detail at
	// Debug. Keys: unhandled (MFID,opcode) pairs and first-seen
	// talker-alias source units (namespaced by the high byte). Written
	// only from the dispatchTSBK path, which runs on the single Process
	// goroutine (the sole writer of decode state), so no lock is needed
	// — unlike stats, this is never read off-goroutine. Issue #376 (CBD).
	diagSeen map[uint32]int
}

// CCStats is the snapshot Stats() returns. All counters are
// monotonically increasing from ControlChannel construction.
//
// NIDTrusted counts NIDs that cleared the BCH+parity gate at
// errs ≤ NIDAcceptErrs (the searchNID "trusted tier"). NIDMarginal
// counts NIDs in (NIDAcceptErrs, NIDMarginalMaxErrs] that were
// admitted only because the frame's 98-dibit TSBK ALSO Viterbi+CRC
// decoded under the same alignment. NIDFailed counts FSW hits where
// no alignment in the grid produced an acceptable NID under either
// tier — the "all 28 BCH-uncorrectable" shape from the issue-#402
// report.
//
// TSBKDecoded counts frames whose 98-dibit channel block cleared
// Viterbi + CRC and reached dispatchTSBK. TSBKTrellisFailed and
// TSBKCRCFailed split the post-NID failure mode (Viterbi diverged
// vs. trellis decoded but the trailer CRC didn't match) — both also
// surface as events.KindDecodeError on the bus, but the bus event
// is sampled by tests / Prometheus and may miss frames a synchronous
// snapshot needs.
// NetStatusSeen, RFSSStatusSeen, and AdjacentSeen count how many of the
// system-identity broadcasts decoded: the Network Status Broadcast (0x3B,
// the sole carrier of WACN + System ID), the RFSS Status Broadcast (0x3A,
// System ID), and the Adjacent Site Status Broadcast (0x3C, neighbours).
// They let the hunt / signal-lab layers explain *why* WACN/System ID are
// blank — a locked control channel with TSBKDecoded > 0 but NetStatusSeen
// == 0 means the periodic NSB simply never landed in the captured window
// (too-short dwell or marginal signal), not that the system withholds it.
type CCStats struct {
	NIDTrusted        int64
	NIDMarginal       int64
	NIDFailed         int64
	TSBKDecoded       int64
	TSBKTrellisFailed int64
	TSBKCRCFailed     int64
	NetStatusSeen     int64
	RFSSStatusSeen    int64
	AdjacentSeen      int64
	// LocRegSeen counts accepted Location Registration Responses (0x2B):
	// the high-rate RFSS/Site source. A non-zero LocRegSeen with RFSS/Site
	// resolved but NetStatusSeen == 0 is the normal shape of a system that
	// names its site via registrations but withholds the WACN-bearing NSB.
	LocRegSeen int64
}

// TSBKErrorRate returns the percentage (0..100) of decode-attempted TSBK
// blocks that failed Viterbi or CRC on this control channel, together with
// the total number of attempts (the denominator, which doubles as a
// confidence weight). attempts == 0 means no TSBK has been attempted yet —
// a fresh lock — in which case the rate is undefined and returned as 0.
// This is a frame-error rate, not a pre-FEC bit-error rate. Issue #858.
func (s CCStats) TSBKErrorRate() (ratePct float64, attempts int64) {
	failed := s.TSBKTrellisFailed + s.TSBKCRCFailed
	attempts = s.TSBKDecoded + failed
	if attempts == 0 {
		return 0, 0
	}
	return 100 * float64(failed) / float64(attempts), attempts
}

// NetworkSnapshot returns the system topology accumulated from the
// site's Network / RFSS / Secondary-CC / Adjacent-Site status TSBKs.
func (c *ControlChannel) NetworkSnapshot() NetworkConfig {
	return c.netModel.Snapshot()
}

// BandPlanSnapshot returns the IDEN_UP band-plan entries the control channel
// accumulated, in ascending channel-ID order. Combined with NetworkSnapshot it
// lets the signal-lab / hunt layers fully document a discovered P25 system.
func (c *ControlChannel) BandPlanSnapshot() []IdentifierUpdate {
	return c.bandPlan.Snapshot()
}

// LastNAC returns the NAC of the most recent corrected NID. Zero before the
// control channel locks. Used to stamp the network-configuration report.
func (c *ControlChannel) LastNAC() uint16 { return c.lastNAC }

// SystemName returns the operator-assigned system name, if any.
func (c *ControlChannel) SystemName() string { return c.systemName }

// P25Phase2FEC reports the per-system Phase 2 FEC mode codes this
// Phase 1 CC stamps onto the TDMA voice grants it publishes for hybrid
// (Phase 1 CC + Phase 2 TC) systems — trellis / RS / interleave /
// scrambler, in that order (issue #376). Exposed read-only so callers
// and tests can confirm the decode config was wired in; the wideband
// path relied on this to catch issue #882, where the modes defaulted to
// zero and every Phase 2 grant decoded with the FEC chain disabled.
func (c *ControlChannel) P25Phase2FEC() (trellis, rs, interleave, scrambler uint8) {
	return c.p25Phase2Trellis, c.p25Phase2RS, c.p25Phase2Interleave, c.p25Phase2Scrambler
}

// P25Phase1DemodMode reports the raw demod-mode string this CC stamps
// onto the voice grants it publishes (issue #935). Empty means the
// composer falls back to its own default (c4fm). Exposed read-only so
// tests can confirm a per-site override reached the grant path — the
// wideband multi-site fix relied on this to prove an LSM tap's grants
// carry the LSM mode rather than defaulting to c4fm.
func (c *ControlChannel) P25Phase1DemodMode() string { return c.p25Phase1DemodMode }

// TopologySnapshot builds the protocol-neutral topology snapshot for this
// control channel: identity, primary/secondary control channels, neighbours,
// and band plan, with every channel's downlink frequency resolved through the
// accumulated band plan. It is the single source the siglab engine and the live
// ccdecoder pipeline both surface (the latter via trunking.TopologyProvider),
// and the input to the human-readable network-configuration report. Returns nil
// when nothing usable has been observed yet.
func (c *ControlChannel) TopologySnapshot() *trunking.TopologySnapshot {
	net := c.netModel.Snapshot()
	t := &trunking.TopologySnapshot{
		SystemName: c.systemName,
		Protocol:   "p25",
		WACN:       net.WACN,
		SystemID:   uint32(net.SystemID),
		NAC:        c.lastNAC,
		RFSS:       net.RFSS,
		Site:       net.Site,
		LRA:        net.LRA,
	}
	if net.PrimaryCC.ChannelID != 0 || net.PrimaryCC.ChannelNumber != 0 {
		t.PrimaryCC = c.channelRef(net.PrimaryCC)
	}
	for _, s := range net.Secondary {
		t.Secondary = append(t.Secondary, *c.channelRef(s))
	}
	for _, n := range net.Neighbors {
		ref := trunking.TopoNeighborRef{
			RFSS:          n.RFSS,
			Site:          n.Site,
			ChannelID:     n.ChannelID,
			ChannelNumber: n.ChannelNumber,
		}
		if hz, err := c.bandPlan.Frequency(n.ChannelID, n.ChannelNumber); err == nil {
			ref.FrequencyHz = hz
		}
		t.Neighbors = append(t.Neighbors, ref)
	}
	for _, u := range c.bandPlan.Snapshot() {
		t.BandPlan = append(t.BandPlan, trunking.TopoBandPlanSlot{
			ChannelID:   u.ChannelID,
			BaseHz:      u.BaseHz,
			SpacingHz:   u.SpacingHz,
			BandwidthHz: u.BandwidthHz,
			TxOffsetHz:  u.TxOffsetHz,
			AccessTDMA:  u.AccessTDMA,
		})
	}
	if t.Empty() {
		return nil
	}
	return t
}

// channelRef resolves a (band-plan ID, channel number) pair into a
// TopoChannelRef, attaching the downlink frequency when the band plan is known.
func (c *ControlChannel) channelRef(ch Channel) *trunking.TopoChannelRef {
	ref := &trunking.TopoChannelRef{ChannelID: ch.ChannelID, ChannelNumber: ch.ChannelNumber}
	if hz, err := c.bandPlan.Frequency(ch.ChannelID, ch.ChannelNumber); err == nil {
		ref.FrequencyHz = hz
	}
	return ref
}

// Stats returns a snapshot of the per-frame outcome counters. Safe
// for concurrent calls — each counter is read with atomic.LoadInt64.
// See CCStats for the meaning of each field. Issue #402.
func (c *ControlChannel) Stats() CCStats {
	return CCStats{
		NIDTrusted:        atomic.LoadInt64(&c.stats.NIDTrusted),
		NIDMarginal:       atomic.LoadInt64(&c.stats.NIDMarginal),
		NIDFailed:         atomic.LoadInt64(&c.stats.NIDFailed),
		TSBKDecoded:       atomic.LoadInt64(&c.stats.TSBKDecoded),
		TSBKTrellisFailed: atomic.LoadInt64(&c.stats.TSBKTrellisFailed),
		TSBKCRCFailed:     atomic.LoadInt64(&c.stats.TSBKCRCFailed),
		NetStatusSeen:     atomic.LoadInt64(&c.stats.NetStatusSeen),
		RFSSStatusSeen:    atomic.LoadInt64(&c.stats.RFSSStatusSeen),
		AdjacentSeen:      atomic.LoadInt64(&c.stats.AdjacentSeen),
		LocRegSeen:        atomic.LoadInt64(&c.stats.LocRegSeen),
	}
}

// pendingHit is an FSW match awaiting enough buffered dibits to decode.
// end is the absolute dibit index of the FSW's last dibit; rot is the
// cyclic rotation the sync detector matched under.
//
// A hit also serves as the resume token for a frame whose trailing TSBK
// blocks have not all buffered yet (issue #402 multi-block decode). Once
// the NID has decoded, cont is set and the alignment the search settled
// on is cached so later Process calls finish the data unit's remaining
// blocks without re-running the FSW/NID search. nextStart and fswStart
// are absolute dibit indices so they survive buffer trimming.
type pendingHit struct {
	end int
	rot uint8

	// corroborate ⇒ this hit was found only by the wider fallback FSW
	// tolerance (distance > fswStrictTolerance), so its NID is accepted only
	// when the frame's TSBK CRC corroborates it (never on the trusted tier).
	corroborate bool

	cont       bool   // true ⇒ resume trailing TSBK blocks (NID already decoded)
	blocksDone int    // TSBK blocks already dispatched for this data unit
	nextStart  int    // absolute index of the next TSBK block to decode
	fswStart   int    // absolute index of the frame's first FSW dibit
	strip      bool   // status-symbol stripping the alignment search chose
	nac        uint16 // NAC the NID decoded to (for dispatch)
}

// FSWFallbackToleranceDefault is the wider frame-sync-word tolerance the
// opt-in soft-sync fallback uses, versus the strict default of 4 of 24 dibits.
// An FSW hit looser than the strict gate is admitted ONLY through the
// TSBK-CRC-corroborated marginal NID tier, so the looser sync cannot
// manufacture a false lock — it only extends reach into marginal-SNR captures
// whose sync word carries 5–8 symbol errors but whose NID + TSBK still decode.
// Exposed so the siglab/replay flag plumbing names one value.
const FSWFallbackToleranceDefault = 8

// fswStrictTolerance is the production FSW correlator tolerance. A hit at or
// below this distance uses the normal trusted/marginal NID gate unchanged; a
// hit above it (only reachable when a wider fallback tolerance is configured)
// is forced to corroborate via the TSBK CRC.
const fswStrictTolerance = 4

// Options configure a ControlChannel.
type Options struct {
	Bus         *events.Bus
	Log         *slog.Logger
	SystemName  string
	FrequencyHz uint32
	BandPlan    *BandPlan // optional; a new empty BandPlan is used if nil
	Now         func() time.Time
	// Rotations restricts the dibit-alphabet rotation set both the
	// FSW correlator and the NID alignment search probe. Zero value
	// (nil) keeps the legacy all-four-rotation behaviour, which is
	// correct for the CQPSK / π/4-DQPSK path. The ccdecoder pipeline
	// passes RotationsC4FM on the C4FM demod path — rot 1 / rot 3 are
	// non-physical on an FM-discriminator stream, so allowing them
	// only lets the BCH decoder miscorrect misaligned dibits into a
	// parity-valid pseudo-NID at a wrong rotation (issue #275).
	Rotations RotationSet

	// PDUSink, when non-nil, receives one SignalingBlock per TSBK the control
	// channel decodes (including CRC/trellis-failed blocks) — the data-PDU
	// dissection feed for SigLab's signaling inspector. It runs on the decode
	// hot path, so it is invoked only when set; production wiring leaves it nil
	// and pays nothing.
	PDUSink func(SignalingBlock)

	// NIDSearchSpan overrides the per-instance NID-alignment grid
	// radius. Zero falls back to the package-level NIDSearchSpan
	// constant — the default the ccdecoder pipeline uses in
	// production. The replay subcommand exposes this as a bisect knob
	// (-nid-search-span) so issue #275 retests can widen the search
	// to ±12 / ±18 / ±36 without rebuilding, to tell a span-bounded
	// failure from a demod-quality-bounded one. The two acceptance
	// tiers (BCH+parity + TSBK CRC) reject wrong alignments regardless
	// of span, so widening cannot manufacture a false lock — only let
	// the search reach a true alignment that lies past the default
	// edge.
	NIDSearchSpan int

	// P25Phase1DemodMode is the raw system-level demod-mode string
	// (e.g. "cqpsk" / "c4fm") from the system config. Stamped onto
	// every published trunking.Grant so the voice composer can route
	// the voice IQ through the matching symbol-recovery path. Empty
	// preserves the C4FM default in the voice chain — and is what the
	// existing replay / unit tests pass.
	P25Phase1DemodMode string

	// FSWFallbackTolerance opts the FSW correlator into a wider fallback
	// tolerance (see FSWFallbackToleranceDefault). Zero (the default, and what
	// the daemon pipelines pass) keeps the strict 4-of-24 behaviour byte-for-
	// byte. When set above fswStrictTolerance, FSW hits in the (strict, this]
	// band are decoded only if the frame's TSBK CRC corroborates the NID, so a
	// looser sync extends reach into marginal captures without ever
	// manufacturing a false lock. Surfaced by `replay -soft-sync` (issue #771).
	FSWFallbackTolerance int

	// P25Phase2Trellis / P25Phase2RS / P25Phase2Interleave /
	// P25Phase2Scrambler hold the per-system FEC mode the voice
	// composer's Phase 2 chain needs to decode MAC PDUs that
	// interleave with H-DQPSK voice on a Phase 2 TDMA traffic
	// channel. Encoded as the uint8 values
	// trunking.P25Phase2Decode round-trips (numerically aligned to
	// the phase2 enum constants of the same name). The ccdecoder
	// pipeline parses the matching YAML keys via phase2.Parse*Mode
	// and passes the result through; left zero means
	// Trellis=off/RS=off/Interleave=off/Scrambler=off, which matches
	// the phase2 ccdecoder's empty-YAML default for symmetry. Issue
	// #376: needed so a Phase 1 CC can publish grants that target
	// Phase 2 TDMA carriers (MMR-class systems) with FEC config the
	// voice composer can use to decode in-call source ID + alias +
	// encryption-sync PDUs.
	P25Phase2Trellis    uint8
	P25Phase2RS         uint8
	P25Phase2Interleave uint8
	P25Phase2Scrambler  uint8
	// P25Phase2SoftDecision mirrors phase2's soft-decision toggle onto
	// hybrid Phase 2 TDMA voice grants (issue #915).
	P25Phase2SoftDecision bool
	// P25Phase2Equalizer mirrors phase2's blind CMA equalizer toggle onto
	// hybrid Phase 2 TDMA voice grants (issue #915).
	P25Phase2Equalizer bool

	// CarrierOffsetHz, when non-nil, reports the demodulator's current carrier
	// offset (Hz) of the locked control carrier relative to the tuned centre.
	// publishSiteUpdate snapshots it onto SiteUpdate.ControlChannelCarrierOffsetHz
	// so an operator can spot a control lock that sits far off the configured
	// frequency — an adjacent site bleeding through (issue #815) — from the
	// /api/v1/sites surface without dropping to `capture`/`spectrum`. The
	// ccdecoder pipeline wires this to the receiver's AFCOffsetHz; nil (the
	// default, and what the decode/replay tests pass) leaves the field zero.
	CarrierOffsetHz func() float64
}

// New constructs a ControlChannel from Options. SystemName ends up on
// every trunking.Grant the channel publishes; the daemon passes it
// through from config.
func New(opts Options) *ControlChannel {
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	bp := opts.BandPlan
	if bp == nil {
		bp = &BandPlan{}
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	fswTol := fswStrictTolerance
	if opts.FSWFallbackTolerance > fswTol {
		fswTol = opts.FSWFallbackTolerance
	}
	det := NewSyncDetector(fswTol)
	if len(opts.Rotations) > 0 {
		det.SetRotations(opts.Rotations)
	}
	span := opts.NIDSearchSpan
	if span <= 0 {
		span = NIDSearchSpan
	}
	return &ControlChannel{
		bus:                   opts.Bus,
		log:                   log,
		det:                   det,
		systemName:            opts.SystemName,
		freqHz:                opts.FrequencyHz,
		bandPlan:              bp,
		now:                   now,
		carrierOffsetHz:       opts.CarrierOffsetHz,
		fswTol:                fswTol,
		aliasAsm:              NewTalkerAliasAssembler(now),
		diagSeen:              make(map[uint32]int),
		rotations:             resolveRotations(opts.Rotations),
		pduSink:               opts.PDUSink,
		nidSearchSpan:         span,
		p25Phase1DemodMode:    opts.P25Phase1DemodMode,
		p25Phase2Trellis:      opts.P25Phase2Trellis,
		p25Phase2RS:           opts.P25Phase2RS,
		p25Phase2Interleave:   opts.P25Phase2Interleave,
		p25Phase2Scrambler:    opts.P25Phase2Scrambler,
		p25Phase2SoftDecision: opts.P25Phase2SoftDecision,
		p25Phase2Equalizer:    opts.P25Phase2Equalizer,
	}
}

// NewControlChannel keeps the legacy positional constructor working —
// it's used by the existing FEC/decode tests that don't care about
// grant publication. New callers should use New(Options{...}).
func NewControlChannel(bus *events.Bus, log *slog.Logger, freqHz uint32) *ControlChannel {
	return New(Options{Bus: bus, Log: log, FrequencyHz: freqHz})
}

// LockState is the payload of CCLocked / CCLost events. It satisfies
// trunking.LockedPayload so the hunter can consume it without
// importing this package (which would create an import cycle now that
// phase1 publishes trunking.Grant events).
type LockState struct {
	FrequencyHz uint32
	NAC         uint16
	DUID        DUID
}

// LockedFrequencyHz / LockedNAC implement trunking.LockedPayload.
func (s LockState) LockedFrequencyHz() uint32 { return s.FrequencyHz }
func (s LockState) LockedNAC() uint16         { return s.NAC }

// noHitsThrottle bounds how often Process emits its "no FSW hits" debug
// log when the sync detector is finding nothing in successive chunks.
// Issue #275 surfaced because that state produced zero logs at all —
// operators couldn't tell whether the IQ pipeline was alive but
// unsynchronized or wholly silent. Throttling at 2 s keeps the signal
// visible without flooding.
const noHitsThrottle = 2 * time.Second

// p25StatusStride is the on-air dibit period of P25's interleaved
// status symbols: 35 data dibits followed by one 2-bit status symbol
// (TIA-102.BAAA — one status symbol per 70 information bits, the same
// cadence ldu.go's LDUStatusInterval encodes for voice frames).
// Counting on-air dibits from the FSW's first dibit, a status symbol
// falls wherever the index mod p25StatusStride is p25StatusStride-1.
const p25StatusStride = 36

// frameLookahead is the number of on-air dibits that must follow the
// FSW for a frame to decode: the 32-dibit NID plus the FIRST 98-dibit
// TSBK channel block — 130 data dibits — plus the 4 status symbols
// interleaved into that span at the p25StatusStride cadence. Process
// defers an FSW hit until this many dibits (plus NIDSearchSpan, so the
// +delta end of the alignment search stays in-buffer) have
// accumulated, so frame assembly no longer depends on the IQ chunking.
//
// The gate covers only the first TSBK block on purpose: a P25 data unit
// carries up to maxTSBKBlocks blocks (see parseFrame), but their count
// varies frame-to-frame, and waiting for the worst case would strand a
// short or stream-final frame that never accumulates that many dibits.
// parseFrame instead decodes the trailing blocks opportunistically,
// from whatever is already buffered past the first.
const frameLookahead = 130 + 4

// maxTSBKBlocks bounds how many 98-dibit TSBK channel blocks parseFrame
// will pull from a single FSW+NID. A P25 trunking data unit (TSDU)
// packs up to three TSBKs after the NID, the last flagged LB=1; a busy
// site fills all three, so decoding only the first (the pre-#402
// behaviour) dropped roughly two-thirds of the signalling.
const maxTSBKBlocks = 3

// tsbkBlockSpan is a safe upper bound on the buffered dibits one 98-dibit
// TSBK block occupies once the status symbols interleaved at the
// p25StatusStride cadence are counted (≈98·36/35); parseFrame requires
// this many dibits past a block's start before gathering it so a
// stream-final frame never indexes past the buffer.
const tsbkBlockSpan = 98 + 6

// NIDSearchSpan bounds the parseFrame NID-alignment search: the NID is
// probed at the FSW-derived start index plus a delta in
// [-NIDSearchSpan, +NIDSearchSpan]. ±6 dibits absorbs a compounded
// post-FSW symbol slip and status-phase fault — issue #275, where the
// field symptom was a reliably-detected FSW followed by an
// always-uncorrectable NID, and the post-#321 retest converged on the
// previous ±2 grid's positive edge every frame (the classic signature
// of a bounded search pegged at its boundary). The BCH(63,16,11) +
// even-parity + DUID acceptance gate plus the TSBK CRC corroboration
// of the marginal tier are what reject wrong alignments, so widening
// the span cannot manufacture a false lock — only let the search
// reach a true alignment that lies past the old edge.
const NIDSearchSpan = 6

// NIDAcceptErrs is the highest BCH-corrected error count searchNID
// treats as a genuine NID on the strength of the BCH + even-parity
// gate alone — the "trusted tier". BCH(63,16,11) corrects up to 11,
// but a parity-valid codeword 7+ corrections from the received word is
// as likely a miscorrection of a misaligned guess as a real noisy NID,
// so BCH+parity cannot safely admit it by itself.
//
// Issue #275: the reporter's strong-site NID never cleared this gate —
// the per-rotation probe sat at 9/10/11 errors. A NID in that 7..11
// band is not discarded outright; it is deferred to the marginal tier,
// which admits it only if the frame's TSBK also decodes (CRC) under the
// same alignment. The TSBK CRC is a far stronger validator than the
// NID's single parity bit, so a wrong alignment cannot fake it.
const NIDAcceptErrs = 6

// NIDCorroborateBudget caps how many marginal-tier NID hypotheses
// (errs in (NIDAcceptErrs, NIDMarginalMaxErrs]) searchNID will
// TSBK-corroborate per FSW hit. The grid yields at most 5×2×4
// hypotheses; only the lowest-errs few are worth a TSBK Viterbi decode,
// and the cap bounds the cost on a noisy channel that never produces a
// trusted NID.
const NIDCorroborateBudget = 8

// NIDMarginalMaxErrs is the upper bound on the marginal tier's BCH
// error count — the hard correction ceiling of BCH(63,16,11), the code
// that protects the NID. Any hypothesis that decodes successfully comes
// back with errs ≤ NIDMarginalMaxErrs; anything beyond is reported
// uncorrectable. Exposed alongside NIDSearchSpan / NIDAcceptErrs so the
// ccdecoder startup log can advertise the full accept envelope and a
// field reporter can confirm at a glance which build is running (issue
// #275: a retest cycle was already invalidated once by a stale build).
const NIDMarginalMaxErrs = 11

// Process consumes a window of dibits and runs detection/parsing.
// baseIdx is the absolute dibit index of dibits[0]. Returns the
// absolute index one past the last consumed dibit.
//
// Dibits are accumulated into an internal buffer that spans calls, so
// a frame whose FSW + NID + TSBK straddles several Process calls is
// still assembled. This matters on live hardware: a 16 KiB RTL-SDR USB
// transfer carries only ~19 P25 symbols — far short of the 154-dibit
// frame — so without cross-call buffering every FSW hit was discarded
// and the control channel never locked (issue #275).
func (c *ControlChannel) Process(dibits []uint8, baseIdx int) int {
	hits, rots, margins, next := c.det.ProcessWithMargin(nil, nil, nil, dibits, baseIdx)
	if len(hits) == 0 && len(dibits) > 0 && !c.locked {
		now := c.now()
		if now.Sub(c.lastNoHitsAt) >= noHitsThrottle {
			c.log.Debug("p25/phase1: no FSW hits in chunk",
				"system", c.systemName, "freq_hz", c.freqHz, "dibits", len(dibits))
			c.lastNoHitsAt = now
		}
	}

	// Accumulate the new dibits. The receiver hands them over in
	// contiguous, in-order batches, so buf stays a faithful copy of
	// the dibit stream from bufBase onward.
	//
	// A non-contiguous baseIdx means the dibit stream jumped (a resync or
	// capture gap): the buffered partial frame and any pending hits —
	// including a multi-block continuation waiting on its next TSBK block
	// — belong to the old stream and cannot be continued across the gap,
	// so drop them and rebase. The receiver otherwise hands dibits over in
	// contiguous, in-order batches, so this fires only on a real break.
	if len(c.buf) > 0 && baseIdx != c.bufBase+len(c.buf) {
		c.buf = c.buf[:0]
		c.pending = c.pending[:0]
	}
	if len(c.buf) == 0 {
		c.bufBase = baseIdx
	}
	c.buf = append(c.buf, dibits...)
	for i, h := range hits {
		// margin = fswTol+1 − bestMismatch, so bestMismatch = fswTol+1 − margin.
		// A hit beyond the strict budget was reachable only via the wider
		// fallback tolerance; require TSBK-CRC corroboration before it can lock.
		bestMis := c.fswTol + 1 - margins[i]
		c.pending = append(c.pending, pendingHit{
			end: h, rot: rots[i], corroborate: bestMis > fswStrictTolerance,
		})
	}

	// Parse every pending FSW hit whose first block has now been buffered,
	// and resume any frame still waiting on its trailing TSBK blocks; keep
	// the rest for a later call once more dibits land.
	kept := c.pending[:0]
	for _, ph := range c.pending {
		if ph.cont {
			// Resume the data unit's remaining TSBK blocks (issue #402).
			if c.resumeTSBKBlocks(&ph) {
				kept = append(kept, ph)
			}
			continue
		}
		// FSW ends at absolute index ph.end; the 32-dibit NID
		// immediately follows.
		nidStart := ph.end + 1 - c.bufBase
		if nidStart < 0 {
			continue // buffer already trimmed past this hit — drop it
		}
		if nidStart+frameLookahead+c.nidSearchSpan > len(c.buf) {
			kept = append(kept, ph) // not enough buffered yet
			continue
		}
		if cont, more := c.parseFrame(c.buf, nidStart, ph.rot, ph.corroborate); more {
			kept = append(kept, cont)
		}
	}
	c.pending = kept
	c.trimBuffer()
	return next
}

// parseFrame decodes the NID + TSBK of one FSW hit. buf[nidStart:]
// must hold at least frameLookahead+NIDSearchSpan dibits — the caller
// guarantees it. fswRot is the FSW-search rotation: the sync detector
// matched after adding fswRot mod 4 to each input dibit; it seeds the
// search and breaks ties so a clean frame binds deterministically.
//
// Issue #275: a reliably-detected FSW was followed by a NID that never
// BCH-decoded — the NID dibits were individually sound (the FSW would
// not correlate otherwise) but mis-aligned by the post-FSW framing.
// parseFrame therefore does not trust a single fixed read: searchNID
// probes a bounded grid of alignment hypotheses (NID start ±
// NIDSearchSpan, status symbols stripped or not, all four dibit
// rotations) and accepts the one whose NID clears the BCH(63,16,11) +
// even-parity gate with the fewest corrections — or, when no NID
// clears that gate cleanly, a marginal NID whose frame TSBK also
// decodes. Either way the accepted alignment is validated, not
// guessed, so a wrong alignment cannot realistically be locked on.
//
// parseFrame returns a continuation pendingHit (and more=true) when the
// data unit's trailing TSBK blocks have not all buffered yet, so the
// caller can resume them on a later Process call without re-searching.
func (c *ControlChannel) parseFrame(buf []uint8, nidStart int, fswRot uint8, requireCorroboration bool) (cont pendingHit, more bool) {
	best, found, diag := c.searchNID(buf, nidStart, fswRot, requireCorroboration)
	if !found {
		atomic.AddInt64(&c.stats.NIDFailed, 1)
		c.log.Debug("nid parse failed", "system", c.systemName,
			"freq_hz", c.freqHz, "diag", diag)
		c.bus.Publish(events.Event{
			Kind:    events.KindDecodeError,
			Payload: events.DecodeError{Protocol: "p25", Stage: events.StageNIDBCH},
		})
		return pendingHit{}, false
	}
	// Trusted vs. marginal tier discrimination matches the
	// "corroborated" log field below: a NID with more than
	// NIDAcceptErrs BCH corrections was only admitted because the
	// frame TSBK ALSO verified under the same alignment.
	if best.errs > NIDAcceptErrs {
		atomic.AddInt64(&c.stats.NIDMarginal, 1)
	} else {
		atomic.AddInt64(&c.stats.NIDTrusted, 1)
	}
	if best.errs > 0 {
		c.log.Debug("nid corrected", "errs", best.errs, "nac", best.nid.NAC,
			"rot", best.rot, "delta", best.delta, "strip", best.strip,
			"corroborated", best.errs > NIDAcceptErrs,
			"at_boundary", atSearchBoundary(best.delta, c.nidSearchSpan),
			"err_pattern", formatErrPattern(best.errPattern))
	}
	if best.nid.DUID != DUIDTrunkingSignaling {
		// Some non-control DUID — record but don't lock.
		c.log.Debug("non-control DUID", "duid", best.nid.DUID, "nac", best.nid.NAC)
		return pendingHit{}, false
	}
	if !c.locked || c.lastNAC != best.nid.NAC {
		c.locked = true
		c.lastNAC = best.nid.NAC
		c.bus.Publish(events.Event{
			Kind:    events.KindCCLocked,
			Payload: LockState{FrequencyHz: c.freqHz, NAC: best.nid.NAC, DUID: best.nid.DUID},
		})
		c.log.Info("control channel locked", "nac", best.nid.NAC, "freq", c.freqHz,
			"rot", best.rot, "delta", best.delta)
	}

	// Decode the data unit's TSBK blocks (issue #402: a TSDU packs up to
	// maxTSBKBlocks 98-dibit blocks after the NID, not just one). The
	// frame's first FSW dibit and the first block start are converted to
	// absolute indices so a frame whose later blocks span Process calls
	// resumes cleanly via resumeTSBKBlocks.
	ph := pendingHit{
		cont:      true,
		rot:       best.rot,
		strip:     best.strip,
		nac:       best.nid.NAC,
		fswStart:  (nidStart + best.delta - len(FrameSyncWord)) + c.bufBase,
		nextStart: best.tsbkStart + c.bufBase,
	}
	if c.resumeTSBKBlocks(&ph) {
		return ph, true
	}
	return pendingHit{}, false
}

// resumeTSBKBlocks decodes the TSBK blocks of a data unit that are fully
// buffered now, advancing ph through the sequence. It dispatches each
// block under the cached frame alignment, stops at the last block (LB),
// a decode error, or maxTSBKBlocks, and returns true when blocks remain
// undecoded for want of buffered dibits — i.e. the caller should keep ph
// pending and call again once more dibits land. Issue #402: this is what
// lets blocks 2 and 3 decode even when the receiver hands the dibit
// stream over in batches smaller than a frame.
// SignalingBlock is one decoded (or attempted) TSBK signaling block, surfaced
// to an optional PDUSink for SigLab's data-PDU inspector. It is decoder-neutral
// and carries the raw opcode/payload plus the FEC/CRC outcome and the absolute
// symbol offset, so the inspector can dissect, filter, and locate each PDU.
type SignalingBlock struct {
	Opcode     uint8
	OpcodeName string
	MFID       uint8
	NAC        uint16
	LastBlock  bool
	Protected  bool
	RawPayload []byte // the 8 opcode octets
	CRCOK      bool
	FECMetric  int // Viterbi path metric (0 = clean)
	DibitStart int // absolute dibit index of this block
	DibitLen   int // 98 for a TSBK block
}

// emitSignalingBlock hands one TSBK to the PDU sink when one is registered. It
// is a no-op (and allocation-free) when the sink is nil, so the production
// decode path is unaffected.
func (c *ControlChannel) emitSignalingBlock(t TSBK, nac uint16, metric int, crcOK bool, dibitStart int) {
	if c.pduSink == nil {
		return
	}
	c.pduSink(SignalingBlock{
		Opcode:     uint8(t.Opcode),
		OpcodeName: t.Opcode.String(),
		MFID:       t.MFID,
		NAC:        nac,
		LastBlock:  t.LB,
		Protected:  t.P,
		RawPayload: append([]byte(nil), t.Payload[:]...),
		CRCOK:      crcOK,
		FECMetric:  metric,
		DibitStart: dibitStart,
		DibitLen:   98,
	})
}

func (c *ControlChannel) resumeTSBKBlocks(ph *pendingHit) bool {
	for ph.blocksDone < maxTSBKBlocks {
		start := ph.nextStart - c.bufBase
		if start < 0 {
			return false // buffer trimmed past it (shouldn't happen) — drop
		}
		if start+tsbkBlockSpan > len(c.buf) {
			return true // next block not buffered yet — resume later
		}
		blockStart := ph.nextStart // absolute dibit index of this TSBK block
		fswStart := ph.fswStart - c.bufBase
		tsbkChannel, next := gatherFrameDibits(c.buf, start, 98, fswStart, ph.strip)
		tsbk, metric, err := DecodeTSBKChannel(rotateDibits(tsbkChannel, ph.rot))
		if err != nil {
			if errors.Is(err, CRCError) {
				atomic.AddInt64(&c.stats.TSBKCRCFailed, 1)
			} else {
				atomic.AddInt64(&c.stats.TSBKTrellisFailed, 1)
			}
			// Surface the failed block too — a protocol analyzer shows the bad
			// frames, not just the clean ones. On CRCError ParseTSBK still
			// returns the opcode/payload; on a trellis failure they are
			// best-effort.
			c.emitSignalingBlock(tsbk, ph.nac, metric, false, blockStart)
			c.log.Debug("tsbk decode failed", "err", err, "metric", metric,
				"nac", ph.nac, "block", ph.blocksDone)
			stage := events.StageTSBKTrellis
			if errors.Is(err, CRCError) {
				stage = events.StageTSBKCRC
			}
			c.bus.Publish(events.Event{
				Kind:    events.KindDecodeError,
				Payload: events.DecodeError{Protocol: "p25", Stage: stage},
			})
			return false // a failed block leaves the next block's alignment unknown
		}
		c.emitSignalingBlock(tsbk, ph.nac, metric, true, blockStart)
		c.dispatchTSBK(tsbk, ph.nac, metric)
		ph.blocksDone++
		ph.nextStart = next + c.bufBase
		if tsbk.LB {
			return false // last block in the data unit
		}
	}
	return false // reached maxTSBKBlocks
}

// nidGuess is one evaluated NID-alignment hypothesis: the NID read from
// buf starting at nidStart+delta, with interleaved status symbols
// stripped (strip) or taken contiguously, and the dibit alphabet
// rotated by rot. tsbkStart is the index the matching TSBK gather
// begins at. errPattern is a 32-entry per-dibit bit-error count of the
// received NID vs the BCH-corrected codeword (issue #275 diag —
// surfaces error clustering on the closest-miss path so timing slip,
// status-phase fault, and SNR-limited corruption are distinguishable).
type nidGuess struct {
	delta      int
	strip      bool
	rot        uint8
	nid        NID
	errs       int
	tsbkStart  int
	errPattern [32]uint8
}

// searchNID evaluates the bounded grid of NID-alignment hypotheses for
// one FSW hit and returns the winning alignment. It accepts in two
// tiers:
//
//   - Trusted: a NID that BCH-decodes with a valid even-parity bit and
//     errs ≤ NIDAcceptErrs. If any exists, the fewest-corrections one
//     wins (betterNID) — the original, fast path, unchanged.
//   - Marginal: only when no trusted NID exists, a NID with errs in
//     (NIDAcceptErrs, 11] is admitted if the frame's 98-dibit TSBK also
//     decodes (Viterbi + CRC) under the same alignment. The TSBK CRC is
//     the second validator a wrong alignment cannot fake — issue #275,
//     where a strong-site NID sat permanently at 9/10/11 BCH errors.
//
// found is false when neither tier admits a hypothesis; diag then
// summarises the closest miss for the failure log, turning a silent
// reject into a measurement (a low marginal errs that fails only TSBK
// corroboration points at demod-quality corruption, not misalignment).
// The diag includes an err_pattern=<32-char> field — the per-dibit
// bit-error count of the closest hypothesis vs its BCH-corrected
// codeword — so a reporter can see where errors cluster (one end =
// timing slip, near dibit 31 = status-phase fault, uniform = SNR
// limited) without re-running the decode.
// requireCorroboration forces every parity-valid NID candidate — even a clean
// errs ≤ NIDAcceptErrs one — through the marginal tier, where the frame's TSBK
// CRC must corroborate the alignment before it is accepted. It is set for FSW
// hits found only by the wider fallback tolerance, so a looser sync extends
// reach without ever bypassing the CRC-gated false-lock guard (issue #771).
func (c *ControlChannel) searchNID(buf []uint8, nidStart int, fswRot uint8, requireCorroboration bool) (nidGuess, bool, string) {
	var best nidGuess
	found := false
	closestMiss := -1 // lowest BCH errs of a parity-rejected near-miss
	var missAt nidGuess
	uncorrectable, tried := 0, 0
	var marginal []nidGuess // parity-valid NIDs with errs > NIDAcceptErrs

	for delta := -c.nidSearchSpan; delta <= c.nidSearchSpan; delta++ {
		start := nidStart + delta
		if start < 0 || start+frameLookahead > len(buf) {
			continue
		}
		fswStart := start - len(FrameSyncWord)
		for _, strip := range [...]bool{true, false} {
			rawNID, tsbkStart := gatherFrameDibits(buf, start, 32, fswStart, strip)
			for _, rot := range c.rotations {
				tried++
				nid, errs, pattern, err := NIDFromDibitsWithErrors(rotateDibits(rawNID, rot))
				if err != nil {
					if errs < 0 {
						uncorrectable++
					} else if closestMiss < 0 || errs < closestMiss {
						closestMiss = errs
						missAt = nidGuess{delta: delta, strip: strip, rot: rot, errs: errs, errPattern: pattern}
					}
					continue
				}
				cand := nidGuess{delta: delta, strip: strip, rot: rot,
					nid: nid, errs: errs, tsbkStart: tsbkStart, errPattern: pattern}
				if errs > NIDAcceptErrs || requireCorroboration {
					// Parity-valid but either too far from the received word for
					// BCH+parity to tell a noisy real NID from a bad alignment's
					// miscorrection, or admitted only by the wider fallback FSW
					// tolerance. Defer to the marginal tier, where the TSBK CRC
					// decides.
					marginal = append(marginal, cand)
					continue
				}
				if !found || betterNID(cand, best, fswRot) {
					best, found = cand, true
				}
			}
		}
	}
	if found {
		return best, true, ""
	}

	// No NID cleared the trusted gate. Fall back to the marginal tier:
	// try the lowest-errs hypotheses first and accept the first whose
	// TSBK corroborates the NID.
	if len(marginal) > 0 {
		sort.Slice(marginal, func(i, j int) bool {
			return betterNID(marginal[i], marginal[j], fswRot)
		})
		budget := len(marginal)
		if budget > NIDCorroborateBudget {
			budget = NIDCorroborateBudget
		}
		for _, g := range marginal[:budget] {
			if tsbkCorroborates(buf, g, nidStart) {
				return g, true, ""
			}
		}
		m := marginal[0]
		return best, false, fmt.Sprintf(
			"no NID corroborated over %d guesses; closest marginal errs=%d at delta=%d strip=%v rot=%d, TSBK uncorroborated%s; err_pattern=%s",
			tried, m.errs, m.delta, m.strip, m.rot, boundaryNote(m.delta, c.nidSearchSpan), formatErrPattern(m.errPattern))
	}
	if closestMiss >= 0 {
		return best, false, fmt.Sprintf(
			"no NID within accept gate over %d guesses; closest errs=%d at delta=%d strip=%v rot=%d%s; err_pattern=%s",
			tried, missAt.errs, missAt.delta, missAt.strip, missAt.rot, boundaryNote(missAt.delta, c.nidSearchSpan), formatErrPattern(missAt.errPattern))
	}
	return best, false, fmt.Sprintf(
		"no NID within accept gate over %d guesses; all %d BCH-uncorrectable", tried, uncorrectable)
}

// atSearchBoundary reports whether an alignment hypothesis sits at the
// edge of the NID-alignment grid. A "best" or closest-miss at the
// boundary is the textbook signature of a bounded search pegged at its
// limit — issue #275, where the post-#321 retest converged on delta=2
// (the ±2 grid's positive edge) every frame. Flagging it in the logs
// turns the next retest into a measurement: still pegged at the new
// edge → the true offset exceeds the span; interior with low errs →
// the framing was the cause and the fix worked. span is the
// per-instance grid radius (see ControlChannel.nidSearchSpan); when
// callers pass the package constant the behaviour is unchanged.
func atSearchBoundary(delta, span int) bool {
	return delta == span || delta == -span
}

// boundaryNote returns a diag-string suffix to append when a closest /
// best hypothesis sits at the search boundary; empty otherwise. span
// is the same per-instance grid radius searchNID is iterating over.
func boundaryNote(delta, span int) string {
	if atSearchBoundary(delta, span) {
		return fmt.Sprintf(" — best alignment at search boundary (±%d); true offset may exceed span", span)
	}
	return ""
}

// formatErrPattern renders a 32-entry per-dibit bit-error count as a
// 32-character ASCII string of '0'/'1'/'2' digits, dibit 0 leftmost.
//
// Reading the string traces the NID from the first dibit after the FSW
// (left) to the last dibit before the TSBK (right). Issue #275: errors
// clustered at one end signal a post-FSW symbol-timing slip; errors
// clustered around dibits 25–31 signal a status-symbol-phase fault;
// errors uniformly distributed across all 32 dibits signal SNR-limited
// demod corruption. Values above 2 are pathological (a dibit has only
// two bits), but the formatter caps at '9' rather than panic so a
// future change to the upstream counter cannot break log parsing.
func formatErrPattern(p [32]uint8) string {
	out := make([]byte, 32)
	for i, c := range p {
		if c > 9 {
			c = 9
		}
		out[i] = '0' + c
	}
	return string(out)
}

// tsbkCorroborates reports whether the 98-dibit TSBK channel block that
// follows the NID under hypothesis g decodes cleanly — Viterbi trellis
// plus CRC trailer, gathered under g's own alignment (delta / strip /
// rot). It is the second, far stronger validator the marginal-NID
// accept tier rests on: a miscorrected NID at a wrong alignment is
// followed by garbage that will not pass the TSBK CRC. nidStart is the
// FSW-derived NID start; g.delta offsets it, exactly as in parseFrame.
func tsbkCorroborates(buf []uint8, g nidGuess, nidStart int) bool {
	fswStart := nidStart + g.delta - len(FrameSyncWord)
	tsbkChannel, _ := gatherFrameDibits(buf, g.tsbkStart, 98, fswStart, g.strip)
	_, _, err := DecodeTSBKChannel(rotateDibits(tsbkChannel, g.rot))
	return err == nil
}

// betterNID reports whether candidate a should replace the current
// best b: fewer BCH corrections wins; ties break toward the most
// canonical alignment (see nidRank) so a clean frame binds to delta=0,
// status-stripped, rot=fswRot exactly as the pre-search code did.
func betterNID(a, b nidGuess, fswRot uint8) bool {
	if a.errs != b.errs {
		return a.errs < b.errs
	}
	return nidRank(a, fswRot) < nidRank(b, fswRot)
}

// nidRank scores how canonical an alignment hypothesis is — lower is
// more canonical. The FSW-reported rotation, a zero start delta and
// status-stripped framing are the expected spec alignment; anything
// else only wins on a strictly lower error count.
func nidRank(g nidGuess, fswRot uint8) int {
	r := 0
	if g.rot != fswRot {
		r += 32
	}
	d := g.delta
	if d < 0 {
		d = -d
	}
	r += 4 * d
	if !g.strip {
		r += 2
	}
	if g.delta < 0 {
		r++
	}
	return r
}

// trimBuffer drops dibits no pending hit needs any more. With no
// pending hits the whole buffer is released; otherwise everything
// before the earliest pending hit's NID is dropped — keeping buf
// bounded to roughly one frame.
func (c *ControlChannel) trimBuffer() {
	keep := len(c.buf)
	for _, ph := range c.pending {
		// A continuation must keep the buffer back to its next undecoded
		// TSBK block; a fresh hit, back to its NID start.
		s := ph.end + 1 - c.bufBase
		if ph.cont {
			s = ph.nextStart - c.bufBase
		}
		if s >= 0 && s < keep {
			keep = s
		}
	}
	if keep > 0 {
		c.buf = append(c.buf[:0], c.buf[keep:]...)
		c.bufBase += keep
	}
}

// rotateDibits returns a copy of src with the FSW-search rotation
// undone. The sync detector reported that adding `rot` mod 4 to each
// received dibit reproduced the canonical FrameSyncWord, so the
// canonical NID / TSBK dibits are recovered the same way — by adding
// `rot` mod 4. rot=0 short-circuits to avoid the copy.
func rotateDibits(src []uint8, rot uint8) []uint8 {
	if rot == 0 {
		return src
	}
	out := make([]uint8, len(src))
	for i, d := range src {
		out[i] = (d + rot) & 3
	}
	return out
}

// gatherDataDibits copies count P25 data dibits out of buf starting at
// index start, skipping the status symbols interleaved at the
// p25StatusStride cadence. fswStart is the index of the frame's first
// FSW dibit, against which the status-symbol phase is measured (it may
// be negative if the buffer was trimmed past the FSW — only the
// distance from it matters, and start is always well past it). It
// returns the gathered dibits and the index one past the last dibit
// consumed, so the next field gathers from there. The caller
// guarantees buf holds enough dibits (see frameLookahead).
func gatherDataDibits(buf []uint8, start, count, fswStart int) ([]uint8, int) {
	out := make([]uint8, 0, count)
	i := start
	for len(out) < count {
		if (i-fswStart)%p25StatusStride != p25StatusStride-1 {
			out = append(out, buf[i])
		}
		i++
	}
	return out, i
}

// gatherFrameDibits copies count data dibits out of buf starting at
// index start. When strip is true the status symbols interleaved at
// the p25StatusStride cadence (phase measured from fswStart) are
// skipped via gatherDataDibits; when false the dibits are taken
// contiguously — the parseFrame alignment search tries both, since a
// status-phase error and a missing status symbol are among issue
// #275's candidate framing faults. It returns the gathered dibits and
// the index one past the last dibit consumed.
func gatherFrameDibits(buf []uint8, start, count, fswStart int, strip bool) ([]uint8, int) {
	if strip {
		return gatherDataDibits(buf, start, count, fswStart)
	}
	out := make([]uint8, count)
	copy(out, buf[start:start+count])
	return out, start + count
}

// InjectControlStatusSymbols interleaves P25 status symbols into a
// contiguous control-channel dibit stream (FSW + NID + TSBK …),
// producing the on-air dibit stream a real transmitter emits: one
// status symbol after every p25StatusStride-1 data dibits (TIA-102.BAAA
// — one status symbol per 70 information bits). It is the inverse of
// the status-symbol stripping parseFrame applies via gatherDataDibits,
// and exists to build spec-faithful synthetic streams for tests — the
// receiver harness, integration tests — that would otherwise feed the
// decoder an unrealistic status-free stream.
func InjectControlStatusSymbols(stream []uint8) []uint8 {
	const dataRun = p25StatusStride - 1
	out := make([]uint8, 0, len(stream)+len(stream)/dataRun+1)
	for i, d := range stream {
		out = append(out, d)
		if i%dataRun == dataRun-1 {
			// Cycle the status value through all four symbols so a
			// test cannot pass by depending on a fixed status value —
			// the receiver must skip them by position.
			out = append(out, uint8((i/dataRun)&3))
		}
	}
	return out
}

// dispatchTSBK routes a successfully-CRC'd TSBK to the right opcode
// handler. Unknown opcodes are still useful for diagnostics — they're
// logged at debug but not republished, since they're the bulk of what
// a busy site emits and would drown signal in noise.
func (c *ControlChannel) dispatchTSBK(t TSBK, nac uint16, metric int) {
	atomic.AddInt64(&c.stats.TSBKDecoded, 1)
	// Manufacturer-specific TSBKs are decoded in the vendor's opcode
	// namespace (Motorola patch/regroup, Harris regroup, talker alias)
	// — see tsbk_vendor.go. The band-plan and network/site/secondary
	// broadcast opcodes (IDEN, RFSS/NSB/adjacent/secondary) live in the
	// standard TIA namespace on every vendor's control channel — Motorola
	// and Harris never redefine them — so they are decoded via the standard
	// switch even when the MFID marks the block vendor-specific. Some
	// Motorola sites emit exactly these under MFID 0x90, which is why their
	// WACN/SysID/neighbours otherwise never surfaced (matches OP25/boatbod).
	if t.IsVendorMFID() && !isStandardBroadcastOpcode(t.Opcode) {
		c.dispatchVendorTSBK(t, nac)
		return
	}
	if t.IsVendorMFID() && c.diagFirst(diagVendorBroadcast|uint32(t.Opcode)) {
		c.log.Info("p25: standard broadcast opcode under vendor MFID",
			"system", c.systemName, "mfid", t.MFID, "opcode", t.Opcode, "nac", nac)
	}
	switch t.Opcode {
	case OpIdentifierUpdate:
		u := ParseIdentifierUpdate(t.Payload)
		c.bandPlan.Apply(u)
		c.log.Debug("p25: identifier update",
			"nac", nac, "id", u.ChannelID,
			"base_hz", u.BaseHz, "spacing_hz", u.SpacingHz,
			"tx_offset_hz", u.TxOffsetHz)
		c.drainPendingGrants(u.ChannelID, nac)
		// A newly-applied band lets a neighbour on it resolve its downlink
		// frequency; republish topology so the site table / API pick it up
		// (symmetric with drainPendingGrants for the grant side). No-op until
		// the site is otherwise identified (publishSiteUpdate's guard).
		c.publishSiteUpdate()
	case OpIdentifierUpdateVUHF:
		u := ParseIdentifierUpdateVUHF(t.Payload)
		c.bandPlan.Apply(u)
		c.log.Debug("p25: identifier update (VUHF)",
			"nac", nac, "id", u.ChannelID,
			"base_hz", u.BaseHz, "spacing_hz", u.SpacingHz,
			"tx_offset_hz", u.TxOffsetHz,
			"bandwidth_hz", u.BandwidthHz)
		c.drainPendingGrants(u.ChannelID, nac)
		c.publishSiteUpdate()
	case OpIdentifierUpdateTDMA:
		u := ParseIdentifierUpdateTDMA(t.Payload)
		c.bandPlan.Apply(u)
		c.log.Debug("p25: identifier update (TDMA)",
			"nac", nac, "id", u.ChannelID,
			"base_hz", u.BaseHz, "spacing_hz", u.SpacingHz,
			"tx_offset_hz", u.TxOffsetHz,
			"bandwidth_hz", u.BandwidthHz)
		c.drainPendingGrants(u.ChannelID, nac)
		c.publishSiteUpdate()
	case OpGroupVoiceChannelGrant:
		c.publishGroupGrant(ParseGroupVoiceChannelGrant(t.Payload), nac)
	case OpGroupVoiceChannelUpdate:
		u := ParseGroupVoiceChannelUpdate(t.Payload)
		c.publishVoiceGrant(voiceGrant{
			groupID: uint32(u.GroupAddressA), channelID: u.ChannelAID,
			channelNumber: u.ChannelANumber,
		}, nac)
		if u.GroupAddressB != 0 {
			c.publishVoiceGrant(voiceGrant{
				groupID: uint32(u.GroupAddressB), channelID: u.ChannelBID,
				channelNumber: u.ChannelBNumber,
			}, nac)
		}
	case OpGroupVoiceChannelUpdateExpl:
		c.publishGroupGrant(ParseGroupVoiceChannelUpdateExplicit(t.Payload), nac)
	case OpUnitToUnitVoiceChannelGrant:
		g := ParseUnitToUnitVoiceChannelGrant(t.Payload)
		c.publishVoiceGrant(voiceGrant{
			groupID: g.TargetID, sourceID: g.SourceID,
			channelID: g.ChannelID, channelNumber: g.ChannelNumber,
			individual: true,
		}, nac)
	case OpUnitToUnitAnswerRequest:
		c.publishUnitToUnitRequest(ParseUnitToUnitAnswerRequest(t.Payload), nac)
	case OpTelephoneInterconnectGrant:
		g := ParseTelephoneInterconnectGrant(t.Payload)
		c.publishVoiceGrant(voiceGrant{
			groupID: g.TargetID, channelID: g.ChannelID,
			channelNumber: g.ChannelNumber, serviceOptions: g.ServiceOptions,
			individual: true,
		}, nac)
	case OpSNDCPDataChannelGrant:
		g := ParseSNDCPDataChannelGrant(t.Payload)
		c.publishVoiceGrant(voiceGrant{
			groupID: g.TargetID, channelID: g.ChannelID,
			channelNumber: g.ChannelNumber, serviceOptions: g.ServiceOptions,
			dataCall: true, individual: true,
		}, nac)
	case OpNetworkStatusBroadcast:
		atomic.AddInt64(&c.stats.NetStatusSeen, 1)
		c.logStatusBroadcast(t, nac)
		c.netModel.ApplyNetworkStatus(ParseNetworkStatusBroadcast(t.Payload))
		// 0x3B is the sole carrier of WACN — publish immediately so it
		// reaches the SiteTracker/API the instant it lands, rather than
		// waiting for a later RFSS/adjacent broadcast to flush the model.
		c.publishSiteUpdate()
	case OpRFSSStatusBroadcast:
		atomic.AddInt64(&c.stats.RFSSStatusSeen, 1)
		c.logStatusBroadcast(t, nac)
		c.netModel.ApplyRFSSStatus(ParseRFSSStatusBroadcast(t.Payload))
		c.publishSiteUpdate()
	case OpSecondaryControlChannel:
		c.netModel.ApplySecondaryControlChannel(ParseSecondaryControlChannelBroadcast(t.Payload))
	case OpSecondaryControlChannelExpl:
		c.netModel.ApplySecondaryControlChannelExplicit(ParseSecondaryControlChannelBroadcastExplicit(t.Payload))
	case OpAdjacentSiteStatusBroadcast:
		atomic.AddInt64(&c.stats.AdjacentSeen, 1)
		c.logStatusBroadcast(t, nac)
		c.netModel.ApplyAdjacentSite(ParseAdjacentSiteStatusBroadcast(t.Payload))
		// On systems that never emit 0x3B/0x3A, the adjacent broadcast is
		// the only System ID source — publish so it reaches the SiteTracker/API.
		c.publishSiteUpdate()
	case OpGroupAffiliationResponse:
		c.publishAffiliation(ParseGroupAffiliationResponse(t.Payload), nac)
	case OpUnitRegistrationResponse:
		c.publishUnitRegistration(ParseUnitRegistrationResponse(t.Payload), nac)
	case OpLocationRegistrationResponse:
		// The location registration response names the camped site's
		// RFSS/Site and is emitted far more often than the RFSS Status
		// Broadcast (0x3A) — frequently the only practical RFSS/Site source.
		atomic.AddInt64(&c.stats.LocRegSeen, 1)
		c.netModel.ApplyLocationRegistration(ParseLocationRegistrationResponse(t.Payload))
		c.publishSiteUpdate()
	default:
		// Census + raw-payload sample — a standard opcode we don't
		// dispatch could be (or carry) the alias transport a given site
		// uses; the bulk per-frame detail stays at Debug. Issue #376 (CBD).
		c.logUnhandledTSBK(t, nac)
		c.log.Debug("tsbk decoded",
			"opcode", t.Opcode, "lb", t.LB, "metric", metric, "nac", nac)
	}
}

// dispatchVendorTSBK routes a manufacturer-specific TSBK (Motorola or
// Harris MFID) through the vendor accessors in tsbk_vendor.go and
// publishes the trunking event it carries.
func (c *ControlChannel) dispatchVendorTSBK(t TSBK, nac uint16) {
	if pg, ok := t.AsMotorolaPatchGroup(); ok {
		members := make([]uint32, len(pg.Patched))
		for i, m := range pg.Patched {
			members[i] = uint32(m)
		}
		c.publishPatch(uint32(pg.SuperGroup), members, "motorola", true)
		return
	}
	if super, ok := t.AsMotorolaPatchDelete(); ok {
		c.publishPatch(uint32(super), nil, "motorola", false)
		return
	}
	if hr, ok := t.AsHarrisRegroup(); ok {
		c.publishPatch(uint32(hr.RegroupGroup), nil, "harris", true)
		return
	}
	if g, ok := t.AsMotorolaPatchGroupChannelGrant(); ok {
		// Patch-group voice channel grant — mirrors the standard
		// OpGroupVoiceChannelGrant path. The 0x02 form carries the
		// source RID + service-options encryption bit, so this backfills
		// the src=0/enc=false patch-group calls MMR/CBD was dropping
		// (issue #376); super-group attribution is handled downstream.
		c.publishGroupGrant(g, nac)
		return
	}
	if u, ok := t.AsMotorolaPatchGroupChannelGrantUpdate(); ok {
		// Two (channel, super-group) activity pairs — mirrors the
		// standard OpGroupVoiceChannelUpdate path. Issue #376.
		c.publishVoiceGrant(voiceGrant{
			groupID: uint32(u.GroupAddressA), channelID: u.ChannelAID,
			channelNumber: u.ChannelANumber,
		}, nac)
		if u.GroupAddressB != 0 {
			c.publishVoiceGrant(voiceGrant{
				groupID: uint32(u.GroupAddressB), channelID: u.ChannelBID,
				channelNumber: u.ChannelBNumber,
			}, nac)
		}
		return
	}
	if f, ok := t.AsTalkerAliasFragment(); ok {
		// Surface the first fragment seen per source at Info so a field
		// tester can tell aliases are arriving on the CC even when the
		// multi-fragment set never completes (e.g. CRC loss dropping a
		// block before the 10s reassembly window closes — issue #376 CBD).
		if c.diagFirst(diagAliasSrc | (f.SourceID & 0xFFFFFF)) {
			c.log.Info("p25: cc talker alias fragment",
				"system", c.systemName, "src", f.SourceID,
				"block", f.BlockIndex, "of", f.BlockCount)
		}
		if alias, src, done := c.aliasAsm.Add(f); done {
			c.publishTalkerAlias(src, alias)
		}
		return
	}
	// Motorola (MFID 0x90) Traffic Channel ID (0x05) and System Loading (0x09)
	// are emitted continuously on real systems. SDRtrunk names them but does not
	// field-decode them (its MotorolaTrafficChannel / ChannelLoading classes are
	// hex-only stubs) and OP25 ignores them; neither carries neighbour or
	// secondary-CC data. So we name and capture the raw payload for the record
	// but decode/publish NOTHING into the system map — a guessed layout would
	// inject phantom data (a Motorola 0x05 decoded with the standard UU_ANS_REQ
	// layout already produced garbage: src=0, target=0x3C0000). See
	// docs/specs/p25-motorola-opcodes.md.
	if t.MFID == MFIDMotorola && (t.Opcode == OpMotorolaTrafficChannelID || t.Opcode == OpMotorolaSystemLoading) {
		c.logVendorProbe(t, nac)
		return
	}
	// Unrecognised vendor opcode: census + raw-payload sample so a field
	// test can name and reverse any alias-bearing transport we don't yet
	// decode, while the per-frame detail stays at Debug.
	c.logUnhandledTSBK(t, nac)
	c.log.Debug("p25: vendor tsbk", "mfid", t.MFID, "opcode", t.Opcode, "nac", nac)
}

// diagnostic key namespaces for diagSeen — the high byte separates
// categories so an unhandled-opcode key and an alias-source key never
// collide. See ControlChannel.diagSeen.
const (
	diagUnhandledTSBK   uint32 = 1 << 24
	diagAliasSrc        uint32 = 2 << 24
	diagVendorBroadcast uint32 = 3 << 24
	diagVendorProbe     uint32 = 4 << 24
	diagStatusBroadcast uint32 = 5 << 24
)

// isStandardBroadcastOpcode reports whether op is one of the band-plan or
// network/site/secondary broadcast opcodes that are always standard TIA
// messages regardless of MFID — so they are decoded via the standard dispatch
// even when a vendor MFID marks the block manufacturer-specific. Grant opcodes
// are deliberately excluded: opcode 0x00 under MFID 0x90 is a Motorola patch,
// not a group voice grant, so it must stay on the vendor path.
func isStandardBroadcastOpcode(op Opcode) bool {
	switch op {
	case OpIdentifierUpdate, OpIdentifierUpdateVUHF, OpIdentifierUpdateTDMA,
		OpSecondaryControlChannel, OpSecondaryControlChannelExpl,
		OpRFSSStatusBroadcast,
		OpNetworkStatusBroadcast, OpAdjacentSiteStatusBroadcast:
		return true
	default:
		return false
	}
}

// maxUnhandledSamples caps how many raw payloads are logged per distinct
// (MFID,opcode) so a busy CC stays bounded; 8 is enough to catch a full
// multi-block alias sequence on a candidate opcode. Issue #376 (CBD).
const maxUnhandledSamples = 8

// diagFirst reports whether key is being seen for the first time,
// recording it so subsequent calls return false. Single-writer
// (Process goroutine) — see diagSeen.
func (c *ControlChannel) diagFirst(key uint32) bool {
	n := c.diagSeen[key]
	c.diagSeen[key] = n + 1
	return n == 0
}

// logUnhandledTSBK emits the field diagnostics for a TSBK we don't
// dispatch: one Info census line per distinct (MFID,opcode) — with the
// numeric opcode, since Opcode.String() mislabels vendor opcodes with
// standard names — plus up to maxUnhandledSamples raw-payload lines so a
// field tester can capture the bytes of a candidate alias transport and
// cross-check them against known RIDs / alias strings. Issue #376 (CBD).
func (c *ControlChannel) logUnhandledTSBK(t TSBK, nac uint16) {
	key := diagUnhandledTSBK | uint32(t.MFID)<<8 | uint32(t.Opcode)
	n := c.diagSeen[key]
	c.diagSeen[key] = n + 1
	if n == 0 {
		c.log.Info("p25: unhandled tsbk",
			"mfid", t.MFID, "opcode", fmt.Sprintf("0x%02X", uint8(t.Opcode)),
			"name", t.Opcode, "nac", nac)
	}
	if n < maxUnhandledSamples {
		c.log.Info("p25: unhandled tsbk payload",
			"mfid", t.MFID, "opcode", fmt.Sprintf("0x%02X", uint8(t.Opcode)),
			"lb", t.LB, "payload", fmt.Sprintf("%x", t.Payload[:]), "nac", nac)
	}
}

// maxStatusBroadcastSamples caps how many raw payloads are dumped per distinct
// status-broadcast opcode. These broadcasts (adjacent 0x3C, network 0x3B, RFSS
// 0x3A) ARE decoded and field-decoded — but the parsed result alone can't prove
// the on-air byte layout when a field tester reports a neighbour/identity
// mismatch. Pinned a little higher than maxUnhandledSamples so a correlation set
// of the first several frames (cross-checked against a reference decoder's
// known neighbours) is enough to confirm or refute a layout theory offline.
const maxStatusBroadcastSamples = 16

// logStatusBroadcast dumps the raw payload and the fields we decoded from a
// HANDLED status broadcast, sampled per distinct opcode. Unlike logUnhandledTSBK
// this changes nothing about parsing or the network model — it is pure
// instrumentation so the bytes that produced a given neighbour/identity are
// captured in the log (and the JSONL event sink) for offline verification. The
// CFVA nibble (byte 1 high nibble: Conventional/Failsoft/Valid/Active) is logged
// for the adjacent broadcast because the "Valid" bit gates whether the channel
// field is meaningful.
func (c *ControlChannel) logStatusBroadcast(t TSBK, nac uint16) {
	key := diagStatusBroadcast | uint32(t.Opcode)
	n := c.diagSeen[key]
	c.diagSeen[key] = n + 1
	if n >= maxStatusBroadcastSamples {
		return
	}
	p := t.Payload
	// Raw payload is always logged — it is the ground truth a field tester
	// cross-checks against a reference decoder. The decoded attrs differ by
	// opcode: the Network Status Broadcast packs WACN across bytes 1-3 and has
	// no RFSS/Site, whereas the adjacent (0x3C) and RFSS (0x3A) broadcasts share
	// the LRA/SysID/RFSS/Site/channel layout.
	attrs := []any{
		"opcode", fmt.Sprintf("0x%02X", uint8(t.Opcode)),
		"name", t.Opcode, "lb", t.LB,
		"payload", fmt.Sprintf("%x", p[:]),
		"lra", p[0],
		"chan", fmt.Sprintf("%X.%03X", p[5]>>4, uint16(p[5]&0x0F)<<8|uint16(p[6])),
	}
	if t.Opcode == OpNetworkStatusBroadcast {
		attrs = append(attrs,
			"wacn", fmt.Sprintf("%05X", uint32(p[1])<<12|uint32(p[2])<<4|uint32(p[3])>>4),
			"sysid", fmt.Sprintf("%03X", uint16(p[3]&0x0F)<<8|uint16(p[4])))
	} else {
		attrs = append(attrs,
			"cfva", p[1]>>4, "rfss", p[3], "site", p[4],
			"sysid", fmt.Sprintf("%03X", uint16(p[1]&0x0F)<<8|uint16(p[2])))
	}
	attrs = append(attrs, "nac", nac, "sample", n+1, "max", maxStatusBroadcastSamples)
	c.log.Info("p25: status broadcast payload", attrs...)
}

// maxVendorProbeSamples bounds how many raw payloads logVendorProbe dumps per
// distinct (MFID,opcode). Larger than maxUnhandledSamples because these are
// deliberate reverse-engineering targets — a correlation set of a few dozen
// frames (paired with the site's known neighbours / secondary CCs from a
// reference decoder) is what reverses the field layout.
const maxVendorProbeSamples = 64

// logVendorProbe records a candidate-but-undecoded vendor TSBK for offline
// analysis: it dumps the full raw payload at Info (up to maxVendorProbeSamples
// per opcode) and does nothing else — no parse, no event, no map mutation — so
// an unconfirmed layout cannot leak phantom data into the system map. See
// docs/specs/p25-motorola-opcodes.md for the reverse-engineering status.
func (c *ControlChannel) logVendorProbe(t TSBK, nac uint16) {
	key := diagVendorProbe | uint32(t.MFID)<<8 | uint32(t.Opcode)
	n := c.diagSeen[key]
	c.diagSeen[key] = n + 1
	if n < maxVendorProbeSamples {
		c.log.Info("p25: motorola opcode (named, not field-decoded — collecting for analysis)",
			"mfid", t.MFID, "opcode", fmt.Sprintf("0x%02X", uint8(t.Opcode)),
			"name", motorolaProbeLabel(t.Opcode),
			"lb", t.LB, "payload", fmt.Sprintf("%x", t.Payload[:]),
			"nac", nac, "sample", n+1, "max", maxVendorProbeSamples)
	}
}

// motorolaProbeLabel maps the probed Motorola opcodes to SDRtrunk's names so
// the diagnostic log reads in known terms rather than a bare opcode number.
func motorolaProbeLabel(op Opcode) string {
	switch op {
	case OpMotorolaTrafficChannelID:
		return "MOTOROLA_OSP_TRAFFIC_CHANNEL_ID"
	case OpMotorolaSystemLoading:
		return "MOTOROLA_OSP_SYSTEM_LOADING"
	default:
		return "MOTOROLA_OSP_UNKNOWN"
	}
}

// publishPatch publishes an events.KindPatch for a vendor patch /
// dynamic-regroup TSBK so the engine can attribute later grants on the
// super-group to its member talkgroups. add=false cancels a patch.
func (c *ControlChannel) publishPatch(superGroup uint32, members []uint32, vendor string, add bool) {
	c.bus.Publish(events.Event{
		Kind: events.KindPatch,
		Payload: trunking.Patch{
			System:     c.systemName,
			Protocol:   "p25",
			SuperGroup: superGroup,
			Members:    members,
			Vendor:     vendor,
			Add:        add,
			At:         c.now(),
		},
	})
	c.log.Debug("p25: patch",
		"system", c.systemName, "vendor", vendor,
		"super", superGroup, "members", members, "add", add)
}

// publishTalkerAlias publishes an events.KindTalkerAlias once a radio's
// display name has been fully reassembled from its fragment TSBKs.
func (c *ControlChannel) publishTalkerAlias(sourceID uint32, alias string) {
	c.bus.Publish(events.Event{
		Kind: events.KindTalkerAlias,
		Payload: trunking.TalkerAlias{
			System:   c.systemName,
			Protocol: "p25",
			SourceID: sourceID,
			Alias:    alias,
			At:       c.now(),
		},
	})
	c.log.Info("p25: cc talker alias",
		"system", c.systemName, "src", sourceID, "alias", alias)
}

// voiceGrant is the protocol-internal shape publishVoiceGrant resolves
// and publishes. It generalises the several P25 grant TSBKs (group,
// group-update, unit-to-unit, telephone, SNDCP data) over one path.
type voiceGrant struct {
	groupID        uint32 // talkgroup, or destination unit / phone target
	sourceID       uint32
	channelID      uint8
	channelNumber  uint16
	serviceOptions uint8
	dataCall       bool
	// individual marks a unit-to-unit / telephone / SNDCP-data grant whose
	// groupID is a 24-bit destination unit, not a 16-bit talkgroup. Carried
	// onto trunking.Grant.Individual so discovery never lists it as a TG.
	individual bool
}

// publishVoiceGrant resolves a grant's channel through the band plan
// and publishes a trunking.Grant on the bus. If the channel ID hasn't
// been seen yet, a `decode.error` with stage="no-bandplan" is
// published instead — the engine can't do anything with a
// zero-frequency grant, and surfacing this lets operators see when a
// site's IdentifierUpdate cadence is too slow for their capture.
func (c *ControlChannel) publishVoiceGrant(g voiceGrant, nac uint16) {
	freq, err := c.bandPlan.Frequency(g.channelID, g.channelNumber)
	if err != nil {
		c.log.Debug("p25: grant before identifier update",
			"nac", nac, "id", g.channelID, "num", g.channelNumber, "err", err)
		c.pendingGrants.add(g.channelID, g, nac, c.now())
		c.bus.Publish(events.Event{
			Kind:    events.KindDecodeError,
			Payload: events.DecodeError{Protocol: "p25", Stage: events.StageNoBandPlan},
		})
		return
	}
	so := ServiceOptions(g.serviceOptions)
	// Hybrid Phase 1 CC + Phase 2 TC sites (issue #376, MMR): when the
	// granted channel ID was advertised as TDMA via opcode 0x33, route
	// the grant to the voice composer's Phase 2 chain so it can decode
	// the MAC PDUs that interleave with H-DQPSK voice on the traffic
	// channel — source-ID + encryption-sync + talker-alias backfill.
	// Without this, every grant landed in the Phase 1 voice chain and
	// the Phase 2 MAC dispatch path PR #409 added was dead code on
	// MMR-class systems.
	protocol := "p25"
	// Snapshot the site topology once: the camped site's RFSS/Site is
	// stamped onto every grant (issue #698) and the WACN/SystemID feeds
	// the Phase 2 PN44 seed below.
	net := c.netModel.Snapshot()
	var p2dec trunking.P25Phase2Decode
	if c.bandPlan.IsTDMA(g.channelID) {
		protocol = "p25-phase2"
		seed := framing.PN44SeedFromIdentity(net.WACN, net.SystemID, nac&0x0FFF)
		p2dec = trunking.P25Phase2Decode{
			Trellis:      c.p25Phase2Trellis,
			RS:           c.p25Phase2RS,
			Interleave:   c.p25Phase2Interleave,
			Scrambler:    c.p25Phase2Scrambler,
			Seed:         seed,
			SoftDecision: c.p25Phase2SoftDecision,
			Equalizer:    c.p25Phase2Equalizer,
		}
	}
	c.bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System:             c.systemName,
			Protocol:           protocol,
			GroupID:            g.groupID,
			SourceID:           g.sourceID,
			FrequencyHz:        freq,
			ChannelID:          g.channelID,
			ChannelNum:         g.channelNumber,
			RFSSID:             net.RFSS,
			SiteID:             net.Site,
			NAC:                nac,
			Encrypted:          so.Encrypted(),
			Emergency:          so.Emergency(),
			Priority:           so.Priority(),
			DataCall:           g.dataCall,
			Individual:         g.individual,
			P25Phase1DemodMode: c.p25Phase1DemodMode,
			P25Phase2Decode:    p2dec,
			At:                 c.now(),
		},
	})
	c.log.Debug("p25: grant",
		"system", c.systemName, "nac", nac, "protocol", protocol,
		"tg", g.groupID, "src", g.sourceID,
		"id", g.channelID, "num", g.channelNumber, "freq_hz", freq,
		"enc", so.Encrypted(), "emer", so.Emergency(), "data", g.dataCall)
	// ALGID/KID are unavailable at grant time on Phase 1 (the LDU2
	// Encryption Sync carries them); the engine backfills via
	// KindCallEncryption once the voice frame lands.
}

// publishSiteUpdate emits a KindSiteUpdate naming the site this control
// channel is camped on, joining the decoded RFSS/Site identity to the
// frequency the decoder is tuned to. The SiteTracker accumulates these
// into the GET /api/v1/sites table (issue #698). Called from
// dispatchTSBK after every RFSS Status Broadcast — cheap, and lets
// control_channel_hz stay accurate as the decoder hops control
// channels. Skipped until the site has actually been identified.
func (c *ControlChannel) publishSiteUpdate() {
	net := c.netModel.Snapshot()
	// Publish once any identity scalar is known. A system that never emits
	// 0x3A/0x3B can still surface its System ID (voted from adjacent-site
	// broadcasts) with RFSS/Site left zero — overlayLiveIdentity lifts
	// SystemID independently of RFSS/Site.
	if net.RFSS == 0 && net.Site == 0 && net.SystemID == 0 && net.WACN == 0 {
		return
	}
	var carrierOffsetHz int32
	if c.carrierOffsetHz != nil {
		carrierOffsetHz = int32(math.Round(c.carrierOffsetHz()))
	}
	// Snapshot the live TSBK decode-quality counters onto the update so the
	// site table can surface a per-site frame-error rate (issue #858).
	tsbkErrorRate, tsbkCount := c.Stats().TSBKErrorRate()
	c.bus.Publish(events.Event{
		Kind: events.KindSiteUpdate,
		Payload: trunking.SiteUpdate{
			System:                        c.systemName,
			RFSSID:                        net.RFSS,
			SiteID:                        net.Site,
			ControlChannelHz:              c.freqHz,
			ControlChannelCarrierOffsetHz: carrierOffsetHz,
			ControlChannelTSBKErrorRate:   tsbkErrorRate,
			ControlChannelTSBKCount:       tsbkCount,
			WACN:                          net.WACN,
			SystemID:                      net.SystemID,
			WACNHex:                       trunking.IDHex(uint64(net.WACN)),
			SystemIDHex:                   trunking.IDHex(uint64(net.SystemID)),
			RFSSIDHex:                     trunking.IDHex(uint64(net.RFSS)),
			SiteIDHex:                     trunking.IDHex(uint64(net.Site)),
			Topology:                      c.TopologySnapshot(),
			At:                            c.now(),
		},
	})
	c.logSiteIdentity(net)
}

// siteLogKey is the dedupe key for the concise site-configuration log line.
type siteLogKey struct {
	wacn  uint32
	sysid uint16
	nac   uint16
	rfss  uint8
	site  uint8
	cc    Channel
}

// logSiteIdentity emits a concise, human-readable site-configuration line the
// first time the camped site's identity is known and again whenever it
// materially changes — the live-daemon equivalent of the network-configuration
// report (cf. trunk-recorder's "Decoding System ID … WACN … NAC …" startup
// lines). It runs on the Process goroutine that owns netModel/bandPlan, so it
// reads them without a data race; the full multi-line report is rendered off
// the same TopologySnapshot by the CLI / hunt / API surfaces. Identity values
// are logged in hex to match how P25 systems are referenced in the field.
func (c *ControlChannel) logSiteIdentity(net NetworkConfig) {
	key := siteLogKey{net.WACN, net.SystemID, c.lastNAC, net.RFSS, net.Site, net.PrimaryCC}
	if key == c.lastSiteLog {
		return
	}
	c.lastSiteLog = key
	c.log.Info("p25 site configuration",
		"system", c.systemName,
		"wacn", fmt.Sprintf("%X", net.WACN),
		"sysid", fmt.Sprintf("%X", net.SystemID),
		"nac", fmt.Sprintf("%X", c.lastNAC),
		"rfss", net.RFSS,
		"site", net.Site,
		"cc_hz", c.freqHz,
	)
}

// drainPendingGrants re-publishes every voice grant that arrived for
// channelID before its IdentifierUpdate landed. Called from
// dispatchTSBK immediately after BandPlan.Apply populates the slot,
// so the second publishVoiceGrant pass resolves through the freshly
// applied band-plan entry. Entries older than pendingGrantTTL are
// dropped silently — the call they describe has already passed.
func (c *ControlChannel) drainPendingGrants(channelID uint8, nac uint16) {
	queued := c.pendingGrants.drain(channelID, c.now())
	if len(queued) == 0 {
		return
	}
	c.log.Debug("p25: draining deferred grants",
		"nac", nac, "id", channelID, "count", len(queued))
	for _, p := range queued {
		c.publishVoiceGrant(p.g, p.nac)
	}
}

// publishGroupGrant publishes a standard group voice grant (opcode
// 0x00 / 0x03) via publishVoiceGrant.
func (c *ControlChannel) publishGroupGrant(g GroupVoiceChannelGrant, nac uint16) {
	c.publishVoiceGrant(voiceGrant{
		groupID:        uint32(g.GroupAddress),
		sourceID:       g.SourceID,
		channelID:      g.ChannelID,
		channelNumber:  g.ChannelNumber,
		serviceOptions: g.ServiceOptions,
	}, nac)
}

// publishAffiliation publishes a trunking.Affiliation on the bus when
// the site issues a Group Affiliation Response (opcode 0x28). No
// band-plan resolution is needed — affiliations don't carry channel
// info.
func (c *ControlChannel) publishAffiliation(g GroupAffiliationResponse, nac uint16) {
	net := c.netModel.Snapshot()
	c.bus.Publish(events.Event{
		Kind: events.KindAffiliation,
		Payload: trunking.Affiliation{
			System:            c.systemName,
			Protocol:          "p25",
			SourceID:          g.TargetID,
			GroupID:           uint32(g.GroupAddress),
			AnnouncementGroup: uint32(g.AnnouncementGroup),
			Response:          trunking.AffiliationResponse(g.Response),
			RFSSID:            net.RFSS,
			SiteID:            net.Site,
			NAC:               nac,
			At:                c.now(),
		},
	})
	c.log.Debug("p25: affiliation",
		"system", c.systemName, "nac", nac,
		"src", g.TargetID, "tg", g.GroupAddress,
		"ann", g.AnnouncementGroup, "rsp", g.Response)
}

// publishUnitRegistration publishes a trunking.UnitRegistration on the
// bus when the site issues a Unit Registration Response (opcode 0x2C).
func (c *ControlChannel) publishUnitRegistration(u UnitRegistrationResponse, nac uint16) {
	net := c.netModel.Snapshot()
	c.bus.Publish(events.Event{
		Kind: events.KindUnitRegistration,
		Payload: trunking.UnitRegistration{
			System:   c.systemName,
			Protocol: "p25",
			SourceID: u.SourceAddress, // the registering radio's unit ID
			// U_REG_RSP carries no WACN, and its System ID has proven
			// unreliable to recover per-message; take both from the
			// NSB-corroborated network model so they match the System
			// Identity panel instead of floating per registration.
			WACN:     net.WACN,
			SystemID: net.SystemID,
			Response: trunking.RegistrationResponse(u.Response),
			RFSSID:   net.RFSS,
			SiteID:   net.Site,
			NAC:      nac,
			At:       c.now(),
		},
	})
	c.log.Debug("p25: registration",
		"system", c.systemName, "nac", nac,
		"src", u.SourceAddress, "wuid", u.SourceID,
		"wacn", net.WACN, "sysid", net.SystemID,
		"rsp", u.Response)
}

// publishUnitToUnitRequest publishes a trunking.UnitToUnitRequest on the bus
// when the site issues a Unit-to-Unit Answer Request (opcode 0x05). No
// band-plan resolution is needed — the answer request carries no channel, only
// the calling/called radio IDs.
func (c *ControlChannel) publishUnitToUnitRequest(u UnitToUnitAnswerRequest, nac uint16) {
	c.bus.Publish(events.Event{
		Kind: events.KindUnitToUnitRequest,
		Payload: trunking.UnitToUnitRequest{
			System:         c.systemName,
			Protocol:       "p25",
			SourceID:       u.SourceID,
			TargetID:       u.TargetID,
			ServiceOptions: uint8(u.ServiceOptions),
			At:             c.now(),
		},
	})
	c.log.Debug("p25: unit-to-unit answer request",
		"system", c.systemName, "nac", nac,
		"src", u.SourceID, "target", u.TargetID)
}

// MarkLost publishes a CCLost event for the current frequency and resets
// the locked flag. Loss tracking is intentionally simple — the engine /
// hunter pair owns the watchdog logic and calls this when the control
// channel goes silent.
func (c *ControlChannel) MarkLost() {
	if !c.locked {
		return
	}
	c.locked = false
	// Drop accumulated dibits + pending hits so a re-acquisition
	// starts from a clean slate.
	c.buf = c.buf[:0]
	c.pending = c.pending[:0]
	c.bus.Publish(events.Event{
		Kind:    events.KindCCLost,
		Payload: LockState{FrequencyHz: c.freqHz, NAC: c.lastNAC},
	})
}
