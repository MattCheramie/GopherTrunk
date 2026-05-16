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
	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// realairCaptureMetadata is the in-test schema for the JSON sidecar
// dropped alongside each *.cfile in samples/<proto>/. Mirrors the
// schema documented in samples/<proto>/README.md "Metadata schema"
// sections, plus two top-level fields the test needs to drive the
// mock SDR (`sample_rate_hz`, `center_freq_hz`) that the operator
// MUST supply since GNU Radio cfiles don't embed them.
type realairCaptureMetadata struct {
	Source         string `json:"source"`
	ToolCrossCheck string `json:"tool_cross_check"`
	// SampleRateHz is the IQ sample rate the capture was recorded
	// at — the mock SDR's SampleRate must match for the receiver's
	// matched-filter cascade + clock recovery to lock. The samples
	// READMEs document acceptable ranges per protocol (NXDN: ≥
	// 48 kHz, TETRA: ≥ 36 kHz, etc.).
	SampleRateHz uint32 `json:"sample_rate_hz"`
	// CenterFreqHz is the IQ centre frequency the operator tuned
	// when recording — the test feeds this into both the SDR
	// device config and the per-system ControlChannels list so the
	// ccdecoder pipeline knows what to tune to.
	CenterFreqHz uint32 `json:"center_freq_hz"`
	// Expected carries the ground-truth values the decoded LockState
	// must match. Protocol-specific shape; see per-protocol parsers
	// below.
	Expected json.RawMessage `json:"expected"`
}

// nxdnExpected is the inner shape for samples/nxdn/*.metadata.json
// "expected" objects. system_id / site_id arrive as hex strings
// ("0x1234") because that's how an operator would copy them from a
// MMDVMHost / DSDcc log. RAN is parsed but not yet asserted —
// nxdn.LockState doesn't carry it; when the field lands the
// assertion is one line.
type nxdnExpected struct {
	SystemID string `json:"system_id"`
	SiteID   string `json:"site_id"`
	RAN      uint8  `json:"ran"`
}

// findRealairCaptures returns the (cfile, metadata.json) path pair
// for the supplied protocol's samples directory, or two empty
// strings if no pair is present. A pair is matched when the cfile's
// stem (basename minus extension) has a sibling .metadata.json.
//
// The contract is intentionally narrow: exactly one capture pair is
// the supported case. A directory with multiple .cfile files
// surfaces an error so the contributor knows to disambiguate
// (typically by deleting older captures rather than running them
// concurrently).
func findRealairCapture(t *testing.T, proto string) (cfilePath, metaPath string) {
	t.Helper()
	dir := filepath.Join("..", "..", "samples", proto)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ""
		}
		t.Fatalf("read samples/%s: %v", proto, err)
	}
	var cfiles []string
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".cfile") {
			continue
		}
		cfiles = append(cfiles, e.Name())
	}
	if len(cfiles) == 0 {
		return "", ""
	}
	if len(cfiles) > 1 {
		t.Fatalf("samples/%s: %d .cfile candidates (%v) — exactly one is supported; delete or rename extras",
			proto, len(cfiles), cfiles)
	}
	stem := strings.TrimSuffix(cfiles[0], ".cfile")
	metaName := stem + ".metadata.json"
	metaFullPath := filepath.Join(dir, metaName)
	if _, err := os.Stat(metaFullPath); err != nil {
		t.Fatalf("samples/%s/%s: no sibling %s (capture without metadata is unusable for validation)",
			proto, cfiles[0], metaName)
	}
	return filepath.Join(dir, cfiles[0]), metaFullPath
}

func loadRealairMetadata(t *testing.T, path string) realairCaptureMetadata {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var m realairCaptureMetadata
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	if m.SampleRateHz == 0 {
		t.Fatalf("%s: sample_rate_hz is required (the GNU Radio cfile doesn't embed it)", path)
	}
	if m.CenterFreqHz == 0 {
		t.Fatalf("%s: center_freq_hz is required (used to tune the mock SDR)", path)
	}
	return m
}

