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
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier3"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// TestDaemonCCDecodesDMRTier3 is the per-protocol sibling of
// TestDaemonCCDecodesP25Phase1 / NXDN: boot the daemon with a
// mock SDR replaying a fully-synthesized DMR Tier III burst
// (132 dibits = 49 first-half payload + 5 slot-type + 24 sync +
// 5 slot-type + 49 second-half payload, with the payload halves
// carrying an Aloha CSBK encoded through BPTC(196, 96)), and
// assert the production newDMRTier3Pipeline + supervisor + API
// + metrics chain recovers the lock.
//
// The modulation is 4800-baud 4-FSK at α = 0.20 with 1944 Hz
// peak deviation per ETSI TS 102 361-1 §6.3 — the same C4FM
// modulator from PR #148 handles it; the deviation is the only
// per-protocol calibration knob and is plumbed through both the
// modulator and the receiver via Options.DeviationHz.
//
// DMR's burst-oriented sync detection (multi-pattern against all
// 9 ETSI sync words) makes this the strictest test of the
// pre-PR #147 race fix on the ccdecoder bus subscription —
// without that fix the supervisor's first HuntProgress race
// against ccdecoder.Run's subscription could swallow the first
// burst window.
func TestDaemonCCDecodesDMRTier3(t *testing.T) {
	const (
		controlFreqHz = 460_000_000
		sampleRateHz  = 48_000
		sps           = 10
		span          = 8
		alpha         = 0.20
		deviationHz   = 1944.0
		colorCode     = 0xA
		systemID      = 0x1234
		burstRepeats  = 80
	)

	dibits := buildDMRTier3CSBKDibits(burstRepeats, colorCode, systemID)
	iq := demod.ModulateC4FM(dibits, sps, span, alpha, sampleRateHz, deviationHz)

	dir := t.TempDir()
	iqPath := filepath.Join(dir, "dmr-cc.cfile")
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
		{Name: "DMRSite", Protocol: "dmr", ControlChannels: []uint32{controlFreqHz}},
	}
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.Metrics.Enabled = true

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "integration-cc-dmr", logger)
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
			ls, ok := ev.Payload.(tier3.LockState)
			if !ok {
				t.Errorf("CCLocked payload type = %T, want tier3.LockState", ev.Payload)
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

	waitForScannerLock(t, base, "DMRSite", 2*time.Second)

	body := scrape(t, base+"/metrics")
	if !strings.Contains(body, "gophertrunk_control_channel_locked{") {
		t.Errorf("/metrics missing gophertrunk_control_channel_locked gauge family:\n%s", body)
	}
	if !strings.Contains(body, `gophertrunk_events_total{kind="cc.locked"} 1`) {
		t.Errorf("/metrics did not count one cc.locked event:\n%s", body)
	}
}

// buildDMRTier3CSBKDibits assembles a DMR Tier III dibit stream
// for the C4FM modulator + receiver chain:
//
//   - 300-dibit warmup cycling 0..3 so the Mueller-Müller clock
//     recovery sees every symbol level
//   - `repeats` × (132-dibit burst + 32 idle dibits)
//   - 100-dibit trailer for clean flush
//
// Each burst layout matches the DMR Tier III in-package
// buildAlohaBurst fixture exactly: BPTC(196, 96)-encoded Aloha
// CSBK split into 49-dibit first half + 49-dibit second half,
// bracketing a 5-dibit slot-type / 24-dibit BS-Data sync /
// 5-dibit slot-type centre. The Aloha CSBK carries the requested
// SystemID so the lock state surfaces it for the integration
// assertion.
func buildDMRTier3CSBKDibits(repeats int, colorCode uint8, systemID uint16) []uint8 {
	csbk := tier3.CSBK{Opcode: tier3.OpAloha, LB: true}
	csbk.Payload[2] = byte(systemID >> 8)
	csbk.Payload[3] = byte(systemID & 0xFF)
	csbkBytes := tier3.AssembleCSBK(csbk)
	infoBits := framing.UnpackBitsMSB(csbkBytes, 96)
	channelBits := framing.EncodeBPTC196_96(infoBits)
	payloadDibits := framing.BitsToDibits(channelBits)

	slotBits := dmr.AssembleSlotType(dmr.SlotType{ColorCode: colorCode, DataType: dmr.DTCSBK})
	slotDibits := framing.BitsToDibits(slotBits)

	burst := make([]uint8, 0, dmr.BurstDibits)
	burst = append(burst, payloadDibits[:dmr.HalfPayloadDibits]...)
	burst = append(burst, slotDibits[:dmr.SlotTypeDibits]...)
	burst = append(burst, dmr.BSData.Dibits[:]...)
	burst = append(burst, slotDibits[dmr.SlotTypeDibits:]...)
	burst = append(burst, payloadDibits[dmr.HalfPayloadDibits:]...)

	// 800-dibit warmup: gives the lower-gain MM loop (ClockGain
	// = 0.025) time to fully converge before the first burst's
	// random payload tests it.
	out := make([]uint8, 0, 800+repeats*(len(burst)+32)+100)
	for i := 0; i < 800; i++ {
		out = append(out, uint8(i&3))
	}
	for r := 0; r < repeats; r++ {
		out = append(out, burst...)
		for i := 0; i < 32; i++ {
			out = append(out, uint8(i&3))
		}
	}
	for i := 0; i < 100; i++ {
		out = append(out, uint8(i&3))
	}
	return out
}
