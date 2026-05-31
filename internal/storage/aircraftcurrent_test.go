package storage

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestCurrentAircraftCoalesces verifies that separate identification,
// position, and velocity messages for one ICAO fold into a single
// live-state record, and that stale aircraft fall outside the horizon.
func TestCurrentAircraftCoalesces(t *testing.T) {
	db := openTestDB(t)
	bus := events.NewBus(8)
	defer bus.Close()
	al, err := NewAircraftLog(db, bus, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer al.Close()

	now := time.Now().UTC()
	ins := func(ageSec int, icao uint32, kind, callsign string, hasPos int, lat, lon float64, gs float64) {
		t.Helper()
		_, err := db.sql.Exec(
			`INSERT INTO aircraft_log
			 (received_at, icao, icao_hex, kind, callsign, category,
			  has_position, latitude, longitude, has_altitude, altitude,
			  ground_speed_kn, track_deg, vertical_rate_fpm, crc_valid, raw_hex, body)
			 VALUES (?, ?, ?, ?, ?, '', ?, ?, ?, 0, 0, ?, 0, 0, 1, '', '')`,
			now.Add(-time.Duration(ageSec)*time.Second).UnixNano(),
			icao, "ABC123", kind, callsign, hasPos, lat, lon, gs)
		if err != nil {
			t.Fatal(err)
		}
	}

	// ICAO 0xABC123: identification (oldest), then position, then velocity.
	ins(120, 0xABC123, "identification", "UAL1", 0, 0, 0, 0)
	ins(60, 0xABC123, "airborne-position", "", 1, 37.5, -122.0, 0)
	ins(10, 0xABC123, "velocity", "", 0, 0, 0, 420.0)
	// A second, stale aircraft outside the 5-min horizon.
	ins(1000, 0xDEAD01, "identification", "STALE", 0, 0, 0, 0)

	cur, err := al.CurrentAircraft(5 * time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(cur) != 1 {
		t.Fatalf("CurrentAircraft = %d aircraft, want 1 (stale excluded)", len(cur))
	}
	a := cur[0]
	if a.Callsign != "UAL1" {
		t.Errorf("Callsign = %q, want UAL1 (from identification msg)", a.Callsign)
	}
	if !a.HasPosition || a.Latitude != 37.5 || a.Longitude != -122.0 {
		t.Errorf("position not coalesced: %+v", a)
	}
	if a.GroundSpeedKn != 420.0 {
		t.Errorf("GroundSpeedKn = %v, want 420 (from velocity msg)", a.GroundSpeedKn)
	}
}

func TestCurrentAircraftEmpty(t *testing.T) {
	db := openTestDB(t)
	bus := events.NewBus(8)
	defer bus.Close()
	al, _ := NewAircraftLog(db, bus, nil)
	defer al.Close()
	cur, err := al.CurrentAircraft(0) // default horizon
	if err != nil {
		t.Fatal(err)
	}
	if len(cur) != 0 {
		t.Errorf("empty table → %d aircraft, want 0", len(cur))
	}
}
