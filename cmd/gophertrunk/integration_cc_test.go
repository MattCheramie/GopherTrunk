//go:build integration

package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestDaemonCCDecodesP25Phase1 is the end-to-end "lights up live
// trunked reception" check from the roadmap. It boots the wired
// daemon with a mock SDR and a stubbed-in P25 Phase 1 pipeline
// factory, then injects a known-good dibit stream into the real
// phase1.ControlChannel — exercising the full chain *above* the
// IQ→dibit demod:
//
//   - daemon construction (pool, supervisor, ccdecoder)
//   - cchunt supervisor publishing KindHuntProgress
//   - ccdecoder factory dispatch + pipeline construction
//   - pipeline.Process invoked on every IQ chunk
//   - cc.Process driving the state machine from synthesized
//     dibits (FSW + NID + TSBK frame fixtures from the in-
//     package phase1 test helpers)
//   - state machine emitting cc.locked on the bus
//   - supervisor consuming cc.locked → state=locked transition
//   - /api/v1/scanner reflecting the lock
//   - gophertrunk_cc_locked_total metric incrementing
//
// The one chain step this test skips is IQ→dibit C4FM
// demodulation — that's covered by the phase1/receiver unit
// tests. A future PR could land a proper C4FM modulator (RRC
// pulse shaping + continuous-phase integration) and remove the
// factory-stubbing, but the integration coverage above the demod
// is what this milestone actually proves.
//
// The plan documented this as the close-out for Workstream A
// ("lights up live trunked reception").
func TestDaemonCCDecodesP25Phase1(t *testing.T) {
	const (
		nac           = 0x293
		controlFreqHz = 851_000_000
	)

	// Build a P25 Phase 1 dibit stream the real
	// phase1.ControlChannel.Process recognises — FSW + valid
	// NID + a trellis-encoded TSBK. Mirrors the in-package
	// buildLockedStream helpers but constructed here to avoid
	// cross-package internal-test dependencies.
	dibits := buildP25LockedDibits(nac)

	// Wire a small mock IQ file. Content doesn't matter — the
	// stubbed factory ignores the IQ; the mock SDR file only
	// has to exist long enough for ccdecoder to keep the
	// pipeline alive while we publish KindHuntProgress.
	dir := t.TempDir()
	iqPath := filepath.Join(dir, "mock.cfile")
	// 4 MiB ≈ 850 ms of streaming at the mock SDR's default
	// 2.4 MHz pacing — comfortably long enough for the
	// supervisor to publish KindHuntProgress, the ccdecoder to
	// construct the pipeline, and at least one Process(iq) call
	// to land afterwards so our pipeline's sync.Once fires.
	if err := os.WriteFile(iqPath, make([]byte, 4*1024*1024), 0o600); err != nil {
		t.Fatal(err)
	}
	sdr.Register(&sdr.MockDriver{Files: []string{iqPath}})

	// Stub the P25 factory with one that constructs a real
	// phase1.ControlChannel + a Pipeline whose Process(iq)
	// pumps a slice of the pre-built dibit stream forward.
	// The pipeline runs through the whole dibit stream within
	// a few IQ chunks, then idles.
	restore := ccdecoder.SetTestFactory(trunking.ProtocolP25, newDibitInjectingP25Factory(dibits))
	t.Cleanup(restore)

	cfg := config.Default()
	cfg.SDR.SampleRate = 480_000 // any value passes config validation; mock SDR doesn't care
	cfg.SDR.Devices = []config.DeviceConfig{
		{Serial: "mock-00", Role: "control"},
	}
	cfg.Trunking.Systems = []config.SystemConfig{
		{Name: "Alpha", Protocol: "p25", ControlChannels: []uint32{controlFreqHz}},
	}
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.Metrics.Enabled = true

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "integration-cc", logger)
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
			ls, ok := ev.Payload.(phase1.LockState)
			if !ok {
				t.Errorf("CCLocked payload type = %T, want phase1.LockState", ev.Payload)
				continue
			}
			if ls.NAC != nac {
				t.Errorf("LockState.NAC = %#x, want %#x", ls.NAC, nac)
			}
			if ls.FrequencyHz != controlFreqHz {
				t.Errorf("LockState.FrequencyHz = %d, want %d",
					ls.FrequencyHz, controlFreqHz)
			}
			locked = true
			break WaitLoop
		case <-deadline:
			t.Fatalf("no cc.locked event arrived within 5s")
		}
	}

	waitForScannerLock(t, base, "Alpha", 2*time.Second)

	// Verify the cc-locked gauge reaches 1 for our system. The
	// gauge is set by the events.KindCCLocked handler in
	// internal/metrics/prom.go; it's labelled by system /
	// repeater, so we check for the family + a value of 1
	// without pinning the exact label content.
	// gophertrunk_control_channel_locked{system="…"} = 1 is the
	// Prometheus-side signal that the daemon's metrics handler
	// saw the same cc.locked event and updated the gauge. The
	// system label can be "unknown" when the phase1 LockState's
	// SystemName isn't populated; that's fine — the metric
	// family + value pair is what we assert.
	body := scrape(t, base+"/metrics")
	if !strings.Contains(body, "gophertrunk_control_channel_locked{") {
		t.Errorf("/metrics missing gophertrunk_control_channel_locked gauge family:\n%s", body)
	}
	if !strings.Contains(body, `gophertrunk_control_channel_locked{system=`) ||
		!strings.Contains(body, "} 1") {
		t.Errorf("/metrics gophertrunk_control_channel_locked did not reach 1 for any system:\n%s", body)
	}
	if !strings.Contains(body, `gophertrunk_events_total{kind="cc.locked"} 1`) {
		t.Errorf("/metrics did not count one cc.locked event")
	}
}

