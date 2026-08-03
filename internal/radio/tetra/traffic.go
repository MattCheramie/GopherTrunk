package tetra

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// Traffic-channel burst extraction for the voice-follow path.
//
// Once the control channel grants a call onto a traffic carrier, a voice
// tap retunes to that carrier and demodulates it to the same π/4-DQPSK
// dibit stream the control channel uses. Each downlink timeslot on the
// carrier is a Normal Continuous Downlink Burst (NCDB), whose layout is
// (ETSI EN 300 392-2 §9.4.4.3.2, matching osmo-tetra), in dibits relative
// to the normal training sequence leading dibit L:
//
//	BKN1 (block 1) : [L-115, L-7)   108 dibits (216 type-5 bits)
//	AACH half 1    : [L-7,   L)       7 dibits ( 14 bits)
//	training seq   : [L,     L+11)   11 dibits ( 22 bits)
//	AACH half 2    : [L+11,  L+19)    8 dibits ( 16 bits)
//	BKN2 (block 2) : [L+19,  L+127)  108 dibits (216 type-5 bits)
//
// TrafficExtractor recovers BKN1 and BKN2 around each detected training
// sequence and emits them concatenated as one 432-bit (54-byte) full-slot
// traffic frame.
//
// The emitted frame is the raw, still-scrambled type-5 payload; TCH/S channel
// decoding (§5.5) and the ACELP vocoder run downstream in the composer's TETRA
// chain (tch.go + internal/voice/acelp), which descrambles with the cell's
// extended colour code (learned from the control channel's BSCH), TCH/S-decodes
// each burst into 137-bit speech frames, and renders them to PCM. The raw frames
// are also written to the recorder's `.raw` sidecar, exactly as the DMR / P25 /
// EDACS-ProVoice voice paths write their post-FEC or raw frames.
//
// Slot demultiplexing: the extractor emits a frame for every NCDB it sees
// on the carrier — all four TDMA timeslots — tagged with both its AACH downlink
// usage marker (the reliable per-slot call identifier, ETSI §21.4.7) and its TDMA
// timeslot number, so concurrent calls on one carrier decode independently.
//
// The usage marker is the demux key the voice chain routes by: the AACH is
// present and decodes in every downlink slot, and a call's marker matches the
// usage marker carried in its grant. The TDMA timeslot number is retained for
// telemetry, but on real air it is NOT a reliable demux key — the SB anchor's
// intra-slot rounding jitters a call's bursts across adjacent slot numbers, and
// the channel-allocation grant timeslot field does not map to the physical slot.
//
// The slot number is derived from the synchronisation burst (SB): the SB's
// synchronisation training sequence is transmitted in slot 1 (TN1) of frame 18,
// so it anchors the 255-dibit slot grid. A burst leading at dibit L is then in
// slot ((round((L - sbAnchor)/255)) mod 4) + 1. Until an SB is seen the slot is
// reported as 0 (unknown). On a traffic-only carrier with no SB the slot stays 0.
const (
	// ndbSlotDibits is the dibit span of one TDMA timeslot (510 bits): four
	// slots make a 1020-dibit TDMA frame. The slot grid the SB anchor pins.
	ndbSlotDibits = 255
	// ndbBKN1Start / ndbBKN2Start are the leading dibit of each data block
	// relative to the training-sequence leading dibit L.
	ndbBKN1Start = -115
	ndbBKN2Start = 19
	// ndbBlockDibits is the dibit count of one NCDB data block (216 type-5
	// bits) — the SCH/HD / traffic half-slot size.
	ndbBlockDibits = 108
	// ndbTrimMargin is how many trailing dibits the rolling buffer keeps so
	// a training-sequence hit near the end still has its BKN1 look-back.
	ndbTrimMargin = 256
)

// TrafficFrameDibits is the number of dibits in one emitted full-slot
// traffic frame (BKN1 + BKN2).
const TrafficFrameDibits = 2 * ndbBlockDibits // 216

