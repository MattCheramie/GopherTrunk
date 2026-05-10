package phase1

// 98-dibit block interleaver for the P25 Phase 1 TSBK / data-block
// channel, per TIA-102.BAAA-A Annex A. The interleaver scrambles the
// trellis-coded dibit stream so a contiguous on-air error burst surfaces
// as scattered single-dibit errors that the Viterbi decoder can still
// correct.
//
// The two permutation tables below are inverses of each other; the
// TestInterleaverIsInvertible check pins this at build time.

// tsbkInterleavePerm[i] = j means channel[i] = coding[j] (encoder side).
// Sourced from TIA-102.BAAA-A Annex A; cross-verified against
// kchmck/p25.rs, OP25, and DSDPlus.
var tsbkInterleavePerm = [98]int{
	0, 1, 8, 9, 16, 17, 24, 25, 32, 33, 40, 41, 48, 49, 56, 57, 64, 65, 72, 73, 80, 81, 88, 89, 96, 97,
	2, 3, 10, 11, 18, 19, 26, 27, 34, 35, 42, 43, 50, 51, 58, 59, 66, 67, 74, 75, 82, 83, 90, 91,
	4, 5, 12, 13, 20, 21, 28, 29, 36, 37, 44, 45, 52, 53, 60, 61, 68, 69, 76, 77, 84, 85, 92, 93,
	6, 7, 14, 15, 22, 23, 30, 31, 38, 39, 46, 47, 54, 55, 62, 63, 70, 71, 78, 79, 86, 87, 94, 95,
}

// tsbkDeinterleavePerm[i] = j means coding[i] = channel[j] (decoder side).
var tsbkDeinterleavePerm = [98]int{
	0, 1, 26, 27, 50, 51, 74, 75,
	2, 3, 28, 29, 52, 53, 76, 77,
	4, 5, 30, 31, 54, 55, 78, 79,
	6, 7, 32, 33, 56, 57, 80, 81,
	8, 9, 34, 35, 58, 59, 82, 83,
	10, 11, 36, 37, 60, 61, 84, 85,
	12, 13, 38, 39, 62, 63, 86, 87,
	14, 15, 40, 41, 64, 65, 88, 89,
	16, 17, 42, 43, 66, 67, 90, 91,
	18, 19, 44, 45, 68, 69, 92, 93,
	20, 21, 46, 47, 70, 71, 94, 95,
	22, 23, 48, 49, 72, 73, 96, 97,
	24, 25,
}

// InterleaveTSBK permutes 98 coding-order dibits into channel-order
// dibits ready for transmission.
func InterleaveTSBK(coding []uint8) []uint8 {
	if len(coding) != 98 {
		panic("p25/phase1: interleaver input must be 98 dibits")
	}
	out := make([]uint8, 98)
	for i := 0; i < 98; i++ {
		out[i] = coding[tsbkInterleavePerm[i]]
	}
	return out
}

// DeinterleaveTSBK permutes 98 received channel-order dibits back into
// coding-order ready for the Viterbi decoder.
func DeinterleaveTSBK(channel []uint8) []uint8 {
	if len(channel) != 98 {
		panic("p25/phase1: deinterleaver input must be 98 dibits")
	}
	out := make([]uint8, 98)
	for i := 0; i < 98; i++ {
		out[i] = channel[tsbkDeinterleavePerm[i]]
	}
	return out
}
