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
		{"sample rate below 2x symbol rate", Options{SampleRateHz: 8_000, DibitSink: func([]uint8, int) {}}},
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

func makePhaseRampIQ(symbols int) []complex64 {
	const sampleRate = 48_000.0
	const sps = 10
	const deviation = 1800.0
	radPerSample := func(symbolValue int) float64 {
		return 2 * math.Pi * float64(symbolValue) * deviation / 3.0 / sampleRate
	}
	iq := make([]complex64, symbols*sps)
	phase := 0.0
	for s := 0; s < symbols; s++ {
		val := []int{-3, -1, +1, +3}[s%4]
		dphi := radPerSample(val)
		base := s * sps
		for k := 0; k < sps; k++ {
			iq[base+k] = complex(float32(math.Cos(phase)), float32(math.Sin(phase)))
			phase += dphi
		}
	}
	return iq
}

func TestReceiverEmitsDibitsFromPhaseRamp(t *testing.T) {
	var batches int
	r := New(Options{
		SampleRateHz: 48_000,
		DibitSink:    func(dibits []uint8, baseIdx int) { batches++ },
	})
	iq := makePhaseRampIQ(1024)
	chunk := 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[i:end])
	}
	if batches == 0 {
		t.Errorf("DibitSink received zero batches; Mueller-Müller loop produced no symbols")
	}
}

func TestReceiverDibitSinkBaseIdxMonotonic(t *testing.T) {
	var baseIdxs []int
	var batchLens []int
	r := New(Options{
		SampleRateHz: 48_000,
		DibitSink: func(dibits []uint8, baseIdx int) {
			baseIdxs = append(baseIdxs, baseIdx)
			batchLens = append(batchLens, len(dibits))
		},
	})

	iq := makePhaseRampIQ(1024)
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
		DibitSink: func(dibits []uint8, baseIdx int) {
			for _, d := range dibits {
				if d > 3 {
					bad++
				}
			}
		},
	})
	r.Process(makePhaseRampIQ(1024))
	if bad > 0 {
		t.Errorf("%d dibit(s) out of 0..3 range", bad)
	}
}

func TestSymbolToDibitMatchesP25Phase1Convention(t *testing.T) {
	cases := []struct {
		sym  int8
		want uint8
	}{
		{1, 0}, {3, 1}, {-1, 2}, {-3, 3},
	}
	for _, tc := range cases {
		if got := SymbolToDibit(tc.sym); got != tc.want {
			t.Errorf("SymbolToDibit(%d) = %d, want %d", tc.sym, got, tc.want)
		}
	}
}
