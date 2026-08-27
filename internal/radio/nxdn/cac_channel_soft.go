package nxdn

import "github.com/MattCheramie/GopherTrunk/internal/radio/framing"

// DecodeCACChannelSoft is the soft-decision analog of DecodeCACChannel: the
// same §4.5.1.1 inverse chain — deinterleave → depuncture → K=5 Viterbi →
// CRC verify — carrying per-bit log-likelihood ratios instead of hard bits.
// Convention (framing/soft_tetra.go): LLR > 0 ⇒ bit 0, LLR < 0 ⇒ bit 1,
// magnitude is reliability; the 50 punctured positions become 0.0 erasures
// (no branch-metric contribution — the soft replacement for
// DepunctureMark). Input is CACChannelBits LLRs in on-air order.
//
// Returns the 155-bit info block and a CRC ok flag, exactly as
// DecodeCACChannel does. Opt-in via nxdn_soft_decision — the coding gain of
// true per-bit soft Viterbi over the hard slicer is what recovers marginal
// on-air CAC bursts (measured on the shared K=5 primitive: ~8× info-bit
// error reduction at the AWGN level where the hard path loses one bit in
// eleven).
func DecodeCACChannelSoft(channel []float32) ([]byte, bool) {
	if len(channel) != CACChannelBits {
		return nil, false
	}
	// Inverse interleave: punctured[perm[k]] = channel[k].
	punctured := make([]float32, CACChannelBits)
	for k := 0; k < CACChannelBits; k++ {
		punctured[cacInterleavePerm[k]] = channel[k]
	}
	// Depuncture: dropped positions stay 0.0 (erasure), restoring the
	// 350-LLR stream the K=5 decoder consumes.
	const preDepunctureLen = 2 * (CACInfoBits + 16 + cacTailBits)
	depunctured := make([]float32, preDepunctureLen)
	src := 0
	punc := 0
	for i := 0; i < preDepunctureLen; i++ {
		if punc < len(cacPuncturePositions) && cacPuncturePositions[punc] == i {
			punc++
			continue
		}
		depunctured[i] = punctured[src]
		src++
	}
	const stages = CACInfoBits + 16 + cacTailBits
	all, _ := framing.ViterbiK5Soft(depunctured, stages)
	info := all[:CACInfoBits]
	want := cacCRC16(info)
	var got uint16
	for i := 0; i < 16; i++ {
		got = (got << 1) | uint16(all[CACInfoBits+i]&1)
	}
	return info, got == want
}
