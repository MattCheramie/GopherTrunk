package tetra

import (
	"sync"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// Exact recovery of the TETRA scrambling seed from a single TCH/S burst.
//
// Every stage between the 30-bit extended colour code and the on-air type-5 bits
// of a TCH/S slot is LINEAR over GF(2):
//
//   - the scrambling sequence p(k) is the output of a 32-stage LFSR whose state
//     is initialised to (e(1..30), 1, 1) (EN 300 392-2 §8.2.5.2), so the
//     432-bit PN sequence is an affine function of the seed:
//     PN(seed) = PN(0) ⊕ Σ_i seed_i · (PN(1<<i) ⊕ PN(0));
//   - the TCH/S channel coding (EN 300 395-2 §5.5) is a block code once the 4
//     tail bits are fixed: class 1 + class 2 bits → CRC (a fixed parity-check
//     matrix, tchCRCTaps) → K=5 convolutional mother code → puncturing. So the
//     330 coded type-3 bits c satisfy H·c = 0 for the code's parity-check
//     matrix H, whatever the speech content (the 102 class-0 bits are uncoded
//     and carry no constraint);
//   - the 24×18 interleave is a permutation.
//
// Hence for an error-free received type-5 block r = interleave(c) ⊕ PN(seed):
//
//	H' · (r ⊕ PN(0)) = (H' · P) · seed
//
// where H' is H moved to type-5 coordinates and P the 432×30 matrix of seed
// impulse responses. That is 158 linear equations in 30 unknowns — an
// error-free burst solves the seed EXACTLY, and a burst with bit errors is
// almost surely inconsistent (128 redundant checks) and rejected. No search over
// candidate colours, no assumption about how the seed is composed (MCC / MNC /
// colour), and no knowledge of the speech: this is what makes it possible to
// read the DM colour code straight off the traffic of a Direct Mode call whose
// DM-SYNC field layout is not trusted (#1003).
//
// RecoverDMScrambleSeed votes the per-burst solutions across a transmission and
// confirms the winner by CRC-decoding the bursts with it, so the result carries
// the same evidence RecoverDMColourCode's brute force did, at a fraction of the
// cost (one 158×31 elimination per burst instead of 64 Viterbi decodes).

const (
	tchSeedBits  = 30
	tchCodedBits = tchClass1Coded + tchClass2Coded // 330
	tchInfoBits  = tchClass1Bits + tchClass2Bits   // 172 free information bits
	tchCheckBits = tchCodedBits - tchInfoBits      // 158 parity checks
	tchWords     = (tchType3Bits + 63) / 64        // 7 uint64 words per 432-bit row
)

// tchSeedLinear is the precomputed linear structure (built once, lazily).
type tchSeedLinear struct {
	// pn0 is PN(seed 0) in type-5 order; pnCol[i] = PN(1<<i) ⊕ pn0.
	pn0   [tchWords]uint64
	pnCol [tchSeedBits][tchWords]uint64
	// checks are the parity-check rows in type-5 coordinates (packed bits).
	checks [][tchWords]uint64
	// a[r] is row r of H'·P as a 30-bit mask (bit i = check r applied to pnCol[i]);
	// b0[r] is check r applied to pn0.
	a  []uint32
	b0 []byte

	// sparse are LOCAL parity checks (supported on a window of ~48 consecutive
	// coded bits, from the convolutional code's short memory), with their
	// type-5 support lists, seed masks and constants. A bit error only spoils
	// the few sparse checks whose window contains it, so a burst that fails the
	// dense solve can still be solved from its RELIABLE checks (see
	// SolveTCHScrambleSeedSoft).
	sparse []sparseCheck
}

type sparseCheck struct {
	support []int // type-5 bit positions
	a       uint32
	b0      byte
}

// Sparse-check window geometry: tchSparseWindow consecutive coded type-3 bits
// per window, windows tchSparseStep apart.
const (
	tchSparseWindow = 48
	tchSparseStep   = 12
)

var (
	tchSeedOnce sync.Once
	tchSeedLin  *tchSeedLinear
)

func packBits432(bits []byte) (w [tchWords]uint64) {
	for i, b := range bits {
		if b&1 != 0 {
			w[i>>6] |= 1 << uint(i&63)
		}
	}
	return w
}

func parityWords(a, b *[tchWords]uint64) byte {
	var x uint64
	for i := range a {
		x ^= a[i] & b[i]
	}
	return byte(popcount64(x) & 1)
}

func popcount64(v uint64) int {
	v = v - ((v >> 1) & 0x5555555555555555)
	v = (v & 0x3333333333333333) + ((v >> 2) & 0x3333333333333333)
	v = (v + (v >> 4)) & 0x0F0F0F0F0F0F0F0F
	return int((v * 0x0101010101010101) >> 56)
}

// tchCodedBitsFor encodes one 172-bit information vector (class 1 ++ class 2)
// through the CRC + RCPC chain to the 330 coded type-3 bits, exactly as
// EncodeTCHS does.
func tchCodedBitsFor(info []byte) []byte {
	class1 := info[:tchClass1Bits]
	class2 := info[tchClass1Bits:]
	conv := make([]byte, 0, tchConvIn)
	conv = append(conv, class1...)
	conv = append(conv, class2...)
	conv = append(conv, crcTCHClass2(class2)...)
	conv = append(conv, make([]byte, tchTailBits)...)
	mother := framing.EncodeRCPCTetraMother(conv)
	split := 3 * tchClass1Bits
	c1 := framing.PunctureRCPCTetra(mother[:split], framing.RCPCTetraPeriod23, framing.RCPCTetraPuncture23, tchClass1Coded)
	c2 := framing.PunctureRCPCTetra(mother[split:], framing.RCPCTetraPeriod818, framing.RCPCTetraPuncture818, tchClass2Coded)
	out := make([]byte, 0, tchCodedBits)
	out = append(out, c1...)
	out = append(out, c2...)
	return out
}

// tchType5IndexOfCoded maps coded type-3 bit k (0..329, following the 102
// class-0 bits) to its type-5 position through the 24×18 interleave
// (tchInterleave: type4[i*24+j] = type3[j*18+i]).
func tchType5IndexOfCoded(k int) int {
	t := tchClass0Bits + k
	return (t%18)*24 + t/18
}

// buildTCHSeedLinear derives the parity checks of the TCH/S coded block by
// Gaussian elimination on the generator matrix, then moves them to type-5
// coordinates and precomputes their action on the seed impulse responses.
func buildTCHSeedLinear() *tchSeedLinear {
	lin := &tchSeedLinear{}

	// PN structure.
	pn0 := framing.NewScramblerTetra(0).Generate(tchType3Bits)
	lin.pn0 = packBits432(pn0)
	for i := 0; i < tchSeedBits; i++ {
		pi := framing.NewScramblerTetra(1 << uint(i)).Generate(tchType3Bits)
		for k := range pi {
			pi[k] ^= pn0[k]
		}
		lin.pnCol[i] = packBits432(pi)
	}

	// Generator rows: g[j] = coded bits for information unit vector j, packed
	// as 330-bit rows (6 words).
	const codedWords = (tchCodedBits + 63) / 64
	type row [codedWords]uint64
	gen := make([]row, tchInfoBits)
	for j := 0; j < tchInfoBits; j++ {
		info := make([]byte, tchInfoBits)
		info[j] = 1
		for k, b := range tchCodedBitsFor(info) {
			if b&1 != 0 {
				gen[j][k>>6] |= 1 << uint(k&63)
			}
		}
	}
	// Reduced row echelon form of the 172×330 generator; record pivot columns.
	pivotCol := make([]int, 0, tchInfoBits)
	r := 0
	for col := 0; col < tchCodedBits && r < tchInfoBits; col++ {
		sel := -1
		for i := r; i < tchInfoBits; i++ {
			if gen[i][col>>6]&(1<<uint(col&63)) != 0 {
				sel = i
				break
			}
		}
		if sel < 0 {
			continue
		}
		gen[r], gen[sel] = gen[sel], gen[r]
		for i := 0; i < tchInfoBits; i++ {
			if i != r && gen[i][col>>6]&(1<<uint(col&63)) != 0 {
				for w := range gen[i] {
					gen[i][w] ^= gen[r][w]
				}
			}
		}
		pivotCol = append(pivotCol, col)
		r++
	}
	isPivot := make([]bool, tchCodedBits)
	for _, c := range pivotCol {
		isPivot[c] = true
	}
	// Null-space basis: for each free column f, the vector with h[f]=1 and
	// h[pivot_i] = gen[i][f] satisfies h·g_j = 0 for every generator row.
	for f := 0; f < tchCodedBits; f++ {
		if isPivot[f] {
			continue
		}
		var h [tchWords]uint64
		set := func(k int) { h[k>>6] |= 1 << uint(k&63) }
		set(tchType5IndexOfCoded(f))
		for i, pc := range pivotCol {
			if gen[i][f>>6]&(1<<uint(f&63)) != 0 {
				set(tchType5IndexOfCoded(pc))
			}
		}
		lin.checks = append(lin.checks, h)
	}
	lin.a = make([]uint32, len(lin.checks))
	lin.b0 = make([]byte, len(lin.checks))
	for ri := range lin.checks {
		h := &lin.checks[ri]
		lin.b0[ri] = parityWords(h, &lin.pn0)
		for i := 0; i < tchSeedBits; i++ {
			if parityWords(h, &lin.pnCol[i]) == 1 {
				lin.a[ri] |= 1 << uint(i)
			}
		}
	}

	// Sparse checks: for each window of coded positions, the null space of the
	// generator restricted to those columns — vectors h on the window with
	// h·g_j = 0 for every information bit j. The original (un-reduced)
	// generator columns are rebuilt here since gen above is in RREF.
	orig := make([]row, tchInfoBits)
	for j := 0; j < tchInfoBits; j++ {
		info := make([]byte, tchInfoBits)
		info[j] = 1
		for k, b := range tchCodedBitsFor(info) {
			if b&1 != 0 {
				orig[j][k>>6] |= 1 << uint(k&63)
			}
		}
	}
	for start := 0; start+tchSparseWindow <= tchCodedBits; start += tchSparseStep {
		// m[j] = generator row j restricted to the window (W bits in a uint64).
		m := make([]uint64, tchInfoBits)
		for j := 0; j < tchInfoBits; j++ {
			var v uint64
			for w := 0; w < tchSparseWindow; w++ {
				k := start + w
				if orig[j][k>>6]&(1<<uint(k&63)) != 0 {
					v |= 1 << uint(w)
				}
			}
			m[j] = v
		}
		// RREF over the window's columns; free columns yield null vectors.
		pivots := make([]int, 0, tchSparseWindow)
		rr := 0
		for col := 0; col < tchSparseWindow && rr < tchInfoBits; col++ {
			sel := -1
			for i := rr; i < tchInfoBits; i++ {
				if m[i]&(1<<uint(col)) != 0 {
					sel = i
					break
				}
			}
			if sel < 0 {
				continue
			}
			m[rr], m[sel] = m[sel], m[rr]
			for i := 0; i < tchInfoBits; i++ {
				if i != rr && m[i]&(1<<uint(col)) != 0 {
					m[i] ^= m[rr]
				}
			}
			pivots = append(pivots, col)
			rr++
		}
		isPiv := make([]bool, tchSparseWindow)
		for _, c := range pivots {
			isPiv[c] = true
		}
		for f := 0; f < tchSparseWindow; f++ {
			if isPiv[f] {
				continue
			}
			var h [tchWords]uint64
			sc := sparseCheck{}
			add := func(k int) {
				i := tchType5IndexOfCoded(start + k)
				h[i>>6] |= 1 << uint(i&63)
				sc.support = append(sc.support, i)
			}
			add(f)
			for i, pc := range pivots {
				if m[i]&(1<<uint(f)) != 0 {
					add(pc)
				}
			}
			sc.b0 = parityWords(&h, &lin.pn0)
			for i := 0; i < tchSeedBits; i++ {
				if parityWords(&h, &lin.pnCol[i]) == 1 {
					sc.a |= 1 << uint(i)
				}
			}
			lin.sparse = append(lin.sparse, sc)
		}
	}
	return lin
}

// solveSeedRows solves the augmented rows (low 30 bits coefficients, bit 30 the
// right-hand side) for the seed. ok is false when the system is inconsistent
// or does not determine every seed bit.
func solveSeedRows(rows []uint32) (seed uint32, ok bool) {
	rank := 0
	for col := 0; col < tchSeedBits; col++ {
		sel := -1
		for i := rank; i < len(rows); i++ {
			if rows[i]&(1<<uint(col)) != 0 {
				sel = i
				break
			}
		}
		if sel < 0 {
			return 0, false
		}
		rows[rank], rows[sel] = rows[sel], rows[rank]
		for i := range rows {
			if i != rank && rows[i]&(1<<uint(col)) != 0 {
				rows[i] ^= rows[rank]
			}
		}
		rank++
	}
	for i := rank; i < len(rows); i++ {
		if rows[i] != 0 {
			return 0, false
		}
	}
	for i := 0; i < rank; i++ {
		if rows[i]>>tchSeedBits&1 != 0 {
			pivot := 0
			for pivot < tchSeedBits && rows[i]&(1<<uint(pivot)) == 0 {
				pivot++
			}
			seed |= 1 << uint(pivot)
		}
	}
	return seed, true
}

func tchSeedLinearInstance() *tchSeedLinear {
	tchSeedOnce.Do(func() { tchSeedLin = buildTCHSeedLinear() })
	return tchSeedLin
}

// SolveTCHScrambleSeed returns the 30-bit scrambling seed (the value
// framing.NewScramblerTetra takes — tetra.ExtendedColourCode packing) that a
// single still-scrambled 432-bit TCH/S type-5 block was scrambled with, by
// solving the block's parity checks for the seed. ok is false when the block is
// not an error-free TCH/S codeword under ANY seed (bit errors, a non-TCH/S
// burst, or too little rank) — the system is over-determined by 128 checks, so
// a false positive is negligible. Bits are bit-per-byte, de-rotated, in type-5
// (on-air) order.
func SolveTCHScrambleSeed(type5 []byte) (seed uint32, ok bool) {
	if len(type5) < tchType3Bits {
		return 0, false
	}
	lin := tchSeedLinearInstance()
	r := packBits432(type5[:tchType3Bits])
	// Augmented rows: low 30 bits = coefficients, bit 30 = right-hand side.
	rows := make([]uint32, len(lin.checks))
	for i := range lin.checks {
		y := parityWords(&lin.checks[i], &r) ^ lin.b0[i]
		rows[i] = lin.a[i] | uint32(y)<<tchSeedBits
	}
	return solveSeedRows(rows)
}

// DMBurstScrambleSeed solves the scramble seed of one DNB's TCH/S block (see
// SolveTCHScrambleSeed); ok is false for a DSB, a malformed DNB, or a DNB whose
// bit errors the soft-assisted search cannot place. When the burst carries the
// receiver's soft differentials (ExtractDMBurstsSoft) the hard bits are solved
// first and, failing that, the least reliable coded bits are flipped one and
// two at a time (SolveTCHScrambleSeedSoft) — a marginal burst with a couple of
// errors among its weakest symbols still solves, which is most of what a live
// tap delivers between the clean bursts.
func DMBurstScrambleSeed(b DMBurst) (seed uint32, ok bool) {
	type5 := dmDNBType5(b)
	if type5 == nil {
		return 0, false
	}
	if seed, ok = SolveTCHScrambleSeed(type5); ok {
		return seed, true
	}
	if len(b.SoftBKN1) != dmBlockDibits || len(b.SoftBKN2) != dmBlockDibits {
		return 0, false
	}
	diffs := make([]complex64, 0, 2*dmBlockDibits)
	diffs = append(diffs, b.SoftBKN1...)
	diffs = append(diffs, b.SoftBKN2...)
	return SolveTCHScrambleSeedSoft(type5, softType5FromDiffs(diffs, (4-b.Rotation)&3))
}

// tchSeedSoftSingles / tchSeedSoftPairs bound the soft-assisted search: single
// flips among the least reliable tchSeedSoftSingles coded bits, then pairs among
// the least reliable tchSeedSoftPairs. ~1 + 12 + 28 eliminations of a 158×31
// system — well under a millisecond per burst.
const (
	tchSeedSoftSingles = 12
	tchSeedSoftPairs   = 8
)

// SolveTCHScrambleSeedSoft is SolveTCHScrambleSeed for a block that did not
// solve as received: it re-tries with the least reliable coded bits (by |LLR|)
// flipped, singly then in pairs. llr is parallel to type5 (the soft convention
// of framing/soft_tetra.go; only magnitudes are used). Class-0 bits are uncoded
// and take no part in the checks, so they are never flipped.
func SolveTCHScrambleSeedSoft(type5 []byte, llr []float32) (seed uint32, ok bool) {
	if len(type5) < tchType3Bits || len(llr) < tchType3Bits {
		return 0, false
	}
	// Rank the coded positions by reliability (partial selection of the
	// weakest tchSeedSoftSingles is all that is needed).
	type cand struct {
		idx int
		mag float32
	}
	weak := make([]cand, 0, tchSeedSoftSingles)
	for k := 0; k < tchCodedBits; k++ {
		i := tchType5IndexOfCoded(k)
		m := llr[i]
		if m < 0 {
			m = -m
		}
		if len(weak) < tchSeedSoftSingles {
			weak = append(weak, cand{i, m})
			continue
		}
		// Replace the strongest of the kept set if this one is weaker.
		maxJ := 0
		for j := 1; j < len(weak); j++ {
			if weak[j].mag > weak[maxJ].mag {
				maxJ = j
			}
		}
		if m < weak[maxJ].mag {
			weak[maxJ] = cand{i, m}
		}
	}
	// Sort ascending by magnitude (tiny slice).
	for a := 1; a < len(weak); a++ {
		for b := a; b > 0 && weak[b].mag < weak[b-1].mag; b-- {
			weak[b], weak[b-1] = weak[b-1], weak[b]
		}
	}
	buf := append([]byte(nil), type5[:tchType3Bits]...)
	for a := 0; a < len(weak); a++ {
		buf[weak[a].idx] ^= 1
		if seed, ok = SolveTCHScrambleSeed(buf); ok {
			return seed, true
		}
		buf[weak[a].idx] ^= 1
	}
	n := len(weak)
	if n > tchSeedSoftPairs {
		n = tchSeedSoftPairs
	}
	for a := 0; a < n; a++ {
		buf[weak[a].idx] ^= 1
		for c := a + 1; c < n; c++ {
			buf[weak[c].idx] ^= 1
			if seed, ok = SolveTCHScrambleSeed(buf); ok {
				return seed, true
			}
			buf[weak[c].idx] ^= 1
		}
		buf[weak[a].idx] ^= 1
	}
	return solveTCHSeedReliableChecks(type5, llr)
}

// tchSparseMinChecks is how many of the most reliable sparse checks must agree
// on a seed for the reliable-check solve to accept it: 30 to determine the seed
// plus 24 redundant ones, so a burst that is not a TCH/S codeword (noise, a
// signalling DNB) passes by chance ~2^-24 per attempt.
const tchSparseMinChecks = tchSeedBits + 24

// solveTCHSeedReliableChecks solves the seed from the burst's most reliable
// LOCAL parity checks: each sparse check spans ~48 consecutive coded bits, so a
// bit error spoils only the checks whose window contains it, and ranking checks
// by their weakest symbol's |LLR| keeps the erroneous ones out of the system.
// Tries the most reliable tchSparseMinChecks checks first, widening in steps
// while the system stays consistent — the widest consistent set wins.
func solveTCHSeedReliableChecks(type5 []byte, llr []float32) (seed uint32, ok bool) {
	lin := tchSeedLinearInstance()
	r := packBits432(type5[:tchType3Bits])
	type scored struct {
		row uint32
		rel float32
	}
	all := make([]scored, 0, len(lin.sparse))
	for i := range lin.sparse {
		sc := &lin.sparse[i]
		rel := float32(0)
		var parity byte
		for k, pos := range sc.support {
			m := llr[pos]
			if m < 0 {
				m = -m
			}
			if k == 0 || m < rel {
				rel = m
			}
			parity ^= byte(r[pos>>6]>>uint(pos&63)) & 1
		}
		all = append(all, scored{row: sc.a | uint32(parity^sc.b0)<<tchSeedBits, rel: rel})
	}
	// Sort by reliability, descending (n log n on ~300 entries).
	for i := 1; i < len(all); i++ {
		for j := i; j > 0 && all[j].rel > all[j-1].rel; j-- {
			all[j], all[j-1] = all[j-1], all[j]
		}
	}
	if len(all) < tchSparseMinChecks {
		return 0, false
	}
	rows := make([]uint32, 0, len(all))
	best, found := uint32(0), false
	for n := tchSparseMinChecks; n <= len(all); n += 8 {
		rows = rows[:0]
		for i := 0; i < n; i++ {
			rows = append(rows, all[i].row)
		}
		s, ok := solveSeedRows(rows)
		if !ok {
			break
		}
		best, found = s, true
	}
	return best, found
}

// dmSeedMinVotes is how many error-free DNBs must solve to the same seed before
// RecoverDMScrambleSeed trusts it. Two agreeing exact solutions already have a
// ~2^-30 chance of coinciding by accident; the CRC confirmation on top makes the
// result at least as strong as the brute-force colour search's dominance gate.
const dmSeedMinVotes = 2

// RecoverDMScrambleSeed recovers the full 30-bit extended colour code a DMO
// transmission's TCH/S traffic is scrambled with, by solving each DNB exactly
// (DMBurstScrambleSeed) and voting. It needs no configured MNI and makes no
// assumption about how the transmitter composed the seed — on air (the 12 Sep
// #1003 capture) the DM colour code turned out to change from one PTT to the
// next, so no per-network constant could ever have described it.
//
// Returns the winning seed, how many error-free bursts solved to it, and
// whether it is trustworthy: at least dmSeedMinVotes agreeing solutions AND the
// seed CRC-decodes at least as many bursts as it was solved from (an exact
// solution of a real TCH/S block always CRC-decodes with its own seed, so this
// is the cross-check that the linear model matches the decoder).
func RecoverDMScrambleSeed(bursts []DMBurst) (seed uint32, votes int, confident bool) {
	counts := map[uint32]int{}
	for i := range bursts {
		if bursts[i].Kind != DMBurstNormal {
			continue
		}
		if s, ok := DMBurstScrambleSeed(bursts[i]); ok {
			counts[s]++
		}
	}
	best, bestN := uint32(0), 0
	for s, n := range counts {
		if n > bestN || (n == bestN && s < best) {
			best, bestN = s, n
		}
	}
	if bestN < dmSeedMinVotes {
		return best, bestN, false
	}
	crc := 0
	for i := range bursts {
		if bursts[i].Kind != DMBurstNormal {
			continue
		}
		if len(DMBurstTCHSpeechSoft(bursts[i], best)) == 2 || len(DMBurstTCHSpeech(bursts[i], best)) == 2 {
			crc++
		}
	}
	return best, bestN, crc >= bestN
}
