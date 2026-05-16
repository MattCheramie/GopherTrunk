//go:build integration

package main

import (
	"context"
	"io"
	"log/slog"
	"math"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	p25phase2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// TestDaemonCCDecodesP25Phase2 is the per-protocol sibling of
// integration-cc-tetra (PR #154). Boots the daemon with a mock SDR
// replaying synthesized P25 Phase 2 H-DQPSK IQ (20-dibit outbound
// sync + 146-dibit trellis-coded MAC PDU) and asserts the
// production newP25Phase2Pipeline + p25_phase2_trellis_mode + Gardner
// clock recovery + supervisor + API + metrics chain recovers the
// lock.
//
// H-DQPSK is π/8-shifted differential QPSK at 6000 sym/s with
// α = 0.20 RRC. The π/4-DQPSK modulator from PR #154 handles it
// directly via the rotation argument (π/8 instead of π/4 for
// TETRA's true π/4-DQPSK). Per-protocol differences are:
//
//   - rotation = π/8 vs π/4
//   - symbol rate (6000 vs 18000)
//   - α (0.20 vs 0.35)
//   - framing (20-dibit outbound sync + 146 trellis-coded dibits
//     vs TETRA's 38-dibit normal training sequence + 108 channel
//     dibits)
//   - FEC chain (4-state ½-rate trellis vs TETRA's K=5 R=1/4 RCPC)
func TestDaemonCCDecodesP25Phase2(t *testing.T) {
	const (
		controlFreqHz = 851_062_500
		// 6000 × 8 = 48 kHz IQ rate; sps = 8 (exact integer).
		sampleRate = 48_000.0
		sps        = 8
		span       = 8
		alpha      = 0.20
		pduRepeats = 80
	)

	dibits := buildP25Phase2MACPTTStream(pduRepeats)
	iq := demod.ModulatePiOver4DQPSK(dibits, sps, span, alpha, math.Pi/8)

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
			Name:                 "P25P2Site",
			Protocol:             "p25-phase2",
			ControlChannels:      []uint32{controlFreqHz},
			P25Phase2TrellisMode: "on",
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

	body := scrape(t, base+"/metrics")
	if !strings.Contains(body, "gophertrunk_control_channel_locked{") {
		t.Errorf("/metrics missing gophertrunk_control_channel_locked gauge family:\n%s", body)
	}
	if !strings.Contains(body, `gophertrunk_events_total{kind="cc.locked"} 1`) {
		t.Errorf("/metrics did not count one cc.locked event:\n%s", body)
	}
}

// buildP25Phase2MACPTTStream assembles a P25 Phase 2 dibit stream
// for the H-DQPSK modulator + receiver chain:
//
//   - 400-dibit warmup cycling 0..3 so the matched filter +
//     differential decoder converge on the constellation centre
//   - `repeats` × (20-dibit outbound sync + 146-dibit
//     trellis-coded MAC PDU + 30 idle dibits)
//   - 100-dibit trailer for clean flush
//
// Each PDU is an OpMACPTT (PTT-on, the canonical "lock me"
// non-idle MAC PDU). The 72 info dibits run through the
// TIA-102 Annex A 4-state ½-rate trellis encoder via
// framing.EncodeP25Trellis.
func buildP25Phase2MACPTTStream(repeats int) []uint8 {
	pdu := p25phase2.MACPDU{Opcode: p25phase2.OpMACPTT, Payload: make([]byte, 17)}
	pduBits := framing.UnpackBitsMSB(p25phase2.AssembleMACPDU(pdu), 144)
	infoDibits := framing.BitsToDibits(pduBits)
	channelDibits := framing.EncodeP25Trellis(infoDibits)

	frame := make([]uint8, 0, 20+len(channelDibits))
	frame = append(frame, p25phase2.OutboundSyncDibits()...)
	frame = append(frame, channelDibits...)

	out := make([]uint8, 0, 400+repeats*(len(frame)+30)+100)
	for i := 0; i < 400; i++ {
		out = append(out, uint8(i&3))
	}
	for r := 0; r < repeats; r++ {
		out = append(out, frame...)
		for i := 0; i < 30; i++ {
			out = append(out, uint8(i&3))
		}
	}
	for i := 0; i < 100; i++ {
		out = append(out, uint8(i&3))
	}
	return out
}
