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
	p25phase2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// TestDaemonCCDecodesP25Phase2 is the per-protocol sibling of
// integration-cc-tetra (PR #154). Boots the daemon with a mock SDR
// replaying synthesized P25 Phase 2 H-DQPSK IQ in the real TDMA air
// format (S-/I-ISCH timeslot grid, clear LCCH signalling bursts with
// RS(63,35) + CRC-16) and asserts the production newP25Phase2Pipeline
// (Gardner clock recovery → air framer → MAC message decode) +
// supervisor + API + metrics chain recovers the lock.
func TestDaemonCCDecodesP25Phase2(t *testing.T) {
	const (
		controlFreqHz = 851_062_500
		// 6000 × 8 = 48 kHz IQ rate; sps = 8 (exact integer).
		sampleRate = 48_000.0
		sps        = 8
		span       = 8
		alpha      = 0.20
		// Each repeat is a full 12-timeslot superframe; a handful is
		// ample for the Gardner clock to settle and the framer to lock.
		superframeRepeats = 8
	)

	dibits := buildP25Phase2LCCHStream(t, superframeRepeats)
	iq := demod.ModulateHDQPSKSpec(dibits, sps, span, alpha)

	dir := t.TempDir()
	iqPath := filepath.Join(dir, "p25p2-cc.cfile")
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
			Name:            "P25P2Site",
			Protocol:        "p25-phase2",
			ControlChannels: []uint32{controlFreqHz},
		},
	}
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.Metrics.Enabled = true

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "integration-cc-p25p2", logger)
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
			ls, ok := ev.Payload.(p25phase2.LockState)
			if !ok {
				t.Errorf("CCLocked payload type = %T, want p25phase2.LockState", ev.Payload)
				continue
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

	waitForScannerLock(t, base, "P25P2Site", 2*time.Second)

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

// buildP25Phase2LCCHStream assembles a P25 Phase 2 control-channel dibit
// stream in the real TDMA air format: a 400-dibit lead-in, then `repeats`
// superframes whose every timeslot is a clear LCCH carrying a MAC_SIGNAL
// PDU with a Group Voice Channel User message (the canonical non-idle
// "lock me" message; a clear LCCH decodes with no site identity), then a
// trailer.
func buildP25Phase2LCCHStream(t *testing.T, repeats int) []uint8 {
	t.Helper()
	gvcu := []byte{byte(p25phase2.OpGroupVoiceChannelUserAbbreviated), 0, 0x4E, 0xEA, 0x04, 0xCF, 0x43}
	pdu, err := p25phase2.AirMACPDUBytes(p25phase2.AirMACSignal, 0, 21, gvcu)
	if err != nil {
		t.Fatal(err)
	}
	var slots [p25phase2.AirSlotsPerSuperframe]p25phase2.AirSlotContent
	for i := range slots {
		slots[i] = p25phase2.AirSlotContent{DUID: p25phase2.DUIDLCCHClear, MAC: pdu}
	}
	out := make([]uint8, 0, 400+(repeats+1)*p25phase2.AirSlotsPerSuperframe*p25phase2.AirPacketDibits+100)
	for i := 0; i < 400; i++ {
		out = append(out, uint8(i&3))
	}
	for r := 0; r <= repeats; r++ {
		sf, err := p25phase2.EncodeAirSuperframe(slots, nil, uint8(r&3))
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, sf...)
	}
	for i := 0; i < 100; i++ {
		out = append(out, uint8(i&3))
	}
	return out
}
