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
	"github.com/MattCheramie/GopherTrunk/internal/radio/ltr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// TestDaemonCCDecodesLTR is the per-protocol sibling of the
// earlier integration-cc tests, and the last item on the
// roadmap's "lights up live trunked reception" punch list.
// First protocol to exercise sub-audible NRZ modulation. Boots
// the daemon with a mock SDR replaying synthesized LTR IQ
// (sub-audible NRZ at 300 baud carrying 41-bit Status words with
// a known Area + Repeater), and asserts the production
// newLTRPipeline + supervisor + API + metrics chain recovers
// the lock.
func TestDaemonCCDecodesLTR(t *testing.T) {
	const (
		controlFreqHz = 935_012_500
		// LTR receiver wants ≥ 2 × 300 baud = 600 Hz. 48 kHz
		// keeps the LPF stopband + transition band comfortable.
		sampleRate          = 48_000.0
		symbolRate          = 300.0
		audioAmp            = 0.05
		area          uint8 = 7
		homeRepeater  uint8 = 4
		statusRepeats       = 80
	)

	bits := buildLTRStatusStream(statusRepeats, area, homeRepeater)
	iq := demod.ModulateSubAudibleNRZ(bits, sampleRate, symbolRate, audioAmp)

	dir := t.TempDir()
	iqPath := filepath.Join(dir, "ltr-cc.cfile")
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
			Name:            "LTRSite",
			Protocol:        "ltr",
			ControlChannels: []uint32{controlFreqHz},
			// The synthesized stream is raw NRZ with no Manchester
			// pre-encode and no Status.FCS populated, so opt out of
			// the new on-air defaults (ManchesterSoft + FCSOn) for
			// this fixture.
			LTRManchesterMode: "off",
			LTRFCSMode:        "off",
		},
	}
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.Metrics.Enabled = true

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "integration-cc-ltr", logger)
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

	// LTR's 300-baud signaling is *slow* compared to the other
	// protocols: each Status word takes ~137 ms of wall time at
	// real-time pacing, and the MM clock loop needs several
	// frames to converge. Give the deadline more headroom than
	// the 5 s the faster protocols use.
	deadline := time.After(15 * time.Second)
	var locked bool
WaitLoop:
	for !locked {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindCCLocked {
				continue
			}
			ls, ok := ev.Payload.(ltr.LockState)
			if !ok {
				t.Errorf("CCLocked payload type = %T, want ltr.LockState", ev.Payload)
				continue
			}
			if ls.Area != area {
				t.Errorf("LockState.Area = %d, want %d", ls.Area, area)
			}
			if ls.Repeater != homeRepeater {
				t.Errorf("LockState.Repeater = %d, want %d", ls.Repeater, homeRepeater)
			}
			if ls.FrequencyHz != controlFreqHz {
				t.Errorf("LockState.FrequencyHz = %d, want %d", ls.FrequencyHz, controlFreqHz)
			}
			locked = true
			break WaitLoop
		case <-deadline:
			t.Fatalf("no cc.locked event arrived within 15s")
		}
	}

	waitForScannerLock(t, base, "LTRSite", 2*time.Second)

	body := scrape(t, base+"/metrics")
	if !strings.Contains(body, "gophertrunk_control_channel_locked{") {
		t.Errorf("/metrics missing gophertrunk_control_channel_locked gauge family:\n%s", body)
	}
	if !strings.Contains(body, `gophertrunk_events_total{kind="cc.locked"} 1`) {
		t.Errorf("/metrics did not count one cc.locked event:\n%s", body)
	}
}

// buildLTRStatusStream assembles a continuous LTR bit stream for
// the sub-audible NRZ modulator + receiver chain:
//
//   - 200-bit all-zero warmup so the parser's sliding 41-bit
//     window doesn't commit to a spurious Sync=1 alignment
//     before the first real frame. (Alternating 0/1 warmup
//     produces many 1 bits at offsets that happen to parse as
//     "valid" Status words with the wrong Area / Repeater.)
//   - `repeats` × 41-bit Status word (no gap — LTR transmits
//     Status words back-to-back continuously)
//   - 100-bit trailer for clean flush
//
// Each Status word carries the requested Area + Repeater, with
// Group = false (idle frame) so the cc.locked emission fires
// without a Grant. FCS is left at zero — the test doesn't enable
// `ltr_fcs_mode: on` so the Ingest path doesn't verify it.
func buildLTRStatusStream(repeats int, area, repeater uint8) []byte {
	status := ltr.Status{
		Sync:    true,
		Area:    area,
		Channel: 3,
		Home:    repeater,
		Free:    5,
	}
	frame := ltr.StatusBits(status)

	out := make([]byte, 0, 200+repeats*len(frame)+100)
	// All-zero warmup — see func doc.
	out = append(out, make([]byte, 200)...)
	for r := 0; r < repeats; r++ {
		out = append(out, frame...)
	}
	out = append(out, make([]byte, 100)...)
	return out
}
