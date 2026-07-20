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

// bestBitMatch slides want over got and returns the largest number of
// bit-exact matches at any alignment — the metric for "did the FM+FFSK
// receiver recover the transmitted bits", independent of clock-recovery
// warm-up latency at the front of the stream.
func bestBitMatch(got []byte, want []int) (best, at int) {
	for off := 0; off+len(want) <= len(got); off++ {
		n := 0
		for i, w := range want {
			if int(got[off+i]) == w {
				n++
			}
		}
		if n > best {
			best, at = n, off
		}
	}
	return best, at
}

// TestReceiverRecoversTransmittedBits is the bit-correctness guard the
// older TestReceiverEmitsBitsFromFMFFSK lacked: it asserts the FM →
// FFSK-discriminator → clock-recovery → slicer chain recovers the actual
// transmitted bit values, not merely that it emits some symbols. A demod
// that produced a plausible-but-wrong 2-level stream (the failure mode
// reported in issue #927) would emit bits but never align to the payload.
func TestReceiverRecoversTransmittedBits(t *testing.T) {
	// A 1010… preamble lets the Mueller-Müller loop settle before the
	// distinctive payload, whose recovery is what the assertion checks.
	var bits []int
	for i := 0; i < 48; i++ {
		bits = append(bits, i%2)
	}
	payload := []int{1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 1, 0, 0, 0, 1, 0,
		1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 0, 1, 1, 1, 0, 1, 1, 0}
	bits = append(bits, payload...)
	// Trailing flush so the payload sits comfortably inside the recovered
	// stream rather than at its truncated edge (clock recovery can drop/add
	// a symbol at the very start/end as it warms up and drains).
	for i := 0; i < 16; i++ {
		bits = append(bits, i%2)
	}

	var got []byte
	r := New(Options{
		SampleRateHz: 48_000,
		BitSink:      func(b []byte, baseIdx int) { got = append(got, b...) },
	})
	iq := makeFMFFSKIQ(bits)
	const chunk = 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[i:end])
	}

	best, at := bestBitMatch(got, payload)
	// Allow one slip at the very edges of clock recovery, but the payload
	// must otherwise come through bit-exact.
	if best < len(payload)-1 {
		t.Errorf("recovered %d/%d payload bits at best alignment (offset %d); got=%v",
			best, len(payload), at, got)
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
