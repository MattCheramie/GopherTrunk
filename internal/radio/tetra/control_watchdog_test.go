package tetra

import (
	"log/slog"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestControlChannelStaleLockPublishesLost is the regression for the flapping
// "CC hunt failed · candidates exhausted" seen while a TETRA control channel is
// still decoding: TETRA's maybeLock is edge-triggered, so a steady lock never
// re-affirms and the cchunt supervisor could never leave StateLocked. CheckStale
// is the watchdog that turns a genuinely silent carrier into a cc.lost so the
// supervisor re-hunts. This asserts: a fresh lock does NOT publish cc.lost, and a
// lock that has decoded nothing for longer than the timeout DOES.
func TestControlChannelStaleLockPublishesLost(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	now := time.Unix(1_000, 0)
	cc := New(Options{
		Bus:         bus,
		Log:         slog.Default(),
		SystemName:  "Sys",
		FrequencyHz: 467_913_000,
		Now:         func() time.Time { return now },
	})

	// Lock the control channel. maybeLock stamps the activity heartbeat.
	cc.maybeLock(LockState{FrequencyHz: 467_913_000})
	if !cc.Locked() {
		t.Fatal("control channel did not lock")
	}

	countLost := func() int {
		n := 0
		for {
			select {
			case ev := <-sub.C:
				if ev.Kind == events.KindCCLost {
					if lp, ok := ev.Payload.(trunking.LockedPayload); ok && lp.LockedFrequencyHz() == 467_913_000 {
						n++
					}
				}
			default:
				return n
			}
		}
	}
	countLost() // drain the cc.locked (and anything else) so far

	// Fresh activity (3 s < 5 s timeout): no cc.lost.
	now = now.Add(3 * time.Second)
	cc.CheckStale(now, 5*time.Second)
	if got := countLost(); got != 0 {
		t.Errorf("cc.lost published %d times while activity is fresh, want 0", got)
	}
	if !cc.Locked() {
		t.Error("control channel unlocked while activity was still fresh")
	}

	// A further sync burst refreshes the heartbeat; the clock advancing past the
	// timeout relative to the ORIGINAL lock must not trip it once refreshed.
	cc.noteActivity()              // simulates a decodeSB heartbeat at t=1003
	now = now.Add(4 * time.Second) // t=1007, 4 s since the refresh
	cc.CheckStale(now, 5*time.Second)
	if got := countLost(); got != 0 {
		t.Errorf("cc.lost published %d times after a heartbeat refresh, want 0", got)
	}

	// Now go silent past the timeout: cc.lost fires and the lock drops.
	now = now.Add(6 * time.Second) // 6 s since the last heartbeat
	cc.CheckStale(now, 5*time.Second)
	if got := countLost(); got != 1 {
		t.Errorf("cc.lost published %d times after going silent, want 1", got)
	}
	if cc.Locked() {
		t.Error("control channel still locked after the staleness watchdog fired")
	}
}

// TestControlChannelCheckStaleNoopBeforeFirstDecode guards the startup case:
// CheckStale must do nothing until the channel has actually decoded something
// (lastActivity==0), so a freshly-constructed channel is never declared lost.
func TestControlChannelCheckStaleNoopBeforeFirstDecode(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	now := time.Unix(2_000, 0)
	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys", FrequencyHz: 1, Now: func() time.Time { return now }})

	now = now.Add(time.Hour)
	cc.CheckStale(now, 5*time.Second)

	select {
	case ev := <-sub.C:
		if ev.Kind == events.KindCCLost {
			t.Error("cc.lost published before the channel ever decoded anything")
		}
	default:
	}
}
