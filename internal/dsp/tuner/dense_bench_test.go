package tuner

import (
	"math"
	"testing"
)

// BenchmarkDense71DDC / BenchmarkDense71Polyphase quantify why widebandt2's
// auto strategy keeps a dense, high-count plan on the polyphase channelizer
// rather than the per-tap DDC. The fixture is the 70-DMR stress-test plan: 71
// repeaters spread across ±1.9 MHz of a 10 MS/s capture.
//
// Real-time budget for the 8192-sample chunk used here is 8192/10e6 ≈ 819 µs;
// a bank must process each chunk within that to keep up with the live pump.
// Measured on a 2.8 GHz Xeon (slow CI core): the channelizer runs ~1.7 ms/chunk
// and the DDC ~11 ms/chunk — the DDC is ~6× heavier because it runs 71
// independent reduced-rate mixer+resampler chains while the channelizer shares
// one FFT. Both exceed the budget on that slow core, but the channelizer's ~6×
// headroom is what lets it stay real-time on the faster multi-core hosts these
// wideband plans actually run on; the DDC would drop samples on every tap.
func benchDenseOffsets() []float64 {
	const center = 441_700_000.0
	freqs := []float64{
		439787500, 440012500, 440062500, 440087500, 440112500, 440187500, 440262500,
		440312500, 440362500, 440387500, 440437500, 440462500, 440487500, 440512500,
		440562500, 440587500, 440712500, 440737500, 440762500, 440787500, 440862500,
		440912500, 440937500, 441025000, 441312500, 441337500, 441362500, 441387500,
		441412500, 441437500, 441462500, 441487500, 441512500, 441537500, 441562500,
		441587500, 441612500, 441637500, 441662500, 441687500, 441787500, 441812500,
		441821500, 441837500, 441862500, 441887500, 441912500, 442287500, 442337500,
		442387500, 442412500, 442432500, 442437500, 442512500, 442562500, 442837500,
		442887500, 442937500, 442962500, 443037500, 443137500, 443162500, 443187500,
		443237500, 443412500, 443437500, 443462500, 443487500, 443587500, 443600000,
		443612500,
	}
	offs := make([]float64, len(freqs))
	for i, f := range freqs {
		offs[i] = f - center
	}
	return offs
}

func BenchmarkDense71DDC(b *testing.B) {
	offs := benchDenseOffsets()
	var maxAbs float64
	for _, o := range offs {
		if a := math.Abs(o); a > maxAbs {
			maxAbs = a
		}
	}
	bank := NewDDCBankForSpan(10_000_000, 48_000, 0.05, maxAbs)
	for _, o := range offs {
		if err := bank.AddTap(o, func([]complex64) {}); err != nil {
			b.Fatalf("AddTap(%.0f): %v", o, err)
		}
	}
	chunk := newToneGen(10_000_000, 0.2, offs...).Next(8192)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bank.Process(chunk)
	}
}

func BenchmarkDense71Polyphase(b *testing.B) {
	offs := benchDenseOffsets()
	bank := NewChannelizerBank(10_000_000, 48_000, 0.05, 64, 16, 9.0)
	for _, o := range offs {
		if err := bank.AddTap(o, func([]complex64) {}); err != nil {
			b.Fatalf("AddTap(%.0f): %v", o, err)
		}
	}
	chunk := newToneGen(10_000_000, 0.2, offs...).Next(8192)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bank.Process(chunk)
	}
}
