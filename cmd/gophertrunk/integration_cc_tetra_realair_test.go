//go:build integration

package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/baseband"
)

// tetraExpected is the inner "expected" shape for
// samples/tetra/*.metadata.json (see samples/tetra/README.md). MCC / MNC
// are the cell identity carried in the BNCH SYSINFO; recovering them
// end-to-end transitively proves the BSCH colour-code recovery worked
// (#648/#662), because the SYSINFO only descrambles under the learned
// colour code. colour_code is parsed for documentation/cross-check but
// is not carried on the cc.locked event, so it is not asserted directly.
type tetraExpected struct {
	MCC          uint16 `json:"mcc"`
	MNC          uint16 `json:"mnc"`
	ColourCode   uint32 `json:"colour_code"`
	LocationArea uint16 `json:"location_area"`
}

// findTETRARealairCapture returns the (capture, metadata.json) path pair
// for samples/tetra/, or ("","") when no validating pair is present.
//
// It is deliberately TETRA-local rather than the shared
// findRealairCapture: (1) the in-tree TETRA capture is a baseband WAV
// (replayed natively through baseband.FileDriver), whereas the shared
// helper matches only *.cfile; and (2) this finder *skips* when no
// metadata sidecar is present instead of failing, because the committed
// samples/tetra/"TETRA IQ.wav" is a reference carrier that does not yet
// decode CRC-clean through the §8.3.1 chain (see samples/tetra/README.md)
// — so it ships without a sidecar and this test stays dormant until a
// validating capture + sidecar lands. A *.wav with a sibling sidecar is
// the signal a contributor has supplied one.
func findTETRARealairCapture(t *testing.T) (capturePath, metaPath string) {
	t.Helper()
	dir := filepath.Join("..", "..", "samples", "tetra")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ""
		}
		t.Fatalf("read samples/tetra: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".wav") {
			continue
		}
		stem := strings.TrimSuffix(name, ".wav")
		meta := filepath.Join(dir, stem+".metadata.json")
		if _, err := os.Stat(meta); err == nil {
			return filepath.Join(dir, name), meta
		}
	}
	return "", ""
}

// TestDaemonCCDecodesTETRARealAir is the skip-gated companion to
// TestDaemonCCDecodesTETRA. The synthesized sibling proves the IQ → CC
// pipeline round-trips against in-tree fixtures; this one proves the same
// chain recovers a lock from an actual on-air TMO downlink capture,
// replayed natively through baseband.FileDriver (the WAV-aware
// "baseband-replay" sdr.Driver), once a contributor drops a baseband WAV
// + .metadata.json pair into samples/tetra/ per samples/tetra/README.md.
//
// Acceptance criteria (samples/tetra/README.md §"Acceptance criteria"):
//
//  1. Lock latency — cc.locked within 5 s wall time of the first burst.
//  2. Colour-code recovery — the decoded MCC/MNC match metadata
//     expected.{mcc,mnc}. These come from the BNCH SYSINFO, which only
//     descrambles under the colour code learned from the BSCH, so a
//     match end-to-end validates the #648/#662 recovery on real RF.
func TestDaemonCCDecodesTETRARealAir(t *testing.T) {
	capturePath, metaPath := findTETRARealairCapture(t)
	if capturePath == "" {
		t.Skip("samples/tetra/: no *.wav + *.metadata.json pair — drop a validating capture to run this test (see samples/tetra/README.md)")
	}

	raw, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("read %s: %v", metaPath, err)
	}
	var meta realairCaptureMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatalf("parse %s: %v", metaPath, err)
	}
	if meta.SampleRateHz == 0 {
		t.Fatalf("%s: sample_rate_hz is required", metaPath)
	}
	if meta.CenterFreqHz == 0 {
		t.Fatalf("%s: center_freq_hz is required", metaPath)
	}
	var exp tetraExpected
	if err := json.Unmarshal(meta.Expected, &exp); err != nil {
		t.Fatalf("parse expected payload: %v", err)
	}

	// Replay the baseband WAV through the WAV-native driver, looping so the
	// offline tuner behaves like a continuous source.
	sdr.Register(baseband.NewFileDriver([]baseband.ReplaySpec{
		{Path: capturePath, Serial: "replay-00", Loop: true},
	}))

	cfg := config.Default()
	cfg.SDR.SampleRate = meta.SampleRateHz
	cfg.SDR.Devices = []config.DeviceConfig{
		{Serial: "replay-00", Role: "control"},
	}
	cfg.Trunking.Systems = []config.SystemConfig{
		{
			Name:            "TETRARealAir",
			Protocol:        "tetra",
			ControlChannels: []uint32{meta.CenterFreqHz},
			// Colour code left zero on purpose: the decoder must auto-learn
			// it from the BSCH (the #648/#662 path under test).
		},
	}
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.Metrics.Enabled = true
	// Exercise the opt-in FEC correction-depth histogram against real RF.
	cfg.Metrics.DetailedFEC = true

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "integration-cc-tetra-realair", logger)
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
	waitReachable(t, base+"/api/v1/health", 5*time.Second)

	// 5 s lock-latency budget per samples/tetra/README.md.
	deadline := time.After(5 * time.Second)
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
			if ls.FrequencyHz != meta.CenterFreqHz {
				t.Errorf("LockState.FrequencyHz = %d, want %d", ls.FrequencyHz, meta.CenterFreqHz)
			}
			if exp.MCC != 0 && ls.MCC != exp.MCC {
				t.Errorf("LockState.MCC = %#x, want %#x (colour-code recovery suspect)", ls.MCC, exp.MCC)
			}
			if exp.MNC != 0 && ls.MNC != exp.MNC {
				t.Errorf("LockState.MNC = %#x, want %#x (colour-code recovery suspect)", ls.MNC, exp.MNC)
			}

			// The opt-in FEC histogram must have recorded at least one
			// recovered-burst correction-depth sample.
			body := scrape(t, base+"/metrics")
			if !strings.Contains(body, "gophertrunk_tetra_viterbi_corrections_bucket") {
				t.Errorf("/metrics missing gophertrunk_tetra_viterbi_corrections histogram after lock:\n%s", body)
			}
			return
		case <-deadline:
			t.Fatalf("no cc.locked event within 5s (capture=%s)", filepath.Base(capturePath))
		}
	}
}
