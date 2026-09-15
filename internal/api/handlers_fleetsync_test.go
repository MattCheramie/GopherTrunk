package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

type fakeFleetSyncProvider struct {
	msgs []storage.FleetSyncMessage
}

func (f *fakeFleetSyncProvider) RecentFleetSyncMessages(limit int) ([]storage.FleetSyncMessage, error) {
	if limit > 0 && limit < len(f.msgs) {
		return f.msgs[:limit], nil
	}
	return f.msgs, nil
}

func newFleetSyncTestServer(t *testing.T, prov FleetSyncProvider) *httptest.Server {
	t.Helper()
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)
	srv, err := NewServer(ServerOptions{
		Addr:      "127.0.0.1:0",
		Bus:       bus,
		FleetSync: prov,
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	ts := httptest.NewServer(srv.routes())
	t.Cleanup(ts.Close)
	return ts
}

func TestFleetSyncMessagesReturns503WhenNotWired(t *testing.T) {
	ts := newFleetSyncTestServer(t, nil)
	resp, err := http.Get(ts.URL + "/api/v1/fleetsync/messages")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", resp.StatusCode)
	}
}

func TestFleetSyncMessagesReturnsList(t *testing.T) {
	prov := &fakeFleetSyncProvider{msgs: []storage.FleetSyncMessage{
		{
			ID:         1,
			ReceivedAt: time.Unix(1735000000, 0).UTC(),
			Fleet:      107,
			Unit:       1772,
			IsFS2:      false,
			CRCOK:      true,
			RawHex:     "FE80083053059B6C",
			Body:       "FleetSync ANI: fleet=107 unit=1772",
		},
		{
			ID:         2,
			ReceivedAt: time.Unix(1735000010, 0).UTC(),
			Fleet:      107,
			Unit:       1772,
			IsFS2:      true,
			CRCOK:      true,
			RawHex:     "FC80083053057E59",
			Body:       "FleetSync II ANI: fleet=107 unit=1772",
		},
	}}
	ts := newFleetSyncTestServer(t, prov)
	resp, err := http.Get(ts.URL + "/api/v1/fleetsync/messages")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got []FleetSyncMessageDTO
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Fleet != 107 || got[0].Unit != 1772 || got[0].FS2 || !got[0].CRCOK {
		t.Errorf("row 0 = %+v", got[0])
	}
	if !got[1].FS2 || got[1].RawHex != "FC80083053057E59" {
		t.Errorf("row 1 = %+v", got[1])
	}
}

func TestFleetSyncMessagesRespectsLimit(t *testing.T) {
	prov := &fakeFleetSyncProvider{}
	for i := 0; i < 10; i++ {
		prov.msgs = append(prov.msgs, storage.FleetSyncMessage{
			ID:   int64(i + 1),
			Unit: 1000 + i,
		})
	}
	ts := newFleetSyncTestServer(t, prov)
	resp, err := http.Get(ts.URL + "/api/v1/fleetsync/messages?limit=3")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	var got []FleetSyncMessageDTO
	_ = json.NewDecoder(resp.Body).Decode(&got)
	if len(got) != 3 {
		t.Errorf("limit=3 len = %d, want 3", len(got))
	}
}
