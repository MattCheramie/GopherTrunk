package trunking

import (
	"context"
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

func TestEngineWatchdogTimesOutSilentCall(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	pool, _ := mkPool(1)
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
