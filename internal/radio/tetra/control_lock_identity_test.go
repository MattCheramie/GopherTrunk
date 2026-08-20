package tetra

import (
	"io"
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// lockStateOf reads the active lock identity under the mutex (in-package).
func lockStateOf(c *ControlChannel) LockState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.last
}

// drainLockEvents returns the KindCCLocked payloads currently queued.
func drainLockEvents(sub *events.Subscription) []LockState {
	var got []LockState
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				if s, ok := ev.Payload.(LockState); ok {
					got = append(got, s)
				}
			}
		default:
			return got
		}
	}
}

// TestMaybeLockSingleBogusBSCHDoesNotRewriteIdentity is the regression for the
// 19 Aug field flaps: a single CRC-passing but FEC-mis-corrected BSCH carried
// a garbage identity (mcc=996 mnc=3941) and rewrote the locked site identity
// wholesale — publishing a bogus cc.locked and site update, corrected only by
// the next good burst. One differing burst must not change anything; neither
// must a different bogus burst later (votes are consecutive).
func TestMaybeLockSingleBogusBSCHDoesNotRewriteIdentity(t *testing.T) {
	cc, bus := debugCC(t, io.Discard, slog.LevelInfo)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	a := LockState{FrequencyHz: 412_000_000, MCC: 250, MNC: 13}
	cc.maybeLock(a)
	if got := drainLockEvents(sub); len(got) != 1 || got[0] != a {
		t.Fatalf("first lock: got events %v, want exactly [%v]", got, a)
	}

	// One bogus burst: no event, identity unchanged.
	b := LockState{FrequencyHz: 412_000_000, MCC: 996, MNC: 3941, LocationArea: 13121}
	cc.maybeLock(b)
	if got := drainLockEvents(sub); len(got) != 0 {
		t.Fatalf("single bogus BSCH published cc.locked: %v", got)
	}
	if lockStateOf(cc) != a {
		t.Fatalf("single bogus BSCH rewrote the identity: %+v", lockStateOf(cc))
	}

	// The real identity resumes, then a DIFFERENT bogus burst appears once:
	// the earlier bogus vote must not corroborate it.
	cc.maybeLock(a)
	c2 := LockState{FrequencyHz: 412_000_000, MCC: 88, MNC: 4698}
	cc.maybeLock(c2)
	if got := drainLockEvents(sub); len(got) != 0 {
		t.Fatalf("scattered bogus BSCHs published cc.locked: %v", got)
	}
	if lockStateOf(cc) != a {
		t.Fatalf("scattered bogus BSCHs rewrote the identity: %+v", lockStateOf(cc))
	}
}

// TestMaybeLockIdentityChangeConfirmsOnSecondAgreeingBSCH pins that a GENUINE
// identity change (two consecutive agreeing bursts, e.g. an LA rehome) is
// adopted and published exactly once.
func TestMaybeLockIdentityChangeConfirmsOnSecondAgreeingBSCH(t *testing.T) {
	cc, bus := debugCC(t, io.Discard, slog.LevelInfo)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	a := LockState{FrequencyHz: 412_000_000, MCC: 250, MNC: 13}
	b := LockState{FrequencyHz: 412_000_000, MCC: 250, MNC: 13, LocationArea: 7}
	cc.maybeLock(a)
	drainLockEvents(sub)

	cc.maybeLock(b)
	cc.maybeLock(b)
	if got := drainLockEvents(sub); len(got) != 1 || got[0] != b {
		t.Fatalf("confirmed identity change: got events %v, want exactly [%v]", got, b)
	}
	if lockStateOf(cc) != b {
		t.Fatalf("identity after confirmation = %+v, want %+v", lockStateOf(cc), b)
	}
}

// TestMaybeLockInterveningActiveIdentityResetsVotes pins the CONSECUTIVE
// requirement: bogus, real, bogus must never accumulate to the threshold.
func TestMaybeLockInterveningActiveIdentityResetsVotes(t *testing.T) {
	cc, bus := debugCC(t, io.Discard, slog.LevelInfo)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	a := LockState{FrequencyHz: 412_000_000, MCC: 250, MNC: 13}
	b := LockState{FrequencyHz: 412_000_000, MCC: 996, MNC: 3941}
	cc.maybeLock(a)
	drainLockEvents(sub)

	cc.maybeLock(b)
	cc.maybeLock(a) // the active identity re-asserts itself
	cc.maybeLock(b)
	if got := drainLockEvents(sub); len(got) != 0 {
		t.Fatalf("non-consecutive bogus bursts were adopted: %v", got)
	}
	if lockStateOf(cc) != a {
		t.Fatalf("identity = %+v, want %+v", lockStateOf(cc), a)
	}
}

// TestMaybeLockFrequencyOnlyLockUnaffected pins the grant-path merge: a
// frequency-only LockState (MCC=0, as publishGrantFromMAC produces) on an
// already-identified lock publishes nothing and changes nothing — the
// identity-vote gate must not turn it into a pending change either.
func TestMaybeLockFrequencyOnlyLockUnaffected(t *testing.T) {
	cc, bus := debugCC(t, io.Discard, slog.LevelInfo)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	a := LockState{FrequencyHz: 412_000_000, MCC: 250, MNC: 13, LocationArea: 5}
	cc.maybeLock(a)
	drainLockEvents(sub)

	cc.maybeLock(LockState{FrequencyHz: 412_000_000})
	if got := drainLockEvents(sub); len(got) != 0 {
		t.Fatalf("frequency-only lock published: %v", got)
	}
	if lockStateOf(cc) != a {
		t.Fatalf("frequency-only lock changed the identity: %+v", lockStateOf(cc))
	}
}
