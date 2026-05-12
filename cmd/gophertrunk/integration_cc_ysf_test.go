//go:build integration

package main

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/ysf"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// TestDaemonCCDecodesYSF is the last per-protocol integration
// test on the original plan's "lights up live trunked reception"
// list. YSF (System Fusion) uses the same 4800-baud C4FM
// modulation as P25 P1 / NXDN / DMR / dPMR, just with different
// framing (480-dibit frames, 20-dibit FSW, 100-dibit FICH). The
// C4FM modulator from PR #148 covers it directly.
//
// Boots the daemon with a mock SDR replaying synthesized YSF IQ
// (back-to-back 480-dibit frames with FSWPattern at offset 0 +
// zero-filled FICH + payload regions) and asserts the production
// newYSFPipeline + supervisor + API + metrics chain recovers the
// lock.
//
// The plan flagged "YSF on-air interleaver / puncture validation"
// as out of scope this round — that's the deeper capture-driven
// validation of the FICH FEC chain against a real-air capture.
// This test covers the basic "feeds CC" path the plan calls for
// in A.2's verification matrix.
func TestDaemonCCDecodesYSF(t *testing.T) {
	const (
		controlFreqHz = 444_525_000
		sampleRateHz  = 48_000
		sps           = 10
		span          = 8
		alpha         = 0.20
		deviationHz   = 1800.0
		frameRepeats  = 30
	)

	dibits := buildYSFFSWStream(frameRepeats)
	iq := demod.ModulateC4FM(dibits, sps, span, alpha, sampleRateHz, deviationHz)

	dir := t.TempDir()
	iqPath := filepath.Join(dir, "ysf-cc.cfile")
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
		{Name: "YSFSite", Protocol: "ysf", ControlChannels: []uint32{controlFreqHz}},
	}
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.Metrics.Enabled = true

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "integration-cc-ysf", logger)
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
			ls, ok := ev.Payload.(ysf.LockState)
			if !ok {
				t.Errorf("CCLocked payload type = %T, want ysf.LockState", ev.Payload)
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

	waitForScannerLock(t, base, "YSFSite", 2*time.Second)

	body := scrape(t, base+"/metrics")
	if !strings.Contains(body, "gophertrunk_control_channel_locked{") {
		t.Errorf("/metrics missing gophertrunk_control_channel_locked gauge family:\n%s", body)
	}
	if !strings.Contains(body, `gophertrunk_events_total{kind="cc.locked"} 1`) {
		t.Errorf("/metrics did not count one cc.locked event:\n%s", body)
	}
}

// buildYSFFSWStream assembles a YSF dibit stream for the C4FM
// modulator + receiver chain:
//
//   - 400-dibit warmup cycling 0..3 so the Mueller-Müller clock
//     recovery sees every symbol level
//   - `repeats` × 480-dibit YSF frame skeleton (FSWPattern at
//     offset 0 + zero-filled FICH + payload regions — same shape
//     as the in-package streamWithFSWAt helper)
//   - 100-dibit trailer for clean flush
//
// YSF's CC state machine emits cc.locked on the first valid
// FSWPattern hit; the FICH FEC chain isn't required for the
// basic lock event the integration test asserts.
func buildYSFFSWStream(repeats int) []uint8 {
	frame := make([]uint8, ysf.FrameDibits)
	copy(frame[ysf.FSWOffset:], ysf.FSWPattern)

	out := make([]uint8, 0, 400+repeats*len(frame)+100)
	for i := 0; i < 400; i++ {
		out = append(out, uint8(i&3))
	}
	for r := 0; r < repeats; r++ {
		out = append(out, frame...)
	}
	for i := 0; i < 100; i++ {
		out = append(out, uint8(i&3))
	}
	return out
}
