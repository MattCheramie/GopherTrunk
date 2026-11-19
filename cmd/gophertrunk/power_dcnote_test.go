package main

import (
	"bufio"
	"bytes"
	"context"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/powersweep"
)

// movingOffsetSource is a zero-IF tuner as `power` sees it: white noise and
// the tuner's own DC offset, overDB over the noise in a bin of an n-point
// FFT. With jumpFrames > 0 the offset jumps to a new, unrelated value that
// often (in FFT frames, on average), as it does when an AGC steps the gain;
// with 0 it holds still.
type movingOffsetSource struct {
	n          int
	overDB     float64
	jumpFrames float64
	cur        complex128
	rng        *rand.Rand
}

const movingOffsetNoise = 0.003

func (s *movingOffsetSource) Tune(uint32) error { s.cur = 0; return nil }

func (s *movingOffsetSource) Capture(_ context.Context, n int) ([]complex64, error) {
	// An offset of amplitude a reads a²·n/1.5 in the centre bin of a Hann
	// FFT, against 2·noise² of noise.
	amp := math.Sqrt(math.Pow(10, s.overDB/10) * 2 * movingOffsetNoise * movingOffsetNoise * 1.5 / float64(s.n))
	out := make([]complex64, n)
	for i := range out {
		switch {
		case s.jumpFrames > 0 && (s.cur == 0 || s.rng.Float64() < 1/(s.jumpFrames*float64(s.n))):
			s.cur = complex(s.rng.NormFloat64(), s.rng.NormFloat64()) * complex(amp/math.Sqrt2, 0)
		case s.cur == 0:
			s.cur = complex(amp*0.8, amp*0.6)
		}
		out[i] = complex64(s.cur + complex(s.rng.NormFloat64()*movingOffsetNoise, s.rng.NormFloat64()*movingOffsetNoise))
	}
	return out, nil
}

// plutoFMPlan is the #1230 PlutoSky sweep: 88–108 MHz in 10 kHz bins at
// 4.8 MS/s, six hops of a 512-point FFT.
func plutoFMPlan(t *testing.T) powersweep.Plan {
	t.Helper()
	r, err := powersweep.ParseRange("88M:108M:10k")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := powersweep.NewPlan(r, 4_800_000, 0.25)
	if err != nil {
		t.Fatal(err)
	}
	if plan.FFTSize != 512 || len(plan.Hops) != 6 {
		t.Fatalf("plan: %d-point FFT, %d hops; want 512 / 6", plan.FFTSize, len(plan.Hops))
	}
	return plan
}

// csvCentreOverMedian returns, for each CSV line of one sweep, how far the
// value at the hop's centre stands over the line's median value, in dB.
func csvCentreOverMedian(t *testing.T, csv string, plan powersweep.Plan) []float64 {
	t.Helper()
	var out []float64
	for _, line := range strings.Split(strings.TrimSpace(csv), "\n") {
		f := strings.Split(line, ", ")
		var v []float64
		for _, s := range f[6:] {
			x, err := strconv.ParseFloat(s, 64)
			if err != nil {
				t.Fatalf("CSV value %q: %v", s, err)
			}
			v = append(v, x)
		}
		centre := v[plan.HopBins/2]
		sort.Float64s(v)
		out = append(out, centre-v[len(v)/2])
	}
	return out
}

// TestRunPowerSweepsSaysWhenTheDCLineIsLeft is the #1230 retest on v1.4.1.
// The reporter's PlutoSky sweep still had the tuner's DC line in the busy
// hops, 7 dB over the bins beside it, and `power` printed nothing about it:
// the line looked like a station and the run looked like a clean one.
//
// The removal takes out what stands still, so it leaves a line whose
// offset moves during the hop. `power` now says so once, names the hops,
// and names the two remedies. A steady offset is removed and nothing is
// said.
func TestRunPowerSweepsSaysWhenTheDCLineIsLeft(t *testing.T) {
	plan := plutoFMPlan(t)
	run := func(jumpFrames float64) (csv, notes string) {
		src := &movingOffsetSource{n: plan.FFTSize, overDB: 10, jumpFrames: jumpFrames, rng: rand.New(rand.NewSource(7))}
		var out, nb bytes.Buffer
		w := bufio.NewWriter(&out)
		err := runPowerSweeps(context.Background(), src, plan, powersweep.Integration{Samples: 800 * plan.FFTSize},
			powerSchedule{every: time.Second, single: true}, w, &nb, time.Now,
			func(context.Context, time.Duration) error { return nil }, nil)
		if err != nil {
			t.Fatal(err)
		}
		return out.String(), nb.String()
	}

	csv, notes := run(4)
	for h, over := range csvCentreOverMedian(t, csv, plan) {
		if over < 5 {
			t.Fatalf("hop %d: the centre stands %.1f dB over the line's median; the fixture should leave the reported ~7 dB line", h, over)
		}
	}
	for _, want := range []string{
		"still at the centre of 6 of 6 hops",
		"89.8000, 93.4000, 97.0000, 100.6000, …",
		"a fixed -gain",
		"-fill-dc",
		"-dc-report",
	} {
		if !strings.Contains(notes, want) {
			t.Errorf("notes lack %q:\n%s", want, notes)
		}
	}

	csv, notes = run(0)
	for h, over := range csvCentreOverMedian(t, csv, plan) {
		if over > 1 {
			t.Errorf("hop %d with a steady offset: the centre stands %.1f dB over the line's median", h, over)
		}
	}
	if notes != "" {
		t.Errorf("a steady offset is removed outright; nothing should be said:\n%s", notes)
	}
}
