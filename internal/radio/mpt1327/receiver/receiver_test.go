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
		{"missing sample rate", Options{BitSink: func([]byte, int) {}}},
		{"missing sink", Options{SampleRateHz: 48_000}},
		{"sample rate below 2x space tone", Options{SampleRateHz: 3000, BitSink: func([]byte, int) {}}},
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

// makeFMFFSKIQ synthesises an MPT-1327-style IQ stream:
//
//  1. Build an audio waveform whose instantaneous frequency steps
//     between MarkHz / SpaceHz at the 1200 baud bit rate (continuous
//     phase so the FFSK discriminator sees a clean signal).
//  2. FM-modulate that audio onto an IQ carrier at the requested
//     deviation. The receiver's FM discriminator inverts step 2
//     and hands the original audio waveform to the FFSK helper.
func makeFMFFSKIQ(bits []int) []complex64 {
	const sampleRate = 48_000.0
	const bitRate = 1200.0
	const sps = int(sampleRate / bitRate) // 40
	const fmDeviation = 4_000.0           // peak FM deviation in Hz

	audio := make([]float32, len(bits)*sps)
	audioPhase := 0.0
	for b, bit := range bits {
		toneHz := SpaceHz
		if bit == 1 {
			toneHz = MarkHz
		}
		dphi := 2 * math.Pi * toneHz / sampleRate
		for k := 0; k < sps; k++ {
			audio[b*sps+k] = float32(math.Sin(audioPhase))
			audioPhase += dphi
		}
	}

	iq := make([]complex64, len(audio))
	rfPhase := 0.0
	for i, a := range audio {
		rfPhase += 2 * math.Pi * float64(a) * fmDeviation / sampleRate
		iq[i] = complex(float32(math.Cos(rfPhase)), float32(math.Sin(rfPhase)))
	}
	return iq
}

func TestReceiverEmitsBitsFromFMFFSK(t *testing.T) {
	bits := []int{1, 0, 1, 0, 1, 1, 0, 0, 1, 0, 1, 1, 1, 0, 0, 1, 0, 1, 1, 0,
		1, 1, 0, 0, 1, 0, 1, 0, 0, 1, 1, 1, 0, 0, 1, 0}
	var batches int
	r := New(Options{
		SampleRateHz: 48_000,
		BitSink:      func(b []byte, baseIdx int) { batches++ },
	})
	iq := makeFMFFSKIQ(bits)
	chunk := 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[i:end])
	}
	if batches == 0 {
		t.Errorf("BitSink received zero batches; the chain produced no symbols")
	}
}

func TestReceiverBitSinkBaseIdxMonotonic(t *testing.T) {
	var baseIdxs []int
	var batchLens []int
	r := New(Options{
		SampleRateHz: 48_000,
		BitSink: func(b []byte, baseIdx int) {
			baseIdxs = append(baseIdxs, baseIdx)
			batchLens = append(batchLens, len(b))
		},
	})

	bits := []int{1, 0, 1, 1, 0, 0, 1, 0, 1, 1, 1, 0, 0, 1, 0, 1}
	iq := makeFMFFSKIQ(bits)
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
		SampleRateHz: 48_000,
		BitSink: func(b []byte, baseIdx int) {
			for _, v := range b {
				if v > 1 {
					bad++
				}
			}
		},
	})
	bits := []int{1, 0, 1, 0, 1, 1, 0, 0}
	r.Process(makeFMFFSKIQ(bits))
	if bad > 0 {
		t.Errorf("%d bit(s) outside 0..1 range", bad)
	}
}
