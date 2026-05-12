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
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/motorola"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// TestDaemonCCDecodesMotorola is the per-protocol sibling of the
// EDACS integration test (PR #152). Reuses the same GFSK
// modulator; the differences are framing layer + per-codeword
// BCH(64, 16, 11) instead of the BCH(40, 28, 2) EDACS uses.
//
// Motorola Type II runs 3600-baud 2-FSK with BT = 0.5 (the
// SmartZone standard's tighter-bandwidth profile vs EDACS' 0.3).
// Each OSW is 32 bits (16-bit Address + 16-bit Command); under
// `motorola_bch_mode: on` each half is wrapped in a 64-bit BCH
// codeword (16 info + 48 parity) for ±11 bit-error correction
// per half, so the on-air OSW occupies 128 bits = ~36 ms of
// channel time.
//
// The test asserts the production newMotorolaPipeline + BCH-on
// opt-in + supervisor + API + metrics chain recovers a
// CmdSystemID-Extended OSW and publishes cc.locked with the
// encoded SystemID.
func TestDaemonCCDecodesMotorola(t *testing.T) {
	const (
		controlFreqHz = 851_012_500
		// 27 sps × 3600 symbol rate = 97_200 Hz IQ rate. Picked so
		// the integer sps matches the receiver's float computation
		// exactly with no rounding drift.
		sampleRate  = 97_200.0
		sps         = 27
		span        = 4
		bt          = 0.5
		deviationHz = 1500.0
		systemID    = uint16(0x4567)
		oswRepeats  = 30
	)

	bits := buildMotorolaSystemIDStream(oswRepeats, systemID)
	iq := demod.ModulateGFSK(bits, sps, span, bt, sampleRate, deviationHz)

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
			MotorolaBCHMode: "on",
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

	body := scrape(t, base+"/metrics")
	if !strings.Contains(body, "gophertrunk_control_channel_locked{") {
		t.Errorf("/metrics missing gophertrunk_control_channel_locked gauge family:\n%s", body)
	}
	if !strings.Contains(body, `gophertrunk_events_total{kind="cc.locked"} 1`) {
		t.Errorf("/metrics did not count one cc.locked event:\n%s", body)
	}
}

// buildMotorolaSystemIDStream assembles a Motorola Type II bit
// stream for the GFSK modulator + receiver chain:
//
//   - 200-bit warmup alternating 0/1 so the Mueller-Müller clock
//     recovery sees a transition every symbol
//   - `repeats` × (24-bit outbound sync + 128-bit BCH-encoded
//     OSW + 16 idle bits)
//   - 100-bit trailer for clean flush
//
// Each OSW carries OpSystemIDExtended with Address = systemID,
// encoded through framing.BCHEncode64_16 (two 64-bit codewords
// for the two 16-bit halves) so the production receiver's
// `motorola_bch_mode: on` BCH layer is exercised on the
// recovered bits.
func buildMotorolaSystemIDStream(repeats int, systemID uint16) []byte {
	// Command = (opcode << 4) | LCN_or_class. For OpSystemIDExtended
	// (0x080), the low nibble is a per-system class — pick 0 since
	// the test only asserts on SystemID.
	command := uint16(motorola.OpSystemIDExtended) << 4

	// BCH-encode each 16-bit half into a 64-bit codeword.
	cw1 := framing.BCHEncode64_16(systemID)
	cw2 := framing.BCHEncode64_16(command)
	encoded := make([]byte, 128)
	for i := 0; i < 64; i++ {
		if cw1&(uint64(1)<<uint(63-i)) != 0 {
			encoded[i] = 1
		}
	}
	for i := 0; i < 64; i++ {
		if cw2&(uint64(1)<<uint(63-i)) != 0 {
			encoded[64+i] = 1
		}
	}

	frame := make([]byte, 0, 24+128)
	frame = append(frame, motorola.OutboundSyncBits()...)
	frame = append(frame, encoded...)

	out := make([]byte, 0, 200+repeats*(len(frame)+16)+100)
	for i := 0; i < 200; i++ {
		out = append(out, byte(i&1))
	}
	for r := 0; r < repeats; r++ {
		out = append(out, frame...)
		for i := 0; i < 16; i++ {
			out = append(out, byte(i&1))
		}
	}
	for i := 0; i < 100; i++ {
		out = append(out, byte(i&1))
	}
	return out
}
