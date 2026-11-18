package main

import (
	"bufio"
	"bytes"
	"context"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/powersweep"
)

// TestRunPowerSweepsFillDC: with -fill-dc the line the removal left is
// filled in from the bins beside it, and the note says that instead of
// offering the flag. It is printed once, however many sweeps run.
func TestRunPowerSweepsFillDC(t *testing.T) {
	plan := plutoFMPlan(t)
	src := &movingOffsetSource{n: plan.FFTSize, overDB: 10, jumpFrames: 4, rng: rand.New(rand.NewSource(7))}
	clock := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	now := func() time.Time { return clock }
	sleep := func(_ context.Context, d time.Duration) error { clock = clock.Add(d); return nil }
	var out, notes bytes.Buffer
	w := bufio.NewWriter(&out)
	err := runPowerSweeps(context.Background(), src, plan,
		powersweep.Integration{Samples: 800 * plan.FFTSize, FillDC: true},
		powerSchedule{every: time.Second, stopAfter: 2500 * time.Millisecond}, w, &notes, now, sleep, nil)
	if err != nil {
		t.Fatal(err)
	}
	over := csvCentreOverMedian(t, out.String(), plan)
	if want := 3 * len(plan.Hops); len(over) != want {
		t.Fatalf("%d CSV lines, want %d (three sweeps)", len(over), want)
	}
	for i, d := range over {
		if d > 1 {
			t.Errorf("CSV line %d: the centre stands %.1f dB over the line's median with -fill-dc", i, d)
		}
	}
	if n := strings.Count(notes.String(), "power: note"); n != 1 {
		t.Errorf("%d notes, want one for the run:\n%s", n, notes.String())
	}
	for _, want := range []string{"left at the centre of 6 of 6 hops", "is filled in from those bins (-fill-dc)", "exactly on a hop's centre"} {
		if !strings.Contains(notes.String(), want) {
			t.Errorf("notes lack %q:\n%s", want, notes.String())
		}
	}
}

// TestRunPowerSweepsDCReport: -dc-report prints one line per hop with what
// the removal found, so a field run can be read without a capture: how
// much stood still, how much is left, and whether what is left is alike
// from frame to frame without turning (an offset that moved) or turns (a
// carrier off the centre).
func TestRunPowerSweepsDCReport(t *testing.T) {
	plan := plutoFMPlan(t)
	report := func(jumpFrames float64) []string {
		src := &movingOffsetSource{n: plan.FFTSize, overDB: 10, jumpFrames: jumpFrames, rng: rand.New(rand.NewSource(7))}
		var out, notes bytes.Buffer
		err := runPowerSweeps(context.Background(), src, plan, powersweep.Integration{Samples: 800 * plan.FFTSize},
			powerSchedule{every: time.Second, single: true, dcReport: true}, bufio.NewWriter(&out), &notes, time.Now,
			func(context.Context, time.Duration) error { return nil }, nil)
		if err != nil {
			t.Fatal(err)
		}
		var lines []string
		for _, l := range strings.Split(notes.String(), "\n") {
			if strings.HasPrefix(l, "power: dc ") {
				lines = append(lines, l)
			}
		}
		if len(lines) != len(plan.Hops) {
			t.Fatalf("%d report lines, want one per hop (%d):\n%s", len(lines), len(plan.Hops), notes.String())
		}
		return lines
	}

	moved := report(4)
	if !strings.HasPrefix(moved[3], "power: dc 98.8000–102.4000 MHz (centre 100.6000):") {
		t.Errorf("fourth hop's line does not name its span and centre: %s", moved[3])
	}
	for _, l := range moved {
		if !strings.Contains(l, "a line like the tuner's is left") || strings.Contains(l, "filled in") {
			t.Errorf("a hop whose offset jumped: %s", l)
		}
	}
	for _, l := range report(0) {
		if !strings.Contains(l, "stood still and was removed") || !strings.HasSuffix(l, ": clean") {
			t.Errorf("a hop with a steady offset: %s", l)
		}
	}
}

// TestPowerDCLineNoteWithAFixedGain: the note puts a line that moved down
// to an AGC and recommends a fixed gain. When the gain is already fixed
// that is wrong on both counts, so it says neither.
func TestPowerDCLineNoteWithAFixedGain(t *testing.T) {
	plan := plutoFMPlan(t)
	rows := []powersweep.Row{
		{LowHz: 88e6, HighHz: 91.6e6, StepHz: plan.BinHz},
		{LowHz: 91.6e6, HighHz: 95.2e6, StepHz: plan.BinHz, DC: powersweep.DCLine{Measured: true, Finding: powersweep.DCLineLeft, LeftDB: 6.9}},
	}
	auto := powerDCLineNote(3, rows, plan, false)
	fixed := powerDCLineNote(3, rows, plan, true)
	if !strings.Contains(auto, "sweep 3") || !strings.Contains(auto, "1 of 2 hops (93.4000 MHz; up to 6.9 dB") || !strings.Contains(auto, "a fixed -gain") {
		t.Errorf("note with automatic gain:\n%s", auto)
	}
	if strings.Contains(fixed, "AGC") || strings.Contains(fixed, "-gain") || !strings.Contains(fixed, "-fill-dc") {
		t.Errorf("note with a fixed gain:\n%s", fixed)
	}
	if note := powerDCLineNote(3, rows[:1], plan, false); note != "" {
		t.Errorf("no hop has a line left, but: %s", note)
	}
}