// buildP25LockedDibits assembles one FSW + NID + TSBK dibit
// frame, padded with a long idle prefix so the receiver chain
// has room to lock its symbol clock if a future revision of this
// test drops the factory stub and runs synthesized IQ end-to-end.
// Mirrors the in-package phase1 test helpers' layout exactly.
func buildP25LockedDibits(nac uint16) []uint8 {
	const idlePrefix = 10
	out := make([]uint8, 0, idlePrefix+24+32+98+16)
	for i := 0; i < idlePrefix; i++ {
		out = append(out, 0)
	}
	out = append(out, phase1.FrameSyncWord[:]...)
	nidBits := phase1.EncodeNIDBits(nac, phase1.DUIDTrunkingSignaling)
	for i := 0; i < 32; i++ {
		out = append(out, (nidBits[2*i]<<1)|nidBits[2*i+1])
	}
	tsbk := phase1.AssembleTSBK(phase1.TSBK{LB: true, Opcode: phase1.OpRFSSStatusBroadcast})
	out = append(out, phase1.EncodeTSBKChannel(tsbk)...)
	for i := 0; i < 16; i++ {
		out = append(out, 0)
	}
	return out
}

// newDibitInjectingP25Factory returns a PipelineFactory that
// constructs a real phase1.ControlChannel for the supplied
// system and wraps it in a pipeline whose Process(iq) is a no-op
// — instead the pipeline pumps the pre-built dibit stream into
// cc.Process exactly once, on first invocation. Subsequent
// Process calls do nothing.
//
// The test calls this once via ccdecoder.SetTestFactory so the
// production factory map's P25 entry is replaced for the
// duration of the test.
func newDibitInjectingP25Factory(dibits []uint8) ccdecoder.PipelineFactory {
	return func(opts ccdecoder.PipelineOptions) (ccdecoder.ProtocolPipeline, error) {
		cc := phase1.NewControlChannel(opts.Bus, opts.Log, opts.FrequencyHz)
		return &dibitInjectingPipeline{cc: cc, dibits: dibits}, nil
	}
}

type dibitInjectingPipeline struct {
	cc     *phase1.ControlChannel
	dibits []uint8
	once   sync.Once
}

func (p *dibitInjectingPipeline) Process(_ []complex64) {
	// One Process call delivers the entire pre-built dibit
	// stream — small enough (~180 dibits) that latency is
	// negligible. The phase1 state machine emits cc.locked on
	// the bus inline as soon as the FSW + NID + valid TSBK land.
	p.once.Do(func() { p.cc.Process(p.dibits, 0) })
}

func (p *dibitInjectingPipeline) Reset()       {}
func (p *dibitInjectingPipeline) Close() error { return nil }

// waitForScannerLock polls /api/v1/scanner until the named system
// reports state=locked or the timeout fires.
func waitForScannerLock(t *testing.T, base, system string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(base + "/api/v1/scanner")
		if err == nil {
			var status struct {
				Systems []struct {
					Name  string `json:"name"`
					State string `json:"state"`
				} `json:"systems"`
			}
			err := json.NewDecoder(resp.Body).Decode(&status)
			resp.Body.Close()
			if err == nil {
				for _, s := range status.Systems {
					if s.Name == system && s.State == "locked" {
						return
					}
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Errorf("/api/v1/scanner did not report state=locked for %q within %v", system, timeout)
}