// TestDaemonCCDecodesNXDNRealAir is the skip-gated companion to
// TestDaemonCCDecodesNXDN. The synthesized sibling proves the IQ →
// CC pipeline round-trips against in-tree fixtures; this one proves
// it against an actual on-air capture once a contributor drops a
// .cfile + .metadata.json pair into samples/nxdn/ per the schema
// documented in samples/nxdn/README.md.
//
// Acceptance criteria (also in samples/nxdn/README.md):
//
//  1. Lock latency. CCLocked event arrives within 3 s wall time of
//     daemon start. NXDN locks faster than TETRA because there's
//     no Gardner step in the receiver chain.
//  2. System metadata match. Decoded SystemID + SiteID + RAN match
//     the metadata.json's "expected" values byte-for-byte.
//
// CRC-verified CAC burst rate (≥ 80% target from the README) is
// not asserted here yet — the in-tree control-channel state machine
// doesn't currently surface a CRC-pass histogram on the bus. The
// test will start asserting that rate when the corresponding metric
// lands (per the "Acceptance criteria" §1 note in the README, the
// histogram and capture land together).
func TestDaemonCCDecodesNXDNRealAir(t *testing.T) {
	cfilePath, metaPath := findRealairCapture(t, "nxdn")
	if cfilePath == "" {
		t.Skipf("samples/nxdn/: no *.cfile present — drop a capture + metadata pair to run this test (see samples/nxdn/README.md)")
	}
	meta := loadRealairMetadata(t, metaPath)

	var exp nxdnExpected
	if err := json.Unmarshal(meta.Expected, &exp); err != nil {
		t.Fatalf("parse expected payload: %v", err)
	}
	wantSystemID, err := parseHex16(exp.SystemID)
	if err != nil {
		t.Fatalf("metadata expected.system_id=%q: %v", exp.SystemID, err)
	}
	wantSiteID, err := parseHex16(exp.SiteID)
	if err != nil {
		t.Fatalf("metadata expected.site_id=%q: %v", exp.SiteID, err)
	}

	sdr.Register(&sdr.MockFloat32Driver{Files: []string{cfilePath}})

	cfg := config.Default()
	cfg.SDR.SampleRate = meta.SampleRateHz
	cfg.SDR.Devices = []config.DeviceConfig{
		{Serial: "mockf32-00", Role: "control"},
	}
	cfg.Trunking.Systems = []config.SystemConfig{
		{
			Name:            "NXDNRealAir",
			Protocol:        "nxdn",
			ControlChannels: []uint32{meta.CenterFreqHz},
			NXDNViterbiMode: "spec",
		},
	}
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.Metrics.Enabled = true

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "integration-cc-nxdn-realair", logger)
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

	// 3 s lock-latency budget per samples/nxdn/README.md §3.
	deadline := time.After(3 * time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindCCLocked {
				continue
			}
			ls, ok := ev.Payload.(nxdn.LockState)
			if !ok {
				t.Errorf("CCLocked payload type = %T, want nxdn.LockState", ev.Payload)
				continue
			}
			if ls.SystemID != wantSystemID {
				t.Errorf("LockState.SystemID = %#x, want %#x (from metadata expected.system_id)",
					ls.SystemID, wantSystemID)
			}
			if ls.SiteID != wantSiteID {
				t.Errorf("LockState.SiteID = %#x, want %#x (from metadata expected.site_id)",
					ls.SiteID, wantSiteID)
			}
			if ls.FrequencyHz != meta.CenterFreqHz {
				t.Errorf("LockState.FrequencyHz = %d, want %d (from metadata center_freq_hz)",
					ls.FrequencyHz, meta.CenterFreqHz)
			}
			return
		case <-deadline:
			t.Fatalf("no cc.locked event arrived within 3s (capture=%s)", filepath.Base(cfilePath))
		}
	}
}

// parseHex16 accepts "0x1234" / "1234" / "0X1234" style strings and
// returns the parsed uint16. Used for metadata fields the operator
// copies from MMDVMHost / DSDcc logs, where hex is the dominant
// convention.
func parseHex16(s string) (uint16, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	var v uint64
	for _, c := range s {
		var nib uint64
		switch {
		case c >= '0' && c <= '9':
			nib = uint64(c - '0')
		case c >= 'a' && c <= 'f':
			nib = uint64(c-'a') + 10
		case c >= 'A' && c <= 'F':
			nib = uint64(c-'A') + 10
		default:
			return 0, errors.New("non-hex digit")
		}
		v = v<<4 | nib
		if v > 0xFFFF {
			return 0, errors.New("value exceeds uint16")
		}
	}
	if s == "" {
		return 0, errors.New("empty")
	}
	return uint16(v), nil
}
