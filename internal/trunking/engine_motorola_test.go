package trunking

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// Issue #1408: a Motorola Type II analog call was never ended. The voice chain
// touches the engine on every chunk of audio that passes squelch, and once the
// system hands the voice channel to another talkgroup that audio keeps coming,
// so the inactivity watchdog never fired. The reporter's log (one voice SDR,
// call_timeout_ms 5000 + voice_hangtime_ms 1500): TG 38416 granted on
// 856.2625 MHz, the same channel then granted to TG 33968, 34000 and 37424,
// every one of those dropped as "no voice device available", and the 38416 call
// held for minutes.

const (
	mot856 = 856_262_500
	mot857 = 857_237_500
)

func motGrant(tg, src uint32, hz uint32) Grant {
	return Grant{System: "MSP", Protocol: "motorola", GroupID: tg, SourceID: src, FrequencyHz: hz}
}

// newMotorolaEngine builds the reporter's shape: one voice SDR, a 6.5 s call
// watchdog, and a fake clock.
func newMotorolaEngine(t *testing.T) (*Engine, *fakeClock, *events.Subscription) {
	t.Helper()
	bus := events.NewBus(256)
	t.Cleanup(bus.Close)
	pool, _ := mkPool(1)
	sub := bus.Subscribe()
	t.Cleanup(sub.Close)
	clock := &fakeClock{t: time.Unix(1_000_000, 0)}
	e, err := NewEngine(EngineOptions{
		Bus:         bus,
		VoicePool:   pool,
		Talkgroups:  NewTalkgroupDB(),
		CallTimeout: 6500 * time.Millisecond,
		Now:         clock.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	return e, clock, sub
}

// analogAudio advances the clock by d in 200 ms steps, touching the call on
// serial each step (the analog chain's heartbeat while the channel carries
// audio) and running the watchdog.
func analogAudio(e *Engine, clock *fakeClock, serial string, d time.Duration) {
	for el := time.Duration(0); el < d; el += 200 * time.Millisecond {
		clock.t = clock.t.Add(200 * time.Millisecond)
		e.Touch(serial)
		e.runWatchdog()
	}
}

func callFor(e *Engine, tg uint32) *ActiveCall {
	for _, ac := range e.ActiveCalls() {
		if ac.Grant.GroupID == tg {
			return ac
		}
	}
	return nil
}

func countCallEnds(sub *events.Subscription) (n int, last CallEnd) {
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCallEnd {
				n++
				last = ev.Payload.(CallEnd)
			}
		default:
			return n, last
		}
	}
}

// The voice channel granted to another talkgroup ends the call it carried, and
// the new talkgroup gets the voice SDR. Fails on the old engine: the 33968
// grant found no free device and could not preempt an equal-priority call.
func TestMotorolaChannelReassignedEndsCall(t *testing.T) {
	e, clock, sub := newMotorolaEngine(t)
	e.HandleGrant(motGrant(38416, 1001, mot856))
	old := callFor(e, 38416)
	if old == nil {
		t.Fatal("38416 call did not start")
	}
	analogAudio(e, clock, old.Device.Serial, time.Second)

	e.HandleGrant(motGrant(33968, 2002, mot856))

	if callFor(e, 38416) != nil {
		t.Fatal("38416 call still open after its channel was granted to 33968")
	}
	if callFor(e, 33968) == nil {
		t.Fatal("33968 did not get the voice SDR")
	}
	n, ce := countCallEnds(sub)
	if n != 1 || ce.Grant.GroupID != 38416 || ce.Reason != EndReasonNormal {
		t.Fatalf("call ends = %d, last %+v; want one normal end of 38416", n, ce)
	}
}

