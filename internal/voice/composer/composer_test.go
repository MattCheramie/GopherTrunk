package composer

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// fakeSource implements IQSource. Each call to StreamIQ returns a
// fresh channel; the test pushes IQ chunks through SendIQ.
type fakeSource struct {
	mu  sync.Mutex
	chs []chan []complex64
}

func newFakeSource() *fakeSource { return &fakeSource{} }

func (f *fakeSource) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	ch := make(chan []complex64, 8)
	f.mu.Lock()
	f.chs = append(f.chs, ch)
	f.mu.Unlock()
	go func() {
		<-ctx.Done()
		// Don't close on cancel — Composer notices ctx.Done first; closing
		// would risk a double-close if multiple paths do it.
	}()
	return ch, nil
}

func (f *fakeSource) SendIQ(samples []complex64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, ch := range f.chs {
		select {
		case ch <- samples:
		default:
		}
	}
}

type fakeDevices struct{ src map[string]IQSource }

func (d *fakeDevices) FindBySerial(s string) IQSource { return d.src[s] }

type recordingSink struct {
	mu   sync.Mutex
	pcm  map[string][]int16
}

func (r *recordingSink) WritePCM(serial string, samples []int16) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pcm == nil {
		r.pcm = make(map[string][]int16)
	}
	r.pcm[serial] = append(r.pcm[serial], samples...)
	return nil
}

func (r *recordingSink) total(serial string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.pcm[serial])
}

type fakeEngine struct {
	touched atomic.Int64
	mu      sync.Mutex
	ended   []string
}

func (e *fakeEngine) Touch(string) { e.touched.Add(1) }
func (e *fakeEngine) EndCall(serial string, _ trunking.EndReason) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.ended = append(e.ended, serial)
	return true
}

func mkComposer(t *testing.T, src *fakeSource) (*Composer, *events.Bus, *recordingSink, *fakeEngine, func()) {
	t.Helper()
	bus := events.NewBus(8)
	sink := &recordingSink{}
	eng := &fakeEngine{}
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{"VOICE-1": src}},
		Sink:          sink,
		Engine:        eng,
		IQSampleRate:  2_400_000,
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go c.Run(ctx)
	teardown := func() {
		cancel()
		c.Close()
		bus.Close()
	}
	return c, bus, sink, eng, teardown
}

// publishStartFM publishes a CallStart for a Voice call on the
// supplied serial. The protocol is tagged "fm" so the composer runs
// the analog passthrough chain.
func publishStartFM(bus *events.Bus, serial string) {
	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			Grant: trunking.Grant{
				System: "Alpha", Protocol: "fm",
				GroupID: 100, FrequencyHz: 851_000_000,
			},
			DeviceSerial: serial,
			StartedAt:    time.Now().UTC(),
		},
	})
}

// publishEnd publishes a CallEnd for the supplied serial.
func publishEnd(bus *events.Bus, serial string) {
	bus.Publish(events.Event{
		Kind: events.KindCallEnd,
		Payload: trunking.CallEnd{
			Grant:        trunking.Grant{Protocol: "fm"},
			DeviceSerial: serial,
			Reason:       trunking.EndReasonNormal,
		},
	})
}

