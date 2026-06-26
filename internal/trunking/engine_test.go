package trunking

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

type testBusCollector struct {
	mu sync.Mutex
	xs []events.Event
}

func (c *testBusCollector) drain(sub *events.Subscription, until int, deadline time.Duration) []events.Event {
	t := time.NewTimer(deadline)
	defer t.Stop()
	for {
		c.mu.Lock()
		n := len(c.xs)
		c.mu.Unlock()
		if n >= until {
			break
		}
		select {
		case ev, ok := <-sub.C:
			if !ok {
				break
			}
			c.mu.Lock()
			c.xs = append(c.xs, ev)
			c.mu.Unlock()
		case <-t.C:
			return c.snapshot()
		}
	}
	return c.snapshot()
}

func (c *testBusCollector) snapshot() []events.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]events.Event, len(c.xs))
	copy(out, c.xs)
	return out
}

func mkEngine(t *testing.T, devices int) (*Engine, *VoicePool, *events.Bus, []*fakeVoiceTuner) {
	t.Helper()
	bus := events.NewBus(32)
	pool, tuners := mkPool(devices)
	e, err := NewEngine(EngineOptions{
		Bus:         bus,
		VoicePool:   pool,
		Talkgroups:  NewTalkgroupDB(),
		CallTimeout: 200 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	return e, pool, bus, tuners
}

func TestEngineStartsCallOnGrant(t *testing.T) {
	e, _, bus, tuners := mkEngine(t, 1)
	defer bus.Close()

	sub := bus.Subscribe()
	defer sub.Close()
	collector := &testBusCollector{}
	go func() { collector.drain(sub, 1, 500*time.Millisecond) }()

	g := Grant{System: "X", Protocol: "p25", GroupID: 100, FrequencyHz: 851_000_000}
	e.HandleGrant(g)

	evs := collector.drain(sub, 1, 500*time.Millisecond)
	if len(evs) == 0 || evs[0].Kind != events.KindCallStart {
		t.Fatalf("events = %+v", evs)
	}
	cs, ok := evs[0].Payload.(CallStart)
	if !ok || cs.Grant.GroupID != 100 || cs.DeviceSerial != "A-voice" {
		t.Errorf("CallStart payload = %+v", evs[0].Payload)
	}
	if got := tuners[0].tuned(); len(got) != 1 || got[0] != 851_000_000 {
		t.Errorf("device retune = %v", got)
	}
}

// TestEngineDiscoversTalkgroupFromGrant covers the control-channel talkgroup
// autofill: a group grant for an unknown TG catalogues a placeholder so the
// Talkgroups database fills in from the air, while an individual (unit-to-unit)
// grant is skipped so a 24-bit subscriber address never lands as a phantom TG.
func TestEngineDiscoversTalkgroupFromGrant(t *testing.T) {
	e, _, bus, _ := mkEngine(t, 1)
	defer bus.Close()

	if e.talkgroups.Lookup(555) != nil {
		t.Fatal("precondition: TG 555 should be unknown")
	}
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 555, FrequencyHz: 851_000_000})
	tg := e.talkgroups.Lookup(555)
	if tg == nil {
		t.Fatalf("group grant should have catalogued TG 555")
	}
	if tg.Tag != "Discovered" || !tg.Scan {
		t.Errorf("discovered TG = %+v, want Tag=Discovered Scan=true (ScanModeAll)", tg)
	}

	// Individual (unit-to-unit) grant: must NOT be catalogued.
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 140957, FrequencyHz: 852_000_000, Individual: true})
	if e.talkgroups.Lookup(140957) != nil {
		t.Errorf("individual grant should not be catalogued as a talkgroup")
	}

	// A discovered TG isn't auto-scanned in ScanModeList, so cataloguing never
	// silently widens a curated scan list.
	e.SetScanMode(ScanModeList)
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 556, FrequencyHz: 853_000_000})
	if tg := e.talkgroups.Lookup(556); tg == nil {
		t.Errorf("group grant should catalogue TG 556 even in ScanModeList")
	} else if tg.Scan {
		t.Errorf("discovered TG in ScanModeList should have Scan=false, got %+v", tg)
	}
}

// TestEngineSkipsOutOfBandVirtualTunerInFavorOfPhysical covers the
// Phase B fallback: when a wideband virtual voice tuner can't serve
// a grant (frequency outside its IQ window) and a physical voice
// SDR is also free, the engine binds the physical one instead of
// dropping. Exercises voicepool.FindFreeForFrequency end-to-end.
func TestEngineSkipsOutOfBandVirtualTunerInFavorOfPhysical(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	// Wideband tap listed first (preferred for in-band grants); a
	// physical tuner second (catches the spillover).
	virtual := &constrainedTuner{lo: 851_000_000, hi: 852_000_000}
	physical := &fakeVoiceTuner{}
	pool := NewVoicePool([]*VoiceDevice{
		{Tuner: virtual, Serial: "wb:00000003:tap-0"},
		{Tuner: physical, Serial: "phys-voice"},
	})
	e, err := NewEngine(EngineOptions{
		Bus: bus, VoicePool: pool, Talkgroups: NewTalkgroupDB(),
		CallTimeout: 1 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Out-of-window grant (900 MHz). Should land on the physical
	// tuner without erroring.
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 1, FrequencyHz: 900_000_000})

	calls := e.ActiveCalls()
	if len(calls) != 1 || calls[0].Device.Serial != "phys-voice" {
		t.Fatalf("expected one call on phys-voice, got %+v", calls)
	}
	if len(virtual.freqs) != 0 {
		t.Errorf("virtual tuner should not have been retuned, got %v", virtual.freqs)
	}
	if len(physical.tuned()) != 1 || physical.tuned()[0] != 900_000_000 {
		t.Errorf("physical tuner = %v, want [900_000_000]", physical.tuned())
	}
}

