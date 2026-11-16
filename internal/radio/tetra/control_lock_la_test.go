package tetra

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// lockLogCC builds a control channel whose INFO log goes to buf and whose
// clock the test drives.
func lockLogCC(t *testing.T, now *time.Time) (*ControlChannel, *events.Bus, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	bus := events.NewBus(64)
	cc := New(Options{
		Bus:         bus,
		Log:         slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})),
		SystemName:  "Sys",
		FrequencyHz: 412_000_000,
		Now:         func() time.Time { return *now },
	})
	return cc, bus, &buf
}

// lockLines returns the "tetra cc locked" / "tetra cc lost" lines in order.
func lockLines(buf *bytes.Buffer) []string {
	var out []string
	for _, l := range strings.Split(buf.String(), "\n") {
		if strings.Contains(l, `msg="tetra cc locked"`) || strings.Contains(l, `msg="tetra cc lost"`) {
			out = append(out, l)
		}
	}
	return out
}

// TestStaleLockLossIsLogged is the #1321 regression for "tetra cc locked"
// repeating with nothing between the lines. A weak cell that falls silent past
// the stale timeout loses its lock through CheckStale, which published cc.lost
// on the bus but wrote no log line; the next sync burst then logged "tetra cc
// locked" again. The reporter's weakest cell showed 19 such lines in 11 minutes
// and no loss anywhere in the log. Every loss must leave a line that pairs with
// the lock line and says why.
func TestStaleLockLossIsLogged(t *testing.T) {
	now := time.Unix(1_000, 0)
	cc, bus, buf := lockLogCC(t, &now)
	defer bus.Close()

	id := LockState{FrequencyHz: 412_000_000, MCC: 222, MNC: 1}
	cc.maybeLock(id)

	// Silent for 7 s against a 5 s timeout: the watchdog drops the lock.
	now = now.Add(7 * time.Second)
	cc.CheckStale(now, 5*time.Second)
	if cc.Locked() {
		t.Fatal("control channel still locked after going silent past the timeout")
	}

	// The carrier comes back.
	cc.maybeLock(id)

	lines := lockLines(buf)
	if len(lines) != 3 {
		t.Fatalf("lock/loss lines = %d, want 3 (locked, lost, locked):\n%s", len(lines), strings.Join(lines, "\n"))
	}
	if !strings.Contains(lines[0], "tetra cc locked") ||
		!strings.Contains(lines[1], "tetra cc lost") ||
		!strings.Contains(lines[2], "tetra cc locked") {
		t.Fatalf("lines out of order, want locked / lost / locked:\n%s", strings.Join(lines, "\n"))
	}
	for _, want := range []string{"silent_for=7s", "timeout=5s", "mcc=222", "mnc=1", "reason="} {
		if !strings.Contains(lines[1], want) {
			t.Errorf("loss line missing %q: %s", want, lines[1])
		}
	}

	// A loss the pipeline declares itself (payload drought) is logged too, with
	// the pipeline's reason; a second MarkLost while unlocked stays silent.
	cc.MarkLostReason("payload drought")
	cc.MarkLost()
	lines = lockLines(buf)
	if len(lines) != 4 || !strings.Contains(lines[3], `reason="payload drought"`) {
		t.Fatalf("want one more loss line carrying the pipeline's reason, got:\n%s", strings.Join(lines, "\n"))
	}
}

