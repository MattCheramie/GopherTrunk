package receiver

// Synthetic BER-vs-SNR sweep + hard CI gate (measurement-harness component C).
//
// This drives the real receiver across a range of injected SNRs on both demod
// paths, measures the recovered-symbol (dibit) error rate and the
// metrics-package SNR estimate at each, and compares the measured curve to the
// closed-form theoretical reference. It answers the harness's founding
// question deterministically: is the demod tracking the theoretical limit
// (signal-limited) or sitting far off it (implementation-limited)?
//
// TestSweepImplementationLossBudget is the hard gate: the Es/N0 each path needs
// to reach a target symbol-error rate must stay within a committed loss budget
// of the reference, the measured SER must fall monotonically with SNR, and the
// SNR estimator must track (or at least rise with) the injected SNR. It is
// deterministic (seeded synthesis, fixed SNR ladder).

import (
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1/metrics"
)

// sweepPoint is one (injected-SNR → measured) sample of the demod's behaviour.
type sweepPoint struct {
	snrDB    float64 // injected per-sample in-band SNR (Impairments.SNRdB)
	esN0dB   float64 // derived per-symbol Es/N0 = snrDB + 10log10(sps)
	ser      float64 // measured recovered-symbol (dibit) error rate
	snrEstDB float64 // metrics-package SNR estimate from the soft taps
	softSyms int     // soft/constellation samples the estimate was formed over
}

// sweepSeed fixes the AWGN draw so the whole sweep is reproducible.
const sweepSeed = 0x5175

// runSweepPoint modulates the canonical control-channel stream for one demod
// path, overlays AWGN at snrDB, runs it through the real receiver, and returns
// the measured symbol-error rate plus the metrics-package SNR estimate taken
// from the receiver's soft (C4FM) or constellation (CQPSK) taps.
func runSweepPoint(mode DemodMode, snrDB float64) sweepPoint {
	canonical := buildHarnessDibits(harnessNAC, harnessFrameRepeats)
	iq := demod.ApplyImpairments(modulateHarness(canonical, mode), harnessSampleRateHz,
		demod.Impairments{SNRdB: snrDB, Seed: sweepSeed})

	var recovered []uint8
	var soft []float32
	var syms []complex64
	opts := Options{
		SampleRateHz: harnessSampleRateHz,
		DeviationHz:  harnessDeviationHz,
		DemodMode:    mode,
		DibitSink:    func(d []uint8, _ int) { recovered = append(recovered, d...) },
		SoftSink:     func(s []float32) { soft = append(soft, s...) },
		SymbolSink:   func(c []complex64) { syms = append(syms, c...) },
	}
	r := New(opts)
	for i := 0; i < len(iq); i += harnessChunk {
		end := i + harnessChunk
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[i:end])
	}

	p := sweepPoint{snrDB: snrDB, esN0dB: metrics.EsN0FromSNR(snrDB, harnessSPS)}
	p.ser = dibitErrorRate(canonical, recovered)

	// Estimate SNR from the demod taps, skipping the AGC/clock warmup so the
	// statistic reflects steady state. The C4FM path surfaces the real
	// 4-level soft axis (SoftSink); the CQPSK path surfaces the complex
	// constellation (SymbolSink).
	const warmupSkip = 256
	if mode == DemodCQPSK {
		if len(syms) > warmupSkip {
			syms = syms[warmupSkip:]
		}
		p.softSyms = len(syms)
		p.snrEstDB = metrics.SNRM2M4Constellation(syms)
	} else {
		if len(soft) > warmupSkip {
			soft = soft[warmupSkip:]
		}
		p.softSyms = len(soft)
		outer := metrics.EstimateOuterRailC4FM(soft)
		p.snrEstDB = metrics.SNRResidualC4FM(soft, outer)
	}
	return p
}

// sweepSNRs is the injected-SNR ladder. It spans the regime from a closed eye
// (high SER) up through a clean lock, so the measured SER crosses the target
// somewhere inside the range on both paths.
var sweepSNRs = []float64{2, 4, 6, 8, 10, 12, 15, 18, 20, 25, 30, 35, 40}

