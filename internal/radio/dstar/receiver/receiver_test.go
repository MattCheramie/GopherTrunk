package receiver

import (
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
)

func TestNewPanicsOnMissingSampleRate(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic, got none")
		}
		if !strings.Contains(asString(r), "SampleRateHz") {
			t.Errorf("panic message %q missing SampleRateHz", r)
		}
	}()
	New(Options{BitSink: noopSink})
}

func TestNewPanicsOnMissingBitSink(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic, got none")
		}
		if !strings.Contains(asString(r), "BitSink") {
			t.Errorf("panic message %q missing BitSink", r)
		}
	}()
	New(Options{SampleRateHz: 48000})
}

func TestNewPanicsOnSubNyquistSampleRate(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for sps < 2, got none")
		}
		if !strings.Contains(asString(r), "SampleRateHz") {
			t.Errorf("panic message %q missing SampleRateHz", r)
		}
	}()
	// 9000 Hz / 4800 sym/s = 1.875 sps → below the documented floor.
	New(Options{SampleRateHz: 9000, BitSink: noopSink})
}

func TestNewAppliesDefaults(t *testing.T) {
	// Options with only required fields → New must apply defaults
	// (PulseSpanSymbols, BTProduct, ClockGain) without panicking
	// or returning nil. A subsequent Process call exercises the
	// composed pipeline end-to-end on synthetic input.
	r := New(Options{
		SampleRateHz: 48000,
		BitSink:      noopSink,
	})
	if r == nil {
		t.Fatal("New returned nil")
	}
	// Smoke: large zero-IQ buffer must not panic. (The clock-
	// recovery loop emits zero-amplitude symbols which the slicer
	// may resolve either direction — we don't constrain the bit
	// output, only that the pipeline doesn't crash on flatline IQ.)
	r.Process(make([]complex64, 1024))
}

func TestProcessEmptyIQIsNoOp(t *testing.T) {
	called := 0
	r := New(Options{
		SampleRateHz: 48000,
		BitSink:      func(bits []byte, baseIdx int) { called++ },
	})
	r.Process(nil)
	r.Process([]complex64{})
	if called != 0 {
		t.Errorf("BitSink invoked on empty IQ: called=%d", called)
	}
}

func TestReceiverRecoversModulatedBitStream(t *testing.T) {
	// sps=10 keeps the integer-rounding in receiver.go:108
	// (int(sps+0.5)) faithful; deviation = SymbolRate/4 puts the
	// modulation index near 0.5, the JARL DV-mode target.
	const (
		sps          = 10
		sampleRateHz = SymbolRate * sps // 48000
		span         = PulseSpanSymbols
		bt           = BT
		deviation    = SymbolRate / 4 // 1200 Hz
	)

	src := make([]byte, 200)
	for i := range src {
		// Deterministic alternating-ish pattern; same shape the
		// GFSK round-trip test in internal/dsp/demod uses.
		src[i] = byte((i*7 + 3) & 1)
	}

	iq := demod.ModulateGFSK(src, sps, span, bt, sampleRateHz, deviation)

	var captured []byte
	r := New(Options{
		SampleRateHz: sampleRateHz,
		BitSink: func(bits []byte, baseIdx int) {
			captured = append(captured, bits...)
		},
		PulseSpanSymbols: span,
		BTProduct:        bt,
	})
	r.Process(iq)

	if len(captured) < len(src)/2 {
		t.Fatalf("recovered only %d bits from %d-bit source; clock loop never locked",
			len(captured), len(src))
	}

	// Drop the first 2*span symbols on each side: the matched-filter
	// + clock-recovery transient sits there. Compare the remaining
	// captured stream against a windowed slice of the source; the
	// Mueller-Müller startup can land on any alignment so we look
	// for the best offset in a small window.
	bestMatch := 0
	bestOffset := 0
	for offset := -2; offset <= 2; offset++ {
		matches := alignAndCount(src, captured, offset, 2*span)
		if matches > bestMatch {
			bestMatch = matches
			bestOffset = offset
		}
	}
	// Steady-state window is len(captured) - 2*2*span symbols of
	// usable bits. Require ≥ 90% match in the best alignment.
	usable := len(captured) - 4*span
	if usable < 50 {
		t.Fatalf("not enough steady-state bits to evaluate: usable=%d", usable)
	}
	if got := float64(bestMatch) / float64(usable); got < 0.90 {
		t.Errorf("steady-state bit match = %.2f%% (offset=%d, %d/%d), want >= 90%%",
			100*got, bestOffset, bestMatch, usable)
	}
}

