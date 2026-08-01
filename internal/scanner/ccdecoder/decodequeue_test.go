package ccdecoder

import "testing"

// TestDecodeQueueBudgetScalesWithRate pins the issue #402 fix: the decode queue's
// depth is a wall-clock sample budget derived from the sample rate, NOT a fixed
// count of chunks. The old fixed 128-chunk depth collapsed to tens of ms on
// small-datagram sources (a remote USRP over SoapyRemote), producing decode
// overruns at idle CPU. The budget must give a usable fraction of a second at
// every realistic rate and clamp memory at very high rates.
func TestDecodeQueueBudgetScalesWithRate(t *testing.T) {
	cases := []struct {
		name       string
		rateHz     float64
		wantMinSec float64 // budget must buy at least this many seconds
	}{
		{"usrp-1Msps", 1_000_000, 0.4}, // the reported case: was ~47 ms at 128 chunks
		{"airspy-2p4Msps", 2_400_000, 0.4},
		{"replay-144k", 144_000, 0.0}, // floored, not zero
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			budget, chanCap := decodeQueueBudget(c.rateHz)
			if budget < minDecodeQueueSamples {
				t.Fatalf("budget %d < floor %d", budget, minDecodeQueueSamples)
			}
			gotSec := float64(budget) / c.rateHz
			if gotSec < c.wantMinSec {
				t.Errorf("rate %.0f: budget %d = %.3f s, want >= %.3f s", c.rateHz, budget, gotSec, c.wantMinSec)
			}
			// The backing channel must hold ~budget/64 chunks (to within
			// integer-division rounding) so the sample budget — not the slot
			// count — is the limiter, unless clamped at the max.
			if chanCap < minDecodeQueueChunks {
				t.Errorf("chanCap %d below floor %d", chanCap, minDecodeQueueChunks)
			}
			if int64(chanCap+1)*64 < budget && chanCap < maxDecodeQueueChunks {
				t.Errorf("chanCap %d too small for budget %d", chanCap, budget)
			}
		})
	}
}

// TestDecodeQueueBudgetClampsMemory guards the upper clamp: a very high sample
// rate must not size an unbounded queue.
func TestDecodeQueueBudgetClampsMemory(t *testing.T) {
	budget, chanCap := decodeQueueBudget(50_000_000) // 50 MS/s
	if budget != maxDecodeQueueSamples {
		t.Errorf("budget at 50 MS/s = %d, want clamp %d", budget, maxDecodeQueueSamples)
	}
	if chanCap > maxDecodeQueueChunks {
		t.Errorf("chanCap %d exceeds clamp %d", chanCap, maxDecodeQueueChunks)
	}
}

// TestDecodeQueueBudgetVsOldFixedDepth documents why the old constant was wrong:
// 128 chunks of ~369 samples (a SoapyRemote cs16 datagram) at 1 MS/s was ~47 ms,
// far under the driver's own 400 ms channel — so the decode queue overflowed
// while the driver reported nothing. The new budget must beat that comfortably.
func TestDecodeQueueBudgetVsOldFixedDepth(t *testing.T) {
	const soapyChunkSamples = (1500 - 24) / 4 // MTU - header, complex-int16
	oldSec := float64(128*soapyChunkSamples) / 1_000_000
	budget, _ := decodeQueueBudget(1_000_000)
	newSec := float64(budget) / 1_000_000
	if newSec <= oldSec {
		t.Fatalf("new budget %.3f s not better than old fixed depth %.3f s", newSec, oldSec)
	}
	if oldSec > 0.1 {
		t.Fatalf("sanity: old fixed depth %.3f s unexpectedly large", oldSec)
	}
}