// runSweep collects a sweepPoint at every SNR for one demod path.
func runSweep(mode DemodMode) []sweepPoint {
	pts := make([]sweepPoint, len(sweepSNRs))
	for i, snr := range sweepSNRs {
		pts[i] = runSweepPoint(mode, snr)
	}
	return pts
}

// esN0AtSER returns the Es/N0 (dB) at which the measured SER curve crosses
// targetSER, by linear interpolation in (Es/N0, log10 SER). Returns NaN if the
// curve never brackets the target within the swept range.
func esN0AtSER(pts []sweepPoint, targetSER float64) float64 {
	lt := math.Log10(targetSER)
	for i := 1; i < len(pts); i++ {
		hi, lo := pts[i-1], pts[i] // SER decreases as Es/N0 rises
		if hi.ser <= 0 || lo.ser <= 0 {
			continue
		}
		lhi, llo := math.Log10(hi.ser), math.Log10(lo.ser)
		if (lhi-lt)*(llo-lt) <= 0 && lhi != llo { // target bracketed
			f := (lt - lhi) / (llo - lhi)
			return hi.esN0dB + f*(lo.esN0dB-hi.esN0dB)
		}
	}
	return math.NaN()
}

// sweepGate is the per-path acceptance configuration for the hard gate.
type sweepGate struct {
	mode DemodMode
	name string
	// ref is the closed-form symbol-error-rate reference the loss is taken
	// against (coherent QPSK for CQPSK, coherent 4-PAM for C4FM).
	ref func(esN0dB float64) float64
	// targetSER is the symbol-error rate at which the measured and reference
	// Es/N0 are compared.
	targetSER float64
	// lossBudgetDB is the maximum tolerated gap between the Es/N0 the demod
	// needs to reach targetSER and the reference's. It is a *regression
	// ceiling* calibrated from the measured baseline plus margin — NOT a
	// claim that the gap is small. The C4FM budget is large precisely
	// because the FM-discriminator path's noise fragility (compounded by
	// timing-slip behaviour the single-lag SER measurement folds in) sits
	// it ~24 dB off the coherent bound; enshrining that as a ceiling is what
	// stops it silently getting worse, and quantifies the headroom that
	// would justify future demod work.
	lossBudgetDB float64
	// estTrackBand is the Es/N0 window where the SNR estimate should track
	// the injected Es/N0 (thermal-noise-dominated, below the implementation
	// EVM floor). Zero-valued (both 0) skips the absolute-tracking check —
	// used for C4FM, whose nonlinear FM discrimination makes the soft-axis
	// SNR a different, lower quantity than the input Es/N0, so only
	// monotonicity is meaningful there.
	estTrackBand  [2]float64
	estTrackTolDB float64
}

// sweepGates calibrate the gate from the committed baseline measured by
// TestSweepImplementationLossBudget's own logs (re-run it and read the "loss="
// lines to re-derive these). CQPSK lands ~3.85 dB off coherent QPSK at 1% SER;
// C4FM ~23.8 dB off coherent 4-PAM. Budgets add ~2–3 dB of margin.
var sweepGates = []sweepGate{
	{
		mode: DemodCQPSK, name: "cqpsk",
		ref: metrics.SERCoherentQPSK, targetSER: 1e-2, lossBudgetDB: 6.0,
		estTrackBand: [2]float64{12, 20}, estTrackTolDB: 3.0,
	},
	{
		mode: DemodC4FM, name: "c4fm",
		ref: metrics.SER4PAM, targetSER: 1e-2, lossBudgetDB: 27.0,
		// FM discrimination decouples soft-axis SNR from input Es/N0:
		// monotonicity only.
		estTrackBand: [2]float64{}, estTrackTolDB: 0,
	},
}

