package metrics

import (
	"math"
	"testing"
)

func TestCollectorRates(t *testing.T) {
	var c Collector
	c.AddBitErrors(2, 64) // NID-like unit
	c.AddBitErrors(1, 64) // another
	c.AddSymbolErrors(3, 196)
	c.AddUnit(true)
	c.AddUnit(true)
	c.AddUnit(false) // one uncorrectable

	s := c.Summarize()
	approx(t, s.PreFECBitErrorRate, 3.0/128.0, 1e-12, "pre-FEC BER")
	approx(t, s.PreFECSymbolErrorRate, 3.0/196.0, 1e-12, "pre-FEC SER")
	approx(t, s.UncorrectableRate, 1.0/3.0, 1e-12, "uncorrectable rate")
	if s.Units != 3 {
		t.Errorf("units: got %d want 3", s.Units)
	}
}

func TestCollectorEmptyRatesAreNaN(t *testing.T) {
	var c Collector
	s := c.Summarize()
	if !math.IsNaN(s.PreFECBitErrorRate) || !math.IsNaN(s.PreFECSymbolErrorRate) || !math.IsNaN(s.UncorrectableRate) {
		t.Error("zero-denominator rates should be NaN")
	}
}

func TestCollectorSyncMargin(t *testing.T) {
	var c Collector
	for _, m := range []int{5, 3, 4, 1, 2} {
		c.AddSyncMargin(m)
	}
	s := c.Summarize()
	if s.SyncHits != 5 {
		t.Errorf("hits: got %d want 5", s.SyncHits)
	}
	if s.SyncMarginMin != 1 {
		t.Errorf("min: got %d want 1", s.SyncMarginMin)
	}
	if s.SyncMarginMedian != 3 { // sorted {1,2,3,4,5}, index 2
		t.Errorf("median: got %d want 3", s.SyncMarginMedian)
	}
	approx(t, s.SyncMarginMean, 3.0, 1e-12, "mean margin")
}
