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

	// omegaEMA slow-tracks the per-block net estimate (coarse+fine); omegaPrimed
	// guards its first-block seed. The feed-forward estimate re-derived each
	// ~1024-symbol block is noisy: under multipath/ISI the constellation is
	// smeared, so both stages jitter block-to-block (a real reporter saw a
	// fully-locked, calm control channel report carrier_off_hz swinging −1492 →
	// +616 Hz, and the symbol-scope constellation "drift" that a
	// continuous-tracking reference demod does not show). The true carrier
	// offset drifts only slowly (tuner/TCXO), so we derotate by — and report —
	// the smoothed estimate: this strips the per-block jitter without moving the
	// tracked mean, so the differential decode is unaffected while the scope and
	// the status line go steady. A block whose raw estimate jumps by more than
	// the fine window (an ISI-induced ±f_sym/4 alias) is treated as an outlier:
	// the mean holds and the block is derotated by the smoothed value, so one
	// bad block cannot poison the track.
	omegaEMA    float64
	omegaPrimed bool

	// rejStreak/rejMean track CONSECUTIVE rejected raw estimates that agree
	// with each other. The outlier reject above has a failure mode the 19 Aug
	// field log hit for ~37 minutes: after a Reset() the EMA is seeded from a
	// single block, and a weak-signal seed can land in the wrong ±f_sym/4
	// alias bucket — after which every CORRECT estimate differs from the EMA
	// by more than omegaAliasReject and is rejected forever (BSCH still
	// decodes via the per-burst SB correction, so no watchdog fires). A run of
	// rejects that cluster tightly with EACH OTHER is the signature of a wrong
	// latch (or a genuine carrier jump), not of ISI outliers — ISI aliasing is
	// sporadic and interleaved with accepted blocks — so after
	// omegaReprimeStreak of them the EMA is re-primed from their mean.
	rejStreak int
	rejMean   float64
}

// afcBlockSymbols is the feed-forward re-estimation window in symbols.
// ~1024 symbols (~57 ms at 18 kBd) averages out the data in the 4×Δφ
// mean while still following drift.
const afcBlockSymbols = 1024

// omegaEMAAlpha is the per-block smoothing weight for the net offset estimate.
// Small enough to strip the block-to-block jitter yet large enough to follow
// real (slow) carrier drift — a block is ~57 ms, so α=0.08 is a ~0.7 s time
// constant, far faster than tuner/TCXO drift. The EMA is seeded from the first
// block, so a genuine multi-kHz offset (issue #940) is acquired immediately
// regardless of α, and only the residual jitter is smoothed.
const omegaEMAAlpha = 0.08

// omegaAliasReject is the per-block jump above which the raw estimate is treated
// as an ISI-induced alias outlier: the smoothed mean holds rather than tracking
// it. Set at the fine estimator's unambiguous half-window (π/4 ≈ f_sym/8); a
// real carrier cannot slew that far in one ~57 ms block, so only alias jumps and
// gross transients are rejected.
const omegaAliasReject = math.Pi / 4

// omegaRejectClusterTol is how tightly consecutive rejected estimates must
// agree with each other to count as one cluster (evidence of a wrong EMA
// latch rather than scattered ISI outliers). Half the reject window: tight
// enough that noise-scattered aliases restart the cluster, wide enough that
// the fine estimator's jitter around the true carrier does not.
const omegaRejectClusterTol = omegaAliasReject / 2

// omegaReprimeStreak is how many consecutive clustered rejects re-prime the
// EMA (~3 × 57 ms ≈ 170 ms of signal). A single accepted block in between
// zeroes the streak, so sporadic outliers can never trip it.
const omegaReprimeStreak = 3

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
		// Bring the block within the fine estimator's unambiguous window and
		// refine to the raw per-block net estimate. derotate(coarse) is applied
		// only to let the fine estimator measure the residual; the block's final
		// correction below is the *smoothed* net, which composes additively with
		// the coarse already removed (all derotations are absolute from phase 0).
		a.derotate(block, coarse)
		fine := estimateOmega(block, a.sps)
		rawOmega := coarse + fine

		// Smooth the net estimate across blocks: strip the per-block jitter that
		// spins the constellation and swings the reported offset, while the mean
		// (seeded from the first block, so a real multi-kHz offset is acquired at
		// once) still follows slow carrier drift. Reject alias-sized outliers,
		// but escape a wrong latch when the rejects agree with each other.
		a.track(rawOmega)
		// Finish derotating the block to the smoothed net (coarse is already
		// removed, so apply the remainder).
		a.derotate(block, a.omegaEMA-coarse)
		a.omega = a.omegaEMA

		dst = append(dst, block...)
		a.buf = append(a.buf[:0], a.buf[blockSamples:]...)
		if haveWide {
			a.wideBuf = append(a.wideBuf[:0], a.wideBuf[blockSamples:]...)
		}
	}
	return dst
}

// track folds one per-block raw estimate into the smoothed EMA. Accepted
// estimates (within omegaAliasReject of the EMA) update it and clear any
// reject streak. Rejected estimates accumulate into a cluster while they
// agree with each other (omegaRejectClusterTol); omegaReprimeStreak
// consecutive clustered rejects re-prime the EMA from their mean — the
// escape hatch for a wrong first-block seed that would otherwise reject
// every correct estimate forever (19 Aug field log: −3630 Hz latched for
// 12 minutes while BSCH decoded and SCH sat at zero).
func (a *carrierAFC) track(rawOmega float64) {
	if !a.omegaPrimed {
		a.omegaEMA = rawOmega
		a.omegaPrimed = true
		a.rejStreak = 0
		return
	}
	if math.Abs(rawOmega-a.omegaEMA) < omegaAliasReject {
		a.omegaEMA += omegaEMAAlpha * (rawOmega - a.omegaEMA)
		a.rejStreak = 0
		return
	}
	if a.rejStreak > 0 && math.Abs(rawOmega-a.rejMean) < omegaRejectClusterTol {
		a.rejStreak++
		a.rejMean += (rawOmega - a.rejMean) / float64(a.rejStreak)
	} else {
		a.rejStreak = 1
		a.rejMean = rawOmega
	}
	if a.rejStreak >= omegaReprimeStreak {
		a.omegaEMA = a.rejMean
		a.rejStreak = 0
	}
}

// OffsetHz reports the most recent per-symbol offset estimate in Hz.
func (a *carrierAFC) OffsetHz(symRate float64) float64 { return a.omega * symRate / (2 * math.Pi) }

func (a *carrierAFC) Reset() {
	a.omega = 0
	a.buf = a.buf[:0]
	a.wideBuf = a.wideBuf[:0]
	a.omegaEMA = 0
	a.omegaPrimed = false
	a.rejStreak = 0
	a.rejMean = 0
}

func conjf(c complex64) complex64 { return complex(real(c), -imag(c)) }
func anglef(c complex64) float32  { return float32(math.Atan2(float64(imag(c)), float64(real(c)))) }
