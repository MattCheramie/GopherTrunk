package siglab

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/fft"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1/metrics"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr"
)

// SignalQuality is the protocol-agnostic demod-quality summary the analyzer
// produces when Config.CollectIQDiag is set. It generalizes the parts of the
// historical P25-only iqdiag report that apply to every protocol: the
// recovered-symbol distribution (which exposes a collapsed or mis-calibrated
// slicer) and the raw front-end I/Q imbalance (the leading cause of an
// asymmetric eye). The deeper P25-specific FSW/NID landscape lives in
// P25P1Detail.
type SignalQuality struct {
	// SymbolCardinality is 4 for the dibit (4-level) protocols and 2 for
	// the bit (2-level) protocols.
	SymbolCardinality int `json:"symbol_cardinality" yaml:"symbol_cardinality"`
	// SymbolHistogram counts recovered symbols per value (len ==
	// SymbolCardinality). A clean 4-level C4FM control channel sits near
	// 25% per bin; a near-empty bin means the slicer collapsed, a single
	// dominant bin means the signal is below the slicer thresholds.
	SymbolHistogram []int64 `json:"symbol_histogram" yaml:"symbol_histogram"`
	// SymbolHistogramPct is SymbolHistogram normalised to percentages.
	SymbolHistogramPct []float64 `json:"symbol_histogram_pct" yaml:"symbol_histogram_pct"`

	// Raw (pre-DDC) front-end I/Q imbalance. A clean front-end is balanced
	// (≈0 dB gain, ≈0° phase) with image rejection ≳ 40 dB. Populated only
	// when raw IQ was observed.
	IQGainImbalanceDB   float64 `json:"iq_gain_imbalance_db" yaml:"iq_gain_imbalance_db"`
	IQPhaseImbalanceDeg float64 `json:"iq_phase_imbalance_deg" yaml:"iq_phase_imbalance_deg"`
	IQImageRejectionDB  float64 `json:"iq_image_rejection_db" yaml:"iq_image_rejection_db"`
	IQObserved          bool    `json:"iq_observed" yaml:"iq_observed"`

	// RMSPowerDBFS is the raw-capture RMS power in dBFS (20·log10 of the RMS
	// amplitude; 0 dBFS = a full-scale signal). It is the capture-level power
	// figure the `spectrum` CLI reports; populated only when raw IQ was observed.
	RMSPowerDBFS float64 `json:"rms_power_dbfs" yaml:"rms_power_dbfs"`

	// DecodeErrorRate is decode-error events per recovered 1000 symbols — a
	// protocol-neutral proxy for FEC stress.
	DecodeErrorRate float64 `json:"decode_error_rate_per_ksym" yaml:"decode_error_rate_per_ksym"`

	// Demod is the demodulator-quality measurement (EVM + estimated SNR)
	// derived from the per-symbol soft samples on the P25 deep path. Nil for
	// protocols / runs that do not surface the soft stream. It is what turns
	// "the audio sounds bad" into "the eye is N% open / the demod is M dB from
	// the noise floor."
	Demod *DemodMetrics `json:"demod,omitempty" yaml:"demod,omitempty"`
}

