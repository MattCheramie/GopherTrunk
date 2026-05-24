package main

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// flakyIQSource is an IQSource that closes its channel n times
// (simulating n consecutive USB reaper deaths) and then returns a
// healthy never-closing channel — models the daemon's
// reopen-and-restart loop hitting a transient device fault.
type flakyIQSource struct {
	deaths   int32
	maxDeath int32
}

func (f *flakyIQSource) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	ch := make(chan []complex64)
	go func() {
		defer close(ch)
		select {
		case ch <- make([]complex64, 4):
		case <-ctx.Done():
			return
		}
		if atomic.AddInt32(&f.deaths, 1) <= f.maxDeath {
			// Simulate reaper death: close the channel mid-stream
			// without waiting on ctx.
			return
		}
		// Healthy stream — block on ctx.
		<-ctx.Done()
	}()
	return ch, nil
}

// TestRunCCDecoderWithRetry_RecoversAfterTransientDeath pins the
// issue-#345 restart loop: an IQ-stream death triggers a rebuild with
// backoff, and a subsequent healthy run keeps the loop alive (no
// fatal). Then ctx-cancel returns cleanly.
func TestRunCCDecoderWithRetry_RecoversAfterTransientDeath(t *testing.T) {
	// Shrink the backoff schedule so the test doesn't sit for 18s.
	oldBackoffs := ccDecoderRetryBackoffs
	ccDecoderRetryBackoffs = []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		50 * time.Millisecond,
	}
	defer func() { ccDecoderRetryBackoffs = oldBackoffs }()

	bus := events.NewBus(8)
	defer bus.Close()
	src := &flakyIQSource{maxDeath: 2}
	opts := ccdecoder.Options{
		Bus: bus, IQ: src, SampleRateHz: 48000,
		Log: slog.New(slog.DiscardHandler),
	}
	dec, err := ccdecoder.New(opts)
	if err != nil {
		t.Fatalf("ccdecoder.New: %v", err)
	}
	d := &Daemon{
		log:           slog.New(slog.DiscardHandler),
		bus:           bus,
		systems:       []trunking.System{},
		ccDecoder:     dec,
		ccDecoderOpts: opts,
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- d.runCCDecoderWithRetry(ctx) }()

	// Give the loop time to weather both deaths, rebuild, and settle
	// into the healthy run.
	select {
	case err := <-done:
		t.Fatalf("loop exited early: %v", err)
	case <-time.After(500 * time.Millisecond):
	}
	// Confirm no fatal was recorded.
	if got := d.takeFatal(); got != nil {
		t.Errorf("recordFatal fired during transient deaths: %v", got)
	}

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("loop exit err = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("loop did not exit after ctx cancel")
	}
}

// TestRunCCDecoderWithRetry_FatalsAfterRetriesExhausted pins the
// terminal escalation: if the IQ stream dies more times than the
// backoff schedule covers (and never gets a healthy run in between),
// the loop records a fatal so the daemon exits non-zero.
func TestRunCCDecoderWithRetry_FatalsAfterRetriesExhausted(t *testing.T) {
	oldBackoffs := ccDecoderRetryBackoffs
	ccDecoderRetryBackoffs = []time.Duration{
		5 * time.Millisecond,
		5 * time.Millisecond,
	}
	defer func() { ccDecoderRetryBackoffs = oldBackoffs }()

	bus := events.NewBus(8)
	defer bus.Close()
	// maxDeath=999 means every reopen dies — no healthy window ever
	// hits, retries get exhausted.
	src := &flakyIQSource{maxDeath: 999}
	opts := ccdecoder.Options{
		Bus: bus, IQ: src, SampleRateHz: 48000,
		Log: slog.New(slog.DiscardHandler),
	}
	dec, err := ccdecoder.New(opts)
	if err != nil {
		t.Fatalf("ccdecoder.New: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d := &Daemon{
		log:           slog.New(slog.DiscardHandler),
		bus:           bus,
		systems:       []trunking.System{},
		ccDecoder:     dec,
		ccDecoderOpts: opts,
		cancel:        cancel,
	}

	got := d.runCCDecoderWithRetry(ctx)
	if !errors.Is(got, ccdecoder.ErrIQStreamClosed) {
		t.Errorf("Run = %v, want wrapped ErrIQStreamClosed", got)
	}
	if d.takeFatal() == nil {
		t.Error("expected recordFatal to fire after retries exhausted")
	}
}

// usbDisconnectIQSource models the exact failure pattern reported in
// issue #345 after the v0.2.1 retry shipped: the first StreamIQ call
// hands out a healthy channel that the caller-side reaper closes
// (mid-stream EOF), and every subsequent StreamIQ call returns the
// underlying USB-disconnect error directly — i.e. the rebuilt decoder
// hits a dead Tuner handle that rejects the ResetBuffer control
// transfer. Without the issue-#345 follow-up fix this open-time error
// is NOT classified as ErrIQStreamClosed, so the daemon's retry loop
// returns it as-is and (with essential=false) silently swallows it.
type usbDisconnectIQSource struct {
	opens atomic.Int32
}

func (u *usbDisconnectIQSource) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	n := u.opens.Add(1)
	if n == 1 {
		ch := make(chan []complex64)
		go func() {
			defer close(ch)
			select {
			case ch <- make([]complex64, 4):
			case <-ctx.Done():
				return
			}
			// Reaper death: close mid-stream without ctx-cancel.
		}()
		return ch, nil
	}
	// Every retry sees the device gone.
	return nil, errors.New("rtl2832u: write block=1 addr=0x2148 val=0x1002: usb: device disconnected")
}

