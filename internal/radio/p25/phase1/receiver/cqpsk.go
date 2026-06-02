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

// cqpskTimingSps is the samples-per-symbol the Gardner timing-recovery
// loop is fed. The loop's linear-interpolation timing-error detector has
// a pull-in range that scales with oversampling: at the production
// channel rate (48 kHz ⇒ 10 sps) it locks for only ~10% of arbitrary
// starting symbol phases and false-locks on the rest, so a real capture —
// whose symbol clock is never aligned to sample 0 — almost never acquires
// the control channel (issue #492; the C4FM path is unaffected because
// its Mueller-Müller loop is decision-directed). Synthetic test fixtures
// start perfectly symbol-aligned, which is why this hid behind green
// tests. Upsampling the matched-filter output to ~50 sps before the loop
// widens pull-in to ~80% of phases — matching the lock rate the demod
// already showed when fed an un-decimated 2 MHz capture (~417 sps).
//
// The matched filter and AGC still run at the native rate (cheap); only
// the post-AGC interpolation and the Gardner walk run oversampled, and on
// a single control-channel stream that cost is negligible. Closing the
// remaining pull-in gap to the C4FM path's reliability needs a better
// timing interpolator in sync.Gardner itself — a follow-up that benefits
// every Gardner consumer (TETRA, P25 Phase 2).
const cqpskTimingSps = 50

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
	up      *dsp.Resampler // nil ⇒ already ≥ cqpskTimingSps; no upsampling
	gardner *sync.Gardner
	agc     *dsp.AGC
	cma     *equalizer.CMA

	// Scratch buffers reused across calls.
	matched []complex64
	upBuf   []complex64
	symbols []complex64
	dibits  []uint8
}

// newCQPSKDemod builds a CQPSK / LSM demod for the supplied sample
// rate and RRC parameters. sps must already be the integer samples-
// per-symbol; span / alpha are the standard P25 RRC parameters
// (span=8 symbols half-width, α=0.20).
//
// The matched filter runs at the native sps; its output is then
// interpolated up to ≥ cqpskTimingSps so the Gardner timing loop has the
// oversampling it needs to pull in from an arbitrary symbol phase (issue
// #492). A stream already at or above that rate skips the interpolator.
func newCQPSKDemod(sps int, span int, alpha float64, gardnerGain float64) *cqpskDemod {
	if gardnerGain <= 0 {
		gardnerGain = defaultGardnerGain
	}
	up := 1
	if sps < cqpskTimingSps {
		up = (cqpskTimingSps + sps - 1) / sps // ceil(target/sps)
	}
	c := &cqpskDemod{
		dq:      demod.NewPiOver4DQPSK(sps, span, alpha, lsmRotation),
		gardner: sync.NewGardner(float64(sps*up), gardnerGain),
		agc:     dsp.NewAGC(cqpskAGCReference, cqpskAGCRate, cqpskAGCMaxGain),
		cma:     equalizer.NewCMA(cqpskEqualizerTaps, cqpskEqualizerStep, 1.0),
	}
	if up > 1 {
		// Polyphase interpolation by `up` (Kaiser-windowed LPF, the same
		// primitive the down-converter uses). The matched-filter output is
		// already band-limited well inside the native Nyquist, so a modest
		// per-branch length suffices to suppress the interpolation images.
		c.up = dsp.NewResampler(up, 1, 8, 7.0)
	}
	return c
}

// process pushes one chunk of complex IQ through the chain and returns
// the (possibly empty) batch of dibits this call produced. Reusable
// internal buffers carry state across calls so chunk boundaries do
// not corrupt the stream.
func (c *cqpskDemod) process(iq []complex64) []uint8 {
	c.matched = c.dq.MatchedFilter(c.matched, iq)
	// AGC: normalise the matched-filter output to the amplitude the
	// downstream loops are tuned for. The Gardner timing-error detector
	// and the CMA weight update both use un-normalised, amplitude-
	// dependent error terms, so without this the CQPSK path is
	// gain-sensitive and only locks in a narrow RTL-SDR gain window
	// (issue #275 regression).
	c.matched = c.agc.Process(c.matched, c.matched)
	// Interpolate up to the timing loop's oversampling target so Gardner
	// can pull in from an arbitrary symbol phase (issue #492). At the
	// native rate skip straight to the loop.
	timing := c.matched
	if c.up != nil {
		c.upBuf = c.up.Process(c.upBuf, c.matched)
		timing = c.upBuf
	}
	c.symbols = c.symbols[:0]
	c.symbols = c.gardner.Process(c.symbols, timing)
	if len(c.symbols) == 0 {
		c.dibits = c.dibits[:0]
		return c.dibits
	}
	// Blind equalizer: CMA pulls the symbols back to constant modulus,
	// undoing the simulcast-multipath ISI that closes the constellation.
	for i, s := range c.symbols {
		y, _ := c.cma.Process(s)
		c.symbols[i] = y
	}
	c.dibits = c.dq.Decode(c.dibits, c.symbols)
	for i, d := range c.dibits {
		c.dibits[i] = lsmDibitRemap[d&3]
	}
	return c.dibits
}

// reset clears the matched-filter history, the Gardner loop state and
// the differential reference sample so the next process call starts
// from a fresh stream.
func (c *cqpskDemod) reset() {
	c.dq.Reset()
	if c.up != nil {
		c.up.Reset()
	}
	c.gardner.Reset()
	c.agc.Reset()
	c.cma.Reset()
}
