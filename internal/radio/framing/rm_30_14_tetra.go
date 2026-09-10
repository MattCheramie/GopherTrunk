package framing

import (
	"math"
	"math/bits"
	"sync"
)

// Shortened (30,14) Reed-Muller block code per ETSI EN 300 392-2
// §8.2.3.2. Used by the TETRA AACH (Access Assignment Channel) to
// encode 14 type-1 information bits into 30 type-2 channel bits with
// no further convolutional coding or interleaving — AACH's
// type-4 block equals its type-2 block.
//
// Generator matrix G is [I_14 | P] where I_14 is the 14×14 identity
// (so the code is systematic — the first 14 type-2 bits equal the
// 14 type-1 bits) and P is the 14×16 parity matrix from
// equation (8.13):
//
//	row 1:  1 0 0 1 1 0 1 1 0 1 1 0 0 0 0 0
//	row 2:  0 0 1 0 1 1 0 1 1 1 1 0 0 0 0 0
//	row 3:  1 1 1 1 1 1 0 0 0 0 1 0 0 0 0 0
//	row 4:  1 1 1 0 0 0 0 0 0 0 1 1 1 1 0 0
//	row 5:  1 0 0 1 1 0 0 0 0 0 1 1 1 0 1 0
//	row 6:  0 1 0 1 0 1 0 0 0 0 1 1 0 1 1 0
//	row 7:  0 0 1 0 1 1 0 0 0 0 1 0 1 1 1 0
//	row 8:  1 1 1 1 1 1 1 1 1 1 0 1 1 1 1 1
//	row 9:  1 0 0 0 0 0 1 1 0 0 1 1 1 0 0 1
//	row 10: 0 1 0 0 0 0 1 0 1 0 1 1 0 1 0 1
//	row 11: 0 0 1 0 0 0 0 1 1 0 1 0 1 1 0 1
//	row 12: 0 0 0 1 0 0 1 0 0 1 1 1 0 0 1 1
//	row 13: 0 0 0 0 1 0 0 1 0 1 1 0 1 0 1 1
//	row 14: 0 0 0 0 0 1 0 0 1 1 1 0 0 1 1 1
//
// Encoding rule: b_2 = b_1 * G over GF(2).

// rm3014ParityMatrix is the 14×16 parity portion P of the (30,14)
// generator matrix. rm3014ParityMatrix[r][c] is row r, column c.
var rm3014ParityMatrix = [14][16]byte{
	{1, 0, 0, 1, 1, 0, 1, 1, 0, 1, 1, 0, 0, 0, 0, 0},
	{0, 0, 1, 0, 1, 1, 0, 1, 1, 1, 1, 0, 0, 0, 0, 0},
	{1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0},
	{1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 0, 0},
	{1, 0, 0, 1, 1, 0, 0, 0, 0, 0, 1, 1, 1, 0, 1, 0},
	{0, 1, 0, 1, 0, 1, 0, 0, 0, 0, 1, 1, 0, 1, 1, 0},
	{0, 0, 1, 0, 1, 1, 0, 0, 0, 0, 1, 0, 1, 1, 1, 0},
	{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 1, 1, 1},
	{1, 0, 0, 0, 0, 0, 1, 1, 0, 0, 1, 1, 1, 0, 0, 1},
	{0, 1, 0, 0, 0, 0, 1, 0, 1, 0, 1, 1, 0, 1, 0, 1},
	{0, 0, 1, 0, 0, 0, 0, 1, 1, 0, 1, 0, 1, 1, 0, 1},
	{0, 0, 0, 1, 0, 0, 1, 0, 0, 1, 1, 1, 0, 0, 1, 1},
	{0, 0, 0, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 1, 1},
	{0, 0, 0, 0, 0, 1, 0, 0, 1, 1, 1, 0, 0, 1, 1, 1},
}

