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

// newResyncTestPipeline builds a locked TETRA pipeline at the 144 kHz channel
// rate with a debug logger, arms its heartbeat with one clean BSCH decode, and
// runs the sync Process call so lastSeenActivity tracks the armed heartbeat. It
// returns the pipeline, the log buffer, and the resync sample budget so each test
// drives the signal-time trigger by feeding sample chunks rather than a clock.
func newResyncTestPipeline(t *testing.T) (*tetraPipeline, *bytes.Buffer, int64) {
	t.Helper()
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)

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

	// Arm the heartbeat with a clean BSCH decode, then run one Process to sync
	// lastSeenActivity to the armed heartbeat (the first post-arm call always
	// resets the budget rather than accumulating).
	p.cc.Process(buildArmingSBStream(), 0)
	if !p.cc.Locked() {
		t.Fatal("control channel did not lock on the clean arming burst")
	}
	p.Process(nil)
	if strings.Contains(buf.String(), "tetra: dsp resync") {
		t.Fatalf("resync fired on the initial sync call\n%s", buf.String())
	}
	buf.Reset()

	budget := int64(tetraResyncTimeout.Seconds() * p.rateHz) // 1.5 s × 144 kHz = 216000
	if budget <= 0 {
		t.Fatalf("resync sample budget = %d, want > 0", budget)
	}
	return p, buf, budget
}

// feedNoDecode drives the real Process path with n samples of a fixed low-level
// constant — nonzero (so the receiver's AGC has no zero-RMS divide) but with no
// synchronisation training sequence, so it never decodes a valid BSCH. The
// heartbeat holds and checkResync accumulates the samples against its budget — the
// signal-time analogue of "the receiver processed n samples of signal without a
// CRC-clean decode".
func feedNoDecode(p *tetraPipeline, n int64) {
	buf := make([]complex64, n)
	for i := range buf {
		buf[i] = complex(0.01, 0.01)
	}
	p.Process(buf)
}

// TestTETRAResyncFiresOnSignalTimeDrought pins the signal-time trigger: once the
// receiver has PROCESSED a full tetraResyncTimeout-worth of samples with no CC
// decode, exactly one "tetra: dsp resync" fires, and the budget resets so an
// immediate sub-budget feed does not fire a second (the inherent throttle).
func TestTETRAResyncFiresOnSignalTimeDrought(t *testing.T) {
	p, buf, budget := newResyncTestPipeline(t)

	// One full budget of processed samples with no decode → exactly one resync.
	feedNoDecode(p, budget)
	if got := strings.Count(buf.String(), "tetra: dsp resync"); got != 1 {
		t.Fatalf("want exactly one resync after a full-budget drought, got %d\n%s", got, buf.String())
	}

	// The budget reset on that resync throttles the next one: a sub-budget feed
	// must not fire again.
	buf.Reset()
	feedNoDecode(p, budget-1)
	if strings.Contains(buf.String(), "tetra: dsp resync") {
		t.Fatalf("resync fired again before another full budget was processed (throttle failed)\n%s", buf.String())
	}

	// Crossing the budget once more fires exactly one more.
	feedNoDecode(p, 1)
	if got := strings.Count(buf.String(), "tetra: dsp resync"); got != 1 {
		t.Fatalf("want one resync once the budget was crossed again, got %d\n%s", got, buf.String())
	}
}

