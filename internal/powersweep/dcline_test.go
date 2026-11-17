package powersweep

import (
	"context"
	"math"
	"math/cmplx"
	"math/rand"
	"sort"
	"testing"
	"time"
)

// dcSource is a tunable source as a zero-IF tuner delivers it: white noise,
// the tuner's own DC offset (the same at every tuning), and optionally one
// carrier at an absolute frequency.
type dcSource struct {
	rate, noise float64
	dc          complex128
	// dcAfter, when set, replaces dc from sample stepAt of each hop on: the
	// offset moving with a gain step.
	dcAfter complex128
	stepAt  int
	// carrierHz / carrierAmp: an unmodulated carrier; fmHz makes it wander
	// that far either side, as a modulated one does.
	carrierHz, carrierAmp, fmHz float64

	center float64
	n      int // sample index within the hop
	ph     float64
	rng    *rand.Rand
}

func (s *dcSource) Tune(c uint32) error {
	s.center, s.n = float64(c), 0
	return nil
}

func (s *dcSource) Capture(_ context.Context, n int) ([]complex64, error) {
	out := make([]complex64, n)
	for i := range out {
		v := s.dc
		if s.dcAfter != 0 && s.n >= s.stepAt {
			v = s.dcAfter
		}
		if off := s.carrierHz - s.center; s.carrierAmp != 0 && math.Abs(off) < s.rate/2 {
			f := off
			if s.fmHz != 0 {
				f += s.fmHz * math.Sin(2*math.Pi*1000*float64(s.n)/s.rate)
			}
			s.ph += 2 * math.Pi * f / s.rate
			v += complex(s.carrierAmp, 0) * cmplx.Exp(complex(0, s.ph))
		}
		v += complex(s.rng.NormFloat64()*s.noise, s.rng.NormFloat64()*s.noise)
		out[i] = complex64(v)
		s.n++
	}
	return out, nil
}

// centreOverFloor returns, for one row, how far the highest of its three
// centre bins stands over the row's median bin, in dB, and the centre bin's
// own level over it.
func centreOverFloor(row Row) (peak, centre float64) {
	srt := append([]float32(nil), row.DB...)
	sort.Slice(srt, func(i, j int) bool { return srt[i] < srt[j] })
	floor := float64(srt[len(srt)/2])
	c := len(row.DB) / 2
	peak = -999
	for _, v := range row.DB[c-1 : c+2] {
		peak = math.Max(peak, float64(v)-floor)
	}
	return peak, float64(row.DB[c]) - floor
}

// dcPlan is the layout of the #1230 PlutoSky sweep: 4.8 MS/s, 512-point
// FFT, 384 bins a hop, here over three hops.
func dcPlan(t *testing.T) Plan {
	t.Helper()
	p, err := NewPlan(Range{LowHz: 88e6, HighHz: 98.8e6, BinHz: 10e3}, 4_800_000, 0.25)
	if err != nil {
		t.Fatal(err)
	}
	if p.FFTSize != 512 || p.HopBins != 384 || len(p.Hops) != 3 {
		t.Fatalf("plan %d-point FFT, %d bins a hop, %d hops; want 512 / 384 / 3", p.FFTSize, p.HopBins, len(p.Hops))
	}
	return p
}

