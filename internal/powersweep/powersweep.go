// Package powersweep is an rtl_power-style wideband power logger (issue
// #1230): it steps a tunable IQ source across a frequency range, averages an
// FFT power spectrum at each step, and writes the result in rtl_power's CSV
// layout so existing tooling (heatmap.py, flatten.py, spreadsheet imports)
// reads it unchanged.
//
// The sweep is laid out in whole FFT bins. One "hop" is one tuning: it keeps
// the HopBins bins around the centre of the FFT (the rest — the 1−crop edges,
// where the tuner's anti-alias filter rolls off — are dropped), and
// consecutive hops abut exactly, so bin j of hop h sits at
// Low + (h·HopBins + j)·BinHz with no gap or overlap across the whole range.
package powersweep

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/fft"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/window"
)

// MaxFFTSize bounds the FFT a bin size can ask for. At 2.4 MS/s it allows
// ~9 Hz bins; finer than that is not a power survey.
const MaxFFTSize = 1 << 18

// MinFFTSize is the smallest FFT a plan uses, however wide the bins.
const MinFFTSize = 8

// Range is a sweep request: [LowHz, HighHz] in steps of at most BinHz.
type Range struct {
	LowHz, HighHz, BinHz float64
}

// ParseRange parses rtl_power's -f syntax, "lower:upper:bin_size", where each
// field takes an optional k / M / G suffix (e.g. "88M:108M:10k").
func ParseRange(s string) (Range, error) {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) != 3 {
		return Range{}, fmt.Errorf("frequency range %q: want lower:upper:bin_size (e.g. 88M:108M:10k)", s)
	}
	var v [3]float64
	for i, p := range parts {
		f, err := ParseHz(p)
		if err != nil {
			return Range{}, fmt.Errorf("frequency range %q: %w", s, err)
		}
		v[i] = f
	}
	r := Range{LowHz: v[0], HighHz: v[1], BinHz: v[2]}
	if r.LowHz <= 0 || r.HighHz <= r.LowHz {
		return Range{}, fmt.Errorf("frequency range %q: need 0 < lower < upper", s)
	}
	if r.BinHz <= 0 {
		return Range{}, fmt.Errorf("frequency range %q: bin size must be > 0", s)
	}
	if r.HighHz > math.MaxUint32 {
		return Range{}, fmt.Errorf("frequency range %q: upper edge above %d Hz", s, uint32(math.MaxUint32))
	}
	return r, nil
}

// ParseHz parses a frequency with an optional k / M / G suffix (case-
// insensitive, as rtl_power's atofs).
func ParseHz(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty frequency")
	}
	mult := 1.0
	switch s[len(s)-1] {
	case 'k', 'K':
		mult = 1e3
	case 'm', 'M':
		mult = 1e6
	case 'g', 'G':
		mult = 1e9
	}
	if mult != 1 {
		s = s[:len(s)-1]
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, fmt.Errorf("bad frequency %q", s)
	}
	return f * mult, nil
}

// Hop is one tuning of a sweep.
type Hop struct {
	// CenterHz is the frequency the source is tuned to.
	CenterHz uint32
	// LowHz is the frequency of the hop's first kept bin.
	LowHz float64
	// Bins is how many bins the hop contributes (HopBins, except possibly
	// fewer on the last hop).
	Bins int
}

// Plan is a laid-out sweep.
type Plan struct {
	SampleRateHz float64
	FFTSize      int
	BinHz        float64
	// HopBins is the number of bins every full hop keeps, centred in the
	// FFT: FFT indices [FirstBin, FirstBin+HopBins).
	HopBins  int
	FirstBin int
	Hops     []Hop
}

// NewPlan lays a Range out over a source running at sampleRateHz. crop is the
// fraction of each FFT discarded at the band edges (0 ≤ crop < 1), split
// evenly between them. The FFT size is the smallest power of two whose bins
// are no wider than r.BinHz.
func NewPlan(r Range, sampleRateHz, crop float64) (Plan, error) {
	if sampleRateHz <= 0 {
		return Plan{}, errors.New("sample rate must be > 0")
	}
	if crop < 0 || crop >= 1 {
		return Plan{}, fmt.Errorf("crop %.2f must be in [0, 1)", crop)
	}
	n := MinFFTSize
	for sampleRateHz/float64(n) > r.BinHz {
		n <<= 1
		if n > MaxFFTSize {
			return Plan{}, fmt.Errorf("bin size %.1f Hz needs an FFT over %d points at %.0f S/s; use a wider bin", r.BinHz, MaxFFTSize, sampleRateHz)
		}
	}
	binHz := sampleRateHz / float64(n)
	hopBins := int(float64(n) * (1 - crop))
	hopBins &^= 1 // even, so the hop centres on a bin boundary of the FFT
	if hopBins < 2 {
		return Plan{}, fmt.Errorf("crop %.2f leaves no usable bins in a %d-point FFT", crop, n)
	}
	count := int(math.Ceil((r.HighHz - r.LowHz) / binHz))
	if count < 1 {
		count = 1
	}
	p := Plan{
		SampleRateHz: sampleRateHz,
		FFTSize:      n,
		BinHz:        binHz,
		HopBins:      hopBins,
		FirstBin:     n/2 - hopBins/2,
	}
	for done := 0; done < count; done += hopBins {
		low := r.LowHz + float64(done)*binHz
		// Bin j of the hop sits at center + (FirstBin + j − n/2)·binHz =
		// center + (j − hopBins/2)·binHz, so bin 0 is at low.
		center := low + float64(hopBins/2)*binHz
		if center > math.MaxUint32 {
			return Plan{}, fmt.Errorf("hop centre %.0f Hz out of range", center)
		}
		p.Hops = append(p.Hops, Hop{
			CenterHz: uint32(math.Round(center)),
			LowHz:    low,
			Bins:     min(hopBins, count-done),
		})
	}
	return p, nil
}

