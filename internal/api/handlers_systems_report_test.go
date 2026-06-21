package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// fakeReporter is a SitesProvider that also implements NetworkReporter.
type fakeReporter struct {
	sites  []trunking.SiteInfo
	report string
	ok     bool
}

func (f fakeReporter) Sites() []trunking.SiteInfo   { return f.sites }
func (f fakeReporter) Report(string) (string, bool) { return f.report, f.ok }

func TestGetSystemReport(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	systems := []trunking.System{{Name: "MMR"}}
	prov := fakeReporter{report: "P25 Network Configuration — MMR\nNetwork\n  WACN:BEE00[781824]", ok: true}

	base, teardown := mkServer(t, ServerOptions{Bus: bus, Systems: systems, Sites: prov})
	defer teardown()

	resp := mustGet(t, base+"/api/v1/systems/MMR/report")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		System    string `json:"system"`
		Report    string `json:"report"`
		Available bool   `json:"available"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !body.Available || !strings.Contains(body.Report, "WACN:BEE00") {
		t.Errorf("report not returned: %+v", body)
	}
}

// A configured-but-quiet system returns 200 with available=false rather than 404.
func TestGetSystemReportQuiet(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	systems := []trunking.System{{Name: "MMR"}}
	prov := fakeReporter{ok: false}

	base, teardown := mkServer(t, ServerOptions{Bus: bus, Systems: systems, Sites: prov})
	defer teardown()

	resp := mustGet(t, base+"/api/v1/systems/MMR/report")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		Available bool `json:"available"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Available {
		t.Error("quiet system should report available=false")
	}
}

// An entirely unknown system 404s.
func TestGetSystemReportUnknown(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{Bus: bus})
	defer teardown()

	resp := mustGet(t, base+"/api/v1/systems/nope/report")
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}
