// Package randomness implements a battery of statistical randomness tests
// drawn from NIST SP 800-22 (the "STS" suite), specialised for the question
// an RF analyst actually asks of a captured payload or a recovered keystream:
// is this stream indistinguishable from random (strong keyed encryption — no
// weak structure to exploit), or does it carry exploitable structure (a
// keyless scrambler, a short LFSR, a biased generator)?
//
// Each test returns a p-value in [0,1]; a stream "passes" a test at
// significance level alpha when p >= alpha. A truly random stream passes all
// tests; a structured one fails one or more (most diagnostically the spectral
// and linear-complexity tests for RF scramblers). The toolkit's existing
// entropy / index-of-coincidence triage (engine/stats) answers the same
// question coarsely; this battery answers it rigorously.
//
// Bit sequences are represented as a []byte holding one 0/1 value per element
// (the same convention as engine/lfsr), so callers convert packed capture
// bytes with lfsr.BitsFromBytes. The incomplete-gamma p-values use gonum's
// mathext.GammaIncRegComp and the spectral test uses gonum's real FFT, so the
// suite stays pure-Go with no new dependencies.
package randomness

import (
	"math"
	"math/cmplx"
	"sort"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/lfsr"
	"gonum.org/v1/gonum/dsp/fourier"
	"gonum.org/v1/gonum/mathext"
)

// DefaultAlpha is the conventional NIST significance level.
const DefaultAlpha = 0.01

// TestResult is the outcome of one randomness test.
type TestResult struct {
	Name string `json:"name"`
	// PValue is the test statistic's p-value in [0,1]; only meaningful when
	// Applicable is true.
	PValue float64 `json:"p_value"`
	// Passed reports p >= alpha. Only meaningful when Applicable is true.
	Passed bool `json:"passed"`
	// Applicable is false when the stream is too short for the test's
	// preconditions; Note explains what is missing.
	Applicable bool   `json:"applicable"`
	Note       string `json:"note,omitempty"`
}

// Report is the full battery outcome over one bit stream.
type Report struct {
	Bits    int          `json:"bits"`
	Alpha   float64      `json:"alpha"`
	Tests   []TestResult `json:"tests"`
	Passed  int          `json:"passed"`
	Failed  int          `json:"failed"`
	Skipped int          `json:"skipped"`
}

// igamc is the regularised upper incomplete gamma function Q(a,x), the
// p-value form NIST uses for its chi-square-distributed statistics.
func igamc(a, x float64) float64 {
	if x < 0 || a <= 0 {
		return 0
	}
	return mathext.GammaIncRegComp(a, x)
}

// normCDF is the standard normal cumulative distribution function.
func normCDF(x float64) float64 { return 0.5 * math.Erfc(-x/math.Sqrt2) }

// minus1pow returns (-1)^k without floating-point pow.
func minus1pow(k int) float64 {
	if k%2 == 0 {
		return 1
	}
	return -1
}

func na(name, note string) TestResult {
	return TestResult{Name: name, Applicable: false, Note: note}
}

// Monobit (frequency) test: are roughly half the bits one? The most basic
// test; everything else assumes this passes.
func Monobit(bits []byte) TestResult {
	n := len(bits)
	if n < 1 {
		return na("monobit", "no bits")
	}
	s := 0
	for _, b := range bits {
		if b&1 == 1 {
			s++
		} else {
			s--
		}
	}
	sObs := math.Abs(float64(s)) / math.Sqrt(float64(n))
	p := math.Erfc(sObs / math.Sqrt2)
	return TestResult{Name: "monobit", PValue: p, Passed: p >= DefaultAlpha, Applicable: true}
}

