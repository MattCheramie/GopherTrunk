package conventional

import (
	"context"
	"sync"
	"testing"
	"time"
)

// slowGate is a call gate that needs `need` of air before it reports a call
// and takes `slow` times real time to process it, like a real gate's
// receiver on a CPU that cannot keep up (a Pi under load, -race on CI).
type slowGate struct {
	mu      sync.Mutex
	rate    float64
	slow    float64
	need    time.Duration
	samples int
	present bool
}

func (g *slowGate) Process(iq []complex64) bool {
	time.Sleep(time.Duration(g.slow * float64(len(iq)) / g.rate * float64(time.Second)))
	g.mu.Lock()
	defer g.mu.Unlock()
	g.samples += len(iq)
	g.present = time.Duration(float64(g.samples)/g.rate*float64(time.Second)) >= g.need
	return g.present
}

func (g *slowGate) Present() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.present
}

func (g *slowGate) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.samples, g.present = 0, false
}

func (g *slowGate) gateName() string        { return "slow" }
func (g *slowGate) minDwell() time.Duration { return time.Second }

// A call gate's minDwell is the air it needs, and the scan window used to
// enforce it on the wall clock: a gate that runs slower than real time saw
// only part of its signal before the window closed, and the scanner moved
// on from a real call. This is how CI lost the nxdn48 scan-list call
// (NXDN48 needs 440-770 ms of signal inside a 1 s window) under -race.
// The window now extends until the gate has heard minDwell of signal.
func TestCallGateWindowCountsSignalTimeNotWallTime(t *testing.T) {
	const freq = 442_387_500
	const rate = 48_000.0
	sig := make([]complex64, int(2*rate))
	for i := range sig {
		sig[i] = 0.5
	}
	tuner := &fakeTuner{}
	eng := &fakeEngine{}
	src := &fakeIQ{chunks: map[uint32][][]complex64{freq: chunked(sig, 960)}, tuner: tuner}
	s, err := New(Options{
		Tuner: tuner, IQ: src, Engine: eng, Recorder: fakeRecorder{},
		DeviceSerial: "CONV-SLOW-GATE",
		Channels: []Channel{{
			Label: "Rptr", FrequencyHz: freq, Mode: ModeNXDN48,
			SquelchDbFS: -50, Hangtime: 200 * time.Millisecond,
		}},
		SampleRateHz:       rate,
		MinDwellPerChannel: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	// 600 ms of air at 3x real time: 1.8 s of wall clock, past the 1 s
	// minDwell the old window allowed.
	gate := &slowGate{rate: rate, slow: 3, need: 600 * time.Millisecond}
	s.mu.Lock()
	s.detectors[0] = gate
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	go func() { _ = s.Run(ctx) }()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) && eng.startCount() == 0 {
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	if eng.startCount() == 0 {
		t.Fatal("a call gate slower than real time never opened: the scan window closed on the wall clock before the gate had heard its minDwell of signal")
	}
}
