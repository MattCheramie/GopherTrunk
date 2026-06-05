package receiver

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/equalizer"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/sync"
)

// lsmRotation is the per-symbol constellation offset for P25 LSM. The
// TIA-102.BAAA LSM constellation places dibit 0b00 at +π/4, dibit 0b01
// at +3π/4, dibit 0b10 at -π/4 and dibit 0b11 at -3π/4 — same π/4-DQPSK
// family OP25's CQPSK receiver decodes. Subtracting π/4 inside the
// differential decoder centres these on {0, π/2, ±π, -π/2} so the
// standard DQPSK quadrant classifier produces stable bit pairs.
const lsmRotation = math.Pi / 4

// lsmDibitRemap converts DQPSK quadrant output to the canonical
// TIA-102.BAAA dibit convention SymbolToDibit produces from C4FM. The
// DQPSK quadrants land at:
//
//	+0    → 0b00 = 0   (matches spec)
//	+π/2  → 0b01 = 1   (matches spec)
//	±π    → 0b10 = 2   (spec dibit for ±π is 3)
//	-π/2  → 0b11 = 3   (spec dibit for -π/2 is 2)
//
// The remap swaps the last two entries so the on-air FSW dibit values
// (1 and 3) line up with the canonical FrameSyncWord pattern after
// demodulation — no rotation search needed for the CQPSK path itself.
var lsmDibitRemap = [4]uint8{0, 1, 3, 2}

// cqpskFSE* configure the T/2 fractionally-spaced blind equalizer on the CQPSK
// symbol stream. The post-Gardner symbols are unit-modulus π/4-DQPSK points, so
// the Constant Modulus Algorithm applies. A *fractionally*-spaced equalizer (two
// taps per symbol, fed the on-time and half-symbol interpolants from Gardner)
// synthesizes the receive matched filter implicitly, so it opens both simulcast
// multipath ISI AND the C4FM-transmit-vs-RRC-receive pulse mismatch that a
// symbol-spaced equalizer cannot — the latter is what left a real C4FM signal's
// constellation closed through the linear path (issue #492).
//
// cqpskFSESymbolSpan is the equalizer length in symbols (2*span T/2 taps; even,
// so the centre tap aligns with an on-time sample). cqpskFSEStep is the CMA
// step: brisk because real control-channel captures are short, so the blind
// equalizer must open the constellation within a few hundred symbols of lead-in
// before the FSW arrives; the leakage keeps that brisk step from over-adapting
// at steady state. cqpskFSELeak pulls the equalizer's larger (fractionally-
// spaced) null space toward identity so the taps do not wander on clean,
// ISI-free input (where noise alone, with no ISI gradient, would drift them).
// Tuned against real P25 C4FM captures (issue #492) while keeping the synthetic
// offset-sweep, multipath and round-trip guards green.
const (
	cqpskFSESymbolSpan = 6
	cqpskFSEStep       = 0.025
	cqpskFSELeak       = 5e-4
)

// cqpskAGC* configure the AGC that normalises the matched-filter output
// ahead of Gardner timing recovery. Both the Gardner timing-error
// detector and the CMA weight update use un-normalised, amplitude-
// dependent error terms, so the CQPSK path is gain-sensitive without
// this — issue #275's regression report measured it locking only in a
// narrow RTL-SDR gain window. cqpskAGCReference is the matched-filter
// RMS the two loops are tuned against; normalising every capture to it
// presents identical signal amplitude downstream regardless of the
// front-end gain. cqpskAGCRate is the power-EMA coefficient at the IQ
// sample rate — small, so the gain is effectively static across the
// CMA's adaptation window and the two loops do not fight.
const (
	cqpskAGCReference = 0.95
	cqpskAGCRate      = 1e-3
	cqpskAGCMaxGain   = 1e4
)

