//go:build integration

package main

import (
	"context"
	"io"
	"log/slog"
	"math/rand"
	"path/filepath"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dpmr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestDaemonCCDecodesDPMR boots the daemon with a mock SDR replaying a dPMR
// Mode 3 beacon channel in the air format (ETSI TS 102 658 clause 4.2.3.7:
// FS1, then framesets of a message and three SYScasts), and asserts the
// production newDPMRPipeline + supervisor + API + metrics chain locks on the
// beacon's System Identity Code and turns its Goto Channel into a "dpmr"
// grant (#1374).
//
// dPMR is 4-level FSK at 2400 symbols a second, +-1050 Hz at the outer
// symbols, root raised cosine 0.2.
func TestDaemonCCDecodesDPMR(t *testing.T) {
	const (
		controlFreqHz = 446_018_750
		sampleRateHz  = 48_000
		sps           = 20 // 2400 symbols a second at 48 kHz
		span          = 8
		alpha         = 0.20
		deviationHz   = 1050.0
		systemID      = uint32(0x0C55)
		framesets     = 40
	)

	dibits := buildDPMRBeaconDibits(framesets, uint16(systemID))
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

	// The beacon's Goto Channel becomes a dpmr grant for the right call.
	grantDeadline := time.After(30 * time.Second)
GrantLoop:
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindGrant {
				continue
			}
			g, ok := ev.Payload.(trunking.Grant)
			if !ok || g.Protocol != "dpmr" {
				continue
			}
			if g.GroupID != 0x02C00A || g.SourceID != 0x1B91FD || g.FrequencyHz != 446_006_250 {
				t.Errorf("grant %+v, want talkgroup 02C00A from 1B91FD on 446.00625 MHz", g)
			}
			break GrantLoop
		case <-grantDeadline:
			t.Fatalf("no dpmr grant arrived within 30s")
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

// buildDPMRBeaconDibits assembles a beacon downlink: random dibits for the
// receiver's loops to settle on, FS1, then framesets of an Aloha carrying
// the System Identity Code and, every fourth, a Goto Channel sending
// talkgroup 012345* to 446.00625 MHz with the calling party 1234567 in its
// SYScasts.
func buildDPMRBeaconDibits(framesets int, sys uint16) []uint8 {
	r := rand.New(rand.NewSource(1))
	var out []uint8
	for i := 0; i < 400; i++ {
		out = append(out, uint8(r.Intn(4)))
	}
	out = append(out, dpmr.AirFS1Dibits()...)
	sy1 := dpmr.EncodeSYScast1(dpmr.SYScast1{SystemCode: sys})
	band, sep, field, _ := dpmr.FrequencyFields(446_006_250)
	for i := 0; i < framesets; i++ {
		if i%4 == 3 {
			g := dpmr.EncodeGotoChannel(dpmr.GotoChannel{Called: 0x02C00A, RxField: field, TxField: field,
				Band: band, Sep5k: sep, Mode: dpmr.AirModeVoice})
			a, b := dpmr.EncodeCallingParty(0x1B91FD)
			out = append(out, dpmr.EncodeBeaconFrameset(g, dpmr.BeaconChannelCode0, [3]uint32{sy1, a, b})...)
			continue
		}
		cfc := dpmr.EncodeSYScast2(dpmr.SYTypeFrameCounter, uint16(i))
		out = append(out, dpmr.EncodeBeaconFrameset(dpmr.EncodeAloha(dpmr.Aloha{MSID: 0xFFFFFF, SystemCode: sys}),
			dpmr.BeaconChannelCode0, [3]uint32{sy1, cfc, cfc})...)
	}
	for i := 0; i < 100; i++ {
		out = append(out, uint8(r.Intn(4)))
	}
	return out
}
