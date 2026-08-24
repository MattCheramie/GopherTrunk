package trunking

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// windowedFakeTuner is a fakeVoiceTuner whose wideband IQ window covers only
// frequencies canTune reports true for — the virtual voice-tap shape that
// produces the issue #356 "every call is observed" symptom when the window
// misses the system's voice channels.
type windowedFakeTuner struct {
	fakeVoiceTuner
	canTune func(hz uint32) bool
}

func (w *windowedFakeTuner) CanTune(hz uint32) bool { return w.canTune(hz) }

// TestObservedCallReasonAllBusy pins that a call observed while every voice
// tuner is serving an equal-priority call carries the "all voice tuners busy"
// reason (issue #356): the diagnostic that distinguishes a busy system from a
// decode failure.
func TestObservedCallReasonAllBusy(t *testing.T) {
	e, _, bus, _ := mkEngine(t, 1) // ONE voice tuner
	defer bus.Close()

	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 100, FrequencyHz: 851_000_000})
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 200, FrequencyHz: 851_012_500})

	observed := e.ObservedCalls()
	if len(observed) != 1 {
		t.Fatalf("observed = %d, want 1", len(observed))
	}
	if got := observed[0].UnfollowedReason; got != UnfollowedAllBusy {
		t.Errorf("UnfollowedReason = %q, want %q", got, UnfollowedAllBusy)
	}
	// The followed call must NOT carry a reason.
	for _, ac := range e.ActiveCalls() {
		if ac.UnfollowedReason != "" {
			t.Errorf("followed call carries UnfollowedReason %q, want empty", ac.UnfollowedReason)
		}
	}
}

// TestObservedCallReasonNoVoiceSDR pins the zero-voice-device shape: trunking
// configured with no `role: voice` SDR at all.
func TestObservedCallReasonNoVoiceSDR(t *testing.T) {
	e, _, bus, _ := mkEngine(t, 0)
	defer bus.Close()

	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 100, FrequencyHz: 851_000_000})

	observed := e.ObservedCalls()
	if len(observed) != 1 {
		t.Fatalf("observed = %d, want 1", len(observed))
	}
	if got := observed[0].UnfollowedReason; got != UnfollowedNoVoiceSDR {
		t.Errorf("UnfollowedReason = %q, want %q", got, UnfollowedNoVoiceSDR)
	}
}

// TestObservedCallReasonCoverageGap pins the wideband-tap shape from the
// issue #356 report: a voice device exists but its IQ window excludes the
// grant's frequency, so the call can never be followed and must say so.
func TestObservedCallReasonCoverageGap(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	tap := &windowedFakeTuner{canTune: func(hz uint32) bool { return false }}
	pool := NewVoicePool([]*VoiceDevice{{Tuner: tap, Serial: "TAP-1"}})
	e, err := NewEngine(EngineOptions{
		Bus: bus, VoicePool: pool, Talkgroups: NewTalkgroupDB(),
		CallTimeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 100, FrequencyHz: 851_000_000})

	observed := e.ObservedCalls()
	if len(observed) != 1 {
		t.Fatalf("observed = %d, want 1", len(observed))
	}
	if got := observed[0].UnfollowedReason; got != UnfollowedNoCoverage {
		t.Errorf("UnfollowedReason = %q, want %q", got, UnfollowedNoCoverage)
	}
}

// TestUnfollowedReasonClearedWhenTunerFollows pins the clear-on-follow rule:
// once a tuner frees up and a repeat grant binds the previously-observed call,
// the stale reason must not survive on the tracker entry.
func TestUnfollowedReasonClearedWhenTunerFollows(t *testing.T) {
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

	// TG 1 takes the only tuner; TG 2 is observed with the busy reason.
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 1, FrequencyHz: 851_000_000, At: clk.t})
	clk.t = clk.t.Add(800 * time.Millisecond)
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 2, FrequencyHz: 851_012_500, At: clk.t})
	if obs := e.ObservedCalls(); len(obs) != 1 || obs[0].UnfollowedReason != UnfollowedAllBusy {
		t.Fatalf("observed = %+v, want one entry with the all-busy reason", obs)
	}

	// TG 1 goes stale and the watchdog frees the tuner; TG 2's tracker entry
	// (refreshed 800 ms in) survives the same sweep.
	clk.t = clk.t.Add(1200 * time.Millisecond)
	e.runWatchdog()
	if got := len(e.ActiveCalls()); got != 0 {
		t.Fatalf("active after watchdog = %d, want 0", got)
	}

	// The repeat grant now binds — the call is followed and the reason clears.
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 2, FrequencyHz: 851_012_500, At: clk.t})
	if got := len(e.ActiveCalls()); got != 1 {
		t.Fatalf("active = %d, want 1 (tuner freed)", got)
	}
	if obs := e.ObservedCalls(); len(obs) != 0 {
		t.Fatalf("observed = %+v, want none (call is followed)", obs)
	}
	e.mu.Lock()
	ac := e.observed[observedKey(Grant{System: "X", GroupID: 2})]
	e.mu.Unlock()
	if ac == nil {
		t.Fatal("tracker entry for TG 2 vanished")
	}
	if ac.UnfollowedReason != "" {
		t.Errorf("UnfollowedReason = %q after follow, want empty", ac.UnfollowedReason)
	}
}
