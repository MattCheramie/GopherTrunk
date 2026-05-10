package events

import (
	"testing"
	"time"
)

func TestBusFanout(t *testing.T) {
	b := NewBus(4)
	defer b.Close()

	a := b.Subscribe()
	defer a.Close()
	c := b.Subscribe()
	defer c.Close()

	b.Publish(Event{Kind: KindCCLocked, Payload: "851000000"})

	for _, sub := range []*Subscription{a, c} {
		select {
		case e := <-sub.C:
			if e.Kind != KindCCLocked {
				t.Errorf("kind = %q", e.Kind)
			}
			if e.Timestamp.IsZero() {
				t.Errorf("timestamp not stamped")
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for event")
		}
	}
}

func TestBusUnsubscribe(t *testing.T) {
	b := NewBus(2)
	defer b.Close()
	s := b.Subscribe()
	s.Close()
	b.Publish(Event{Kind: KindError})
	select {
	case _, ok := <-s.C:
		if ok {
			t.Fatal("received event after Close()")
		}
	case <-time.After(100 * time.Millisecond):
	}
}

func TestBusDropsWhenSlow(t *testing.T) {
	b := NewBus(1)
	defer b.Close()
	s := b.Subscribe()
	defer s.Close()

	b.Publish(Event{Kind: KindCallStart})
	dropped := b.Publish(Event{Kind: KindCallEnd})
	if dropped != 1 {
		t.Errorf("dropped = %d, want 1", dropped)
	}
}

func TestStageConstantsHaveStableWireValues(t *testing.T) {
	// Stage strings end up as Prometheus label values, so renaming a
	// constant silently breaks dashboards + alerts. This test pins
	// each constant to its current wire value — if you really need
	// to rename one, update the table here so the breaking change
	// shows up in code review.
	cases := map[Stage]string{
		StageNIDBCH:          "nid-bch",
		StageTSBKTrellis:     "tsbk-trellis",
		StageTSBKCRC:         "tsbk-crc",
		StageNoBandPlan:      "no-bandplan",
		StageSlotTypeHamming: "slottype-hamming",
		StageVoiceHeaderBPTC: "voiceheader-bptc",
		StageVoiceHeaderRS:   "voiceheader-rs",
		StageSACCHTrellis:    "sacch-trellis",
	}
	for stage, want := range cases {
		if string(stage) != want {
			t.Errorf("stage = %q, want %q", string(stage), want)
		}
	}
}
