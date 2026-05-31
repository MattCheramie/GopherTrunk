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

type fakeADSBProvider struct {
	reports []storage.AircraftReport
	current []storage.AircraftReport
}

func (f *fakeADSBProvider) RecentAircraftReports(limit int) ([]storage.AircraftReport, error) {
	if limit > 0 && limit < len(f.reports) {
		return f.reports[:limit], nil
	}
	return f.reports, nil
}

func (f *fakeADSBProvider) CurrentAircraft(maxAge time.Duration) ([]storage.AircraftReport, error) {
	return f.current, nil
}

func newADSBTestServer(t *testing.T, prov ADSBProvider) *httptest.Server {
	t.Helper()
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)
	srv, err := NewServer(ServerOptions{
		Addr: "127.0.0.1:0",
		Bus:  bus,
		ADSB: prov,
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	ts := httptest.NewServer(srv.routes())
	t.Cleanup(ts.Close)
	return ts
}

func TestADSBAircraftReturns503WhenNotWired(t *testing.T) {
	ts := newADSBTestServer(t, nil)
	resp, err := http.Get(ts.URL + "/api/v1/adsb/aircraft")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", resp.StatusCode)
	}
}

func TestADSBAircraftReturnsList(t *testing.T) {
	prov := &fakeADSBProvider{reports: []storage.AircraftReport{
		{
			ID:          1,
			ReceivedAt:  time.Unix(1735000000, 0).UTC(),
			ICAO:        0x40621D,
			ICAOHex:     "40621D",
			Kind:        "airborne-pos",
			Body:        "AIRBORNE-POS 40621D alt=38000ft",
			CRCValid:    true,
			Latitude:    52.2572,
			Longitude:   3.91937,
			Altitude:    38000,
			HasPosition: true,
			HasAltitude: true,
		},
		{
			ID:         2,
			ReceivedAt: time.Unix(1735000010, 0).UTC(),
			ICAO:       0x4840D6,
			ICAOHex:    "4840D6",
			Kind:       "ident",
			Body:       "IDENT 4840D6 \"KLM1023\" cat=4",
			CRCValid:   true,
			Callsign:   "KLM1023",
			Category:   4,
		},
	}}
	ts := newADSBTestServer(t, prov)
	resp, err := http.Get(ts.URL + "/api/v1/adsb/aircraft")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got []AircraftReportDTO
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].ICAOHex != "40621D" || !got[0].HasPosition {
		t.Errorf("row 0 = %+v", got[0])
	}
	if got[1].Callsign != "KLM1023" {
		t.Errorf("row 1 callsign = %q", got[1].Callsign)
	}
}

func TestADSBAircraftRespectsLimit(t *testing.T) {
	prov := &fakeADSBProvider{}
	for i := uint32(0); i < 10; i++ {
		prov.reports = append(prov.reports, storage.AircraftReport{
			ID:   int64(i + 1),
			ICAO: 0x4840D0 + i,
		})
	}
	ts := newADSBTestServer(t, prov)
	resp, err := http.Get(ts.URL + "/api/v1/adsb/aircraft?limit=3")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	var got []AircraftReportDTO
	_ = json.NewDecoder(resp.Body).Decode(&got)
	if len(got) != 3 {
		t.Errorf("limit=3 len = %d, want 3", len(got))
	}
}

func TestADSBAircraftCurrentReturns503WhenNotWired(t *testing.T) {
	ts := newADSBTestServer(t, nil)
	resp, err := http.Get(ts.URL + "/api/v1/adsb/aircraft/current")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", resp.StatusCode)
	}
}

func TestADSBAircraftCurrentReturnsList(t *testing.T) {
	prov := &fakeADSBProvider{current: []storage.AircraftReport{
		{ICAOHex: "DEF456", Callsign: "UAL1", HasPosition: true, Latitude: 37.5},
	}}
	ts := newADSBTestServer(t, prov)
	resp, err := http.Get(ts.URL + "/api/v1/adsb/aircraft/current?max_age_s=120")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got []AircraftReportDTO
	_ = json.NewDecoder(resp.Body).Decode(&got)
	if len(got) != 1 || got[0].Callsign != "UAL1" {
		t.Errorf("unexpected body: %+v", got)
	}
}
