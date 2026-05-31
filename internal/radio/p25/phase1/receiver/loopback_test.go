package receiver

import (
	"math"
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
)

// These loopback tests answer issue #402's open question — "is the outer-rail
// asymmetry the OP measured (true +3 ≈ +0.19 vs −3 ≈ −0.23) introduced by our
// receiver chain, or is it in the received signal?" — with evidence rather
// than an assertion. We run KNOWN symbols through our own receiver and measure
// the true outer-rail eye; ApplyImpairments injects the multipath the OP
// reports for the (off-axis) Murradoc Hill site.
//
// The loopback runs at sps=40 (192 kHz), not the production sps=10: chain
// symmetry is a *structural* property of the discriminator / matched filter /
// AGC and is independent of sps, while ModulateP25C4FM renders a degenerate,
// non-decodable *closed* eye at sps=10 (a synthesis artifact — real hardware
// is open at sps=10). So sps=40 is the faithful way to ask "does our chain
// introduce a +/- asymmetry?". (The sps=10 eye closure the OP sees in the
// capture is symmetric ISI — it compresses both outer rails equally — which
// these tests confirm is not turned into an asymmetry by our chain.)

const (
	loopSR  = 192_000.0 // sps=40 — the lowest rate ModulateP25C4FM keeps the eye open
	loopDev = 1800.0
	// Post-AGC outer-symbol level the slicer is calibrated to (2π·dev/sr).
	loopSlicerScale = 2 * math.Pi * loopDev / loopSR
)

// loopbackDibitSymbol maps a P25 dibit (0..3) to its C4FM symbol: 0→+1, 1→+3,
// 2→−1, 3→−3.
var loopbackDibitSymbol = [4]int8{1, 3, -1, -3}

// runLoopback modulates dibits at sps=40, optionally degrades the IQ, runs it
// through a fresh receiver, and returns the post-AGC soft stream.
func runLoopback(dibits []uint8, imp *demod.Impairments) []float32 {
	iq := demod.ModulateP25C4FM(dibits, loopSR, loopDev)
	if imp != nil {
		iq = demod.ApplyImpairments(iq, loopSR, *imp)
	}
	var soft []float32
	r := New(Options{
		SampleRateHz: loopSR,
		DeviationHz:  loopDev,
		SoftSink:     func(s []float32) { soft = append(soft, s...) },
		DibitSink:    func([]uint8, int) {},
	})
	r.Process(iq)
	return soft
}

// alignSoftToDibits slices the soft stream, finds the integer lag at which the
// sliced symbols best match the known transmitted dibits (the receiver drops
// leading symbols during clock/AGC acquisition, so the soft stream lags the
// input), and returns each post-warmup soft sample paired with its TRUE
// transmitted symbol, plus the best match fraction (≈1 on a cleanly-decoding
// stream — proves the loopback aligned).
func alignSoftToDibits(soft []float32, dibits []uint8, warmup int) (vals []float32, truth []int8, matchFrac float64, rot int) {
	fixed := demod.NewC4FMWithTaps([]float32{1}, loopSlicerScale)
	sliced := fixed.SliceMany(nil, soft)
	slicedDibits := make([]uint8, len(sliced))
	for i, s := range sliced {
		slicedDibits[i] = phase1.SymbolToDibit(s)
	}
	// Search over both the symbol lag and the dibit rotation: the C4FM
	// discriminator may emit the inverted polarity (rotation 2), which the
	// real control-channel path also handles via rotations [0 2].
	// Search lag and rotation. The lag can be negative (the soft stream leads
	// the input index by the net group delay), so scan both directions; the
	// C4FM discriminator may also emit the inverted polarity (rotation 2),
	// which the real control-channel path handles via rotations [0 2].
	bestOff, bestRot, bestMatches, bestCnt := 0, 0, -1, 1
	const minOff, maxOff = -200, 200
	for rr := 0; rr < 4; rr++ {
		for off := minOff; off <= maxOff; off++ {
			m, cnt := 0, 0
			for i := warmup; i < len(slicedDibits); i++ {
				j := i + off
				if j < 0 || j >= len(dibits) {
					continue
				}
				if (slicedDibits[i]+uint8(rr))&3 == dibits[j] {
					m++
				}
				cnt++
			}
			if cnt > 100 && m > bestMatches {
				bestMatches, bestOff, bestRot, bestCnt = m, off, rr, cnt
			}
		}
	}
	for i := warmup; i < len(soft); i++ {
		j := i + bestOff
		if j < 0 || j >= len(dibits) {
			continue
		}
		sym := loopbackDibitSymbol[dibits[j]&3]
		if bestRot == 2 { // global polarity inversion: the soft eye is inverted
			sym = -sym
		}
		vals = append(vals, soft[i])
		truth = append(truth, sym)
	}
	if bestCnt > 0 {
		matchFrac = float64(bestMatches) / float64(bestCnt)
	}
	return vals, truth, matchFrac, bestRot
}

