package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/iqtap"
)

// singleStreamDevice is a minimal sdr.Device that models the real driver
// contract the bug hinges on: only ONE StreamIQ session may be active at a
// time. A second concurrent StreamIQ returns errStreamActive, exactly like the
// rtlsdr backend's "stream already active" that issue #1075 reports. The active
// session self-releases on ctx cancel.
type singleStreamDevice struct {
	serial string
	out    chan []complex64
	active atomic.Bool
}

var errStreamActive = &streamActiveErr{}

type streamActiveErr struct{}

func (*streamActiveErr) Error() string { return "rtlsdr: stream already active" }

func newSingleStreamDevice(serial string) *singleStreamDevice {
	return &singleStreamDevice{serial: serial, out: make(chan []complex64, 4)}
}

func (d *singleStreamDevice) Info() sdr.Info             { return sdr.Info{Driver: "fake", Serial: d.serial} }
func (d *singleStreamDevice) SetCenterFreq(uint32) error { return nil }
func (d *singleStreamDevice) SetSampleRate(uint32) error { return nil }
func (d *singleStreamDevice) SetGain(int) error          { return nil }
func (d *singleStreamDevice) SetPPM(int) error           { return nil }
func (d *singleStreamDevice) SetBiasTee(bool) error      { return nil }
func (d *singleStreamDevice) Close() error               { return nil }

func (d *singleStreamDevice) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	if !d.active.CompareAndSwap(false, true) {
		return nil, errStreamActive
	}
	go func() {
		<-ctx.Done()
		d.active.Store(false)
	}()
	return d.out, nil
}

