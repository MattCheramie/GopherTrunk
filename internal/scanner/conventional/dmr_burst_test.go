package conventional

import (
	"context"
	"testing"
	"time"
)

// dmrDirectModeQueue models a direct-mode / simplex DMR over at the fake
// source's 2 ms chunk cadence: one 28 ms burst (14 loud chunks) per 60 ms
// TDMA frame, off for the other 32 ms (16 silent chunks), for frames
// frames, then silence.
func dmrDirectModeQueue(frames int) [][]complex64 {
	var q [][]complex64
	for range frames {
		for range 14 {
			q = append(q, loudChunk(256))
		}
		for range 16 {
			q = append(q, make([]complex64, 256))
		}
	}
	return q
}

// TestConvScannerDMRDirectModeOverIsOneCall is the issue #1261 field
// regression: a DMR handheld on a simplex channel produced a string of
// 1.5–2.5 s recordings for one over ("DMR jumps out of decode even with a
// good signal, the test audio is segmented"). The power squelch fell in
// every 32 ms TDMA gap and no 28 ms burst could outlast the 50 ms
// activity debounce, so hangtime expired mid-over, over and over. One
// over must be one call.
func TestConvScannerDMRDirectModeOverIsOneCall(t *testing.T) {
	const freq = 438_100_000
	tuner := &fakeTuner{}
	// 40 frames = 2.4 s on air at 60 ms per frame, 8× the hangtime.
	iq := &fakeIQ{tuner: tuner, chunks: map[uint32][][]complex64{freq: dmrDirectModeQueue(40)}}
	eng := &fakeEngine{}
	s, err := New(Options{
		Tuner: tuner, IQ: iq, Engine: eng, Recorder: fakeRecorder{},
		DeviceSerial: "CONV-DMR-DM",
		SystemName:   "test",
		Channels: []Channel{{
			Label: "Simplex", FrequencyHz: freq, Mode: ModeDMR,
			SquelchDbFS: -10, Hangtime: 300 * time.Millisecond,
		}},
		MinDwellPerChannel: 20 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = s.Run(ctx) }()
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) && eng.normalEndCount() == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	// Let a split (if any) re-open before counting.
	time.Sleep(200 * time.Millisecond)
	cancel()
	if got := eng.startCount(); got != 1 {
		t.Fatalf("one DMR direct-mode over opened %d calls, want 1 (end reasons %v)", got, eng.endReasonsCopy())
	}
	if eng.normalEndCount() != 1 {
		t.Fatalf("over never released on hangtime after the carrier went away (end reasons %v)", eng.endReasonsCopy())
	}
}

// TestConvScannerAnalogDoesNotBridgeGaps: the bridge is DMR-only. The
// same burst pattern on an analog FM channel is not a TDMA signal, and
// the #1090 rule (brief activity must not hold squelch open) still holds
// there, so the call releases on hangtime while the pattern runs.
func TestConvScannerAnalogDoesNotBridgeGaps(t *testing.T) {
	const freq = 200_000_000
	tuner := &fakeTuner{}
	iq := &fakeIQ{tuner: tuner, chunks: map[uint32][][]complex64{freq: dmrDirectModeQueue(40)}}
	eng := &fakeEngine{}
	s, err := New(Options{
		Tuner: tuner, IQ: iq, Engine: eng, Recorder: fakeRecorder{},
		DeviceSerial: "CONV-FM",
		SystemName:   "test",
		Channels: []Channel{{
			Label: "FM", FrequencyHz: freq,
			SquelchDbFS: -10, Hangtime: 300 * time.Millisecond,
		}},
		MinDwellPerChannel: 20 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.Run(ctx)
	if eng.startCount() < 2 {
		t.Fatalf("analog channel opened %d call(s) over a 2.4 s gapped pattern with a 300 ms hangtime; the gap bridge must stay DMR-only", eng.startCount())
	}
}
