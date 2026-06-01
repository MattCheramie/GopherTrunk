package iqtap

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// fakeDevice is a programmable in-memory sdr.Device for broker tests.
// It exposes a hand-driven channel: tests send chunks on Out and the
// device forwards them to whatever consumer called StreamIQ.
type fakeDevice struct {
	serial string

	mu         sync.Mutex
	Out        chan []complex64
	streamErr  error
	streamReqs int // number of StreamIQ calls received

	centerFreq atomic.Uint32
	sampleRate atomic.Uint32
	gain       atomic.Int32
	closes     atomic.Int32
}

func newFakeDevice(serial string) *fakeDevice {
	return &fakeDevice{serial: serial, Out: make(chan []complex64, 4)}
}

func (f *fakeDevice) Info() sdr.Info {
	return sdr.Info{Driver: "fake", Serial: f.serial}
}

func (f *fakeDevice) SetCenterFreq(hz uint32) error { f.centerFreq.Store(hz); return nil }
func (f *fakeDevice) SetSampleRate(hz uint32) error { f.sampleRate.Store(hz); return nil }
func (f *fakeDevice) SetGain(g int) error           { f.gain.Store(int32(g)); return nil }
func (f *fakeDevice) SetPPM(int) error              { return nil }
func (f *fakeDevice) SetBiasTee(bool) error         { return nil }
func (f *fakeDevice) Close() error                  { f.closes.Add(1); return nil }

func (f *fakeDevice) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	f.mu.Lock()
	f.streamReqs++
	err := f.streamErr
	out := f.Out
	f.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (f *fakeDevice) RotateChannel() {
	f.mu.Lock()
	close(f.Out)
	f.Out = make(chan []complex64, 4)
	f.mu.Unlock()
}

func TestBrokerForwardsPrimaryWithoutSubscribers(t *testing.T) {
	dev := newFakeDevice("dev-a")
	b := New(dev, 0, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out, err := b.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}

	want := []complex64{complex(1, 0), complex(2, 0), complex(3, 0)}
	dev.Out <- want
	got, ok := <-out
	if !ok {
		t.Fatal("primary channel closed unexpectedly")
	}
	if len(got) != len(want) {
		t.Fatalf("primary chunk len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("primary[%d] = %v, want %v", i, got[i], want[i])
		}
	}

	// Primary is zero-copy: the slice the primary received is the
	// same backing array the test sent. Mutating it must NOT affect
	// any future deliveries (no subscribers exist).
	got[0] = complex(99, 0)
}

func TestBrokerFansOutToSubscribers(t *testing.T) {
	dev := newFakeDevice("dev-b")
	b := New(dev, 0, nil)
	defer b.Close()

	subA := b.Subscribe()
	subB := b.Subscribe()
	defer subA.Close()
	defer subB.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out, err := b.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}

	chunk := []complex64{complex(1, 1), complex(2, 2)}
	dev.Out <- chunk

	// Primary must receive the original chunk.
	got := <-out
	if len(got) != len(chunk) {
		t.Fatalf("primary len = %d, want %d", len(got), len(chunk))
	}

	// Both subscribers must receive a copy.
	for name, sub := range map[string]*Subscriber{"A": subA, "B": subB} {
		select {
		case s := <-sub.C:
			if len(s) != len(chunk) {
				t.Errorf("sub %s len = %d, want %d", name, len(s), len(chunk))
			}
			for i := range chunk {
				if s[i] != chunk[i] {
					t.Errorf("sub %s [%d] = %v, want %v", name, i, s[i], chunk[i])
				}
			}
		case <-time.After(time.Second):
			t.Errorf("sub %s did not receive within 1s", name)
		}
	}
}

func TestBrokerSubscriberCopiesAreIndependent(t *testing.T) {
	dev := newFakeDevice("dev-c")
	b := New(dev, 0, nil)

	sub := b.Subscribe()
	defer sub.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out, err := b.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}

	original := []complex64{complex(1, 1), complex(2, 2), complex(3, 3)}
	dev.Out <- original

	primary := <-out
	copy1 := <-sub.C

	// Mutating the subscriber's copy must not touch the primary's
	// slice nor the next subscriber's copy.
	copy1[0] = complex(0, 0)
	if primary[0] == copy1[0] {
		t.Error("subscriber copy shares backing array with primary")
	}

	// Send a second chunk; subscriber's next copy must reflect the
	// device's data, not the mutated previous copy.
	original2 := []complex64{complex(4, 4), complex(5, 5)}
	dev.Out <- original2
	<-out
	copy2 := <-sub.C
	if copy2[0] != complex(4, 4) {
		t.Errorf("second copy[0] = %v, want (4+4i)", copy2[0])
	}
}