// TrafficFrameBytes is the packed size of one emitted traffic frame
// (432 type-5 bits, MSB-first).
const TrafficFrameBytes = (2 * ndbBlockDibits * 2) / 8 // 54

// TrafficExtractor scans a π/4-DQPSK dibit stream for Normal Continuous
// Downlink Bursts and emits each burst's BKN1+BKN2 as a raw 54-byte
// full-slot traffic frame. It is the traffic-channel analog of the control
// channel's processSB: a rolling buffer with look-back to BKN1 and
// look-ahead to BKN2 around each detected normal training sequence.
//
// Not safe for concurrent use; construct one per followed call.
type TrafficExtractor struct {
	dets       []*SyncDetector // NTS1 + NTS2, each under all four constellation rotations
	stsDets    []*SyncDetector // synchronisation training sequence, all four rotations (slot-1 anchor)
	scratch    []int
	stsScratch []int
	buf        []uint8
	bufBase    int
	pending    []int // training-sequence leading indices awaiting look-ahead
	colourCode uint32
	onBurst    func(frame []byte, softType5 []float32, slot, usage uint8)

	// softBuf holds the per-dibit complex differentials (the receiver's soft
	// info) strictly parallel to buf, so emit can build a soft-decision type-5
	// LLR stream for the burst. It is either empty (no soft data supplied — the
	// hard-only path used by tests and any caller that never StashSofts) or
	// exactly len(buf). pendingSoft carries the differentials stashed for the
	// NEXT Process call (StashSoft), keyed by pendingSoftBase and consumed once.
	softBuf         []complex64
	pendingSoft     []complex64
	pendingSoftBase int

	// sbAnchor is the absolute dibit index of the most recent SB
	// synchronisation-training-sequence leading dibit (TN1), pinning the
	// 255-dibit slot grid; haveAnchor is false until the first SB is seen.
	sbAnchor   int
	haveAnchor bool
}

// NewTrafficExtractor returns an extractor that calls onBurst with each
// recovered 54-byte traffic frame, the burst's soft-decision type-5 stream
// (432 descrambled LLRs, or nil when no soft info was stashed — see StashSoft),
// its TDMA timeslot (1..4, or 0 when the slot grid is not yet anchored to a
// synchronisation burst), and the burst's AACH downlink usage marker (§21.4.7;
// the per-slot call identifier, >= DLUsageTraffic for a traffic slot, 0 when the
// AACH did not decode or the slot is not traffic).
// When colourCode is non-zero the frame is descrambled with the cell's extended
// colour code (learned from the control channel's BSCH) before onBurst, so the
// sidecar holds descrambled type-5 — the input the TCH/S channel decoder expects.
// The usage marker is what lets a per-call voice chain demultiplex concurrent
// same-carrier calls reliably (the channel-allocation timeslot field does not map
// to the physical slot on real air). onBurst must not retain the slice.
func NewTrafficExtractor(colourCode uint32, onBurst func(frame []byte, softType5 []float32, slot, usage uint8)) *TrafficExtractor {
	te := &TrafficExtractor{colourCode: colourCode, onBurst: onBurst}
	// π/4-DQPSK leaves a constant 0..3 dibit rotation (residual CFO), so
	// correlate the training sequence under all four rotations of the
	// pattern — the same trick processSB uses for the sync burst.
	for _, base := range [][]uint8{NormalSyncDibits(), NormalSyncDibits2()} {
		for r := uint8(0); r < 4; r++ {
			te.dets = append(te.dets, NewSyncDetector(rotateDibits(base, r), 2))
		}
	}
	// Synchronisation-training-sequence detectors (all four rotations) pin
	// the slot grid: the SB is transmitted in TN1, so its STS leading dibit
	// anchors slot 1. Threshold 3 matches the control channel's processSB.
	sts := SyncTrainingDibits()
	for r := uint8(0); r < 4; r++ {
		te.stsDets = append(te.stsDets, NewSyncDetector(rotateDibits(sts, r), 3))
	}
	return te
}