// TestTETRAResyncIgnoresWallClockStarvation is the fail-first guard for the
// multislot "losing dsp sync" fix. Under concurrent-call CPU load the CC decode
// goroutine is descheduled and its IQ is dropped upstream, so wall-clock time
// passes while NO signal is processed. The old wall-clock trigger fired a
// destructive reset-to-centre in exactly this case, discarding a still-good lock
// (~46 resyncs in the reporter's capture). The signal-time trigger must NOT: zero
// processed samples ⇒ zero budget ⇒ no resync, no matter how much wall clock
// elapses. Fail-first: revert checkResync to the wall-clock NeedsResync trigger and
// the aged clock below fires a resync, turning the no-resync assertion red.
func TestTETRAResyncIgnoresWallClockStarvation(t *testing.T) {
	p, buf, _ := newResyncTestPipeline(t)

	// Simulate a long CPU/scheduling stall: wall clock jumps far past the window,
	// but the starved goroutine processed no IQ (dropped upstream) — Process(nil).
	base := time.Unix(0, p.cc.LastActivityNano())
	p.now = func() time.Time { return base.Add(3 * tetraResyncTimeout) }
	for i := 0; i < 8; i++ {
		p.Process(nil)
	}
	if strings.Contains(buf.String(), "tetra: dsp resync") {
		t.Fatalf("resync fired under wall-clock starvation with no processed signal (destroys a good lock)\n%s", buf.String())
	}
}

// TestTETRAResyncIgnoresTransientStalls pins the anti-churn property that is the
// heart of the multislot fix: the trigger is signal-time, so a starved cadence of
// small chunks with large WALL-CLOCK gaps between them must not fire until the
// cumulative PROCESSED samples cross the budget. The old wall-clock trigger fired
// on the very first chunk once the clock aged past the window, resetting a
// still-good lock every window under concurrent-call CPU load. Fail-first: revert
// checkResync to the wall-clock trigger and the aged clock fires on chunk one.
func TestTETRAResyncIgnoresTransientStalls(t *testing.T) {
	p, buf, budget := newResyncTestPipeline(t)

	// Race the wall clock far ahead of the window; only cumulative processed
	// samples must matter.
	base := time.Unix(0, p.cc.LastActivityNano())
	p.now = func() time.Time { return base.Add(10 * tetraResyncTimeout) }

	const chunk = 10000
	var fed int64
	for fed = 0; fed+chunk < budget; fed += chunk {
		feedNoDecode(p, chunk)
		if strings.Contains(buf.String(), "tetra: dsp resync") {
			t.Fatalf("resync fired at %d/%d cumulative samples despite the wall clock — signal-time trigger broken\n%s",
				fed+chunk, budget, buf.String())
		}
	}

	// Crossing the cumulative budget finally fires exactly one reset.
	feedNoDecode(p, budget-fed)
	if got := strings.Count(buf.String(), "tetra: dsp resync"); got != 1 {
		t.Fatalf("want one resync once cumulative samples crossed the budget, got %d\n%s", got, buf.String())
	}
}

// TestTETRAResyncDecodeClearsBudget pins the reset-on-decode branch: a CC decode
// (the heartbeat advancing past the value checkResync last saw, exactly as
// noteActivity stamps it) clears the accumulated sample budget, so an intermittent
// decoder never accumulates toward a reset. White-box (in-package): it feeds a
// near-budget drought, then simulates the heartbeat advancing and asserts the
// budget zeroes.
func TestTETRAResyncDecodeClearsBudget(t *testing.T) {
	p, buf, budget := newResyncTestPipeline(t)

	feedNoDecode(p, budget-1)
	if p.samplesSinceDecode != budget-1 {
		t.Fatalf("samplesSinceDecode = %d after a sub-budget feed, want %d", p.samplesSinceDecode, budget-1)
	}
	if strings.Contains(buf.String(), "tetra: dsp resync") {
		t.Fatalf("resync fired before the budget was reached\n%s", buf.String())
	}

	// A fresh decode: the heartbeat is now newer than the value checkResync last
	// saw. The next Process must clear the budget rather than accumulate.
	p.lastSeenActivity = p.cc.LastActivityNano() - 1
	p.Process(nil)
	if p.samplesSinceDecode != 0 {
		t.Fatalf("a decode did not clear the accumulated budget: samplesSinceDecode = %d, want 0", p.samplesSinceDecode)
	}
}
