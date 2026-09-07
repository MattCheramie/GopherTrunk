//go:build integration

package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier2"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestDaemonDMRTier2CampsThenGrantsOnKeyup pins the full issue #1036 transition
// end-to-end through the production daemon: a conventional DMR / IPSC repeater
// that is IDLE before the first transmission must CAMP on the channel (no
// `cchunt.failed`, scanner state "camped") and then, when a station keys up,
// LOCK and publish a GRANT carrying the talkgroup + source RID decoded off the
// Voice LC Header.
//
// The camp state machine is unit-tested at the supervisor level
// (internal/scanner/cchunt.TestSupervisorCampsIdleConventionalDMR). This is the
// integration counterpart the issue asked for — it exercises the real
// receiver → tier2 → grant path across the camp→keyup boundary, so the whole
// chain can't silently break later. Reporter's repeater is colour code 10, so
// the fixture uses that.
func TestDaemonDMRTier2CampsThenGrantsOnKeyup(t *testing.T) {
	const (
		controlFreqHz = 460_500_000
		sampleRateHz  = 48_000
		sps           = 10
		span          = 8
		alpha         = 0.20
		deviationHz   = 1944.0
		colorCode     = 0xA // colour code 10, matching the reporter's IPSC repeater
		groupID       = 0x2A2
		sourceID      = 0x315AC5
		burstRepeats  = 60
	)

	// ~1.5 s of low-level noise: the idle channel before anyone keys up. It
	// never decodes, so the supervisor dwells, fails to lock, and — because
	// dmr-tier2 CampsWhenIdle — camps instead of raising cchunt.failed.
	idle := makeNoiseIQ(sampleRateHz*3/2, 0.05, 1)
	// The keyup: repeated Voice LC Header bursts carrying the TG + source RID.
	keyupDibits := buildDMRTier2VoiceLCHeaderDibits(burstRepeats, colorCode, groupID, sourceID)
	keyup := demod.ModulateC4FM(keyupDibits, sps, span, alpha, sampleRateHz, deviationHz)
	iq := append(idle, keyup...)

	dir := t.TempDir()
	iqPath := filepath.Join(dir, "dmr-tier2-camp.cfile")
	if err := writeIQToU8File(iqPath, iq); err != nil {
		t.Fatalf("write IQ: %v", err)
	}
	sdr.Register(&sdr.MockDriver{Files: []string{iqPath}})

	cfg := config.Default()
	cfg.SDR.SampleRate = sampleRateHz
	cfg.SDR.Devices = []config.DeviceConfig{{Serial: "mock-00", Role: "control"}}
	cfg.Trunking.Systems = []config.SystemConfig{
		{Name: "IPSCRepeater", Protocol: "dmr-tier2", ControlChannels: []uint32{controlFreqHz}},
	}
	// Short dwell so the camp decision (and this test) is quick; the ~1.5 s idle
	// prefix still spans several dwells before the keyup. Enabled must be set
	// explicitly: any non-zero CCHunt field takes the config out of its
	// "default (enabled)" zero value, so the supervisor would otherwise be
	// switched off (daemon.go's cchEnabled gate).
	cfg.Scanner.CCHunt.Enabled = true
	cfg.Scanner.CCHunt.DwellMs = 400
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.Metrics.Enabled = true

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "integration-cc-dmr-tier2-camp", logger)
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}

	sub := d.Bus().Subscribe()
	defer sub.Close()

	// Drain the bus for the life of the run: record a cchunt.failed for our
	// system (the exact #1036 regression), and capture the lock + grant.
	var (
		mu       sync.Mutex
		locked   bool
		gotGrant bool
		grantTG  uint32
		grantSrc uint32
	)
	huntFailed := make(chan struct{}, 4)
	stopDrain := make(chan struct{})
	drainDone := make(chan struct{})
	go func() {
		defer close(drainDone)
		for {
			select {
			case <-stopDrain:
				return
			case ev, ok := <-sub.C:
				if !ok {
					return
				}
				switch ev.Kind {
				case events.KindHuntFailed:
					select {
					case huntFailed <- struct{}{}:
					default:
					}
				case events.KindCCLocked:
					if ls, ok := ev.Payload.(tier2.LockState); ok && ls.ColorCode == colorCode {
						mu.Lock()
						locked = true
						mu.Unlock()
					}
				case events.KindGrant:
					if g, ok := ev.Payload.(trunking.Grant); ok && g.GroupID == groupID {
						mu.Lock()
						gotGrant = true
						grantTG = g.GroupID
						grantSrc = g.SourceID
						mu.Unlock()
					}
				}
			}
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	runErrCh := make(chan error, 1)
	go func() { runErrCh <- d.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		close(stopDrain)
		<-drainDone
		select {
		case <-runErrCh:
		case <-time.After(3 * time.Second):
		}
	})

	base := "http://" + cfg.API.HTTPAddr
	waitReachable(t, base+"/api/v1/health", 3*time.Second)

	// 1) During the idle prefix the daemon must reach state=camped, and must not
	//    have raised cchunt.failed.
	if !waitForScannerState(t, base, "IPSCRepeater", "camped", 4*time.Second) {
		t.Fatalf("daemon never reported state=camped on the idle conventional DMR channel (issue #1036)")
	}
	select {
	case <-huntFailed:
		t.Fatalf("idle conventional DMR raised cchunt.failed — it must camp, not fail (issue #1036)")
	default:
	}

	// 2) On keyup, the daemon must lock and publish a grant with the TG + source.
	deadline := time.After(10 * time.Second)
	for {
		mu.Lock()
		l, g, tg, src := locked, gotGrant, grantTG, grantSrc
		mu.Unlock()
		if l && g {
			if tg != groupID {
				t.Errorf("grant GroupID = %#x, want %#x", tg, groupID)
			}
			if src != sourceID {
				t.Errorf("grant SourceID = %#x, want %#x (source RID off the Voice LC Header)", src, sourceID)
			}
			break
		}
		select {
		case <-huntFailed:
			t.Fatalf("conventional DMR raised cchunt.failed (issue #1036)")
		case <-deadline:
			t.Fatalf("no lock+grant after keyup within 10s (locked=%v grant=%v)", l, g)
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// makeNoiseIQ returns n samples of low-level complex Gaussian noise — a stand-in
// for an idle conventional channel that carries no decodable traffic.
func makeNoiseIQ(n int, amp float32, seed int64) []complex64 {
	rng := rand.New(rand.NewSource(seed))
	out := make([]complex64, n)
	for i := range out {
		out[i] = complex(amp*float32(rng.NormFloat64()), amp*float32(rng.NormFloat64()))
	}
	return out
}

// waitForScannerState polls /api/v1/scanner until the named system reports the
// wanted state, returning true on success and false on timeout. Generalises
// waitForScannerLock so a test can wait for "camped" as well as "locked".
func waitForScannerState(t *testing.T, base, system, want string, timeout time.Duration) bool {
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
					if s.Name == system && s.State == want {
						return true
					}
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}