// A call whose grants stop is ended within grantLivenessTimeout even while the
// analog chain keeps touching it. Fails on the old engine: the call stays open
// for the whole 60 s of audio.
func TestMotorolaCallEndsWhenGrantsStop(t *testing.T) {
	e, clock, sub := newMotorolaEngine(t)
	e.HandleGrant(motGrant(38416, 1001, mot856))
	ac := callFor(e, 38416)
	if ac == nil {
		t.Fatal("call did not start")
	}
	serial := ac.Device.Serial
	// The control channel repeats the update while the call runs.
	for i := 0; i < 10; i++ {
		analogAudio(e, clock, serial, time.Second)
		e.HandleGrant(motGrant(38416, 0, mot856))
	}
	if callFor(e, 38416) == nil {
		t.Fatal("call ended while its grants were still being repeated")
	}
	lastGrant := clock.t

	// Grants stop; the channel keeps carrying audio for a minute.
	for el := time.Duration(0); el < time.Minute && callFor(e, 38416) != nil; el += 200 * time.Millisecond {
		clock.t = clock.t.Add(200 * time.Millisecond)
		e.Touch(serial)
		e.runWatchdog()
	}
	if callFor(e, 38416) != nil {
		t.Fatal("call still open a minute after its grants stopped")
	}
	if held := clock.t.Sub(lastGrant); held > grantLivenessTimeout+time.Second {
		t.Fatalf("call held %v after its last grant, want <= %v", held, grantLivenessTimeout+time.Second)
	}
	if n, ce := countCallEnds(sub); n != 1 || ce.Reason != EndReasonNormal {
		t.Fatalf("call ends = %d, last reason %v; want one normal end", n, ce.Reason)
	}
}

// A grant for the same talkgroup after its grants had stopped starts a new
// call instead of following the stale one (the 22:47:00 "call followed to new
// frequency" line). Driven through HandleGrant alone, without the watchdog, so
// it does not depend on a watchdog tick landing in between. Fails on the old
// engine: the call is retuned and keeps its 2.5-minute-old StartedAt.
func TestMotorolaGrantAfterGapStartsNewCall(t *testing.T) {
	e, clock, sub := newMotorolaEngine(t)
	e.HandleGrant(motGrant(38416, 1001, mot856))
	first := callFor(e, 38416)
	if first == nil {
		t.Fatal("call did not start")
	}
	for i := 0; i < 150; i++ { // 30 s of audio, no grants, no watchdog
		clock.t = clock.t.Add(200 * time.Millisecond)
		e.Touch(first.Device.Serial)
	}

	e.HandleGrant(motGrant(38416, 1001, mot857))

	second := callFor(e, 38416)
	if second == nil {
		t.Fatal("no call for 38416 after the new grant")
	}
	if second.Grant.CallID == first.Grant.CallID || !second.StartedAt.Equal(clock.t) {
		t.Fatalf("new grant extended the old call (call id %d -> %d, started %v)",
			first.Grant.CallID, second.Grant.CallID, second.StartedAt)
	}
	if second.Grant.FrequencyHz != mot857 {
		t.Fatalf("new call on %d Hz, want %d", second.Grant.FrequencyHz, mot857)
	}
	if n, _ := countCallEnds(sub); n != 1 {
		t.Fatalf("call ends = %d, want 1 (the stale call)", n)
	}
}

// No harm: a Motorola call moved to a new channel while its grants are fresh
// is still followed as one call.
func TestMotorolaFreshCallStillFollowsNewFrequency(t *testing.T) {
	e, clock, _ := newMotorolaEngine(t)
	e.HandleGrant(motGrant(38416, 1001, mot856))
	first := callFor(e, 38416)
	analogAudio(e, clock, first.Device.Serial, time.Second)
	e.HandleGrant(motGrant(38416, 0, mot857))
	ac := callFor(e, 38416)
	if ac == nil || ac.Grant.CallID != first.Grant.CallID || ac.Grant.FrequencyHz != mot857 {
		t.Fatalf("fresh call not followed: %+v", ac)
	}
}

// No harm: P25 keeps its #915 behaviour. A grant for another talkgroup on an
// active call's channel neither ends that call nor takes its device, and a P25
// call touched by its voice chain is not ended by the grant liveness rule.
func TestP25UnaffectedByGrantLiveness(t *testing.T) {
	e, clock, sub := newMotorolaEngine(t)
	p25 := func(tg, src uint32) Grant {
		return Grant{System: "P25", Protocol: "p25", GroupID: tg, SourceID: src, FrequencyHz: 851_000_000}
	}
	e.HandleGrant(p25(100, 0))
	ac := callFor(e, 100)
	if ac == nil {
		t.Fatal("call did not start")
	}
	e.HandleGrant(p25(200, 7))
	analogAudio(e, clock, ac.Device.Serial, 20*time.Second)
	if callFor(e, 100) == nil {
		t.Fatal("P25 call ended by the Motorola rules")
	}
	if n, _ := countCallEnds(sub); n != 0 {
		t.Fatalf("P25 call ends = %d, want 0", n)
	}
}
