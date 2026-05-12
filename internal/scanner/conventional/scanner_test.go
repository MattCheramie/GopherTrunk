package conventional

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

type fakeTuner struct {
	mu    sync.Mutex
	calls []uint32
}

func (f *fakeTuner) SetCenterFreq(hz uint32) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, hz)
	return nil
}

func (f *fakeTuner) tuned() []uint32 {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]uint32, len(f.calls))
	copy(out, f.calls)
	return out
}

// fakeIQ emits the supplied chunks per-channel keyed by frequency.
// When a frequency runs out of chunks it sends silence (zero IQ).
type fakeIQ struct {
	mu     sync.Mutex
	chunks map[uint32][][]complex64
	last   uint32
	tuner  *fakeTuner
}

func (f *fakeIQ) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	out := make(chan []complex64, 8)
	go func() {
		defer close(out)
		// Look up the most recent SetCenterFreq call to know which
		// frequency to source samples from.
		ticker := time.NewTicker(2 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				f.mu.Lock()
				if f.tuner != nil {
					f.tuner.mu.Lock()
					if len(f.tuner.calls) > 0 {
						f.last = f.tuner.calls[len(f.tuner.calls)-1]
					}
					f.tuner.mu.Unlock()
				}
				queue := f.chunks[f.last]
				var chunk []complex64
				if len(queue) > 0 {
					chunk = queue[0]
					f.chunks[f.last] = queue[1:]
				} else {
					// Silence.
					chunk = make([]complex64, 256)
				}
				f.mu.Unlock()
				select {
				case <-ctx.Done():
					return
				case out <- chunk:
				}
			}
		}
	}()
	return out, nil
}

type fakeEngine struct {
	mu       sync.Mutex
	starts   []trunking.Grant
	ends     []string
	touches  int
}

func (e *fakeEngine) HandleSyntheticCall(g trunking.Grant, deviceSerial string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.starts = append(e.starts, g)
}
func (e *fakeEngine) EndSyntheticCall(deviceSerial string, reason trunking.EndReason) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.ends = append(e.ends, deviceSerial)
	return true
}
func (e *fakeEngine) Touch(deviceSerial string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.touches++
}
func (e *fakeEngine) startCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.starts)
}
func (e *fakeEngine) endCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.ends)
}

type fakeRecorder struct{}

func (fakeRecorder) WritePCM(_ string, _ []int16) error { return nil }

func loudChunk(n int) []complex64 {
	out := make([]complex64, n)
	for i := range out {
		out[i] = complex(0.7, 0.7)
	}
	return out
}

