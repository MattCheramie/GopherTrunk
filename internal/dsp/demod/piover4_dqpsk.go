package demod

import (
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// PiOver4DQPSK is a π/4-shifted differential QPSK demodulator with an
// integrated RRC matched filter. The same primitive serves a couple
// of trunked-radio control-channel modulations; the rotation argument
// in the constructor selects between them:
//
//   - math.Pi/4 — true π/4-DQPSK as used by TETRA TMO (18000 sym/s,
//     α = 0.35) and IS-136
//   - math.Pi/8 — the π/8-shifted variant P25 Phase 2 H-DQPSK
//     (6000 sym/s, α = 0.20)
//
// Pipeline: IQ → MatchedFilter → clock recovery (one sample per
// symbol) → Decode (one dibit per symbol).
//
// MatchedFilter and Decode keep their own state; the receiver is not
// safe for concurrent calls. Instantiate one per call chain. Use
// Delay to look up the RRC group delay when synthesising test
// signals; downstream clock recovery in real use self-aligns.
type PiOver4DQPSK struct {
	rrc   *filter.FIR
	dqpsk *DQPSK
	delay int
}

// NewPiOver4DQPSK constructs a π/4-DQPSK demod with an RRC matched
// filter parameterised by samples-per-symbol, span (in symbols), and
// roll-off α. rotation selects the constellation offset (see the type
// doc). Panics if any of sps, span, alpha is non-positive.
func NewPiOver4DQPSK(sps, span int, alpha, rotation float64) *PiOver4DQPSK {
	if sps <= 0 || span <= 0 || alpha <= 0 {
		panic("demod: NewPiOver4DQPSK requires positive sps, span, alpha")
	}
	rrcTaps := filter.RootRaisedCosine(sps, span, alpha)
	dq := NewDQPSK()
	dq.SetRotation(rotation)
	return &PiOver4DQPSK{
		rrc:   filter.NewFIR(rrcTaps),
		dqpsk: dq,
		delay: (len(rrcTaps) - 1) / 2,
	}
}

// MatchedFilter applies the RRC filter to the complex IQ stream. State
// carries across calls so chunk boundaries don't corrupt the output.
func (p *PiOver4DQPSK) MatchedFilter(dst, src []complex64) []complex64 {
	return p.rrc.Process(dst, src)
}

// Decode emits one dibit (per π/4-DQPSK quadrant + rotation offset)
// per input sample. Caller is expected to have decimated the output
// of MatchedFilter to one sample per symbol via a clock-recovery
// stage (sync.MuellerMuller etc.).
func (p *PiOver4DQPSK) Decode(dst []uint8, src []complex64) []uint8 {
	return p.dqpsk.Decode(dst, src)
}

// DecodeBoth emits one hard dibit and the complex differential
// (soft information) per symbol in a single pass. See DQPSK.DecodeBoth.
func (p *PiOver4DQPSK) DecodeBoth(dstDibits []uint8, dstDiff []complex64, src []complex64) ([]uint8, []complex64) {
	return p.dqpsk.DecodeBoth(dstDibits, dstDiff, src)
}

// Reset clears the matched-filter history and the differential
// reference sample. Call on stream re-sync.
func (p *PiOver4DQPSK) Reset() {
	p.rrc.Reset()
	p.dqpsk.Reset()
}

// Delay returns the RRC matched-filter group delay (in samples).
// Test fixtures that synthesise a known signal at fixed positions use
// this to pick the correct symbol-time sample; live receivers don't
// need it because clock recovery self-aligns.
func (p *PiOver4DQPSK) Delay() int { return p.delay }