// cqpskCarrier* configure the carrier-frequency recovery the CQPSK path
// gained for issue #492. The differential π/4-DQPSK decoder removes a
// constant carrier *phase* but not a constant per-symbol *rotation*
// (2π·Δf/baud) from a residual offset Δf, so a real tuner's offset spins
// the differential constellation and the Frame Sync Word never
// correlates. (The C4FM path has CoarseAFC for exactly this; TETRA's
// π/4-DQPSK tolerates having none only because its 18000-baud rate
// rotates 3.75× less per symbol than P25's 4800.)
//
// cqpskSeedClampHz bounds the one-shot coarse seed; a residual that needs
// more than this is an un-channelised / mis-tuned stream the upstream DDC
// + PPM correction should have handled (the channel would not even fit the
// 48 kHz passband otherwise). cqpskSeedMinSamples is the sample count the
// coarse estimate needs to be usable. cqpskCostasLoopBWHz / cqpskCostasDamping
// tune the fine tracking loop: ~2.5% of baud gives a few-hundred-symbol
// acquisition, well inside the control-channel warmup.
//
// cqpskSeedMaxLag / cqpskSeedMinCoherence / cqpskSeedMaxModulusCV gate the
// coarse estimate (issue #492). The bare lag-1 (Kay) estimate reads a
// multipath / simulcast channel's autocorrelation phase as a spurious carrier
// offset (~650–750 Hz on the reported captures) because a deep simulcast null
// shifts the spectral centroid — a bias an autocorrelation estimator cannot
// tell from a real offset. robustSeedHz therefore (a) sharpens the estimate
// with a short multi-lag phase-ramp fit and (b) *gates* it: de-rotate by the
// estimate, run the matched filter, and measure the coefficient of variation
// of the symbol modulus. A pure carrier offset only rotates the constant-
// modulus π/4-DQPSK symbols (low CV); multipath ISI blurs the modulus (high
// CV). When the CV is high the offset is multipath bias, not a carrier offset,
// so the estimate is rejected and the NCO left identity — the CMA→Costas loop
// then acquires the (necessarily in-range) true offset on its own.
const (
	cqpskSeedClampHz      = 6000.0
	cqpskSeedMinSamples   = 2048
	cqpskSeedMaxLag       = 4
	cqpskSeedMinCoherence = 0.30
	cqpskSeedMaxModulusCV = 0.24
	cqpskCostasLoopBWHz   = 120.0
	cqpskCostasDamping    = 0.707
)

// robustSeedHz estimates the residual carrier offset (Hz) of an oversampled
// complex stream and reports whether the estimate is trustworthy. A pure
// carrier offset Δf rotates the stream by a constant kΔω at autocorrelation
// lag k, so angle(R_k) = k·Δω is linear in k; fitting a least-squares slope
// through lags 1..cqpskSeedMaxLag is a lower-variance estimate than the
// single-lag (Kay) angle. The estimate is then validated against the failure
// mode behind issue #492 (modulusCV): a multipath / simulcast channel shifts
// the spectral centroid so the angle reads as a spurious offset, but de-
// rotating by it and matched-filtering leaves the symbol modulus blurred by
// ISI, which a clean offset never does. Weak lag-1 coherence (noise) or a
// high modulus CV (multipath) marks the seed untrustworthy; the caller then
// leaves the NCO identity rather than de-rotating by a spurious offset.
func (c *cqpskDemod) robustSeedHz(x []complex64) (hz float64, ok bool) {
	fs := c.fs
	if len(x) <= cqpskSeedMaxLag || fs <= 0 {
		return 0, false
	}
	var energy float64
	for _, s := range x {
		energy += float64(real(s))*float64(real(s)) + float64(imag(s))*float64(imag(s))
	}
	if energy == 0 {
		return 0, false
	}
	// Autocorrelation angle at each lag: R_k = Σ x[n]·conj(x[n−k]).
	var ang [cqpskSeedMaxLag + 1]float64
	var coh float64
	for k := 1; k <= cqpskSeedMaxLag; k++ {
		var accR, accI float64
		for n := k; n < len(x); n++ {
			ar, ai := float64(real(x[n])), float64(imag(x[n]))
			br, bi := float64(real(x[n-k])), float64(imag(x[n-k]))
			accR += ar*br + ai*bi
			accI += ai*br - ar*bi
		}
		if k == 1 {
			coh = math.Hypot(accR, accI) / energy
		}
		ang[k] = math.Atan2(accI, accR)
	}
	if coh < cqpskSeedMinCoherence {
		return 0, false // too weak / noisy to seed from — let the fine loop acquire
	}
	// Unwrap each lag around k·(lag-1 angle), then least-squares fit a slope
	// θ (rad/sample) through (k, angle_k) constrained through the origin.
	a1 := ang[1]
	var num, den float64
	for k := 1; k <= cqpskSeedMaxLag; k++ {
		ak := float64(k)*a1 + math.Remainder(ang[k]-float64(k)*a1, 2*math.Pi)
		num += float64(k) * ak
		den += float64(k * k)
	}
	slope := num / den
	if c.seedModulusCV(x, slope) > cqpskSeedMaxModulusCV {
		return 0, false // ISI after de-rotation ⇒ multipath bias, not a carrier offset
	}
	hz = slope * fs / (2 * math.Pi)
	if hz > cqpskSeedClampHz {
		hz = cqpskSeedClampHz
	} else if hz < -cqpskSeedClampHz {
		hz = -cqpskSeedClampHz
	}
	return hz, true
}