func TestEngineTalkgroupForDevice(t *testing.T) {
	e, _, bus, _ := mkEngine(t, 1)
	defer bus.Close()
	e.talkgroups.Add(&TalkGroup{ID: 100, AlphaTag: "OPS", Mute: true})

	if e.TalkgroupForDevice("A-voice") != nil {
		t.Fatal("no call active yet — TalkgroupForDevice should be nil")
	}
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 100, FrequencyHz: 851_000_000})

	tg := e.TalkgroupForDevice("A-voice")
	if tg == nil || tg.ID != 100 || !tg.Mute {
		t.Fatalf("TalkgroupForDevice = %+v, want the muted OPS talkgroup", tg)
	}
	if e.TalkgroupForDevice("no-such-device") != nil {
		t.Error("unknown device should return nil")
	}
}

func TestEngineDropsLockedOutGrant(t *testing.T) {
	e, _, bus, tuners := mkEngine(t, 1)
	defer bus.Close()
	e.talkgroups.Add(&TalkGroup{ID: 50, AlphaTag: "BLOCKED", Lockout: true})

	sub := bus.Subscribe()
	defer sub.Close()

	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 50, FrequencyHz: 1_000_000})

	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event for locked-out grant: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
	if got := tuners[0].tuned(); len(got) != 0 {
		t.Errorf("locked-out grant should not retune; got %v", got)
	}
}

func TestEngineScanModeListDropsTGOutsideList(t *testing.T) {
	e, _, bus, tuners := mkEngine(t, 1)
	defer bus.Close()
	e.SetScanMode(ScanModeList)
	// TG with Scan=false should be dropped.
	e.talkgroups.Add(&TalkGroup{ID: 77, AlphaTag: "OFF-LIST", Scan: false})

	sub := bus.Subscribe()
	defer sub.Close()
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 77, FrequencyHz: 1_000_000})

	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event for off-list grant: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
	if got := tuners[0].tuned(); len(got) != 0 {
		t.Errorf("off-list grant should not retune; got %v", got)
	}
}

func TestEngineScanModeListAllowsScanTrueTG(t *testing.T) {
	e, _, bus, _ := mkEngine(t, 1)
	defer bus.Close()
	e.SetScanMode(ScanModeList)
	e.talkgroups.Add(&TalkGroup{ID: 78, AlphaTag: "ON-LIST", Scan: true})

	sub := bus.Subscribe()
	defer sub.Close()
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 78, FrequencyHz: 1_000_000})

	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCallStart {
				return
			}
		case <-deadline:
			t.Fatal("no CallStart published for on-list grant")
		}
	}
}

func TestEngineScanModeListBypassedByEmergency(t *testing.T) {
	e, _, bus, _ := mkEngine(t, 1)
	defer bus.Close()
	e.SetScanMode(ScanModeList)
	// Off-list (Scan=false) but Emergency — must still fire.
	e.talkgroups.Add(&TalkGroup{ID: 79, AlphaTag: "OFF-LIST-EMER", Scan: false})

	sub := bus.Subscribe()
	defer sub.Close()
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 79, FrequencyHz: 1_000_000, Emergency: true})

	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCallStart {
				return
			}
		case <-deadline:
			t.Fatal("Emergency grant should bypass scan-list gate")
		}
	}
}

func TestEngineScanModeListUnknownTGDropped(t *testing.T) {
	// TG not in the DB at all (Lookup returns nil) → dropped.
	e, _, bus, _ := mkEngine(t, 1)
	defer bus.Close()
	e.SetScanMode(ScanModeList)

	sub := bus.Subscribe()
	defer sub.Close()
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 999, FrequencyHz: 1_000_000})

	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event for unknown TG: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestEngineScanModeAllIgnoresScanFlag(t *testing.T) {
	// Default mode (all): a TG with Scan=false still fires.
	e, _, bus, _ := mkEngine(t, 1)
	defer bus.Close()
	e.talkgroups.Add(&TalkGroup{ID: 80, AlphaTag: "OFF-LIST", Scan: false})

	sub := bus.Subscribe()
	defer sub.Close()
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 80, FrequencyHz: 1_000_000})

	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCallStart {
				return
			}
		case <-deadline:
			t.Fatal("ScanModeAll should ignore Scan flag")
		}
	}
}

