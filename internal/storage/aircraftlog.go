// ADS-B / aircraft log writer — drains KindAircraftReport events
// off the shared bus and writes one row per decoded Mode-S frame
// to the SQLite aircraft_log table. Mirrors vessellog.go /
// aprslog.go / dsclog.go.
package storage

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// AircraftReport is one persisted decoded Mode-S frame. Different
// message kinds populate different columns: identification frames
// fill Callsign + Category, airborne-position fills lat/lon +
// altitude, velocity fills GroundSpeedKn + TrackDeg +
// VerticalRateFPM, etc. Everything else stays empty so the column
// shape is stable across kinds.
type AircraftReport struct {
	ID         int64     `json:"id"`
	ReceivedAt time.Time `json:"received_at"`

	// 24-bit ICAO aircraft address — rendered as a 6-char hex
	// string in the JSON ("4840D6") for readability while staying
	// integer in the SQL column for index efficiency.
	ICAO     uint32 `json:"icao"`
	ICAOHex  string `json:"icao_hex"`
	Kind     string `json:"kind"` // "ident" | "airborne-pos" | "velocity" | ...
	Body     string `json:"body"` // kind-specific summary
	CRCValid bool   `json:"crc_valid"`

	// Identification fields (KindIdentification).
	Callsign string `json:"callsign,omitempty"`
	Category int    `json:"category,omitempty"`

	// Position fields (KindAirbornePosition / KindSurfacePosition).
	// Surfaces the globally-decoded lat/lon when the per-ICAO
	// CPR pairing succeeded; HasPosition distinguishes that from
	// "raw CPR fields preserved but no global decode".
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
	Altitude    int     `json:"altitude_ft,omitempty"`
	HasPosition bool    `json:"has_position"`
	HasAltitude bool    `json:"has_altitude"`

	// Velocity fields (KindAirborneVelocity).
	GroundSpeedKn   int     `json:"ground_speed_kn,omitempty"`
	TrackDeg        float64 `json:"track_deg,omitempty"`
	VerticalRateFPM int     `json:"vertical_rate_fpm,omitempty"`

	// Raw 112-bit (or 56-bit) frame hex. Round-trips into raw_hex
	// for debugging frames the parser doesn't fully decode.
	RawHex string `json:"raw_hex"`
}

// AircraftLog drains KindAircraftReport events until ctx cancels
// or the bus closes.
type AircraftLog struct {
	db        *DB
	bus       *events.Bus
	log       *slog.Logger
	sub       *events.Subscription
	runDone   chan struct{}
	closeOnce sync.Once
}

// NewAircraftLog wires the log to the bus. Subscription happens
// at construction so events published before Run() begins aren't
// lost.
func NewAircraftLog(db *DB, bus *events.Bus, logger *slog.Logger) (*AircraftLog, error) {
	if db == nil {
		return nil, errors.New("storage/aircraftlog: DB is required")
	}
	if bus == nil {
		return nil, errors.New("storage/aircraftlog: events.Bus is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	a := &AircraftLog{db: db, bus: bus, log: logger, runDone: make(chan struct{})}
	a.sub = bus.Subscribe()
	return a, nil
}

// Run drains KindAircraftReport events until ctx cancels or the
// bus closes.
func (a *AircraftLog) Run(ctx context.Context) error {
	defer close(a.runDone)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-a.sub.C:
			if !ok {
				return nil
			}
			if ev.Kind != events.KindAircraftReport {
				continue
			}
			r, ok := ev.Payload.(AircraftReport)
			if !ok {
				continue
			}
			if err := a.insert(r); err != nil {
				a.log.Error("aircraftlog: insert failed", "err", err)
			}
		}
	}
}

