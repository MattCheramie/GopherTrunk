package api

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestListSystemsOverlaysLiveIdentity verifies GET /api/v1/systems fills the
// WACN/SystemID/RFSS/Site network-identity fields from the live SiteTracker
// snapshot rather than the (empty) static config copy — the fix for the
// "System Information panel stuck on Awaiting status broadcasts" bug (#673).
func TestListSystemsOverlaysLiveIdentity(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()

	// Config carries no decoded identity (operators don't set it); the
	// SiteTracker has heard one site over the air.
	systems := []trunking.System{{Name: "MMR", Protocol: trunking.ProtocolP25}}
	discovered := fakeSites{sites: []trunking.SiteInfo{
		{System: "MMR", RFSSID: 1, SiteID: 4, WACN: 0xBEE00, SystemID: 0x123, LastSeen: time.Unix(100, 0)},
	}}

	base, teardown := mkServer(t, ServerOptions{Bus: bus, Systems: systems, Sites: discovered})
	defer teardown()

	resp := mustGet(t, base+"/api/v1/systems")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		Systems []SystemDTO `json:"systems"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Systems) != 1 {
		t.Fatalf("got %d systems, want 1", len(body.Systems))
	}
	got := body.Systems[0]
	if got.WACN != 0xBEE00 {
		t.Errorf("WACN = %#x, want 0xBEE00", got.WACN)
	}
	if got.SystemID != 0x123 {
		t.Errorf("SystemID = %#x, want 0x123", got.SystemID)
	}
	if got.RFSS != 1 {
		t.Errorf("RFSS = %d, want 1", got.RFSS)
	}
	if got.Site != 4 {
		t.Errorf("Site = %d, want 4", got.Site)
	}
}

// TestListSystemsUsesMostRecentSite checks that for a multi-site system the
// RFSS/Site shown reflect the most-recently-heard site (the currently camped
// one), while WACN/SystemID — system-wide values — come from the latest
// non-zero observation.
func TestListSystemsUsesMostRecentSite(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()

	systems := []trunking.System{{Name: "MMR", Protocol: trunking.ProtocolP25}}
	discovered := fakeSites{sites: []trunking.SiteInfo{
		{System: "MMR", RFSSID: 1, SiteID: 4, WACN: 0xBEE00, SystemID: 0x123, LastSeen: time.Unix(100, 0)},
		{System: "MMR", RFSSID: 1, SiteID: 9, LastSeen: time.Unix(200, 0)}, // newer, no identity bytes decoded yet
	}}

	base, teardown := mkServer(t, ServerOptions{Bus: bus, Systems: systems, Sites: discovered})
	defer teardown()

	resp := mustGet(t, base+"/api/v1/systems")
	defer resp.Body.Close()
	var body struct {
		Systems []SystemDTO `json:"systems"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	got := body.Systems[0]
	if got.Site != 9 {
		t.Errorf("Site = %d, want 9 (most recently heard)", got.Site)
	}
	// WACN/SystemID held from the earlier row that actually carried them.
	if got.WACN != 0xBEE00 || got.SystemID != 0x123 {
		t.Errorf("WACN/SystemID = %#x/%#x, want 0xBEE00/0x123", got.WACN, got.SystemID)
	}
}
