package composer

import (
	"context"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// fakeSource implements IQSource. Each call to StreamIQ returns a
// fresh channel; the test pushes IQ chunks through SendIQ. Returns
// 0 from SampleRateHz so the composer falls back to its
// daemon-wide IQSampleRate — the default for existing tests.
type fakeSource struct {
	mu  sync.Mutex
	chs []chan []complex64
	// baseHz / exactHz let a test drive a per-source rate. Both default to
	// 0 so existing tests keep the daemon-wide fallback; exactHz>0 exercises
	// the fractional-rate path the digital voice chains clock from.
	baseHz  uint32
	exactHz float64
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

func (f *fakeSource) SampleRateHz() uint32 { return f.baseHz }

// SampleRateExactHz implements the optional exactRateSource capability so a
// test can drive a fractional delivered rate into the voice chains. Returns 0
// (no override) unless exactHz was set, preserving the daemon-wide fallback
// for the existing tests.
func (f *fakeSource) SampleRateExactHz() float64 { return f.exactHz }

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
	mu  sync.Mutex
	pcm map[string][]int16
	raw map[string][][]byte
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

func (r *recordingSink) WriteRawFrame(serial string, frame []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.raw == nil {
		r.raw = make(map[string][][]byte)
	}
	r.raw[serial] = append(r.raw[serial], append([]byte(nil), frame...))
	return nil
}

func (r *recordingSink) total(serial string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.pcm[serial])
}

func (r *recordingSink) rawFrames(serial string) [][]byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([][]byte, len(r.raw[serial]))
	copy(out, r.raw[serial])
	return out
}

type fakeEngine struct {
	touched atomic.Int64
	mu      sync.Mutex
	ended   []string
	reasons map[string]trunking.EndReason
	signals map[string]float64
}

func (e *fakeEngine) Touch(string) { e.touched.Add(1) }
func (e *fakeEngine) EndCall(serial string, reason trunking.EndReason) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.ended = append(e.ended, serial)
	if e.reasons == nil {
		e.reasons = make(map[string]trunking.EndReason)
	}
	e.reasons[serial] = reason
	return true
}

func (e *fakeEngine) UpdateSignal(serial string, dbfs float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.signals == nil {
		e.signals = make(map[string]float64)
	}
	e.signals[serial] = dbfs
}

// signal returns the last dBFS UpdateSignal(serial) carried and whether
// it was ever called for serial.
func (e *fakeEngine) signal(serial string) (float64, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	v, ok := e.signals[serial]
	return v, ok
}

// endReason returns the reason the most recent EndCall(serial) carried.
func (e *fakeEngine) endReason(serial string) trunking.EndReason {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.reasons[serial]
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

// TestComposerMeasuresSignalDbFSFromIQ drives constant-magnitude IQ through
// the real FM voice chain and asserts the chain's bt.observe wiring measures
// the received channel power and stamps it onto the engine at end-of-call.
// |0.1|² = 0.01 mean power ⇒ ~-20 dBFS.
func TestComposerMeasuresSignalDbFSFromIQ(t *testing.T) {
	src := newFakeSource()
	bus := events.NewBus(8)
	eng := &fakeEngine{}
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{"VOICE-1": src}},
		Sink:          &recordingSink{},
		Engine:        eng,
		IQSampleRate:  2_400_000,
		PCMSampleRate: 8000,
		TouchInterval: 20 * time.Millisecond,
		VoiceHangtime: 120 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)
	defer c.Close()
	defer bus.Close()

	publishStartFM(bus, "VOICE-1")
	waitFor(t, time.Second, func() bool {
		src.mu.Lock()
		defer src.mu.Unlock()
		return len(src.chs) > 0
	})

	chunk := make([]complex64, 4096)
	for i := range chunk {
		chunk[i] = complex(0.1, 0)
	}
	for range 3 {
		src.SendIQ(chunk)
		time.Sleep(10 * time.Millisecond)
	}

	// The chain ends via the hangtime / no-voice window; UpdateSignal is
	// stamped just before EndCall.
	waitFor(t, 2*time.Second, func() bool {
		_, ok := eng.signal("VOICE-1")
		return ok
	})
	got, ok := eng.signal("VOICE-1")
	if !ok {
		t.Fatal("UpdateSignal never called for the FM chain")
	}
	if got >= 0 {
		t.Errorf("SignalDbFS = %v, want negative (below full scale)", got)
	}
	if math.Abs(got+20) > 3 {
		t.Errorf("SignalDbFS = %v dBFS, want ~-20 (mean|iq|² = 0.01)", got)
	}
}

