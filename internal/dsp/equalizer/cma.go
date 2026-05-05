package equalizer

// CMA is the Constant Modulus Algorithm — a blind adaptive equaliser
// that requires no training sequence. It exploits the fact that
// PSK-family signals (BPSK / QPSK / π/4-DQPSK / 8PSK) have a constant
// modulus on the air; multipath / simulcast distortion blurs that
// constant-magnitude property, and CMA drives the output back toward
// it.
//
// Cost function and gradient (Godard / CMA-2):
//
//   J = E[(|y|^2 - R^2)^2]
//   ∂J/∂w*  ∝  (|y|^2 - R^2) · y · conj(x)
//   w[n+1]  =  w[n]  -  μ · (|y|^2 - R^2) · y · conj(x)
//
// Pick R^2 so the equilibrium weight scaling matches the expected
// constellation. For unit-magnitude PSK use R^2 = 1; QPSK with
// Gray-coded ±1±j has |y|^2 = 2 so R^2 = 2 is conventional.
//
// Caveats:
//
//   - CMA is phase-blind. After convergence the constellation may
//     sit at any rotation; downstream symbol mapping must apply a
//     constellation-aware phase recovery (a per-constellation rotator
//     using known training symbols, or differential decoding).
//   - On non-constant-modulus signals (FM, OQPSK, QAM) CMA's cost
//     function isn't zero at the right answer; use LMS in
//     decision-directed mode there.
type CMA struct {
	taps      []complex64
	hist      []complex64
	histPos   int
	stepSize  float32
	target    float32 // R^2
}

// NewCMA constructs a blind equaliser. `target` is the desired squared
// modulus (R^2): use 1.0 for unit-modulus PSK, 2.0 for ±1±j QPSK.
func NewCMA(taps int, stepSize, target float32) *CMA {
	if taps <= 0 {
		panic("equalizer: CMA taps must be > 0")
	}
	if target <= 0 {
		panic("equalizer: CMA target must be > 0")
	}
	c := &CMA{
		taps:     make([]complex64, taps),
		hist:     make([]complex64, taps),
		stepSize: stepSize,
		target:   target,
	}
	c.taps[taps/2] = complex(1, 0)
	return c
}

// Reset returns the equaliser to centre-spike initial state.
func (c *CMA) Reset() {
	for i := range c.taps {
		c.taps[i] = 0
	}
	c.taps[len(c.taps)/2] = complex(1, 0)
	for i := range c.hist {
		c.hist[i] = 0
	}
	c.histPos = 0
}

// Taps returns a copy of the current weight vector.
func (c *CMA) Taps() []complex64 {
	out := make([]complex64, len(c.taps))
	copy(out, c.taps)
	return out
}

// Process consumes one input sample and returns the equalised output.
// The error proxy (|y|^2 - R^2) is also returned for diagnostics /
// convergence-monitoring; once it settles near zero the equaliser
// has opened the constellation.
func (c *CMA) Process(x complex64) (complex64, float32) {
	c.hist[c.histPos] = x
	c.histPos = (c.histPos + 1) % len(c.hist)

	// FIR output (same convolution as LMS).
	var yr, yi float32
	idx := c.histPos - 1
	if idx < 0 {
		idx = len(c.hist) - 1
	}
	for i := 0; i < len(c.taps); i++ {
		hr, hi := real(c.hist[idx]), imag(c.hist[idx])
		tr, ti := real(c.taps[i]), imag(c.taps[i])
		yr += tr*hr - ti*hi
		yi += tr*hi + ti*hr
		idx--
		if idx < 0 {
			idx = len(c.hist) - 1
		}
	}
	y := complex(yr, yi)

	// Error proxy = |y|^2 - R^2.
	mag2 := yr*yr + yi*yi
	err := mag2 - c.target

	// Weight update: w_k -= μ · err · y · conj(x[n-k]).
	mu := c.stepSize
	idx = c.histPos - 1
	if idx < 0 {
		idx = len(c.hist) - 1
	}
	for i := 0; i < len(c.taps); i++ {
		xr, xi := real(c.hist[idx]), imag(c.hist[idx])
		// y · conj(x) = (yr + j*yi)(xr - j*xi) = (yr*xr + yi*xi) + j*(yi*xr - yr*xi)
		ur := yr*xr + yi*xi
		ui := yi*xr - yr*xi
		// scale by err and step size, subtract.
		tr, ti := real(c.taps[i]), imag(c.taps[i])
		c.taps[i] = complex(tr-mu*err*ur, ti-mu*err*ui)
		idx--
		if idx < 0 {
			idx = len(c.hist) - 1
		}
	}
	return y, err
}