// Source is the tunable IQ source a sweep drives. hunt.IQSource satisfies it.
type Source interface {
	// Tune retunes the source; IQ captured afterwards is centred there.
	Tune(centerHz uint32) error
	// Capture returns the next n samples at the current centre.
	Capture(ctx context.Context, n int) ([]complex64, error)
}

// Row is one hop's averaged power, in rtl_power's CSV fields.
type Row struct {
	Time    time.Time
	LowHz   float64
	HighHz  float64
	StepHz  float64
	Samples int
	DB      []float32
}

// captureBlock bounds how many samples one Capture call asks for, so a long
// integration streams through a fixed buffer instead of being held in memory.
const captureBlock = 1 << 16

// Sweep runs one pass of p over src, integrating samplesPerHop IQ samples
// (rounded up to whole FFTs, at least one) per hop. stamp is the time written
// on every row of the pass, as rtl_power does.
func Sweep(ctx context.Context, src Source, p Plan, samplesPerHop int, stamp time.Time) ([]Row, error) {
	acc := newAccumulator(p.FFTSize)
	frames := (samplesPerHop + p.FFTSize - 1) / p.FFTSize
	if frames < 1 {
		frames = 1
	}
	perBlock := max(1, captureBlock/p.FFTSize)
	rows := make([]Row, 0, len(p.Hops))
	for _, h := range p.Hops {
		if err := src.Tune(h.CenterHz); err != nil {
			return rows, fmt.Errorf("tune %d Hz: %w", h.CenterHz, err)
		}
		acc.reset()
		for acc.frames < frames {
			want := min(perBlock, frames-acc.frames) * p.FFTSize
			iq, err := src.Capture(ctx, want)
			if err != nil {
				return rows, err
			}
			if len(iq) < p.FFTSize {
				return rows, io.ErrUnexpectedEOF
			}
			acc.add(iq)
		}
		rows = append(rows, Row{
			Time:    stamp,
			LowHz:   h.LowHz,
			HighHz:  h.LowHz + float64(h.Bins)*p.BinHz,
			StepHz:  p.BinHz,
			Samples: acc.frames * p.FFTSize,
			DB:      acc.dB(p.FirstBin, h.Bins),
		})
	}
	return rows, nil
}

// accumulator averages FFT periodograms in linear power, with the same Hann
// window and normalisation as spectrum.PowerDB (a unit-amplitude tone reads
// ~0 dBFS in its bin).
type accumulator struct {
	n      int
	plan   fft.Plan
	win    []float64
	norm   float64
	in     []complex128
	out    []complex128
	sum    []float64 // FFT-shifted linear power
	frames int
}

func newAccumulator(n int) *accumulator {
	win := window.Hann(n)
	var ws float64
	for _, w := range win {
		ws += w * w
	}
	return &accumulator{
		n:    n,
		plan: fft.New(n),
		win:  win,
		norm: 1 / (float64(n) * ws),
		in:   make([]complex128, n),
		out:  make([]complex128, n),
		sum:  make([]float64, n),
	}
}

func (a *accumulator) reset() {
	clear(a.sum)
	a.frames = 0
}

// add folds every whole FFT frame of iq into the average.
func (a *accumulator) add(iq []complex64) {
	half := a.n / 2
	for off := 0; off+a.n <= len(iq); off += a.n {
		for i, s := range iq[off : off+a.n] {
			w := a.win[i]
			a.in[i] = complex(float64(real(s))*w, float64(imag(s))*w)
		}
		out := a.plan.Forward(a.out, a.in)
		for i, v := range out {
			re, im := real(v), imag(v)
			a.sum[(i+half)%a.n] += (re*re + im*im) * a.norm
		}
		a.frames++
	}
}

// dB returns the averaged power of FFT-shifted bins [first, first+n) in dBFS.
func (a *accumulator) dB(first, n int) []float32 {
	out := make([]float32, n)
	for j := range out {
		p := a.sum[first+j] / float64(max(a.frames, 1))
		if p < 1e-30 {
			p = 1e-30
		}
		out[j] = float32(10 * math.Log10(p))
	}
	return out
}

// WriteCSV writes rows in rtl_power's layout, one line per row:
//
//	date, time, Hz low, Hz high, Hz step, samples, dB, dB, ...
//
// Date and time are the row's local time ("2006-01-02", "15:04:05").
func WriteCSV(w io.Writer, rows []Row) error {
	var b strings.Builder
	for _, r := range rows {
		b.Reset()
		fmt.Fprintf(&b, "%s, %s, %d, %d, %.2f, %d",
			r.Time.Format("2006-01-02"), r.Time.Format("15:04:05"),
			int64(math.Round(r.LowHz)), int64(math.Round(r.HighHz)), r.StepHz, r.Samples)
		for _, v := range r.DB {
			fmt.Fprintf(&b, ", %.2f", v)
		}
		b.WriteByte('\n')
		if _, err := io.WriteString(w, b.String()); err != nil {
			return err
		}
	}
	return nil
}
