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

// TestObservedCallRecordsBusyReason pins that a call left observed because the
// only voice tuner is busy with an equal-priority call carries the "tuners
// busy" reason — so the operator can tell it apart from a decode failure
// (issue #356). Fails first: before the fix Unfollowed is always "".
func TestObservedCallRecordsBusyReason(t *testing.T) {
	e, _, bus, _ := mkEngine(t, 1) // one physical voice tuner
	defer bus.Close()

	// First grant binds the only tuner; a second, equal-priority grant on a
	// different frequency can't preempt it, so it surfaces as observed.
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 100, FrequencyHz: 851_000_000})
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 200, FrequencyHz: 851_012_500})

	obs := e.ObservedCalls()
	if len(obs) != 1 {
		t.Fatalf("observed = %d, want 1", len(obs))
	}
	if obs[0].Unfollowed != unfollowedTunersBusy {
		t.Errorf("unfollowed reason = %q, want %q", obs[0].Unfollowed, unfollowedTunersBusy)
	}
}

// TestObservedCallRecordsCoverageReason is the RCCRAFT1 / single-wideband-SDR
// case (issue #356): a voice grant whose frequency is outside the only voice
// device's tuning window can't be followed, so it surfaces as observed with
// the coverage reason — NOT as "the talkgroup has no voice". A wider front end
// (e.g. SDRTrunk) would decode the same call.
func TestObservedCallRecordsCoverageReason(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	tap := &constrainedTuner{lo: 851_000_000, hi: 852_000_000}
	pool := NewVoicePool([]*VoiceDevice{{Tuner: tap, Serial: "wb:dev:tap-0"}})
	e, err := NewEngine(EngineOptions{
		Bus: bus, VoicePool: pool, Talkgroups: NewTalkgroupDB(),
	})
	if err != nil {
		t.Fatal(err)
	}

	// 900 MHz is outside the tap's [851, 852] MHz window.
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 1, FrequencyHz: 900_000_000})

	obs := e.ObservedCalls()
	if len(obs) != 1 {
		t.Fatalf("observed = %d, want 1", len(obs))
	}
	if obs[0].Device != nil {
		t.Errorf("observed call must not be bound to a device, got %q", obs[0].Device.Serial)
	}
	if obs[0].Unfollowed != unfollowedNoCoverage {
		t.Errorf("unfollowed reason = %q, want %q", obs[0].Unfollowed, unfollowedNoCoverage)
	}
}

// TestObservedCallRecordsNoSDRReason pins the "trunking configured but no
// voice device attached" case: every grant is observed-only and carries the
// no-voice-SDR reason (issue #356).
func TestObservedCallRecordsNoSDRReason(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	pool, _ := mkPool(0) // valid, empty voice pool
	e, err := NewEngine(EngineOptions{
		Bus: bus, VoicePool: pool, Talkgroups: NewTalkgroupDB(),
	})
	if err != nil {
		t.Fatal(err)
	}

	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 1, FrequencyHz: 851_000_000})

	obs := e.ObservedCalls()
	if len(obs) != 1 {
		t.Fatalf("observed = %d, want 1", len(obs))
	}
	if obs[0].Unfollowed != unfollowedNoVoiceSDR {
		t.Errorf("unfollowed reason = %q, want %q", obs[0].Unfollowed, unfollowedNoVoiceSDR)
	}
}

// TestFollowedCallClearsObservedReason pins that once a tuner frees and follows
// a previously-observed call, the unfollowed reason is cleared rather than
// lingering stale.
func TestFollowedCallClearsObservedReason(t *testing.T) {
	e, _, bus, _ := mkEngine(t, 1)
	defer bus.Close()

	// Fill the tuner, then observe a second call (tuners busy).
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 100, FrequencyHz: 851_000_000})
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 200, FrequencyHz: 851_012_500})
	if obs := e.ObservedCalls(); len(obs) != 1 || obs[0].Unfollowed != unfollowedTunersBusy {
		t.Fatalf("precondition: want one observed call with busy reason, got %+v", obs)
	}

	// End the first call so the tuner frees, then re-grant the observed one:
	// it should now be followed, with no lingering reason. (Ending group 100
	// leaves its own observed-tracker entry until the watchdog reaps it, so we
	// assert on group 200 specifically rather than the total observed count.)
	for _, ac := range e.ActiveCalls() {
		e.endCall(ac, EndReasonNormal)
	}
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 200, FrequencyHz: 851_012_500})

	for _, ac := range e.ObservedCalls() {
		if ac.Grant.GroupID == 200 {
			t.Fatalf("group 200 still observed after a tuner freed and followed it: reason=%q", ac.Unfollowed)
		}
	}
	var followed *ActiveCall
	for _, ac := range e.ActiveCalls() {
		if ac.Grant.GroupID == 200 {
			followed = ac
		}
	}
	if followed == nil {
		t.Fatalf("group 200 not followed after tuner freed; active = %+v", e.ActiveCalls())
	}
	if followed.Unfollowed != "" {
		t.Errorf("followed call carries stale reason %q, want empty", followed.Unfollowed)
	}
}