// TestRunCCDecoderWithRetry_USBDisconnectEscalatesToFatal pins the
// issue-#345 follow-up: after the initial reaper-death triggers a
// rebuild and every subsequent StreamIQ open fails with
// `usb: device disconnected`, the retry loop must classify those
// open failures as ErrIQStreamClosed (so it stays in the backoff
// schedule rather than bailing out early) and, on exhaustion, fire
// recordFatal so the supervisor restarts the process. Pre-fix this
// test fails because runCCDecoderWithRetry returns the unwrapped
// open error after one attempt and never records a fatal.
func TestRunCCDecoderWithRetry_USBDisconnectEscalatesToFatal(t *testing.T) {
	oldBackoffs := ccDecoderRetryBackoffs
	ccDecoderRetryBackoffs = []time.Duration{
		5 * time.Millisecond,
		5 * time.Millisecond,
	}
	defer func() { ccDecoderRetryBackoffs = oldBackoffs }()

	bus := events.NewBus(8)
	defer bus.Close()
	src := &usbDisconnectIQSource{}
	opts := ccdecoder.Options{
		Bus: bus, IQ: src, SampleRateHz: 48000,
		Log: slog.New(slog.DiscardHandler),
	}
	dec, err := ccdecoder.New(opts)
	if err != nil {
		t.Fatalf("ccdecoder.New: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d := &Daemon{
		log:           slog.New(slog.DiscardHandler),
		bus:           bus,
		systems:       []trunking.System{},
		ccDecoder:     dec,
		ccDecoderOpts: opts,
		cancel:        cancel,
	}

	got := d.runCCDecoderWithRetry(ctx)
	if !errors.Is(got, ccdecoder.ErrIQStreamClosed) {
		t.Errorf("Run = %v, want wrapped ErrIQStreamClosed", got)
	}
	if d.takeFatal() == nil {
		t.Error("expected recordFatal to fire after USB-disconnect retries exhausted")
	}
	// Sanity: we should have seen at least one mid-stream death and
	// one open-time disconnect, proving the new classification covers
	// both shapes.
	if got, want := src.opens.Load(), int32(2); got < want {
		t.Errorf("StreamIQ opens = %d, want >= %d", got, want)
	}
}

// reacquireSDRDevice is a minimal sdr.Device that surfaces an IQ
// stream whose behaviour is dictated by the parent driver's `dead`
// flag. The first device produced by the driver dies mid-stream; the
// driver flips dead=false before handing out the second one so the
// re-acquired handle stays healthy.
type reacquireSDRDevice struct {
	info       sdr.Info
	closed     atomic.Bool
	sampleRate uint32
	healthy    bool
}

func (d *reacquireSDRDevice) Info() sdr.Info                { return d.info }
func (d *reacquireSDRDevice) SetCenterFreq(uint32) error    { return nil }
func (d *reacquireSDRDevice) SetSampleRate(hz uint32) error { d.sampleRate = hz; return nil }
func (d *reacquireSDRDevice) SetGain(int) error             { return nil }
func (d *reacquireSDRDevice) SetPPM(int) error              { return nil }
func (d *reacquireSDRDevice) SetBiasTee(bool) error         { return nil }
func (d *reacquireSDRDevice) Close() error                  { d.closed.Store(true); return nil }
func (d *reacquireSDRDevice) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	if d.closed.Load() {
		return nil, errors.New("usb: device disconnected")
	}
	ch := make(chan []complex64)
	go func() {
		defer close(ch)
		select {
		case ch <- make([]complex64, 4):
		case <-ctx.Done():
			return
		}
		if d.healthy {
			<-ctx.Done()
			return
		}
		// Reaper death simulation: close mid-stream.
	}()
	return ch, nil
}