func TestComposerTouchesEngineWhileChainRuns(t *testing.T) {
	src := newFakeSource()
	_, bus, _, eng, teardown := mkComposer(t, src)
	defer teardown()

	publishStartFM(bus, "VOICE-1")
	// Wait until StreamIQ has been opened so SendIQ doesn't race.
	waitFor(t, time.Second, func() bool {
		src.mu.Lock()
		defer src.mu.Unlock()
		return len(src.chs) > 0
	})

	// Touch is gated on actual PCM emission (issue #356) — keep feeding
	// IQ so the chain produces PCM batches across multiple touch ticks.
	chunk := make([]complex64, 4096)
	for i := range chunk {
		chunk[i] = complex(0.5, 0.5)
	}
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
			}
			src.SendIQ(chunk)
			time.Sleep(10 * time.Millisecond)
		}
	}()

	waitFor(t, 2*time.Second, func() bool { return eng.touched.Load() >= 2 })
}

// TestComposerStopsTouchingEngineWhenIQStops asserts the activity gate
// introduced for issue #356: once PCM emission stops (e.g. simulcast
// garbage, vocoder hang) the chain MUST stop touching the engine so
// the trunking watchdog can fire and release the voice SDR. Before
// the fix the touch ticker ran unconditionally and the call lived
// forever.
func TestComposerStopsTouchingEngineWhenIQStops(t *testing.T) {
	src := newFakeSource()
	_, bus, _, eng, teardown := mkComposer(t, src)
	defer teardown()

	publishStartFM(bus, "VOICE-1")
	waitFor(t, time.Second, func() bool {
		src.mu.Lock()
		defer src.mu.Unlock()
		return len(src.chs) > 0
	})

	// Push enough IQ to establish baseline Touch activity.
	chunk := make([]complex64, 4096)
	for i := range chunk {
		chunk[i] = complex(0.5, 0.5)
	}
	src.SendIQ(chunk)
	waitFor(t, time.Second, func() bool { return eng.touched.Load() >= 1 })

	// Stop feeding IQ; sample the touch count after several touch
	// intervals (TouchInterval is 30 ms above; 5× ≈ 150 ms is plenty).
	baseline := eng.touched.Load()
	time.Sleep(200 * time.Millisecond)
	if got := eng.touched.Load(); got != baseline {
		t.Fatalf("expected no further touches after IQ stopped; baseline=%d got=%d", baseline, got)
	}
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

// TestComposerTailFadeOnCallEnd verifies the 10ms linear fade-out
// that smooths the squelch-close click: the recorder's last few
// samples should ramp from the previous non-zero value down to (or
// near) zero rather than cutting abruptly.
func TestComposerTailFadeOnCallEnd(t *testing.T) {
	src := newFakeSource()
	_, bus, sink, _, teardown := mkComposer(t, src)
	defer teardown()

	publishStartFM(bus, "VOICE-1")
	waitFor(t, time.Second, func() bool {
		src.mu.Lock()
		defer src.mu.Unlock()
		return len(src.chs) > 0
	})

	// Push enough IQ to produce PCM, then end the call.
	chunk := make([]complex64, 4096)
	for i := range chunk {
		chunk[i] = complex(0.5, 0.5)
	}
	src.SendIQ(chunk)
	waitFor(t, time.Second, func() bool { return sink.total("VOICE-1") > 0 })

	publishEnd(bus, "VOICE-1")
	// Give the chain time to emit the fade-out tail before we read.
	waitFor(t, time.Second, func() bool {
		// 10 ms at 8 kHz = 80 samples appended after the last
		// IQ-driven sample.
		return sink.total("VOICE-1") > 0
	})
	time.Sleep(50 * time.Millisecond)

	sink.mu.Lock()
	pcm := sink.pcm["VOICE-1"]
	sink.mu.Unlock()
	if len(pcm) < 80 {
		t.Fatalf("not enough PCM to inspect tail: %d", len(pcm))
	}
	// Final sample must be 0 (linear ramp ends at zero).
	if pcm[len(pcm)-1] != 0 {
		t.Errorf("tail final sample = %d, want 0", pcm[len(pcm)-1])
	}
}

func TestComposerSkipsDigitalProtocol(t *testing.T) {
	src := newFakeSource()
	c, bus, sink, _, teardown := mkComposer(t, src)
	defer teardown()

	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			// NXDN has no composer voice chain yet — it must be bypassed.
			Grant:        trunking.Grant{Protocol: "nxdn", GroupID: 1, FrequencyHz: 851_000_000},
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

func TestComposerRunsFMChainForAnalogTrunking(t *testing.T) {
	for _, proto := range []string{"motorola", "edacs", "ltr", "mpt1327"} {
		t.Run(proto, func(t *testing.T) {
			src := newFakeSource()
			c, bus, _, _, teardown := mkComposer(t, src)
			defer teardown()
			bus.Publish(events.Event{
				Kind: events.KindCallStart,
				Payload: trunking.CallStart{
					Grant:        trunking.Grant{Protocol: proto, GroupID: 1, FrequencyHz: 851_000_000},
					DeviceSerial: "VOICE-1",
					StartedAt:    time.Now().UTC(),
				},
			})
			waitFor(t, time.Second, func() bool {
				for _, s := range c.ActiveChains() {
					if s == "VOICE-1" {
						return true
					}
				}
				return false
			})
		})
	}
}

func TestComposerBypassesEDACSProVoice(t *testing.T) {
	src := newFakeSource()
	c, bus, _, _, teardown := mkComposer(t, src)
	defer teardown()
	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			// EDACS ProVoice is digital — it must not be FM-demodulated.
			Grant:        trunking.Grant{Protocol: "edacs", ProVoice: true, GroupID: 1, FrequencyHz: 851_000_000},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})
	time.Sleep(80 * time.Millisecond)
	if got := len(c.ActiveChains()); got != 0 {
		t.Errorf("EDACS ProVoice grant spawned %d chains, want 0", got)
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

// attrCaptureHandler is a minimal slog.Handler that records the float64 value
// of a named attribute the first time a record with a given message is logged.
// Used to observe the rate the voice chains clock their receiver at, which is
// otherwise internal to the chain goroutine.
type attrCaptureHandler struct {
	mu      sync.Mutex
	wantMsg string
	wantKey string
	got     float64
	seen    bool
}

func (h *attrCaptureHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *attrCaptureHandler) WithAttrs([]slog.Attr) slog.Handler       { return h }
func (h *attrCaptureHandler) WithGroup(string) slog.Handler            { return h }
func (h *attrCaptureHandler) Handle(_ context.Context, r slog.Record) error {
	if r.Message != h.wantMsg {
		return nil
	}
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == h.wantKey {
			h.mu.Lock()
			h.got = a.Value.Float64()
			h.seen = true
			h.mu.Unlock()
			return false
		}
		return true
	})
	return nil
}
func (h *attrCaptureHandler) value() (float64, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.got, h.seen
}

