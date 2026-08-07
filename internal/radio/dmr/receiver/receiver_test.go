package receiver

import (
	"math"
	"testing"
)

// dibitToSymbol is the inverse of SymbolToDibit — used by the test
// modulator to turn a dibit stream back into C4FM symbol deviations.
func dibitToSymbol(d uint8) int {
	switch d {
	case 0:
		return +1
	case 1:
		return +3
	case 2:
		return -1
	case 3:
		return -3
	}
	return 0
}

// makeC4FMIQWithOffset modulates a dibit stream to 48 kHz / 10 sps C4FM
// IQ (unfiltered NRFM, same shape as makePhaseRampIQ) and rotates the
// whole buffer by a constant carrier offset in Hz — the exact impairment
// an uncorrected tuner ppm error imposes (issue #836).
func makeC4FMIQWithOffset(dibits []uint8, offsetHz float64) []complex64 {
	const sampleRate = 48_000.0
	const sps = 10
	const deviation = 1944.0
	iq := make([]complex64, len(dibits)*sps)
	phase := 0.0
	for i, d := range dibits {
		dphi := 2 * math.Pi * float64(dibitToSymbol(d)) * deviation / 3.0 / sampleRate
		base := i * sps
		for k := 0; k < sps; k++ {
			iq[base+k] = complex(float32(math.Cos(phase)), float32(math.Sin(phase)))
			phase += dphi
		}
	}
	if offsetHz != 0 {
		for n := range iq {
			th := 2 * math.Pi * offsetHz * float64(n) / sampleRate
			iq[n] *= complex(float32(math.Cos(th)), float32(math.Sin(th)))
		}
	}
	return iq
}

// balancedStream builds a deterministic, zero-mean dibit stream long
// enough for the CoarseAFC to converge and be measured over a steady-state
// tail.
func balancedStream(n int) []uint8 {
	// A period-4 walk visits every symbol equally (mean 0) and a
	// period-7 twist breaks the trivial repetition so the comparison
	// exercises all four decision regions rather than one steady rail.
	out := make([]uint8, n)
	for i := range out {
		out[i] = uint8((i%4 + i/7) % 4)
	}
	return out
}

// decodeDibits runs one IQ buffer through a DeviationHz-calibrated DMR
// receiver and returns the recovered dibit stream.
func decodeDibits(iq []complex64) []uint8 {
	var got []uint8
	r := New(Options{
		SampleRateHz: 48_000,
		DeviationHz:  1944.0,
		DibitSink:    func(dibits []uint8, baseIdx int) { got = append(got, dibits...) },
	})
	r.Process(iq)
	return got
}

// TestReceiverIsCarrierOffsetInvariant pins issue #836: a narrowband DMR
// C4FM signal carrying an uncorrected tuner carrier offset far beyond the
// slicer's ~±75 Hz tolerance must decode to the SAME dibit stream as the
// zero-offset signal, because the receiver's CoarseAFC removes the
// offset's constant DC bias from the symbols before the slicer sees them.
// The offset is the only difference between the two inputs, so after the
// AFC converges the two recovered streams must agree. Without the AFC the
// shifted 4-level eye mis-slices and the streams diverge (verified
// failing-first by disabling the AFC stage: agreement collapses ~0.99 →
// ~0.79 at this offset).
func TestReceiverIsCarrierOffsetInvariant(t *testing.T) {
	const offsetHz = 400.0 // >5× the ~±75 Hz decode cliff
	stream := balancedStream(600)

	ref := decodeDibits(makeC4FMIQWithOffset(stream, 0))
	off := decodeDibits(makeC4FMIQWithOffset(stream, offsetHz))

	n := len(ref)
	if len(off) < n {
		n = len(off)
	}
	// Skip a warm-up region so the measurement is over the AFC's
	// steady state, not its convergence transient.
	const warmup = 250
	if n <= warmup {
		t.Fatalf("recovered only %d dibits, need > %d", n, warmup)
	}
	agree := 0
	for i := warmup; i < n; i++ {
		if ref[i] == off[i] {
			agree++
		}
	}
	frac := float64(agree) / float64(n-warmup)
	// With the AFC on this sits ~0.99; disabling it drops it to ~0.79
	// (the shifted eye mis-slices). 0.95 cleanly separates the two while
	// leaving margin for the coarse AFC's small steady-state residual.
	if frac < 0.95 {
		t.Fatalf("offset-vs-baseline dibit agreement = %.3f over %d symbols, want >=0.95 — CoarseAFC should make decode invariant to a %.0f Hz carrier offset (issue #836)", frac, n-warmup, offsetHz)
	}
}

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

// makePhaseRampIQ synthesises a 48 kHz / 10 sps IQ buffer whose
// instantaneous frequency cycles through the four C4FM symbols.
// Same fixture shape as the YSF / P25 P1 receiver tests; DMR shares
// the modulation so the signal works without modification.
func makePhaseRampIQ(symbols int) []complex64 {
	const sampleRate = 48_000.0
	const sps = 10
	const deviation = 1944.0 // DMR nominal outer-deviation (per ETSI)
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

// TestSymbolToDibitMatchesP25Phase1Convention pins the slicer-to-
// dibit mapping so a future change here doesn't silently desync
// from the AllSyncs / SyncDetector patterns in the parent dmr
// package. ETSI TS 102 361-1's symbol-to-dibit table matches
// the Gray-coded P25 Phase 1 convention; DSDcc + OP25 confirm.
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
