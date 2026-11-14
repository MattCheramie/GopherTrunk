package conventional

import (
	"context"
	"math"
	"testing"
	"time"
)

// clickTrain is n "clicks": one loud 256-sample chunk followed by three
// silent ones — impulsive noise, nothing sustained.
func clickTrain(n int) [][]complex64 {
	silent := make([]complex64, 256)
	out := make([][]complex64, 0, 4*n)
	for i := 0; i < n; i++ {
		out = append(out, loudChunk(256), silent, silent, silent)
	}
	return out
}

// runOpenDelayScanner runs one channel carrying chunks with the given open
// delay and reports how many calls it opened within the window.
func runOpenDelayScanner(t *testing.T, chunks [][]complex64, delay time.Duration, window time.Duration) (*fakeEngine, *Scanner) {
	t.Helper()
	const hz = 150_000_000
	tuner := &fakeTuner{}
	iq := &fakeIQ{tuner: tuner, chunks: map[uint32][][]complex64{hz: chunks}}
	eng := &fakeEngine{}
	s, err := New(Options{
		Tuner: tuner, IQ: iq, Engine: eng, Recorder: fakeRecorder{},
		DeviceSerial: "CONV-1", MinDwellPerChannel: 30 * time.Millisecond,
		// 48 kHz: the in-channel power meter runs undecimated, and a
		// 256-sample chunk is 5.33 ms of stream time.
		SampleRateHz: 48_000,
		Channels: []Channel{{
			Label: "A", FrequencyHz: hz, SquelchDbFS: -10,
			Hangtime: 20 * time.Millisecond, SquelchOpenDelay: delay,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), window)
	defer cancel()
	_ = s.Run(ctx)
	return eng, s
}

// TestSquelchOpenDelayRejectsClicks is the #1321 follow-up feature request:
// on a channel with impulsive noise every click above squelch_dbfs used to
// open a hangtime-length "call". With squelch_open_ms the carrier must stay
// above the threshold that long before a call opens.
func TestSquelchOpenDelayRejectsClicks(t *testing.T) {
	// Default (0): the clicks open calls, as before — which is also what
	// makes the delayed case below meaningful.
	eng, _ := runOpenDelayScanner(t, clickTrain(60), 0, 400*time.Millisecond)
	if eng.startCount() == 0 {
		t.Fatal("with no open delay the click train opened no call: the fixture does not break squelch")
	}
	eng, _ = runOpenDelayScanner(t, clickTrain(60), 40*time.Millisecond, 400*time.Millisecond)
	if n := eng.startCount(); n != 0 {
		t.Errorf("squelch_open_ms=40 still opened %d call(s) on isolated clicks", n)
	}
}

// TestSquelchOpenDelayOpensOnSustainedCarrier: a real carrier still opens,
// even when the delay is longer than the channel's scan window (the window
// is extended while a run is in progress).
func TestSquelchOpenDelayOpensOnSustainedCarrier(t *testing.T) {
	eng, _ := runOpenDelayScanner(t, loudRun(60), 80*time.Millisecond, 600*time.Millisecond)
	if eng.startCount() == 0 {
		t.Fatal("a sustained carrier never opened with squelch_open_ms=80 (> the 30 ms scan window)")
	}
}

// TestSnapshotReportsChannelLevel: /api/v1/scanner carries the in-channel
// level and the threshold it is compared with, so an operator can set the
// squelch without computing it from the spectrum (#1321 follow-up).
func TestSnapshotReportsChannelLevel(t *testing.T) {
	_, s := runOpenDelayScanner(t, nil, 0, 150*time.Millisecond) // silence only
	snap := s.Snapshot()
	if len(snap.Channels) != 1 {
		t.Fatalf("channels = %d", len(snap.Channels))
	}
	ch := snap.Channels[0]
	if ch.LevelDbFS == nil {
		t.Fatal("level_dbfs not reported for a visited FM channel")
	}
	if !(*ch.LevelDbFS < -60) {
		t.Errorf("level_dbfs = %.1f on a silent channel, want far below the -10 dBFS squelch", *ch.LevelDbFS)
	}
	if ch.LevelAt.IsZero() {
		t.Error("level_at not set")
	}
	if ch.SquelchDbFS != -10 {
		t.Errorf("squelch_dbfs = %v, want -10", ch.SquelchDbFS)
	}

	_, s = runOpenDelayScanner(t, loudRun(200), 0, 150*time.Millisecond)
	ch = s.Snapshot().Channels[0]
	if ch.LevelDbFS == nil || math.Abs(*ch.LevelDbFS) > 3 {
		t.Errorf("level_dbfs on a 0 dBFS carrier = %v, want ≈0", ch.LevelDbFS)
	}
}
