package demod

import (
	"math"
	"testing"
)

// TestCoarseAFCRemovesConstantBias feeds a balanced 4-level stream with
// a constant DC bias (the signature of a carrier-frequency offset on an
// FM-discriminator output) and confirms the tracker estimates the bias
// and subtracts it back to a near-zero-mean stream.
func TestCoarseAFCRemovesConstantBias(t *testing.T) {
	const sps = 10
	const bias = 0.0654 // ≈ 500 Hz offset at 48 kHz, in rad/sample
	levels := []float32{1, 3, -1, -3}

	buf := make([]float32, 20000)
	for i := range buf {
		buf[i] = levels[i%4] + float32(bias)
	}
	afc := NewCoarseAFC(sps)
	afc.Process(buf)

	if math.Abs(afc.Offset()-bias) > 0.05*bias {
		t.Errorf("Offset() = %.5f, want within 5%% of %.5f", afc.Offset(), bias)
	}
	// After convergence the corrected stream's mean should be ~0 (the
	// balanced levels), not the input mean of `bias`.
	var sum float64
	tail := buf[len(buf)-4000:]
	for _, x := range tail {
		sum += float64(x)
	}
	if mean := sum / float64(len(tail)); math.Abs(mean) > 0.01 {
		t.Errorf("post-AFC tail mean = %.4f, want ≈ 0", mean)
	}
}

// TestCoarseAFCResetClearsEstimate confirms Reset returns the tracker
// to its initial state so a re-tune doesn't carry a stale offset.
func TestCoarseAFCResetClearsEstimate(t *testing.T) {
	afc := NewCoarseAFC(10)
	buf := make([]float32, 5000)
	for i := range buf {
		buf[i] = 0.2
	}
	afc.Process(buf)
	if afc.Offset() == 0 {
		t.Fatalf("Offset() = 0 after processing a biased stream, want non-zero")
	}
	afc.Reset()
	if afc.Offset() != 0 {
		t.Errorf("Offset() = %v after Reset, want 0", afc.Offset())
	}
}

// TestNewCoarseAFCRejectsBadSps documents the constructor precondition.
func TestNewCoarseAFCRejectsBadSps(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("NewCoarseAFC(0) did not panic")
		}
	}()
	NewCoarseAFC(0)
}