// TestSweepImplementationLossBudget is the hard CI gate. For each demod path it
// sweeps injected SNR through the real receiver, logs the measured SER and SNR
// estimate at each step (the calibration record), and asserts:
//
//   - the measured SER is monotone non-increasing in Es/N0 (a demod whose
//     errors climb with SNR is broken);
//   - the Es/N0 needed to reach the path's target SER stays within the
//     committed loss budget of the theoretical reference (the regression
//     ceiling), so the demod can never silently fall further behind theory;
//   - the SNR estimator tracks (CQPSK) or at least rises monotonically with
//     (C4FM) the injected SNR over the locked regime — validating the
//     estimator itself, not just the demod.
//
// Deterministic: seeded synthesis, fixed SNR ladder.
func TestSweepImplementationLossBudget(t *testing.T) {
	for _, g := range sweepGates {
		t.Run(g.name, func(t *testing.T) {
			pts := runSweep(g.mode)
			for _, p := range pts {
				t.Logf("sweep %-5s  SNR=%+5.1f dB  Es/N0=%+5.1f dB  SER=%.5f  SNRest=%5.1f dB (n=%d)",
					g.name, p.snrDB, p.esN0dB, p.ser, p.snrEstDB, p.softSyms)
			}

			// 1. SER monotone non-increasing in Es/N0 (small epsilon absorbs
			//    the seed's run-to-run noise wiggle at a fixed eye).
			const serWiggle = 0.03
			for i := 1; i < len(pts); i++ {
				if pts[i].ser > pts[i-1].ser+serWiggle {
					t.Errorf("SER rose with SNR: Es/N0 %.0f→%.0f dB gave SER %.4f→%.4f",
						pts[i-1].esN0dB, pts[i].esN0dB, pts[i-1].ser, pts[i].ser)
				}
			}

			// 2. Implementation loss within the committed ceiling.
			meas := esN0AtSER(pts, g.targetSER)
			theory := metrics.InverseBERForEbN0(g.ref, g.targetSER, -10, 60)
			if math.IsNaN(meas) {
				t.Fatalf("measured SER never reached the target %g within the swept range — "+
					"extend sweepSNRs or the demod regressed below the gate's reach", g.targetSER)
			}
			loss := meas - theory
			t.Logf("%s SER=%g: measured Es/N0=%.2f dB, theory=%.2f dB, loss=%.2f dB (budget %.1f dB)",
				g.name, g.targetSER, meas, theory, loss, g.lossBudgetDB)
			if loss > g.lossBudgetDB {
				t.Errorf("%s implementation loss %.2f dB exceeds budget %.1f dB at SER=%g — "+
					"the demod fell behind theory; investigate before raising the budget",
					g.name, loss, g.lossBudgetDB, g.targetSER)
			}

			// 3. SNR estimator behaviour over the locked regime.
			assertEstimator(t, g, pts)
		})
	}
}

// assertEstimator checks the SNR estimate against the injected Es/N0: absolute
// tracking within tolerance for paths with a defined estTrackBand (CQPSK), and
// monotone-non-decreasing over the locked regime otherwise (C4FM).
func assertEstimator(t *testing.T, g sweepGate, pts []sweepPoint) {
	t.Helper()
	if g.estTrackBand[1] > g.estTrackBand[0] {
		var prev float64
		var have bool
		for _, p := range pts {
			if p.esN0dB < g.estTrackBand[0] || p.esN0dB > g.estTrackBand[1] {
				continue
			}
			if d := math.Abs(p.snrEstDB - p.esN0dB); d > g.estTrackTolDB {
				t.Errorf("%s SNR estimate %.1f dB strays %.1f dB from injected Es/N0 %.1f dB (tol %.1f)",
					g.name, p.snrEstDB, d, p.esN0dB, g.estTrackTolDB)
			}
			if have && p.snrEstDB < prev-0.5 {
				t.Errorf("%s SNR estimate fell as injected SNR rose: %.1f→%.1f dB", g.name, prev, p.snrEstDB)
			}
			prev, have = p.snrEstDB, true
		}
		return
	}
	// Monotonicity over the locked regime (where SER < 0.5 — the eye is open
	// enough that the estimate measures signal, not noise).
	var prev float64
	var have bool
	for _, p := range pts {
		if p.ser >= 0.5 {
			continue
		}
		if have && p.snrEstDB < prev-1.0 {
			t.Errorf("%s SNR estimate fell as injected SNR rose over the locked regime: %.1f→%.1f dB",
				g.name, prev, p.snrEstDB)
		}
		prev, have = p.snrEstDB, true
	}
}