// TestSweepRemovesTunerDCLine is the #1230 regression for the carrier that
// is not on the air. The reporter's PlutoSky sweep of the FM band showed a
// three-bin spike at the centre of every hop, 11 dB over the noise around
// it: the tuner's own DC offset, which a sweep shows once per retune. Here
// the offset is set to stand 11 dB over the noise in the centre bin.
func TestSweepRemovesTunerDCLine(t *testing.T) {
	p := dcPlan(t)
	// Noise per bin is 2·noise²; a DC offset of amplitude a reads
	// a²·(Σw)²/(n·Σw²) = a²·n/1.5 in the centre bin under a Hann window.
	const noise = 0.003
	amp := math.Sqrt(math.Pow(10, 1.1) * 2 * noise * noise * 1.5 / float64(p.FFTSize))
	newSrc := func() *dcSource {
		return &dcSource{rate: p.SampleRateHz, noise: noise, dc: cmplx.Rect(amp, 0.7), rng: rand.New(rand.NewSource(11))}
	}
	in := Integration{Samples: 400 * p.FFTSize}

	kept, err := Sweep(context.Background(), newSrc(), p, Integration{Samples: in.Samples, KeepDC: true}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	for h, row := range kept {
		if peak, _ := centreOverFloor(row); peak < 9 {
			t.Fatalf("hop %d with the DC line kept: centre %.1f dB over the floor; the fixture should show the reported 11 dB spike", h, peak)
		}
	}

	rows, err := Sweep(context.Background(), newSrc(), p, in, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	for h, row := range rows {
		peak, centre := centreOverFloor(row)
		// 400 frames put a bin's average within about ±0.5 dB of the floor.
		if peak > 1 {
			t.Errorf("hop %d: a centre bin stands %.1f dB over the floor: the tuner's DC line reads as a carrier", h, peak)
		}
		// And the noise there is still measured, not notched out.
		if centre < -1 {
			t.Errorf("hop %d: the centre bin reads %.1f dB under the floor", h, -centre)
		}
	}
}

// TestSweepDCLineRemovalFollowsAGainStep: the offset changes when the
// tuner's gain does (an AGC step in the middle of a hop; here it grows 40 %
// and turns 0.6 rad). The removal works in blocks of frames, so only the
// block the step falls in keeps a trace: about 0.4 dB here, where taking
// the line as constant over the whole hop would leave 4.6 dB.
func TestSweepDCLineRemovalFollowsAGainStep(t *testing.T) {
	p := dcPlan(t)
	const noise = 0.003
	amp := math.Sqrt(math.Pow(10, 1.1) * 2 * noise * noise * 1.5 / float64(p.FFTSize))
	src := &dcSource{
		rate: p.SampleRateHz, noise: noise, rng: rand.New(rand.NewSource(12)),
		dc: cmplx.Rect(amp, 0.7), dcAfter: cmplx.Rect(1.4*amp, 1.3), stepAt: 150*p.FFTSize + 77,
	}
	rows, err := Sweep(context.Background(), src, p, Integration{Samples: 400 * p.FFTSize}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	for h, row := range rows {
		if peak, _ := centreOverFloor(row); peak > 1.5 {
			t.Errorf("hop %d: a centre bin stands %.1f dB over the floor after the offset stepped mid-hop", h, peak)
		}
	}
}

// TestSweepKeepsSignalsAtTheHopCentre: what is removed is what stands still
// from frame to frame. A modulated carrier on a hop's centre frequency, and
// an unmodulated one a third of a bin off it, must read the same with the
// removal as without.
func TestSweepKeepsSignalsAtTheHopCentre(t *testing.T) {
	p := dcPlan(t)
	centre := float64(p.Hops[1].CenterHz)
	for _, tc := range []struct {
		name          string
		carrierHz, fm float64
	}{
		{"modulated carrier on the hop centre", centre, 3000},
		{"unmodulated carrier a third of a bin off it", centre + p.BinHz/3, 0},
	} {
		run := func(keep bool) []Row {
			src := &dcSource{
				rate: p.SampleRateHz, noise: 0.003, rng: rand.New(rand.NewSource(13)),
				carrierHz: tc.carrierHz, carrierAmp: 0.01, fmHz: tc.fm,
			}
			rows, err := Sweep(context.Background(), src, p, Integration{Samples: 400 * p.FFTSize, KeepDC: keep}, time.Time{})
			if err != nil {
				t.Fatal(err)
			}
			return rows
		}
		want, got := run(true), run(false)
		c := len(want[1].DB) / 2
		if peak, _ := centreOverFloor(want[1]); peak < 10 {
			t.Fatalf("%s: only %.1f dB over the floor even with the DC line kept; the fixture is too weak", tc.name, peak)
		}
		for k := c - 1; k <= c+1; k++ {
			if d := math.Abs(float64(got[1].DB[k] - want[1].DB[k])); d > 0.5 {
				t.Errorf("%s: bin %+d reads %.2f dBFS with the DC line removed and %.2f without", tc.name, k-c, got[1].DB[k], want[1].DB[k])
			}
		}
	}
}

// TestSweepSingleFrameHopFillsTheCentre: one frame cannot tell the DC line
// from a signal, so a hop that short has its three centre bins filled in
// from the bins either side, as rtl_power does with its DC bin.
func TestSweepSingleFrameHopFillsTheCentre(t *testing.T) {
	p := dcPlan(t)
	src := &dcSource{rate: p.SampleRateHz, noise: 0.003, dc: complex(0.02, 0), rng: rand.New(rand.NewSource(14))}
	rows, err := Sweep(context.Background(), src, p, Integration{Samples: p.FFTSize}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	for h, row := range rows {
		c := len(row.DB) / 2
		side := 10 * math.Log10((math.Pow(10, float64(row.DB[c-2])/10)+math.Pow(10, float64(row.DB[c+2])/10))/2)
		for k := c - 1; k <= c+1; k++ {
			if d := math.Abs(float64(row.DB[k]) - side); d > 0.01 {
				t.Errorf("hop %d bin %+d: %.2f dBFS, want the mean of the bins two away (%.2f)", h, k-c, row.DB[k], side)
			}
		}
	}
}
