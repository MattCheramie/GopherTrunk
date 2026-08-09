package tetra

import "github.com/MattCheramie/GopherTrunk/internal/radio/framing"

// DMO (Direct Mode Operation) channel decode — the descramble → channel-decode
// layer that turns the scrambled type-5 blocks ExtractDMBursts slices out of a
// DSB/DNB (dmo.go) into decoded content: the DSB's SCH/S synchronisation block,
// the DSB's SCH/H, and the DNB's TCH/S speech or SCH/F short-data.
//
// The headline finding of the #1003 investigation, confirmed against ETSI
// EN 300 396-2 V1.4.1, is that DMO's channel coding is NOT a new code family —
// it is the SAME per-channel chains this package already implements for TMO,
// with two DMO-specific rules the spec pins exactly:
//
//   - SCH/S  §8.3.1.1:  60 type-1 → (76,60) block code + 4 tail → RCPC r=2/3 →
//     (120,11) interleave → scramble. Identical to the TMO BSCH chain, so
//     DecodeBSCH decodes it. Per §8.2.5.2 the SCH/S (and SCH/H) of a DSB are
//     scrambled with the colour code set to ZERO — exactly like TMO's BSCH.
//   - SCH/H  §8.3.1.2:  124 type-1 → (140,124) + 4 tail → RCPC r=2/3 →
//     (216,101) interleave → scramble. Identical to TMO SCH/HD (DecodeSCHHD).
//   - SCH/F  §8.3.1.3:  268 type-1 → (284,268) + 4 tail → RCPC r=2/3 →
//     (432,103) interleave → scramble. Identical to TMO SCH/F (DecodeSCHF).
//   - TCH/S  §8.3.2.4:  the 432 type-4 speech bits are defined by EN 300 395-2
//     (the SAME ACELP speech construction GT already decodes for TMO), then
//     scrambled and split into BKN1(216)+BKN2(216). So DecodeTCHS decodes it
//     once the two blocks are concatenated and descrambled.
//
// The scrambler polynomial (§8.2.5.2) is byte-for-byte the TMO one
// (framing.ScrambleTetra); only the seed differs — DMO seeds it from the 30-bit
// DM colour code, versus TMO's MCC/MNC/colour extended code. For the DSB
// signalling the seed is 0 (the rule above), and for the DNB traffic it is the
// DM colour code. The DM colour code itself is signalled in the SYNC PDU that
// rides in SCH/S, whose message layout is EN 300 396-3 (a separate spec); until
// that PDU parser lands, callers pass the colour code explicitly (0 is both the
// spec default for the DSB blocks and the common radio-to-radio DMO value).
//
// Both a hard-decision path (DMBurstTCHSpeech, off the sliced dibits) and a
// soft-decision path (DMBurstTCHSpeechSoft, off the per-symbol differentials
// ExtractDMBurstsSoft carries in DMBurst.SoftBKN1/SoftBKN2) are provided. The
// soft path is the DMO analog of the TMO TrafficExtractor.softFrame bridge and
// gives the same ~2× same-carrier yield lever (#1001); callers try soft first
// and fall back to hard when no differentials were stashed.

// dmDerotatedBits converts a burst's sliced dibit block back to on-air type-5
// bits, undoing the residual π/4-DQPSK rotation (0..3) the detector reported for
// the burst. ExtractDMBursts correlates the training sequence under all four
// rotations and records which one matched in DMBurst.Rotation; de-rotating by
// (4-Rotation)&3 returns the transmitted dibits before the Gray→bit expansion
// (the same relationship TestExtractDMBurstsRotation pins).
func dmDerotatedBits(block []uint8, rot uint8) []byte {
	return TetraDibitsToBits(rotateDibits(block, (4-rot)&3))
}

// DecodeDMSCHS decodes the SCH/S carried in a DSB's block 1 (120 type-5 bits) to
// its 60 type-1 synchronisation bits, returning a CRC-pass flag. Per
// EN 300 396-2 §8.2.5.2 a DSB's SCH/S is scrambled with colour code 0, so this
// is the DMO analog of the TMO BSCH decode. The 60 type-1 bits are the SYNC PDU
// (EN 300 396-3), which carries the DM colour code and the master's slot/frame
// numbering used to anchor the DNB traffic that follows.
func DecodeDMSCHS(b DMBurst) (type1 []byte, crcOK bool) {
	if b.Kind != DMBurstSync || len(b.SCHS) != dmSCHSDibits {
		return nil, false
	}
	return DecodeBSCH(dmDerotatedBits(b.SCHS, b.Rotation))
}

// DecodeDMSCHH decodes the SCH/H carried in a DSB's block 2 (216 type-5 bits) to
// its 124 type-1 bits, with a CRC-pass flag. Colour code 0 for a DSB's SCH/H
// (§8.2.5.2), so it reuses the TMO SCH/HD chain.
func DecodeDMSCHH(b DMBurst) (type1 []byte, crcOK bool) {
	if b.Kind != DMBurstSync || len(b.BKN2) != dmBlockDibits {
		return nil, false
	}
	return DecodeSCHHD(dmDerotatedBits(b.BKN2, b.Rotation), 0)
}

