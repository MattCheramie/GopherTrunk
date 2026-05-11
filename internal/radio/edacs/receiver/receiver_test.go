package receiver

import (
	"math"
	"testing"
)

func TestReceiverConstructsAndProcessesSilence(t *testing.T) {
	r := New(Options{
		SampleRateHz: 48_000,
		BitSink:      func(bits []byte, baseIdx int) {},
	})
	silence := make([]complex64, 9600)
	for range 4 {
		r.Process(silence)
	}
}

func TestReceiverConstructorPanicsOnBadParams(t *testing.T) {
	cases := []struct {
		name string
		opts Options
	}{
		{"missing sample rate", Options{BitSink: func([]byte, int) {}}},
		{"missing sink", Options{SampleRateHz: 48_000}},
		{"sample rate below 2x symbol rate", Options{SampleRateHz: 16_000, BitSink: func([]byte, int) {}}},
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

// makeNRZIQ at 96 kHz / 10 sps cycles through alternating ±NRZ
// symbols modulated as FM tones around ±deviation. Mirrors the
// GFSK helper's own round-trip test fixture — the dPMR / NXDN
// versions don't apply here because EDACS is 2-level.
func makeNRZIQ(nBits int) []complex64 {
	const sampleRate = 96_000.0
	const sps = 10 // 96000 / 9600
	const deviation = 2400.0
	radPerSample := func(bit int) float64 {
		v := -1.0
		if bit == 1 {
			v = +1.0
		}
		return 2 * math.Pi * v * deviation / sampleRate
	}
	iq := make([]complex64, nBits*sps)
	phase := 0.0
	for b := 0; b < nBits; b++ {
		dphi := radPerSample(b % 2)
		base := b * sps
		for k := 0; k < sps; k++ {
			iq[base+k] = complex(float32(math.Cos(phase)), float32(math.Sin(phase)))
			phase += dphi
		}
	}
	return iq
}

func TestReceiverEmitsBitsFromNRZ(t *testing.T) {
	var batches int
	r := New(Options{
		SampleRateHz: 96_000,
		BitSink:      func(bits []byte, baseIdx int) { batches++ },
	})
	iq := makeNRZIQ(1024)
	chunk := 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[i:end])
	}
	if batches == 0 {
		t.Errorf("BitSink received zero batches; Mueller-Müller loop produced no symbols")
	}
}

func TestReceiverBitSinkBaseIdxMonotonic(t *testing.T) {
	var baseIdxs []int
	var batchLens []int
	r := New(Options{
		SampleRateHz: 96_000,
		BitSink: func(bits []byte, baseIdx int) {
			baseIdxs = append(baseIdxs, baseIdx)
			batchLens = append(batchLens, len(bits))
		},
	})

	iq := makeNRZIQ(1024)
	chunk := 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[i:end])
	}

	if len(baseIdxs) == 0 {
		t.Fatalf("expected BitSink to receive at least one batch")
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
		t.Fatalf("post-Reset: expected BitSink to receive at least one batch")
	}
	if baseIdxs[0] != 0 {
		t.Errorf("post-Reset: first baseIdx = %d, want 0", baseIdxs[0])
	}
}

func TestReceiverEmittedBitsAreBinary(t *testing.T) {
	var bad int
	r := New(Options{
		SampleRateHz: 96_000,
		BitSink: func(bits []byte, baseIdx int) {
			for _, b := range bits {
				if b > 1 {
					bad++
				}
			}
		},
	})
	r.Process(makeNRZIQ(1024))
	if bad > 0 {
		t.Errorf("%d bit(s) outside 0..1 range", bad)
	}
}
