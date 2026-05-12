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
	"github.com/MattCheramie/GopherTrunk/internal/radio/edacs"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// TestDaemonCCDecodesEDACS is the per-protocol sibling of the
// P25 P1 / NXDN / DMR / dPMR integration-cc tests, the first
// covering a non-C4FM protocol. Boots the daemon with a mock SDR
// replaying synthesized EDACS Standard / GE-Marc IQ (24-bit
// outbound sync + 40-bit BCH(40, 28, 2)-encoded CmdSystemID CCW)
// and asserts the production newEDACSPipeline + BCH-on opt-in +
// supervisor + API + metrics chain recovers the lock.
//
// EDACS runs 2-level GFSK at 9600 baud with BT = 0.3 and
// ±2.4 kHz peak deviation — the first integration test to
// exercise the new GFSKModulator + Gaussian pulse-shaping
// primitive in internal/dsp/demod (paired with the existing
// GFSK demod that the production edacs/receiver uses).
//
// The CCW carries a real CmdSystemID announcement encoded
// through framing.BCHEncodeEDACS so the daemon's `edacs_bch_mode:
// on` BCH(40, 28, 2) decode path is exercised end-to-end — a
// 1-bit error in either the codeword or its parity gets
// corrected on the receiver side.
func TestDaemonCCDecodesEDACS(t *testing.T) {
	const (
		controlFreqHz = 866_000_000
		sampleRateHz  = 96_000
		sps           = 10 // 96000 / 9600
		span          = 4
		bt            = 0.3
		deviationHz   = 2400.0
		systemID      = uint16(0x7B)
		ccwRepeats    = 30
	)

	bits := buildEDACSSystemIDStream(ccwRepeats, systemID)
	iq := demod.ModulateGFSK(bits, sps, span, bt, sampleRateHz, deviationHz)

	dir := t.TempDir()
	iqPath := filepath.Join(dir, "edacs-cc.cfile")
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
			Name:            "EDACSSite",
			Protocol:        "edacs",
			ControlChannels: []uint32{controlFreqHz},
			EDACSBCHMode:    "on",
		},
	}
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.Metrics.Enabled = true

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "integration-cc-edacs", logger)
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
			ls, ok := ev.Payload.(edacs.LockState)
			if !ok {
				t.Errorf("CCLocked payload type = %T, want edacs.LockState", ev.Payload)
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

	waitForScannerLock(t, base, "EDACSSite", 2*time.Second)

	body := scrape(t, base+"/metrics")
	if !strings.Contains(body, "gophertrunk_control_channel_locked{") {
		t.Errorf("/metrics missing gophertrunk_control_channel_locked gauge family:\n%s", body)
	}
	if !strings.Contains(body, `gophertrunk_events_total{kind="cc.locked"} 1`) {
		t.Errorf("/metrics did not count one cc.locked event:\n%s", body)
	}
}

// buildEDACSSystemIDStream assembles an EDACS bit stream for the
// GFSK modulator + receiver chain:
//
//   - 200-bit warmup alternating 0/1 so the Mueller-Müller clock
//     recovery sees a transition every symbol
//   - `repeats` × (24-bit outbound sync + 40-bit BCH(40, 28, 2)-
//     encoded CmdSystemID CCW + 16 idle bits)
//   - 100-bit trailer for clean flush
//
// Each CCW carries CmdSystemID with Address = systemID, encoded
// through framing.BCHEncodeEDACS so the production receiver's
// `edacs_bch_mode: on` BCH layer is exercised on the recovered
// bits.
func buildEDACSSystemIDStream(repeats int, systemID uint16) []byte {
	// CmdSystemID layout under BCHOn: Address carries the SystemID;
	// LCN's low bit becomes BCH parity, so leave LCN = 0 to avoid
	// surprises.
	info := uint32(edacs.CmdSystemID&0xF)<<24 |
		uint32(systemID&0xFFFF)<<4
	cw := framing.BCHEncodeEDACS(info)
	wire := make([]byte, 40)
	for i := 0; i < 40; i++ {
		if cw&(uint64(1)<<uint(39-i)) != 0 {
			wire[i] = 1
		}
	}

	frame := make([]byte, 0, 24+40)
	frame = append(frame, edacs.OutboundSyncBits()...)
	frame = append(frame, wire...)

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