func TestEngineDropsZeroFrequencyGrant(t *testing.T) {
	e, _, bus, _ := mkEngine(t, 1)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	e.HandleGrant(Grant{System: "X", GroupID: 100, FrequencyHz: 0})

	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestEnginePreemptsLowerPriority(t *testing.T) {
	e, _, bus, _ := mkEngine(t, 1)
	defer bus.Close()
	e.talkgroups.Add(&TalkGroup{ID: 100, Priority: 5, AlphaTag: "LOW"})
	e.talkgroups.Add(&TalkGroup{ID: 200, Priority: 1, AlphaTag: "HIGH"})

	sub := bus.Subscribe()
	defer sub.Close()
	collector := &testBusCollector{}

	low := Grant{System: "X", Protocol: "p25", GroupID: 100, FrequencyHz: 851_000_000}
	high := Grant{System: "X", Protocol: "p25", GroupID: 200, FrequencyHz: 852_000_000}

	// HandleGrant publishes synchronously (Bus.Publish is buffered at 32
	// here), so events land on the subscription channel before we drain.
	e.HandleGrant(low)
	e.HandleGrant(high) // should preempt

	evs := collector.drain(sub, 3, 1*time.Second)
	if len(evs) < 3 {
		t.Fatalf("got %d events, want >= 3 (start, end, start)", len(evs))
	}
	if evs[0].Kind != events.KindCallStart {
		t.Errorf("evs[0] = %s, want call.start", evs[0].Kind)
	}
	if evs[1].Kind != events.KindCallEnd {
		t.Errorf("evs[1] = %s, want call.end", evs[1].Kind)
	} else if ce, ok := evs[1].Payload.(CallEnd); ok && ce.Reason != EndReasonPreempted {
		t.Errorf("CallEnd reason = %s, want preempted", ce.Reason)
	}
	if evs[2].Kind != events.KindCallStart {
		t.Errorf("evs[2] = %s, want call.start", evs[2].Kind)
	} else if cs, ok := evs[2].Payload.(CallStart); ok && cs.Grant.GroupID != 200 {
		t.Errorf("preempting grant.GroupID = %d, want 200", cs.Grant.GroupID)
	}
}

func TestEngineDoesNotPreemptEqualPriority(t *testing.T) {
	e, _, bus, _ := mkEngine(t, 1)
	defer bus.Close()
	e.talkgroups.Add(&TalkGroup{ID: 100, Priority: 3})
	e.talkgroups.Add(&TalkGroup{ID: 200, Priority: 3})

	g1 := Grant{GroupID: 100, FrequencyHz: 851_000_000}
	g2 := Grant{GroupID: 200, FrequencyHz: 852_000_000}
	e.HandleGrant(g1)
	e.HandleGrant(g2) // should be dropped

	if calls := e.ActiveCalls(); len(calls) != 1 || calls[0].Grant.GroupID != 100 {
		t.Errorf("active = %+v, want only original call", calls)
	}
}

func TestEngineEmergencyPreemptsAnything(t *testing.T) {
	e, _, bus, _ := mkEngine(t, 1)
	defer bus.Close()
	e.talkgroups.Add(&TalkGroup{ID: 100, Priority: 1, AlphaTag: "TOP"})

	g1 := Grant{GroupID: 100, FrequencyHz: 851_000_000}
	emer := Grant{GroupID: 999, FrequencyHz: 853_000_000, Emergency: true}
	e.HandleGrant(g1)
	e.HandleGrant(emer)

	calls := e.ActiveCalls()
	if len(calls) != 1 || calls[0].Grant.GroupID != 999 {
		t.Errorf("after emergency: active = %+v, want emergency call", calls)
	}
}

func TestEngineExplicitEndCall(t *testing.T) {
	e, pool, bus, _ := mkEngine(t, 1)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	g := Grant{GroupID: 100, FrequencyHz: 851_000_000}
	e.HandleGrant(g)
	d := pool.Active()[0].Device

	if !e.EndCall(d.Serial, EndReasonNormal) {
		t.Fatal("EndCall reported no active call")
	}
	if got := e.ActiveCalls(); len(got) != 0 {
		t.Errorf("active after EndCall = %+v, want []", got)
	}

	// drain the call.start + call.end events
	got := []events.Event{}
	deadline := time.After(500 * time.Millisecond)
loop:
	for {
		select {
		case ev := <-sub.C:
			got = append(got, ev)
			if ev.Kind == events.KindCallEnd {
				break loop
			}
		case <-deadline:
			break loop
		}
	}
	if len(got) < 2 {
		t.Fatalf("events = %+v", got)
	}
	end, ok := got[len(got)-1].Payload.(CallEnd)
	if !ok || end.Reason != EndReasonNormal {
		t.Errorf("last event reason = %v", got[len(got)-1])
	}
}

// TestEngineWatchdogTimesOutSilentCall covers the true silent-decode
// failure mode: a call that never received any frames (LastHeardAt
// stayed at StartedAt) must reap as EndReasonTimeout — the
// operator-actionable signal that the chain is broken.
func TestEngineWatchdogTimesOutSilentCall(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	pool, _ := mkPool(1)
	sub := bus.Subscribe()
	defer sub.Close()
	now := time.Unix(1000, 0)
	clock := &fakeClock{t: now}
	e, _ := NewEngine(EngineOptions{
		Bus:         bus,
		VoicePool:   pool,
		Talkgroups:  NewTalkgroupDB(),
		CallTimeout: 1 * time.Second,
		Now:         clock.Now,
	})

	g := Grant{GroupID: 100, FrequencyHz: 851_000_000}
	e.HandleGrant(g)
	if len(e.ActiveCalls()) != 1 {
		t.Fatal("call did not start")
	}

	// Advance the clock past the timeout and run the watchdog.
	clock.t = clock.t.Add(2 * time.Second)
	e.runWatchdog()
	if got := e.ActiveCalls(); len(got) != 0 {
		t.Errorf("watchdog should have ended the call; active = %+v", got)
	}

	// Reason must be EndReasonTimeout — no Touch ever fired, so
	// LastHeardAt is still equal to StartedAt.
	end := drainCallEnd(t, sub, 500*time.Millisecond)
	if end.Reason != EndReasonTimeout {
		t.Errorf("CallEnd.Reason = %v, want EndReasonTimeout (no frames decoded)", end.Reason)
	}
}

// TestEngineWatchdogReapsTouchedCallAsNormal covers the carrier-drop
// natural-end case: a call that received at least one Touch (an LDU
// was decoded) but then went quiet must reap as EndReasonNormal, not
// EndReasonTimeout. P25 trunking has no explicit channel-release on
// the CC for most calls, so this watchdog path IS how a healthy call
// ends — the operator-visible "timeout" label was a terminology bug
// that made v2maldo's field report read as a decode failure when the
// decode was actually working (issue #356 follow-up).
func TestEngineWatchdogReapsTouchedCallAsNormal(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	pool, _ := mkPool(1)
	sub := bus.Subscribe()
	defer sub.Close()
	clock := &fakeClock{t: time.Unix(1000, 0)}
	e, _ := NewEngine(EngineOptions{
		Bus:         bus,
		VoicePool:   pool,
		Talkgroups:  NewTalkgroupDB(),
		CallTimeout: 1 * time.Second,
		Now:         clock.Now,
	})

	e.HandleGrant(Grant{GroupID: 200, FrequencyHz: 852_000_000})
	if len(e.ActiveCalls()) != 1 {
		t.Fatal("call did not start")
	}
	serial := e.ActiveCalls()[0].Device.Serial

	// Simulate the composer delivering frames: advance the clock,
	// fire Touch (mirrors what runP25Phase1VoiceChain does when the
	// receiver's LDU sink fires). Then advance past the grace window
	// without any further Touch and run the watchdog.
	clock.t = clock.t.Add(100 * time.Millisecond)
	e.Touch(serial)
	clock.t = clock.t.Add(2 * time.Second)
	e.runWatchdog()

	if got := e.ActiveCalls(); len(got) != 0 {
		t.Errorf("watchdog should have ended the call; active = %+v", got)
	}
	end := drainCallEnd(t, sub, 500*time.Millisecond)
	if end.Reason != EndReasonNormal {
		t.Errorf("CallEnd.Reason = %v, want EndReasonNormal (frames received then carrier dropped)", end.Reason)
	}
}

// drainCallEnd pulls events off sub until it sees a CallEnd or the
// deadline fires. Tests use it to assert the reason published with a
// natural-end / silent-timeout reap.
func drainCallEnd(t *testing.T, sub *events.Subscription, deadline time.Duration) CallEnd {
	t.Helper()
	timer := time.NewTimer(deadline)
	defer timer.Stop()
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCallEnd {
				ce, ok := ev.Payload.(CallEnd)
				if !ok {
					t.Fatalf("CallEnd payload type = %T", ev.Payload)
				}
				return ce
			}
		case <-timer.C:
			t.Fatalf("no CallEnd event within %v", deadline)
		}
	}
}

