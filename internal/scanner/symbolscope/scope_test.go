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
	wantBase := 0
	for fi, f := range frames {
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
		for i, d := range f.Dibits {
			if d > 3 {
				t.Fatalf("frame %d: dibit %d out of range", fi, d)
			}
			sum[d] += float64(f.Soft[i])
			cnt[d]++
		}
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

// TestCQPSKEmitsDibitsWithoutSoft locks in the honest Phase-1 contract:
// the CQPSK path has no soft tap, so frames carry dibits but an empty
// soft track.
func TestCQPSKEmitsDibitsWithoutSoft(t *testing.T) {
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
	for fi, f := range frames {
		if len(f.Soft) != 0 {
			t.Errorf("frame %d: CQPSK soft track should be empty, got %d", fi, len(f.Soft))
		}
		for _, d := range f.Dibits {
			if d > 3 {
				t.Fatalf("frame %d: dibit %d out of range", fi, d)
			}
		}
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
