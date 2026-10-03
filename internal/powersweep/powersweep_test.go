package powersweep

import (
	"bytes"
	"context"
	"math"
	"math/cmplx"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestParseRange(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Range
	}{
		{"88M:108M:10k", Range{88e6, 108e6, 10e3}},
		{"144m:148m:1K", Range{144e6, 148e6, 1e3}},
		{"1.2G:1.3G:125k", Range{1.2e9, 1.3e9, 125e3}},
		{"100000000:101000000:5000", Range{100e6, 101e6, 5e3}},
	} {
		got, err := ParseRange(tc.in)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if math.Abs(got.LowHz-tc.want.LowHz) > 1e-3 || math.Abs(got.HighHz-tc.want.HighHz) > 1e-3 || math.Abs(got.BinHz-tc.want.BinHz) > 1e-6 {
			t.Errorf("%q = %+v, want %+v", tc.in, got, tc.want)
		}
	}
	for _, bad := range []string{"", "88M:108M", "108M:88M:10k", "88M:108M:0", "x:108M:10k", "88M:5G:10k"} {
		if _, err := ParseRange(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

// TestNewPlanTilesTheRange: bins are no wider than asked, hops abut exactly
// (no gap, no overlap) and the last hop is trimmed to the range.
func TestNewPlanTilesTheRange(t *testing.T) {
	r := Range{LowHz: 88e6, HighHz: 108e6, BinHz: 10e3}
	p, err := NewPlan(r, 2_400_000, 0.25)
	if err != nil {
		t.Fatal(err)
	}
	if p.FFTSize != 256 || p.BinHz > r.BinHz {
		t.Fatalf("fft %d bin %.1f Hz, want 256-point bins ≤ 10 kHz", p.FFTSize, p.BinHz)
	}
	if p.HopBins != 192 || p.FirstBin != 32 {
		t.Fatalf("hop bins %d first %d, want 192 centred bins from 32", p.HopBins, p.FirstBin)
	}
	total := 0
	next := r.LowHz
	for i, h := range p.Hops {
		if math.Abs(h.LowHz-next) > 1e-6 {
			t.Fatalf("hop %d starts at %.1f, want %.1f (contiguous)", i, h.LowHz, next)
		}
		// Bin 0 sits at the centre minus half a hop.
		if d := float64(h.CenterHz) - (h.LowHz + float64(p.HopBins/2)*p.BinHz); math.Abs(d) > 0.5 {
			t.Fatalf("hop %d centre %d off by %.2f Hz", i, h.CenterHz, d)
		}
		next = h.LowHz + float64(h.Bins)*p.BinHz
		total += h.Bins
	}
	if next < r.HighHz || next-r.HighHz >= p.BinHz {
		t.Fatalf("sweep ends at %.1f, want within one bin above %.1f", next, r.HighHz)
	}
	if want := int(math.Ceil((r.HighHz - r.LowHz) / p.BinHz)); total != want {
		t.Fatalf("%d bins, want %d", total, want)
	}
}

func TestNewPlanRejects(t *testing.T) {
	r := Range{LowHz: 100e6, HighHz: 101e6, BinHz: 1e3}
	if _, err := NewPlan(r, 2_400_000, 1); err == nil {
		t.Error("crop 1 accepted")
	}
	if _, err := NewPlan(Range{LowHz: 100e6, HighHz: 101e6, BinHz: 1}, 2_400_000, 0.25); err == nil {
		t.Error("1 Hz bins at 2.4 MS/s accepted")
	}
}

// toneSource is a tunable source with one carrier at an absolute frequency:
// tuned to c, it delivers exp(j2π(f−c)t) (when f is inside the tuned band)
// plus white noise, phase-continuous across calls, as a real receiver does.
type toneSource struct {
	rate, toneHz, amp float64
	noise             float64
	center            float64
	n                 int
	rng               *rand.Rand
	tunes             []uint32
}

func (s *toneSource) Tune(c uint32) error {
	s.center = float64(c)
	s.tunes = append(s.tunes, c)
	return nil
}

func (s *toneSource) Capture(_ context.Context, n int) ([]complex64, error) {
	out := make([]complex64, n)
	for i := range out {
		t := float64(s.n) / s.rate
		var v complex128
		// An ideal anti-alias filter, as a tuner has: a carrier outside
		// ±rate/2 of the tuned centre does not appear (it would alias).
		if off := s.toneHz - s.center; math.Abs(off) < s.rate/2 {
			v = complex(s.amp, 0) * cmplx.Exp(complex(0, 2*math.Pi*off*t))
		}
		v += complex(s.rng.NormFloat64()*s.noise, s.rng.NormFloat64()*s.noise)
		out[i] = complex64(v)
		s.n++
	}
	return out, nil
}

// TestSweepFindsToneAtItsFrequency runs a whole sweep over a source with one
// carrier and checks the peak lands in the bin at the carrier's frequency —
// row LowHz + index·StepHz, the mapping rtl_power's heatmap tools use — at
// the carrier's power, with the rest of the range at the noise floor.
func TestSweepFindsToneAtItsFrequency(t *testing.T) {
	const rate = 2_400_000.0
	r := Range{LowHz: 100e6, HighHz: 106e6, BinHz: 10e3}
	p, err := NewPlan(r, rate, 0.25)
	if err != nil {
		t.Fatal(err)
	}
	const toneHz = 103_337_000.0
	src := &toneSource{rate: rate, toneHz: toneHz, amp: 0.1, noise: 1e-4, rng: rand.New(rand.NewSource(1))}
	stamp := time.Date(2026, 10, 3, 21, 30, 5, 0, time.UTC)
	rows, err := Sweep(context.Background(), src, p, 20*p.FFTSize, stamp)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != len(p.Hops) || len(src.tunes) != len(p.Hops) {
		t.Fatalf("%d rows / %d tunes, want %d", len(rows), len(src.tunes), len(p.Hops))
	}
	bestDB, bestHz := float32(-999), 0.0
	for _, row := range rows {
		if row.Samples != 20*p.FFTSize {
			t.Fatalf("row samples %d, want %d", row.Samples, 20*p.FFTSize)
		}
		if !row.Time.Equal(stamp) {
			t.Fatalf("row time %v, want the sweep stamp", row.Time)
		}
		if got := int(math.Round((row.HighHz - row.LowHz) / row.StepHz)); got != len(row.DB) {
			t.Fatalf("row spans %d bins but carries %d values", got, len(row.DB))
		}
		for j, v := range row.DB {
			if v > bestDB {
				bestDB, bestHz = v, row.LowHz+float64(j)*row.StepHz
			}
		}
	}
	if math.Abs(bestHz-toneHz) > p.BinHz {
		t.Fatalf("peak at %.0f Hz, want within one bin (%.0f Hz) of %.0f", bestHz, p.BinHz, toneHz)
	}
	// A 0.1-amplitude tone is −20 dBFS of power. Under this normalisation
	// (spectrum.PowerDB's |X|²/(n·Σw²)) a Hann window reads a bin-centred
	// tone 10·log10(3/2) = 1.76 dB low, and a tone half-way between bins a
	// further 1.42 dB low: −21.8..−23.2 dBFS.
	if bestDB < -23.3 || bestDB > -21.7 {
		t.Fatalf("peak %.1f dBFS, want −21.8..−23.2", bestDB)
	}
	// Far from the tone every bin is near the noise floor
	// (2·1e-8 per sample ⇒ about −77 dBFS per bin at this size).
	for _, row := range rows {
		for j, v := range row.DB {
			f := row.LowHz + float64(j)*row.StepHz
			if math.Abs(f-toneHz) > 50e3 && v > -60 {
				t.Fatalf("bin at %.0f Hz reads %.1f dBFS, want the noise floor", f, v)
			}
		}
	}
}

// TestWriteCSVMatchesRTLPowerLayout pins the line format rtl_power writes
// (rtl_power.c csv_dbm): "date, time, Hz low, Hz high, Hz step, samples,
// dB, ..." with ", " separators and two decimals on step and dB.
func TestWriteCSVMatchesRTLPowerLayout(t *testing.T) {
	loc := time.FixedZone("x", 0)
	rows := []Row{{
		Time:    time.Date(2026, 10, 3, 9, 4, 5, 0, loc),
		LowHz:   88_000_000,
		HighHz:  88_028_125,
		StepHz:  9375,
		Samples: 2560,
		DB:      []float32{-40.123, -41, -7.5},
	}}
	var b bytes.Buffer
	if err := WriteCSV(&b, rows); err != nil {
		t.Fatal(err)
	}
	want := "2026-10-03, 09:04:05, 88000000, 88028125, 9375.00, 2560, -40.12, -41.00, -7.50\n"
	if b.String() != want {
		t.Fatalf("got  %q\nwant %q", b.String(), want)
	}
	// heatmap.py reads the bins as frange(low, high, step).
	f := strings.Split(strings.TrimSpace(b.String()), ", ")
	low, _ := strconv.ParseFloat(f[2], 64)
	high, _ := strconv.ParseFloat(f[3], 64)
	step, _ := strconv.ParseFloat(f[4], 64)
	if n := int(math.Round((high - low) / step)); n != len(f)-6 {
		t.Fatalf("header spans %d bins, line carries %d", n, len(f)-6)
	}
}