// TestEngineHandleGrantDeduplicatesDuplicateGrant covers the issue
// #356 follow-up where the Phase 1 CC repeats voice-grant TSBKs while
// the call is active. Before the dedup, the engine bound a second
// voice SDR to the same TG/freq, producing duplicate WAV files and
// tying up a tuner the operator needed for other grants. After the
// fix the second grant refreshes the existing bind's LastHeardAt
// (treating the repeat as the CC re-asserting "this call is still
// going") and does not allocate a second device.
func TestEngineHandleGrantDeduplicatesDuplicateGrant(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	pool, tuners := mkPool(2)
	clock := &fakeClock{t: time.Unix(1000, 0)}
	e, _ := NewEngine(EngineOptions{
		Bus:         bus,
		VoicePool:   pool,
		Talkgroups:  NewTalkgroupDB(),
		CallTimeout: 5 * time.Second,
		Now:         clock.Now,
	})

	sub := bus.Subscribe()
	defer sub.Close()

	g := Grant{System: "X", Protocol: "p25", GroupID: 32181, FrequencyHz: 773_431_250}
	e.HandleGrant(g)
	if got := len(e.ActiveCalls()); got != 1 {
		t.Fatalf("first grant: active calls = %d, want 1", got)
	}
	firstHeard := e.ActiveCalls()[0].LastHeardAt
	firstDevice := e.ActiveCalls()[0].Device.Serial

	// Advance the clock so the dedup's Touch produces a distinguishable
	// LastHeardAt and fire the identical grant. The second call must
	// not allocate device #2 — only one tuner should ever be retuned.
	clock.t = clock.t.Add(20 * time.Millisecond)
	e.HandleGrant(g)

	if got := len(e.ActiveCalls()); got != 1 {
		t.Fatalf("after duplicate grant: active calls = %d, want 1", got)
	}
	if got := e.ActiveCalls()[0].Device.Serial; got != firstDevice {
		t.Errorf("duplicate grant rebound device: got %q, want %q", got, firstDevice)
	}
	if got := e.ActiveCalls()[0].LastHeardAt; !got.After(firstHeard) {
		t.Errorf("duplicate grant did not refresh LastHeardAt: got %v, want >%v", got, firstHeard)
	}
	// Verify the second device was never touched.
	for i, tn := range tuners {
		if i == 0 {
			continue
		}
		if got := tn.tuned(); len(got) != 0 {
			t.Errorf("device %d was retuned on duplicate grant: tuned=%v", i, got)
		}
	}

	// Drain the bus and confirm exactly one CallStart was published —
	// the second grant must not emit a second start event.
	deadline := time.Now().Add(200 * time.Millisecond)
	starts := 0
	for time.Now().Before(deadline) {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCallStart {
				starts++
			}
		case <-time.After(50 * time.Millisecond):
		}
	}
	if starts != 1 {
		t.Errorf("CallStart events = %d, want 1", starts)
	}
}

