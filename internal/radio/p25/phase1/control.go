package phase1

import (
	"errors"
	"fmt"
	"log/slog"
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
	systemName string
	freqHz     uint32
	bandPlan   *BandPlan
	now        func() time.Time
	locked     bool
	lastNAC    uint16
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
	// through the matching symbol-recovery path; without this an LSM
	// simulcast site's voice grants would land in a hardcoded C4FM
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

	// stats is the per-frame outcome counter exposed via Stats().
	// Atomic so the daemon's API / diagnostic goroutines can read it
	// concurrently with Process. Issue #402: gives replay and any
	// future operator dashboard a per-run "of the frames the FSW
	// correlator delivered, how many cleared each acceptance gate"
	// breakdown — the steady-state shape that distinguishes "demod
	// is broken" from "demod is fine but one frame in N is corrupt".
	stats CCStats
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
type CCStats struct {
	NIDTrusted        int64
	NIDMarginal       int64
	NIDFailed         int64
	TSBKDecoded       int64
	TSBKTrellisFailed int64
	TSBKCRCFailed     int64
}

// NetworkSnapshot returns the system topology accumulated from the
// site's Network / RFSS / Secondary-CC / Adjacent-Site status TSBKs.
func (c *ControlChannel) NetworkSnapshot() NetworkConfig {
	return c.netModel.Snapshot()
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

	cont       bool   // true ⇒ resume trailing TSBK blocks (NID already decoded)
	blocksDone int    // TSBK blocks already dispatched for this data unit
	nextStart  int    // absolute index of the next TSBK block to decode
	fswStart   int    // absolute index of the frame's first FSW dibit
	strip      bool   // status-symbol stripping the alignment search chose
	nac        uint16 // NAC the NID decoded to (for dispatch)
}

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
	det := NewSyncDetector(4)
	if len(opts.Rotations) > 0 {
		det.SetRotations(opts.Rotations)
	}
	span := opts.NIDSearchSpan
	if span <= 0 {
		span = NIDSearchSpan
	}
	return &ControlChannel{
		bus:                 opts.Bus,
		log:                 log,
		det:                 det,
		systemName:          opts.SystemName,
		freqHz:              opts.FrequencyHz,
		bandPlan:            bp,
		now:                 now,
		aliasAsm:            NewTalkerAliasAssembler(now),
		rotations:           resolveRotations(opts.Rotations),
		nidSearchSpan:       span,
		p25Phase1DemodMode:  opts.P25Phase1DemodMode,
		p25Phase2Trellis:    opts.P25Phase2Trellis,
		p25Phase2RS:         opts.P25Phase2RS,
		p25Phase2Interleave: opts.P25Phase2Interleave,
		p25Phase2Scrambler:  opts.P25Phase2Scrambler,
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
	hits, rots, next := c.det.ProcessWithRotation(nil, nil, dibits, baseIdx)
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
		c.pending = append(c.pending, pendingHit{end: h, rot: rots[i]})
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
		if cont, more := c.parseFrame(c.buf, nidStart, ph.rot); more {
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
func (c *ControlChannel) parseFrame(buf []uint8, nidStart int, fswRot uint8) (cont pendingHit, more bool) {
	best, found, diag := c.searchNID(buf, nidStart, fswRot)
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
func (c *ControlChannel) resumeTSBKBlocks(ph *pendingHit) bool {
	for ph.blocksDone < maxTSBKBlocks {
		start := ph.nextStart - c.bufBase
		if start < 0 {
			return false // buffer trimmed past it (shouldn't happen) — drop
		}
		if start+tsbkBlockSpan > len(c.buf) {
			return true // next block not buffered yet — resume later
		}
		fswStart := ph.fswStart - c.bufBase
		tsbkChannel, next := gatherFrameDibits(c.buf, start, 98, fswStart, ph.strip)
		tsbk, metric, err := DecodeTSBKChannel(rotateDibits(tsbkChannel, ph.rot))
		if err != nil {
			if errors.Is(err, CRCError) {
				atomic.AddInt64(&c.stats.TSBKCRCFailed, 1)
			} else {
				atomic.AddInt64(&c.stats.TSBKTrellisFailed, 1)
			}
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
func (c *ControlChannel) searchNID(buf []uint8, nidStart int, fswRot uint8) (nidGuess, bool, string) {
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
				if errs > NIDAcceptErrs {
					// Parity-valid but too far from the received word for
					// BCH+parity to tell a noisy real NID from a bad
					// alignment's miscorrection. Defer to the marginal
					// tier, where the TSBK CRC decides.
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
	// — see tsbk_vendor.go.
	if t.IsVendorMFID() {
		c.dispatchVendorTSBK(t, nac)
		return
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
	case OpIdentifierUpdateVUHF:
		u := ParseIdentifierUpdateVUHF(t.Payload)
		c.bandPlan.Apply(u)
		c.log.Debug("p25: identifier update (VUHF)",
			"nac", nac, "id", u.ChannelID,
			"base_hz", u.BaseHz, "spacing_hz", u.SpacingHz,
			"tx_offset_hz", u.TxOffsetHz,
			"bandwidth_hz", u.BandwidthHz)
		c.drainPendingGrants(u.ChannelID, nac)
	case OpIdentifierUpdateTDMA:
		u := ParseIdentifierUpdateTDMA(t.Payload)
		c.bandPlan.Apply(u)
		c.log.Debug("p25: identifier update (TDMA)",
			"nac", nac, "id", u.ChannelID,
			"base_hz", u.BaseHz, "spacing_hz", u.SpacingHz,
			"tx_offset_hz", u.TxOffsetHz,
			"bandwidth_hz", u.BandwidthHz)
		c.drainPendingGrants(u.ChannelID, nac)
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
		}, nac)
	case OpTelephoneInterconnectGrant:
		g := ParseTelephoneInterconnectGrant(t.Payload)
		c.publishVoiceGrant(voiceGrant{
			groupID: g.TargetID, channelID: g.ChannelID,
			channelNumber: g.ChannelNumber, serviceOptions: g.ServiceOptions,
		}, nac)
	case OpSNDCPDataChannelGrant:
		g := ParseSNDCPDataChannelGrant(t.Payload)
		c.publishVoiceGrant(voiceGrant{
			groupID: g.TargetID, channelID: g.ChannelID,
			channelNumber: g.ChannelNumber, serviceOptions: g.ServiceOptions,
			dataCall: true,
		}, nac)
	case OpNetworkStatusBroadcast:
		c.netModel.ApplyNetworkStatus(ParseNetworkStatusBroadcast(t.Payload))
	case OpRFSSStatusBroadcast:
		c.netModel.ApplyRFSSStatus(ParseRFSSStatusBroadcast(t.Payload))
	case OpSecondaryControlChannel:
		c.netModel.ApplySecondaryControlChannel(ParseSecondaryControlChannelBroadcast(t.Payload))
	case OpAdjacentSiteStatusBroadcast:
		c.netModel.ApplyAdjacentSite(ParseAdjacentSiteStatusBroadcast(t.Payload))
	case OpGroupAffiliationResponse:
		c.publishAffiliation(ParseGroupAffiliationResponse(t.Payload), nac)
	case OpUnitRegistrationResponse:
		c.publishUnitRegistration(ParseUnitRegistrationResponse(t.Payload), nac)
	default:
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
	if f, ok := t.AsTalkerAliasFragment(); ok {
		if alias, src, done := c.aliasAsm.Add(f); done {
			c.publishTalkerAlias(src, alias)
		}
		return
	}
	c.log.Debug("p25: vendor tsbk", "mfid", t.MFID, "opcode", t.Opcode, "nac", nac)
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
	c.log.Debug("p25: talker alias",
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
	var p2dec trunking.P25Phase2Decode
	if c.bandPlan.IsTDMA(g.channelID) {
		protocol = "p25-phase2"
		net := c.netModel.Snapshot()
		seed := framing.PN44SeedFromIdentity(net.WACN, net.SystemID, nac&0x0FFF)
		p2dec = trunking.P25Phase2Decode{
			Trellis:    c.p25Phase2Trellis,
			RS:         c.p25Phase2RS,
			Interleave: c.p25Phase2Interleave,
			Scrambler:  c.p25Phase2Scrambler,
			Seed:       seed,
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
			Encrypted:          so.Encrypted(),
			Emergency:          so.Emergency(),
			DataCall:           g.dataCall,
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
	c.bus.Publish(events.Event{
		Kind: events.KindAffiliation,
		Payload: trunking.Affiliation{
			System:            c.systemName,
			Protocol:          "p25",
			SourceID:          g.TargetID,
			GroupID:           uint32(g.GroupAddress),
			AnnouncementGroup: uint32(g.AnnouncementGroup),
			Response:          trunking.AffiliationResponse(g.Response),
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
	c.bus.Publish(events.Event{
		Kind: events.KindUnitRegistration,
		Payload: trunking.UnitRegistration{
			System:   c.systemName,
			Protocol: "p25",
			SourceID: u.SourceID,
			WACN:     u.WACN,
			SystemID: u.SystemID,
			Response: trunking.RegistrationResponse(u.Response),
			At:       c.now(),
		},
	})
	c.log.Debug("p25: registration",
		"system", c.systemName, "nac", nac,
		"src", u.SourceID, "wacn", u.WACN, "sysid", u.SystemID,
		"rsp", u.Response)
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
