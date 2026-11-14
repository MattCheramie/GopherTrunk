package conventional

import (
	"context"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// loudRun is n copies of one loud chunk: a carrier that never drops.
func loudRun(n int) [][]complex64 {
	c := loudChunk(256)
	out := make([][]complex64, n)
	for i := range out {
		out[i] = c
	}
	return out
}

// waitFor polls cond every 5 ms until it holds or d elapses.
func waitFor(d time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return cond()
}

// TestRemoveTemporaryChannelEndsDwellInProgress is the regression for the
// #1321 follow-up (13 Nov): DELETE /api/v1/scanner/manual_tune/<i> on a
// temp channel that was dwelling left the scanner on the OLD channel. The
// remove ended the engine call and reset the snapshot, but the dwell loop in
// the Run goroutine kept reading the old stream with the old frequency and
// squelch until hangtime — with the squelch under the noise floor, forever —
// so a replacement manual_tune was never tuned.
func TestRemoveTemporaryChannelEndsDwellInProgress(t *testing.T) {
	const oldHz, newHz = 150_000_000, 160_000_000
	tuner := &fakeTuner{}
	iq := &fakeIQ{tuner: tuner, chunks: map[uint32][][]complex64{
		oldHz: loudRun(5000), // ~10 s of carrier: never drops during the test
	}}
	eng := &fakeEngine{}
	s, err := New(Options{
		Tuner: tuner, IQ: iq, Engine: eng, Recorder: fakeRecorder{},
		DeviceSerial: "CONV-1", MinDwellPerChannel: 30 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { _ = s.Run(ctx); close(done) }()
	defer func() { cancel(); <-done }()

	old := s.AddTemporaryChannel(Channel{Label: "VFO", FrequencyHz: oldHz, SquelchDbFS: -10, Hangtime: 5 * time.Second})
	if !waitFor(time.Second, func() bool { return eng.startCount() == 1 }) {
		t.Fatal("temp channel never opened a dwell")
	}
	if !s.RemoveTemporaryChannel(old) {
		t.Fatal("RemoveTemporaryChannel returned false")
	}
	s.AddTemporaryChannel(Channel{Label: "VFO", FrequencyHz: newHz, SquelchDbFS: -10})

	if !waitFor(time.Second, func() bool {
		tuned := tuner.tuned()
		return len(tuned) > 0 && tuned[len(tuned)-1] == newHz
	}) {
		t.Fatalf("scanner never tuned the replacement channel; tunes=%v state=%s", tuner.tuned(), s.Snapshot().State)
	}
	// The removed dwell must not open another call on the old carrier, and
	// the remove's EndReasonManual must be the only end for that call.
	time.Sleep(100 * time.Millisecond)
	if n := eng.startCount(); n != 1 {
		t.Errorf("calls started = %d, want 1 (the removed channel re-opened?)", n)
	}
	reasons := eng.endReasonsCopy()
	if len(reasons) != 1 || reasons[0] != trunking.EndReasonManual {
		t.Errorf("end reasons = %v, want exactly [manual]", reasons)
	}
	if st := s.Snapshot().State; st != StateScanning {
		t.Errorf("state = %s, want scanning", st)
	}
}

// TestLockoutChannelEndsDwellInProgress pins LockoutChannel's documented
// promise ("takes effect within one IQ chunk rather than waiting for
// hangtime"). It ended the engine call but, like RemoveTemporaryChannel, left
// the dwell loop on the locked-out channel until hangtime.
func TestLockoutChannelEndsDwellInProgress(t *testing.T) {
	const aHz, bHz = 150_000_000, 160_000_000
	tuner := &fakeTuner{}
	iq := &fakeIQ{tuner: tuner, chunks: map[uint32][][]complex64{aHz: loudRun(5000)}}
	eng := &fakeEngine{}
	s, err := New(Options{
		Tuner: tuner, IQ: iq, Engine: eng, Recorder: fakeRecorder{},
		DeviceSerial: "CONV-1", MinDwellPerChannel: 30 * time.Millisecond,
		Channels: []Channel{
			{Label: "A", FrequencyHz: aHz, SquelchDbFS: -10, Hangtime: 5 * time.Second},
			{Label: "B", FrequencyHz: bHz, SquelchDbFS: -10},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { _ = s.Run(ctx); close(done) }()
	defer func() { cancel(); <-done }()

	if !waitFor(time.Second, func() bool { return eng.startCount() == 1 }) {
		t.Fatal("channel A never opened a dwell")
	}
	if !s.LockoutChannel(0) {
		t.Fatal("LockoutChannel returned false")
	}
	if !waitFor(time.Second, func() bool {
		tuned := tuner.tuned()
		return len(tuned) > 0 && tuned[len(tuned)-1] == bHz
	}) {
		t.Fatalf("scanner stayed on the locked-out channel; tunes=%v", tuner.tuned())
	}
	time.Sleep(100 * time.Millisecond)
	if n := eng.startCount(); n != 1 {
		t.Errorf("calls started = %d, want 1", n)
	}
	if r := eng.endReasonsCopy(); len(r) != 1 || r[0] != trunking.EndReasonLockout {
		t.Errorf("end reasons = %v, want exactly [lockout]", r)
	}
}

// TestRemoveTemporaryChannelDuringScanWindowOpensNoCall: a temp channel
// removed while its scan window is still measuring must not go on to open a
// call (on a freed, possibly reused, index) when its squelch breaks.
func TestRemoveTemporaryChannelDuringScanWindowOpensNoCall(t *testing.T) {
	const hz = 150_000_000
	tuner := &fakeTuner{}
	// Silence first, so the window is measuring when the remove lands,
	// then a carrier the old window would have broken squelch on.
	chunks := append(make([][]complex64, 0, 5100), func() [][]complex64 {
		q := make([][]complex64, 40)
		for i := range q {
			q[i] = make([]complex64, 256)
		}
		return q
	}()...)
	iq := &fakeIQ{tuner: tuner, chunks: map[uint32][][]complex64{hz: append(chunks, loudRun(5000)...)}}
	eng := &fakeEngine{}
	s, err := New(Options{
		Tuner: tuner, IQ: iq, Engine: eng, Recorder: fakeRecorder{},
		DeviceSerial: "CONV-1", MinDwellPerChannel: 2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { _ = s.Run(ctx); close(done) }()
	defer func() { cancel(); <-done }()

	idx := s.AddTemporaryChannel(Channel{Label: "VFO", FrequencyHz: hz, SquelchDbFS: -10})
	if !waitFor(time.Second, func() bool { return len(tuner.tuned()) == 1 }) {
		t.Fatal("temp channel never tuned")
	}
	if !s.RemoveTemporaryChannel(idx) {
		t.Fatal("RemoveTemporaryChannel returned false")
	}
	time.Sleep(300 * time.Millisecond) // well past the silence, into the carrier
	if n := eng.startCount(); n != 0 {
		t.Errorf("a removed channel opened %d call(s)", n)
	}
}