// TestEngineHandleGrantFollowsFrequencyChange covers the duplicate-call bug:
// a re-grant for an already-active talkgroup arriving on a NEW frequency (a
// band-plan IdentifierUpdate re-mapped the channel, or the call was handed to
// a new channel). The old code keyed the call on frequency, so the re-grant
// missed the dedup and bound a SECOND tuner — two "Active calls" rows for the
// same TG. Now the call is keyed on (System, GroupID, Timeslot) and the bound
// device follows the call to the new frequency in place.
func TestEngineHandleGrantFollowsFrequencyChange(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	pool, tuners := mkPool(2) // universal tuners (no FrequencyChecker)
	clock := &fakeClock{t: time.Unix(1000, 0)}
	e, _ := NewEngine(EngineOptions{
		Bus:         bus,
		VoicePool:   pool,
		Talkgroups:  NewTalkgroupDB(),
		CallTimeout: 5 * time.Second,
		Now:         clock.Now,
	})
	sub := bus.Subscribe()
	defer sub.Close()

	const f1, f2 = uint32(773_431_250), uint32(773_456_250)
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 1412, FrequencyHz: f1})
	if got := len(e.ActiveCalls()); got != 1 {
		t.Fatalf("first grant: active calls = %d, want 1", got)
	}
	firstDevice := e.ActiveCalls()[0].Device.Serial
	firstStarted := e.ActiveCalls()[0].StartedAt

	// Same logical call, new frequency, 20 ms later.
	clock.t = clock.t.Add(20 * time.Millisecond)
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 1412, FrequencyHz: f2})

	if got := len(e.ActiveCalls()); got != 1 {
		t.Fatalf("after frequency change: active calls = %d, want 1 (duplicate-call bug)", got)
	}
	ac := e.ActiveCalls()[0]
	if ac.Device.Serial != firstDevice {
		t.Errorf("call moved devices: got %q, want %q (should follow in place)", ac.Device.Serial, firstDevice)
	}
	if ac.Grant.FrequencyHz != f2 {
		t.Errorf("stored grant frequency = %d, want %d (retune did not update grant)", ac.Grant.FrequencyHz, f2)
	}
	if !ac.StartedAt.Equal(firstStarted) {
		t.Errorf("StartedAt changed on follow: got %v, want %v (call continuity broken)", ac.StartedAt, firstStarted)
	}
	if got := tuners[0].tuned(); len(got) != 2 || got[0] != f1 || got[1] != f2 {
		t.Errorf("device 0 tune sequence = %v, want [%d %d]", got, f1, f2)
	}
	if got := tuners[1].tuned(); len(got) != 0 {
		t.Errorf("device 1 was bound on a frequency change: tuned=%v (should never happen)", got)
	}

	// Exactly one CallStart — following a call must not emit a new start.
	deadline := time.Now().Add(200 * time.Millisecond)
	starts := 0
	for time.Now().Before(deadline) {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCallStart {
				starts++
			}
		case <-time.After(50 * time.Millisecond):
		}
	}
	if starts != 1 {
		t.Errorf("CallStart events = %d, want 1", starts)
	}
}

// TestEngineHandleGrantFollowsToCapableDevice covers the follow path when the
// bound device CANNOT reach the new frequency (the call moved outside a
// wideband tap's window). The stale bind is released and a capable device is
// bound to the new frequency — still exactly one active call for the TG.
func TestEngineHandleGrantFollowsToCapableDevice(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	// Device order: a band-limited tap first, an unconstrained SDR second.
	tap := &constrainedTuner{lo: 773_000_000, hi: 774_000_000}
	universal := &fakeVoiceTuner{}
	pool := NewVoicePool([]*VoiceDevice{
		{Tuner: tap, Serial: "wb:dev:tap-0"},
		{Tuner: universal, Serial: "phys"},
	})
	clock := &fakeClock{t: time.Unix(1000, 0)}
	e, _ := NewEngine(EngineOptions{
		Bus:         bus,
		VoicePool:   pool,
		Talkgroups:  NewTalkgroupDB(),
		CallTimeout: 5 * time.Second,
		Now:         clock.Now,
	})

	const fIn, fOut = uint32(773_431_250), uint32(850_000_000)
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 1412, FrequencyHz: fIn})
	if got := e.ActiveCalls(); len(got) != 1 || got[0].Device.Serial != "wb:dev:tap-0" {
		t.Fatalf("first grant should bind the tap; got %+v", got)
	}

	clock.t = clock.t.Add(20 * time.Millisecond)
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 1412, FrequencyHz: fOut})

	got := e.ActiveCalls()
	if len(got) != 1 {
		t.Fatalf("after out-of-window move: active calls = %d, want 1", len(got))
	}
	if got[0].Device.Serial != "phys" {
		t.Errorf("call did not rebind to a capable device: on %q, want phys", got[0].Device.Serial)
	}
	if got[0].Grant.FrequencyHz != fOut {
		t.Errorf("rebound grant frequency = %d, want %d", got[0].Grant.FrequencyHz, fOut)
	}
	if len(universal.freqs) != 1 || universal.freqs[0] != fOut {
		t.Errorf("universal device tune = %v, want [%d]", universal.freqs, fOut)
	}
}

// TestEngineHandleGrantSeparatesTimeslots is the DMR Tier III guard:
// two grants on the same carrier but different TDMA slots are two
// independent calls and must each bind a device, while a repeat of a
// slot's own grant still dedups to the existing call. Without the
// timeslot dimension in the dedup the TS2 grant would be folded into
// the active TS1 call and the second call would be silently dropped.
func TestEngineHandleGrantSeparatesTimeslots(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	pool, _ := mkPool(2)
	clock := &fakeClock{t: time.Unix(1000, 0)}
	e, _ := NewEngine(EngineOptions{
		Bus:         bus,
		VoicePool:   pool,
		Talkgroups:  NewTalkgroupDB(),
		CallTimeout: 5 * time.Second,
		Now:         clock.Now,
	})

	const freq = 460_000_000
	ts1 := Grant{System: "X", Protocol: "dmr-tier3", GroupID: 100, FrequencyHz: freq, Timeslot: 1}
	ts2 := Grant{System: "X", Protocol: "dmr-tier3", GroupID: 100, FrequencyHz: freq, Timeslot: 2}

	e.HandleGrant(ts1)
	if got := len(e.ActiveCalls()); got != 1 {
		t.Fatalf("after TS1 grant: active calls = %d, want 1", got)
	}

	// Same TG + frequency, other slot → a distinct call on a second
	// device, NOT a dedup of the TS1 call.
	e.HandleGrant(ts2)
	if got := len(e.ActiveCalls()); got != 2 {
		t.Fatalf("after TS2 grant: active calls = %d, want 2 (one per slot)", got)
	}

	// Confirm both slots are represented exactly once on distinct devices.
	slots := map[uint8]string{}
	for _, ac := range e.ActiveCalls() {
		if prev, dup := slots[ac.Grant.Timeslot]; dup {
			t.Fatalf("timeslot %d bound twice (devices %q, %q)", ac.Grant.Timeslot, prev, ac.Device.Serial)
		}
		slots[ac.Grant.Timeslot] = ac.Device.Serial
	}
	if slots[1] == "" || slots[2] == "" {
		t.Fatalf("expected both TS1 and TS2 active, got slots=%v", slots)
	}
	if slots[1] == slots[2] {
		t.Errorf("both slots bound to the same device %q", slots[1])
	}

	// A repeat of the TS1 grant must still dedup (refresh), not allocate
	// a third call — proving the slot dimension didn't disable dedup.
	clock.t = clock.t.Add(20 * time.Millisecond)
	e.HandleGrant(ts1)
	if got := len(e.ActiveCalls()); got != 2 {
		t.Fatalf("after TS1 repeat: active calls = %d, want 2 (refresh, not a new call)", got)
	}
}