// narrowbandHzForTest mirrors the wbvoice virtual-tuner nominal rate so the
// test's source advertises a realistic integer base rate alongside its exact
// fractional rate.
const narrowbandHzForTest = 48_000

// TestP25P1ChainClocksAtExactSourceRate verifies the P25 Phase 1 voice chain
// clocks its receiver from the source's exact fractional rate, not the rounded
// integer one — the parity-with-control-channel fix (issue #550) that keeps a
// symbol clock from drifting and slipping (audible spikes/glitches). The chain
// logs the rate it builds the receiver at as "symbol_rate_hz"; we assert it
// equals the fractional rate the source advertised.
func TestP25P1ChainClocksAtExactSourceRate(t *testing.T) {
	const fractionalHz = 47983.5 // not an integer, and < 96 kHz so decim clamps to 1
	capture := &attrCaptureHandler{
		wantMsg: "composer: p25p1 voice chain started",
		wantKey: "symbol_rate_hz",
	}
	src := newFakeSource()
	src.baseHz = narrowbandHzForTest
	src.exactHz = fractionalHz

	bus := events.NewBus(8)
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{"VOICE-1": src}},
		Sink:          &recordingSink{},
		Engine:        &fakeEngine{},
		IQSampleRate:  2_400_000,
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
		Log:           slog.New(capture),
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go c.Run(ctx)
	defer func() { cancel(); c.Close(); bus.Close() }()

	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			Grant: trunking.Grant{
				System: "Alpha", Protocol: "p25",
				GroupID: 100, FrequencyHz: 851_000_000,
			},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})

	waitFor(t, time.Second, func() bool { _, ok := capture.value(); return ok })
	if got, _ := capture.value(); got != fractionalHz {
		t.Errorf("receiver clocked at %v Hz, want the exact source rate %v Hz", got, fractionalHz)
	}
}