// BlockFrequency: is the proportion of ones uniform across M-bit blocks?
func BlockFrequency(bits []byte, m int) TestResult {
	n := len(bits)
	if m < 1 {
		m = 1
	}
	N := n / m
	if N < 1 {
		return na("block_frequency", "need at least one full block")
	}
	var sum float64
	for i := 0; i < N; i++ {
		ones := 0
		for j := 0; j < m; j++ {
			ones += int(bits[i*m+j] & 1)
		}
		pi := float64(ones)/float64(m) - 0.5
		sum += pi * pi
	}
	chi := 4 * float64(m) * sum
	p := igamc(float64(N)/2, chi/2)
	return TestResult{Name: "block_frequency", PValue: p, Passed: p >= DefaultAlpha, Applicable: true}
}

// Runs: is the number of uninterrupted runs of identical bits what a random
// stream would produce? Detects oscillation that is too fast or too slow.
func Runs(bits []byte) TestResult {
	n := len(bits)
	if n < 2 {
		return na("runs", "need at least 2 bits")
	}
	ones := 0
	for _, b := range bits {
		ones += int(b & 1)
	}
	pi := float64(ones) / float64(n)
	if math.Abs(pi-0.5) >= 2/math.Sqrt(float64(n)) {
		return TestResult{Name: "runs", PValue: 0, Passed: false, Applicable: true,
			Note: "fails the monobit precondition; runs p-value forced to 0"}
	}
	v := 1
	for i := 1; i < n; i++ {
		if bits[i]&1 != bits[i-1]&1 {
			v++
		}
	}
	num := math.Abs(float64(v) - 2*float64(n)*pi*(1-pi))
	den := 2 * math.Sqrt(2*float64(n)) * pi * (1 - pi)
	p := math.Erfc(num / den)
	return TestResult{Name: "runs", PValue: p, Passed: p >= DefaultAlpha, Applicable: true}
}

// LongestRun: is the longest run of ones within a block consistent with
// randomness? Uses the NIST block size / category table selected by length.
func LongestRun(bits []byte) TestResult {
	n := len(bits)
	var m int
	var pi []float64
	var bucket func(int) int
	switch {
	case n < 128:
		return na("longest_run", "need >= 128 bits")
	case n < 6272:
		m = 8
		pi = []float64{0.2148, 0.3672, 0.2305, 0.1875}
		bucket = func(v int) int {
			switch {
			case v <= 1:
				return 0
			case v == 2:
				return 1
			case v == 3:
				return 2
			default:
				return 3
			}
		}
	case n < 750000:
		m = 128
		pi = []float64{0.1174, 0.2430, 0.2493, 0.1752, 0.1027, 0.1124}
		bucket = func(v int) int {
			switch {
			case v <= 4:
				return 0
			case v >= 9:
				return 5
			default:
				return v - 4
			}
		}
	default:
		m = 10000
		pi = []float64{0.0882, 0.2092, 0.2483, 0.1933, 0.1208, 0.0675, 0.0727}
		bucket = func(v int) int {
			switch {
			case v <= 10:
				return 0
			case v >= 16:
				return 6
			default:
				return v - 10
			}
		}
	}
	N := n / m
	v := make([]int, len(pi))
	for i := 0; i < N; i++ {
		longest, cur := 0, 0
		for j := 0; j < m; j++ {
			if bits[i*m+j]&1 == 1 {
				cur++
				if cur > longest {
					longest = cur
				}
			} else {
				cur = 0
			}
		}
		v[bucket(longest)]++
	}
	var chi float64
	for i := range pi {
		e := float64(N) * pi[i]
		d := float64(v[i]) - e
		chi += d * d / e
	}
	p := igamc(float64(len(pi)-1)/2, chi/2)
	return TestResult{Name: "longest_run", PValue: p, Passed: p >= DefaultAlpha, Applicable: true}
}