// TestEngineHandleGrantSameTGNewFrequencyFollowsOneCall asserts the
// logical-call identity: a same-system, same-TG (same-timeslot) grant on a
// DIFFERENT frequency is the SAME call that moved channels — it must follow on
// one device, NOT spawn a second active call. This previously asserted two
// calls ("rare but legitimate site rebind"), which was exactly the duplicate-
// "Active calls" bug operators reported (the same TG shown twice on two taps).
func TestEngineHandleGrantSameTGNewFrequencyFollowsOneCall(t *testing.T) {
	e, _, bus, _ := mkEngine(t, 2)
	defer bus.Close()

	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 100, FrequencyHz: 851_000_000})
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 100, FrequencyHz: 852_000_000})

	if got := len(e.ActiveCalls()); got != 1 {
		t.Errorf("active calls = %d, want 1 (one call followed to the new frequency)", got)
	}
	if got := e.ActiveCalls()[0].Grant.FrequencyHz; got != 852_000_000 {
		t.Errorf("followed-call frequency = %d, want 852000000", got)
	}

	// A grant for the same TG on a DIFFERENT system is a distinct call and
	// must still bind its own device.
	e.HandleGrant(Grant{System: "Y", Protocol: "p25", GroupID: 100, FrequencyHz: 853_000_000})
	if got := len(e.ActiveCalls()); got != 2 {
		t.Errorf("active calls = %d, want 2 (different systems are distinct calls)", got)
	}
}