// DemodMetrics is the demodulator-quality summary computed from the recovered
// soft symbols: error-vector magnitude (constellation dispersion) and the SNR
// implied by the residual about the ideal symbol positions. It is the
// siglab-surfaced face of internal/radio/p25/phase1/metrics.
type DemodMetrics struct {
	// Modulation names the estimator that produced these numbers: "c4fm" (the
	// 4-level soft eye) or "cqpsk" (the complex π/4-DQPSK constellation).
	Modulation string `json:"modulation" yaml:"modulation"`
	// EVMPct is the RMS error-vector magnitude as a percentage of the ideal
	// symbol level. A clean lock sits at a few percent; it climbs toward the
	// inter-rail half-spacing (~33% for C4FM) as the eye closes.
	EVMPct float64 `json:"evm_pct" yaml:"evm_pct"`
	// SNREstimateDB is the symbol SNR implied by the residual, in dB. For
	// C4FM this is the post-discriminator soft-axis SNR (lower than the input
	// Es/N0 by the FM detection loss); for CQPSK it is the constellation SNR.
	// Capped at demodSNRCapDB for a noise-free synthetic input.
	SNREstimateDB float64 `json:"snr_estimate_db" yaml:"snr_estimate_db"`
	// SymbolsAnalyzed is the soft-sample count the estimate was formed over
	// (after the warmup skip).
	SymbolsAnalyzed int64 `json:"symbols_analyzed" yaml:"symbols_analyzed"`

	// The fields below are the vector-signal-analyzer (VSA) decomposition of
	// the same recovered symbols — the breakdown a lab VSA reports beside the
	// constellation. PeakEVMPct, MagErrPct and the EVMTrace populate on both
	// paths; PhaseErrDeg, IQGainImbalanceDB, QuadratureErrorDeg and
	// OriginOffsetPct require the I/Q plane and are populated on the CQPSK
	// (constellation) path only — on the 1-D C4FM soft axis they stay zero.

	// PeakEVMPct is the worst single-symbol EVM (≥ EVMPct).
	PeakEVMPct float64 `json:"peak_evm_pct" yaml:"peak_evm_pct"`
	// MagErrPct / PhaseErrDeg split the error vector into its amplitude
	// (RMS magnitude error, %) and angular (RMS phase error, degrees) parts.
	MagErrPct   float64 `json:"mag_err_pct" yaml:"mag_err_pct"`
	PhaseErrDeg float64 `json:"phase_err_deg" yaml:"phase_err_deg"`
	// CarrierFreqErrorHz is the residual carrier-frequency error the receiver's
	// AFC/CQPSK loop settled at, in Hz (surfaced from the receiver, not
	// re-estimated). Zero when the deep receiver did not expose it.
	CarrierFreqErrorHz float64 `json:"carrier_freq_error_hz" yaml:"carrier_freq_error_hz"`
	// IQGainImbalanceDB / QuadratureErrorDeg are the I/Q gain imbalance and
	// quadrature-skew measured on the *recovered* constellation — distinct from
	// SignalQuality's pre-DDC front-end IQ figures.
	IQGainImbalanceDB  float64 `json:"iq_gain_imbalance_db" yaml:"iq_gain_imbalance_db"`
	QuadratureErrorDeg float64 `json:"quadrature_error_deg" yaml:"quadrature_error_deg"`
	// OriginOffsetPct is the residual DC (carrier-leakage) offset as a percent
	// of the reference radius.
	OriginOffsetPct float64 `json:"origin_offset_pct" yaml:"origin_offset_pct"`
	// EVMTrace is the per-symbol EVM% time series, stride-decimated to the IQ
	// cap. ErrorVectorSpectrum is the FFT-shifted magnitude spectrum (dB) of
	// the residual error vectors — a spur in it points at a periodic
	// impairment rather than white noise. Both omitempty so the CSV/summary
	// path is unaffected.
	EVMTrace            []float32 `json:"evm_trace,omitempty" yaml:"evm_trace,omitempty"`
	ErrorVectorSpectrum []float32 `json:"error_vector_spectrum,omitempty" yaml:"error_vector_spectrum,omitempty"`
}

// demodWarmupSkip drops the leading soft samples so the AGC/clock/AFC
// acquisition transient does not bias the EVM/SNR estimate toward the closed
// eye it starts from. Matches the sweep harness's warmup skip.
const demodWarmupSkip = 256

// demodSNRCapDB bounds the reported SNR so a noise-free synthetic input (whose
// residual is ~0 → SNR → +Inf) yields a finite, JSON-marshalable number rather
// than an infinity encoding/json refuses to emit.
const demodSNRCapDB = 99.0

// analyzer accumulates the observations behind a SignalQuality. It is fed
// from the engine's SymbolTap (symbols) and read loop (raw IQ), so it works
// for every protocol the factory map drives.
type analyzer struct {
	cardinality int
	hist        []int64
	symbols     int64
	iqStats     rtlsdr.IQImbalanceStats
	iqObserved  bool
	// rmsSqSum / rmsN accumulate Σ|x|² and the sample count over the raw IQ so
	// result() can report the capture's RMS power without buffering it.
	rmsSqSum float64
	rmsN     int64

	// bufferSymbols retains the full recovered-symbol stream for the
	// protocol-specific deep dive (P25 P1's FSW/NID landscape). Off by
	// default since it is O(symbols) memory.
	bufferSymbols bool
	symBuf        []uint8
	// softBuf retains the pre-slicer per-symbol soft samples (deep P25 C4FM
	// path, fed from the receiver's SoftSink), aligned index-for-index with
	// symBuf, for the true-symbol eye analysis.
	softBuf []float32
	// constBuf retains the per-symbol complex constellation points (deep P25
	// CQPSK path, fed from the receiver's SymbolSink). Populated only on the
	// linear path; its presence is what tells result() to use the
	// constellation estimators rather than the C4FM soft-axis ones.
	constBuf []complex64
}

