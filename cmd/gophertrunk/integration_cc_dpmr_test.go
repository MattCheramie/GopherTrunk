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
	"github.com/MattCheramie/GopherTrunk/internal/radio/dpmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// TestDaemonCCDecodesDPMR is the per-protocol sibling of the
// P25 P1 / NXDN / DMR Tier III integration-cc tests. Boots the
// daemon with a mock SDR replaying a fully-synthesized dPMR
// Mode 3 control-channel stream (24-dibit FS3 sync + 40-dibit /
// 80-bit StandingServiceStatus CSBK), and asserts the production
// newDPMRPipeline + supervisor + API + metrics chain recovers
// the lock.
//
// dPMR Mode 3 runs 4-level C4FM at 2400 sym/s — half the symbol
// rate of P25 P1 / DMR — with 900 Hz peak deviation and α = 0.20
// RRC. The C4FM modulator from PR #148 handles it; per-protocol
// differences from the P25 P1 test are the symbol rate (2400
// instead of 4800), the deviation (900 instead of 1800), and the
// framing (FS3 sync + 80-bit CSBK vs FSW + NID + TSBK).
func TestDaemonCCDecodesDPMR(t *testing.T) {
	const (
		controlFreqHz = 446_018_750
		sampleRateHz  = 48_000
		// At 2400 sym/s with 48 kHz sample rate, sps = 20 — twice
		// the P25 P1 / DMR / NXDN value because dPMR is half-rate.
		sps         = 20
		span        = 8
		alpha       = 0.20
		deviationHz = 900.0
		systemID    = uint32(0x123456)
		csbkRepeats = 30
	)

	dibits := buildDPMRSiteBroadcastDibits(csbkRepeats, systemID)
	iq := demod.ModulateC4FM(dibits, sps, span, alpha, sampleRateHz, deviationHz)

	dir := t.TempDir()
	iqPath := filepath.Join(dir, "dpmr-cc.cfile")
	if err := writeIQToU8File(iqPath, iq); err != nil {
		t.Fatalf("write IQ: %v", err)
	}
	sdr.Register(&sdr.MockDriver{Files: []string{iqPath}})

	cfg := config.Default()
	cfg.SDR.SampleRate = sampleRateHz
	cfg.SDR.Devices = []config.DeviceConfig{
		{Serial: "mock-00", Role: "control"},
	}
	cfg.Trunking.Systems = []config.SystemConfig{
		{Name: "DPMRSite", Protocol: "dpmr", ControlChannels: []uint32{controlFreqHz}},
	}
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.Metrics.Enabled = true

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "integration-cc-dpmr", logger)
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

	// 30s deadline — six times the value the P25 / DMR / NXDN
	// integration tests use, but well over the empirically observed
	// worst-case under -race (about 10 s on 1-in-30 runs locally; the
	// other 29 finish in 0.4 s). dPMR runs at half the symbol rate
	// (2400 vs 4800 sym/s) so the same mock-SDR IQ chunk carries half
	// the dibits per second, and the daemon goroutine occasionally
	// gets scheduled out under the race detector long enough that the
	// 5 s deadline the sibling tests use is borderline. The 30 s ceiling
	// keeps the CI cycle bounded while removing the flake (the proper
	// fix — making the test deterministic on event ordering rather
	// than wall time — is a larger refactor across all the
	// integration-cc tests).
	deadline := time.After(30 * time.Second)
	var locked bool
WaitLoop:
	for !locked {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindCCLocked {
				continue
			}
			ls, ok := ev.Payload.(dpmr.LockState)
			if !ok {
				t.Errorf("CCLocked payload type = %T, want dpmr.LockState", ev.Payload)
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
			t.Fatalf("no cc.locked event arrived within 30s")
		}
	}

	waitForScannerLock(t, base, "DPMRSite", 2*time.Second)

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

// buildDPMRSiteBroadcastDibits assembles a dPMR Mode 3 dibit
// stream for the C4FM modulator + receiver chain:
//
//   - 400-dibit warmup cycling 0..3 so the Mueller-Müller clock
//     recovery sees every symbol level (dPMR's half-rate
//     symbol stream means each warmup dibit covers twice the
//     wall-time of a P25 P1 / DMR dibit; the longer count
//     keeps the convergence time comparable in absolute terms)
//   - `repeats` × (24-dibit FS3 sync + 40-dibit CSBK + 16 idle
//     dibits)
//   - 100-dibit trailer for clean flush
//
// Each frame's CSBK is a StandingServiceStatus carrying the
// requested SystemID — the dPMR control state machine's
// canonical "lock me" message.
func buildDPMRSiteBroadcastDibits(repeats int, systemID uint32) []uint8 {
	csbk := dpmr.CSBK{Type: dpmr.MsgStandingServiceStatus, DestID: systemID, Extra: 0x42}
	csbkBits := dpmr.CSBKBits(csbk)
	csbkDibits := framing.BitsToDibits(csbkBits)

	frame := make([]uint8, 0, 24+len(csbkDibits))
	frame = append(frame, dpmr.FS3Dibits()...)
	frame = append(frame, csbkDibits...)

	out := make([]uint8, 0, 400+repeats*(len(frame)+16)+100)
	for i := 0; i < 400; i++ {
		out = append(out, uint8(i&3))
	}
	for r := 0; r < repeats; r++ {
		out = append(out, frame...)
		for i := 0; i < 16; i++ {
			out = append(out, uint8(i&3))
		}
	}
	for i := 0; i < 100; i++ {
		out = append(out, uint8(i&3))
	}
	return out
}