// reacquireSDRDriver simulates a USB device that disconnects after
// first use and re-enumerates with a new index. Tracks how many times
// Open was called so the test can assert Reacquire actually drove a
// fresh device open.
type reacquireSDRDriver struct {
	name    string
	serial  string
	opens   atomic.Int32
	healthy atomic.Bool // flipped to true before the second Open
	devices []*reacquireSDRDevice
	nextIdx int
}

func (r *reacquireSDRDriver) Name() string { return r.name }
func (r *reacquireSDRDriver) Enumerate() ([]sdr.Info, error) {
	return []sdr.Info{{Driver: r.name, Index: r.nextIdx, Serial: r.serial}}, nil
}
func (r *reacquireSDRDriver) Open(idx int) (sdr.Device, error) {
	r.opens.Add(1)
	d := &reacquireSDRDevice{
		info:    sdr.Info{Driver: r.name, Index: idx, Serial: r.serial},
		healthy: r.healthy.Load(),
	}
	r.devices = append(r.devices, d)
	// Pretend the kernel re-enumerated with a different device number.
	r.nextIdx++
	return d, nil
}

// TestRunCCDecoderWithRetry_RecoversViaPoolReacquire is the
// end-to-end pin for the issue-#345 follow-up: after the first IQ
// stream dies, runCCDecoderWithRetry must call Pool.Reacquire to swap
// the dead device handle for a freshly-opened one of the same serial
// before rebuilding the decoder, then the rebuilt decoder must stream
// happily from the new handle without recordFatal firing.
func TestRunCCDecoderWithRetry_RecoversViaPoolReacquire(t *testing.T) {
	oldBackoffs := ccDecoderRetryBackoffs
	ccDecoderRetryBackoffs = []time.Duration{
		5 * time.Millisecond,
		10 * time.Millisecond,
		25 * time.Millisecond,
	}
	defer func() { ccDecoderRetryBackoffs = oldBackoffs }()

	drv := &reacquireSDRDriver{name: "fake-reacquire-iq", serial: "TEST-CC"}
	// First Open hands out a dying device; flipping healthy=true makes
	// every subsequent Open return a healthy handle.
	sdr.Register(drv)

	pool := sdr.NewPool(slog.New(slog.DiscardHandler))
	if err := pool.Open(2_048_000, []sdr.Hint{{Serial: "TEST-CC", Role: sdr.RoleControl}}); err != nil {
		t.Fatalf("pool.Open: %v", err)
	}
	defer pool.Close()
	// Now any further Open()s from the driver hand out healthy
	// devices — this matches the reporter's scenario where the dongle
	// successfully re-enumerated under the same serial.
	drv.healthy.Store(true)

	controlEntry := pool.FirstByRole(sdr.RoleControl)
	if controlEntry == nil {
		t.Fatal("no control entry after pool.Open")
	}
	bus := events.NewBus(8)
	defer bus.Close()
	opts := ccdecoder.Options{
		Bus:          bus,
		IQ:           controlEntry.Device,
		Tuner:        controlEntry.Device,
		SampleRateHz: 2_048_000,
		Log:          slog.New(slog.DiscardHandler),
	}
	dec, err := ccdecoder.New(opts)
	if err != nil {
		t.Fatalf("ccdecoder.New: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	d := &Daemon{
		log:               slog.New(slog.DiscardHandler),
		bus:               bus,
		systems:           []trunking.System{},
		pool:              pool,
		controlSerial:     "TEST-CC",
		controlSampleRate: 2_048_000,
		ccDecoder:         dec,
		ccDecoderOpts:     opts,
		cancel:            cancel,
	}

	done := make(chan error, 1)
	go func() { done <- d.runCCDecoderWithRetry(ctx) }()

	// Wait long enough for: first death -> backoff -> reacquire ->
	// rebuild against healthy handle -> steady-state Run.
	select {
	case err := <-done:
		t.Fatalf("loop exited early: %v", err)
	case <-time.After(300 * time.Millisecond):
	}

	if got := d.takeFatal(); got != nil {
		t.Errorf("recordFatal fired during reacquire recovery: %v", got)
	}
	// Driver should have been opened at least twice: once at pool.Open
	// and once via Reacquire.
	if got := drv.opens.Load(); got < 2 {
		t.Errorf("driver.Open count = %d, want >= 2 (initial + reacquire)", got)
	}
	// The first (dying) device must have been Closed during Reacquire.
	if !drv.devices[0].closed.Load() {
		t.Error("stale device handle was not Closed during Reacquire")
	}

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("loop exit err = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("loop did not exit after ctx cancel")
	}
}