// Process consumes a window of dibits from the traffic-channel receiver.
// baseIdx is the absolute dibit index of dibits[0]; it must be
// monotonically non-decreasing across calls (Receiver.Reset restarts it).
func (te *TrafficExtractor) Process(dibits []uint8, baseIdx int) {
	if len(te.buf) == 0 {
		te.bufBase = baseIdx
	}
	// Keep softBuf strictly parallel to buf. Append the stashed differentials
	// only when they match this dibit block (same base + length) AND softBuf is
	// already in lockstep with buf; otherwise drop the soft path (reset to empty)
	// rather than risk a misalignment — the burst then decodes hard-only. In the
	// live path SoftSink stashes every call, so softBuf tracks buf from the first
	// call and stays parallel; tests never stash, so it stays empty (hard-only).
	if te.pendingSoft != nil && te.pendingSoftBase == baseIdx &&
		len(te.pendingSoft) == len(dibits) && len(te.softBuf) == len(te.buf) {
		te.softBuf = append(te.softBuf, te.pendingSoft...)
	} else {
		te.softBuf = te.softBuf[:0]
	}
	te.pendingSoft = nil
	te.buf = append(te.buf, dibits...)

	// Anchor the slot grid on the synchronisation burst. The SB's STS is
	// transmitted in TN1, so its leading dibit pins slot 1; refresh on every
	// SB (once per multiframe) so the anchor tracks any slow clock drift.
	for _, det := range te.stsDets {
		hits, _ := det.Process(te.stsScratch[:0], dibits, baseIdx)
		for _, trailing := range hits {
			te.sbAnchor = trailing - (stsDibits - 1)
			te.haveAnchor = true
		}
	}

	ntsLen := len(NormalSyncDibits()) // 11
	for _, det := range te.dets {
		var hits []int
		hits, _ = det.Process(te.scratch[:0], dibits, baseIdx)
		for _, trailing := range hits {
			L := trailing - (ntsLen - 1)
			dup := false
			for _, q := range te.pending {
				if q == L {
					dup = true
					break
				}
			}
			if !dup {
				te.pending = append(te.pending, L)
			}
		}
	}

	bufEnd := te.bufBase + len(te.buf)
	kept := make([]int, 0, len(te.pending))
	for _, L := range te.pending {
		needStart := L + ndbBKN1Start
		needEnd := L + ndbBKN2Start + ndbBlockDibits
		if needStart < te.bufBase {
			continue // look-back already trimmed; give up on this hit
		}
		if needEnd > bufEnd {
			kept = append(kept, L) // not enough look-ahead yet
			continue
		}
		te.emit(L)
	}
	te.pending = kept

	// Trim, keeping the trailing margin plus any unresolved hit's look-back.
	keepFrom := bufEnd - ndbTrimMargin
	for _, L := range te.pending {
		if ns := L + ndbBKN1Start; ns < keepFrom {
			keepFrom = ns
		}
	}
	if keepFrom > te.bufBase {
		drop := keepFrom - te.bufBase
		if drop > len(te.buf) {
			drop = len(te.buf)
		}
		te.buf = append(te.buf[:0], te.buf[drop:]...)
		// Trim softBuf by the same amount to keep it parallel; if it is not in
		// lockstep (empty / short), drop the soft path rather than misalign.
		if len(te.softBuf) >= drop {
			te.softBuf = append(te.softBuf[:0], te.softBuf[drop:]...)
		} else {
			te.softBuf = te.softBuf[:0]
		}
		te.bufBase += drop
	}
}

// StashSoft hands the extractor the per-dibit complex differentials (the
// receiver's SoftSink output) for the dibits the NEXT Process call will
// deliver, keyed by the same baseIdx. It lets the traffic path carry soft
// (LLR) information for soft-decision TCH/S decoding without changing the hard
// DibitSink contract — the same stash bridge the control channel uses. A caller
// that never StashSofts leaves the extractor hard-only.
func (te *TrafficExtractor) StashSoft(diffs []complex64, baseIdx int) {
	te.pendingSoft = diffs
	te.pendingSoftBase = baseIdx
}

