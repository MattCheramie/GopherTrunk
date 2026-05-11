package ltr

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestStrictValidationDropsMalformedStatus: a Status word with
// Channel == 0 (outside the documented 1..20 range) must be dropped
// by Ingest under SetStrictValidation(true). Without strict mode the
// same status would still pass through Ingest because Sync is set.
func TestStrictValidationDropsMalformedStatus(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})
	cc.SetStrictValidation(true)

	cc.Ingest(Status{Sync: true, Area: 1, Channel: 0, Home: 5})

	select {
	case ev := <-sub.C:
		t.Errorf("strict-mode malformed Status published %v", ev.Kind)
	default:
	}
}

// TestStrictValidationDropsOutOfRangeHome: a Status word with Home
// outside 1..20 must be dropped under strict mode.
func TestStrictValidationDropsOutOfRangeHome(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})
	cc.SetStrictValidation(true)

	cc.Ingest(Status{Sync: true, Area: 1, Channel: 3, Home: 25})

	select {
	case ev := <-sub.C:
		t.Errorf("strict-mode Status with Home=25 published %v", ev.Kind)
	default:
	}
}

// TestStrictValidationKeepsWellFormedStatus: a Status word with all
// fixed-range fields in the documented spans must still publish a
// cc.locked under strict mode.
func TestStrictValidationKeepsWellFormedStatus(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})
	cc.SetStrictValidation(true)

	cc.Ingest(Status{
		Sync:    true,
		Area:    1,
		Channel: 3,
		Home:    5,
		GroupID: 42,
		Group:   true,
	})

	got := 0
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				got++
			}
		default:
			if got == 0 {
				t.Errorf("strict-mode well-formed Status did not publish a CCLocked")
			}
			return
		}
	}
}

func TestIsWellFormedCoversValidAndInvalidRanges(t *testing.T) {
	for ch := uint8(1); ch <= 20; ch++ {
		s := Status{Sync: true, Channel: ch, Home: 10}
		if !s.IsWellFormed() {
			t.Errorf("Channel %d should be well-formed", ch)
		}
	}
	for _, ch := range []uint8{0, 21, 22, 31} {
		s := Status{Sync: true, Channel: ch, Home: 10}
		if s.IsWellFormed() {
			t.Errorf("Channel %d should NOT be well-formed", ch)
		}
	}
	for _, h := range []uint8{0, 21, 25, 31} {
		s := Status{Sync: true, Channel: 1, Home: h}
		if s.IsWellFormed() {
			t.Errorf("Home %d should NOT be well-formed", h)
		}
	}
	if (Status{Sync: false, Channel: 1, Home: 1}).IsWellFormed() {
		t.Errorf("status without Sync should NOT be well-formed")
	}
}
