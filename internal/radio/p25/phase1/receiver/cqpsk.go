package receiver

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
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

// cqpskDemod is the LSM / linear-CQPSK symbol recovery chain for P25
// Phase 1. It wraps the shared PiOver4DQPSK primitive at rotation π/4
// and applies lsmDibitRemap so the dibits it emits are interchangeable
// with the C4FM path downstream.
//
// Gardner timing-recovery is mandatory on this path (the demod operates
// on complex IQ at the sample rate; naive every-sps-th decimation
// off complex IQ produces meaningless symbols at any non-trivial
// timing offset). The receiver enforces this in New.
type cqpskDemod struct {
	dq      *demod.PiOver4DQPSK
	gardner *sync.Gardner

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
	}
}

// process pushes one chunk of complex IQ through the chain and returns
// the (possibly empty) batch of dibits this call produced. Reusable
// internal buffers carry state across calls so chunk boundaries do
// not corrupt the stream.
func (c *cqpskDemod) process(iq []complex64) []uint8 {
	c.matched = c.dq.MatchedFilter(c.matched, iq)
	c.symbols = c.symbols[:0]
	c.symbols = c.gardner.Process(c.symbols, c.matched)
	if len(c.symbols) == 0 {
		c.dibits = c.dibits[:0]
		return c.dibits
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
}
