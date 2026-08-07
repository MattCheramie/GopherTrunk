package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// fakeDDCTap is a stub ddcVoiceTap: SubscribeVoiceIQ pre-loads a buffered channel
// with the configured chunks (then blocks, so the capture ends on its timer),
// letting the DDC-tap capture path be exercised without a live decoder.
type fakeDDCTap struct {
	rate   float64
	center uint32
	chunks [][]complex64
	drops  uint64 // reported by the unsubscribe func, mirroring the fan-out
}

func (f *fakeDDCTap) SubscribeVoiceIQ() (<-chan []complex64, func() uint64) {
	ch := make(chan []complex64, len(f.chunks)+1)
	for _, c := range f.chunks {
		ch <- c
	}
	return ch, func() uint64 { return f.drops }
}

func (f *fakeDDCTap) PipelineRateHz() float64 { return f.rate }
func (f *fakeDDCTap) CenterFreqHz() uint32    { return f.center }

// TestCaptureDDCToFile checks the narrowband DDC capture writes every delivered
// sample in the chosen format (cs16 = 4 bytes/sample) and ends cleanly on the
// duration timer.
func TestCaptureDDCToFile(t *testing.T) {
	tap := &fakeDDCTap{rate: 144000, center: 467913000, chunks: [][]complex64{
		make([]complex64, 128), make([]complex64, 128), make([]complex64, 128),
	}}
	path := filepath.Join(t.TempDir(), "ddc.cs16")
	samples, _, err := captureDDCToFile(context.Background(), tap, path, siglab.FormatS16, 1)
	if err != nil {
		t.Fatalf("captureDDCToFile: %v", err)
	}
	if samples != 384 {
		t.Errorf("samples = %d, want 384", samples)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size() != 384*4 {
		t.Errorf("file size = %d, want %d (384 samples × 4 bytes cs16)", fi.Size(), 384*4)
	}
}

// TestCaptureDDCToFileReportsDrops confirms the capture surfaces the fan-out's
// subscriber drop count (a gappy grab) instead of always reporting 0.
func TestCaptureDDCToFileReportsDrops(t *testing.T) {
	tap := &fakeDDCTap{rate: 144000, center: 467913000, drops: 7, chunks: [][]complex64{
		make([]complex64, 64),
	}}
	path := filepath.Join(t.TempDir(), "ddc.cs16")
	_, drops, err := captureDDCToFile(context.Background(), tap, path, siglab.FormatS16, 1)
	if err != nil {
		t.Fatalf("captureDDCToFile: %v", err)
	}
	if drops != 7 {
		t.Errorf("drops = %d, want 7 (the fan-out drop count must be surfaced, not hard-coded 0)", drops)
	}
}

// TestCaptureDDCToFileNilTap errors clearly when the control decoder is absent.
func TestCaptureDDCToFileNilTap(t *testing.T) {
	if _, _, err := captureDDCToFile(context.Background(), nil, filepath.Join(t.TempDir(), "x.cs16"), siglab.FormatS16, 1); err == nil {
		t.Error("captureDDCToFile(nil tap) = nil error, want an error")
	}
}

// TestAutoRecordDDCTapMetadata drives a full triggered capture with tap: ddc and
// confirms the file + metadata carry the DDC pipeline rate (144 kHz) and control
// centre, not the wideband SDR rate — and that a missing dir is created.
func TestAutoRecordDDCTapMetadata(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "made", "on", "demand") // exercises MkdirAll
	cfg := config.BasebandAutoRecordConfig{
		Enabled: true, Dir: dir, Seconds: 1, Format: "cs16", Tap: "ddc",
	}
	tap := &fakeDDCTap{rate: 144000, center: 467913000, chunks: [][]complex64{make([]complex64, 144)}}
	a := newIQAutoRecorder(cfg, "TETRA_Site_1", "tetra", "cc:same-carrier", nil, func() ddcVoiceTap { return tap }, nil)
	if a == nil {
		t.Fatal("newIQAutoRecorder returned nil")
	}
	now := time.Date(2026, 7, 24, 8, 0, 0, 0, time.UTC)
	a.now = func() time.Time { return now }
	a.runCapture(context.Background(), "concurrent", "TETRA_Site_1", "tetra", now)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	var cs16, meta string
	for _, e := range entries {
		switch {
		case strings.HasSuffix(e.Name(), ".cs16"):
			cs16 = e.Name()
		case strings.HasSuffix(e.Name(), ".metadata.json"):
			meta = filepath.Join(dir, e.Name())
		}
	}
	if cs16 == "" {
		t.Fatalf("no .cs16 capture written; dir has %v", entries)
	}
	if !strings.Contains(cs16, "144000hz") {
		t.Errorf("filename %q does not carry the DDC rate 144000hz", cs16)
	}
	if !strings.Contains(cs16, "467913000") {
		t.Errorf("filename %q does not carry the control centre 467913000", cs16)
	}
	b, err := os.ReadFile(meta)
	if err != nil {
		t.Fatalf("read metadata: %v", err)
	}
	var m struct {
		SampleRateHz float64 `json:"sample_rate_hz"`
		CenterFreqHz uint32  `json:"center_freq_hz"`
		Format       string  `json:"format"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("parse metadata: %v", err)
	}
	if m.SampleRateHz != 144000 {
		t.Errorf("metadata sample_rate_hz = %v, want 144000 (DDC pipeline rate)", m.SampleRateHz)
	}
	if m.CenterFreqHz != 467913000 {
		t.Errorf("metadata center_freq_hz = %d, want 467913000", m.CenterFreqHz)
	}
	if m.Format != "cs16" {
		t.Errorf("metadata format = %q, want cs16", m.Format)
	}
}

// TestAutoRecordSameSecondCapturesDoNotOverwrite pins that two captures triggered
// within the same wall-clock second get distinct files. autoRecordFilename stamps
// the name to 1-second resolution, and autoRecordMaxInFlight allows concurrent
// captures by design, so before the per-capture path reservation two same-second
// triggers built the identical <system>_<ts>_<reason>_<freq>_<rate>hz.cs16 name
// and the second os.Create truncated/raced the first — the operator's observed
// "two captures, same path" overwrite.
func TestAutoRecordSameSecondCapturesDoNotOverwrite(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "iq", "auto")
	cfg := config.BasebandAutoRecordConfig{
		Enabled: true, Dir: dir, Seconds: 1, Format: "cs16", Tap: "ddc",
	}
	a := newIQAutoRecorder(cfg, "250_013", "tetra", "AIRSPY", nil, nil, nil)
	if a == nil {
		t.Fatal("newIQAutoRecorder returned nil")
	}

	// Stub the actuation: record each path handed to a capture and write a couple
	// of bytes there, so a collision manifests as one file instead of two.
	var mu sync.Mutex
	var paths []string
	a.capture = func(_ context.Context, path string, _ siglab.SampleFormat, _ int) (int64, uint64, error) {
		mu.Lock()
		paths = append(paths, path)
		mu.Unlock()
		if err := os.WriteFile(path, []byte("iq"), 0o644); err != nil {
			return 0, 0, err
		}
		return 1, 0, nil
	}

	// Both triggers land in the same UTC second (the reported collision second).
	at := time.Date(2026, 8, 7, 10, 9, 23, 0, time.UTC)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.runCapture(context.Background(), "concurrent", "250_013", "tetra", at)
		}()
	}
	wg.Wait()

	mu.Lock()
	got := append([]string(nil), paths...)
	mu.Unlock()
	if len(got) != 2 {
		t.Fatalf("expected 2 capture invocations, got %d", len(got))
	}
	if got[0] == got[1] {
		t.Fatalf("both captures wrote the same path %q — they overwrite each other", got[0])
	}

	var cs16 int
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".cs16") {
			cs16++
		}
	}
	if cs16 != 2 {
		t.Fatalf("found %d .cs16 files, want 2 — same-second captures overwrote each other", cs16)
	}
}

// fakeClock is a manually-advanced clock for deterministic cooldown tests.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// captureRecorder records every capture the auto-recorder actuates and blocks
// until released, so in-flight bookkeeping can be tested.
type captureRecorder struct {
	mu    sync.Mutex
	paths []string
	block chan struct{} // when non-nil, capture blocks on it
}

func (r *captureRecorder) fn(_ context.Context, path string, _ siglab.SampleFormat, _ int) (int64, uint64, error) {
	r.mu.Lock()
	r.paths = append(r.paths, path)
	block := r.block
	r.mu.Unlock()
	if block != nil {
		<-block
	}
	return 100, 0, nil
}

func (r *captureRecorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.paths)
}

// newTestAutoRecorder wires an auto-recorder with a fake clock + capture stub.
func newTestAutoRecorder(t *testing.T, cfg config.BasebandAutoRecordConfig) (*iqAutoRecorder, *captureRecorder, *fakeClock) {
	t.Helper()
	cfg.Enabled = true
	if cfg.Dir == "" {
		cfg.Dir = t.TempDir()
	}
	if cfg.Seconds == 0 {
		cfg.Seconds = 4
	}
	a := newIQAutoRecorder(cfg, "TETRA_Site_1", "tetra", "cc:same-carrier", nil, nil, nil)
	if a == nil {
		t.Fatal("newIQAutoRecorder returned nil for enabled config")
	}
	clock := &fakeClock{t: time.Date(2026, 7, 23, 23, 0, 0, 0, time.UTC)}
	a.now = clock.now
	rec := &captureRecorder{}
	a.capture = rec.fn
	// A capture goroutine writes its metadata sidecar into cfg.Dir AFTER the
	// capture func returns, so wait for the recorder to go idle before t.TempDir's
	// RemoveAll runs (registered earlier ⇒ this LIFO cleanup runs first). Release
	// any capture still blocked on rec.block so the drain can complete.
	t.Cleanup(func() {
		rec.mu.Lock()
		if rec.block != nil {
			select {
			case <-rec.block: // already closed
			default:
				close(rec.block)
			}
			rec.block = nil
		}
		rec.mu.Unlock()
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) && a.inFlightCount() > 0 {
			time.Sleep(2 * time.Millisecond)
		}
	})
	return a, rec, clock
}

// waitFor polls until pred is true or the deadline elapses (captures run in
// their own goroutine).
func waitFor(t *testing.T, pred func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if pred() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("condition not met before deadline")
}

func grant(system string, tg uint32, ts uint8) trunking.Grant {
	return trunking.Grant{System: system, Protocol: "tetra", GroupID: tg, Timeslot: ts, FrequencyHz: 467913000}
}

func TestAutoRecordDisabledReturnsNil(t *testing.T) {
	if a := newIQAutoRecorder(config.BasebandAutoRecordConfig{Enabled: false}, "s", "tetra", "x", nil, nil, nil); a != nil {
		t.Fatal("expected nil for disabled config")
	}
}

func TestAutoRecordConcurrentTrigger(t *testing.T) {
	a, rec, _ := newTestAutoRecorder(t, config.BasebandAutoRecordConfig{OnConcurrentCalls: 2})

	// One active call: below threshold, no capture.
	a.onGrant(grant("TETRA_Site_1", 1001, 1))
	if rec.count() != 0 {
		t.Fatalf("single call should not trigger; got %d captures", rec.count())
	}
	// A distinct second call within the window: at threshold, capture fires.
	a.onGrant(grant("TETRA_Site_1", 1002, 2))
	waitFor(t, func() bool { return rec.count() == 1 })
}

func TestAutoRecordConcurrencyWindowExpiry(t *testing.T) {
	a, rec, clock := newTestAutoRecorder(t, config.BasebandAutoRecordConfig{OnConcurrentCalls: 2})
	a.onGrant(grant("TETRA_Site_1", 1001, 1))
	// Advance past the window so the first call ages out before the second.
	clock.advance(autoRecordConcurrencyWindow + time.Second)
	a.onGrant(grant("TETRA_Site_1", 1002, 2))
	// Only one call is live in the window -> below threshold -> no capture.
	time.Sleep(20 * time.Millisecond)
	if rec.count() != 0 {
		t.Fatalf("aged-out call should not count toward concurrency; got %d captures", rec.count())
	}
}

func TestAutoRecordEncryptedAndEmergency(t *testing.T) {
	a, rec, clock := newTestAutoRecorder(t, config.BasebandAutoRecordConfig{OnEncrypted: true, OnEmergency: true, Cooldown: "0s"})
	g := grant("TETRA_Site_1", 1001, 1)
	g.Encrypted = true
	a.onGrant(g)
	waitFor(t, func() bool { return rec.count() == 1 })

	clock.advance(time.Second)
	g2 := grant("TETRA_Site_1", 1002, 1)
	g2.Emergency = true
	a.onGrant(g2)
	waitFor(t, func() bool { return rec.count() == 2 })
}

func TestAutoRecordNoTriggerWhenClear(t *testing.T) {
	a, rec, _ := newTestAutoRecorder(t, config.BasebandAutoRecordConfig{OnEncrypted: true, OnEmergency: true, OnConcurrentCalls: 5})
	a.onGrant(grant("TETRA_Site_1", 1001, 1)) // clear, single call
	time.Sleep(20 * time.Millisecond)
	if rec.count() != 0 {
		t.Fatalf("clear single call should not trigger; got %d captures", rec.count())
	}
}

func TestAutoRecordNoVoiceDeviceTrigger(t *testing.T) {
	a, rec, _ := newTestAutoRecorder(t, config.BasebandAutoRecordConfig{OnNoVoiceDevice: true})
	a.onGrantUnserved(grant("TETRA_Site_1", 1001, 1))
	waitFor(t, func() bool { return rec.count() == 1 })

	// With the trigger off, an unserved grant is ignored.
	b, rec2, _ := newTestAutoRecorder(t, config.BasebandAutoRecordConfig{OnConcurrentCalls: 2})
	b.onGrantUnserved(grant("TETRA_Site_1", 1001, 1))
	time.Sleep(20 * time.Millisecond)
	if rec2.count() != 0 {
		t.Fatalf("on_no_voice_device off should ignore unserved grant; got %d captures", rec2.count())
	}
}

// fakeLostPayload is a trunking.LockedPayload for the cc.lost event.
type fakeLostPayload struct{ freq uint32 }

func (f fakeLostPayload) LockedFrequencyHz() uint32 { return f.freq }
func (f fakeLostPayload) LockedNAC() uint16         { return 0 }

// TestAutoRecordCCSyncLossTrigger pins the new on_cc_sync_loss trigger: a cc.lost
// event (a locked control channel that suddenly lost sync) fires a capture
// labelled "cc_sync_loss" when enabled, and is ignored when off.
func TestAutoRecordCCSyncLossTrigger(t *testing.T) {
	a, rec, _ := newTestAutoRecorder(t, config.BasebandAutoRecordConfig{OnCCSyncLoss: true})
	a.onCCLost(fakeLostPayload{freq: 467_912_500})
	waitFor(t, func() bool { return rec.count() == 1 })
	rec.mu.Lock()
	path := rec.paths[0]
	rec.mu.Unlock()
	if !strings.Contains(path, "cc_sync_loss") {
		t.Errorf("capture path %q missing cc_sync_loss reason", path)
	}

	// With the trigger off, a sync-loss event is ignored.
	b, rec2, _ := newTestAutoRecorder(t, config.BasebandAutoRecordConfig{OnConcurrentCalls: 2})
	b.onCCLost(fakeLostPayload{freq: 467_912_500})
	time.Sleep(20 * time.Millisecond)
	if rec2.count() != 0 {
		t.Fatalf("on_cc_sync_loss off should ignore cc.lost; got %d captures", rec2.count())
	}
}

func TestAutoRecordCooldownThrottlesAuto(t *testing.T) {
	a, rec, clock := newTestAutoRecorder(t, config.BasebandAutoRecordConfig{OnEncrypted: true, Cooldown: "10s"})
	enc := func(tg uint32) trunking.Grant { g := grant("TETRA_Site_1", tg, 1); g.Encrypted = true; return g }

	a.onGrant(enc(1))
	waitFor(t, func() bool { return rec.count() == 1 })

	// Within cooldown: suppressed.
	clock.advance(3 * time.Second)
	a.onGrant(enc(2))
	time.Sleep(20 * time.Millisecond)
	if rec.count() != 1 {
		t.Fatalf("trigger within cooldown should be suppressed; got %d captures", rec.count())
	}
	// Past cooldown: fires again.
	clock.advance(8 * time.Second)
	a.onGrant(enc(3))
	waitFor(t, func() bool { return rec.count() == 2 })
}

func TestAutoRecordManualBypassesCooldown(t *testing.T) {
	a, rec, _ := newTestAutoRecorder(t, config.BasebandAutoRecordConfig{OnEncrypted: true, Cooldown: "1h"})
	a.TriggerManual()
	waitFor(t, func() bool { return rec.count() == 1 })
	// A second manual trigger fires despite the long cooldown.
	a.TriggerManual()
	waitFor(t, func() bool { return rec.count() == 2 })
}

func TestAutoRecordInFlightCap(t *testing.T) {
	a, rec, _ := newTestAutoRecorder(t, config.BasebandAutoRecordConfig{OnEncrypted: true, Cooldown: "0s"})
	rec.mu.Lock()
	rec.block = make(chan struct{})
	rec.mu.Unlock()

	// Fire more than the cap while captures are blocked.
	for i := 0; i < autoRecordMaxInFlight+3; i++ {
		a.TriggerManual()
	}
	waitFor(t, func() bool { return rec.count() == autoRecordMaxInFlight })
	// Give any surplus goroutines a chance; the cap must hold.
	time.Sleep(20 * time.Millisecond)
	if rec.count() != autoRecordMaxInFlight {
		t.Fatalf("in-flight cap breached: got %d, want %d", rec.count(), autoRecordMaxInFlight)
	}
	// The blocked captures are released and drained by the t.Cleanup registered
	// in newTestAutoRecorder before the temp dir is removed.
}

func TestAutoRecordFilename(t *testing.T) {
	at := time.Date(2026, 7, 23, 20, 16, 8, 0, time.UTC)
	name := autoRecordFilename("TETRA_Site_1", "concurrent", at, 467913000, 50000, siglab.FormatS16)
	want := "TETRA_Site_1_20260723T201608Z_concurrent_467913000_50000hz.cs16"
	if name != want {
		t.Fatalf("filename = %q, want %q", name, want)
	}
}

// TestAutoRecordRunDispatch drives Run with a real event bus and checks the
// event switch routes KindGrant / KindGrantUnserved to the right handlers.
func TestAutoRecordRunDispatch(t *testing.T) {
	a, rec, _ := newTestAutoRecorder(t, config.BasebandAutoRecordConfig{OnNoVoiceDevice: true, OnEncrypted: true, Cooldown: "0s"})
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = a.Run(ctx, sub) }()

	enc := grant("TETRA_Site_1", 1, 1)
	enc.Encrypted = true
	bus.Publish(events.Event{Kind: events.KindGrant, Payload: enc})
	bus.Publish(events.Event{Kind: events.KindGrantUnserved, Payload: grant("TETRA_Site_1", 2, 1)})
	waitFor(t, func() bool { return rec.count() == 2 })
}
