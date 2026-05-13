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

// TestDaemonCCDecodesDMRTier2 is the Tier II conventional counterpart
// of TestDaemonCCDecodesDMRTier3. Boots the daemon with a mock SDR
// replaying a fully-synthesized DMR Tier II burst (132 dibits =
// 49 first-half payload + 5 slot-type + 24 sync + 5 slot-type + 49
// second-half payload, with the payload halves carrying a Voice LC
// Header encoded through BPTC(196, 96) + RS(12, 9) parity), and
// asserts the production newDMRTier2Pipeline + supervisor + API +
// metrics chain recovers the lock.
//
// Tier II is conventional — the same C4FM 4800-baud modulation as
// Tier III, but the call-setup mechanism is a Voice LC Header burst
// at the start of every transmission rather than a CSBK on a
// dedicated control channel.
func TestDaemonCCDecodesDMRTier2(t *testing.T) {
	// Fixture was previously t.Skip'd because the synthesized Voice
	// LC Header payload's symbol distribution slipped the receiver's
	// Mueller-Müller clock loop at ClockGain = 0.025 (the value
	// shared with Tier III). The diagnostic in
	// dmr_tier2_diagnostic_test.go localised the divergent statistic
	// to the BPTC(196, 96)-encoded payload's class-3 dibit fraction
	// (21.4% Tier II vs 5.1% Tier III) and the matching mean
	// transition magnitude (1.27 vs 0.90); the RS(12, 9) seed
	// 0x96 0x96 0x96 and the BPTC parity rows distribute
	// high-Hamming-weight bits throughout the channel-bit output,
	// and the resulting rapid-transition stream needs a more
	// conservative loop gain. Lowering Tier II's pipeline ClockGain
	// to 0.015 (see newDMRTier2Pipeline in
	// internal/scanner/ccdecoder/pipelines.go) closes the gap; the
	// receiver locks within ~100 ms of the first burst.

	const (
		controlFreqHz = 460_500_000
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

	dibits := buildDMRTier2VoiceLCHeaderDibits(burstRepeats, colorCode, groupID, sourceID)
	iq := demod.ModulateC4FM(dibits, sps, span, alpha, sampleRateHz, deviationHz)

	dir := t.TempDir()
	iqPath := filepath.Join(dir, "dmr-tier2-cc.cfile")
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
		{Name: "DMRTier2Repeater", Protocol: "dmr-tier2", ControlChannels: []uint32{controlFreqHz}},
	}
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.Metrics.Enabled = true

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "integration-cc-dmr-tier2", logger)
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
			t.Fatalf("no cc.locked event arrived within 5s")
		}
	}

	waitForScannerLock(t, base, "DMRTier2Repeater", 2*time.Second)

	body := scrape(t, base+"/metrics")
	if !strings.Contains(body, "gophertrunk_control_channel_locked{") {
		t.Errorf("/metrics missing gophertrunk_control_channel_locked gauge family:\n%s", body)
	}
	if !strings.Contains(body, `gophertrunk_events_total{kind="cc.locked"} 1`) {
		t.Errorf("/metrics did not count one cc.locked event:\n%s", body)
	}
}

// buildDMRTier2VoiceLCHeaderDibits builds a DMR Tier II Voice LC
// Header burst stream. Each 132-dibit burst carries an FLC
// (FLCOGroupVoiceUser with the supplied groupID + sourceID) encoded
// through RS(12, 9) parity + BPTC(196, 96), with the slot-type field
// set to DTVoiceLCHeader so the Tier II Process adapter routes it
// into the Voice LC Header handler.
func buildDMRTier2VoiceLCHeaderDibits(repeats int, colorCode uint8, groupID, sourceID uint32) []uint8 {
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
	burst = append(burst, dmr.BSData.Dibits[:]...)
	burst = append(burst, slotDibits[dmr.SlotTypeDibits:]...)
	burst = append(burst, payloadDibits[dmr.HalfPayloadDibits:]...)

	// 800-dibit warmup + 32-dibit inter-burst gap, matching Tier III.
	// Tier II's harder symbol distribution (mean transition magnitude
	// 1.27 vs Tier III's 0.90 per TestDMRTier2VsTier3SymbolDensity
	// in dmr_tier2_diagnostic_test.go) is handled by the lower
	// ClockGain (0.015) on the Tier II pipeline, not by fixture
	// padding — see internal/scanner/ccdecoder/pipelines.go's
	// newDMRTier2Pipeline.
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
