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
	"github.com/MattCheramie/GopherTrunk/internal/radio/dstar"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// TestDaemonCCDecodesDStar is the per-protocol sibling of the
// EDACS / Motorola / LTR integration-cc tests, the first amateur-
// radio entry on the IQ → CC chain. Boots the daemon with a mock
// SDR replaying synthesized D-STAR DV-mode IQ (24-bit Frame Sync
// 0xEAA060 + 41-byte PCH header carrying a CQCQCQ group call) and
// asserts the production newDStarPipeline + supervisor + API +
// metrics chain recovers the lock.
//
// D-STAR runs 2-level GMSK at 4800 baud with BT = 0.5 and roughly
// ±1.2 kHz peak deviation. The integration test reuses the existing
// GFSKModulator + Gaussian pulse-shaping primitive in
// internal/dsp/demod (paired with the GFSK demod the production
// dstar/receiver uses).
//
// The stream is short transmissions in the on-air format (#1261:
// preamble, frame sync, FEC-encoded header, DV frames, end pattern),
// decoded under the default dstar_fec_mode. TestDaemonCCDecodesDStarFECOn
// sets `dstar_fec_mode: on` explicitly.
func TestDaemonCCDecodesDStar(t *testing.T) {
	const (
		controlFreqHz = 145_670_000
		// D-STAR receiver wants ≥ 2 × 4800 baud = 9600 Hz.
		// 48 kHz keeps the LPF stopband + transition band
		// comfortable.
		sampleRateHz = 48_000
		sps          = 10 // 48000 / 4800
		span         = 4
		bt           = 0.5
		deviationHz  = 1200.0
		repeats      = 30
	)

	bits := buildDStarHeaderStream(repeats, "CQCQCQ  ", "WB7XYZ  ", "KD0AAA B", "KD0AAA G")
	iq := demod.ModulateGFSK(bits, sps, span, bt, sampleRateHz, deviationHz)

	dir := t.TempDir()
	iqPath := filepath.Join(dir, "dstar-cc.cfile")
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
			Name:            "DStarRepeater",
			Protocol:        "dstar",
			ControlChannels: []uint32{controlFreqHz},
		},
	}
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.Metrics.Enabled = true

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "integration-cc-dstar", logger)
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

	// D-STAR's 4800-baud signaling is slower than EDACS but faster
	// than LTR; 5 seconds is enough headroom for repeated PCH
	// transmissions to land at least one valid frame.
	deadline := time.After(8 * time.Second)
	var locked bool
WaitLoop:
	for !locked {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindCCLocked {
				continue
			}
			ls, ok := ev.Payload.(dstar.LockState)
			if !ok {
				t.Errorf("CCLocked payload type = %T, want dstar.LockState", ev.Payload)
				continue
			}
			if ls.FrequencyHz != controlFreqHz {
				t.Errorf("LockState.FrequencyHz = %d, want %d", ls.FrequencyHz, controlFreqHz)
			}
			if strings.TrimSpace(ls.Repeater) != "KD0AAA B" {
				t.Errorf("LockState.Repeater = %q, want %q", ls.Repeater, "KD0AAA B")
			}
			locked = true
			break WaitLoop
		case <-deadline:
			t.Fatalf("no cc.locked event arrived within 8s")
		}
	}

	waitForScannerLock(t, base, "DStarRepeater", 2*time.Second)

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

// buildDStarHeaderStream assembles `repeats` short D-STAR transmissions in
// the on-air format for the GFSK modulator + receiver chain: bit-sync
// preamble, frame sync + FEC-encoded header (a CQCQCQ group call with the
// supplied callsigns and a valid CRC), 21 null-AMBE DV frames and the end
// pattern.
func buildDStarHeaderStream(repeats int, ur, my1, rpt2, rpt1 string) []byte {
	hdr := dstar.SealHeader(dstar.Header{RPT2: rpt2, RPT1: rpt1, UR: ur, MY1: my1, MY2: "SUFX"})
	out := make([]byte, 0, repeats*3000)
	for r := 0; r < repeats; r++ {
		out = append(out, dstar.BitSyncBits(256)...)
		out = append(out, dstar.EncodeHeaderBurst(hdr)...)
		for n := 0; n < dstar.DVFramesPerSyncCycle; n++ {
			out = append(out, dstar.EncodeDVFrame(dstar.NullAMBEFrame(), n, [3]byte{'f', 'f', 'f'})...)
		}
		out = append(out, dstar.EndSyncBitsSlice()...)
	}
	return append(out, dstar.BitSyncBits(100)...)
}

// TestDaemonCCDecodesDStarFECOn exercises the daemon's FECOn opt-in.
// The stream carries the FEC-encoded 660-bit on-wire payload after
// the 24-bit Frame Sync, and the connector is configured with
// `dstar_fec_mode: on` so the Process adapter runs the full chain
// (deinterleave 22×30 → PN15 descramble → depuncture → K=5 R=1/2
// Viterbi) to recover the 41-byte header before parsing.
//
// Confirms the JARL DV-mode FEC chain lights up end-to-end through
// the production newDStarPipeline + supervisor + API + metrics chain.
func TestDaemonCCDecodesDStarFECOn(t *testing.T) {
	const (
		controlFreqHz = 145_670_000
		sampleRateHz  = 48_000
		sps           = 10
		span          = 4
		bt            = 0.5
		deviationHz   = 1200.0
		repeats       = 30
	)

	bits := buildDStarHeaderStreamFECOnIntegration(repeats, "CQCQCQ  ", "WB7XYZ  ", "KD0AAA B", "KD0AAA G")
	iq := demod.ModulateGFSK(bits, sps, span, bt, sampleRateHz, deviationHz)

	dir := t.TempDir()
	iqPath := filepath.Join(dir, "dstar-fec-cc.cfile")
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
			Name:            "DStarRepeaterFEC",
			Protocol:        "dstar",
			ControlChannels: []uint32{controlFreqHz},
			DStarFECMode:    "on",
		},
	}
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.Metrics.Enabled = true

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "integration-cc-dstar-fec", logger)
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

	deadline := time.After(8 * time.Second)
	var locked bool
WaitLoop:
	for !locked {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindCCLocked {
				continue
			}
			ls, ok := ev.Payload.(dstar.LockState)
			if !ok {
				t.Errorf("CCLocked payload type = %T, want dstar.LockState", ev.Payload)
				continue
			}
			if ls.FrequencyHz != controlFreqHz {
				t.Errorf("LockState.FrequencyHz = %d, want %d", ls.FrequencyHz, controlFreqHz)
			}
			if strings.TrimSpace(ls.Repeater) != "KD0AAA B" {
				t.Errorf("LockState.Repeater = %q, want %q", ls.Repeater, "KD0AAA B")
			}
			locked = true
			break WaitLoop
		case <-deadline:
			t.Fatalf("FECOn: no cc.locked event arrived within 8s")
		}
	}
}

// buildDStarHeaderStreamFECOnIntegration is buildDStarHeaderStream.
func buildDStarHeaderStreamFECOnIntegration(repeats int, ur, my1, rpt2, rpt1 string) []byte {
	return buildDStarHeaderStream(repeats, ur, my1, rpt2, rpt1)
}
