package siglab

import "github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr"

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

	// DecodeErrorRate is decode-error events per recovered 1000 symbols — a
	// protocol-neutral proxy for FEC stress.
	DecodeErrorRate float64 `json:"decode_error_rate_per_ksym" yaml:"decode_error_rate_per_ksym"`
}

// analyzer accumulates the observations behind a SignalQuality. It is fed
// from the engine's SymbolTap (symbols) and read loop (raw IQ), so it works
// for every protocol the factory map drives.
type analyzer struct {
	cardinality int
	hist        []int64
	symbols     int64
	iqStats     rtlsdr.IQImbalanceStats
	iqObserved  bool

	// bufferSymbols retains the full recovered-symbol stream for the
	// protocol-specific deep dive (P25 P1's FSW/NID landscape). Off by
	// default since it is O(symbols) memory.
	bufferSymbols bool
	symBuf        []uint8
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
}

// result builds the SignalQuality from the accumulated observations.
// decodeErrors / symbols give the per-ksym error rate.
func (a *analyzer) result(decodeErrors int64) *SignalQuality {
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
	return sq
}
