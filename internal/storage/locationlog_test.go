package storage

import (
	"context"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func TestLocationLogPersistsAndQueries(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	bus := events.NewBus(16)
	defer bus.Close()
	ll, err := NewLocationLog(db, bus, nil)
	if err != nil {
		t.Fatalf("NewLocationLog: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go ll.Run(ctx)
	t.Cleanup(func() { cancel(); ll.Close() })

	bus.Publish(events.Event{
		Kind: events.KindLocation,
		Payload: trunking.Location{
			System:    "Metro",
			Protocol:  "dmr",
			RadioID:   4242,
			Talkgroup: 100,
			Latitude:  40.7128,
			Longitude: -74.0060,
			At:        time.Now(),
		},
	})

	var rows []LocationRow
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		rows, err = ll.Recent(50)
		if err != nil {
			t.Fatalf("Recent: %v", err)
		}
		if len(rows) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	r := rows[0]
	if r.RadioID != 4242 || r.Talkgroup != 100 || r.System != "Metro" {
		t.Errorf("row metadata wrong: %+v", r)
	}
	if r.Latitude < 40.7 || r.Latitude > 40.72 || r.Longitude > -74 || r.Longitude < -74.01 {
		t.Errorf("row coords wrong: %f,%f", r.Latitude, r.Longitude)
	}
}

func TestLocationLogRequiresDBAndBus(t *testing.T) {
	if _, err := NewLocationLog(nil, events.NewBus(1), nil); err == nil {
		t.Error("NewLocationLog without a DB should error")
	}
	db, _ := Open(":memory:")
	defer db.Close()
	if _, err := NewLocationLog(db, nil, nil); err == nil {
		t.Error("NewLocationLog without a bus should error")
	}
}