func TestEngineRunDispatchesGrantEvents(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	pool, tuners := mkPool(1)
	e, _ := NewEngine(EngineOptions{
		Bus:         bus,
		VoicePool:   pool,
		Talkgroups:  NewTalkgroupDB(),
		CallTimeout: 5 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go e.Run(ctx)

	bus.Publish(events.Event{
		Kind:    events.KindGrant,
		Payload: Grant{GroupID: 7, FrequencyHz: 851_000_000},
	})

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if len(tuners[0].tuned()) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := tuners[0].tuned(); len(got) != 1 || got[0] != 851_000_000 {
		t.Errorf("device retune = %v, want [851000000]", got)
	}
}

type fakeClock struct {
	t time.Time
}

func (c *fakeClock) Now() time.Time { return c.t }

func TestEngineHandleCallEncryptionBackfillsActiveCall(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	pool, _ := mkPool(1)
	e, _ := NewEngine(EngineOptions{
		Bus:         bus,
		VoicePool:   pool,
		Talkgroups:  NewTalkgroupDB(),
		CallTimeout: 5 * time.Second,
	})

	// Start a Phase 1 call. ALGID/KID are zero (Phase 1 grant TSBK
	// doesn't carry them).
	e.HandleGrant(Grant{
		System: "MMR", Protocol: "p25",
		GroupID: 4321, FrequencyHz: 851_000_000,
		Encrypted: true,
	})

	actives := e.ActiveCalls()
	if len(actives) != 1 {
		t.Fatalf("expected 1 active call, got %d", len(actives))
	}
	dev := actives[0].Device.Serial
	if actives[0].Grant.AlgorithmID != 0 || actives[0].Grant.KeyID != 0 {
		t.Fatalf("pre-backfill alg/key should be zero, got %v/%v",
			actives[0].Grant.AlgorithmID, actives[0].Grant.KeyID)
	}

	// Subscribe BEFORE driving the encryption update so we observe
	// the enriched republish without racing the event loop.
	sub := bus.Subscribe()
	defer sub.Close()

	// Composer publishes the raw event (System/Protocol/GroupID empty);
	// the engine must backfill, enrich, and republish.
	e.handleCallEncryption(CallEncryption{
		DeviceSerial: dev,
		AlgorithmID:  0x84, // AES-256
		KeyID:        0x1234,
		At:           time.Now(),
	})

	// The pool's ActiveCall.Grant should now carry the values so the
	// next CallEnd payload includes them.
	updated := e.ActiveCalls()
	if len(updated) != 1 {
		t.Fatalf("expected 1 active call after backfill, got %d", len(updated))
	}
	if updated[0].Grant.AlgorithmID != 0x84 || updated[0].Grant.KeyID != 0x1234 {
		t.Errorf("backfill did not land on Grant: alg=0x%X key=0x%X",
			updated[0].Grant.AlgorithmID, updated[0].Grant.KeyID)
	}

	// Enriched republish should be on the bus with system / tg filled.
	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCallEncryption {
			t.Fatalf("expected KindCallEncryption republish, got %s", ev.Kind)
		}
		ce, ok := ev.Payload.(CallEncryption)
		if !ok {
			t.Fatalf("payload type = %T", ev.Payload)
		}
		if ce.System != "MMR" || ce.Protocol != "p25" || ce.GroupID != 4321 {
			t.Errorf("enriched payload missing identity fields: %+v", ce)
		}
		if ce.AlgorithmID != 0x84 || ce.KeyID != 0x1234 {
			t.Errorf("enriched payload alg/key = 0x%X/0x%X", ce.AlgorithmID, ce.KeyID)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("never received enriched republish")
	}
}

func TestEngineHandleCallEncryptionUnknownDeviceDoesNotPanic(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	pool, _ := mkPool(1)
	e, _ := NewEngine(EngineOptions{
		Bus:         bus,
		VoicePool:   pool,
		Talkgroups:  NewTalkgroupDB(),
		CallTimeout: 5 * time.Second,
	})

	// No active call on this device serial. Should silently drop.
	e.handleCallEncryption(CallEncryption{
		DeviceSerial: "no-such-device",
		AlgorithmID:  0x84,
		KeyID:        0x1234,
	})
}

// TestEngineHandleCallSourceUpdateBackfillsActiveCall guards the
// Phase 2 traffic-channel source / encryption backfill flow (the
// MMR fix for issue #376). The CC publishes a grant with src=0 and
// enc=false (compressed grant form); the voice composer recovers
// the real source RID + encrypted state from an in-call
// GROUP_VOICE_CHANNEL_USER PDU on the traffic channel and publishes
// CallSourceUpdate. The engine must (1) patch the bound
// ActiveCall.Grant via VoicePool.UpdateSource, (2) republish with
// System / Protocol / GroupID filled.
func TestEngineHandleCallSourceUpdateBackfillsActiveCall(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	pool, _ := mkPool(1)
	e, _ := NewEngine(EngineOptions{
		Bus:         bus,
		VoicePool:   pool,
		Talkgroups:  NewTalkgroupDB(),
		CallTimeout: 5 * time.Second,
	})

	// Phase 2 grant arrives in a compressed form: src=0, enc=false.
	e.HandleGrant(Grant{
		System: "MMR", Protocol: "p25-phase2",
		GroupID: 20202, FrequencyHz: 421_387_500,
	})
	actives := e.ActiveCalls()
	if len(actives) != 1 {
		t.Fatalf("expected 1 active call, got %d", len(actives))
	}
	dev := actives[0].Device.Serial
	if actives[0].Grant.SourceID != 0 || actives[0].Grant.Encrypted {
		t.Fatalf("pre-backfill src/enc should be 0/false, got %v/%v",
			actives[0].Grant.SourceID, actives[0].Grant.Encrypted)
	}

	sub := bus.Subscribe()
	defer sub.Close()

	e.handleCallSourceUpdate(CallSourceUpdate{
		DeviceSerial: dev,
		SourceID:     315203, // @er-imagery's example MMR radio
		Encrypted:    true,
		At:           time.Now(),
	})

	updated := e.ActiveCalls()
	if len(updated) != 1 {
		t.Fatalf("expected 1 active call after backfill, got %d", len(updated))
	}
	if updated[0].Grant.SourceID != 315203 || !updated[0].Grant.Encrypted {
		t.Errorf("backfill did not land on Grant: src=%d enc=%v",
			updated[0].Grant.SourceID, updated[0].Grant.Encrypted)
	}

	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCallSourceUpdate {
			t.Fatalf("expected KindCallSourceUpdate republish, got %s", ev.Kind)
		}
		c, ok := ev.Payload.(CallSourceUpdate)
		if !ok {
			t.Fatalf("payload type = %T", ev.Payload)
		}
		if c.System != "MMR" || c.Protocol != "p25-phase2" || c.GroupID != 20202 {
			t.Errorf("enriched payload missing identity fields: %+v", c)
		}
		if c.SourceID != 315203 || !c.Encrypted {
			t.Errorf("enriched payload src/enc = %d/%v", c.SourceID, c.Encrypted)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("never received enriched republish")
	}
}

// TestEngineHandleCallSourceUpdateUnknownDeviceDoesNotPanic guards
// the late-arriving PDU after a call ends.
func TestEngineHandleCallSourceUpdateUnknownDeviceDoesNotPanic(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	pool, _ := mkPool(1)
	e, _ := NewEngine(EngineOptions{
		Bus:         bus,
		VoicePool:   pool,
		Talkgroups:  NewTalkgroupDB(),
		CallTimeout: 5 * time.Second,
	})
	e.handleCallSourceUpdate(CallSourceUpdate{
		DeviceSerial: "no-such-device",
		SourceID:     999,
	})
}

func TestEngineWidebandOnlyOutOfWindowGrantWarnsNotEngineBug(t *testing.T) {
	// Issue #422 follow-up: a wideband-only pool (one frequency-
	// constrained tap, no physical voice SDR) that receives a grant
	// outside its IQ window must not log the misleading "voice pool
	// full but no actives (engine bug)" Error. It's a coverage gap:
	// warn once, drop the rest at DEBUG.
	bus := events.NewBus(8)
	defer bus.Close()
	tap := &constrainedTuner{lo: 851_000_000, hi: 852_000_000}
	pool := NewVoicePool([]*VoiceDevice{{Tuner: tap, Serial: "wb:dev:tap-0"}})

	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	e, err := NewEngine(EngineOptions{
		Bus: bus, Log: log, VoicePool: pool, Talkgroups: NewTalkgroupDB(),
	})
	if err != nil {
		t.Fatal(err)
	}

	// 900 MHz is outside the tap's [851, 852] MHz window.
	g := Grant{System: "X", Protocol: "p25", GroupID: 1, FrequencyHz: 900_000_000}
	e.HandleGrant(g)
	e.HandleGrant(g)
	e.HandleGrant(g)

	out := buf.String()
	if strings.Contains(out, "engine bug") || strings.Contains(out, "voice pool full but no actives") {
		t.Errorf("out-of-window grant mislogged as engine bug:\n%s", out)
	}
	if c := strings.Count(out, "level=WARN"); c != 1 {
		t.Errorf("expected exactly one WARN, got %d:\n%s", c, out)
	}
	if !strings.Contains(out, "outside every voice device") {
		t.Errorf("expected actionable coverage WARN:\n%s", out)
	}
	if c := strings.Count(out, `msg="dropping grant: no voice device covers frequency"`); c != 3 {
		t.Errorf("expected 3 DEBUG drops, got %d:\n%s", c, out)
	}
	// The tap must never have been retuned to the out-of-window freq.
	if len(tap.freqs) != 0 {
		t.Errorf("constrained tap should not have been retuned, got %v", tap.freqs)
	}
}

func TestEnginePreemptsFrequencyCapableDeviceNotLowestPriority(t *testing.T) {
	// Issue #422 follow-up: preemption must target a device that can
	// actually tune the incoming grant. Here the globally lowest-
	// priority call sits on a wideband tap whose window excludes the
	// grant; the engine must instead preempt the (higher-priority but
	// frequency-capable) physical SDR, leaving the tap's call intact.
	bus := events.NewBus(8)
	defer bus.Close()
	tap := &constrainedTuner{lo: 851_000_000, hi: 852_000_000}
	phys := &fakeVoiceTuner{}
	pool := NewVoicePool([]*VoiceDevice{
		{Tuner: tap, Serial: "wb:dev:tap-0"}, // listed first; preferred in-window
		{Tuner: phys, Serial: "phys-voice"},
	})
	e, err := NewEngine(EngineOptions{
		Bus: bus, VoicePool: pool, Talkgroups: NewTalkgroupDB(),
	})
	if err != nil {
		t.Fatal(err)
	}
	e.talkgroups.Add(&TalkGroup{ID: 200, Priority: 9}) // tap call: lowest priority
	e.talkgroups.Add(&TalkGroup{ID: 100, Priority: 5}) // phys call
	e.talkgroups.Add(&TalkGroup{ID: 300, Priority: 1}) // incoming: high priority

	// Bind the tap with an in-window call (group 200).
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 200, FrequencyHz: 851_500_000})
	// Bind the physical SDR with an out-of-tap-window call (group 100).
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 100, FrequencyHz: 900_000_000})
	// Incoming high-priority grant at 900 MHz — only the physical SDR
	// can serve it. It must preempt the physical call (group 100), not
	// the lower-priority tap call (group 200).
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 300, FrequencyHz: 900_000_001})

	byGroup := map[uint32]*ActiveCall{}
	for _, ac := range e.ActiveCalls() {
		byGroup[ac.Grant.GroupID] = ac
	}
	if _, ok := byGroup[200]; !ok {
		t.Error("tap's group-200 call should still be active (tap can't serve 900 MHz)")
	}
	if _, ok := byGroup[100]; ok {
		t.Error("physical's group-100 call should have been preempted")
	}
	c300, ok := byGroup[300]
	if !ok || c300.Device.Serial != "phys-voice" {
		t.Errorf("incoming group-300 call should be bound to phys-voice, got %+v", c300)
	}
}

