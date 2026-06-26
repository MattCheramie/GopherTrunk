package tuner

import (
	"runtime"
	"sync"
	"testing"
)

// buildDenseChannelizer returns a ChannelizerBank with the 70-DMR stress-test
// plan's taps registered, each sink appending its output to a per-tap capture
// slice. Returns the bank and the capture slices (indexed by tap order).
func buildDenseChannelizer() (*ChannelizerBank, [][]complex64) {
	offs := benchDenseOffsets()
	bank := NewChannelizerBank(10_000_000, 48_000, 0.05, 64, 16, 9.0)
	caps := make([][]complex64, len(offs))
	for i, o := range offs {
		i := i
		if err := bank.AddTap(o, func(out []complex64) {
			caps[i] = append(caps[i], out...)
		}); err != nil {
			panic(err)
		}
	}
	return bank, caps
}

// TestChannelizerWorkersDeterministic is the Fix-C correctness regression: the
// parallel per-tap path must produce bit-identical output to the serial path.
// Taps are independent (each owns its NCO/resampler/buffers, reads bins
// read-only), so the result must not depend on whether they run on one
// goroutine or many.
func TestChannelizerWorkersDeterministic(t *testing.T) {
	serial, serialOut := buildDenseChannelizer()
	parallel, parallelOut := buildDenseChannelizer()
	parallel.EnableWorkers(8)
	defer parallel.Close()
	if parallel.pool == nil {
		t.Fatalf("EnableWorkers(8) left the bank serial; want a pool for %d taps", len(parallel.taps))
	}

	// Feed both banks the identical wideband stream, several chunks so the
	// resamplers settle and accumulate plenty of output.
	gen1 := newToneGen(10_000_000, 0.2, benchDenseOffsets()...)
	gen2 := newToneGen(10_000_000, 0.2, benchDenseOffsets()...)
	for c := 0; c < 20; c++ {
		chunk1 := gen1.Next(8192)
		chunk2 := gen2.Next(8192)
		serial.Process(chunk1)
		parallel.Process(chunk2)
	}

	if len(serialOut) != len(parallelOut) {
		t.Fatalf("tap count mismatch: serial %d, parallel %d", len(serialOut), len(parallelOut))
	}
	for i := range serialOut {
		s, p := serialOut[i], parallelOut[i]
		if len(s) != len(p) {
			t.Fatalf("tap %d output length: serial %d, parallel %d", i, len(s), len(p))
		}
		for j := range s {
			if s[j] != p[j] {
				t.Fatalf("tap %d sample %d differs: serial %v, parallel %v", i, j, s[j], p[j])
			}
		}
	}
}

// TestEnableWorkersBelowThresholdStaysSerial confirms a small plan keeps the
// byte-identical serial path (no pool created).
func TestEnableWorkersBelowThresholdStaysSerial(t *testing.T) {
	bank := NewChannelizerBank(2_400_000, 48_000, 0.05, 16, 16, 9.0)
	for _, o := range []float64{-200_000, 0, 200_000} {
		if err := bank.AddTap(o, func([]complex64) {}); err != nil {
			t.Fatal(err)
		}
	}
	bank.EnableWorkers(8)
	if bank.pool != nil {
		t.Errorf("EnableWorkers created a pool for %d taps (< threshold %d)", len(bank.taps), parallelTapThreshold)
	}
	// Close must be safe with no pool.
	bank.Close()
}

// TestTapPoolRunsEveryIndexOnce checks the pool's shard layout covers [0,total)
// exactly once across an uneven split, with concurrent workers.
func TestTapPoolRunsEveryIndexOnce(t *testing.T) {
	const total = 71
	p := newTapPool(total, 8)
	defer p.close()

	var mu sync.Mutex
	counts := make([]int, total)
	p.run(func(i int) {
		mu.Lock()
		counts[i]++
		mu.Unlock()
	})
	for i, c := range counts {
		if c != 1 {
			t.Errorf("index %d ran %d times, want 1", i, c)
		}
	}

	// A second round must rerun every index (reusable barrier, no re-spawn).
	for i := range counts {
		counts[i] = 0
	}
	p.run(func(i int) {
		mu.Lock()
		counts[i]++
		mu.Unlock()
	})
	for i, c := range counts {
		if c != 1 {
			t.Errorf("round 2 index %d ran %d times, want 1", i, c)
		}
	}
}

func TestDefaultTapWorkers(t *testing.T) {
	got := DefaultTapWorkers()
	if got < 1 {
		t.Errorf("DefaultTapWorkers = %d, want >= 1", got)
	}
	if got > 8 {
		t.Errorf("DefaultTapWorkers = %d, want <= 8 (cap)", got)
	}
	if want := runtime.NumCPU() - 2; want >= 1 && want <= 8 && got != want {
		t.Errorf("DefaultTapWorkers = %d, want %d (NumCPU-2)", got, want)
	}
}

// BenchmarkDense71PolyphaseWorkers measures the parallel per-tap path against
// the serial BenchmarkDense71Polyphase so the speedup on the 70-DMR plan is
// visible. Same 8192-sample chunk and ~819 µs real-time budget.
func BenchmarkDense71PolyphaseWorkers(b *testing.B) {
	offs := benchDenseOffsets()
	bank := NewChannelizerBank(10_000_000, 48_000, 0.05, 64, 16, 9.0)
	for _, o := range offs {
		if err := bank.AddTap(o, func([]complex64) {}); err != nil {
			b.Fatalf("AddTap(%.0f): %v", o, err)
		}
	}
	bank.EnableWorkers(DefaultTapWorkers())
	defer bank.Close()
	chunk := newToneGen(10_000_000, 0.2, offs...).Next(8192)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bank.Process(chunk)
	}
}