// gf2Rank returns the GF(2) rank of an r×c bit matrix whose rows are packed
// MSB-first into uint64 (c <= 64).
func gf2Rank(rows []uint64, r, c int) int {
	rank := 0
	for col := 0; col < c && rank < r; col++ {
		mask := uint64(1) << uint(c-1-col)
		pivot := -1
		for i := rank; i < r; i++ {
			if rows[i]&mask != 0 {
				pivot = i
				break
			}
		}
		if pivot < 0 {
			continue
		}
		rows[rank], rows[pivot] = rows[pivot], rows[rank]
		for i := 0; i < r; i++ {
			if i != rank && rows[i]&mask != 0 {
				rows[i] ^= rows[rank]
			}
		}
		rank++
	}
	return rank
}

// BinaryMatrixRank: do the ranks of disjoint 32×32 sub-matrices follow the
// expected distribution? Detects linear dependence among bit positions.
func BinaryMatrixRank(bits []byte) TestResult {
	const M, Q = 32, 32
	n := len(bits)
	N := n / (M * Q)
	if N < 1 {
		return na("binary_matrix_rank", "need >= 1024 bits")
	}
	var fm, fm1, rest int
	for blk := 0; blk < N; blk++ {
		rows := make([]uint64, M)
		base := blk * M * Q
		for r := 0; r < M; r++ {
			var row uint64
			for c := 0; c < Q; c++ {
				if bits[base+r*Q+c]&1 == 1 {
					row |= uint64(1) << uint(Q-1-c)
				}
			}
			rows[r] = row
		}
		switch gf2Rank(rows, M, Q) {
		case M:
			fm++
		case M - 1:
			fm1++
		default:
			rest++
		}
	}
	nf := float64(N)
	chi := sq(float64(fm)-0.2888*nf)/(0.2888*nf) +
		sq(float64(fm1)-0.5776*nf)/(0.5776*nf) +
		sq(float64(rest)-0.1336*nf)/(0.1336*nf)
	p := math.Exp(-chi / 2)
	res := TestResult{Name: "binary_matrix_rank", PValue: p, Passed: p >= DefaultAlpha, Applicable: true}
	if N < 38 {
		res.Note = "fewer than 38 matrices; p-value is approximate"
	}
	return res
}

// Spectral (DFT): does the discrete Fourier transform of the ±1 sequence
// show periodic features (spikes) inconsistent with randomness? This is the
// sharpest test for periodic RF scramblers and rolling codes.
func Spectral(bits []byte) TestResult {
	n := len(bits)
	if n < 128 {
		return na("spectral_dft", "need >= 128 bits")
	}
	seq := make([]float64, n)
	for i, b := range bits {
		if b&1 == 1 {
			seq[i] = 1
		} else {
			seq[i] = -1
		}
	}
	coeff := fourier.NewFFT(n).Coefficients(nil, seq)
	half := n / 2
	t := math.Sqrt(math.Log(1/0.05) * float64(n))
	n0 := 0.95 * float64(n) / 2
	n1 := 0
	for i := 0; i < half; i++ {
		if cmplx.Abs(coeff[i]) < t {
			n1++
		}
	}
	d := (float64(n1) - n0) / math.Sqrt(float64(n)*0.95*0.05/4.0)
	p := math.Erfc(math.Abs(d) / math.Sqrt2)
	return TestResult{Name: "spectral_dft", PValue: p, Passed: p >= DefaultAlpha, Applicable: true}
}

// patternCounts counts occurrences of each m-bit pattern in the cyclically
// extended sequence (NIST convention for ApEn and Serial).
func patternCounts(bits []byte, m int) []int {
	n := len(bits)
	counts := make([]int, 1<<uint(m))
	for i := 0; i < n; i++ {
		idx := 0
		for j := 0; j < m; j++ {
			idx = (idx << 1) | int(bits[(i+j)%n]&1)
		}
		counts[idx]++
	}
	return counts
}

