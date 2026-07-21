package tetra

import "github.com/MattCheramie/GopherTrunk/internal/radio/framing"

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
// on the carrier — all four TDMA timeslots, not just the granted one —
// because absolute slot alignment needs frame-number tracking the tap does
// not yet have. That is an honest first-pass limitation; the granted
// Timeslot is carried on the grant for a future per-slot filter.
const (
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
	dets    []*SyncDetector // NTS1 + NTS2, each under all four constellation rotations
	scratch []int
	buf     []uint8
	bufBase int
	pending []int // training-sequence leading indices awaiting look-ahead
	onBurst func(frame []byte)
}

// NewTrafficExtractor returns an extractor that calls onBurst with each
// recovered 54-byte traffic frame. onBurst must not retain the slice.
func NewTrafficExtractor(onBurst func(frame []byte)) *TrafficExtractor {
	te := &TrafficExtractor{onBurst: onBurst}
	// π/4-DQPSK leaves a constant 0..3 dibit rotation (residual CFO), so
	// correlate the training sequence under all four rotations of the
	// pattern — the same trick processSB uses for the sync burst.
	for _, base := range [][]uint8{NormalSyncDibits(), NormalSyncDibits2()} {
		for r := uint8(0); r < 4; r++ {
			te.dets = append(te.dets, NewSyncDetector(rotateDibits(base, r), 2))
		}
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
	te.onBurst(framing.PackBitsMSB(TetraDibitsToBits(d)))
}
