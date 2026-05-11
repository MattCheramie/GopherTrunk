package sdr

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

func TestPoolPublishesAttachAndDetachEvents(t *testing.T) {
	drv := &fakeDriver{name: "fake-events", infos: []Info{
		{Driver: "fake-events", Index: 0, Serial: "EV1", TunerName: "R820T2", Gains: []int{0, 49, 100}},
	}}
	registryMu.Lock()
	registry["fake-events"] = drv
	registryMu.Unlock()
	t.Cleanup(func() {
		registryMu.Lock()
		delete(registry, "fake-events")
		registryMu.Unlock()
	})

	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	p := NewPool(nil)
	p.SetBus(bus)
	if err := p.Open([]Hint{Hint{Serial: "EV1", PPM: 3, BiasTee: true}.WithGain(496)}); err != nil {
		t.Fatal(err)
	}

	ev := waitForEvent(t, sub.C, events.KindSDRAttached)
	st, ok := ev.Payload.(SDRStatus)
	if !ok {
		t.Fatalf("attach payload type = %T, want SDRStatus", ev.Payload)
	}
	if st.Serial != "EV1" {
		t.Errorf("Serial = %q, want EV1", st.Serial)
	}
	if !st.Attached {
		t.Errorf("Attached = false, want true")
	}
	if st.GainTenthDB != 496 || st.GainAuto {
		t.Errorf("Gain = %d auto=%v, want 496 auto=false", st.GainTenthDB, st.GainAuto)
	}
	if st.PPM != 3 {
		t.Errorf("PPM = %d, want 3", st.PPM)
	}
	if !st.BiasTee {
		t.Errorf("BiasTee = false, want true")
	}
	if st.TunerName != "R820T2" {
		t.Errorf("TunerName = %q, want R820T2", st.TunerName)
	}

	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	ev = waitForEvent(t, sub.C, events.KindSDRDetached)
	st, ok = ev.Payload.(SDRStatus)
	if !ok {
		t.Fatalf("detach payload type = %T, want SDRStatus", ev.Payload)
	}
	if st.Attached {
		t.Errorf("Attached = true on detach, want false")
	}
}

func TestPoolSnapshotMatchesEntries(t *testing.T) {
	drv := &fakeDriver{name: "fake-snap", infos: []Info{
		{Driver: "fake-snap", Index: 0, Serial: "S0"},
		{Driver: "fake-snap", Index: 1, Serial: "S1"},
	}}
	registryMu.Lock()
	registry["fake-snap"] = drv
	registryMu.Unlock()
	t.Cleanup(func() {
		registryMu.Lock()
		delete(registry, "fake-snap")
		registryMu.Unlock()
	})

	p := NewPool(nil)
	if err := p.Open(nil); err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	snap := p.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("snapshot len = %d, want 2", len(snap))
	}
	roles := map[string]string{}
	for _, s := range snap {
		roles[s.Serial] = s.Role
	}
	if roles["S0"] != "control" {
		t.Errorf("S0 role = %q, want control", roles["S0"])
	}
	if roles["S1"] != "voice" {
		t.Errorf("S1 role = %q, want voice", roles["S1"])
	}
}

func waitForEvent(t *testing.T, ch <-chan events.Event, kind events.Kind) events.Event {
	t.Helper()
	timeout := time.After(time.Second)
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				t.Fatalf("bus channel closed before %s arrived", kind)
			}
			if ev.Kind == kind {
				return ev
			}
		case <-timeout:
			t.Fatalf("timeout waiting for %s event", kind)
		}
	}
}
