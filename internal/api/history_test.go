package api

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func TestCallHistoryEndpointSurfacesPersistedRows(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	dbPath := filepath.Join(t.TempDir(), "calls.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	cl, err := storage.NewCallLog(db, bus, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cl.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cl.Run(ctx)

	startedAt := time.Now().UTC().Truncate(time.Microsecond)
	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			Grant: trunking.Grant{
				System: "Alpha", Protocol: "p25",
				GroupID: 7777, FrequencyHz: 851_000_000,
			},
			Talkgroup:    &trunking.TalkGroup{ID: 7777, AlphaTag: "FIRE-DISP"},
			DeviceSerial: "VOICE-1",
			StartedAt:    startedAt,
		},
	})

	// Wait for the row.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		rows, _ := db.History(context.Background(), storage.HistoryFilter{Limit: 1})
		if len(rows) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	base, teardown := mkServer(t, ServerOptions{
		Bus:     bus,
		History: HistoryFromStorage(db),
	})
	defer teardown()

	resp := mustGet(t, base+"/api/v1/calls/history")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var body struct {
		Calls []CallRow `json:"calls"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if len(body.Calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(body.Calls))
	}
	got := body.Calls[0]
	if got.System != "Alpha" || got.GroupID != 7777 || got.TalkgroupAlpha != "FIRE-DISP" {
		t.Errorf("row = %+v", got)
	}
}

func TestCallHistoryEndpointFiltersBySystem(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	db, _ := storage.Open(":memory:")
	defer db.Close()
	cl, _ := storage.NewCallLog(db, bus, nil)
	defer cl.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cl.Run(ctx)

	startedAt := time.Now().UTC().Truncate(time.Microsecond)
	for i, sys := range []string{"Alpha", "Bravo", "Alpha"} {
		bus.Publish(events.Event{
			Kind: events.KindCallStart,
			Payload: trunking.CallStart{
				Grant:        trunking.Grant{System: sys, GroupID: uint32(100 + i), FrequencyHz: 1},
				DeviceSerial: "X" + sys,
				StartedAt:    startedAt.Add(time.Duration(i) * time.Second),
			},
		})
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		rows, _ := db.History(context.Background(), storage.HistoryFilter{Limit: 10})
		if len(rows) == 3 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	base, teardown := mkServer(t, ServerOptions{
		Bus:     bus,
		History: HistoryFromStorage(db),
	})
	defer teardown()

	resp := mustGet(t, base+"/api/v1/calls/history?system=Alpha")
	defer resp.Body.Close()
	var body struct {
		Calls []CallRow `json:"calls"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if len(body.Calls) != 2 {
		t.Errorf("Alpha-filtered rows = %d, want 2", len(body.Calls))
	}
}

func TestCallHistoryEndpointReturns503WithoutHistory(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{Bus: bus})
	defer teardown()
	resp := mustGet(t, base+"/api/v1/calls/history")
	defer resp.Body.Close()
	if resp.StatusCode != 503 {
		t.Errorf("status = %d, want 503", resp.StatusCode)
	}
}

func TestCallHistoryEndpointRejectsBadParams(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()
	db, _ := storage.Open(":memory:")
	defer db.Close()
	base, teardown := mkServer(t, ServerOptions{
		Bus:     bus,
		History: HistoryFromStorage(db),
	})
	defer teardown()

	for _, q := range []string{
		"group_id=abc",
		"since=not-a-date",
		"until=not-a-date",
		"limit=-1",
	} {
		resp := mustGet(t, base+"/api/v1/calls/history?"+q)
		if resp.StatusCode != 400 {
			t.Errorf("%s status = %d, want 400", q, resp.StatusCode)
		}
		resp.Body.Close()
	}
}