func (a *AircraftLog) insert(r AircraftReport) error {
	at := r.ReceivedAt
	if at.IsZero() {
		at = time.Now()
	}
	crcOK := 0
	if r.CRCValid {
		crcOK = 1
	}
	hasPos := 0
	if r.HasPosition {
		hasPos = 1
	}
	hasAlt := 0
	if r.HasAltitude {
		hasAlt = 1
	}
	_, err := a.db.SQL().Exec(
		`INSERT INTO aircraft_log
		 (received_at, icao, icao_hex, kind, body, crc_valid,
		  callsign, category, latitude, longitude, altitude_ft,
		  has_position, has_altitude, ground_speed_kn, track_deg,
		  vertical_rate_fpm, raw_hex)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		at.UnixNano(), r.ICAO, r.ICAOHex, r.Kind, r.Body, crcOK,
		r.Callsign, r.Category, r.Latitude, r.Longitude, r.Altitude,
		hasPos, hasAlt, r.GroundSpeedKn, r.TrackDeg,
		r.VerticalRateFPM, r.RawHex,
	)
	return err
}

// Recent returns the most recent reports, newest first, capped at
// limit. limit ≤ 0 picks 200; limit > 5000 caps at 5000. ADS-B is
// the highest-rate decoder (aircraft transmit ~2 msg/s), so
// operators commonly want a tighter window — set limit lower if
// rendering full panels live.
func (a *AircraftLog) Recent(limit int) ([]AircraftReport, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 5000 {
		limit = 5000
	}
	rows, err := a.db.SQL().Query(
		`SELECT id, received_at, icao, icao_hex, kind, body, crc_valid,
		        callsign, category, latitude, longitude, altitude_ft,
		        has_position, has_altitude, ground_speed_kn,
		        track_deg, vertical_rate_fpm, raw_hex
		 FROM aircraft_log ORDER BY received_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("storage/aircraftlog: query: %w", err)
	}
	defer rows.Close()
	var out []AircraftReport
	for rows.Next() {
		var (
			r      AircraftReport
			ns     int64
			crcOK  int
			hasPos int
			hasAlt int
		)
		if err := rows.Scan(&r.ID, &ns, &r.ICAO, &r.ICAOHex,
			&r.Kind, &r.Body, &crcOK, &r.Callsign, &r.Category,
			&r.Latitude, &r.Longitude, &r.Altitude,
			&hasPos, &hasAlt, &r.GroundSpeedKn, &r.TrackDeg,
			&r.VerticalRateFPM, &r.RawHex); err != nil {
			return nil, fmt.Errorf("storage/aircraftlog: scan: %w", err)
		}
		r.ReceivedAt = time.Unix(0, ns)
		r.CRCValid = crcOK != 0
		r.HasPosition = hasPos != 0
		r.HasAltitude = hasAlt != 0
		out = append(out, r)
	}
	return out, rows.Err()
}

// CurrentAircraft returns the latest known state of each aircraft seen
// within maxAge, one row per ICAO. Because aircraft_log stores one
// message type per row (identification, position, and velocity arrive
// as separate Mode-S messages), this coalesces the most recent
// non-empty value of each field group — callsign, position, altitude,
// velocity — into a single live-state record. ReceivedAt is the
// aircraft's last-seen time. Rows are newest-last-seen first.
//
// maxAge ≤ 0 defaults to 5 minutes. This powers the "currently visible
// aircraft" panel, distinct from the raw message log Recent returns.
func (a *AircraftLog) CurrentAircraft(maxAge time.Duration) ([]AircraftReport, error) {
	if maxAge <= 0 {
		maxAge = 5 * time.Minute
	}
	cutoff := time.Now().Add(-maxAge).UnixNano()
	rows, err := a.db.SQL().Query(
		`SELECT received_at, icao, icao_hex, kind, callsign, category,
		        has_position, latitude, longitude, has_altitude, altitude_ft,
		        ground_speed_kn, track_deg, vertical_rate_fpm
		 FROM aircraft_log WHERE received_at >= ? ORDER BY received_at ASC`,
		cutoff)
	if err != nil {
		return nil, fmt.Errorf("storage/aircraftlog: current query: %w", err)
	}
	defer rows.Close()

	// Fold rows in ascending time order so the last write of each field
	// group wins. cur keys by ICAO; order preserves first-seen sequence
	// for a stable sort tiebreak.
	cur := make(map[uint32]*AircraftReport)
	var order []uint32
	for rows.Next() {
		var (
			r      AircraftReport
			ns     int64
			hasPos int
			hasAlt int
		)
		if err := rows.Scan(&ns, &r.ICAO, &r.ICAOHex, &r.Kind, &r.Callsign,
			&r.Category, &hasPos, &r.Latitude, &r.Longitude, &hasAlt,
			&r.Altitude, &r.GroundSpeedKn, &r.TrackDeg, &r.VerticalRateFPM); err != nil {
			return nil, fmt.Errorf("storage/aircraftlog: current scan: %w", err)
		}
		at := time.Unix(0, ns)
		c := cur[r.ICAO]
		if c == nil {
			c = &AircraftReport{ICAO: r.ICAO}
			cur[r.ICAO] = c
			order = append(order, r.ICAO)
		}
		c.ReceivedAt = at // ASC order → final assignment is last-seen
		if r.ICAOHex != "" {
			c.ICAOHex = r.ICAOHex
		}
		if r.Callsign != "" {
			c.Callsign = r.Callsign
		}
		if r.Category != 0 {
			c.Category = r.Category
		}
		if hasPos != 0 {
			c.HasPosition = true
			c.Latitude = r.Latitude
			c.Longitude = r.Longitude
		}
		if hasAlt != 0 {
			c.HasAltitude = true
			c.Altitude = r.Altitude
		}
		if r.Kind == "velocity" {
			c.GroundSpeedKn = r.GroundSpeedKn
			c.TrackDeg = r.TrackDeg
			c.VerticalRateFPM = r.VerticalRateFPM
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]AircraftReport, 0, len(order))
	for _, icao := range order {
		out = append(out, *cur[icao])
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ReceivedAt.After(out[j].ReceivedAt)
	})
	return out, nil
}

// Close releases the bus subscription and waits for Run to drain.
func (a *AircraftLog) Close() error {
	a.closeOnce.Do(func() {
		a.sub.Close()
		select {
		case <-a.runDone:
		case <-time.After(time.Second):
		}
	})
	return nil
}
