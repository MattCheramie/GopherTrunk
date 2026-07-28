package tetra

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestResyncResetRestoresDecodeAfterBaseIdxRestart is the regression for the
// slow TETRA CC recovery after a noise burst. The recovery fix resets the
// receiver's DSP loops to centre so they reacquire fast — but rx.Reset restarts
// the receiver's dibit index at 0, and the control channel's SB decoder indexes
// its rolling buffer by ABSOLUTE dibit index. Without a paired ResyncReset the
// stale bufBase drives every future sync burst into the "look-back already
// trimmed" branch and the channel never decodes another SB (the exact
// zero-sb_bursts stall seen in the field debug.log). This asserts: a baseIdx
// restart WITHOUT ResyncReset stops decoding, and ResyncReset restores it while
// preserving the learned identity so re-lock stays fast.
func TestResyncResetRestoresDecodeAfterBaseIdxRestart(t *testing.T) {
	cc, bus := debugCC(t, io.Discard, slog.LevelDebug)
	defer bus.Close()

	stream := buildCleanSBStream(t, 0x2D, 3, 5, cleanSysinfoPDU())

	// Phase 1: decode a clean SB at a large absolute baseIdx (as on a
	// long-running live stream). The channel locks and learns the colour code.
	cc.Process(stream, 1_000_000)
	if got := cc.DrainStats().BSCHOK; got < 1 {
		t.Fatalf("phase 1 BSCHOK = %d, want >= 1 (clean SB should decode)", got)
	}
	wantColour := cc.ColourCode()
	wantTopo := cc.Topology()

	// Phase 2: the receiver reset restarts the dibit index at 0, but WITHOUT the
	// paired ResyncReset the proc's absolute bufBase is still ~1e6, so the SB
	// look-back is judged already-trimmed and nothing decodes. This captures the
	// corruption as a passing assertion.
	cc.Process(stream, 0)
	if got := cc.DrainStats().BSCHOK; got != 0 {
		t.Fatalf("phase 2 BSCHOK = %d, want 0 (stale bufBase must strand the SB decode)", got)
	}

	// Phase 3: ResyncReset drops the dibit-sync scratch so the next Process
	// rebuilds it from the fresh baseIdx and decoding resumes immediately.
	cc.ResyncReset()
	cc.Process(stream, 0)
	if got := cc.DrainStats().BSCHOK; got < 1 {
		t.Fatalf("phase 3 BSCHOK = %d, want >= 1 (ResyncReset must restore decode)", got)
	}

	// Identity that makes re-lock fast must survive the reset.
	if got := cc.ColourCode(); got != wantColour {
		t.Errorf("colour code after ResyncReset = %#x, want %#x (must be preserved)", got, wantColour)
	}
	if got := cc.Topology(); got.MCC != wantTopo.MCC || got.MNC != wantTopo.MNC {
		t.Errorf("identity after ResyncReset = MCC %d/MNC %d, want %d/%d",
			got.MCC, got.MNC, wantTopo.MCC, wantTopo.MNC)
	}
}

// TestNeedsResyncPredicate pins the heartbeat-age predicate the pipeline uses to
// trigger a DSP reacquire: no-op before the first decode, false while activity is
// fresh, true once the heartbeat is older than the timeout — and, crucially,
// still true after MarkLost (MarkLost does not clear the heartbeat), so the
// reacquire keeps firing to bound the post-loss recovery tail.
func TestNeedsResyncPredicate(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()

	base := time.Unix(1_000, 0)
	now := base
	cc := New(Options{
		Bus: bus, Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		SystemName: "Sys", FrequencyHz: 412_000_000,
		Now: func() time.Time { return now },
	})
	const timeout = 500 * time.Millisecond

	// Cold: no decode yet ⇒ never needs resync.
	if cc.NeedsResync(now.Add(time.Hour), timeout) {
		t.Fatal("NeedsResync true before the first decode, want false")
	}

	// A lock stamps the heartbeat at t=base.
	cc.maybeLock(LockState{FrequencyHz: 412_000_000})

	if cc.NeedsResync(base.Add(300*time.Millisecond), timeout) {
		t.Error("NeedsResync true while activity is fresh (300ms < 500ms), want false")
	}
	if !cc.NeedsResync(base.Add(600*time.Millisecond), timeout) {
		t.Error("NeedsResync false after the heartbeat went stale (600ms > 500ms), want true")
	}

	// After MarkLost the heartbeat is preserved, so a stale channel still asks to
	// reacquire — this is what bounds the recovery tail past cc.lost.
	cc.MarkLost()
	if cc.Locked() {
		t.Fatal("channel still locked after MarkLost")
	}
	if !cc.NeedsResync(base.Add(600*time.Millisecond), timeout) {
		t.Error("NeedsResync false after MarkLost, want true (heartbeat must survive loss)")
	}
}

// TestResyncResetPreservesLearnedIndividuals pins the invariant that a resync
// keeps the learned individual/group classification: ResyncReset clears only the
// dibit-sync scratch (proc) and the transient soft / fragment stashes, never the
// individuals set. If a resync dropped it, every noise burst would regress a
// subscriber ISSI back to being surfaced as a phantom talkgroup until it was
// seen as a party again. The reassembly buffer, by contrast, MUST be cleared —
// its continuation belonged to the pre-resync bit index.
func TestResyncResetPreservesLearnedIndividuals(t *testing.T) {
	cc := New(Options{SystemName: "Sys", FrequencyHz: 412_000_000})

	cc.noteIndividual(0x0123AB)
	if !cc.isIndividual(0x0123AB) {
		t.Fatal("noteIndividual did not record the SSI")
	}

	// Seed an in-flight fragment reassembly to prove the resync abandons it.
	cc.stashFragment(MACResource{}, []byte{1, 0, 1})

	cc.ResyncReset()

	if !cc.isIndividual(0x0123AB) {
		t.Error("ResyncReset dropped a learned individual — classification would regress after every resync")
	}
	cc.mu.Lock()
	fragActive := cc.fragActive
	cc.mu.Unlock()
	if fragActive {
		t.Error("ResyncReset left a half-reassembled TM-SDU active; it must abandon the pre-resync fragment")
	}
}