// emit slices BKN1 and BKN2 around the training sequence leading at L and
// forwards the concatenated raw type-5 frame, tagged with the burst's TDMA slot
// and its AACH downlink usage marker.
func (te *TrafficExtractor) emit(L int) {
	block := func(off int) []uint8 {
		s := L + off - te.bufBase
		return te.buf[s : s+ndbBlockDibits]
	}
	d := make([]uint8, 0, TrafficFrameDibits)
	d = append(d, block(ndbBKN1Start)...)
	d = append(d, block(ndbBKN2Start)...)
	bits := TetraDibitsToBits(d)
	if te.colourCode != 0 {
		bits = framing.DescrambleTetra(bits, te.colourCode)
	}
	te.onBurst(framing.PackBitsMSB(bits), te.softFrame(L), te.slotOf(L), te.usageOf(L))
}

// softFrame builds the descrambled 432-LLR type-5 stream for the burst leading
// at L from the parallel soft buffer, mirroring emit's hard BKN1+BKN2 slicing,
// or nil when soft info is unavailable for this burst (hard-only fallback). The
// receiver's AFC locks the constellation to rotation 0 (the same assumption the
// hard BKN descramble relies on), so softType5FromDiffs is called with rot 0;
// its hard-slice equals the emitted hard bits, and DescrambleTetraSoft applies
// the same colour-code sign flips the hard DescrambleTetra does.
func (te *TrafficExtractor) softFrame(L int) []float32 {
	if len(te.softBuf) != len(te.buf) {
		return nil
	}
	block := func(off int) []complex64 {
		s := L + off - te.bufBase
		if s < 0 || s+ndbBlockDibits > len(te.softBuf) {
			return nil
		}
		return te.softBuf[s : s+ndbBlockDibits]
	}
	b1 := block(ndbBKN1Start)
	b2 := block(ndbBKN2Start)
	if b1 == nil || b2 == nil {
		return nil
	}
	diffs := make([]complex64, 0, TrafficFrameDibits)
	diffs = append(diffs, b1...)
	diffs = append(diffs, b2...)
	llr := softType5FromDiffs(diffs, 0)
	if te.colourCode != 0 {
		llr = framing.DescrambleTetraSoft(llr, te.colourCode)
	}
	return llr
}

// usageOf recovers the AACH downlink usage marker of the burst leading at L. The
// 30-bit access-assignment sits in two halves either side of the normal training
// sequence (same geometry as the control-channel downlinkNCDB); it is scrambled
// with the cell colour code and RM(30,14)-coded. Returns 0 when the AACH is not
// buffered, does not decode, or the slot is not carrying traffic — callers treat
// 0 as "unknown" and fall back to CRC-gated isolation. The receiver's AFC locks
// the constellation to rotation 0 (the same assumption the BKN descramble above
// relies on), so no rotation search is needed here.
func (te *TrafficExtractor) usageOf(L int) uint8 {
	half := func(off, n int) []uint8 {
		s := L + off - te.bufBase
		if s < 0 || s+n > len(te.buf) {
			return nil
		}
		return te.buf[s : s+n]
	}
	a1 := half(ndbAACH1Start, ndbAACH1Len)
	a2 := half(ndbAACH2Start, ndbAACH2Len)
	if a1 == nil || a2 == nil {
		return 0
	}
	di := make([]uint8, 0, ndbAACH1Len+ndbAACH2Len)
	di = append(di, a1...)
	di = append(di, a2...)
	if rec, errs := DecodeAACH(TetraDibitsToBits(di), te.colourCode); errs >= 0 {
		if aa, ok := ParseAccessAssign(rec); ok {
			if u := aa.DownlinkUsage(); u >= DLUsageTraffic {
				return u
			}
		}
	}
	// Hard decode found no routable traffic marker. On a marginal AACH (a common
	// RM(30,14) miss under concurrent-call load) the soft-decision decode recovers
	// it from the per-symbol confidences, so the burst routes by marker instead of
	// being dropped and garbling that slot's recording. Only used as a fallback and
	// gated on a confident, valid decode so it never invents a marker that would
	// misroute another call's speech (a cross-slot leak).
	if u, ok := te.usageOfSoft(L); ok {
		return u
	}
	return 0
}

