package symbolscope

import (
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	p25phase2rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestP25Phase2EmitsComplexConstellation pins the Phase 2 scope support: an
// H-DQPSK stream drives the Phase 2 receiver on its soft-decision path,
// whose complex differentials (the constellation points, rotated onto the
// diagonals) reach the frame as SymI/SymQ aligned with the dibits, at the
// 6000-baud Phase 2 symbol rate, with no C4FM soft/eye track — the TETRA
// posture on a second linear protocol.
func TestP25Phase2EmitsComplexConstellation(t *testing.T) {
	// 8 samples/symbol × 6000 baud = 48 kHz, the C4FM-family channel rate
	// the DDC targets, so the input feeds the receiver near-unity.
	const sps = 8
	dibits := make([]uint8, 4000)
	for i := range dibits {
		dibits[i] = uint8((i*7 + 3) & 3)
	}
	iq := demod.ModulateHDQPSKSpec(dibits, sps, p25phase2rx.PulseSpanSymbols, p25phase2rx.RolloffAlpha)
	for i := range iq {
		iq[i] *= 100
	}

	var frames []Frame
	e, err := New(Options{
		Emit:         func(f Frame) { frames = append(frames, f) },
		InRateHz:     6000 * sps,
		Protocol:     trunking.ProtocolP25Phase2,
		FrameSymbols: 64,
	})
	if err != nil {
		t.Fatalf("New(P25Phase2): %v", err)
	}
	if math.Abs(e.SymbolRateHz()-p25phase2rx.SymbolRate) > 1 {
		t.Fatalf("SymbolRateHz = %v, want %v", e.SymbolRateHz(), p25phase2rx.SymbolRate)
	}
	e.Process(iq)

	if len(frames) == 0 {
		t.Fatal("no frames emitted on the P25 Phase 2 path")
	}
	sawSymbols := false
	for fi, f := range frames {
		if math.Abs(f.SymbolRateHz-p25phase2rx.SymbolRate) > 1 {
			t.Errorf("frame %d: SymbolRateHz = %v, want %v", fi, f.SymbolRateHz, p25phase2rx.SymbolRate)
		}
		if len(f.Soft) != 0 || len(f.EyeSoft) != 0 {
			t.Errorf("frame %d: Phase 2 has no soft/eye track, got soft=%d eye=%d", fi, len(f.Soft), len(f.EyeSoft))
		}
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
		t.Fatal("P25 Phase 2 path emitted no complex constellation points")
	}
}