func meanStd(v []float64) (mean, std float64) {
	if len(v) == 0 {
		return 0, 0
	}
	for _, x := range v {
		mean += x
	}
	mean /= float64(len(v))
	for _, x := range v {
		d := x - mean
		std += d * d
	}
	return mean, math.Sqrt(std / float64(len(v)))
}

// outerRailStats returns the mean/std of the soft values whose TRUE symbol is
// +3 and −3.
func outerRailStats(vals []float32, truth []int8) (meanP, stdP, meanN, stdN float64) {
	var p, n []float64
	for i, v := range vals {
		switch truth[i] {
		case 3:
			p = append(p, float64(v))
		case -3:
			n = append(n, float64(v))
		}
	}
	meanP, stdP = meanStd(p)
	meanN, stdN = meanStd(n)
	return
}

func symbolErrorRate(vals []float32, truth []int8) float64 {
	if len(vals) == 0 {
		return 1
	}
	fixed := demod.NewC4FMWithTaps([]float32{1}, loopSlicerScale)
	sliced := fixed.SliceMany(nil, vals)
	errs := 0
	for i, s := range sliced {
		if s != truth[i] {
			errs++
		}
	}
	return float64(errs) / float64(len(vals))
}

func randDibits(seed int64, n int) []uint8 {
	rng := rand.New(rand.NewSource(seed))
	d := make([]uint8, n)
	for i := range d {
		d[i] = uint8(rng.Intn(4))
	}
	return d
}

// railSignature is the per-outer-rail eye fingerprint used to attribute the
// mechanism of the #402 outer-rail spread. It mirrors the statistic the
// replay -diag "true-symbol outer-rail eye" block reports (cmd/gophertrunk/
// iqdiag.go), but computed here over the *whole* loopback stream with full
// ground truth (every transmitted symbol is known) rather than just the FSW
// hits — so the numbers are directly comparable to the OP's, with far more
// samples per rail.
//
// transRatio is post-transition std / steady std (the convention iqdiag uses;
// the OP reported its inverse). A ratio ≫ 1 means the spread is concentrated
// right after a ±6 swing — the ISI / symbol-timing signature. asym is
// stdN/stdP: >1 means the −3 rail is wider than +3 (the OP measured ~1.6).
type railSignature struct {
	meanP, stdP float64 // +3 rail
	meanN, stdN float64 // −3 rail
	transRatioP float64 // +3 post-transition/steady std ratio
	transRatioN float64 // −3 post-transition/steady std ratio
	asym        float64 // stdN/stdP
}

// outerRailSignature attributes each soft value to its TRUE outer symbol and
// splits by transition context: steady = this symbol equals the previous
// symbol, transition = it differs (a ±6 swing across the outer rails or an
// inner→outer step). Inner (±1) symbols are ignored. The transition flag is
// computed against the previous *true* symbol regardless of its rail, matching
// how iqdiag splits on the canonical FSW.
func outerRailSignature(vals []float32, truth []int8) railSignature {
	var p, n []float64
	var pSteady, pTrans, nSteady, nTrans []float64
	for i, v := range vals {
		switch truth[i] {
		case 3:
			p = append(p, float64(v))
			if i > 0 {
				if truth[i-1] == truth[i] {
					pSteady = append(pSteady, float64(v))
				} else {
					pTrans = append(pTrans, float64(v))
				}
			}
		case -3:
			n = append(n, float64(v))
			if i > 0 {
				if truth[i-1] == truth[i] {
					nSteady = append(nSteady, float64(v))
				} else {
					nTrans = append(nTrans, float64(v))
				}
			}
		}
	}
	var sig railSignature
	sig.meanP, sig.stdP = meanStd(p)
	sig.meanN, sig.stdN = meanStd(n)
	_, ssP := meanStd(pSteady)
	_, tsP := meanStd(pTrans)
	_, ssN := meanStd(nSteady)
	_, tsN := meanStd(nTrans)
	ratio := func(ts, ss float64) float64 {
		if ss < 1e-9 {
			return 0
		}
		return ts / ss
	}
	sig.transRatioP = ratio(tsP, ssP)
	sig.transRatioN = ratio(tsN, ssN)
	if sig.stdP > 1e-9 {
		sig.asym = sig.stdN / sig.stdP
	}
	return sig
}