// dmDNBType5 concatenates a DNB's two 108-dibit blocks (BKN1 then BKN2) into the
// 432 on-air type-5 bits of the full slot, de-rotated but still scrambled.
// Returns nil if b is not a well-formed DNB.
func dmDNBType5(b DMBurst) []byte {
	if b.Kind != DMBurstNormal || len(b.BKN1) != dmBlockDibits || len(b.BKN2) != dmBlockDibits {
		return nil
	}
	bits := make([]byte, 0, 2*dmBlockDibits*2)
	bits = append(bits, dmDerotatedBits(b.BKN1, b.Rotation)...)
	bits = append(bits, dmDerotatedBits(b.BKN2, b.Rotation)...)
	return bits
}

// DMBurstTCHSpeech decodes a DNB carrying full-slot TCH/S speech into its two
// 137-bit speech frames (packed MSB-first, 18 bytes each), ready for the ACELP
// vocoder — or nil when the class-2 CRC fails (a non-speech burst, a signalling
// slot, or a corrupted slot), exactly like the TMO TCHSpeechFrames gate. colour
// is the 30-bit DM colour code the traffic is scrambled with (§8.2.5.2); pass 0
// for the spec default / common radio-to-radio DMO.
//
// The descramble is UNCONDITIONAL — including at colour 0. TETRA scrambling is
// non-identity at colour 0 (NewScramblerTetra(0) seeds the LFSR to 0xC0000000,
// §8.2.5.2 eq. 8.42), so a clear (TEA0) colour-0 DMO transmitter still scrambles
// TCH/S with that seed, exactly as the DSB signalling is scrambled with colour 0
// and DecodeDMSCHS/DecodeBSCH always descramble it. The earlier `if colour != 0`
// skip (copied from TMO traffic.go, where the extended colour code is never 0)
// left a real colour-0 DNB scrambled going into the Viterbi/CRC — the uniform
// ~1/256 "half-wrong" metric the #1003 investigation measured and misread as
// air-interface encryption. Descrambling with 0 is the fix (issue #1003).
func DMBurstTCHSpeech(b DMBurst, colour uint32) [][]byte {
	type5 := dmDNBType5(b)
	if type5 == nil {
		return nil
	}
	type5 = framing.DescrambleTetra(type5, colour)
	return TCHSpeechFrames(framing.PackBitsMSB(type5))
}

// DMBurstTCHSpeechSoft is the soft-decision analog of DMBurstTCHSpeech: it
// decodes a DNB's TCH/S speech from the per-symbol complex differentials
// ExtractDMBurstsSoft stashed in b.SoftBKN1/SoftBKN2 (rather than the hard-sliced
// dibits), running the soft-Viterbi TCH/S chain that recovers several more
// corrupted bursts per the #1001 soft-decision win. Returns the two 137-bit
// speech frames when the class-2 CRC passes, or nil — including when no soft
// info is present (a hard-only extraction), so a caller can fall back to
// DMBurstTCHSpeech. colour is the 30-bit DM colour code (0 for the default).
//
// The differentials are de-rotated by the same (4-Rotation)&3 the hard
// dmDerotatedBits applies: softType5FromDiffs(diffs, r) hard-slices to exactly
// TetraDibitsToBits(rotateDibits(dibits, r)), so passing r=(4-Rotation)&3 makes
// the soft LLR stream the differential twin of the hard dmDNBType5 bits, and
// DescrambleTetraSoft applies the same colour-code sign flips as DescrambleTetra.
func DMBurstTCHSpeechSoft(b DMBurst, colour uint32) [][]byte {
	if b.Kind != DMBurstNormal || len(b.SoftBKN1) != dmBlockDibits || len(b.SoftBKN2) != dmBlockDibits {
		return nil
	}
	diffs := make([]complex64, 0, 2*dmBlockDibits)
	diffs = append(diffs, b.SoftBKN1...)
	diffs = append(diffs, b.SoftBKN2...)
	llr := softType5FromDiffs(diffs, (4-b.Rotation)&3)
	// Descramble unconditionally, including at colour 0 (seed 0xC0000000 is
	// non-identity) — the soft twin of the DMBurstTCHSpeech fix (issue #1003).
	llr = framing.DescrambleTetraSoft(llr, colour)
	return TCHSpeechFramesSoft(llr)
}

// DecodeDMSCHF decodes a DNB carrying SCH/F short-data signalling (the full-slot
// 432→268 chain) to its 268 type-1 bits, with a CRC-pass flag. colour is the
// 30-bit DM colour code (0 for the default). SCH/F rides a DNB (a DM-SDU short
// data message), distinct from the TCH/S speech a DNB carries during a voice
// call; a caller distinguishes them by which decode's CRC passes.
func DecodeDMSCHF(b DMBurst, colour uint32) (type1 []byte, crcOK bool) {
	type5 := dmDNBType5(b)
	if type5 == nil {
		return nil, false
	}
	return DecodeSCHF(type5, colour)
}
