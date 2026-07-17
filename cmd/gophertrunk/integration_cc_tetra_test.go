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
	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// TestDaemonCCDecodesTETRA is the per-protocol sibling of the
// earlier integration-cc tests, the first to exercise π/4-DQPSK
// modulation. Boots the daemon with a mock SDR replaying
// synthesized TETRA TMO IQ (38-dibit normal training-sequence
// sync + 108-dibit SCH/HD-encoded MLE SYSINFO PDU) and asserts
// the production newTETRAPipeline + tetra_channel_coding +
// tetra_colour_code config + supervisor + API + metrics chain
// recovers the lock.
//
// TETRA is 18000 sym/s π/4-DQPSK with α = 0.35 RRC. Each burst's
// channel coding chain (per ETSI EN 300 392-2 §8.3.1) is:
// 124 type-1 info bits → +CRC + tail → K=5 R=1/4 RCPC encode +
// puncture (rate 2/3) → (216, 101) block interleave → 30-bit
// extended-colour scrambler → 216 type-5 bits = 108 dibits on
// the wire. This test exercises the chain end-to-end.
func TestDaemonCCDecodesTETRA(t *testing.T) {
	const (
		// TETRA symbol rate is 18000. Pick 72000 = 4 × 18000 so
		// the receiver's float sps computation rounds to an exact
		// integer (4) with no drift.
		sampleRate = 72_000.0
		sps        = 4
		span       = 8
		alpha      = 0.35
	)
	dibits := buildTETRASCHHDStream(100, tetraCCColourCode, tetraCCLocationArea)
	iq := demod.ModulatePiOver4DQPSK(dibits, sps, span, alpha, math.Pi/4)
	assertDaemonTETRALocks(t, iq, sampleRate)
}

// TestDaemonCCDecodesTETRAAt48kHz is the sub-target sibling of
// TestDaemonCCDecodesTETRA. It feeds the daemon a 48 kHz TETRA capture —
// below the 144 kHz channel target, and at a fractional 2.667 sps that
// the receiver cannot decode directly — and asserts the production
// pipeline still locks because the Downconverter now interpolates the
// stream up to the 144 kHz / 8-sps design point.
//
// 48000/18000 is not an integer, so the fixture cannot be modulated at
// 48 kHz directly; it is modulated at 144 kHz (8 sps) and downsampled
// 3:1 to a genuine 48 kHz capture, exactly the rate an SDR++ baseband
// recording or the samples/tetra reference WAV lands at.
//
// Before the DDC up-sample fix the daemon passed the 48 kHz stream
// through untouched, the receiver rounded 2.667 → 3 sps and decoded at
// 16000 baud, and no lock ever fired. This test fails without the fix.
func TestDaemonCCDecodesTETRAAt48kHz(t *testing.T) {
	const (
		span  = 8
		alpha = 0.35
	)
	dibits := buildTETRASCHHDStream(160, tetraCCColourCode, tetraCCLocationArea)
	iq144 := demod.ModulatePiOver4DQPSK(dibits, 8, span, alpha, math.Pi/4)
	// Downsample 144 kHz → 48 kHz with a sharp anti-alias prototype so
	// the capture is clean; the daemon's DDC does the inverse 3:1
	// interpolation back to 144 kHz.
	iq48 := dsp.NewResampler(1, 3, 64, 8.6).Process(nil, iq144)
	assertDaemonTETRALocks(t, iq48, 48_000)
}

const (
	tetraCCControlFreqHz = 412_062_500
	tetraCCColourCode    = uint32(0x12345)
	// LocationArea is what nxdn / dpmr / edacs / motorola got plumbed
	// into LockedNAC; for TETRA we use LocationArea the same way.
	// 0x42 = 66 is a small recognisable value.
	tetraCCLocationArea uint16 = 0x42
)