// observeConstellation appends a chunk of complex symbol-decision points to
// the rolling buffer (deep P25 CQPSK path only).
func (a *analyzer) observeConstellation(pts []complex64) {
	a.constBuf = append(a.constBuf, pts...)
}

// observeSoft appends a chunk of pre-slicer soft samples to the rolling
// buffer (deep P25 path only). Aligned with the dibit stream when the
// receiver emits one soft sample per recovered symbol.
func (a *analyzer) observeSoft(soft []float32) {
	a.softBuf = append(a.softBuf, soft...)
}

func newAnalyzer() *analyzer { return &analyzer{} }

// observeSymbols folds a recovered-symbol chunk into the histogram. The
// cardinality (2 vs 4) is inferred from the isBits flag the SymbolTap
// carries; it is set on the first chunk and stays fixed thereafter.
func (a *analyzer) observeSymbols(symbols []uint8, isBits bool) {
	if a.hist == nil {
		if isBits {
			a.cardinality = 2
		} else {
			a.cardinality = 4
		}
		a.hist = make([]int64, a.cardinality)
	}
	mask := uint8(a.cardinality - 1)
	for _, s := range symbols {
		a.hist[s&mask]++
	}
	a.symbols += int64(len(symbols))
	if a.bufferSymbols {
		a.symBuf = append(a.symBuf, symbols...)
	}
}

// observeIQ folds a chunk of raw (pre-DDC) IQ into the imbalance moments.
func (a *analyzer) observeIQ(raw []complex64) {
	a.iqStats.Observe(raw)
	a.iqObserved = true
	for _, v := range raw {
		re, im := float64(real(v)), float64(imag(v))
		a.rmsSqSum += re*re + im*im
	}
	a.rmsN += int64(len(raw))
}

// result builds the SignalQuality from the accumulated observations.
// decodeErrors / symbols give the per-ksym error rate; maxTracePoints caps the
// decimated VSA EVM trace.
func (a *analyzer) result(decodeErrors int64, maxTracePoints int) *SignalQuality {
	sq := &SignalQuality{
		SymbolCardinality:  a.cardinality,
		SymbolHistogram:    a.hist,
		SymbolHistogramPct: make([]float64, len(a.hist)),
	}
	if a.symbols > 0 {
		for i, n := range a.hist {
			sq.SymbolHistogramPct[i] = 100 * float64(n) / float64(a.symbols)
		}
		sq.DecodeErrorRate = 1000 * float64(decodeErrors) / float64(a.symbols)
	}
	if a.iqObserved && a.iqStats.Count() > 0 {
		sq.IQObserved = true
		sq.IQGainImbalanceDB = a.iqStats.GainImbalanceDB()
		sq.IQPhaseImbalanceDeg = a.iqStats.PhaseImbalanceDeg()
		sq.IQImageRejectionDB = a.iqStats.ImageRejectionDB()
	}
	if a.rmsN > 0 {
		sq.RMSPowerDBFS = dsp.Mag2dB(math.Sqrt(a.rmsSqSum / float64(a.rmsN)))
	}
	sq.Demod = a.demodMetrics(maxTracePoints)
	return sq
}

