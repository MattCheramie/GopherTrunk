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
	sps     int
	omega   float64     // most recent per-symbol offset estimate (for OffsetHz)
	buf     []complex64 // oversampled matched-filter samples awaiting a full block
	wideBuf []complex64 // parallel pre-matched (channel-filtered) samples for the coarse stage
}

// afcBlockSymbols is the feed-forward re-estimation window in symbols.
// ~1024 symbols (~57 ms at 18 kBd) averages out the data in the 4×Δφ
// mean while still following drift.
const afcBlockSymbols = 1024

func newCarrierAFC(sps int) *carrierAFC {
	if sps < 1 {
		sps = 1
	}
	return &carrierAFC{
		sps:     sps,
		buf:     make([]complex64, 0, afcBlockSymbols*sps),
		wideBuf: make([]complex64, 0, afcBlockSymbols*sps),
	}
}

// coarseOmega estimates the mean carrier offset over a block as the angle of
// its lag-1 autocorrelation — the power-weighted spectral centroid of a
// symmetric π/4-DQPSK spectrum — scaled to rad/symbol. Unlike the 4×Δφ
// differential estimator (estimateOmega), it has no ±π/4-per-symbol wrap: its
// unambiguous range spans the whole passband, so it fixes which
// π/2-per-symbol (f_sym/4 Hz) alias bucket the fine estimator's result sits
// in. It must be read from the *pre-matched-filter* (channel-filtered) stream:
// the matched filter is centred at 0 Hz and pulls a spectral estimate of an
// off-centre carrier back toward 0, which would defeat the point. The channel
// filter bounds the usable acquisition range to roughly ±(cutoff − signal
// half-bandwidth) ≈ ±6 kHz at the production 15 kHz cutoff.
func coarseOmega(block []complex64, sps int) float64 {
	var sr, si float64
	for k := 1; k < len(block); k++ {
		d := block[k] * conjf(block[k-1])
		sr += float64(real(d))
		si += float64(imag(d))
	}
	if sr == 0 && si == 0 {
		return 0
	}
	// atan2 is the mean phase advance per sample (rad/sample); ×sps → rad/symbol.
	return math.Atan2(si, sr) * float64(sps)
}

// derotate multiplies each sample of block by e^{-jωk/sps}, k counting from 0
// at the block start — a per-block absolute derotation by ω rad/symbol.
func (a *carrierAFC) derotate(block []complex64, omega float64) {
	perSample := omega / float64(a.sps)
	theta := 0.0
	for i := range block {
		rot := complex(float32(math.Cos(-theta)), float32(math.Sin(-theta)))
		block[i] *= rot
		theta += perSample
	}
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
//
// Each block is corrected in two stages. A coarse mean-frequency estimate on
// the parallel pre-matched-filter (channel-filtered) block picks the alias
// bucket — extending the acquisition range past the differential estimator's
// ±f_sym/8 ≈ ±2250 Hz wrap to roughly ±6 kHz — and the existing 4×Δφ
// estimator then refines the residual on the coarse-derotated matched block.
// wide is that pre-matched stream (parallel to matched, same length); pass nil
// to disable the coarse stage and fall back to fine-only (the historical
// behaviour), which the receiver does when no channel filter is active and the
// wideband stream would still carry adjacent carriers.
func (a *carrierAFC) Process(dst, matched, wide []complex64) []complex64 {
	dst = dst[:0]
	a.buf = append(a.buf, matched...)
	// Keep wideBuf strictly parallel to buf, but only while the pre-matched
	// stream is supplied every call at the matching length; otherwise drop the
	// coarse stage rather than risk a misaligned read (matches the soft-buffer
	// discipline in the control-channel SB path).
	if wide != nil && len(wide) == len(matched) && len(a.wideBuf) == len(a.buf)-len(matched) {
		a.wideBuf = append(a.wideBuf, wide...)
	} else if len(a.wideBuf) != len(a.buf) {
		a.wideBuf = a.wideBuf[:0]
	}
	blockSamples := afcBlockSymbols * a.sps
	for len(a.buf) >= blockSamples {
		block := a.buf[:blockSamples]

		var coarse float64
		haveWide := len(a.wideBuf) >= blockSamples
		if haveWide {
			coarse = coarseOmega(a.wideBuf[:blockSamples], a.sps)
		}
		// Bring the block within the fine estimator's unambiguous window, then
		// refine. Total offset = coarse + fine; both derotations are absolute
		// from phase 0, so per-block errors never accumulate.
		a.derotate(block, coarse)
		fine := estimateOmega(block, a.sps)
		a.derotate(block, fine)
		a.omega = coarse + fine

		dst = append(dst, block...)
		a.buf = append(a.buf[:0], a.buf[blockSamples:]...)
		if haveWide {
			a.wideBuf = append(a.wideBuf[:0], a.wideBuf[blockSamples:]...)
		}
	}
	return dst
}

// OffsetHz reports the most recent per-symbol offset estimate in Hz.
func (a *carrierAFC) OffsetHz(symRate float64) float64 { return a.omega * symRate / (2 * math.Pi) }

func (a *carrierAFC) Reset() {
	a.omega = 0
	a.buf = a.buf[:0]
	a.wideBuf = a.wideBuf[:0]
}

func conjf(c complex64) complex64 { return complex(real(c), -imag(c)) }
func anglef(c complex64) float32  { return float32(math.Atan2(float64(imag(c)), float64(real(c)))) }
