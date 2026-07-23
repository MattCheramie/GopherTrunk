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
// The emitted frame is the raw, still-scrambled type-5 payload. TCH/S
// channel decoding (the unequal-error-protection scheme of §8.4) and the
// TETRA ACELP vocoder are deliberately NOT applied here — they are the
// labelled follow-ups. The raw frames go to the recorder's `.raw` sidecar
// for out-of-band decode, exactly as the DMR / P25 / EDACS-ProVoice voice
// paths write their post-FEC or raw frames. Being scrambled, a consumer
// descrambles with the cell's extended colour code (learned from the
// control channel's BSCH) before TCH/S FEC.
//
// Slot demultiplexing: the extractor emits a frame for every NCDB it sees
// on the carrier — all four TDMA timeslots — and tags each with its TDMA
// timeslot (TN, 1..4) so a per-slot voice chain can keep only its granted
// slot and up to four calls on one carrier decode independently. The slot is
// numbered from the synchronisation burst (SB), which the same detector
// correlates: the SB's synchronisation training sequence is transmitted in
// slot 1 (TN1) of frame 18, so it anchors the 255-dibit slot grid. A burst
// leading at dibit L is then in slot ((round((L - sbAnchor)/255)) mod 4) + 1;
// the ~15-dibit intra-slot offset between the normal and synchronisation
// training sequences is far below the half-slot (127-dibit) rounding margin,
// so it need not be modelled exactly. Until an SB is seen the slot is reported
// as 0 (unknown) and the consumer falls back to CRC-gated single-call
// behaviour. On a traffic-only carrier with no SB the slot stays 0.
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
	onBurst    func(frame []byte, slot uint8)

	// sbAnchor is the absolute dibit index of the most recent SB
	// synchronisation-training-sequence leading dibit (TN1), pinning the
	// 255-dibit slot grid; haveAnchor is false until the first SB is seen.
	sbAnchor   int
	haveAnchor bool
}

// NewTrafficExtractor returns an extractor that calls onBurst with each
// recovered 54-byte traffic frame and its TDMA timeslot (1..4, or 0 when the
// slot grid is not yet anchored to a synchronisation burst). When colourCode is
// non-zero the frame is descrambled with the cell's extended colour code
// (learned from the control channel's BSCH) before onBurst, so the sidecar
// holds descrambled type-5 — the input the TCH/S channel decoder expects.
// onBurst must not retain the slice.
func NewTrafficExtractor(colourCode uint32, onBurst func(frame []byte, slot uint8)) *TrafficExtractor {
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
		te.bufBase += drop
	}
}

// emit slices BKN1 and BKN2 around the training sequence leading at L and
// forwards the concatenated raw type-5 frame.
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
	te.onBurst(framing.PackBitsMSB(bits), te.slotOf(L))
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