// ApproximateEntropy: compares the frequency of overlapping m- and
// (m+1)-bit patterns; structure lowers the apparent entropy.
func ApproximateEntropy(bits []byte, m int) TestResult {
	n := len(bits)
	if m < 1 || n < (1<<uint(m+5)) {
		return na("approximate_entropy", "stream too short for the chosen block length")
	}
	phi := func(k int) float64 {
		counts := patternCounts(bits, k)
		var s float64
		for _, c := range counts {
			if c > 0 {
				ci := float64(c) / float64(n)
				s += ci * math.Log(ci)
			}
		}
		return s
	}
	apen := phi(m) - phi(m+1)
	chi := 2 * float64(n) * (math.Ln2 - apen)
	p := igamc(math.Exp2(float64(m-1)), chi/2)
	return TestResult{Name: "approximate_entropy", PValue: p, Passed: p >= DefaultAlpha, Applicable: true}
}

// psi2 is the NIST serial-test statistic for block length m.
func psi2(bits []byte, m int) float64 {
	if m <= 0 {
		return 0
	}
	n := len(bits)
	counts := patternCounts(bits, m)
	var s float64
	for _, c := range counts {
		s += float64(c) * float64(c)
	}
	return s*math.Exp2(float64(m))/float64(n) - float64(n)
}

// Serial: are all overlapping m-bit patterns equally likely? Returns a single
// result that passes only when both NIST sub-p-values pass.
func Serial(bits []byte, m int) TestResult {
	n := len(bits)
	if m < 3 || n < (1<<uint(m+5)) {
		return na("serial", "stream too short for the chosen block length")
	}
	p0 := psi2(bits, m)
	p1 := psi2(bits, m-1)
	p2 := psi2(bits, m-2)
	del1 := p0 - p1
	del2 := p0 - 2*p1 + p2
	pv1 := igamc(math.Exp2(float64(m-2)), del1/2)
	pv2 := igamc(math.Exp2(float64(m-3)), del2/2)
	pv := math.Min(pv1, pv2)
	return TestResult{Name: "serial", PValue: pv, Passed: pv1 >= DefaultAlpha && pv2 >= DefaultAlpha, Applicable: true}
}

// CumulativeSums (forward): treats the ±1 sequence as a random walk and tests
// whether the maximum excursion from zero is too large.
func CumulativeSums(bits []byte) TestResult {
	n := len(bits)
	if n < 2 {
		return na("cumulative_sums", "need at least 2 bits")
	}
	z, s := 0, 0
	for _, b := range bits {
		if b&1 == 1 {
			s++
		} else {
			s--
		}
		if a := abs(s); a > z {
			z = a
		}
	}
	if z == 0 {
		return TestResult{Name: "cumulative_sums", PValue: 1, Passed: true, Applicable: true}
	}
	zf, nf := float64(z), float64(n)
	sqn := math.Sqrt(nf)
	var sum1, sum2 float64
	for k := int(math.Floor((-nf/zf + 1) / 4)); k <= int(math.Floor((nf/zf-1)/4)); k++ {
		kf := float64(k)
		sum1 += normCDF((4*kf+1)*zf/sqn) - normCDF((4*kf-1)*zf/sqn)
	}
	for k := int(math.Floor((-nf/zf - 3) / 4)); k <= int(math.Floor((nf/zf-1)/4)); k++ {
		kf := float64(k)
		sum2 += normCDF((4*kf+3)*zf/sqn) - normCDF((4*kf+1)*zf/sqn)
	}
	p := 1 - sum1 + sum2
	p = clamp01(p)
	return TestResult{Name: "cumulative_sums", PValue: p, Passed: p >= DefaultAlpha, Applicable: true}
}