func TestBrokerSlowSubscriberDoesNotBlockPrimary(t *testing.T) {
	dev := newFakeDevice("dev-d")
	// Tiny subscriber buffer (2 chunks) so we can overflow it fast.
	b := New(dev, 2, nil)

	sub := b.Subscribe()
	// Deliberately don't drain sub.C.

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out, err := b.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}

	// Drain primary as fast as possible while flooding more chunks
	// than the subscriber can hold.
	const N = 50
	go func() {
		for i := 0; i < N; i++ {
			dev.Out <- []complex64{complex(float32(i), 0)}
		}
	}()

	deadline := time.After(2 * time.Second)
	for i := 0; i < N; i++ {
		select {
		case <-out:
		case <-deadline:
			t.Fatalf("primary stalled at chunk %d — slow subscriber must not block primary", i)
		}
	}

	// Subscriber should have dropped some chunks.
	if sub.Dropped() == 0 {
		t.Error("expected non-zero drops on slow subscriber")
	}
	if b.Stats().DroppedTotal == 0 {
		t.Error("expected non-zero broker DroppedTotal")
	}
	sub.Close()
}

func TestBrokerSubscriberCloseStopsDelivery(t *testing.T) {
	dev := newFakeDevice("dev-e")
	b := New(dev, 0, nil)

	sub := b.Subscribe()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out, err := b.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}

	dev.Out <- []complex64{complex(1, 0)}
	<-out
	if _, ok := <-sub.C; !ok {
		t.Fatal("subscriber channel closed before Close() called")
	}

	sub.Close()

	// Channel must be closed.
	if _, ok := <-sub.C; ok {
		t.Error("subscriber channel still open after Close")
	}

	// Subsequent chunks deliver to primary but not to the closed sub.
	dev.Out <- []complex64{complex(2, 0)}
	<-out

	if b.Stats().Subscribers != 0 {
		t.Errorf("Stats.Subscribers = %d, want 0", b.Stats().Subscribers)
	}

	// Close idempotency.
	sub.Close()
}

