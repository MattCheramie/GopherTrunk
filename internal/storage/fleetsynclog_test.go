package storage

import (
	"context"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

func openFleetSyncTestStore(t *testing.T) (*FleetSyncLog, *events.Bus) {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	bus := events.NewBus(8)
	log, err := NewFleetSyncLog(db, bus, nil)
	if err != nil {
		t.Fatalf("NewFleetSyncLog: %v", err)
	}
	t.Cleanup(func() {
		_ = log.Close()
		bus.Close()
		_ = db.Close()
	})
	return log, bus
}

func waitFleetSyncRows(log *FleetSyncLog, n int) []FleetSyncMessage {
	var recent []FleetSyncMessage
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		recent, _ = log.Recent(10)
		if len(recent) >= n {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	return recent
}

func TestFleetSyncLogInsertsBurst(t *testing.T) {
	log, bus := openFleetSyncTestStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = log.Run(ctx) }()

	bus.Publish(events.Event{
		Kind: events.KindFleetSyncMessage,
		Payload: FleetSyncMessage{
			ReceivedAt: time.Unix(1735000000, 0),
			Fleet:      107,
			Unit:       1772,
			IsFS2:      true,
			CRCOK:      true,
			RawHex:      "FC80083053057E59",
			Body:        "FleetSync II ANI: fleet=107 unit=1772",
			Serial:      "R1",
			FrequencyHz: 462_562_500,
		},
	})

	recent := waitFleetSyncRows(log, 1)
	if len(recent) != 1 {
		t.Fatalf("Recent = %d, want 1", len(recent))
	}
	r := recent[0]
	if r.Fleet != 107 || r.Unit != 1772 || !r.IsFS2 || !r.CRCOK {
		t.Errorf("identity not round-tripped: %+v", r)
	}
	if r.Serial != "R1" || r.FrequencyHz != 462_562_500 {
		t.Errorf("channel provenance not round-tripped (#1184): serial=%q freq=%d", r.Serial, r.FrequencyHz)
	}
	if r.RawHex != "FC80083053057E59" || r.Body != "FleetSync II ANI: fleet=107 unit=1772" {
		t.Errorf("Recent[0] = %+v", r)
	}
	if !r.ReceivedAt.Equal(time.Unix(1735000000, 0)) {
		t.Errorf("ReceivedAt = %v", r.ReceivedAt)
	}
}

func TestFleetSyncLogIgnoresUnrelatedEvents(t *testing.T) {
	log, bus := openFleetSyncTestStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = log.Run(ctx) }()

	bus.Publish(events.Event{Kind: events.KindBookmarkCreated, Payload: Bookmark{Name: "x"}})
	bus.Publish(events.Event{Kind: events.KindMDC1200Message, Payload: MDC1200Message{UnitID: 1}})
	bus.Publish(events.Event{Kind: events.KindFleetSyncMessage, Payload: "wrong type"})

	time.Sleep(50 * time.Millisecond)
	recent, _ := log.Recent(10)
	if len(recent) != 0 {
		t.Errorf("Recent = %d, want 0", len(recent))
	}
}

func TestFleetSyncLogRecentOrderNewestFirst(t *testing.T) {
	log, bus := openFleetSyncTestStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = log.Run(ctx) }()

	for i := 0; i < 3; i++ {
		bus.Publish(events.Event{
			Kind: events.KindFleetSyncMessage,
			Payload: FleetSyncMessage{
				ReceivedAt: time.Unix(1735000000+int64(i)*60, 0),
				Fleet:      107,
				Unit:       1000 + i,
				CRCOK:      true,
			},
		})
	}
	recent := waitFleetSyncRows(log, 3)
	if len(recent) != 3 {
		t.Fatalf("Recent = %d, want 3", len(recent))
	}
	if recent[0].Unit != 1002 || recent[2].Unit != 1000 {
		t.Errorf("ordering wrong: %+v", recent)
	}
}

// TestFleetSyncLogSweptByRetention pins fleetsync_log into the decoder-log
// retention sweep: a row older than LogRowMaxAge is deleted, a fresh one kept.
func TestFleetSyncLogSweptByRetention(t *testing.T) {
	log, bus := openFleetSyncTestStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = log.Run(ctx) }()

	bus.Publish(events.Event{
		Kind:    events.KindFleetSyncMessage,
		Payload: FleetSyncMessage{ReceivedAt: time.Now().Add(-48 * time.Hour), Fleet: 1, Unit: 1},
	})
	bus.Publish(events.Event{
		Kind:    events.KindFleetSyncMessage,
		Payload: FleetSyncMessage{ReceivedAt: time.Now(), Fleet: 1, Unit: 2},
	})
	if got := waitFleetSyncRows(log, 2); len(got) != 2 {
		t.Fatalf("Recent = %d, want 2 before the sweep", len(got))
	}

	r, err := NewRetention(RetentionOptions{DB: log.db, LogRowMaxAge: 24 * time.Hour})
	if err != nil {
		t.Fatalf("NewRetention: %v", err)
	}
	r.SweepOnce(ctx)

	recent, _ := log.Recent(10)
	if len(recent) != 1 || recent[0].Unit != 2 {
		t.Errorf("after sweep Recent = %+v, want only the fresh row (unit 2)", recent)
	}
}
