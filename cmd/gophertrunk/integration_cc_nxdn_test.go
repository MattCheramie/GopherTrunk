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
	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// TestDaemonCCDecodesNXDN is the per-protocol sibling of
// TestDaemonCCDecodesP25Phase1: boot the daemon with a mock SDR
// replaying a fully-synthesized NXDN-TS-1-A §4.6 RCCH outbound
// frame (FSW + LICH + 150-dibit CAC carrying a SITE_INFO RCCH
// message through the §4.5.1.1 spec-correct FEC chain), assert
// the production newNXDNPipeline + `nxdn_viterbi_mode: spec`
// recovers the lock and surfaces it through the bus + API +
// metrics.
//
// The synthesized IQ is the same 9600-baud / 4-FSK / α = 0.20 /
// 1800 Hz peak deviation modulation used by P25 Phase 1 — NXDN
// shares the modulation params, just differs in the framing and
// channel-coding layers above the demod. The newly-shipped C4FM
// modulator (PR #148) carries straight over; the receiver-side
// slicer calibration via Options.DeviationHz is the only other
// piece the test needs to round-trip cleanly.
func TestDaemonCCDecodesNXDN(t *testing.T) {
	const (
		controlFreqHz = 851_062_500
		sampleRateHz  = 48_000
		sps           = 10
		span          = 8
		alpha         = 0.20
		deviationHz   = 1800.0
		frameRepeats  = 20
		// A Global location: system 0x042 (10 bits), site 0xEEF (12 bits).
		location = nxdn.LocationID(0x042EEF)
		siteID   = 0xEEF
		systemID = 0x042
	)

	dibits := buildNXDNAirControlDibits(frameRepeats, location)
	iq := demod.ModulateC4FM(dibits, sps, span, alpha, sampleRateHz, deviationHz)

	dir := t.TempDir()
	iqPath := filepath.Join(dir, "nxdn-cc.cfile")
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
		{
			Name:            "NXDNSite",
			Protocol:        "nxdn",
			ControlChannels: []uint32{controlFreqHz},
			NXDNViterbiMode: "spec",
		},
	}
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.Metrics.Enabled = true

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "integration-cc-nxdn", logger)
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
			ls, ok := ev.Payload.(nxdn.LockState)
			if !ok {
				t.Errorf("CCLocked payload type = %T, want nxdn.LockState", ev.Payload)
				continue
			}
			if ls.SiteID != siteID {
				t.Errorf("LockState.SiteID = %#x, want %#x", ls.SiteID, siteID)
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

	waitForScannerLock(t, base, "NXDNSite", 2*time.Second)

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

// buildNXDNAirControlDibits assembles a long NXDN control-channel dibit
// stream on the real air format (#1261) for the C4FM modulator + receiver
// chain: 300 pseudo-random dibits of lead-in, `repeats` back-to-back RCCH
// frames (nxdn.EncodeAirControlFrame: FSW, outbound RCCH LICH and a CAC
// carrying SITE_INFO for loc, every post-FSW dibit scrambled), then 100 more
// pseudo-random dibits. The earlier fixture used an invented FSW, LICH coding
// and CAC chain that no transmitter sends; the production control channel
// now locks through the real one.
func buildNXDNAirControlDibits(repeats int, loc nxdn.LocationID) []uint8 {
	msg := make([]byte, 19)
	msg[0] = nxdn.AirMsgSiteInfo
	msg[1], msg[2], msg[3] = byte(loc>>16), byte(loc>>8), byte(loc)
	_, _, site := loc.Split()
	frame := nxdn.EncodeAirControlFrame(uint8(site%64), msg)

	rng := uint32(7)
	noise := func(n int) []uint8 {
		out := make([]uint8, n)
		for i := range out {
			rng = rng*1664525 + 1013904223
			out[i] = uint8(rng >> 30)
		}
		return out
	}
	out := noise(300)
	for r := 0; r < repeats; r++ {
		out = append(out, frame...)
	}
	return append(out, noise(100)...)
}
