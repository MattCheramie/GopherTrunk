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
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier2"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// TestDaemonCCDecodesDMRTier1 is the Tier I direct-mode counterpart of the
// Tier II conventional test. DMR Tier I is license-free simplex (PMR446): the
// wire format is identical to Tier II (Voice LC Header via BPTC(196,96) +
// RS(12,9)), but the burst carries a *direct-mode* sync word (DM-Voice-TS1)
// rather than the base-station sync. The production newDMRTier1Pipeline drives
// a DM-sync-restricted ConventionalChannel tagged "dmr-tier1"; this asserts the
// full daemon + supervisor + API + metrics chain recovers the lock.
func TestDaemonCCDecodesDMRTier1(t *testing.T) {
	const (
		controlFreqHz = 446_006_250 // PMR446 channel 1
		sampleRateHz  = 48_000
		sps           = 10
		span          = 8
		alpha         = 0.20
		deviationHz   = 1944.0
		colorCode     = 0x7
		groupID       = 0x123
		sourceID      = 0x456789
		burstRepeats  = 80
	)

	dibits := buildDMRTier1VoiceLCHeaderDibits(burstRepeats, colorCode, groupID, sourceID)
	iq := demod.ModulateC4FM(dibits, sps, span, alpha, sampleRateHz, deviationHz)

	dir := t.TempDir()
	iqPath := filepath.Join(dir, "dmr-tier1-cc.cfile")
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
		{Name: "DMRTier1Direct", Protocol: "dmr-tier1", ControlChannels: []uint32{controlFreqHz}},
	}
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.Metrics.Enabled = true

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "integration-cc-dmr-tier1", logger)
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

	deadline := time.After(10 * time.Second)
	var sawLock bool
WaitLoop:
	for !sawLock {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindCCLocked {
				continue
			}
			ls, ok := ev.Payload.(tier2.LockState)
			if !ok {
				t.Errorf("CCLocked payload type = %T, want tier2.LockState", ev.Payload)
				continue
			}
			if ls.ColorCode != colorCode {
				t.Errorf("LockState.ColorCode = %#x, want %#x", ls.ColorCode, colorCode)
			}
			if ls.FrequencyHz != controlFreqHz {
				t.Errorf("LockState.FrequencyHz = %d, want %d", ls.FrequencyHz, controlFreqHz)
			}
			sawLock = true
			break WaitLoop
		case <-deadline:
			t.Fatalf("no cc.locked event arrived within 10s")
		}
	}

	waitForScannerLock(t, base, "DMRTier1Direct", 2*time.Second)

	body := scrape(t, base+"/metrics")
	if !strings.Contains(body, "gophertrunk_control_channel_locked{") {
		t.Errorf("/metrics missing gophertrunk_control_channel_locked gauge family:\n%s", body)
	}
	if !strings.Contains(body, `gophertrunk_events_total{kind="cc.locked"} 1`) {
		t.Errorf("/metrics did not count one cc.locked event:\n%s", body)
	}
}

// buildDMRTier1VoiceLCHeaderDibits builds a DMR Tier I direct-mode Voice LC
// Header burst stream — identical to the Tier II builder but with the
// direct-mode DM-Voice-TS1 sync word in place of the base-station sync, so the
// DM-sync-restricted Tier I pipeline locks on it.
func buildDMRTier1VoiceLCHeaderDibits(repeats int, colorCode uint8, groupID, sourceID uint32) []uint8 {
	flc := dmr.FLC{
		FLCO:    dmr.FLCOGroupVoiceUser,
		DstAddr: groupID,
		SrcAddr: sourceID,
	}
	flcBytes := dmr.AssembleFLC(flc)
	var data [9]byte
	copy(data[:], flcBytes)
	cw := framing.EncodeRS12_9(data)
	for i := 0; i < 3; i++ {
		cw[9+i] ^= framing.RS129SeedVoiceLCHeader[i]
	}
	info := cw[:]
	bits := make([]byte, 96)
	for i := 0; i < 96; i++ {
		bits[i] = (info[i>>3] >> uint(7-(i&7))) & 1
	}
	channelBits := framing.EncodeBPTC196_96(bits)
	payloadDibits := framing.BitsToDibits(channelBits)

	slotBits := dmr.AssembleSlotType(dmr.SlotType{ColorCode: colorCode, DataType: dmr.DTVoiceLCHeader})
	slotDibits := framing.BitsToDibits(slotBits)

	burst := make([]uint8, 0, dmr.BurstDibits)
	burst = append(burst, payloadDibits[:dmr.HalfPayloadDibits]...)
	burst = append(burst, slotDibits[:dmr.SlotTypeDibits]...)
	burst = append(burst, dmr.DMVoice1.Dibits[:]...)
	burst = append(burst, slotDibits[dmr.SlotTypeDibits:]...)
	burst = append(burst, payloadDibits[dmr.HalfPayloadDibits:]...)

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
