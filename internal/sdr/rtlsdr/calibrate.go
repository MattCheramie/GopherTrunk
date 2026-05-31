package rtlsdr

import "math"

// DCBlocker subtracts a slowly-tracked DC bias from a complex sample stream.
// Useful because RTL-SDR dongles park noticeable DC at the tuned center bin.
type DCBlocker struct {
	alpha float64
	avgI  float64
	avgQ  float64
}

func NewDCBlocker(alpha float64) *DCBlocker {
	if alpha <= 0 || alpha >= 1 {
		alpha = 0.001
	}
	return &DCBlocker{alpha: alpha}
}

func (d *DCBlocker) Process(in []complex64) {
	for i, s := range in {
		ri := float64(real(s))
		qi := float64(imag(s))
		d.avgI += d.alpha * (ri - d.avgI)
		d.avgQ += d.alpha * (qi - d.avgQ)
		in[i] = complex(float32(ri-d.avgI), float32(qi-d.avgQ))
	}
}

// IQBalancer applies a single-coefficient IQ-imbalance correction
// (gain + phase) measured against a calibration tone. Coefficients are 1.0
// and 0.0 by default (passthrough).
type IQBalancer struct {
	GainQ float32 // amplitude scale for Q
	Phase float32 // radians, applied to Q via Q' = Q*cos - I*sin
}

func (b IQBalancer) Process(in []complex64) {
	cos := float32(math.Cos(float64(b.Phase)))
	sin := float32(math.Sin(float64(b.Phase)))
	gain := b.GainQ
	if gain == 0 {
		gain = 1
	}
	for i, s := range in {
		ri := real(s)
		qi := imag(s) * gain
		in[i] = complex(ri, qi*cos-ri*sin)
	}
}

// PPMToHz returns the absolute-Hz offset implied by a PPM value at the given
// center frequency. Useful when reporting calibration results.
func PPMToHz(ppm int, centerHz uint32) int64 {
	return int64(ppm) * int64(centerHz) / 1_000_000
}

// IQImbalanceStats accumulates the second-order moments of a complex stream —
// E[I²], E[Q²], E[I·Q] — and derives the front-end I/Q gain/phase imbalance
// from them. For a constant-envelope signal (FM / C4FM) the ideal baseband is
// circularly symmetric (E[I²] = E[Q²], E[I·Q] = 0), so any departure measures
// the receiver's I/Q imbalance directly. Used both as a replay diagnostic and
// as the estimator behind IQImbalanceCorrector (issue #402: an uncorrected
// RTL-SDR imbalance, worst at the on-channel DC the DDC sits on, is the
// leading explanation for the asymmetric demodulated eye).
type IQImbalanceStats struct {
	sumII, sumQQ, sumIQ float64
	n                   uint64
}

// Observe folds a chunk of raw IQ into the running moments.
func (s *IQImbalanceStats) Observe(in []complex64) {
	for _, c := range in {
		i := float64(real(c))
		q := float64(imag(c))
		s.sumII += i * i
		s.sumQQ += q * q
		s.sumIQ += i * q
	}
	s.n += uint64(len(in))
}

// Count returns the number of samples observed so far.
func (s *IQImbalanceStats) Count() uint64 { return s.n }

// ok reports whether enough non-degenerate data has accumulated to derive.
func (s *IQImbalanceStats) ok() bool {
	return s.n > 0 && s.sumII > 1e-20 && s.sumQQ > 1e-20
}

// GainImbalanceDB is 10·log10(E[I²]/E[Q²]): the I-vs-Q power imbalance in dB
// (0 = balanced). Equivalently 20·log10(GainQ).
func (s *IQImbalanceStats) GainImbalanceDB() float64 {
	if !s.ok() {
		return 0
	}
	return 10 * math.Log10(s.sumII/s.sumQQ)
}

// PhaseImbalanceDeg is the I/Q quadrature error in degrees, from the
// normalized I·Q correlation (0 = perfect quadrature).
func (s *IQImbalanceStats) PhaseImbalanceDeg() float64 {
	if !s.ok() {
		return 0
	}
	return math.Asin(s.correlation()) * 180 / math.Pi
}

