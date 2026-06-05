package sync

import "testing"

// TestGardnerProcess2xCadence verifies Process2x emits exactly two samples per
// recovered symbol — the same symbol cadence as Process — with the pair ordered
// [mid, on-time] so the on-time sample equals what Process emits. Identical
// cadence is required so chunked CQPSK consumers stay in sync (issue #275).
func TestGardnerProcess2xCadence(t *testing.T) {
	const sps = 10.0
	mk := func() []complex64 {
		out := make([]complex64, 0, 2000)
		// A simple alternating pattern at sps so timing has something to track.
		for i := 0; i < 2000; i++ {
			if (i/int(sps))%2 == 0 {
				out = append(out, complex(1, 0))
			} else {
				out = append(out, complex(-1, 0))
			}
		}
		return out
	}
	src := mk()

	g1 := NewGardner(sps, 0.01)
	one := g1.Process(nil, src)

	g2 := NewGardner(sps, 0.01)
	two := g2.Process2x(nil, src)

	if len(two) != 2*len(one) {
		t.Fatalf("Process2x emitted %d samples; want 2*%d = %d", len(two), len(one), 2*len(one))
	}
	// The on-time sample of each pair (odd index) must match Process's symbol.
	for i := range one {
		if two[2*i+1] != one[i] {
			t.Fatalf("pair %d on-time = %v, want %v (Process2x must emit [mid, on-time])", i, two[2*i+1], one[i])
		}
	}
}
