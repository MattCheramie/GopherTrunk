package receiver

import "math"

// carrierAFC removes a residual carrier-frequency offset from the
// recovered π/4-DQPSK symbol stream. The live DDC has no AFC, so a
// channel that is not perfectly centred (the common case for a wideband
// capture replayed with a coarse -tune-hz, or an SDR with a few-hundred-
// Hz tuner error) leaves a constant per-symbol phase advance. π/4-DQPSK
// carries data in the *differential* phase, so a constant offset φ adds
// φ to every symbol transition and slides all four decision regions —
// the dibits come out systematically biased and the training-sequence
// correlator never aligns (issue #553: clean signal, never locks).
//
// The offset is corrected by derotating each symbol s[k] by a phase
// accumulator that advances by the tracked per-symbol offset ω. Since
// the differential y[k]·conj(y[k-1]) of the derotated stream subtracts
// ω from every transition, driving ω to the value that zeroes the mean
// transition bias recentres the constellation.
//
// ω is acquired open-loop (data-blind) from the first window via the
// 4×Δφ estimator — for ideal transitions {±π/4,±3π/4}, 4·Δφ ≡ π, so
// arg(mean(exp(j·4·Δφ))) = π + 4ω reveals ω regardless of the data —
// then tracked closed-loop with a slow decision-directed update so the
// loop follows thermal drift without the data modulation pulling it.
type carrierAFC struct {
	acquired bool
	omega    float64 // tracked per-symbol offset (rad/symbol)
	theta    float64 // running derotation phase
	mu       float64 // closed-loop tracking gain

	acqBuf  []complex64 // acquisition window (raw symbols)
	acqWant int

	prev     complex64 // previous derotated symbol
	havePrev bool
}

// afcAcquireSymbols is the data-blind acquisition window. ~512 symbols
// (~28 ms at 18 kBd) is long enough for the 4×Δφ mean to average out the
// data yet short enough to acquire well inside a warmup.
const afcAcquireSymbols = 512

func newCarrierAFC(mu float64) *carrierAFC {
	if mu <= 0 {
		mu = 0.002
	}
	return &carrierAFC{mu: mu, acqWant: afcAcquireSymbols, acqBuf: make([]complex64, 0, afcAcquireSymbols)}
}

var afcIdeal = [4]float64{math.Pi / 4, 3 * math.Pi / 4, -3 * math.Pi / 4, -math.Pi / 4}

func nearestIdeal(p float64) float64 {
	best, bd := afcIdeal[0], math.Pi
	for _, c := range afcIdeal {
		d := math.Abs(wrapPi(p - c))
		if d < bd {
			bd, best = d, c
		}
	}
	return best
}

func wrapPi(x float64) float64 {
	for x < -math.Pi {
		x += 2 * math.Pi
	}
	for x >= math.Pi {
		x -= 2 * math.Pi
	}
	return x
}

// Process derotates src in place-equivalent into dst (reusing dst's
// capacity) and returns it. The phase accumulator and tracked offset
// carry across calls so a chunked stream converges once.
func (a *carrierAFC) Process(dst, src []complex64) []complex64 {
	dst = dst[:0]
	for _, s := range src {
		if !a.acquired {
			a.acqBuf = append(a.acqBuf, s)
			if len(a.acqBuf) >= a.acqWant {
				a.acquire()
			}
			// During acquisition pass symbols through untouched; the
			// detector tolerates the short un-AFC'd warmup.
			dst = append(dst, s)
			continue
		}
		rot := complex(float32(math.Cos(-a.theta)), float32(math.Sin(-a.theta)))
		y := s * rot
		a.theta = wrapPi(a.theta + a.omega)
		if a.havePrev {
			// Decision-directed: nudge ω toward the value that zeroes the
			// transition-phase bias.
			d := y * conjf(a.prev)
			e := wrapPi(float64(anglef(d)) - nearestIdeal(float64(anglef(d))))
			a.omega += a.mu * e
		}
		a.prev = y
		a.havePrev = true
		dst = append(dst, y)
	}
	return dst
}

// acquire runs the data-blind 4×Δφ estimator over the buffered window
// to seed omega, then switches to closed-loop tracking.
func (a *carrierAFC) acquire() {
	var sr, si float64
	for k := 1; k < len(a.acqBuf); k++ {
		d := a.acqBuf[k] * conjf(a.acqBuf[k-1])
		p := 4 * float64(anglef(d))
		sr += math.Cos(p)
		si += math.Sin(p)
	}
	// arg(mean) = π + 4ω  ⇒  ω = (arg − π)/4, wrapped into ±π/4.
	off := (math.Atan2(si, sr) - math.Pi) / 4
	for off > math.Pi/4 {
		off -= math.Pi / 2
	}
	for off < -math.Pi/4 {
		off += math.Pi / 2
	}
	a.omega = off
	a.acquired = true
	a.acqBuf = a.acqBuf[:0]
}

// OffsetHz reports the tracked offset in Hz given the symbol rate.
func (a *carrierAFC) OffsetHz(symRate float64) float64 { return a.omega * symRate / (2 * math.Pi) }

func (a *carrierAFC) Reset() {
	a.acquired = false
	a.omega = 0
	a.theta = 0
	a.acqBuf = a.acqBuf[:0]
	a.prev = 0
	a.havePrev = false
}

func conjf(c complex64) complex64 { return complex(real(c), -imag(c)) }
func anglef(c complex64) float32  { return float32(math.Atan2(float64(imag(c)), float64(real(c)))) }