// EncodeRM3014Tetra encodes 14 type-1 information bits (each entry
// 0/1, MSB-first per the spec convention) into 30 type-2 channel
// bits via the shortened (30,14) Reed-Muller code from
// EN 300 392-2 §8.2.3.2. The first 14 output bits equal the input
// (systematic encoding); the trailing 16 bits are the parity sum
// of the input vector dotted into each column of the parity matrix.
func EncodeRM3014Tetra(info []byte) []byte {
	if len(info) != 14 {
		return nil
	}
	out := make([]byte, 30)
	// Systematic: first 14 output bits equal the input bits.
	for i := 0; i < 14; i++ {
		out[i] = info[i] & 1
	}
	// Parity: for each of the 16 parity columns, XOR the input
	// bits whose row has a 1 in that column.
	for c := 0; c < 16; c++ {
		var p byte
		for r := 0; r < 14; r++ {
			if info[r]&1 != 0 && rm3014ParityMatrix[r][c] != 0 {
				p ^= 1
			}
		}
		out[14+c] = p
	}
	return out
}

// rm3014Codebook is every valid (30,14) codeword packed MSB-first into a
// uint32 (codeword bit i at bit position 29-i), indexed by its 14 information
// bits (the systematic prefix, so index v ⇔ info bits = v MSB-first). Built once
// on first use: the AACH decoders below are maximum-likelihood searches over all
// 2^14 codewords, and re-ENCODING every codeword per call (the previous
// implementation: 16 384 EncodeRM3014Tetra calls, each allocating) made one
// hard AACH decode cost ~16 k allocations and several hundred microseconds.
// On a locked TETRA control channel that decode runs once per downlink slot
// (~70/s per carrier) plus once per traffic burst in the voice demux, so it was
// ~65% of the wideband pump's CPU on the 10 Sep dual-TETRA X310 rig and the
// dominant source of its GC churn (~19 collections/s on a 9 MB heap) — i.e. the
// "host overruns at a tiny 200 kS/s" report. A table lookup + popcount search
// is ~50× cheaper and allocation-free on the search path.
var (
	rm3014Once     sync.Once
	rm3014Codebook [1 << 14]uint32
	// rm3014RowMask[r] is parity row r of rm3014ParityMatrix packed as a
	// 16-bit mask (column c at bit 15-c), so a codeword's parity is the XOR
	// of the row masks of its set information bits.
	rm3014RowMask [14]uint32
)

func rm3014Table() *[1 << 14]uint32 {
	rm3014Once.Do(func() {
		for r := 0; r < 14; r++ {
			var m uint32
			for c := 0; c < 16; c++ {
				if rm3014ParityMatrix[r][c] != 0 {
					m |= 1 << uint(15-c)
				}
			}
			rm3014RowMask[r] = m
		}
		for v := 0; v < 1<<14; v++ {
			var parity uint32
			for r := 0; r < 14; r++ {
				if v&(1<<uint(13-r)) != 0 {
					parity ^= rm3014RowMask[r]
				}
			}
			rm3014Codebook[v] = uint32(v)<<16 | parity
		}
	})
	return &rm3014Codebook
}

// rm3014Pack packs 30 hard bits MSB-first into the codebook's uint32 layout.
func rm3014Pack(bits []byte) uint32 {
	var w uint32
	for i := 0; i < 30; i++ {
		w = w<<1 | uint32(bits[i]&1)
	}
	return w
}

// rm3014Info unpacks the 14 information bits of codebook index v MSB-first.
func rm3014Info(v int) []byte {
	out := make([]byte, 14)
	for i := 0; i < 14; i++ {
		out[i] = byte((v >> uint(13-i)) & 1)
	}
	return out
}