// seedModulusCV de-rotates x by the candidate per-sample carrier rotation
// slopeRadPerSample, runs the RRC matched filter, decimates at the best of
// sps timing phases (max mean modulus), and returns the coefficient of
// variation (std/mean) of the symbol modulus. It is the multipath gate for
// robustSeedHz: a correct carrier estimate centres the channel and leaves the
// π/4-DQPSK symbols at constant modulus (low CV), whereas a multipath-biased
// estimate leaves the ISI that blurs the modulus (high CV).
func (c *cqpskDemod) seedModulusCV(x []complex64, slopeRadPerSample float64) float64 {
	der := make([]complex64, len(x))
	for n := range x {
		ph := -slopeRadPerSample * float64(n)
		cs, sn := math.Cos(ph), math.Sin(ph)
		xr, xi := float64(real(x[n])), float64(imag(x[n]))
		der[n] = complex(float32(xr*cs-xi*sn), float32(xr*sn+xi*cs))
	}
	mf := demod.NewPiOver4DQPSK(c.seedSps, c.seedSpan, c.seedAlpha, lsmRotation)
	y := mf.MatchedFilter(nil, der)
	if len(y) < c.seedSps {
		return 0
	}
	bestPhase, bestMean := 0, -1.0
	for ph := 0; ph < c.seedSps; ph++ {
		var sum float64
		var n int
		for i := ph; i < len(y); i += c.seedSps {
			sum += math.Hypot(float64(real(y[i])), float64(imag(y[i])))
			n++
		}
		if n > 0 && sum/float64(n) > bestMean {
			bestMean, bestPhase = sum/float64(n), ph
		}
	}
	var sum, sum2 float64
	var n int
	for i := bestPhase; i < len(y); i += c.seedSps {
		m := math.Hypot(float64(real(y[i])), float64(imag(y[i])))
		sum += m
		sum2 += m * m
		n++
	}
	if n == 0 {
		return 0
	}
	mean := sum / float64(n)
	if mean == 0 {
		return 0
	}
	varr := sum2/float64(n) - mean*mean
	if varr < 0 {
		varr = 0
	}
	return math.Sqrt(varr) / mean
}

