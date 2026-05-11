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
		{"sample rate below 2x symbol rate", Options{SampleRateHz: 500, BitSink: func([]byte, int) {}}},
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

// makeLTRFMIQ synthesises an IQ stream whose FM-demodulator output
// is a sub-audible NRZ waveform at 300 baud — the underlying signal
// shape an LTR Status word rides on (Manchester encoding aside).
// Each bit period holds the audio constant at ±1; the receiver's
// LPF + clock recovery should pull the same alternation back out.
func makeLTRFMIQ(bits []int) []complex64 {
	const sampleRate = 48_000.0
	const bitRate = 300.0
	const sps = int(sampleRate / bitRate) // 160
	const fmDeviation = 2_000.0           // peak FM deviation in Hz

	audio := make([]float32, len(bits)*sps)
	for b, bit := range bits {
		v := float32(-1)
		if bit == 1 {
			v = +1
		}
		for k := 0; k < sps; k++ {
			audio[b*sps+k] = v
		}
	}

	iq := make([]complex64, len(audio))
	phase := 0.0
	for i, a := range audio {
		phase += 2 * math.Pi * float64(a) * fmDeviation / sampleRate
		iq[i] = complex(float32(math.Cos(phase)), float32(math.Sin(phase)))
	}
	return iq
}

func TestReceiverEmitsBitsFromSubAudibleNRZ(t *testing.T) {
	bits := []int{1, 0, 1, 0, 1, 1, 0, 0, 1, 0, 1, 1, 1, 0, 0, 1, 0, 1, 1, 0,
		1, 1, 0, 0, 1, 0, 1, 0, 0, 1, 1, 1, 0, 0, 1, 0}
	var batches int
	r := New(Options{
		SampleRateHz: 48_000,
		BitSink:      func(b []byte, baseIdx int) { batches++ },
	})
	iq := makeLTRFMIQ(bits)
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
	iq := makeLTRFMIQ(bits)
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
	r.Process(makeLTRFMIQ([]int{1, 0, 1, 0, 1, 1, 0, 0}))
	if bad > 0 {
		t.Errorf("%d bit(s) outside 0..1 range", bad)
	}
}
