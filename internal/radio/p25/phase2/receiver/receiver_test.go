package receiver

import (
	"math"
	"testing"
)

func TestReceiverConstructsAndProcessesSilence(t *testing.T) {
	r := New(Options{
		SampleRateHz: 48_000,
		DibitSink:    func(dibits []uint8, baseIdx int) {},
	})
	silence := make([]complex64, 4800)
	for range 4 {
		r.Process(silence)
	}
}

func TestReceiverConstructorPanicsOnBadParams(t *testing.T) {
	cases := []struct {
		name string
		opts Options
	}{
		{"missing sample rate", Options{DibitSink: func([]uint8, int) {}}},
		{"missing sink", Options{SampleRateHz: 48_000}},
		{"sample rate below 2x symbol rate", Options{SampleRateHz: 10_000, DibitSink: func([]uint8, int) {}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic, got nil")
				}
			}()
			_ = New(tc.opts)
		})
	}
}

// makeP2HDQPSKIQ synthesises a P25 Phase 2 H-DQPSK IQ stream by
// walking the phase per-dibit through the standard encoding map
// (with the π/8 rotation offset baked in) and emitting `sps`
// constant complex samples per symbol. The matched filter rounds
// the symbol edges; the naive decimation in the receiver picks
// the centre sample of each symbol.
func makeP2HDQPSKIQ(dibits []uint8) ([]complex64, int) {
	const sampleRate = 48_000.0
	const sps = int(sampleRate / SymbolRate) // 8

	// Encoding map for π/8-rotated H-DQPSK (matches the PiOver4DQPSK
	// helper's decoder when rotation = π/8 is configured): each
	// dibit contributes a phase delta which when reduced by π/8
	// lands in one of the four quadrants.
	deltas := map[uint8]float64{
		0b00: math.Pi/8 + 0,
		0b01: math.Pi/8 + math.Pi/2,
		0b10: math.Pi/8 + math.Pi,
		0b11: math.Pi/8 - math.Pi/2,
	}

	iq := make([]complex64, len(dibits)*sps)
	phase := 0.0
	for d, dibit := range dibits {
		phase += deltas[dibit]
		c := complex(float32(math.Cos(phase)), float32(math.Sin(phase)))
		for k := 0; k < sps; k++ {
			iq[d*sps+k] = c
		}
	}
	return iq, sps
}

func TestReceiverEmitsDibitsFromHDQPSK(t *testing.T) {
	dibits := []uint8{
		0b00, 0b01, 0b10, 0b11, 0b00, 0b01, 0b10, 0b11,
		0b11, 0b10, 0b01, 0b00, 0b11, 0b10, 0b01, 0b00,
	}
	var batches int
	r := New(Options{
		SampleRateHz: 48_000,
		DibitSink:    func(d []uint8, baseIdx int) { batches++ },
	})
	iq, _ := makeP2HDQPSKIQ(dibits)
	chunk := 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[i:end])
	}
	if batches == 0 {
		t.Errorf("DibitSink received zero batches; the chain produced no symbols")
	}
}

func TestReceiverDibitSinkBaseIdxMonotonic(t *testing.T) {
	var baseIdxs []int
	var batchLens []int
	r := New(Options{
		SampleRateHz: 48_000,
		DibitSink: func(d []uint8, baseIdx int) {
			baseIdxs = append(baseIdxs, baseIdx)
			batchLens = append(batchLens, len(d))
		},
	})

	dibits := []uint8{0b00, 0b01, 0b10, 0b11, 0b00, 0b01, 0b10, 0b11,
		0b00, 0b01, 0b10, 0b11, 0b00, 0b01, 0b10, 0b11}
	iq, _ := makeP2HDQPSKIQ(dibits)
	chunk := 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[i:end])
	}

	if len(baseIdxs) == 0 {
		t.Fatalf("expected DibitSink to receive at least one batch")
	}
	if baseIdxs[0] != 0 {
		t.Errorf("first baseIdx = %d, want 0", baseIdxs[0])
	}
	cumulative := 0
	for i := range baseIdxs {
		if baseIdxs[i] != cumulative {
			t.Errorf("baseIdx[%d]=%d, want %d", i, baseIdxs[i], cumulative)
		}
		cumulative += batchLens[i]
	}

	r.Reset()
	baseIdxs = baseIdxs[:0]
	batchLens = batchLens[:0]
	r.Process(iq)
	if len(baseIdxs) == 0 {
		t.Fatalf("post-Reset: expected DibitSink to receive at least one batch")
	}
	if baseIdxs[0] != 0 {
		t.Errorf("post-Reset: first baseIdx = %d, want 0", baseIdxs[0])
	}
}

func TestReceiverEmittedDibitsAreValid(t *testing.T) {
	var bad int
	r := New(Options{
		SampleRateHz: 48_000,
		DibitSink: func(d []uint8, baseIdx int) {
			for _, v := range d {
				if v > 3 {
					bad++
				}
			}
		},
	})
	dibits := []uint8{0b00, 0b01, 0b10, 0b11, 0b00, 0b01, 0b10, 0b11}
	iq, _ := makeP2HDQPSKIQ(dibits)
	r.Process(iq)
	if bad > 0 {
		t.Errorf("%d dibit(s) outside 0..3 range", bad)
	}
}

// TestParseClockMode covers the config-string → ClockMode mapping
// the ccdecoder connector uses to translate the
// `p25_phase2_clock_mode` YAML field into Options.ClockMode.
func TestParseClockMode(t *testing.T) {
	cases := []struct {
		in   string
		want ClockMode
		ok   bool
	}{
		{"", ClockGardner, true},
		{"gardner", ClockGardner, true},
		{"Gardner", ClockGardner, true},
		{"on", ClockGardner, true},
		{"true", ClockGardner, true},
		{"1", ClockGardner, true},
		{" gardner ", ClockGardner, true},
		{"naive", ClockNaive, true},
		{"NAIVE", ClockNaive, true},
		{"off", ClockNaive, true},
		{"false", ClockNaive, true},
		{"0", ClockNaive, true},
		{"nonsense", ClockGardner, false},
	}
	for _, tc := range cases {
		got, ok := ParseClockMode(tc.in)
		if got != tc.want || ok != tc.ok {
			t.Errorf("ParseClockMode(%q) = (%v, %v), want (%v, %v)",
				tc.in, got, ok, tc.want, tc.ok)
		}
	}
}
