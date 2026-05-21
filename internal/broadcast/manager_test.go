package broadcast

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// fakeBackend records the calls it receives and can be told to fail a
// fixed number of leading Send attempts.
type fakeBackend struct {
	name      string
	filter    systemFilter
	failFirst int

	mu       sync.Mutex
	attempts int
	got      []*Call
}

func (f *fakeBackend) Name() string          { return f.name }
func (f *fakeBackend) Accepts(s string) bool { return f.filter.Accepts(s) }
func (f *fakeBackend) Send(_ context.Context, c *Call) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.attempts++
	if f.attempts <= f.failFirst {
		return errors.New("transient failure")
	}
	f.got = append(f.got, c)
	return nil
}
func (f *fakeBackend) delivered() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.got)
}
func (f *fakeBackend) tries() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.attempts
}

// waitFor polls cond until it holds or the deadline elapses.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met before deadline")
}

// newTestManager builds a Manager wired to bus with fast retries and
// runs its event loop until the returned cancel is called.
func newTestManager(t *testing.T, bus *events.Bus, opts Options) *Manager {
	t.Helper()
	opts.Bus = bus
	opts.RetryBase = time.Millisecond
	m, err := NewManager(opts)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go m.Run(ctx)
	t.Cleanup(func() {
		cancel()
		m.Close()
	})
	return m
}

func completeEvent(system string, tg uint32, audioPath string, dur time.Duration, tgRec *trunking.TalkGroup) events.Event {
	now := time.Now()
	return events.Event{
		Kind: events.KindCallComplete,
		Payload: trunking.CallComplete{
			Grant:      trunking.Grant{System: system, GroupID: tg},
			Talkgroup:  tgRec,
			StartedAt:  now.Add(-dur),
			EndedAt:    now,
			AudioPath:  audioPath,
			SampleRate: 8000,
		},
	}
}

func TestManagerStreamsCompletedCall(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	be := &fakeBackend{name: "fake"}
	m := newTestManager(t, bus, Options{Backends: []Backend{be}})

	bus.Publish(completeEvent("Metro", 100, writeWAV(t, 0.4), 2*time.Second, nil))
	waitFor(t, func() bool { return be.delivered() == 1 })

	if got := be.got[0]; got.System != "Metro" || got.Talkgroup != 100 {
		t.Fatalf("delivered wrong call: %+v", got)
	}
	if s := m.Stats(); s.Sent["fake"] != 1 {
		t.Fatalf("stats Sent[fake] = %d, want 1", s.Sent["fake"])
	}
}

func TestManagerSkipsStreamFalseTalkgroup(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	be := &fakeBackend{name: "fake"}
	newTestManager(t, bus, Options{Backends: []Backend{be}})

	muted := &trunking.TalkGroup{ID: 200, Stream: false}
	bus.Publish(completeEvent("Metro", 200, writeWAV(t, 0.4), 2*time.Second, muted))
	// Give the event loop time to process and (correctly) drop it.
	time.Sleep(200 * time.Millisecond)
	if be.delivered() != 0 {
		t.Fatal("call on a Stream=false talkgroup must not be streamed")
	}
}

func TestManagerDropsShortCalls(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	be := &fakeBackend{name: "fake"}
	newTestManager(t, bus, Options{Backends: []Backend{be}, MinDuration: time.Second})

	bus.Publish(completeEvent("Metro", 300, writeWAV(t, 0.4), 250*time.Millisecond, nil))
	time.Sleep(200 * time.Millisecond)
	if be.delivered() != 0 {
		t.Fatal("call shorter than MinDuration must be dropped")
	}
}

func TestManagerRespectsSystemFilter(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	be := &fakeBackend{name: "fake", filter: newSystemFilter([]string{"Alpha"})}
	newTestManager(t, bus, Options{Backends: []Backend{be}})

	bus.Publish(completeEvent("Bravo", 400, writeWAV(t, 0.4), 2*time.Second, nil))
	time.Sleep(200 * time.Millisecond)
	if be.delivered() != 0 {
		t.Fatal("backend received a call from an unlisted system")
	}

	bus.Publish(completeEvent("Alpha", 401, writeWAV(t, 0.4), 2*time.Second, nil))
	waitFor(t, func() bool { return be.delivered() == 1 })
}

func TestManagerRetriesTransientFailure(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	be := &fakeBackend{name: "flaky", failFirst: 2}
	m := newTestManager(t, bus, Options{Backends: []Backend{be}, MaxRetries: 3})

	bus.Publish(completeEvent("Metro", 500, writeWAV(t, 0.4), 2*time.Second, nil))
	waitFor(t, func() bool { return be.delivered() == 1 })
	if be.tries() != 3 {
		t.Fatalf("attempts = %d, want 3 (2 failures + 1 success)", be.tries())
	}
	if s := m.Stats(); s.Sent["flaky"] != 1 || s.Failed["flaky"] != 0 {
		t.Fatalf("stats wrong after eventual success: %+v", s)
	}
}

func TestManagerGivesUpAfterMaxRetries(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	be := &fakeBackend{name: "dead", failFirst: 99}
	m := newTestManager(t, bus, Options{Backends: []Backend{be}, MaxRetries: 2})

	bus.Publish(completeEvent("Metro", 600, writeWAV(t, 0.4), 2*time.Second, nil))
	waitFor(t, func() bool { return m.Stats().Failed["dead"] == 1 })
	if be.tries() != 3 {
		t.Fatalf("attempts = %d, want 3 (MaxRetries 2 + initial)", be.tries())
	}
}

func TestManagerRequiresBus(t *testing.T) {
	if _, err := NewManager(Options{}); err == nil {
		t.Fatal("NewManager without a bus should error")
	}
}
