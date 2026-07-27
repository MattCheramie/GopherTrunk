package ccdecoder

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// buildArmingSBStream assembles a clean TETRA synchronisation burst as a hard
// dibit stream — enough for the control channel to decode the BSCH and stamp its
// activity heartbeat, so NeedsResync/checkResync have a live heartbeat to age
// out. Mirrors the in-package tetra buildCleanSBStream but only needs the BSCH to
// decode (the BNCH may be idle), built here from exported tetra symbols.
func buildArmingSBStream() []uint8 {
	setBits := func(bits []byte, off, n int, v uint32) {
		for i := 0; i < n; i++ {
			bits[off+i] = byte((v >> uint(n-1-i)) & 1)
		}
	}
	syncInfo := make([]byte, 60)
	setBits(syncInfo, 4, 6, 0x2D) // colour code
	setBits(syncInfo, 31, 10, 3)  // MCC
	setBits(syncInfo, 41, 14, 5)  // MNC
	bschDibits := tetra.TetraBitsToDibits(tetra.EncodeBSCH(syncInfo))

	var s []uint8
	s = append(s, make([]uint8, 50)...)          // pad before BSCH (look-back)
	s = append(s, bschDibits...)                 // BSCH (60)
	s = append(s, tetra.SyncTrainingDibits()...) // STS (19)
	s = append(s, make([]uint8, 15)...)          // broadcast (15)
	s = append(s, make([]uint8, 108)...)         // BNCH (108; idle, need not decode)
	s = append(s, make([]uint8, 300)...)         // trailing look-ahead margin
	return s
}

// TestTETRAResyncTriggerAndThrottle drives the pipeline's fast DSP re-acquire on
// a deterministic clock: a fresh heartbeat produces no resync, a heartbeat aged
// past tetraResyncTimeout produces exactly one "tetra: dsp resync" line, and a
// second call in the same window is throttled. This is the fail-first guard for
// the recovery fix — reverting the single p.checkResync() call in Process turns
// it red.
func TestTETRAResyncTriggerAndThrottle(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	bus := events.NewBus(8)
	defer bus.Close()

	pp, err := newTETRAPipeline(PipelineOptions{
		Bus: bus, Log: logger, SystemName: "Test", FrequencyHz: 412_000_000,
		SampleRateHz: 144_000,
		System: trunking.System{
			Name: "Test", Protocol: trunking.ProtocolTETRA,
			ControlChannels: []uint32{412_000_000},
		},
	})
	if err != nil {
		t.Fatalf("newTETRAPipeline: %v", err)
	}
	p := pp.(*tetraPipeline)

	// Fixed pipeline clock, anchored just before arming so the freshly-stamped
	// heartbeat (real-time, stamped inside cc.Process) reads as ~0 age at t0.
	t0 := time.Now()
	cur := t0
	p.now = func() time.Time { return cur }

	// Arm the heartbeat with a clean BSCH decode.
	p.cc.Process(buildArmingSBStream(), 0)
	if !p.cc.Locked() {
		t.Fatal("control channel did not lock on the clean arming burst")
	}

	// Drive the real Process path (empty IQ ⇒ rx is a no-op) so the test also
	// guards the p.checkResync() wiring in Process, not just checkResync itself.
	// Fresh heartbeat: no resync.
	p.Process(nil)
	if strings.Contains(buf.String(), "tetra: dsp resync") {
		t.Fatalf("resync fired while heartbeat was fresh\n%s", buf.String())
	}

	// Age the heartbeat past the timeout: exactly one resync.
	cur = t0.Add(tetraResyncTimeout + 100*time.Millisecond)
	p.Process(nil)
	if !strings.Contains(buf.String(), "tetra: dsp resync") {
		t.Fatalf("expected a resync after the heartbeat went stale\n%s", buf.String())
	}

	// A second call within the same window is throttled (no second line).
	buf.Reset()
	p.Process(nil)
	if strings.Contains(buf.String(), "tetra: dsp resync") {
		t.Errorf("resync fired twice within one timeout window (throttle failed)\n%s", buf.String())
	}
}