// signatureFor modulates a known stream, applies imp, runs it through the
// clean chain, aligns, and returns the outer-rail signature.
func signatureFor(t *testing.T, seed int64, imp *demod.Impairments) railSignature {
	t.Helper()
	dibits := randDibits(seed, 6000)
	const warmup = 1500
	soft := runLoopback(dibits, imp)
	// Measuring +3-vs-−3 asymmetry requires a polarity-STABLE alignment: the
	// general alignSoftToDibits searches rotation 2 (polarity inversion) too,
	// which would silently swap the two rails between runs and destroy the
	// asymmetry signal. The loopback's clean eye aligns at rotation 0, and a
	// linear channel impairment does not invert polarity, so we pin rotation 0
	// and search only the lag here.
	vals, truth, frac := alignRot0(soft, dibits, warmup)
	if frac < 0.5 {
		t.Logf("warning: low alignment frac=%.3f for imp=%+v", frac, imp)
	}
	return outerRailSignature(vals, truth)
}

// alignRot0 finds the integer lag at which the fixed-sliced soft stream best
// matches the known dibits at rotation 0 (no polarity inversion), then returns
// each post-warmup soft sample paired with its TRUE transmitted symbol. Unlike
// alignSoftToDibits it never flips polarity, so the +3/−3 attribution is stable
// across impairments — required to measure rail asymmetry.
func alignRot0(soft []float32, dibits []uint8, warmup int) (vals []float32, truth []int8, matchFrac float64) {
	fixed := demod.NewC4FMWithTaps([]float32{1}, loopSlicerScale)
	sliced := fixed.SliceMany(nil, soft)
	slicedDibits := make([]uint8, len(sliced))
	for i, s := range sliced {
		slicedDibits[i] = phase1.SymbolToDibit(s)
	}
	bestOff, bestMatches, bestCnt := 0, -1, 1
	const minOff, maxOff = -200, 200
	for off := minOff; off <= maxOff; off++ {
		m, cnt := 0, 0
		for i := warmup; i < len(slicedDibits); i++ {
			j := i + off
			if j < 0 || j >= len(dibits) {
				continue
			}
			if slicedDibits[i] == dibits[j] {
				m++
			}
			cnt++
		}
		if cnt > 100 && m > bestMatches {
			bestMatches, bestOff, bestCnt = m, off, cnt
		}
	}
	for i := warmup; i < len(soft); i++ {
		j := i + bestOff
		if j < 0 || j >= len(dibits) {
			continue
		}
		vals = append(vals, soft[i])
		truth = append(truth, loopbackDibitSymbol[dibits[j]&3])
	}
	if bestCnt > 0 {
		matchFrac = float64(bestMatches) / float64(bestCnt)
	}
	return vals, truth, matchFrac
}