// aachSoftMaxDist gates the soft AACH fallback: the soft maximum-likelihood
// codeword must sit within this Hamming distance of the hard-sliced received bits
// to be trusted. A genuine marginal burst the soft decoder rescues differs from
// the hard bits by only the few symbols the LLRs re-decided; a decode this far from
// the received word is a low-confidence guess that could misroute, so it is
// dropped (as it is today). Validated against the reporter's concurrent-call
// captures: soft-recovered markers keep mapping cleanly to a single physical slot.
const aachSoftMaxDist = 6

// usageOfSoft is the soft-decision fallback for usageOf: it pulls the per-symbol
// differentials for the two AACH halves from the parallel soft buffer, builds the
// type-5 LLR stream (rotation 0, the AFC-locked assumption usageOf relies on) and
// runs the soft RM(30,14) decode. Returns the traffic usage marker and true only
// on a confident, valid traffic decode. A no-op (false) when no soft info is
// buffered, so the hard-only path is unchanged.
func (te *TrafficExtractor) usageOfSoft(L int) (uint8, bool) {
	if len(te.softBuf) != len(te.buf) {
		return 0, false
	}
	half := func(off, n int) []complex64 {
		s := L + off - te.bufBase
		if s < 0 || s+n > len(te.softBuf) {
			return nil
		}
		return te.softBuf[s : s+n]
	}
	a1 := half(ndbAACH1Start, ndbAACH1Len)
	a2 := half(ndbAACH2Start, ndbAACH2Len)
	if a1 == nil || a2 == nil {
		return 0, false
	}
	diffs := make([]complex64, 0, ndbAACH1Len+ndbAACH2Len)
	diffs = append(diffs, a1...)
	diffs = append(diffs, a2...)
	rec, dist, _ := DecodeAACHSoft(softType5FromDiffs(diffs, 0), te.colourCode)
	if dist < 0 || dist > aachSoftMaxDist {
		return 0, false
	}
	aa, ok := ParseAccessAssign(rec)
	if !ok {
		return 0, false
	}
	if u := aa.DownlinkUsage(); u >= DLUsageTraffic {
		return u, true
	}
	return 0, false
}

// ndbSBSlotShift aligns the synchronisation-burst anchor to the NDB slot grid.
// The SB is transmitted in TN1, but its synchronisation training sequence sits
// late in the SB burst (after the frequency-correction + BSCH preamble), so the
// detected STS leading dibit lands one NDB slot before the TN1 traffic burst's
// normal-training-sequence position. Adding 3 (= −1 mod 4) makes a burst one
// slot after the anchor read as TN1, matching the control channel's granted
// timeslots — verified against the reporter's real same-carrier capture
// (grant ts1↔decoded slot, ts2↔decoded slot line up once shifted).
const ndbSBSlotShift = 3

// slotOf returns the TDMA timeslot (1..4) of a burst whose training sequence
// leads at absolute dibit L, or 0 when no synchronisation burst has anchored the
// slot grid yet. The SB anchors the grid (see ndbSBSlotShift); each further slot
// is 255 dibits on, and rounding to the nearest slot absorbs the small intra-slot
// offset between the normal and synchronisation training sequences.
func (te *TrafficExtractor) slotOf(L int) uint8 {
	if !te.haveAnchor {
		return 0
	}
	si := (int(math.Round(float64(L-te.sbAnchor)/float64(ndbSlotDibits))) + ndbSBSlotShift) % 4
	if si < 0 {
		si += 4
	}
	return uint8(si) + 1
}