// cqpskDemod is the LSM / linear-CQPSK symbol recovery chain for P25
// Phase 1. It wraps the shared PiOver4DQPSK primitive at rotation π/4
// and applies lsmDibitRemap so the dibits it emits are interchangeable
// with the C4FM path downstream.
//
// A CMA blind equalizer sits on the recovered symbol stream. LSM is a
// linear modulation, so simulcast multipath is a linear distortion of
// the complex symbols and an equalizer can invert it — this is the
// path simulcast P25 sites need (issue #275: strong multipath closed
// the constellation and the Frame Sync Word never correlated). An AGC
// on the matched-filter output normalises signal amplitude ahead of
// the gain-sensitive Gardner and CMA loops, so the path locks
// regardless of the RTL-SDR front-end gain.
//
// Gardner timing-recovery is mandatory on this path (the demod operates
// on complex IQ at the sample rate; naive every-sps-th decimation
// off complex IQ produces meaningless symbols at any non-trivial
// timing offset). The receiver enforces this in New.
type cqpskDemod struct {
	dq      *demod.PiOver4DQPSK
	gardner *sync.Gardner
	agc     *dsp.AGC
	fse     *equalizer.FSE   // T/2 fractionally-spaced blind equalizer
	nco     *dsp.NCO         // removes the coarse carrier seed from the raw IQ
	costas  *sync.QPSKCostas // fine carrier tracking on the symbol stream

	fs     float64 // IQ sample rate, for the NCO seed and Hz reporting
	seeded bool    // coarse carrier seed applied
	seedHz float64 // coarse carrier offset the NCO removes (Hz)

	// RRC parameters, retained so the seed's multipath gate can build a
	// throwaway matched filter to measure post-de-rotation ISI (issue #492).
	seedSps   int
	seedSpan  int
	seedAlpha float64

	// seedBuf collects raw IQ until cqpskSeedMinSamples are available for the
	// one-shot coarse carrier estimate. Production feeds the receiver only
	// ~160–200 complex samples per process() call (the ccdecoder DDC output),
	// far below cqpskSeedMinSamples, so the estimate cannot key off a single
	// chunk's length — it must accumulate across calls, then fire once. Without
	// this the per-call gate never trips on a streamed input and the NCO stays
	// an identity mix (issue #492). The buffer is an estimation tap only: the
	// live stream is still de-rotated and demodulated chunk by chunk below, and
	// the buffer is released once the seed fires.
	seedBuf []complex64

	// cmaErr is the equalizer's most recent |y|²−R² convergence proxy,
	// retained for the replay receiver-state diagnostic (issue #492).
	cmaErr float32

	// Scratch buffers reused across calls.
	rotated []complex64 // NCO-de-rotated IQ, fed to the matched filter
	matched []complex64
	twoSps  []complex64 // Gardner T/2 output: [mid, on-time] pairs, len = 2*nSymbols
	symbols []complex64
	dibits  []uint8
}

// newCQPSKDemod builds a CQPSK / LSM demod for the supplied sample
// rate and RRC parameters. sampleRateHz is the IQ rate (used by the
// carrier-recovery NCO); sps must already be the integer samples-
// per-symbol; span / alpha are the standard P25 RRC parameters
// (span=8 symbols half-width, α=0.20). gardnerGain ≤ 0 selects
// defaultGardnerGain, whose value is tuned for this path's 10 sps so the
// timing loop pulls in from any sub-symbol phase (issue #492).
func newCQPSKDemod(sampleRateHz float64, sps int, span int, alpha float64, gardnerGain float64) *cqpskDemod {
	if gardnerGain <= 0 {
		gardnerGain = defaultGardnerGain
	}
	return &cqpskDemod{
		dq:        demod.NewPiOver4DQPSK(sps, span, alpha, lsmRotation),
		gardner:   sync.NewGardner(float64(sps), gardnerGain),
		agc:       dsp.NewAGC(cqpskAGCReference, cqpskAGCRate, cqpskAGCMaxGain),
		fse:       equalizer.NewFSE(cqpskFSESymbolSpan, cqpskFSEStep, 1.0, cqpskFSELeak),
		nco:       dsp.NewNCO(0, sampleRateHz),
		costas:    sync.NewQPSKCostas(SymbolRate, cqpskCostasLoopBWHz, cqpskCostasDamping),
		fs:        sampleRateHz,
		seedSps:   sps,
		seedSpan:  span,
		seedAlpha: alpha,
	}
}