// TestLockIdentityCarriesSysInfoLocationArea is the #1321 regression for the
// lock line reporting la=0 on a cell whose location area is 46. On real air a
// BSCH names MCC/MNC only; the location area arrives in D-MLE-SYSINFO, which
// was logged but never reached the lock identity — so the lock line, the
// cc.locked / cc.lost payloads and the topology snapshot (the Systems panel)
// all read location area 0.
func TestLockIdentityCarriesSysInfoLocationArea(t *testing.T) {
	now := time.Unix(1_000, 0)
	cc, bus, buf := lockLogCC(t, &now)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	bsch := LockState{FrequencyHz: 412_000_000, MCC: 222, MNC: 1} // real-air BSCH: no LA
	cc.maybeLock(bsch)
	if lines := lockLines(buf); len(lines) != 1 || strings.Contains(lines[0], "la=") {
		t.Fatalf("the first lock line must not claim a location area before any SYSINFO decoded:\n%s",
			strings.Join(lines, "\n"))
	}
	drainLockEvents(sub)

	// SYSINFO repeats every multiframe; two agreeing decodes confirm the LA.
	ext := SysInfoExt{SCCHInUse: 1, LocationArea: 46}
	cc.learnSysInfoExt(ext)
	if got := cc.Topology().LocationArea; got != 0 {
		t.Fatalf("location area adopted from a single SYSINFO: %d (one mis-corrected block must not rewrite the identity)", got)
	}
	cc.learnSysInfoExt(ext)
	if got := cc.Topology().LocationArea; got != 46 {
		t.Fatalf("topology location area = %d after two agreeing SYSINFOs, want 46", got)
	}
	if got := cc.TopologySnapshot().LocationArea; got != 46 {
		t.Fatalf("topology snapshot location area = %d, want 46", got)
	}
	if !strings.Contains(buf.String(), `msg="tetra cc location area learned"`) ||
		!strings.Contains(buf.String(), "la=46") {
		t.Errorf("no line announced the learned location area:\n%s", buf.String())
	}

	// The steady stream of LA-less BSCHs agrees with the completed identity:
	// no re-announcement, and the LA is not voted away again.
	for i := 0; i < 5; i++ {
		cc.maybeLock(bsch)
		cc.learnSysInfoExt(ext)
	}
	if got := drainLockEvents(sub); len(got) != 0 {
		t.Fatalf("steady BSCHs re-published cc.locked %d times after the LA was learned: %v", len(got), got)
	}
	if got := lockStateOf(cc).LocationArea; got != 46 {
		t.Fatalf("LA-less BSCHs stripped the location area: %d", got)
	}

	// A loss and re-lock announce the full identity.
	now = now.Add(7 * time.Second)
	cc.CheckStale(now, 5*time.Second)
	cc.maybeLock(bsch)
	got := drainLockEvents(sub)
	want := LockState{FrequencyHz: 412_000_000, MCC: 222, MNC: 1, LocationArea: 46}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("re-lock published %v, want exactly [%v]", got, want)
	}
	lines := lockLines(buf)
	last := lines[len(lines)-1]
	if !strings.Contains(last, "tetra cc locked") || !strings.Contains(last, "la=46") {
		t.Errorf("re-lock line does not carry la=46: %s", last)
	}
}

// TestSysInfoLocationAreaNeedsTwoAgreeingDecodes: a single SYSINFO carrying a
// different location area (a CRC-passing, mis-corrected block) must not rewrite
// the locked identity; two in a row (a real change) must.
func TestSysInfoLocationAreaNeedsTwoAgreeingDecodes(t *testing.T) {
	now := time.Unix(1_000, 0)
	cc, bus, _ := lockLogCC(t, &now)
	defer bus.Close()

	cc.maybeLock(LockState{FrequencyHz: 412_000_000, MCC: 222, MNC: 1})
	good := SysInfoExt{LocationArea: 46}
	bogus := SysInfoExt{LocationArea: 9001}
	cc.learnSysInfoExt(good)
	cc.learnSysInfoExt(good)

	cc.learnSysInfoExt(bogus)
	cc.learnSysInfoExt(good)
	cc.learnSysInfoExt(bogus)
	if got := lockStateOf(cc).LocationArea; got != 46 {
		t.Fatalf("isolated bogus SYSINFOs rewrote the location area: %d, want 46", got)
	}

	moved := SysInfoExt{LocationArea: 47}
	cc.learnSysInfoExt(moved)
	cc.learnSysInfoExt(moved)
	if got := lockStateOf(cc).LocationArea; got != 47 {
		t.Fatalf("location area = %d after two agreeing SYSINFOs with a new LA, want 47", got)
	}
}

// TestSysInfoLocationAreaDroppedOnNetworkChange: the learned LA belongs to the
// cell it was broadcast by. A confirmed change of network identity on the
// carrier must not carry the old cell's location area over.
func TestSysInfoLocationAreaDroppedOnNetworkChange(t *testing.T) {
	now := time.Unix(1_000, 0)
	cc, bus, _ := lockLogCC(t, &now)
	defer bus.Close()

	cc.maybeLock(LockState{FrequencyHz: 412_000_000, MCC: 222, MNC: 1})
	cc.learnSysInfoExt(SysInfoExt{LocationArea: 46})
	cc.learnSysInfoExt(SysInfoExt{LocationArea: 46})

	other := LockState{FrequencyHz: 412_000_000, MCC: 228, MNC: 7}
	cc.maybeLock(other)
	cc.maybeLock(other) // second agreeing BSCH confirms the identity change
	if got := lockStateOf(cc); got != other {
		t.Fatalf("identity after a confirmed network change = %+v, want %+v (no inherited LA)", got, other)
	}
}