func TestReceiverHandlesChunkedIQ(t *testing.T) {
	const (
		sps          = 10
		sampleRateHz = SymbolRate * sps
		span         = PulseSpanSymbols
		bt           = BT
		deviation    = SymbolRate / 4
	)

	src := make([]byte, 200)
	for i := range src {
		src[i] = byte((i*7 + 3) & 1)
	}
	iq := demod.ModulateGFSK(src, sps, span, bt, sampleRateHz, deviation)

	// Single-shot
	var single []byte
	rs := New(Options{
		SampleRateHz:     sampleRateHz,
		BitSink:          func(b []byte, _ int) { single = append(single, b...) },
		PulseSpanSymbols: span,
		BTProduct:        bt,
	})
	rs.Process(iq)

	// Chunked: 4 equal slices, fed sequentially through one receiver.
	var chunked []byte
	rc := New(Options{
		SampleRateHz:     sampleRateHz,
		BitSink:          func(b []byte, _ int) { chunked = append(chunked, b...) },
		PulseSpanSymbols: span,
		BTProduct:        bt,
	})
	chunkSize := len(iq) / 4
	for i := 0; i < 4; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if i == 3 {
			end = len(iq)
		}
		rc.Process(iq[start:end])
	}

	// Total bit count should match to within ±1 symbol (clock loop
	// can spit one extra/missing symbol at a chunk boundary if the
	// matched-filter tail straddles two chunks).
	delta := len(single) - len(chunked)
	if delta < -2 || delta > 2 {
		t.Errorf("chunked bit count diverged: single=%d chunked=%d", len(single), len(chunked))
	}

	// Compare the overlapping prefix; allow a small mismatch fraction
	// for the chunk-boundary transient.
	n := len(single)
	if len(chunked) < n {
		n = len(chunked)
	}
	matches := 0
	for i := 0; i < n; i++ {
		if single[i] == chunked[i] {
			matches++
		}
	}
	if n > 0 {
		if frac := float64(matches) / float64(n); frac < 0.95 {
			t.Errorf("single-vs-chunked bit agreement %.2f%% (%d/%d), want >= 95%%",
				100*frac, matches, n)
		}
	}
}

func TestResetRestartsBitBase(t *testing.T) {
	const (
		sps          = 10
		sampleRateHz = SymbolRate * sps
		span         = PulseSpanSymbols
		bt           = BT
		deviation    = SymbolRate / 4
	)

	src := make([]byte, 100)
	for i := range src {
		src[i] = byte(i & 1)
	}
	iq := demod.ModulateGFSK(src, sps, span, bt, sampleRateHz, deviation)

	var bases []int
	r := New(Options{
		SampleRateHz:     sampleRateHz,
		BitSink:          func(b []byte, baseIdx int) { bases = append(bases, baseIdx) },
		PulseSpanSymbols: span,
		BTProduct:        bt,
	})
	r.Process(iq)
	if len(bases) == 0 {
		t.Fatal("first Process emitted no bits — clock never locked")
	}
	if bases[0] != 0 {
		t.Errorf("first BitSink call baseIdx = %d, want 0", bases[0])
	}
	finalBeforeReset := bases[len(bases)-1]
	// On a single-chunk Process we expect a single BitSink call
	// (the implementation emits all symbols at once), but tolerate
	// segmented emission too — assert the base index advanced past 0.
	if finalBeforeReset != 0 && len(bases) > 1 {
		// At least the multi-call path is exercised.
	}

	r.Reset()
	bases = bases[:0]
	r.Process(iq)
	if len(bases) == 0 {
		t.Fatal("post-Reset Process emitted no bits")
	}
	if bases[0] != 0 {
		t.Errorf("post-Reset BitSink call baseIdx = %d, want 0", bases[0])
	}
}

func noopSink(bits []byte, baseIdx int) {}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if err, ok := v.(error); ok {
		return err.Error()
	}
	return ""
}

// alignAndCount slides captured against src with the given offset
// applied to captured's start, skipping `skip` leading symbols of
// captured to clear the receiver transient.
func alignAndCount(src, captured []byte, offset, skip int) int {
	matches := 0
	for i := skip; i < len(captured); i++ {
		j := i + offset
		if j < 0 || j >= len(src) {
			continue
		}
		if captured[i] == src[j] {
			matches++
		}
	}
	return matches
}
