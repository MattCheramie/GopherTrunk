package metrics

import (
	"math"
	"sort"
)

// Collector aggregates the pre-FEC error budget and sync-correlation margins
// observed over a decode run into a Summary. It is the bridge between the
// decoders' corrected-error counts (ParseNID, ParseLinkControl, DecodeTrellis,
// imbe.DecodeChannelToFrame all return "how many symbols/bits the FEC had to
// correct") and the SER/BER numbers a benchmark reports. The zero value is
// ready to use; it is not safe for concurrent use.
type Collector struct {
	correctedSymbols int64 // FEC-corrected symbol errors across all units
	totalSymbols     int64 // symbol budget those corrections were measured over
	correctedBits    int64 // FEC-corrected bit errors across all units
	totalBits        int64 // bit budget
	uncorrectable    int64 // units the FEC could not resolve (CRC/parity still bad)
	units            int64 // total units seen (for the uncorrectable rate)
	margins          []int // per-FSW-hit correlation margin (tolerance+1 − bestMismatch)
}

// AddSymbolErrors records corrected symbol errors against a symbol budget
// (e.g. a Trellis-decoded TSBK: corrected = the path metric, total = the coded
// symbol count). Use for the symbol-domain pre-FEC SER.
func (c *Collector) AddSymbolErrors(corrected, total int) {
	if corrected < 0 {
		corrected = 0
	}
	c.correctedSymbols += int64(corrected)
	c.totalSymbols += int64(total)
}

// AddBitErrors records corrected bit errors against a bit budget (e.g. a
// NID/BCH or Link-Control/RS unit: corrected = the decoder's error return,
// total = the coded bit count). Use for the bit-domain pre-FEC BER.
func (c *Collector) AddBitErrors(corrected, total int) {
	if corrected < 0 {
		corrected = 0
	}
	c.correctedBits += int64(corrected)
	c.totalBits += int64(total)
}

// AddUnit records one decoded unit and whether the FEC failed to resolve it
// (ok == false → uncorrectable). It drives the uncorrectable rate, which
// distinguishes "many small corrections" (marginal but recoverable) from
// "decode falling off a cliff."
func (c *Collector) AddUnit(ok bool) {
	c.units++
	if !ok {
		c.uncorrectable++
	}
}

// AddSyncMargin records the correlation margin of one FSW hit: the number of
// symbols by which the match cleared tolerance, margin = tolerance+1 −
// bestMismatch (a perfect 24/24 hit at tolerance 4 gives margin 5). Higher is
// healthier; a distribution pressed against 1 means sync is barely holding.
func (c *Collector) AddSyncMargin(margin int) {
	c.margins = append(c.margins, margin)
}

// Summary is the aggregated pre-FEC error and sync-margin report.
type Summary struct {
	// PreFECSymbolErrorRate is correctedSymbols / totalSymbols (NaN when no
	// symbols were budgeted).
	PreFECSymbolErrorRate float64 `json:"pre_fec_ser" yaml:"pre_fec_ser"`
	// PreFECBitErrorRate is correctedBits / totalBits (NaN when no bits were
	// budgeted).
	PreFECBitErrorRate float64 `json:"pre_fec_ber" yaml:"pre_fec_ber"`
	// UncorrectableRate is uncorrectable / units (NaN when no units seen).
	UncorrectableRate float64 `json:"uncorrectable_rate" yaml:"uncorrectable_rate"`
	// Units is the total decoded-unit count behind UncorrectableRate.
	Units int64 `json:"units" yaml:"units"`
	// SyncMarginMin / Mean / P50 summarize the FSW correlation-margin
	// distribution; zero-valued when no FSW hits were recorded.
	SyncMarginMin    int     `json:"sync_margin_min" yaml:"sync_margin_min"`
	SyncMarginMean   float64 `json:"sync_margin_mean" yaml:"sync_margin_mean"`
	SyncMarginMedian int     `json:"sync_margin_median" yaml:"sync_margin_median"`
	SyncHits         int     `json:"sync_hits" yaml:"sync_hits"`
}

// Summarize computes the Summary. Rates with a zero denominator are reported
// as NaN so a consumer can tell "measured zero" from "nothing measured."
func (c *Collector) Summarize() Summary {
	s := Summary{
		PreFECSymbolErrorRate: ratioOrNaN(c.correctedSymbols, c.totalSymbols),
		PreFECBitErrorRate:    ratioOrNaN(c.correctedBits, c.totalBits),
		UncorrectableRate:     ratioOrNaN(c.uncorrectable, c.units),
		Units:                 c.units,
	}
	if n := len(c.margins); n > 0 {
		sorted := append([]int(nil), c.margins...)
		sort.Ints(sorted)
		s.SyncHits = n
		s.SyncMarginMin = sorted[0]
		s.SyncMarginMedian = sorted[n/2]
		var sum int
		for _, m := range sorted {
			sum += m
		}
		s.SyncMarginMean = float64(sum) / float64(n)
	}
	return s
}

func ratioOrNaN(num, den int64) float64 {
	if den <= 0 {
		return math.NaN()
	}
	return float64(num) / float64(den)
}