// LinearComplexity: blocks the stream and tests the distribution of each
// block's shortest-LFSR length (via Berlekamp–Massey). Low linear complexity
// is the defining weakness of an LFSR-based scrambler, so this is the most
// RF-diagnostic test in the suite.
func LinearComplexity(bits []byte, m int) TestResult {
	if m <= 0 {
		m = 500
	}
	n := len(bits)
	N := n / m
	if N < 1 {
		return na("linear_complexity", "need at least one full block (default 500 bits)")
	}
	pi := []float64{0.010417, 0.03125, 0.125, 0.5, 0.25, 0.0625, 0.020833}
	mu := float64(m)/2 + (9+minus1pow(m+1))/36 - (float64(m)/3+2.0/9.0)/math.Exp2(float64(m))
	v := make([]int, 7)
	for i := 0; i < N; i++ {
		block := bits[i*m : (i+1)*m]
		l := lfsr.BerlekampMassey(block).LinearComplexity
		ti := minus1pow(m)*(float64(l)-mu) + 2.0/9.0
		switch {
		case ti <= -2.5:
			v[0]++
		case ti <= -1.5:
			v[1]++
		case ti <= -0.5:
			v[2]++
		case ti <= 0.5:
			v[3]++
		case ti <= 1.5:
			v[4]++
		case ti <= 2.5:
			v[5]++
		default:
			v[6]++
		}
	}
	var chi float64
	for i := range pi {
		e := float64(N) * pi[i]
		d := float64(v[i]) - e
		chi += d * d / e
	}
	p := igamc(3, chi/2) // K=6 categories -> K/2 = 3
	res := TestResult{Name: "linear_complexity", PValue: p, Passed: p >= DefaultAlpha, Applicable: true}
	if N < 200 {
		res.Note = "fewer than 200 blocks; p-value is approximate"
	}
	return res
}

// Battery runs the full suite over the bit stream, choosing block sizes from
// the stream length. alpha <= 0 selects DefaultAlpha.
func Battery(bits []byte, alpha float64) Report {
	if alpha <= 0 {
		alpha = DefaultAlpha
	}
	n := len(bits)
	blockM := n / 100
	if blockM < 20 {
		blockM = 20
	}
	if blockM > 10000 {
		blockM = 10000
	}
	tests := []TestResult{
		Monobit(bits),
		BlockFrequency(bits, blockM),
		Runs(bits),
		LongestRun(bits),
		BinaryMatrixRank(bits),
		Spectral(bits),
		ApproximateEntropy(bits, 2),
		Serial(bits, 3),
		CumulativeSums(bits),
		LinearComplexity(bits, 0),
	}
	return finalize(bits, alpha, tests)
}

// Quick runs the fast, low-data subset suitable for short captures.
func Quick(bits []byte, alpha float64) Report {
	if alpha <= 0 {
		alpha = DefaultAlpha
	}
	n := len(bits)
	blockM := n / 50
	if blockM < 20 {
		blockM = 20
	}
	tests := []TestResult{
		Monobit(bits),
		BlockFrequency(bits, blockM),
		Runs(bits),
		CumulativeSums(bits),
	}
	return finalize(bits, alpha, tests)
}

func finalize(bits []byte, alpha float64, tests []TestResult) Report {
	rep := Report{Bits: len(bits), Alpha: alpha, Tests: tests}
	for i := range rep.Tests {
		t := &rep.Tests[i]
		if !t.Applicable {
			rep.Skipped++
			continue
		}
		t.Passed = t.PValue >= alpha
		if t.Passed {
			rep.Passed++
		} else {
			rep.Failed++
		}
	}
	return rep
}

// WeakestTests returns the applicable tests sorted by ascending p-value (the
// strongest evidence of non-randomness first).
func (r Report) WeakestTests() []TestResult {
	out := make([]TestResult, 0, len(r.Tests))
	for _, t := range r.Tests {
		if t.Applicable {
			out = append(out, t)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].PValue < out[j].PValue })
	return out
}

// LooksRandom reports whether every applicable test passed — i.e. the stream
// is statistically indistinguishable from random at the report's alpha.
func (r Report) LooksRandom() bool { return r.Failed == 0 && r.Passed > 0 }

func sq(x float64) float64 { return x * x }
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}
