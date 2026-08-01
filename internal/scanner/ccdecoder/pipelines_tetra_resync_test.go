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

// TestTETRAResyncBacksOffWhenIneffective pins the exponential back-off that
// breaks the self-perpetuating resync loop: when a resync does not reacquire
// (the heartbeat stays stale, as under concurrent-call CPU starvation), the
// pipeline must NOT keep firing reset-to-centre every tetraResyncTimeout —
// it doubles the wait. A decode (heartbeat advance) snaps the cadence back to
// the base window. Fail-first: with a fixed tetraResyncTimeout throttle a
// resync fires at every window and the +1 window assertion below goes red.
func TestTETRAResyncBacksOffWhenIneffective(t *testing.T) {
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

	t0 := time.Now()
	cur := t0
	p.now = func() time.Time { return cur }
	p.cc.Process(buildArmingSBStream(), 0)
	if !p.cc.Locked() {
		t.Fatal("control channel did not lock on the clean arming burst")
	}

	// resyncFires advances the clock to t0+off and reports whether a resync line
	// was emitted on that Process call.
	resyncFires := func(off time.Duration) bool {
		buf.Reset()
		cur = t0.Add(off)
		p.Process(nil)
		return strings.Contains(buf.String(), "tetra: dsp resync")
	}

	// The heartbeat never advances (no decode), so every resync is ineffective.
	// #1 at one base window: fires (back-off starts at the base).
	if !resyncFires(tetraResyncTimeout + 100*time.Millisecond) {
		t.Fatalf("expected the first resync after the heartbeat went stale\n%s", buf.String())
	}
	// #2 one base window later: still fires; firing #1 with no decode since then
	// doubles the wait to 2×base for the NEXT one.
	if !resyncFires(2*tetraResyncTimeout + 100*time.Millisecond) {
		t.Fatalf("expected the second resync one window later\n%s", buf.String())
	}
	if p.resyncBackoff != 2*tetraResyncTimeout {
		t.Fatalf("back-off = %v after two ineffective resyncs, want %v", p.resyncBackoff, 2*tetraResyncTimeout)
	}
	// One base window after #2 the wait has doubled, so a resync must NOT fire
	// yet — this is the back-off a plain fixed throttle would violate.
	if resyncFires(3*tetraResyncTimeout + 100*time.Millisecond) {
		t.Fatalf("resync fired at the base window after an ineffective resync; back-off did not apply\n%s", buf.String())
	}
	// After the doubled wait elapses it fires again.
	if !resyncFires(4*tetraResyncTimeout + 100*time.Millisecond) {
		t.Fatalf("expected a resync once the doubled back-off elapsed\n%s", buf.String())
	}

	// A fresh heartbeat (the decode recovered) snaps the back-off back to the
	// base window so the next stall reacquires promptly again. Anchor the clock
	// to the re-armed heartbeat (stamped in real time inside cc.Process) so the
	// age reads well under one window regardless of test wall-clock timing.
	p.cc.Process(buildArmingSBStream(), 0)
	cur = time.Unix(0, p.cc.LastActivityNano()).Add(tetraResyncTimeout / 2)
	p.Process(nil)
	if p.resyncBackoff != 0 {
		t.Fatalf("back-off = %v after a fresh heartbeat, want it reset to 0 (base)", p.resyncBackoff)
	}
}