// TestC4FMOuterRailAsymmetryMechanism is the diagnose-first answer to the OP's
// question — is the transition-driven outer-rail spread on mmr-s9.cfile caused
// by amplitude noise (FM clicks / SNR) or by channel multipath ISI? It sweeps
// single impairments through our *clean* chain and measures the true-symbol
// outer-rail fingerprint (full ground truth, far more samples than the OP's 14
// FSW hits), then asserts the discriminating result:
//
//   - AWGN widens both rails but is transition-INDEPENDENT (post-transition std
//     ≈ steady std) — amplitude noise is not the transition-driven cause.
//   - A multipath echo (real OR complex) produces a transition-DRIVEN spread
//     (post-transition std ≫ steady std): the ISI signature, and the
//     reproducible cause of the degraded outer-rail eye.
//
// On the OP's *one-sidedness* (−3 rail ~1.6× wider than +3, from only 14 FSW
// hits): this sweep shows a two-ray echo — real or complex — spreads BOTH rails
// near-equally (stdN/stdP ≈ 1.0), and conjugating the echo's phase does not move
// it. So our symmetric chain does NOT manufacture a one-sided rail from
// multipath; the OP's left/right difference is within small-sample noise on top
// of symmetric ISI. The mechanism to fix is therefore (symmetric) multipath ISI
// — a complex IQ-domain equalizer ahead of the discriminator, since C4FM is
// constant-envelope and multipath blurs that envelope (Phase B). This test is
// the evidence; it deliberately does NOT assert a one-sided effect, because the
// data does not support one.
func TestC4FMOuterRailAsymmetryMechanism(t *testing.T) {
	echo := func(g complex64) *demod.Impairments {
		mp := make([]complex64, 21)
		mp[0] = 1
		mp[20] = g
		return &demod.Impairments{Multipath: mp}
	}

	clean := signatureFor(t, 1, nil)
	awgn := signatureFor(t, 1, &demod.Impairments{SNRdB: 12, Seed: 7})
	realEcho := signatureFor(t, 1, echo(complex(0.6, 0)))
	cplxEcho := signatureFor(t, 1, echo(complex(0.6, 0.3)))
	conjEcho := signatureFor(t, 1, echo(complex(0.6, -0.3)))

	show := func(name string, s railSignature) {
		t.Logf("%-12s +3 std=%.4f (trans/steady=%.2f)  -3 std=%.4f (trans/steady=%.2f)  stdN/stdP=%.2f",
			name, s.stdP, s.transRatioP, s.stdN, s.transRatioN, s.asym)
	}
	t.Log("outer-rail fingerprint by mechanism (post-transition/steady ratio):")
	show("clean", clean)
	show("awgn", awgn)
	show("real-echo", realEcho)
	show("complex-echo", cplxEcho)
	show("conj-echo", conjEcho)

	symmetric := func(asym float64) bool { return asym >= 0.7 && asym <= 1.4 }

	// 1. Clean chain: symmetric rails (the loopback control).
	if !symmetric(clean.asym) {
		t.Errorf("clean stdN/stdP=%.2f, want ≈1 (chain is symmetric)", clean.asym)
	}

	// 2. AWGN: both rails widen but transition-INDEPENDENT and symmetric —
	//    rules out amplitude-intrinsic noise as the transition-driven cause.
	if maxF(awgn.transRatioP, awgn.transRatioN) >= 1.25 {
		t.Errorf("AWGN transition ratios +3=%.2f -3=%.2f, expected ~1 (noise is transition-independent)",
			awgn.transRatioP, awgn.transRatioN)
	}
	if !symmetric(awgn.asym) {
		t.Errorf("AWGN stdN/stdP=%.2f, want ≈1 (noise hits both rails equally)", awgn.asym)
	}

	// 3. Multipath echo: transition-DRIVEN spread on the outer rails — the ISI
	//    signature, and the reproducible cause of the degraded eye. It also
	//    spreads both rails near-equally: multipath does NOT create a one-sided
	//    rail in our chain.
	for _, e := range []struct {
		name string
		sig  railSignature
	}{{"real-echo", realEcho}, {"complex-echo", cplxEcho}} {
		if maxF(e.sig.transRatioP, e.sig.transRatioN) < 1.3 {
			t.Errorf("%s not transition-driven (+3=%.2f -3=%.2f), want ≫1 — ISI model too weak",
				e.name, e.sig.transRatioP, e.sig.transRatioN)
		}
		if !symmetric(e.sig.asym) {
			t.Errorf("%s stdN/stdP=%.2f — multipath unexpectedly one-sided in our chain", e.name, e.sig.asym)
		}
	}

	// 4. The echo's phase does not move the asymmetry: complex and conjugate
	//    echoes leave the (near-1) rail ratio essentially unchanged. Explicit
	//    refutation of "complex multipath through the discriminator nonlinearity
	//    makes one rail wider" — the OP's one-sidedness is not a phase-driven
	//    chain effect.
	if d := math.Abs(cplxEcho.asym - conjEcho.asym); d > 0.15 {
		t.Errorf("echo conjugation moved the rail ratio by %.2f (cplx=%.2f conj=%.2f) — unexpected phase-driven asymmetry",
			d, cplxEcho.asym, conjEcho.asym)
	}
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// TestC4FMLoopbackChainSymmetric is the evidence answering the OP: a clean,
// known C4FM stream run through our receiver must produce SYMMETRIC outer
// rails. If +3 came out compressed relative to −3 on ideal input (as the OP
// measured on the capture: true +3≈+0.19 vs −3≈−0.23), our chain would be the
// cause; it is not — so the asymmetry is in the received signal.
func TestC4FMLoopbackChainSymmetric(t *testing.T) {
	dibits := randDibits(1, 6000)
	soft := runLoopback(dibits, nil)
	if len(soft) < 2500 {
		t.Fatalf("too few soft symbols: %d", len(soft))
	}
	const warmup = 1500
	vals, truth, frac, rot := alignSoftToDibits(soft, dibits, warmup)
	t.Logf("clean loopback: align=%.3f rot=%d over %d symbols", frac, rot, len(vals))
	if frac < 0.9 {
		t.Fatalf("loopback failed to decode clean synthetic C4FM (match=%.2f rot=%d) — test setup issue", frac, rot)
	}
	meanP, stdP, meanN, stdN := outerRailStats(vals, truth)
	t.Logf("clean outer rails (sps=40, open eye): +3 mean=%.4f std=%.4f   -3 mean=%.4f std=%.4f   (nominal ±%.4f)",
		meanP, stdP, meanN, stdN, loopSlicerScale)

	// Symmetry: the outer-rail magnitudes must match closely. Our linear,
	// symbol-agnostic chain treats +f and −f identically, so a clean input
	// yields |+3| ≈ |−3|. (The capture's sps=10 eye closure compresses both
	// rails — symmetric ISI — which is a separate effect from a +/− asymmetry.)
	asym := math.Abs((meanP + meanN) / meanP) // meanN<0 ⇒ sum→0 when symmetric
	if asym > 0.10 {
		t.Errorf("clean-input outer rails asymmetric by %.1f%% (|+3|=%.4f |-3|=%.4f) — OUR chain introduces asymmetry", asym*100, meanP, -meanN)
	}
	if r := stdP / stdN; r < 0.6 || r > 1.6 {
		t.Errorf("clean-input outer-rail std ratio +3/-3 = %.2f, want ≈1 (neither rail intrinsically wider)", r)
	}
}

// TestC4FMLoopbackMultipathDegradesDecode shows that injecting the Murradoc
// multipath condition into a clean stream — through our otherwise-clean chain
// — is what degrades decode. This confirms the residual is impairment-driven
// (an equalizer is the right follow-up), not a chain defect.
func TestC4FMLoopbackMultipathDegradesDecode(t *testing.T) {
	dibits := randDibits(2, 6000)
	const warmup = 1500

	clean := runLoopback(dibits, nil)
	cVals, cTruth, cFrac, cRot := alignSoftToDibits(clean, dibits, warmup)
	cleanSER := symbolErrorRate(cVals, cTruth)
	if cleanSER > 0.02 || cFrac < 0.95 {
		t.Fatalf("clean baseline broken: SER=%.4f align=%.3f rot=%d", cleanSER, cFrac, cRot)
	}

	// Two-ray multipath: a strong echo ~half a symbol late (20 samples at
	// sps=40) — the off-axis/simulcast condition the OP reports for Murradoc.
	mp := make([]complex64, 21)
	mp[0] = 1
	mp[20] = complex(0.6, 0.3)
	imp := demod.Impairments{Multipath: mp}
	dirty := runLoopback(dibits, &imp)
	dVals, dTruth, dFrac, _ := alignSoftToDibits(dirty, dibits, warmup)
	dirtySER := symbolErrorRate(dVals, dTruth)

	t.Logf("clean: align=%.3f SER=%.4f   multipath: align=%.3f SER=%.4f", cFrac, cleanSER, dFrac, dirtySER)
	// The clean chain decodes the clean stream; multipath through the SAME
	// chain visibly degrades it (more symbol errors and/or lost alignment).
	if dirtySER < cleanSER*3 && dFrac > cFrac-0.1 {
		t.Errorf("multipath did not materially degrade decode (clean SER=%.4f align=%.3f vs multipath SER=%.4f align=%.3f) — model too weak to demonstrate",
			cleanSER, cFrac, dirtySER, dFrac)
	}
}