// correlation is the normalized E[I·Q] (the sine of the quadrature error).
func (s *IQImbalanceStats) correlation() float64 {
	if !s.ok() {
		return 0
	}
	rho := s.sumIQ / math.Sqrt(s.sumII*s.sumQQ)
	switch {
	case rho > 1:
		rho = 1
	case rho < -1:
		rho = -1
	}
	return rho
}

// ImageRejectionDB approximates the front-end image-rejection ratio implied by
// the measured imbalance (higher is better; a clean front-end is ≳ 40 dB). The
// small-imbalance approximation image/signal ≈ (εgain² + φ²)/4.
func (s *IQImbalanceStats) ImageRejectionDB() float64 {
	if !s.ok() {
		return 0
	}
	gainErr := s.balancerGain() - 1
	phi := math.Asin(s.correlation())
	denom := gainErr*gainErr + phi*phi
	if denom < 1e-12 {
		return 99
	}
	return 10 * math.Log10(4/denom)
}

// balancerGain is the GainQ the IQBalancer needs to equalize the I/Q powers.
func (s *IQImbalanceStats) balancerGain() float64 {
	if !s.ok() {
		return 1
	}
	return math.Sqrt(s.sumII / s.sumQQ)
}

// Balancer returns the IQBalancer coefficients that correct the measured
// imbalance: GainQ equalizes the I/Q powers and Phase de-correlates I and Q
// (Q' = GainQ·Q·cosΦ − I·sinΦ ⇒ E[I·Q']=0, E[Q'²]=E[I²]). Identity when
// insufficient/degenerate data has been seen.
func (s *IQImbalanceStats) Balancer() IQBalancer {
	if !s.ok() {
		return IQBalancer{GainQ: 1, Phase: 0}
	}
	return IQBalancer{
		GainQ: float32(s.balancerGain()),
		Phase: float32(math.Atan(s.correlation())),
	}
}

// IQImbalanceCorrector blindly estimates and removes the front-end I/Q
// imbalance from a streaming complex signal. It tracks the second-order
// moments of the *raw* input with a slow EMA, derives the correcting
// IQBalancer once a warmup has elapsed, and applies it in-place. Because the
// moments are always measured from the raw input (never the corrected output)
// there is no feedback loop, and a static hardware imbalance converges and
// stays. Off the hot path by default — opt-in (issue #402). Not safe for
// concurrent use.
type IQImbalanceCorrector struct {
	alpha         float64
	mII, mQQ, mIQ float64
	seen, warmup  uint64
	bal           IQBalancer
}

// NewIQImbalanceCorrector returns a corrector with a slow moment EMA (~100k
// samples, ≈40 ms at 2.4 MSPS) and a short warmup before it starts correcting.
func NewIQImbalanceCorrector() *IQImbalanceCorrector {
	return &IQImbalanceCorrector{
		alpha:  1.0 / 100_000.0,
		warmup: 50_000,
		bal:    IQBalancer{GainQ: 1, Phase: 0},
	}
}

// Process updates the imbalance estimate from the raw input, then corrects the
// chunk in-place. During warmup it is a passthrough (GainQ=1, Phase=0).
func (c *IQImbalanceCorrector) Process(in []complex64) {
	for _, s := range in {
		i := float64(real(s))
		q := float64(imag(s))
		c.mII += c.alpha * (i*i - c.mII)
		c.mQQ += c.alpha * (q*q - c.mQQ)
		c.mIQ += c.alpha * (i*q - c.mIQ)
	}
	c.seen += uint64(len(in))
	if c.seen >= c.warmup && c.mII > 1e-20 && c.mQQ > 1e-20 {
		rho := c.mIQ / math.Sqrt(c.mII*c.mQQ)
		switch {
		case rho > 1:
			rho = 1
		case rho < -1:
			rho = -1
		}
		c.bal.GainQ = float32(math.Sqrt(c.mII / c.mQQ))
		c.bal.Phase = float32(math.Atan(rho))
	}
	c.bal.Process(in)
}

// Coefficients returns the corrector's current GainQ and Phase (radians) — for
// surfacing the converged estimate in diagnostics.
func (c *IQImbalanceCorrector) Coefficients() (gainQ, phaseRad float32) {
	return c.bal.GainQ, c.bal.Phase
}
