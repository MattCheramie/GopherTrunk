package filter

// CICDecimator is a multi-stage CIC decimator with rate factor R and N stages.
// Output is at input/R rate. Suitable for an initial coarse decimation
// before a sharper FIR; CIC has a sin(x)/x droop that should be compensated
// by a downstream FIR.
type CICDecimator struct {
	r          int
	n          int
	integrator []int64 // I(z) = 1/(1 - z^-1), one per stage
	combDelay  []int64 // comb tap state (one slot per stage)
	count      int
}

func NewCICDecimator(rate, stages int) *CICDecimator {
	if rate < 1 || stages < 1 {
		panic("cic: rate and stages must be >= 1")
	}
	return &CICDecimator{
		r:          rate,
		n:          stages,
		integrator: make([]int64, stages),
		combDelay:  make([]int64, stages),
	}
}

// ProcessReal decimates a real signal scaled into int16 (caller scales as
// needed). Returns the decimated samples appended to dst.
func (c *CICDecimator) ProcessReal(dst []int64, src []int64) []int64 {
	for _, s := range src {
		// Integrators
		acc := s
		for i := 0; i < c.n; i++ {
			c.integrator[i] += acc
			acc = c.integrator[i]
		}
		c.count++
		if c.count == c.r {
			c.count = 0
			// Comb stages — z^-1 of the decimated signal.
			y := acc
			for i := 0; i < c.n; i++ {
				prev := c.combDelay[i]
				c.combDelay[i] = y
				y = y - prev
			}
			dst = append(dst, y)
		}
	}
	return dst
}

// Gain returns the DC gain of the cascade: R^N. Callers typically divide
// the output by this value (or shift right by ceil(N*log2(R))).
func (c *CICDecimator) Gain() int64 {
	g := int64(1)
	for i := 0; i < c.n; i++ {
		g *= int64(c.r)
	}
	return g
}
