package sync

import "math"

// QPSKCostas is a non-data-aided carrier-recovery loop for the
// π/4-DQPSK / QPSK family. It de-rotates a recovered symbol stream to
// remove a residual carrier-frequency offset — the error a differential
// decoder cannot fix on its own, because the differential decode cancels
// a constant carrier *phase* but not a constant per-symbol *rotation*
// (2π·Δf/baud), which instead spins the whole differential constellation
// and lands every symbol in the wrong quadrant (issue #492).
//
// The error detector is the four-fold-symmetric differential phase: for
// an ideal symbol the differential product y[n]·conj(y[n−1]) sits at the
// constellation's nominal differential phases, so 4× its angle is a
// constant (mod 2π) regardless of the data and the constellation
// alternation. That constant is 4× the per-symbol constellation rotation:
// π for the π/4-DQPSK / LSM family (rotation π/4), and π/2 for P25 Phase 2
// H-DQPSK (rotation π/8). lockPhase carries it; pass it via
// NewQPSKCostasForRotation, or use NewQPSKCostas for the π/4 default. The
// wrapped residual of (4·angle − lockPhase), scaled back by 4, is the
// leftover per-symbol carrier rotation — recovered without any symbol
// decision, so the loop converges before polarity is known and is immune
// to the alternation that defeats a plain 4th-power-of-symbol estimator.
// Using the wrong lockPhase leaves a constant per-symbol bias that eats
// the differential decision margin (a π/8 bias halves it for H-DQPSK).
//
// It is a second-order loop: a proportional path corrects phase quickly
// and an integrator accumulates the frequency estimate (reported in Hz
// via OffsetHz). Its unambiguous pull-in is only ±baud/8 (±π/4 per
// symbol), so a coarse acquisition — e.g. dsp.EstimateCarrierOffsetHz on
// the matched-filter output — must bring the residual inside that range
// before this loop is engaged.
type QPSKCostas struct {
	baud      float64 // symbol rate (Hz), for the Hz <-> rad/symbol conversion
	kp, ki    float64 // proportional / integral loop-filter gains
	freq      float64 // integrator state: estimated offset, rad/symbol
	phase     float64 // running de-rotation phase, rad
	maxFreq   float64 // clamp on |freq| (rad/symbol) — keeps a noisy start bounded
	lockPhase float64 // detector lock constant = 4× constellation rotation
	rail      int     // consecutive symbols pinned at ±maxFreq pushing further in
	prev      complex64
	have      bool
}

// costasRailReacquire is how many consecutive symbols the integrator may sit
// pinned at its ±maxFreq clamp — with the error still driving it further into
// the rail — before the loop gives up that estimate and re-acquires from zero.
// A coarse carrier mis-seed (e.g. a multipath-biased estimate, issue #492) can
// otherwise drive the integrator into the clamp and hold it there, stuck, so
// the FSW never correlates. Genuine acquisition settles in tens of symbols,
// well under this, so a converging loop is never disturbed.
const costasRailReacquire = 256

// NewQPSKCostas builds a loop tuned to loopBWHz (the loop's noise
// bandwidth) at the given symbol rate, with the supplied damping
// (≈0.707 critically). Non-positive baud panics; non-positive loopBWHz
// or damping fall back to sane defaults. The discrete proportional /
// integral gains follow the standard second-order PLL design from the
// normalised loop bandwidth θ = loopBWHz / baudHz.
func NewQPSKCostas(baudHz, loopBWHz, damping float64) *QPSKCostas {
	// Default lock phase π = 4×(π/4): the π/4-DQPSK / LSM constellation.
	return NewQPSKCostasForRotation(baudHz, loopBWHz, damping, math.Pi/4)
}

