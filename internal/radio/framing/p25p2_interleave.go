package framing

// P25 Phase 2 per-burst MAC PDU block interleaver.
//
// TIA-102.BBAC wraps the trellis-coded MAC PDU in a block interleaver
// before it goes on air, so a burst error on the channel is spread
// across the trellis trellis-decoder's input rather than landing as one
// long run the Viterbi decoder cannot absorb. The exact permutation
// table is not in the repo's spec PDFs; this is the project's working
// model — a 2-row block interleaver (write row-major, read
// column-major), which spreads any pair of originally-adjacent dibits
// half a burst apart. InterleaveMACBurst and DeinterleaveMACBurst are
// exact inverses, so a fixture round-trips regardless of whether the
// permutation matches the spec; a correction here is one local change.

// macInterleaveRows is the row count of the MAC-burst block interleaver.
const macInterleaveRows = 2

// InterleaveMACBurst applies the block interleaver to a MAC-burst dibit
// slice (write row-major into a macInterleaveRows×N matrix, read
// column-major). A slice whose length is not a multiple of
// macInterleaveRows is returned unchanged — the interleaver is a no-op
// rather than a corrupting partial permutation.
func InterleaveMACBurst(in []uint8) []uint8 {
	n := len(in)
	if n == 0 || n%macInterleaveRows != 0 {
		return append([]uint8(nil), in...)
	}
	cols := n / macInterleaveRows
	out := make([]uint8, n)
	for r := 0; r < macInterleaveRows; r++ {
		for c := 0; c < cols; c++ {
			out[c*macInterleaveRows+r] = in[r*cols+c]
		}
	}
	return out
}

// DeinterleaveMACBurstC applies the exact same de-interleave permutation as
// DeinterleaveMACBurst to a parallel complex64 slice (the soft-decision
// per-dibit differential, issue #915), keeping the soft sample aligned with its
// dibit through the block interleaver. Lengths not a multiple of
// macInterleaveRows are returned unchanged, matching the dibit path.
func DeinterleaveMACBurstC(in []complex64) []complex64 {
	n := len(in)
	if n == 0 || n%macInterleaveRows != 0 {
		return append([]complex64(nil), in...)
	}
	cols := n / macInterleaveRows
	out := make([]complex64, n)
	for r := 0; r < macInterleaveRows; r++ {
		for c := 0; c < cols; c++ {
			out[r*cols+c] = in[c*macInterleaveRows+r]
		}
	}
	return out
}

// DeinterleaveMACBurst is the exact inverse of InterleaveMACBurst: it
// recovers the original dibit order from a block-interleaved MAC burst.
func DeinterleaveMACBurst(in []uint8) []uint8 {
	n := len(in)
	if n == 0 || n%macInterleaveRows != 0 {
		return append([]uint8(nil), in...)
	}
	cols := n / macInterleaveRows
	out := make([]uint8, n)
	for r := 0; r < macInterleaveRows; r++ {
		for c := 0; c < cols; c++ {
			out[r*cols+c] = in[c*macInterleaveRows+r]
		}
	}
	return out
}
