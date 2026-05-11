package receiver

import "testing"

// TestReceiverGardnerProcessesIQEndToEnd: configure the receiver
// with ClockGardner and confirm the Gardner-driven decimation path
// produces a non-zero number of dibits on a clean P25 Phase 2
// H-DQPSK stream. We don't assert symbol-for-symbol equality
// (Gardner's timing-recovery loop picks slightly different
// sampling points than naive decimation during convergence); the
// invariants checked are:
//
//   - DibitSink fires at least once
//   - Every emitted dibit is in 0..3
//   - baseIdx is monotonically non-decreasing
func TestReceiverGardnerProcessesIQEndToEnd(t *testing.T) {
	var batches int
	var bad int
	var lastBase int
	r := New(Options{
		SampleRateHz: 48_000,
		DibitSink: func(d []uint8, baseIdx int) {
			batches++
			if baseIdx < lastBase {
				t.Errorf("baseIdx regressed: %d < %d", baseIdx, lastBase)
			}
			lastBase = baseIdx + len(d)
			for _, v := range d {
				if v > 3 {
					bad++
				}
			}
		},
		ClockMode: ClockGardner,
	})

	dibits := []uint8{
		0b00, 0b01, 0b10, 0b11, 0b00, 0b01, 0b10, 0b11,
		0b11, 0b10, 0b01, 0b00, 0b11, 0b10, 0b01, 0b00,
		0b00, 0b01, 0b10, 0b11, 0b00, 0b01, 0b10, 0b11,
		0b11, 0b10, 0b01, 0b00, 0b11, 0b10, 0b01, 0b00,
	}
	iq, _ := makeP2HDQPSKIQ(dibits)
	chunk := 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[i:end])
	}
	if batches == 0 {
		t.Errorf("ClockGardner DibitSink received zero batches")
	}
	if bad > 0 {
		t.Errorf("%d dibit(s) outside 0..3 range under ClockGardner", bad)
	}
}

func TestReceiverGardnerDefaultGain(t *testing.T) {
	r := New(Options{
		SampleRateHz: 48_000,
		DibitSink:    func([]uint8, int) {},
		ClockMode:    ClockGardner,
	})
	if r.gardner == nil {
		t.Fatalf("ClockGardner did not construct a Gardner loop")
	}
}

func TestReceiverGardnerExplicitGain(t *testing.T) {
	r := New(Options{
		SampleRateHz: 48_000,
		DibitSink:    func([]uint8, int) {},
		ClockMode:    ClockGardner,
		GardnerGain:  0.05,
	})
	if r.gardner == nil {
		t.Fatalf("ClockGardner did not construct a Gardner loop")
	}
}

func TestReceiverClockNaiveSkipsGardnerConstruction(t *testing.T) {
	r := New(Options{
		SampleRateHz: 48_000,
		DibitSink:    func([]uint8, int) {},
	})
	if r.gardner != nil {
		t.Errorf("ClockNaive (default) constructed a Gardner loop")
	}
}

func TestReceiverGardnerResetClearsState(t *testing.T) {
	r := New(Options{
		SampleRateHz: 48_000,
		DibitSink:    func([]uint8, int) {},
		ClockMode:    ClockGardner,
	})
	dibits := []uint8{0b00, 0b01, 0b10, 0b11, 0b00, 0b01, 0b10, 0b11}
	iq, _ := makeP2HDQPSKIQ(dibits)
	r.Process(iq)
	r.Reset()
	// After Reset, the dibitBase counter must start back at zero
	// on the next emission.
	var firstBase int = -1
	r.dibitSink = func(d []uint8, baseIdx int) {
		if firstBase < 0 {
			firstBase = baseIdx
		}
	}
	r.Process(iq)
	if firstBase != 0 {
		t.Errorf("post-Reset first baseIdx = %d, want 0", firstBase)
	}
}