// demodMetrics computes the EVM + estimated SNR from the buffered soft symbols,
// choosing the estimator by which buffer the receiver populated: the complex
// constellation (CQPSK / linear path) when present, otherwise the 4-level soft
// eye (C4FM). Returns nil when neither buffer holds enough post-warmup samples
// to form a stable estimate.
func (a *analyzer) demodMetrics(maxTracePoints int) *DemodMetrics {
	if len(a.constBuf) > demodWarmupSkip {
		pts := a.constBuf[demodWarmupSkip:]
		v := metrics.VSAMetricsConstellation(pts)
		// I/Q gain imbalance + quadrature skew of the recovered constellation,
		// via the same moment estimator used for the front-end figures.
		var iqs rtlsdr.IQImbalanceStats
		iqs.Observe(pts)
		dm := &DemodMetrics{
			Modulation:          "cqpsk",
			EVMPct:              v.RMSEVMPct,
			SNREstimateDB:       capSNR(metrics.SNRM2M4Constellation(pts)),
			SymbolsAnalyzed:     int64(len(pts)),
			PeakEVMPct:          v.PeakEVMPct,
			MagErrPct:           v.MagErrPct,
			PhaseErrDeg:         v.PhaseErrDeg,
			OriginOffsetPct:     v.OriginOffsetPct,
			IQGainImbalanceDB:   iqs.GainImbalanceDB(),
			QuadratureErrorDeg:  iqs.PhaseImbalanceDeg(),
			EVMTrace:            decimateF32(v.PerSymbolEVMPct, maxTracePoints),
			ErrorVectorSpectrum: errorVectorSpectrum(v.ErrorVectors),
		}
		return dm
	}
	if len(a.softBuf) > demodWarmupSkip {
		soft := a.softBuf[demodWarmupSkip:]
		outer := metrics.EstimateOuterRailC4FM(soft)
		if outer <= 0 {
			return nil
		}
		v := metrics.VSAMetricsC4FM(soft, outer)
		return &DemodMetrics{
			Modulation:      "c4fm",
			EVMPct:          v.RMSEVMPct,
			SNREstimateDB:   capSNR(metrics.SNRResidualC4FM(soft, outer)),
			SymbolsAnalyzed: int64(len(soft)),
			PeakEVMPct:      v.PeakEVMPct,
			MagErrPct:       v.MagErrPct,
			EVMTrace:        decimateF32(v.PerSymbolEVMPct, maxTracePoints),
		}
	}
	return nil
}

// decimateF32 stride-decimates xs to at most maxPoints values, preserving the
// first sample and the overall shape (the same cap discipline as the IQ taps).
func decimateF32(xs []float32, maxPoints int) []float32 {
	n := len(xs)
	if n == 0 || maxPoints <= 0 {
		return nil
	}
	stride := 1
	if n > maxPoints {
		stride = (n + maxPoints - 1) / maxPoints
	}
	out := make([]float32, 0, n/stride+1)
	for i := 0; i < n; i += stride {
		out = append(out, xs[i])
	}
	return out
}

// evSpectrumSize is the fixed FFT length for the error-vector spectrum: small,
// power-of-two, enough to reveal a periodic impairment line.
const evSpectrumSize = 256

// errorVectorSpectrum returns the FFT-shifted magnitude spectrum (dB) of the
// residual error-vector time series — a peak at a non-zero bin marks a periodic
// impairment (a spur / residual tone), a flat floor marks white noise. Returns
// nil when there are too few error vectors to transform.
func errorVectorSpectrum(evs []complex64) []float32 {
	if len(evs) < evSpectrumSize {
		return nil
	}
	in := make([]complex128, evSpectrumSize)
	fft.Cmplx64ToCmplx128(in, evs[:evSpectrumSize])
	out := fft.New(evSpectrumSize).Forward(make([]complex128, evSpectrumSize), in)
	// FFT-shift so bin 0 is the most-negative frequency, DC at the centre.
	half := evSpectrumSize / 2
	mag := make([]float32, evSpectrumSize)
	for i := 0; i < evSpectrumSize; i++ {
		c := out[(i+half)%evSpectrumSize]
		m := math.Hypot(real(c), imag(c)) / float64(evSpectrumSize)
		mag[i] = float32(dsp.Mag2dB(m))
	}
	return mag
}

// capSNR clamps a possibly-infinite SNR estimate (noise-free input) to a
// finite, JSON-safe value.
func capSNR(db float64) float64 {
	if math.IsInf(db, 1) || db > demodSNRCapDB {
		return demodSNRCapDB
	}
	if math.IsInf(db, -1) {
		return -demodSNRCapDB
	}
	return db
}
