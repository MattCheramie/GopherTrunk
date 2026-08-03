package equalizer

import (
	"math"
	"math/cmplx"
	"math/rand"
	"testing"
)

// These tests exercise SnapshotLMS on a π/4-DQPSK differential stream through a
// synthetic multipath channel — the #1001 case: a same-carrier signal whose
// inter-symbol interference closes the constellation eye and garbles the
// differential decode. A training-sequence-aided equalizer trained on a known
// midamble should invert the channel where the raw stream (and, on a burst-length
// sequence, the blind SnapshotCMA) cannot. Metric is differential dibit-error
// rate — the constant-phase-invariant, decode-level number that CLAUDE.md flags
// as the only trustworthy one (EVM is a trap).

// piBy4Inc maps a dibit to its π/4-DQPSK phase increment (matches the receiver /
// soft_test.go dibitIdealDiff convention).
func piBy4Inc(d uint8) float64 {
	switch d & 3 {
	case 0:
		return math.Pi / 4
	case 1:
		return 3 * math.Pi / 4
	case 2:
		return -3 * math.Pi / 4
	default:
		return -math.Pi / 4
	}
}

// piBy4Encode differential-encodes dibits into unit-modulus symbols, starting
// from phase 0. Returns len(dibits)+1 symbols (the extra leading reference).
func piBy4Encode(dibits []uint8) []complex64 {
	sym := make([]complex64, len(dibits)+1)
	phase := 0.0
	sym[0] = 1
	for i, d := range dibits {
		phase += piBy4Inc(d)
		sym[i+1] = complex64(cmplx.Rect(1, phase))
	}
	return sym
}

// piBy4DecodeDibits differential-decodes symbols back to dibits: the dibit whose
// ideal phase increment is closest to arg(s[i]·conj(s[i-1])).
func piBy4DecodeDibits(sym []complex64) []uint8 {
	out := make([]uint8, 0, len(sym)-1)
	for i := 1; i < len(sym); i++ {
		diff := complex128(sym[i]) * cmplx.Conj(complex128(sym[i-1]))
		ang := cmplx.Phase(diff)
		best := uint8(0)
		bestErr := math.Pi * 2
		for d := uint8(0); d < 4; d++ {
			e := math.Abs(math.Remainder(ang-piBy4Inc(d), 2*math.Pi))
			if e < bestErr {
				bestErr = e
				best = d
			}
		}
		out = append(out, best)
	}
	return out
}

// multipath convolves sym with a causal channel h (h[0] is the direct path),
// adding complex Gaussian noise of the given sigma per component. A minimum-phase
// h (dominant first tap) keeps a near-zero-delay causal inverse, so an LMS trained
// with the reference aligned 1:1 (no delay) converges cleanly.
func multipath(sym []complex64, h []complex64, sigma float64, rng *rand.Rand) []complex64 {
	out := make([]complex64, len(sym))
	for n := range sym {
		var acc complex64
		for k := range h {
			if n-k >= 0 {
				acc += h[k] * sym[n-k]
			}
		}
		acc += complex(float32(rng.NormFloat64()*sigma), float32(rng.NormFloat64()*sigma))
		out[n] = acc
	}
	return out
}

func dibitErrRate(got, want []uint8) float64 {
	n := len(got)
	if len(want) < n {
		n = len(want)
	}
	if n == 0 {
		return 0
	}
	bad := 0
	for i := 0; i < n; i++ {
		if got[i] != want[i] {
			bad++
		}
	}
	return float64(bad) / float64(n)
}

// hardMultipath is a minimum-phase channel whose echoes close the π/4-DQPSK eye.
var hardMultipath = []complex64{
	1,
	complex64(cmplx.Rect(0.6, 0.8)),
	complex64(cmplx.Rect(0.35, -1.2)),
}