func TestEngineEmptyVoicePoolWarnsOnceThenDebug(t *testing.T) {
	// Issue #379: a daemon with trunking systems but zero `role: voice`
	// SDRs builds an empty VoicePool. Every grant used to log a
	// misleading "voice pool full but no actives" warning. The engine
	// now logs one actionable WARN and drops the rest at DEBUG.
	bus := events.NewBus(8)
	defer bus.Close()
	pool, _ := mkPool(0)

	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	e, err := NewEngine(EngineOptions{
		Bus:        bus,
		Log:        log,
		VoicePool:  pool,
		Talkgroups: NewTalkgroupDB(),
	})
	if err != nil {
		t.Fatal(err)
	}

	g := Grant{System: "X", Protocol: "p25", GroupID: 1, FrequencyHz: 851_000_000}
	e.HandleGrant(g)
	e.HandleGrant(g)
	e.HandleGrant(g)

	out := buf.String()
	// Misleading legacy message must be gone.
	if strings.Contains(out, "voice pool full but no actives") {
		t.Errorf("legacy misleading warning still present:\n%s", out)
	}
	// Actionable WARN logged exactly once across three grants.
	warnCount := strings.Count(out, "level=WARN")
	if warnCount != 1 {
		t.Errorf("expected exactly one WARN, got %d:\n%s", warnCount, out)
	}
	if !strings.Contains(out, "no voice SDR available") {
		t.Errorf("expected actionable WARN about missing voice SDR:\n%s", out)
	}
	// Every grant drops at DEBUG; the first also emits the one-shot WARN.
	debugCount := strings.Count(out, `msg="dropping grant: no voice SDR"`)
	if debugCount != 3 {
		t.Errorf("expected 3 DEBUG drops (one per grant), got %d:\n%s", debugCount, out)
	}
}

func TestEngineHandleCallEncryptionEnrichedRepublishDoesNotLoop(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	pool, _ := mkPool(1)
	e, _ := NewEngine(EngineOptions{
		Bus:         bus,
		VoicePool:   pool,
		Talkgroups:  NewTalkgroupDB(),
		CallTimeout: 5 * time.Second,
	})
	e.HandleGrant(Grant{
		System: "MMR", Protocol: "p25",
		GroupID: 1, FrequencyHz: 851_000_000, Encrypted: true,
	})
	dev := e.ActiveCalls()[0].Device.Serial

	// An event with System already set is the engine's own republish
	// coming back through the subscription. Must be ignored — otherwise
	// the engine would publish another republish, and so on.
	sub := bus.Subscribe()
	defer sub.Close()
	e.handleCallEncryption(CallEncryption{
		DeviceSerial: dev,
		System:       "MMR",
		AlgorithmID:  0x84,
		KeyID:        0x1234,
	})
	select {
	case ev := <-sub.C:
		t.Fatalf("expected no republish for already-enriched event, got %s", ev.Kind)
	case <-time.After(100 * time.Millisecond):
		// pass
	}
}