// assertDaemonTETRALocks boots the daemon with a mock SDR replaying the
// given TETRA IQ at sampleRate and asserts the production
// newTETRAPipeline + config + supervisor + API + metrics chain recovers
// the lock carrying tetraCCLocationArea.
func assertDaemonTETRALocks(t *testing.T, iq []complex64, sampleRate float64) {
	t.Helper()
	const (
		controlFreqHz = tetraCCControlFreqHz
		colourCode    = tetraCCColourCode
		locationArea  = tetraCCLocationArea
	)

	dir := t.TempDir()
	iqPath := filepath.Join(dir, "tetra-cc.cfile")
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
			Name:            "TETRASite",
			Protocol:        "tetra",
			ControlChannels: []uint32{controlFreqHz},
			TETRAColourCode: colourCode,
			TETRAChannel:    "sch/hd",
		},
	}
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.Metrics.Enabled = true

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "integration-cc-tetra", logger)
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
			ls, ok := ev.Payload.(tetra.LockState)
			if !ok {
				t.Errorf("CCLocked payload type = %T, want tetra.LockState", ev.Payload)
				continue
			}
			if ls.LocationArea != locationArea {
				t.Errorf("LockState.LocationArea = %#x, want %#x", ls.LocationArea, locationArea)
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

	waitForScannerLock(t, base, "TETRASite", 2*time.Second)

	body := scrape(t, base+"/metrics")
	if !strings.Contains(body, "gophertrunk_control_channel_locked{") {
		t.Errorf("/metrics missing gophertrunk_control_channel_locked gauge family:\n%s", body)
	}
	if !strings.Contains(body, `gophertrunk_events_total{kind="cc.locked"} 1`) {
		t.Errorf("/metrics did not count one cc.locked event:\n%s", body)
	}
}

// buildTETRASCHHDStream assembles a TETRA dibit stream for the
// π/4-DQPSK modulator + receiver chain:
//
//   - 400-dibit warmup cycling 0..3 so the matched filter +
//     differential decoder converge on the constellation centre
//   - `repeats` × (38-dibit normal training-sequence sync +
//     108-dibit SCH/HD-encoded MLE SYSINFO PDU + 50 idle dibits)
//   - 100-dibit trailer for clean flush
//
// Each burst's payload is an MLE SYSINFO PDU carrying the
// requested LocationArea (with MCC + MNC zero — the minimum
// payload that drives the cc.locked path in
// tetra.ControlChannel). The 124 info bits run through the full
// §8.3.1 channel coding chain via tetra.EncodeSCHHD.
func buildTETRASCHHDStream(repeats int, colourCode uint32, locationArea uint16) []uint8 {
	// MLE SYSINFO payload: 10 bits MCC + 14 bits MNC + 14 bits LA.
	// Set MCC = MNC = 0; LA carries the requested value. Pack
	// bytes per AsSystemBroadcast's parse layout:
	//   payload[3] = (LA >> 6) & 0xFF
	//   payload[4] = (LA & 0x3F) << 2
	payload := []byte{0x00, 0x00, 0x00, 0, 0}
	payload[3] = byte((locationArea >> 6) & 0xFF)
	payload[4] = byte((locationArea & 0x3F) << 2)

	pdu := tetra.PDU{
		Disc:    tetra.DiscMLE,
		Type:    uint8(tetra.MLESystemInfo),
		Payload: payload,
	}
	info := pduToType1BitsTETRA(pdu, 124)
	type5 := tetra.EncodeSCHHD(info, colourCode)
	burstDibits := tetra.TetraBitsToDibits(type5)

	frame := make([]uint8, 0, 38+len(burstDibits))
	frame = append(frame, tetra.NormalSyncDibits()...)
	frame = append(frame, burstDibits...)

	out := make([]uint8, 0, 400+repeats*(len(frame)+50)+100)
	for i := 0; i < 400; i++ {
		out = append(out, uint8(i&3))
	}
	for r := 0; r < repeats; r++ {
		out = append(out, frame...)
		for i := 0; i < 50; i++ {
			out = append(out, uint8(i&3))
		}
	}
	for i := 0; i < 100; i++ {
		out = append(out, uint8(i&3))
	}
	return out
}

// pduToType1BitsTETRA is a tiny shim mirroring the in-package
// tetra test helper of the same name. Pads a PDU's header + bytes
// into exactly k1 bits, MSB-first per byte, zero-padded.
func pduToType1BitsTETRA(pdu tetra.PDU, k1 int) []byte {
	bytes := tetra.AssemblePDU(pdu)
	if len(bytes)*8 > k1 {
		return nil
	}
	out := make([]byte, k1)
	for i, b := range bytes {
		for j := 0; j < 8; j++ {
			out[i*8+j] = (b >> uint(7-j)) & 1
		}
	}
	return out
}
