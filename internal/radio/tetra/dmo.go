package tetra

// DMO (Direct Mode Operation) radio-layer framing — ETSI EN 300 396-2 (part 2:
// radio aspects). DMO is TETRA's infrastructure-less peer-to-peer mode: a
// transmitting MS sends a Direct Mode Synchronisation Burst (DSB) to let the
// other radios acquire, then Direct Mode Normal Bursts (DNB) carrying the call.
// There is no control channel, so the trunked-mode ingestion path (hunt a CC →
// grants) does not apply; a DMO receiver instead camps a DM channel, detects the
// DSB, and follows the burst train.
//
// The DMO air interface REUSES the trunked-mode (TMO) physical layer this package
// already implements: identical π/4-DQPSK at 18 ksym/s, 25 kHz channels, 255-symbol
// (14.167 ms) timeslots, the 4-slot frame / 18-frame multiframe (frame 18 is the
// DM control frame), and — verified below in dmo_test.go against EN 300 396-2
// §9.4.3.3 equations 63/64/66 — the SAME normal and synchronisation training
// sequences (NormalTrainingSeq1/2, SyncTrainingSeq) and the SAME 32-tap scrambler
// polynomial (colour code 0 for the SCH/S and SCH/H of a DSB, exactly like TMO's
// BSCH). So the receiver, sync-word correlation and channel-coding machinery are
// shared; only the burst FIELD LAYOUT differs, which is what this file adds.
//
// What differs from TMO and is defined here (EN 300 396-2 §9.4.3.2, Tables 15/16):
// the DSB carries an 80-bit frequency-correction field and a 120-bit SCH/S block
// (BKN1) ahead of the sync training sequence, with a 216-bit block (BKN2) after;
// the DNB carries two 216-bit blocks (BKN1/BKN2) around the normal training
// sequence. The block boundaries relative to the training-sequence lead dibit are
// NOT the same as TMO's NDB (traffic.go), so DMO needs its own slicer.
//
// Scope of this file (deliberate, honest): burst DETECTION + BLOCK SLICING only —
// it turns a demodulated DMO dibit stream into per-burst BKN1/BKN2 type-5 (still
// scrambled) blocks, the same shape TrafficExtractor produces for TMO. Descrambling
// then channel-decoding those blocks (SCH/S, SCH/F, TCH/S) reuses the existing
// framing / tch code, and the DM call-control PROTOCOL that rides in SCH/S/SCH/F —
// source/destination SSI, group, call type — is EN 300 396-3 (the MS-MS AI
// protocol), a separate specification. Neither the channel decode nor the protocol
// is wired here, and nothing in this file has been validated against a real DMO
// capture yet — both are prerequisites before DMO voice can be attributed and
// recorded, and both should be validated against captured air (the #764/#771
// lesson: synthetic round-trips can pass while on-air fails).

// DMBurstKind identifies which DMO burst a detection came from.
type DMBurstKind uint8

const (
	// DMBurstSync is a Direct Mode Synchronisation Burst (DSB): frequency
	// correction + SCH/S (BKN1) + sync training sequence + BKN2 (§9.4.3.2.3).
	DMBurstSync DMBurstKind = iota
	// DMBurstNormal is a Direct Mode Normal Burst (DNB): BKN1 + normal training
	// sequence + BKN2 (§9.4.3.2.1).
	DMBurstNormal
)

func (k DMBurstKind) String() string {
	switch k {
	case DMBurstSync:
		return "DSB"
	case DMBurstNormal:
		return "DNB"
	default:
		return "?"
	}
}

// DMO burst geometry, in DIBITS (1 dibit per π/4-DQPSK symbol), relative to the
// LEAD dibit L of the burst's training sequence. Derived from EN 300 396-2
// Tables 15 (DNB) and 16 (DSB) by halving the on-air bit numbers:
//
//	DSB (Table 16):  freq-corr bits 15..94  → dibits  7..46  (40 dibits)
//	                 SCH/S BKN1 bits 95..214 → dibits 47..106 (60 dibits)
//	                 sync train bits 215..252 → dibits 107..125 (19 dibits, lead L=107)
//	                 BKN2       bits 253..468 → dibits 126..233 (108 dibits)
//	DNB (Table 15):  BKN1       bits 15..230  → dibits  7..114 (108 dibits)
//	                 norm train bits 231..252 → dibits 115..125 (11 dibits, lead L=115)
//	                 BKN2       bits 253..468 → dibits 126..233 (108 dibits)
const (
	dmSCHSDibits  = 60  // DSB BKN1 (SCH/S): 120 type-5 bits
	dmBlockDibits = 108 // a 216-type-5-bit block: DSB BKN2, both DNB blocks

	// Offsets relative to the training-sequence lead dibit L.
	dmDSBFreqCorrStart = -100 // L-100 .. L-60 (40 dibits)
	dmDSBBKN1Start     = -60  // L-60  .. L      (SCH/S, 60 dibits)
	dmDSBBKN2Start     = 19   // L+19  .. L+127  (108 dibits; 19-dibit sync train sits at L..L+18)
	dmDNBBKN1Start     = -108 // L-108 .. L      (108 dibits)
	dmDNBBKN2Start     = 11   // L+11  .. L+119  (108 dibits; 11-dibit normal train sits at L..L+10)
)