func TestConvScannerHopsThroughEmptyChannels(t *testing.T) {
	tuner := &fakeTuner{}
	iq := &fakeIQ{chunks: map[uint32][][]complex64{}, tuner: tuner}
	eng := &fakeEngine{}
	s, err := New(Options{
		Tuner: tuner, IQ: iq, Engine: eng, Recorder: fakeRecorder{},
		DeviceSerial: "CONV-1",
		SystemName:   "test",
		Channels: []Channel{
			{Label: "A", FrequencyHz: 100_000_000, SquelchDbFS: -10},
			{Label: "B", FrequencyHz: 200_000_000, SquelchDbFS: -10},
			{Label: "C", FrequencyHz: 300_000_000, SquelchDbFS: -10},
		},
		MinDwellPerChannel: 30 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_ = s.Run(ctx)

	calls := tuner.tuned()
	if len(calls) < 3 {
		t.Fatalf("expected at least 3 tune calls in 500ms, got %d", len(calls))
	}
	// All three frequencies should appear in the hop sequence.
	seen := map[uint32]bool{}
	for _, hz := range calls {
		seen[hz] = true
	}
	if !seen[100_000_000] || !seen[200_000_000] || !seen[300_000_000] {
		t.Errorf("hop sequence missing channels: %v", calls)
	}
	if eng.startCount() != 0 {
		t.Errorf("no squelch break should fire on silence; saw %d starts", eng.startCount())
	}
}

func TestConvScannerBreaksSquelchAndEndsOnHangtime(t *testing.T) {
	tuner := &fakeTuner{}
	// Channel B (200 MHz) emits N loud chunks then silence; A and C
	// stay silent. Hangtime is short for test speed.
	iq := &fakeIQ{
		tuner: tuner,
		chunks: map[uint32][][]complex64{
			200_000_000: {
				loudChunk(256), loudChunk(256), loudChunk(256),
			},
		},
	}
	eng := &fakeEngine{}
	s, err := New(Options{
		Tuner: tuner, IQ: iq, Engine: eng, Recorder: fakeRecorder{},
		DeviceSerial: "CONV-1",
		SystemName:   "test",
		Channels: []Channel{
			{Label: "A", FrequencyHz: 100_000_000, SquelchDbFS: -10},
			{Label: "B", FrequencyHz: 200_000_000, SquelchDbFS: -10, Hangtime: 50 * time.Millisecond},
			{Label: "C", FrequencyHz: 300_000_000, SquelchDbFS: -10},
		},
		MinDwellPerChannel: 30 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = s.Run(ctx)

	if eng.startCount() == 0 {
		t.Fatal("squelch never fired (no HandleSyntheticCall)")
	}
	if eng.endCount() == 0 {
		t.Fatal("hangtime never expired (no EndSyntheticCall)")
	}
	// First start's frequency must match channel B.
	if eng.starts[0].FrequencyHz != 200_000_000 {
		t.Errorf("first start freq = %d, want 200_000_000", eng.starts[0].FrequencyHz)
	}
	if eng.starts[0].Protocol != "fm-conv" {
		t.Errorf("protocol = %q, want fm-conv", eng.starts[0].Protocol)
	}
}

func TestConvScannerHoldAndResume(t *testing.T) {
	s, err := New(Options{
		Tuner: &fakeTuner{}, IQ: &fakeIQ{tuner: &fakeTuner{}}, Engine: &fakeEngine{}, Recorder: fakeRecorder{},
		DeviceSerial: "CONV-1",
		Channels:     []Channel{{Label: "A", FrequencyHz: 1, SquelchDbFS: -10}},
	})
	if err != nil {
		t.Fatal(err)
	}
	s.Hold()
	if !s.IsHeld() {
		t.Error("IsHeld() = false after Hold()")
	}
	if got := s.Snapshot().State; got != StateHeld {
		t.Errorf("state = %q, want held", got)
	}
	s.Resume()
	if s.IsHeld() {
		t.Error("IsHeld() = true after Resume()")
	}
}

func TestConvScannerDwellOn(t *testing.T) {
	s, err := New(Options{
		Tuner: &fakeTuner{}, IQ: &fakeIQ{tuner: &fakeTuner{}}, Engine: &fakeEngine{}, Recorder: fakeRecorder{},
		DeviceSerial: "CONV-1",
		Channels: []Channel{
			{Label: "A", FrequencyHz: 1, SquelchDbFS: -10},
			{Label: "B", FrequencyHz: 2, SquelchDbFS: -10},
			{Label: "C", FrequencyHz: 3, SquelchDbFS: -10},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !s.DwellOn(2) {
		t.Fatal("DwellOn(2) = false")
	}
	if s.DwellOn(99) {
		t.Errorf("DwellOn(99) should fail")
	}
	// Next call to pickNextChannel must return 2.
	idx, _, ok := s.pickNextChannel()
	if !ok {
		t.Fatal("pickNextChannel returned no channel")
	}
	if idx != 2 {
		t.Errorf("pickNextChannel after DwellOn(2) = %d, want 2", idx)
	}
}

func TestAddTemporaryChannelGrowsAndForcesDwell(t *testing.T) {
	s, err := New(Options{
		Engine: &fakeEngine{}, Tuner: &fakeTuner{}, IQ: &fakeIQ{},
		Recorder: fakeRecorder{}, DeviceSerial: "x",
		Channels: []Channel{{Label: "static", FrequencyHz: 100}},
	})
	if err != nil {
		t.Fatal(err)
	}
	idx := s.AddTemporaryChannel(Channel{Label: "VFO", FrequencyHz: 155895000})
	if idx != 1 {
		t.Errorf("new idx = %d, want 1", idx)
	}
	snap := s.Snapshot()
	if len(snap.Channels) != 2 {
		t.Fatalf("Snapshot channels = %d, want 2", len(snap.Channels))
	}
	if snap.Channels[1].FrequencyHz != 155895000 {
		t.Errorf("VFO freq = %d, want 155895000", snap.Channels[1].FrequencyHz)
	}
	// AddTemporaryChannel must force the next pick to land on the new
	// channel (manual tune = "listen now" intent).
	got, _, ok := s.pickNextChannel()
	if !ok || got != 1 {
		t.Errorf("first pick after AddTemporaryChannel = %d ok=%t, want 1 ok=true", got, ok)
	}
}

func TestAddTemporaryChannelDefaultsAppliedOncePerChannel(t *testing.T) {
	s, err := New(Options{
		Engine: &fakeEngine{}, Tuner: &fakeTuner{}, IQ: &fakeIQ{},
		Recorder: fakeRecorder{}, DeviceSerial: "x",
		Channels: []Channel{{Label: "static", FrequencyHz: 100}},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Zero-valued knobs should default the same way config-seeded
	// channels do.
	s.AddTemporaryChannel(Channel{Label: "vfo", FrequencyHz: 1})
	snap := s.Snapshot()
	got := snap.Channels[1]
	if got.Mode != "fm" {
		t.Errorf("default mode = %q, want fm", got.Mode)
	}
}

func TestRemoveTemporaryChannelRevokesAndReindexes(t *testing.T) {
	s, err := New(Options{
		Engine: &fakeEngine{}, Tuner: &fakeTuner{}, IQ: &fakeIQ{},
		Recorder: fakeRecorder{}, DeviceSerial: "x",
		Channels: []Channel{{Label: "static", FrequencyHz: 100}},
	})
	if err != nil {
		t.Fatal(err)
	}
	idx := s.AddTemporaryChannel(Channel{Label: "VFO", FrequencyHz: 1})
	if !s.RemoveTemporaryChannel(idx) {
		t.Fatal("RemoveTemporaryChannel(1) returned false")
	}
	snap := s.Snapshot()
	if len(snap.Channels) != 1 {
		t.Errorf("after remove channels = %d, want 1", len(snap.Channels))
	}
	// Removing again returns false (not a temp channel anymore).
	if s.RemoveTemporaryChannel(idx) {
		t.Error("second remove should return false")
	}
	// Removing a static (config-seeded) channel returns false.
	if s.RemoveTemporaryChannel(0) {
		t.Error("removing static channel should return false")
	}
}

func TestRunWithEmptyChannelsIdlesUntilAdd(t *testing.T) {
	tnr := &fakeTuner{}
	s, err := New(Options{
		Engine: &fakeEngine{}, Tuner: tnr, IQ: &fakeIQ{tuner: tnr},
		Recorder: fakeRecorder{}, DeviceSerial: "x",
		// No Channels — operator only ever drives this via
		// AddTemporaryChannel from the TUI / API.
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		_ = s.Run(ctx)
		close(done)
	}()
	// Idle for a tick; no tune should happen with zero channels.
	time.Sleep(60 * time.Millisecond)
	if len(tnr.tuned()) != 0 {
		t.Errorf("empty-channels scanner tuned anyway: %v", tnr.tuned())
	}
	cancel()
	<-done
}

func TestConvScannerAcceptsEmptyChannels(t *testing.T) {
	// Distinct from TestConvScannerRejectsBadConfig — empty Channels
	// used to fail validation but now must succeed to support manual
	// VFO tune via AddTemporaryChannel.
	_, err := New(Options{
		Engine: &fakeEngine{}, Tuner: &fakeTuner{}, IQ: &fakeIQ{},
		Recorder: fakeRecorder{}, DeviceSerial: "x",
	})
	if err != nil {
		t.Fatalf("New with empty channels: %v", err)
	}
}

func TestConvScannerRejectsBadConfig(t *testing.T) {
	cases := []struct {
		name string
		opts Options
	}{
		{"no engine", Options{Tuner: &fakeTuner{}, IQ: &fakeIQ{}, Recorder: fakeRecorder{}, DeviceSerial: "x", Channels: []Channel{{FrequencyHz: 1}}}},
		{"no tuner", Options{Engine: &fakeEngine{}, IQ: &fakeIQ{}, Recorder: fakeRecorder{}, DeviceSerial: "x", Channels: []Channel{{FrequencyHz: 1}}}},
		{"no device serial", Options{Engine: &fakeEngine{}, Tuner: &fakeTuner{}, IQ: &fakeIQ{}, Recorder: fakeRecorder{}, Channels: []Channel{{FrequencyHz: 1}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := New(tc.opts); err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}
