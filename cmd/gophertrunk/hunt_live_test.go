package main

import (
	"context"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/hunt"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// TestWholeDeviceBand verifies the full-spectrum band is derived from a
// device's reported tuning range, and that a device with no range errors
// (so the CLI can fall back to an explicit -band).
func TestWholeDeviceBand(t *testing.T) {
	dev, err := (&sdr.MockDriver{Files: []string{"x.cfile"}}).Open(0)
	if err != nil {
		t.Fatalf("open mock: %v", err)
	}
	band, err := wholeDeviceBand(dev)
	if err != nil {
		t.Fatalf("wholeDeviceBand: %v", err)
	}
	// MockDevice reports the R820T family span.
	if band.LowHz != 24_000_000 || band.HighHz != 1_766_000_000 {
		t.Errorf("band = %d..%d, want 24_000_000..1_766_000_000", band.LowHz, band.HighHz)
	}

	if _, err := wholeDeviceBand(noRangeDevice{}); err == nil {
		t.Error("wholeDeviceBand on a device without FreqRange: want error, got nil")
	}
}

// TestSweepStepCountMatchesSweeper pins the whole-device ETA estimator to
// the real sweeper: the predicted step count must equal the number of
// tunes the sweeper actually performs over the same band.
func TestSweepStepCountMatchesSweeper(t *testing.T) {
	const rate = 2_400_000
	bands := []hunt.Band{
		{LowHz: 450_000_000, HighHz: 470_000_000},  // ~20 MHz
		{LowHz: 24_000_000, HighHz: 1_766_000_000}, // whole R820T span
		{LowHz: 451_000_000, HighHz: 451_500_000},  // narrower than one step
	}
	for _, band := range bands {
		src := hunt.NewFileIQSource(make([]complex64, 4096), rate) // zero IQ: only step count matters
		sw, err := hunt.NewSweeper(hunt.SweepOptions{
			Source:   src,
			Bands:    []hunt.Band{band},
			FFTSize:  1024,
			PeakOpts: hunt.PeakOptions{ThresholdDb: 10},
		})
		if err != nil {
			t.Fatalf("NewSweeper: %v", err)
		}
		if _, err := sw.Sweep(context.Background(), nil); err != nil {
			t.Fatalf("Sweep: %v", err)
		}
		want := len(src.TuneCalls())
		got := sweepStepCount(band, rate)
		if got != want {
			t.Errorf("band %d..%d: sweepStepCount = %d, sweeper took %d steps",
				band.LowHz, band.HighHz, got, want)
		}
	}
}

// noRangeDevice is an sdr.Device that deliberately does NOT implement
// sdr.FreqRanger, exercising the whole-device fallback error path.
type noRangeDevice struct{}

func (noRangeDevice) Info() sdr.Info                                       { return sdr.Info{} }
func (noRangeDevice) SetCenterFreq(uint32) error                           { return nil }
func (noRangeDevice) SetSampleRate(uint32) error                           { return nil }
func (noRangeDevice) SetGain(int) error                                    { return nil }
func (noRangeDevice) SetPPM(int) error                                     { return nil }
func (noRangeDevice) SetBiasTee(bool) error                                { return nil }
func (noRangeDevice) StreamIQ(context.Context) (<-chan []complex64, error) { return nil, nil }
func (noRangeDevice) Close() error                                         { return nil }
