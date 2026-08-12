//go:build integration

package main

import (
	"context"
	"io"
	"log/slog"
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// TestDaemonDecodesTETRADMO drives the full daemon (config → protocol parse → DDC →
// newTETRADMOPipeline → cchunt supervisor) with a synthetic TETRA DMO transmission and
// asserts the DMO system LOCKS on the DSB SCH/S — the production-path proof that a DMO
// capture no longer runs the wrong TMO control-channel pipeline (the operator's
// symptom). The voice/recording half is on-air A/B gated (#1003); this test pins the
// control-path wiring end to end.
func TestDaemonDecodesTETRADMO(t *testing.T) {
	const (
		controlFreqHz = 438_900_000
		sampleRate    = 144_000.0 // ddcTargetForProtocol(ProtocolTETRADMO)
		sps           = 8
		span          = 8
		alpha         = 0.35
		dmColour      = uint32(3)
	)

	dibits := buildDMODaemonDibits(dmColour, 30)
	iq := demod.ModulatePiOver4DQPSK(dibits, sps, span, alpha, math.Pi/4)
	iq = demod.ApplyImpairments(iq, sampleRate, demod.Impairments{SNRdB: 40, Seed: 5})

	dir := t.TempDir()
	iqPath := filepath.Join(dir, "tetra-dmo.cfile")
	if err := writeIQToU8File(iqPath, iq); err != nil {
		t.Fatalf("write IQ: %v", err)
	}
	sdr.Register(&sdr.MockDriver{Files: []string{iqPath}})

	cfg := config.Default()
	cfg.SDR.SampleRate = uint32(sampleRate)
	cfg.SDR.Devices = []config.DeviceConfig{{Serial: "mock-00", Role: "control"}}
	cfg.Trunking.Systems = []config.SystemConfig{
		{
			Name:            "DMO",
			Protocol:        "tetra-dmo",
			ControlChannels: []uint32{controlFreqHz},
		},
	}
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.Metrics.Enabled = true

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "integration-cc-tetra-dmo", logger)
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
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

	deadline := time.After(6 * time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindCCLocked {
				continue
			}
			ls, ok := ev.Payload.(tetra.LockState)
			if !ok {
				t.Errorf("CCLocked payload type = %T, want tetra.LockState", ev.Payload)
				continue
			}
			if ls.FrequencyHz != controlFreqHz {
				t.Errorf("LockState.FrequencyHz = %d, want %d", ls.FrequencyHz, controlFreqHz)
			}
			// Confirm the scanner surfaces the lock too, then done.
			waitForScannerLock(t, base, "DMO", 2*time.Second)
			return
		case <-deadline:
			t.Fatalf("no cc.locked event arrived within 6s for the DMO system")
		}
	}
}

// buildDMODaemonDibits assembles a TETRA DMO dibit stream (warmup + one DSB + nDNB
// TCH/S DNBs scrambled with colour) for the π/4-DQPSK modulator, mirroring the
// EN 300 396-2 Table 15/16 layout from the exported encoders.
func buildDMODaemonDibits(colour uint32, nDNB int) []uint8 {
	b2d := tetra.TetraBitsToDibits
	filler := func(seed, n int) []uint8 {
		s := make([]uint8, n)
		for i := range s {
			s[i] = uint8((seed + i*3) & 3)
		}
		return s
	}
	seq := func(seed, n int) []byte {
		s := make([]byte, n)
		for i := range s {
			s[i] = byte((seed + i*5) % 2)
		}
		return s
	}

	out := filler(0, 900) // receiver warmup
	// DSB: 40-dibit freq-corr + 60-dibit SCH/S + 19-dibit sync train + 108-dibit SCH/H.
	out = append(out, filler(1, 40)...)
	out = append(out, b2d(tetra.EncodeBSCH(seq(1, 60)))...)
	out = append(out, tetra.SyncTrainingDibits()...)
	out = append(out, b2d(tetra.EncodeSCHHD(seq(2, 124), 0))...)
	// Repeat the DSB a few times through the stream so the supervisor's lock-wait sees
	// a sync burst regardless of when its receiver converges.
	for rep := 0; rep < nDNB; rep++ {
		frameA := seq(7+rep, 137)
		frameB := seq(9+rep, 137)
		onair := framing.ScrambleTetra(framing.UnpackBitsMSB(tetra.EncodeTCHS(frameA, frameB), 432), colour)
		out = append(out, filler(11+rep, 30)...)
		out = append(out, b2d(onair[:216])...)
		out = append(out, tetra.NormalSyncDibits()...)
		out = append(out, b2d(onair[216:])...)
		if rep%8 == 7 {
			// Periodic DSB re-sync, as a real DMO transmitter interleaves.
			out = append(out, filler(2, 20)...)
			out = append(out, filler(1, 40)...)
			out = append(out, b2d(tetra.EncodeBSCH(seq(1, 60)))...)
			out = append(out, tetra.SyncTrainingDibits()...)
			out = append(out, b2d(tetra.EncodeSCHHD(seq(2, 124), 0))...)
		}
	}
	out = append(out, filler(99, 600)...)
	return out
}
