package receiver

import "math"

// carrierAFC removes a residual carrier-frequency offset from the
// oversampled matched-filter stream, BEFORE symbol-timing recovery. The
// live DDC has no AFC, so a channel that is not perfectly centred (a
// coarse -tune-hz, or a tuner a few hundred Hz off) leaves a per-symbol
// phase advance. π/4-DQPSK carries data in the *differential* phase, so
// a constant offset φ slides all four decision regions and biases every
// dibit. Worse, a spinning constellation corrupts the Gardner timing
// error metric (which assumes consecutive symbols share an orientation),
// so the timing loop never converges and the symbols come out garbage —
// the dominant failure on a real off-centre capture (issue #553).
//
// The offset is removed per block by a data-blind feed-forward estimate:
// for the ideal transitions {±π/4,±3π/4}, 4·Δφ ≡ π, so
// arg(mean(exp(j·4·Δφ))) over a block of symbols (decimated at sps for
// the estimate only) reveals the mean per-symbol offset ω regardless of
// the data. Every sample of the block is then derotated at ω/sps, from
// phase 0, so the Gardner downstream sees a non-spinning constellation
// and locks. Re-estimating per block tracks slow clock/thermal drift; a
// single whole-stream estimate would leave a growing residual.
type carrierAFC struct {
	sps   int
	theta float64     // derotation phase within the current block
	omega float64     // most recent per-symbol offset estimate (for OffsetHz)
	buf   []complex64 // oversampled samples awaiting a full block
}

// afcBlockSymbols is the feed-forward re-estimation window in symbols.
// ~1024 symbols (~57 ms at 18 kBd) averages out the data in the 4×Δφ
// mean while still following drift.
const afcBlockSymbols = 1024

func newCarrierAFC(sps int) *carrierAFC {
	if sps < 1 {
		sps = 1
	}
	return &carrierAFC{sps: sps, buf: make([]complex64, 0, afcBlockSymbols*sps)}
}

// estimateOmega returns the mean per-symbol phase offset over an
// oversampled block via the 4×Δφ estimator on the sps-decimated symbols,
// wrapped into (-π/4, π/4].
func estimateOmega(block []complex64, sps int) float64 {
	var sr, si float64
	for k := sps; k < len(block); k += sps {
		d := block[k] * conjf(block[k-sps])
		p := 4 * float64(anglef(d))
		sr += math.Cos(p)
		si += math.Sin(p)
	}
	if sr == 0 && si == 0 {
		return 0
	}
	off := (math.Atan2(si, sr) - math.Pi) / 4
	for off > math.Pi/4 {
		off -= math.Pi / 2
	}
	for off <= -math.Pi/4 {
		off += math.Pi / 2
	}
	return off
}

// Process derotates an oversampled (sps/symbol) matched-filter stream in
// fixed afcBlockSymbols*sps blocks, carrying nothing across blocks (each
// block is derotated from phase 0 by its own estimate, so per-block
// estimate errors don't accumulate into a drifting phase). Samples that
// do not yet fill a block are buffered for the next call; the trailing
// partial block at end-of-stream is not emitted.
func (a *carrierAFC) Process(dst, src []complex64) []complex64 {
	dst = dst[:0]
	a.buf = append(a.buf, src...)
	blockSamples := afcBlockSymbols * a.sps
	for len(a.buf) >= blockSamples {
		block := a.buf[:blockSamples]
		a.omega = estimateOmega(block, a.sps)
		perSample := a.omega / float64(a.sps)
		a.theta = 0
		for i := range block {
			rot := complex(float32(math.Cos(-a.theta)), float32(math.Sin(-a.theta)))
			block[i] *= rot
			a.theta += perSample
		}
		dst = append(dst, block...)
		a.buf = append(a.buf[:0], a.buf[blockSamples:]...)
	}
	return dst
}

// OffsetHz reports the most recent per-symbol offset estimate in Hz.
func (a *carrierAFC) OffsetHz(symRate float64) float64 { return a.omega * symRate / (2 * math.Pi) }

func (a *carrierAFC) Reset() {
	a.theta = 0
	a.omega = 0
	a.buf = a.buf[:0]
}

func conjf(c complex64) complex64 { return complex(real(c), -imag(c)) }
func anglef(c complex64) float32  { return float32(math.Atan2(float64(imag(c)), float64(real(c)))) }
