//go:build integration

package main

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/motorola"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// TestDaemonCCDecodesMotorola asserts the production daemon chain —
// mock SDR at a wideband 90 kHz rate → effectiveStreamRate → DDC to
// the 18 kHz Motorola channel target → FSK receiver → SmartNet framer
// (8-bit sync-bracketed frames, interleave + parity ECC + CRC-10) →
// OSW sequencer → supervisor + API + metrics — recovers the two-OSW
// system-ID broadcast and publishes cc.locked with the system ID.
//
// The stream is rendered in the REAL air format via
// motorola.EncodeOSWFrame (constants ported from OP25 rx_smartnet;
// issue #1143 — the previous fixture used a made-up framing that only
// its own decoder understood, so this test was green while no real
// system could ever lock).
func TestDaemonCCDecodesMotorola(t *testing.T) {
	const (
		controlFreqHz = 854_562_500
		// Wideband input rate: the daemon's DDC must decimate 5:1
		// to the 18 kHz channel target, as it would from a real SDR.
		sampleRate = 90_000.0
		sps        = 25 // 90 kHz / 3600 baud
		systemID   = uint16(0x4567)
		repeats    = 12
	)

	seq := []motorola.OSW{
		{Address: systemID, Command: motorola.CmdFirstNormal},
		{Address: 0x1F00, Command: 0x8E}, // CC broadcast: channel 0x8E = 854.5625 MHz
		{Address: 0x02F8, Command: motorola.CmdIdle},
	}
	var bits []byte
	for i := 0; i < 200; i++ {
		bits = append(bits, byte(i&1))
	}
	for r := 0; r < repeats; r++ {
		for _, o := range seq {
			bits = append(bits, motorola.EncodeOSWFrame(o)...)
		}
	}
	bits = append(bits, motorola.OutboundSyncBits()...)

	// Real SmartNet FSK deviation is ±1.2 kHz.
	iq := demod.ModulateGFSK(bits, sps, 4, 0.5, sampleRate, 1200.0)

	dir := t.TempDir()
	iqPath := filepath.Join(dir, "motorola-cc.cfile")
	if err := writeIQToU8File(iqPath, iq); err != nil {
		t.Fatalf("write IQ: %v", err)
	}
	sdr.Register(&sdr.MockDriver{Files: []string{iqPath}})

	cfg := config.Default()
	cfg.SDR.SampleRate = uint32(sampleRate)
	cfg.SDR.Devices = []config.DeviceConfig{
		{Serial: "mock-00", Role: "control"},
	}
	cfg.Trunking.Systems = []config.SystemConfig{
		{
			Name:            "MotoSite",
			Protocol:        "motorola",
			ControlChannels: []uint32{controlFreqHz},
		},
	}
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.Metrics.Enabled = true

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "integration-cc-motorola", logger)
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	if d.ccDecoder == nil {
		t.Fatalf("ccDecoder is nil; daemon should have constructed one")
	}

	sub := d.Bus().Subscribe()
	defer sub.Close()

	ctx, cancel := context.WithCancel(context.Background())
	runErrCh := make(chan error, 1)
	go func() { runErrCh <- d.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-runErrCh:
		case <-time.After(3 * time.Second):
		}
	})

	base := "http://" + cfg.API.HTTPAddr
	waitReachable(t, base+"/api/v1/health", 3*time.Second)

	deadline := time.After(5 * time.Second)
	var locked bool
WaitLoop:
	for !locked {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindCCLocked {
				continue
			}
			ls, ok := ev.Payload.(motorola.LockState)
			if !ok {
				t.Errorf("CCLocked payload type = %T, want motorola.LockState", ev.Payload)
				continue
			}
			if ls.SystemID != systemID {
				t.Errorf("LockState.SystemID = %#x, want %#x", ls.SystemID, systemID)
			}
			if ls.FrequencyHz != controlFreqHz {
				t.Errorf("LockState.FrequencyHz = %d, want %d", ls.FrequencyHz, controlFreqHz)
			}
			locked = true
			break WaitLoop
		case <-deadline:
			t.Fatalf("no cc.locked event arrived within 5s")
		}
	}

	waitForScannerLock(t, base, "MotoSite", 2*time.Second)

	// Poll rather than scrape once. The metrics collector consumes the events
	// bus on its own goroutine (internal/metrics/prom.go observeEvent), so when
	// this test's own subscriber sees cc.locked the collector may still be
	// mid-handler — it materialises the events_total child before incrementing
	// it and only reaches the gauge afterwards, so a single scrape can catch
	// `cc.locked 0` with the gauge family absent. That is the exact state a
	// loaded -race CI runner produced.
	waitForMetric(t, base, "gophertrunk_control_channel_locked{", 2*time.Second)
	waitForMetric(t, base, `gophertrunk_events_total{kind="cc.locked"} 1`, 2*time.Second)
}
