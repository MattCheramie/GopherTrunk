package trunking

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestObservedCallsSurfacesUnfollowedTalkgroups is the P25 "only one active
// talkgroup" regression: with a single voice tuner, three distinct talkgroups
// keying up at once must surface as three active calls — one followed (bound to
// the tuner) and two control-channel-observed (no tuner free) — so an operator
// sees every talkgroup up on the system, not just the one being decoded.
func TestObservedCallsSurfacesUnfollowedTalkgroups(t *testing.T) {
	e, _, bus, _ := mkEngine(t, 1) // ONE voice tuner
	defer bus.Close()

	// Three distinct talkgroups, all up on the same system at once.
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 100, FrequencyHz: 851_000_000})
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 200, FrequencyHz: 851_012_500})
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 300, FrequencyHz: 851_025_000})

	followed := e.ActiveCalls()
	if len(followed) != 1 {
		t.Fatalf("followed calls = %d, want 1 (only one voice tuner)", len(followed))
	}
	observed := e.ObservedCalls()
	if len(observed) != 2 {
		t.Fatalf("observed (unfollowed) calls = %d, want 2", len(observed))
	}

	// Union of followed + observed must be exactly the three talkgroups, with
	// no talkgroup appearing in both sets.
	seen := map[uint32]int{}
	for _, ac := range followed {
		seen[ac.Grant.GroupID]++
	}
	for _, ac := range observed {
		if ac.Device != nil {
			t.Errorf("observed call should not be bound to a device, got %q", ac.Device.Serial)
		}
		seen[ac.Grant.GroupID]++
	}
	for _, tg := range []uint32{100, 200, 300} {
		if seen[tg] != 1 {
			t.Errorf("talkgroup %d appeared %d times across followed+observed, want exactly 1", tg, seen[tg])
		}
	}
}

// TestObservedCallExpiresAfterTimeout pins that a control-channel-observed call
// ages out of ObservedCalls once the control channel stops repeating its grant,
// reusing the engine's existing call-timeout grace window.
func TestObservedCallExpiresAfterTimeout(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	pool, _ := mkPool(1)
	clk := &fakeClock{t: time.Unix(1_700_000_000, 0)}
	e, err := NewEngine(EngineOptions{
		Bus: bus, VoicePool: pool, Talkgroups: NewTalkgroupDB(),
		CallTimeout: time.Second, Now: clk.Now,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Fill the single tuner, then a second talkgroup is observed-only.
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 1, FrequencyHz: 851_000_000})
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 2, FrequencyHz: 851_012_500})
	if got := len(e.ObservedCalls()); got != 1 {
		t.Fatalf("observed = %d, want 1 before timeout", got)
	}

	// Past the grace window with no repeat grant, the watchdog drops it.
	clk.t = clk.t.Add(2 * time.Second)
	e.runWatchdog()
	if got := len(e.ObservedCalls()); got != 0 {
		t.Fatalf("observed = %d, want 0 after timeout", got)
	}
}