func TestBrokerSubscribersPersistAcrossStreamSessions(t *testing.T) {
	dev := newFakeDevice("dev-f")
	b := New(dev, 0, nil)

	sub := b.Subscribe()
	defer sub.Close()

	// Session 1.
	ctx1, cancel1 := context.WithCancel(context.Background())
	out1, err := b.StreamIQ(ctx1)
	if err != nil {
		t.Fatalf("StreamIQ #1: %v", err)
	}
	dev.Out <- []complex64{complex(1, 0)}
	<-out1
	<-sub.C

	// End session 1 by closing the device channel.
	dev.RotateChannel()
	cancel1()
	// Drain primary's close.
	for range out1 {
	}

	// Session 2 — fresh StreamIQ call on the same broker.
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	out2, err := b.StreamIQ(ctx2)
	if err != nil {
		t.Fatalf("StreamIQ #2: %v", err)
	}

	dev.Out <- []complex64{complex(2, 0)}
	if _, ok := <-out2; !ok {
		t.Fatal("primary #2 closed unexpectedly")
	}
	select {
	case s, ok := <-sub.C:
		if !ok {
			t.Fatal("subscriber channel closed across sessions")
		}
		if s[0] != complex(2, 0) {
			t.Errorf("subscriber session-2 chunk = %v, want (2+0i)", s[0])
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber did not receive session-2 chunk")
	}
}

func TestBrokerSetInnerSwapsDevice(t *testing.T) {
	devA := newFakeDevice("dev-g-1")
	devB := newFakeDevice("dev-g-2")

	b := New(devA, 0, nil)

	if got := b.Info().Serial; got != "dev-g-1" {
		t.Errorf("Info().Serial = %q, want dev-g-1", got)
	}

	b.SetInner(devB)

	if got := b.Info().Serial; got != "dev-g-2" {
		t.Errorf("after SetInner, Info().Serial = %q, want dev-g-2", got)
	}

	// SetCenterFreq must hit the new inner, not the old one.
	if err := b.SetCenterFreq(123_000_000); err != nil {
		t.Fatalf("SetCenterFreq: %v", err)
	}
	if got := devB.centerFreq.Load(); got != 123_000_000 {
		t.Errorf("devB.centerFreq = %d, want 123000000", got)
	}
	if got := devA.centerFreq.Load(); got != 0 {
		t.Errorf("devA.centerFreq = %d, want 0 (untouched after SetInner)", got)
	}
}

func TestBrokerStreamIQPropagatesError(t *testing.T) {
	dev := newFakeDevice("dev-h")
	dev.streamErr = errors.New("usb gone")

	b := New(dev, 0, nil)

	_, err := b.StreamIQ(context.Background())
	if err == nil || err.Error() != "usb gone" {
		t.Errorf("StreamIQ err = %v, want usb gone", err)
	}
}

func TestBrokerStreamIQRefusesNilInner(t *testing.T) {
	b := New(nil, 0, nil)
	_, err := b.StreamIQ(context.Background())
	if err == nil {
		t.Fatal("StreamIQ with nil inner: want error, got nil")
	}
}

func TestBrokerStatsReflectsState(t *testing.T) {
	dev := newFakeDevice("dev-i")
	b := New(dev, 0, nil)

	if s := b.Stats(); s.Subscribers != 0 || s.Streaming {
		t.Errorf("initial Stats = %+v, want zero", s)
	}

	subA := b.Subscribe()
	subB := b.Subscribe()
	if got := b.Stats().Subscribers; got != 2 {
		t.Errorf("Subscribers = %d, want 2", got)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if _, err := b.StreamIQ(ctx); err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}
	// Briefly give the goroutine a turn so streamActive flips.
	deadline := time.Now().Add(time.Second)
	for !b.Stats().Streaming {
		if time.Now().After(deadline) {
			t.Fatal("Streaming never became true")
		}
		time.Sleep(time.Millisecond)
	}

	subA.Close()
	subB.Close()
	if got := b.Stats().Subscribers; got != 0 {
		t.Errorf("after Close, Subscribers = %d, want 0", got)
	}
}

// TestBrokerSurvivesSetInnerAcrossSessions models the pool.Reacquire
// path: a primary stream ends (USB disconnect simulated by closing the
// inner channel), the daemon calls SetInner with a fresh device, and a
// fresh StreamIQ on the broker reads from the new device while
// secondary subscribers continue receiving uninterrupted.
func TestBrokerSurvivesSetInnerAcrossSessions(t *testing.T) {
	devA := newFakeDevice("dev-old")
	devB := newFakeDevice("dev-new")
	b := New(devA, 0, nil)

	sub := b.Subscribe()
	defer sub.Close()

	// Session 1 — old device.
	ctx1, cancel1 := context.WithCancel(context.Background())
	out1, err := b.StreamIQ(ctx1)
	if err != nil {
		t.Fatalf("StreamIQ #1: %v", err)
	}
	devA.Out <- []complex64{complex(1, 0)}
	<-out1
	if got := (<-sub.C)[0]; got != complex(1, 0) {
		t.Errorf("sub session-1 = %v, want (1+0i)", got)
	}

	// Simulate USB disconnect: close the old device's channel and the
	// in-flight StreamIQ goroutine winds down.
	devA.RotateChannel()
	cancel1()
	for range out1 {
	}

	// Daemon does its Reacquire dance: swap broker inner to the new
	// device.
	b.SetInner(devB)
	if b.Inner() != devB {
		t.Fatal("Inner() did not reflect SetInner")
	}

	// Session 2 — new device.
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	out2, err := b.StreamIQ(ctx2)
	if err != nil {
		t.Fatalf("StreamIQ #2: %v", err)
	}
	devB.Out <- []complex64{complex(2, 0)}
	if got := <-out2; got[0] != complex(2, 0) {
		t.Errorf("primary session-2 = %v, want (2+0i)", got[0])
	}
	if got := (<-sub.C)[0]; got != complex(2, 0) {
		t.Errorf("sub session-2 = %v, want (2+0i)", got)
	}
}

func TestBrokerSetterPassThroughs(t *testing.T) {
	dev := newFakeDevice("dev-j")
	b := New(dev, 0, nil)

	if err := b.SetCenterFreq(851_012_500); err != nil {
		t.Fatalf("SetCenterFreq: %v", err)
	}
	if got := dev.centerFreq.Load(); got != 851_012_500 {
		t.Errorf("centerFreq = %d", got)
	}

	if err := b.SetSampleRate(2_048_000); err != nil {
		t.Fatalf("SetSampleRate: %v", err)
	}
	if got := dev.sampleRate.Load(); got != 2_048_000 {
		t.Errorf("sampleRate = %d", got)
	}

	if err := b.SetGain(420); err != nil {
		t.Fatalf("SetGain: %v", err)
	}
	if got := dev.gain.Load(); got != 420 {
		t.Errorf("gain = %d", got)
	}

	if err := b.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if got := dev.closes.Load(); got != 1 {
		t.Errorf("close count = %d, want 1", got)
	}
}

func TestBrokerSeedPopulatesCacheWithoutInnerCall(t *testing.T) {
	dev := newFakeDevice("dev-seed")
	b := New(dev, 0, nil)

	b.Seed(0, 2_048_000)

	if got := b.SampleRateHz(); got != 2_048_000 {
		t.Errorf("SampleRateHz after Seed = %d, want 2_048_000", got)
	}
	if got := b.CenterHz(); got != 0 {
		t.Errorf("CenterHz after Seed(0, ...) = %d, want 0 (untouched)", got)
	}
	if got := dev.sampleRate.Load(); got != 0 {
		t.Errorf("inner SetSampleRate was called: dev.sampleRate = %d, want 0", got)
	}
	if got := dev.centerFreq.Load(); got != 0 {
		t.Errorf("inner SetCenterFreq was called: dev.centerFreq = %d, want 0", got)
	}
}

func TestBrokerSeedZeroLeavesFieldsUnchanged(t *testing.T) {
	dev := newFakeDevice("dev-seed-zero")
	b := New(dev, 0, nil)

	if err := b.SetCenterFreq(100_000_000); err != nil {
		t.Fatalf("SetCenterFreq: %v", err)
	}
	if err := b.SetSampleRate(2_400_000); err != nil {
		t.Fatalf("SetSampleRate: %v", err)
	}

	b.Seed(0, 0)

	if got := b.CenterHz(); got != 100_000_000 {
		t.Errorf("CenterHz after Seed(0,0) = %d, want 100_000_000", got)
	}
	if got := b.SampleRateHz(); got != 2_400_000 {
		t.Errorf("SampleRateHz after Seed(0,0) = %d, want 2_400_000", got)
	}

	// Now seed only the rate to a new value; center must remain
	// unchanged because the new value's zero arg is a sentinel.
	b.Seed(0, 1_024_000)
	if got := b.CenterHz(); got != 100_000_000 {
		t.Errorf("CenterHz after Seed(0, 1_024_000) = %d, want 100_000_000 (untouched)", got)
	}
	if got := b.SampleRateHz(); got != 1_024_000 {
		t.Errorf("SampleRateHz after Seed(0, 1_024_000) = %d, want 1_024_000", got)
	}
}

// TestBrokerFanoutCloseRace exercises the concurrent-Close-vs-fanout path
// that previously panicked with "send on closed channel": fanout snapshots
// the subscriber set and sends after dropping subsMu, so a Close that ran
// close(s.ch) between the closed-flag check and the send would crash. With
// trySend / Close sharing s.sendMu the send can never land on a closed
// channel. Run under `go test -race` to also surface data races.
func TestBrokerFanoutCloseRace(t *testing.T) {
	dev := newFakeDevice("dev-race")
	b := New(dev, 1, nil) // tiny buffer so fanout frequently hits the send

	chunk := []complex64{complex(1, 0), complex(2, 0)}

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Fanout hammer: repeatedly fan the same chunk out to whatever
	// subscribers currently exist. This is the goroutine that used to
	// panic.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				b.fanout(chunk)
			}
		}
	}()

	// Churn: continuously Subscribe, (maybe) read, and Close so a Close
	// races concurrent fanout sends. A second goroutine drains some
	// subscribers to keep buffers cycling.
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			deadline := time.Now().Add(200 * time.Millisecond)
			for time.Now().Before(deadline) {
				s := b.Subscribe()
				// Drain a few chunks then close; also exercise the
				// drop path by sometimes not draining at all.
				done := make(chan struct{})
				go func() {
					for range s.C {
					}
					close(done)
				}()
				time.Sleep(time.Millisecond)
				s.Close()
				<-done
				// Close again to assert idempotency under concurrency.
				s.Close()
			}
		}()
	}

	time.Sleep(220 * time.Millisecond)
	close(stop)
	wg.Wait()
}