// process pushes one chunk of complex IQ through the chain and returns
// the (possibly empty) batch of dibits this call produced. Reusable
// internal buffers carry state across calls so chunk boundaries do
// not corrupt the stream.
func (c *cqpskDemod) process(iq []complex64) []uint8 {
	// Carrier recovery, coarse stage: a real tuner's residual offset is
	// far outside the fine loop's ±baud/8 pull-in, so estimate it from the
	// raw IQ and tune it out with the NCO. Estimate on the raw
	// (pre-matched-filter) stream: the RRC matched filter is a lowpass
	// centred at DC, so it clips the high sideband of an offset signal and
	// would bias the estimate toward zero. De-rotating before the matched
	// filter also presents the filter a centred channel.
	//
	// Production delivers only ~160–200 samples per call, so the estimate must
	// accumulate across calls until it has enough, rather than keying off one
	// chunk's length — otherwise the gate never trips on a streamed input and
	// the NCO stays an identity mix (issue #492). Until seeded the NCO is
	// identity, so a centred/zero-offset stream is unaffected; the fine Costas
	// loop then cleans up the residual and tracks drift. robustSeedHz only
	// trusts the estimate when its multi-lag phase fit is clean — a multipath /
	// simulcast channel biases the bare lag-1 angle into a spurious offset that
	// would mis-tune the NCO and rail the loop (issue #492), so on that signal
	// the NCO is left identity and the in-range offset is recovered by the loop.
	if !c.seeded {
		c.seedBuf = append(c.seedBuf, iq...)
		if len(c.seedBuf) >= cqpskSeedMinSamples {
			if hz, ok := c.robustSeedHz(c.seedBuf); ok {
				c.seedHz = hz
				c.nco.SetOffset(c.seedHz, c.fs)
			}
			c.seeded = true
			c.seedBuf = nil
			// The pre-seed samples ran through with an identity NCO, so the
			// Costas frequency integrator wound toward the uncorrected offset
			// (railing at its ±baud/8 clamp) and the equalizer adapted to a
			// spinning constellation. Reset both so they re-acquire on the
			// now-centred signal instead of over-de-rotating. Gardner is left
			// running — it re-locks on clean input within a few symbols, and
			// leaving it preserves symbol timing/alignment.
			c.costas.Reset()
			c.fse.Reset()
		}
	}
	c.rotated = c.nco.Mix(c.rotated, iq)
	c.matched = c.dq.MatchedFilter(c.matched, c.rotated)

	// AGC: normalise the matched-filter output to the amplitude the
	// downstream loops are tuned for. The Gardner timing-error detector
	// and the CMA weight update both use un-normalised, amplitude-
	// dependent error terms, so without this the CQPSK path is
	// gain-sensitive and only locks in a narrow RTL-SDR gain window
	// (issue #275 regression).
	c.matched = c.agc.Process(c.matched, c.matched)
	// Gardner emits the T/2 pair [mid, on-time] per symbol so the
	// fractionally-spaced equalizer sees both sub-symbol phases.
	c.twoSps = c.twoSps[:0]
	c.twoSps = c.gardner.Process2x(c.twoSps, c.matched)
	nSym := len(c.twoSps) / 2
	if nSym == 0 {
		c.dibits = c.dibits[:0]
		return c.dibits
	}
	// Blind T/2 equalizer + fine carrier recovery. The FSE synthesizes the
	// matched filter and undoes ISI (simulcast multipath and the C4FM-vs-RRC
	// pulse mismatch) so the constellation opens; the Costas loop then
	// de-rotates each symbol to remove the residual per-symbol carrier rotation
	// the coarse seed left behind, so the differential phases land on grid.
	c.symbols = c.symbols[:0]
	for i := 0; i < nSym; i++ {
		y, e := c.fse.Process(c.twoSps[2*i], c.twoSps[2*i+1])
		c.cmaErr = e
		c.symbols = append(c.symbols, c.costas.Update(y))
	}
	c.dibits = c.dq.Decode(c.dibits, c.symbols)
	for i, d := range c.dibits {
		c.dibits[i] = lsmDibitRemap[d&3]
	}
	return c.dibits
}

// carrierOffsetHz reports the carrier-recovery loop's current estimate
// of the residual carrier-frequency offset in Hz: the coarse block seed
// plus the fine Costas loop's tracked residual.
func (c *cqpskDemod) carrierOffsetHz() float64 { return c.seedHz + c.costas.OffsetHz() }

// reset clears the matched-filter history, the Gardner loop state, the
// carrier-recovery seed/loop and the differential reference sample so the
// next process call starts from a fresh stream.
func (c *cqpskDemod) reset() {
	c.dq.Reset()
	c.gardner.Reset()
	c.agc.Reset()
	c.fse.Reset()
	c.nco.Reset()
	c.costas.Reset()
	c.seeded = false
	c.seedHz = 0
	c.seedBuf = nil
}
