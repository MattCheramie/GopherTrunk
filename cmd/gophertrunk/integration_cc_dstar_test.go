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
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
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
// This test exercises the FECOff (default) path — the integration
// stream carries pre-FEC bits straight to the Process adapter.
// TestDaemonCCDecodesDStarFECOn exercises the full JARL DV-mode
// chain (conv + scramble + interleave) the dstar package now ships.
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

	body := scrape(t, base+"/metrics")
	if !strings.Contains(body, "gophertrunk_control_channel_locked{") {
		t.Errorf("/metrics missing gophertrunk_control_channel_locked gauge family:\n%s", body)
	}
	if !strings.Contains(body, `gophertrunk_events_total{kind="cc.locked"} 1`) {
		t.Errorf("/metrics did not count one cc.locked event:\n%s", body)
	}
}

// buildDStarHeaderStream assembles a D-STAR bit stream for the GFSK
// modulator + receiver chain:
//
//   - 256-bit warmup of constant ones so the Mueller-Müller clock
//     recovery sees a stable baseline before the sync burst arrives
//   - `repeats` × (24-bit Frame Sync 0xEAA060 + 328-bit PCH header)
//   - 100-bit trailer for clean flush
//
// Each PCH header carries a CQCQCQ group call with the supplied
// MY1 / RPT2 / RPT1 callsigns and a valid CRC-CCITT trailer. Used
// to exercise the FECOff path through the daemon's pipeline.
func buildDStarHeaderStream(repeats int, ur, my1, rpt2, rpt1 string) []byte {
	hdr := dstar.Header{
		Flag1: 0,
		Flag2: 0,
		Flag3: 0,
		RPT2:  rpt2,
		RPT1:  rpt1,
		UR:    ur,
		MY1:   my1,
		MY2:   "SUFX",
	}
	asm := dstar.AssembleHeader(hdr)
	hdr.CRC = dstar.ComputeCRC(asm[:39])
	asm = dstar.AssembleHeader(hdr)

	headerBits := make([]byte, 0, 328)
	for _, b := range asm {
		for i := 0; i < 8; i++ {
			headerBits = append(headerBits, (b>>uint(7-i))&1)
		}
	}

	frame := make([]byte, 0, 32+328)
	frame = append(frame, dstar.FrameSyncBitsSlice()...)
	frame = append(frame, headerBits...)

	out := make([]byte, 0, 256+repeats*(len(frame)+32)+100)
	for i := 0; i < 256; i++ {
		out = append(out, 1)
	}
	for r := 0; r < repeats; r++ {
		out = append(out, frame...)
		for i := 0; i < 32; i++ {
			out = append(out, 1)
		}
	}
	for i := 0; i < 100; i++ {
		out = append(out, 1)
	}
	return out
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

// buildDStarHeaderStreamFECOnIntegration is the integration-test
// counterpart of buildDStarHeaderStream that emits the FEC-encoded
// 660-bit on-wire payload after each 24-bit Frame Sync. Matches the
// runtime "warmup + Frame Sync + payload + idle" framing the
// production GFSK modulator + receiver chain wants to see.
func buildDStarHeaderStreamFECOnIntegration(repeats int, ur, my1, rpt2, rpt1 string) []byte {
	hdr := dstar.Header{
		Flag1: 0,
		Flag2: 0,
		Flag3: 0,
		RPT2:  rpt2,
		RPT1:  rpt1,
		UR:    ur,
		MY1:   my1,
		MY2:   "SUFX",
	}
	asm := dstar.AssembleHeader(hdr)
	hdr.CRC = dstar.ComputeCRC(asm[:39])
	asm = dstar.AssembleHeader(hdr)
	channelBits := framing.EncodeDStarHeaderFEC(asm)

	frame := make([]byte, 0, 24+len(channelBits))
	frame = append(frame, dstar.FrameSyncBitsSlice()...)
	frame = append(frame, channelBits...)

	out := make([]byte, 0, 256+repeats*(len(frame)+32)+100)
	for i := 0; i < 256; i++ {
		out = append(out, 1)
	}
	for r := 0; r < repeats; r++ {
		out = append(out, frame...)
		for i := 0; i < 32; i++ {
			out = append(out, 1)
		}
	}
	for i := 0; i < 100; i++ {
		out = append(out, 1)
	}
	return out
}