func waitFor(t *testing.T, deadline time.Duration, fn func() bool) {
	t.Helper()
	end := time.Now().Add(deadline)
	for time.Now().Before(end) {
		if fn() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("timed out waiting for condition")
}

func TestComposerStartsChainOnCallStart(t *testing.T) {
	src := newFakeSource()
	c, bus, _, _, teardown := mkComposer(t, src)
	defer teardown()

	publishStartFM(bus, "VOICE-1")

	waitFor(t, time.Second, func() bool {
		for _, s := range c.ActiveChains() {
			if s == "VOICE-1" {
				return true
			}
		}
		return false
	})
}

func TestComposerWritesPCMFromIQ(t *testing.T) {
	src := newFakeSource()
	_, bus, sink, _, teardown := mkComposer(t, src)
	defer teardown()

	publishStartFM(bus, "VOICE-1")
	// Wait until StreamIQ has been opened.
	waitFor(t, time.Second, func() bool {
		src.mu.Lock()
		defer src.mu.Unlock()
		return len(src.chs) > 0
	})

	// Push a chunk of IQ samples; magnitude doesn't matter because we
	// only verify PCM bytes flow.
	chunk := make([]complex64, 4096)
	for i := range chunk {
		chunk[i] = complex(0.5, 0.5)
	}
	src.SendIQ(chunk)

	waitFor(t, time.Second, func() bool { return sink.total("VOICE-1") > 0 })
}

func TestComposerTouchesEngineWhileChainRuns(t *testing.T) {
	src := newFakeSource()
	_, bus, _, eng, teardown := mkComposer(t, src)
	defer teardown()

	publishStartFM(bus, "VOICE-1")
	waitFor(t, time.Second, func() bool { return eng.touched.Load() >= 2 })
}

func TestComposerStopsChainOnCallEnd(t *testing.T) {
	src := newFakeSource()
	c, bus, _, _, teardown := mkComposer(t, src)
	defer teardown()

	publishStartFM(bus, "VOICE-1")
	waitFor(t, time.Second, func() bool { return len(c.ActiveChains()) == 1 })

	publishEnd(bus, "VOICE-1")
	waitFor(t, time.Second, func() bool { return len(c.ActiveChains()) == 0 })
}

func TestComposerSkipsDigitalProtocol(t *testing.T) {
	src := newFakeSource()
	c, bus, sink, _, teardown := mkComposer(t, src)
	defer teardown()

	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			Grant:        trunking.Grant{Protocol: "p25", GroupID: 1, FrequencyHz: 851_000_000},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})

	// Give the loop a few ticks; nothing should happen.
	time.Sleep(80 * time.Millisecond)
	if got := len(c.ActiveChains()); got != 0 {
		t.Errorf("active chains for digital grant = %d, want 0", got)
	}
	if sink.total("VOICE-1") != 0 {
		t.Errorf("digital grant produced PCM samples")
	}
}

func TestComposerNoDeviceForSerialIsBenign(t *testing.T) {
	src := newFakeSource()
	c, bus, _, _, teardown := mkComposer(t, src)
	defer teardown()

	publishStartFM(bus, "VOICE-DOES-NOT-EXIST")
	time.Sleep(80 * time.Millisecond)
	if got := len(c.ActiveChains()); got != 0 {
		t.Errorf("unknown serial spawned a chain: %v", c.ActiveChains())
	}
}

func TestNewValidates(t *testing.T) {
	if _, err := New(Options{}); err == nil {
		t.Error("expected error for empty options")
	}
	if _, err := New(Options{Bus: events.NewBus(1)}); err == nil {
		t.Error("expected error for missing Devices")
	}
	if _, err := New(Options{Bus: events.NewBus(1), Devices: &fakeDevices{}}); err == nil {
		t.Error("expected error for missing IQSampleRate")
	}
}

func TestComposerEqualizerEnabledStillProducesPCM(t *testing.T) {
	src := newFakeSource()
	bus := events.NewBus(8)
	sink := &recordingSink{}
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{"VOICE-1": src}},
		Sink:          sink,
		Engine:        &fakeEngine{},
		IQSampleRate:  2_400_000,
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
		Equalizer: EqualizerConfig{
			Enabled:  true,
			Taps:     8,
			StepSize: 1e-4,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		c.Close()
		bus.Close()
	}()
	go c.Run(ctx)

	publishStartFM(bus, "VOICE-1")
	waitFor(t, time.Second, func() bool {
		src.mu.Lock()
		defer src.mu.Unlock()
		return len(src.chs) > 0
	})

	chunk := make([]complex64, 4096)
	for i := range chunk {
		chunk[i] = complex(0.5, 0.5)
	}
	src.SendIQ(chunk)

	waitFor(t, time.Second, func() bool { return sink.total("VOICE-1") > 0 })
}

func TestComposerEqualizerDefaultsApplied(t *testing.T) {
	bus := events.NewBus(2)
	defer bus.Close()
	c, err := New(Options{
		Bus:          bus,
		Devices:      &fakeDevices{},
		IQSampleRate: 2_400_000,
		Equalizer:    EqualizerConfig{Enabled: true}, // taps/step zero
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if c.eqCfg.Taps != 8 {
		t.Errorf("default Taps = %d, want 8", c.eqCfg.Taps)
	}
	if c.eqCfg.StepSize != 1e-4 {
		t.Errorf("default StepSize = %g, want 1e-4", c.eqCfg.StepSize)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	bus := events.NewBus(2)
	defer bus.Close()
	c, err := New(Options{
		Bus: bus, Devices: &fakeDevices{}, IQSampleRate: 2_400_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}
