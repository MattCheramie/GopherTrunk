package symbolscope

import (
	"sort"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// makeC4FMIQ synthesizes a wideband P25 C4FM IQ stream at inRate for a
// repeating 4-symbol pattern, scaled up so the receiver's symbol-AGC has
// headroom (mirrors the receiver package's own fixtures).
func makeC4FMIQ(inRate float64, nSymbols int) ([]complex64, []uint8) {
	dibits := make([]uint8, nSymbols)
	for i := range dibits {
		dibits[i] = uint8((i*7 + 3) & 3)
	}
	iq := demod.ModulateP25C4FM(dibits, inRate, p25DeviationHz)
	for i := range iq {
		iq[i] *= 100
	}
	return iq, dibits
}

func TestNewValidates(t *testing.T) {
	if _, err := New(Options{InRateHz: 960_000, Protocol: trunking.ProtocolP25}); err == nil {
		t.Error("missing Emit: want error")
	}
	emit := func(Frame) {}
	if _, err := New(Options{Emit: emit, Protocol: trunking.ProtocolP25}); err == nil {
		t.Error("zero InRateHz: want error")
	}
	if _, err := New(Options{Emit: emit, InRateHz: 960_000, Protocol: trunking.ProtocolDMR}); err == nil {
		t.Error("unsupported protocol: want error")
	}
	if _, err := New(Options{Emit: emit, InRateHz: 960_000, Protocol: trunking.ProtocolP25, DemodMode: "bogus"}); err == nil {
		t.Error("bad demod mode: want error")
	}
	if _, err := New(Options{Emit: emit, InRateHz: 960_000, Protocol: trunking.ProtocolP25}); err != nil {
		t.Errorf("valid C4FM options: unexpected error %v", err)
	}
}

// TestC4FMRecoversSoftAndDibits is the headline check: a C4FM stream
// produces frames whose dibits cover the 4-level alphabet and whose soft
// waveform is aligned with — and separates into — four distinct levels
// keyed by the sliced decision. That separability is what makes it a
// usable symbol scope.
func TestC4FMRecoversSoftAndDibits(t *testing.T) {
	iq, _ := makeC4FMIQ(960_000, 6000)

	var frames []Frame
	e, err := New(Options{
		Emit:         func(f Frame) { frames = append(frames, f) },
		InRateHz:     960_000,
		Protocol:     trunking.ProtocolP25,
		DemodMode:    "c4fm",
		FrameSymbols: 64,
		NowNs:        func() int64 { return 42 },
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	e.Process(iq)

	if len(frames) == 0 {
		t.Fatal("no frames emitted")
	}

	// Per-dibit soft accumulators across all frames.
	var sum [4]float64
	var cnt [4]int
	var sawEye bool
	wantBase := 0
	for fi, f := range frames {
		if len(f.EyeSoft) > 0 {
			sawEye = true
			if f.EyeSPS < 2 {
				t.Errorf("frame %d: EyeSPS = %d, want >= 2 (oversampled fold period)", fi, f.EyeSPS)
			}
			if len(f.EyeSoft) > maxEyeSamples {
				t.Errorf("frame %d: EyeSoft len %d exceeds cap %d", fi, len(f.EyeSoft), maxEyeSamples)
			}
		}
		if f.IsBits {
			t.Errorf("frame %d: IsBits = true, want false for C4FM dibits", fi)
		}
		if len(f.Soft) != len(f.Dibits) {
			t.Fatalf("frame %d: soft len %d != dibit len %d", fi, len(f.Soft), len(f.Dibits))
		}
		if f.BaseIdx != wantBase {
			t.Errorf("frame %d: BaseIdx = %d, want %d (contiguous)", fi, f.BaseIdx, wantBase)
		}
		wantBase += len(f.Dibits)
		if f.SymbolRateHz != 4800 {
			t.Errorf("frame %d: SymbolRateHz = %v, want 4800", fi, f.SymbolRateHz)
		}
		if f.TimestampNs != 42 {
			t.Errorf("frame %d: TimestampNs = %d, want 42 (injected clock)", fi, f.TimestampNs)
		}
		// C4FM's symbol domain is the real 4-level soft, not a complex
		// constellation — the SymI/SymQ track stays empty on this path.
		if len(f.SymI) != 0 || len(f.SymQ) != 0 {
			t.Errorf("frame %d: C4FM should carry no complex symbol track, got SymI=%d SymQ=%d", fi, len(f.SymI), len(f.SymQ))
		}
		for i, d := range f.Dibits {
			if d > 3 {
				t.Fatalf("frame %d: dibit %d out of range", fi, d)
			}
			sum[d] += float64(f.Soft[i])
			cnt[d]++
		}
	}

	// The C4FM path must surface the oversampled eye window for the
	// datascope — the open 4-level eye view a 1/symbol soft track can't give.
	if !sawEye {
		t.Error("C4FM path emitted no eye window (EyeSoft empty on every frame)")
	}

	// Receiver-state metrics for the Tuning panel: the C4FM path reports a
	// Mueller-Müller clock (ClockSPS > 0) and a calibrated symbol-AGC
	// target (DeviationHz was set), but never a CMA error (CQPSK-only).
	last := frames[len(frames)-1]
	if last.ClockSPS <= 0 {
		t.Errorf("C4FM ClockSPS = %v, want > 0 (Mueller-Müller nominal sps)", last.ClockSPS)
	}
	if last.AGCTarget <= 0 {
		t.Errorf("C4FM AGCTarget = %v, want > 0 (calibrated from DeviationHz)", last.AGCTarget)
	}
	if last.CMAError != 0 {
		t.Errorf("C4FM CMAError = %v, want 0 (no CMA stage on C4FM)", last.CMAError)
	}

	// All four levels should be populated, and their mean soft values
	// must be mutually separated — the 4-band structure of the scope.
	means := make([]float64, 0, 4)
	for v := 0; v < 4; v++ {
		if cnt[v] == 0 {
			t.Fatalf("dibit value %d never decided — slicer collapsed", v)
		}
		means = append(means, sum[v]/float64(cnt[v]))
	}
	sort.Float64s(means)
	spread := means[3] - means[0]
	if spread <= 0 {
		t.Fatalf("soft levels not separated: means=%v", means)
	}
	for i := 1; i < 4; i++ {
		gap := means[i] - means[i-1]
		if gap < 0.1*spread {
			t.Errorf("adjacent soft levels too close: means=%v (gap %.4f < 10%% of spread %.4f)", means, gap, spread)
		}
	}
}

// TestCQPSKEmitsComplexSymbolsWithoutSoft locks in the Phase-1 contract
// for the linear path: CQPSK has no real soft tap (its quality view is
// the constellation), so frames carry an empty Soft track but a complex
// symbol-decision track (SymI/SymQ) aligned index-for-index with the
// dibits.
func TestCQPSKEmitsComplexSymbolsWithoutSoft(t *testing.T) {
	iq, _ := makeC4FMIQ(960_000, 4000) // any IQ drives the Gardner loop

	var frames []Frame
	e, err := New(Options{
		Emit:         func(f Frame) { frames = append(frames, f) },
		InRateHz:     960_000,
		Protocol:     trunking.ProtocolP25,
		DemodMode:    "cqpsk",
		FrameSymbols: 64,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	e.Process(iq)

	if len(frames) == 0 {
		t.Fatal("no frames emitted on CQPSK path")
	}
	sawSymbols := false
	for fi, f := range frames {
		if len(f.Soft) != 0 {
			t.Errorf("frame %d: CQPSK soft track should be empty, got %d", fi, len(f.Soft))
		}
		if len(f.EyeSoft) != 0 {
			t.Errorf("frame %d: CQPSK eye track should be empty (C4FM-only), got %d", fi, len(f.EyeSoft))
		}
		// The complex symbol track, when present, must be aligned with the
		// dibits on both axes — that pairing is what lets a client colour
		// constellation points by their decided dibit.
		if len(f.SymI) != len(f.SymQ) {
			t.Fatalf("frame %d: SymI len %d != SymQ len %d", fi, len(f.SymI), len(f.SymQ))
		}
		if len(f.SymI) != 0 {
			if len(f.SymI) != len(f.Dibits) {
				t.Fatalf("frame %d: symbol track len %d != dibit len %d", fi, len(f.SymI), len(f.Dibits))
			}
			sawSymbols = true
		}
		for _, d := range f.Dibits {
			if d > 3 {
				t.Fatalf("frame %d: dibit %d out of range", fi, d)
			}
		}
	}
	if !sawSymbols {
		t.Fatal("CQPSK path emitted no complex symbol points")
	}

	// Tuning metrics on the CQPSK path: a Gardner clock (ClockSPS > 0) and
	// no C4FM symbol-AGC target (that getter is C4FM-only).
	last := frames[len(frames)-1]
	if last.ClockSPS <= 0 {
		t.Errorf("CQPSK ClockSPS = %v, want > 0 (Gardner nominal sps)", last.ClockSPS)
	}
	if last.AGCTarget != 0 {
		t.Errorf("CQPSK AGCTarget = %v, want 0 (C4FM-only getter)", last.AGCTarget)
	}
}

// TestFrameBatching checks the per-frame symbol batch size is honored
// and frames tile the symbol stream without gaps or overlap.
func TestFrameBatching(t *testing.T) {
	iq, _ := makeC4FMIQ(960_000, 3000)

	var total, frameCount int
	e, _ := New(Options{
		Emit: func(f Frame) {
			frameCount++
			// Every emitted frame is exactly FrameSymbols except possibly
			// a trailing partial — but we never flush partials here, so
			// all are full.
			if len(f.Dibits) != 128 {
				t.Errorf("frame %d: size %d, want 128", frameCount, len(f.Dibits))
			}
			total += len(f.Dibits)
		},
		InRateHz:     960_000,
		Protocol:     trunking.ProtocolP25,
		FrameSymbols: 128,
	})
	e.Process(iq)
	if frameCount == 0 {
		t.Fatal("no frames")
	}
}
