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

	// Scratch buffers reused across calls.
	matched []complex64
	symbols []complex64
	dibits  []uint8
}

// newCQPSKDemod builds a CQPSK / LSM demod for the supplied sample
// rate and RRC parameters. sps must already be the integer samples-
// per-symbol; span / alpha are the standard P25 RRC parameters
// (span=8 symbols half-width, α=0.20).
func newCQPSKDemod(sps int, span int, alpha float64, gardnerGain float64) *cqpskDemod {
	if gardnerGain <= 0 {
		gardnerGain = defaultGardnerGain
	}
	return &cqpskDemod{
		dq:      demod.NewPiOver4DQPSK(sps, span, alpha, lsmRotation),
		gardner: sync.NewGardner(float64(sps), gardnerGain),
		agc:     dsp.NewAGC(cqpskAGCReference, cqpskAGCRate, cqpskAGCMaxGain),
		cma:     equalizer.NewCMA(cqpskEqualizerTaps, cqpskEqualizerStep, 1.0),
	}
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
	c.symbols = c.symbols[:0]
	c.symbols = c.gardner.Process(c.symbols, c.matched)
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
	c.gardner.Reset()
	c.agc.Reset()
	c.cma.Reset()
}