// TestSnapshotLMSInvertsMultipath is the failing-first behavioural regression: a
// multipath channel garbles the raw differential decode, and SnapshotLMS trained
// on a known midamble drives the PAYLOAD (post-training) dibit-error rate to near
// zero — proving it inverts the channel, not just memorises the reference.
func TestSnapshotLMSInvertsMultipath(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	const (
		nDibits  = 600
		trainLen = 160 // midamble length in dibits
		sigma    = 0.02
	)
	dibits := make([]uint8, nDibits)
	for i := range dibits {
		dibits[i] = uint8(rng.Intn(4))
	}
	tx := piBy4Encode(dibits)
	rx := multipath(tx, hardMultipath, sigma, rng)

	// Raw (unequalized) differential decode over the payload region.
	rawErr := dibitErrRate(piBy4DecodeDibits(rx)[trainLen:], dibits[trainLen:])

	// Train on the aligned midamble (reference symbols the receiver knows), then
	// equalize the whole burst and decode the payload.
	eq := NewSnapshotLMS(11, 0.02)
	eq.Train(rx[:trainLen+1], tx[:trainLen+1], 40)
	out := eq.Equalize(nil, rx)
	eqErr := dibitErrRate(piBy4DecodeDibits(out)[trainLen:], dibits[trainLen:])

	t.Logf("payload dibit-error: raw=%.3f equalized=%.3f", rawErr, eqErr)
	if rawErr < 0.10 {
		t.Fatalf("channel not hard enough: raw payload error %.3f (want the ISI to actually garble it)", rawErr)
	}
	if eqErr > 0.03 {
		t.Fatalf("SnapshotLMS did not invert the channel: payload error %.3f (raw %.3f)", eqErr, rawErr)
	}
	if eqErr >= rawErr {
		t.Fatalf("SnapshotLMS did not improve the decode: equalized %.3f vs raw %.3f", eqErr, rawErr)
	}
}

// TestSnapshotLMSNoHarmOnCleanChannel: on an identity channel the frozen taps
// stay a near pass-through, so the equalized differential decode is exactly the
// raw one — no regression on already-clean bursts (the SnapshotCMA no-harm rule).
func TestSnapshotLMSNoHarmOnCleanChannel(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	dibits := make([]uint8, 300)
	for i := range dibits {
		dibits[i] = uint8(rng.Intn(4))
	}
	tx := piBy4Encode(dibits)
	rx := multipath(tx, []complex64{1}, 0, rng) // identity channel, no noise

	eq := NewSnapshotLMS(11, 0.02)
	eq.Train(rx[:121], tx[:121], 40)
	out := eq.Equalize(nil, rx)

	if err := dibitErrRate(piBy4DecodeDibits(out), dibits); err > 0.001 {
		t.Fatalf("SnapshotLMS harmed a clean channel: dibit error %.3f", err)
	}
}

// TestSnapshotLMSBeatsBlindCMAOnBurst substantiates the value claim over the
// already-shipped blind equalizer: on a burst-length multipath sequence the
// training-sequence LMS resolves the channel while blind SnapshotCMA — starved of
// symbols and prone to constant-modulus spurious minima — does materially worse.
func TestSnapshotLMSBeatsBlindCMAOnBurst(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	const (
		nDibits  = 500
		trainLen = 160
		sigma    = 0.02
	)
	dibits := make([]uint8, nDibits)
	for i := range dibits {
		dibits[i] = uint8(rng.Intn(4))
	}
	tx := piBy4Encode(dibits)
	rx := multipath(tx, hardMultipath, sigma, rng)

	lms := NewSnapshotLMS(11, 0.02)
	lms.Train(rx[:trainLen+1], tx[:trainLen+1], 40)
	lmsOut := lms.Equalize(nil, rx)
	lmsErr := dibitErrRate(piBy4DecodeDibits(lmsOut)[trainLen:], dibits[trainLen:])

	// Blind CMA gets the same samples but no reference; snapshot every 64 symbols.
	cma := NewSnapshotCMA(11, 0.01, 64)
	cmaOut := make([]complex64, len(rx))
	for i, x := range rx {
		cmaOut[i] = cma.Process(x)
	}
	cmaErr := dibitErrRate(piBy4DecodeDibits(cmaOut)[trainLen:], dibits[trainLen:])

	t.Logf("payload dibit-error: trained-LMS=%.3f blind-CMA=%.3f", lmsErr, cmaErr)
	if lmsErr >= cmaErr {
		t.Fatalf("training-sequence LMS did not beat blind CMA: lms=%.3f cma=%.3f", lmsErr, cmaErr)
	}
	if lmsErr > 0.03 {
		t.Fatalf("training-sequence LMS payload error too high: %.3f", lmsErr)
	}
}