// TestConvScanVoiceSourceFansOutInsteadOfSecondStream is the issue #1075
// regression. A conventional-scanner SDR is single-consumer: the scanner holds
// the device's StreamIQ open for the whole dwell/call. Before the fix the
// composer resolved that serial to the bare device and opened a SECOND StreamIQ
// for its FM voice chain — which fails with "stream already active", so no PCM
// ever reaches the recorder and no WAV is written.
//
// The fix routes the scanner through the device's iqtap broker (primary) and
// registers the serial in poolDevices.scannerBrokers, so the composer resolves
// it to a convScanVoiceSource that Subscribes to the fan-out. This test proves
// both halves against the SAME device: a direct second StreamIQ still errors
// (documenting the bug), while the broker-subscribe path delivers the exact IQ
// the scanner is reading, concurrently, with no error.
func TestConvScanVoiceSourceFansOutInsteadOfSecondStream(t *testing.T) {
	const serial = "SDRV4-03"
	const rate = 2_400_000

	dev := newSingleStreamDevice(serial)
	broker := iqtap.New(dev, 0, slog.New(slog.NewTextHandler(io.Discard, nil)))

	// The scanner is the broker's primary StreamIQ consumer for the life of
	// the call.
	scanCtx, scanCancel := context.WithCancel(context.Background())
	defer scanCancel()
	primary, err := broker.StreamIQ(scanCtx)
	if err != nil {
		t.Fatalf("scanner primary StreamIQ: %v", err)
	}

	// Sanity: the device really is single-consumer — a bare second StreamIQ
	// (the pre-fix composer path) fails exactly as issue #1075 reports.
	if _, err := dev.StreamIQ(context.Background()); err != errStreamActive {
		t.Fatalf("expected a second StreamIQ on the bare device to fail with %v, got %v — "+
			"the test can't demonstrate the #1075 collision it guards against", errStreamActive, err)
	}

	// The composer resolves the scanner's serial through poolDevices. With the
	// serial registered in scannerBrokers it must return a convScanVoiceSource,
	// NOT the bare device.
	pd := &poolDevices{
		pool:           nil,
		rateHz:         rate,
		scannerBrokers: map[string]*iqtap.Broker{serial: broker},
	}
	src := pd.FindBySerial(serial)
	if src == nil {
		t.Fatal("poolDevices.FindBySerial returned nil for the scanner serial")
	}
	if _, ok := src.(*convScanVoiceSource); !ok {
		t.Fatalf("poolDevices.FindBySerial returned %T, want *convScanVoiceSource — "+
			"the FM voice chain would open a colliding second StreamIQ (#1075)", src)
	}
	if got := src.SampleRateHz(); got != rate {
		t.Fatalf("convScanVoiceSource.SampleRateHz = %d, want %d (the FM chain sizes its decimator from this)", got, rate)
	}

	// The composer's chain opens its IQ via the resolved source. This must NOT
	// error (no second StreamIQ) and must deliver the scanner's IQ.
	voiceCtx, voiceCancel := context.WithCancel(context.Background())
	defer voiceCancel()
	voiceIQ, err := src.StreamIQ(voiceCtx)
	if err != nil {
		t.Fatalf("convScanVoiceSource.StreamIQ errored (%v) — the #1075 collision is not fixed", err)
	}

	// A chunk the scanner reads on its primary stream must fan out to the voice
	// chain unchanged. Drain the primary too so the broker's fan-out goroutine
	// never blocks on the primary handoff.
	want := []complex64{complex(0.1, 0.2), complex(-0.3, 0.4), complex(0.5, -0.6)}
	go func() { dev.out <- want }()
	go func() { <-primary }()

	select {
	case got, ok := <-voiceIQ:
		if !ok {
			t.Fatal("voice IQ channel closed before delivering a chunk")
		}
		if len(got) != len(want) {
			t.Fatalf("voice chunk len = %d, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("voice[%d] = %v, want %v", i, got[i], want[i])
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the scanner's IQ to fan out to the voice chain — " +
			"the composer is not receiving the conventional call's audio (#1075)")
	}

	// Cancelling the voice context closes the subscription so the chain's read
	// loop ends.
	voiceCancel()
	select {
	case _, ok := <-voiceIQ:
		if ok {
			// Drain any in-flight buffered chunk, then confirm close.
			select {
			case _, ok2 := <-voiceIQ:
				if ok2 {
					t.Fatal("voice IQ channel did not close after ctx cancel")
				}
			case <-time.After(time.Second):
				t.Fatal("voice IQ channel did not close after ctx cancel")
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("voice IQ channel did not close after ctx cancel")
	}
}

// TestConventionalScannerRegistersBrokerVoiceSource is the daemon-wiring half of
// the #1075 regression: when a conventional scanner is configured, the daemon
// must route the scanner through its device's iqtap broker and register the
// serial so the composer's poolDevices resolves it to a Subscribe-based source.
// Before the fix the serial was absent from scannerBrokers, so the composer fell
// through to the bare device and opened a colliding second StreamIQ.
func TestConventionalScannerRegistersBrokerVoiceSource(t *testing.T) {
	// A mock voice SDR so the pool has a Voice entry (and thus an iqtap broker)
	// for the scanner to claim — no hardware required.
	iqPath := filepath.Join(t.TempDir(), "voice.cfile")
	// The daemon is constructed but never Run here, so the mock device's file is
	// never actually streamed — it only has to exist for pool.Open to enumerate
	// the device. A short u8 IQ payload is enough.
	if err := os.WriteFile(iqPath, make([]byte, 4096), 0o644); err != nil {
		t.Fatalf("write mock IQ: %v", err)
	}
	sdr.Register(&sdr.MockDriver{Files: []string{iqPath}})

	cfg := config.Default()
	cfg.SDR.SampleRate = 2_400_000
	cfg.SDR.Devices = []config.DeviceConfig{{Serial: "mock-00", Role: "voice"}}
	cfg.Scanner.Conventional = []config.ConvChannelConfig{
		{Label: "KP4-DH", FrequencyHz: 146_670_000, Mode: "nfm", SquelchDbFS: -10},
	}
	cfg.Recordings.Dir = t.TempDir()
	cfg.Storage.Path = ""
	cfg.Storage.CCCacheFile = ""
	cfg.API.HTTPAddr = ""

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "test-convscan", logger)
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	t.Cleanup(d.Close)

	if d.convScan == nil {
		t.Fatal("conventional scanner was not constructed from a scanner.conventional config")
	}
	if _, ok := d.scannerBrokers["mock-00"]; !ok {
		t.Fatal("scanner serial not registered in d.scannerBrokers — the composer will open a " +
			"colliding second StreamIQ on the scanner device (#1075)")
	}
	// The composer's device resolver must return the Subscribe-based source for
	// the scanner serial, not the bare device.
	pd := &poolDevices{pool: d.pool, rateHz: cfg.SDR.SampleRate, virtualMap: d.virtualVoiceMap(), scannerBrokers: d.scannerBrokers}
	src := pd.FindBySerial("mock-00")
	if _, ok := src.(*convScanVoiceSource); !ok {
		t.Fatalf("poolDevices resolved the scanner serial to %T, want *convScanVoiceSource (#1075)", src)
	}
}