// DecodeRM3014Tetra decodes 30 received bits via minimum-Hamming-
// distance search across all 2^14 = 16 384 valid codewords. Returns
// (info, errs) where info is the 14-bit information vector closest
// to the received word and errs is the Hamming distance to that
// codeword. errs = 0 means the received word is a valid codeword.
//
// For a clean channel this recovers the original info exactly; for
// noisier captures the (30,14) RM code has minimum distance ≥ 4
// (giving guaranteed single-bit correction across the 30-bit
// codeword) and detect-up-to-3 capability — beyond that the
// closest-codeword decoder may mis-correct. Ties resolve to the
// lowest information vector, as the original per-codeword re-encode
// search did.
func DecodeRM3014Tetra(received []byte) ([]byte, int) {
	if len(received) != 30 {
		return nil, -1
	}
	table := rm3014Table()
	r := rm3014Pack(received)
	bestDist, bestV := 31, 0
	for v, cw := range table {
		d := bits.OnesCount32(cw ^ r)
		if d < bestDist {
			bestDist, bestV = d, v
			if d == 0 {
				break
			}
		}
	}
	return rm3014Info(bestV), bestDist
}

// DecodeRM3014TetraSoft decodes 30 soft type-4 LLRs via soft-decision
// maximum-likelihood search across the same 2^14 valid codewords as the hard
// DecodeRM3014Tetra, but maximising the soft correlation instead of minimising
// Hamming distance. The LLR sign convention is the one softType5FromDiffs /
// DescrambleTetraSoft produce: positive ⇒ bit 0, negative ⇒ bit 1, magnitude ⇒
// confidence. Soft-decision decoding of this short block code recovers the AACH
// usage marker on marginal bursts a hard decoder mis-corrects (~2 dB of coding
// gain), so more concurrent-call bursts carry a routable marker instead of being
// dropped.
//
// Returns the 14 information bits of the ML codeword, its Hamming distance to the
// hard-sliced input (so callers can keep the DecodeRM3014Tetra confidence gate),
// and margin = (best − second-best correlation) / (2·Σ|llr|) ∈ [0, 1] — 0 when
// two codewords fit equally well, 1 when the input is a clean codeword.
//
// The correlation Σ llr_i·(1−2c_i) = T − 2·Σ_{c_i=1} llr_i is evaluated per
// codeword from three 10-bit partial-sum tables (one per third of the 30-bit
// word) rather than a 30-term dot product, so the 16 384-codeword search is
// ~50 k adds instead of ~500 k and allocation-free.
func DecodeRM3014TetraSoft(llr []float32) (info []byte, dist int, margin float64) {
	if len(llr) != 30 {
		return nil, -1, 0
	}
	table := rm3014Table()
	// tab[k][x]: Σ llr over the set bits of the 10-bit group x, where group 0
	// is codeword bits 0..9 (uint32 bits 29..20), group 1 bits 10..19, group 2
	// bits 20..29.
	var tab [3][1024]float64
	var total float64
	for i := 0; i < 30; i++ {
		total += float64(llr[i])
	}
	for k := 0; k < 3; k++ {
		for x := 1; x < 1024; x++ {
			low := x & -x
			b := bits.TrailingZeros32(uint32(low)) // bit index within the group, LSB = last bit of the group
			i := k*10 + (9 - b)
			tab[k][x] = tab[k][x&(x-1)] + float64(llr[i])
		}
	}
	best, second := math.Inf(-1), math.Inf(-1)
	bestV := 0
	for v, cw := range table {
		s := tab[0][cw>>20] + tab[1][(cw>>10)&1023] + tab[2][cw&1023]
		m := total - 2*s
		switch {
		case m > best:
			second = best
			best = m
			bestV = v
		case m > second:
			second = m
		}
	}
	cw := table[bestV]
	var energy float64
	for i := 0; i < 30; i++ {
		hb := uint32(0)
		if llr[i] < 0 {
			hb = 1
			energy -= float64(llr[i])
		} else {
			energy += float64(llr[i])
		}
		if (cw>>uint(29-i))&1 != hb {
			dist++
		}
	}
	if energy > 0 {
		margin = (best - second) / (2 * energy) // best-second ∈ [0, 2·energy]
	}
	return rm3014Info(bestV), dist, margin
}
