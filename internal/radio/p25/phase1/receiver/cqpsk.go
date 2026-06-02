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

// cqpskEqualizerTaps / cqpskEqualizerStep configure the CMA blind
// equalizer on the CQPSK symbol stream. The post-Gardner LSM symbols
// are unit-modulus QPSK points, so the Constant Modulus Algorithm
// applies: simulcast multipath blurs that constant magnitude and CMA
// drives it back, opening the constellation so the FSW correlates.
//
// cqpskEqualizerStep is tuned for the AGC-normalised symbol amplitude.
// CMA convergence speed scales with the input power, so before the AGC
// the equalizer leaned on the power a multipath echo itself adds; with
// the level now fixed the step is set to converge at that normalised
// amplitude instead.
const (
	cqpskEqualizerTaps = 11
	cqpskEqualizerStep = 0.008
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
// 48 kHz passband otherwise). cqpskSeedMinSamples is the matched-filter
// sample count the coarse estimate needs to be usable. cqpskCostasLoopBWHz
// / cqpskCostasDamping tune the fine tracking loop: ~2.5% of baud gives a
// few-hundred-symbol acquisition, well inside the control-channel warmup.
const (
	cqpskSeedClampHz    = 6000.0
	cqpskSeedMinSamples = 2048
	cqpskCostasLoopBWHz = 120.0
	cqpskCostasDamping  = 0.707
)

// seedCarrierOffsetHz returns a coarse estimate of the residual carrier
// offset (Hz) of an oversampled complex stream, via the lag-1
// autocorrelation (Kay) frequency estimator: the angle of Σ x[n]·conj(x[n−1])
// is the mean per-sample phase increment, which for a suppressed-carrier
// symmetric modulation is the carrier rotation (the data's phase changes
// are symmetric and average out). Unlike peak-picking the PSD, this is
// unbiased on the flat-topped RRC spectrum of an already-channelised
// signal, and unambiguous to ±Fs/2 rather than ±baud/8.
func seedCarrierOffsetHz(x []complex64, fs float64) float64 {
	if len(x) < 2 || fs <= 0 {
		return 0
	}
	var accR, accI float64
	for n := 1; n < len(x); n++ {
		ar, ai := float64(real(x[n])), float64(imag(x[n]))
		br, bi := float64(real(x[n-1])), float64(imag(x[n-1]))
		// x[n] · conj(x[n-1])
		accR += ar*br + ai*bi
		accI += ai*br - ar*bi
	}
	return seedHzFromAcc(accR, accI, fs)
}

// seedHzFromAcc converts running lag-1 autocorrelation accumulators
// (Σ x[n]·conj(x[n−1])) to a clamped coarse carrier estimate in Hz. Shared
// by seedCarrierOffsetHz (slice) and the streaming seed that accumulates
// across process() calls, so both use identical math.
func seedHzFromAcc(accR, accI, fs float64) float64 {
	hz := math.Atan2(accI, accR) * fs / (2 * math.Pi)
	if hz > cqpskSeedClampHz {
		hz = cqpskSeedClampHz
	} else if hz < -cqpskSeedClampHz {
		hz = -cqpskSeedClampHz
	}
	return hz
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
	cma     *equalizer.CMA
	nco     *dsp.NCO         // removes the coarse carrier seed from the raw IQ
	costas  *sync.QPSKCostas // fine carrier tracking on the symbol stream

	fs     float64 // IQ sample rate, for the NCO seed and Hz reporting
	seeded bool    // coarse carrier seed applied
	seedHz float64 // coarse carrier offset the NCO removes (Hz)

	// Coarse-seed accumulators. Production feeds the receiver only
	// ~160–200 complex samples per process() call (the ccdecoder DDC
	// output), far below cqpskSeedMinSamples, so the seed cannot key off a
	// single chunk's length — it must accumulate the lag-1 autocorrelation
	// Σ x[n]·conj(x[n−1]) across calls until it has seen enough raw samples,
	// then fire once. Without this the per-call gate never trips on a
	// streamed input and the NCO stays an identity mix (issue #492).
	seedAccR     float64   // running Re Σ x[n]·conj(x[n−1])
	seedAccI     float64   // running Im Σ x[n]·conj(x[n−1])
	seedCount    int       // lag-1 products folded into the accumulators
	seedHavePrev bool      // seedPrev holds a valid cross-call boundary sample
	seedPrev     complex64 // last raw IQ sample of the previous chunk

	// cmaErr is the CMA's most recent |y|²−R² convergence proxy,
	// retained for the replay receiver-state diagnostic (issue #492).
	cmaErr float32

	// Scratch buffers reused across calls.
	rotated []complex64 // NCO-de-rotated IQ, fed to the matched filter
	matched []complex64
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
		dq:      demod.NewPiOver4DQPSK(sps, span, alpha, lsmRotation),
		gardner: sync.NewGardner(float64(sps), gardnerGain),
		agc:     dsp.NewAGC(cqpskAGCReference, cqpskAGCRate, cqpskAGCMaxGain),
		cma:     equalizer.NewCMA(cqpskEqualizerTaps, cqpskEqualizerStep, 1.0),
		nco:     dsp.NewNCO(0, sampleRateHz),
		costas:  sync.NewQPSKCostas(SymbolRate, cqpskCostasLoopBWHz, cqpskCostasDamping),
		fs:      sampleRateHz,
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
	// Production delivers only ~160–200 samples per call, so the seed must
	// accumulate the lag-1 autocorrelation across calls (carrying the
	// boundary sample) until it has enough, rather than keying off one
	// chunk's length — otherwise the gate never trips on a streamed input
	// and the NCO stays an identity mix (issue #492). Until seeded the NCO
	// is identity, so a centred/zero-offset stream is unaffected; the fine
	// Costas loop then cleans up the residual and tracks drift.
	if !c.seeded {
		for n := 0; n < len(iq); n++ {
			if c.seedHavePrev {
				ar, ai := float64(real(iq[n])), float64(imag(iq[n]))
				br, bi := float64(real(c.seedPrev)), float64(imag(c.seedPrev))
				c.seedAccR += ar*br + ai*bi // Re x[n]·conj(x[n−1])
				c.seedAccI += ai*br - ar*bi // Im x[n]·conj(x[n−1])
				c.seedCount++
			}
			c.seedPrev = iq[n]
			c.seedHavePrev = true
		}
		if c.seedCount >= cqpskSeedMinSamples {
			c.seedHz = seedHzFromAcc(c.seedAccR, c.seedAccI, c.fs)
			c.nco.SetOffset(c.seedHz, c.fs)
			c.seeded = true
			// The pre-seed samples ran through with an identity NCO, so the
			// Costas frequency integrator wound toward the uncorrected offset
			// (railing at its ±baud/8 clamp) and the CMA adapted to a spinning
			// constellation. Reset both so they re-acquire on the now-centred
			// signal instead of over-de-rotating. Gardner is left running — it
			// re-locks on clean input within a few symbols, and leaving it
			// preserves symbol timing/alignment.
			c.costas.Reset()
			c.cma.Reset()
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
	c.symbols = c.symbols[:0]
	c.symbols = c.gardner.Process(c.symbols, c.matched)
	if len(c.symbols) == 0 {
		c.dibits = c.dibits[:0]
		return c.dibits
	}
	// Blind equalizer + fine carrier recovery. CMA pulls the symbols back
	// to constant modulus, undoing the simulcast-multipath ISI that closes
	// the constellation; the Costas loop then de-rotates each symbol to
	// remove the residual per-symbol carrier rotation the coarse seed left
	// behind, so the differential phases land on their nominal grid.
	for i, s := range c.symbols {
		y, e := c.cma.Process(s)
		c.cmaErr = e
		c.symbols[i] = c.costas.Update(y)
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
	c.cma.Reset()
	c.nco.Reset()
	c.costas.Reset()
	c.seeded = false
	c.seedHz = 0
	c.seedAccR = 0
	c.seedAccI = 0
	c.seedCount = 0
	c.seedHavePrev = false
	c.seedPrev = 0
}