// NewQPSKCostasForRotation is NewQPSKCostas for a constellation whose
// per-symbol rotation is `rotation` rad (π/4 for the π/4-DQPSK / LSM
// family, π/8 for P25 Phase 2 H-DQPSK). The detector locks to 4×rotation;
// passing the right value keeps the loop from settling with a constant
// per-symbol bias that would eat the differential decision margin.
func NewQPSKCostasForRotation(baudHz, loopBWHz, damping, rotation float64) *QPSKCostas {
	if baudHz <= 0 {
		panic("qpskcostas: baudHz must be positive")
	}
	if loopBWHz <= 0 {
		loopBWHz = baudHz * 0.0125 // ~1.25% of baud
	}
	if damping <= 0 {
		damping = 0.707
	}
	theta := loopBWHz / baudHz
	denom := 1 + 2*damping*theta + theta*theta
	kp := 4 * damping * theta / denom
	ki := 4 * theta * theta / denom
	return &QPSKCostas{
		baud:      baudHz,
		kp:        kp,
		ki:        ki,
		maxFreq:   math.Pi / 4, // the detector's unambiguous half-range
		lockPhase: wrapPi(4 * rotation),
	}
}

// Update consumes one recovered symbol, de-rotates it by the loop's
// current carrier-phase estimate, advances the loop, and returns the
// de-rotated symbol. Feed the returned symbols straight to the
// differential decoder — the carrier rotation has been removed, so the
// differential phases land back on their nominal {±π/4, ±3π/4} grid.
func (c *QPSKCostas) Update(y complex64) complex64 {
	si, co := math.Sincos(-c.phase)
	rot := complex(float32(co), float32(si))
	yd := y * rot

	if c.have {
		// Differential product of consecutive de-rotated symbols. Its
		// phase is the data phase (±π/4, ±3π/4) plus whatever per-symbol
		// carrier rotation the loop has not yet removed.
		pr, pi := real(c.prev), imag(c.prev)
		yr, yi := real(yd), imag(yd)
		pdr := yr*pr + yi*pi // Re{yd · conj(prev)}
		pdi := yi*pr - yr*pi // Im{yd · conj(prev)}
		ang := math.Atan2(float64(pdi), float64(pdr))
		// Strip the four-fold data modulation (and the constellation
		// alternation): 4·ang ≡ lockPhase for any ideal symbol, so the
		// wrapped residual of (4·ang − lockPhase)/4 is the leftover
		// per-symbol carrier rotation, unambiguous over (−π/4, π/4].
		e := wrapPi(4*ang-c.lockPhase) / 4
		c.freq += c.ki * e
		if c.freq > c.maxFreq {
			c.freq = c.maxFreq
		} else if c.freq < -c.maxFreq {
			c.freq = -c.maxFreq
		}
		// Anti-windup / re-acquire: if the integrator stays pinned at the clamp
		// with the error still pushing it further in, the estimate is wrong, not
		// merely large — collapse it back toward zero so the loop re-acquires
		// instead of staying railed for the whole capture (issue #492).
		if (c.freq == c.maxFreq && e > 0) || (c.freq == -c.maxFreq && e < 0) {
			c.rail++
			if c.rail >= costasRailReacquire {
				c.freq = 0
				c.rail = 0
			}
		} else {
			c.rail = 0
		}
		c.phase += c.freq + c.kp*e
	} else {
		c.phase += c.freq
	}
	c.phase = wrapPi(c.phase)
	c.prev = yd
	c.have = true
	return yd
}

// OffsetHz returns the loop's current estimate of the residual carrier
// frequency offset in Hz (freq · baud / 2π).
func (c *QPSKCostas) OffsetHz() float64 { return c.freq * c.baud / (2 * math.Pi) }

// Reset clears the loop so a stream re-sync starts from zero phase and
// zero frequency estimate.
func (c *QPSKCostas) Reset() {
	c.freq = 0
	c.phase = 0
	c.rail = 0
	c.prev = 0
	c.have = false
}

// wrapPi wraps an angle to (−π, π].
func wrapPi(a float64) float64 {
	a = math.Mod(a, 2*math.Pi)
	if a > math.Pi {
		a -= 2 * math.Pi
	} else if a <= -math.Pi {
		a += 2 * math.Pi
	}
	return a
}