// DMBurst is one detected DMO burst, sliced into its scrambled type-5 blocks —
// the same block shape TrafficExtractor emits for TMO, ready for the shared
// descramble → channel-decode path. Slices are freshly allocated (safe to retain).
type DMBurst struct {
	Kind DMBurstKind
	// Lead is the absolute dibit index (baseIdx-relative) of the burst's
	// training-sequence lead dibit.
	Lead int
	// SCHS is the 60-dibit block 1 of a DSB (the SCH/S, 120 type-5 bits); nil for
	// a DNB, whose block 1 is the full 108-dibit BKN1.
	SCHS []uint8
	// BKN1, BKN2 are the 108-dibit (216 type-5-bit) blocks. BKN2 is set for both
	// burst kinds; BKN1 is set only for a DNB (a DSB's shorter block 1 is in SCHS,
	// so BKN1 is nil for a DSB).
	BKN1, BKN2 []uint8
	// Rotation is the residual π/4-DQPSK dibit rotation (0..3) at which the
	// training sequence correlated — the burst's dibits must be de-rotated by this
	// before channel decoding (rotateDibits(dibits, (4-Rotation)&3)).
	Rotation uint8
}

// ExtractDMBursts scans a demodulated DMO dibit stream and returns every Direct
// Mode Synchronisation Burst (DSB) and Normal Burst (DNB) whose training sequence
// correlates (within tolerance, under any of the four residual rotations) and that
// has enough surrounding dibits for its blocks to be sliced. It is stateless over
// a complete slice — the streaming/rolling-buffer form (mirroring TrafficExtractor)
// is added at scanner-integration time; a self-contained slice keeps the framing
// deterministically testable ahead of that.
//
// baseIdx is the absolute dibit index of dibits[0], carried into DMBurst.Lead so
// callers can order bursts across calls; pass 0 for a one-shot slice.
func ExtractDMBursts(dibits []uint8, baseIdx int) []DMBurst {
	var out []DMBurst

	// DSB: correlate the 19-dibit synchronisation training sequence (shared with
	// TMO's SB). Threshold 3 matches processSB / TrafficExtractor.
	sts := SyncTrainingDibits()
	out = appendDMBursts(out, dibits, baseIdx, DMBurstSync, sts, 3,
		dmDSBBKN1Start, dmSCHSDibits, dmDSBBKN2Start, dmBlockDibits)

	// DNB: correlate either 11-dibit normal training sequence. Threshold 2 matches
	// TrafficExtractor's normal-burst detectors.
	for _, nts := range [][]uint8{NormalSyncDibits(), NormalSyncDibits2()} {
		out = appendDMBursts(out, dibits, baseIdx, DMBurstNormal, nts, 2,
			dmDNBBKN1Start, dmBlockDibits, dmDNBBKN2Start, dmBlockDibits)
	}
	return out
}

// appendDMBursts correlates one training pattern (all four rotations) and slices
// the two blocks at (b1Start,b1Len) and (b2Start,b2Len) dibits relative to the
// training lead. For a DSB the first block is the 60-dibit SCH/S (placed in SCHS);
// for a DNB both blocks are 108-dibit (placed in BKN1/BKN2). De-duplicates a lead
// that correlates under more than one rotation (keeps the first).
func appendDMBursts(out []DMBurst, dibits []uint8, baseIdx int, kind DMBurstKind,
	pattern []uint8, tol, b1Start, b1Len, b2Start, b2Len int) []DMBurst {
	n := len(pattern)
	seen := map[int]struct{}{}
	for rot := uint8(0); rot < 4; rot++ {
		det := NewSyncDetector(rotateDibits(pattern, rot), tol)
		hits, _ := det.Process(nil, dibits, baseIdx)
		for _, trailing := range hits {
			lead := trailing - (n - 1) // absolute lead dibit
			rel := lead - baseIdx      // index into dibits
			if _, dup := seen[lead]; dup {
				continue
			}
			b1From, b1To := rel+b1Start, rel+b1Start+b1Len
			b2From, b2To := rel+b2Start, rel+b2Start+b2Len
			if b1From < 0 || b2To > len(dibits) {
				continue // burst runs off the buffer edge; skip (no partial slice)
			}
			seen[lead] = struct{}{}
			b := DMBurst{Kind: kind, Lead: lead, Rotation: rot}
			blk1 := cloneDibits(dibits[b1From:b1To])
			b.BKN2 = cloneDibits(dibits[b2From:b2To])
			if kind == DMBurstSync {
				b.SCHS = blk1
			} else {
				b.BKN1 = blk1
			}
			out = append(out, b)
		}
	}
	return out
}

func cloneDibits(s []uint8) []uint8 {
	c := make([]uint8, len(s))
	copy(c, s)
	return c
}
