// AIS / vessel log writer — drains KindAISMessage events off the
// shared bus and writes one row per decoded message to the SQLite
// vessel_log table. Mirrors aprslog.go and pagerlog.go in structure.
package storage

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// AISMessage is one persisted decoded AIS message. Position-bearing
// types (1/2/3/4/18/19/27) carry Latitude / Longitude + COG / SOG +
// Heading; static-data types (5/24) carry VesselName / Callsign /
// Destination etc. Everything else stays empty so the column shape
// is stable across types.
type AISMessage struct {
	ID         int64     `json:"id"`
	ReceivedAt time.Time `json:"received_at"`
	MMSI       uint32    `json:"mmsi"`
	Type       string    `json:"type"`
	Body       string    `json:"body"`

	// Position fields — populated when the message carries lat/lon.
	Latitude         float64 `json:"latitude,omitempty"`
	Longitude        float64 `json:"longitude,omitempty"`
	SpeedOverGround  float64 `json:"sog,omitempty"`
	CourseOverGround float64 `json:"cog,omitempty"`
	Heading          int     `json:"heading,omitempty"`
	HasPosition      bool    `json:"has_position"`

	// Static-data fields — populated for types 5 and 24.
	VesselName  string `json:"vessel_name,omitempty"`
	Callsign    string `json:"callsign,omitempty"`
	Destination string `json:"destination,omitempty"`
	ShipType    int    `json:"ship_type,omitempty"`
	IMO         uint32 `json:"imo,omitempty"`

	// RawHex is the wire-bit payload as hex for round-tripping into
	// offline decoders / debugging unrecognised message types.
	RawHex string `json:"raw_hex"`
	FCSOK  bool   `json:"fcs_ok"`
}

// VesselLog drains KindAISMessage events until ctx cancels or the
// bus closes.
type VesselLog struct {
	*eventLog[AISMessage]
	db *DB
}

// NewVesselLog wires the log to the bus. Subscription happens at
// construction so events published before Run() begins aren't lost.
func NewVesselLog(db *DB, bus *events.Bus, logger *slog.Logger) (*VesselLog, error) {
	if db == nil {
		return nil, errors.New("storage/vessellog: DB is required")
	}
	v := &VesselLog{db: db}
	el, err := newEventLog[AISMessage](bus, logger, events.KindAISMessage, "vessellog", v.insert)
	if err != nil {
		return nil, err
	}
	v.eventLog = el
	return v, nil
}

func (v *VesselLog) insert(m AISMessage) error {
	at := m.ReceivedAt
	if at.IsZero() {
		at = time.Now()
	}
	fcsOK := 0
	if m.FCSOK {
		fcsOK = 1
	}
	hasPos := 0
	if m.HasPosition {
		hasPos = 1
	}
	_, err := v.db.SQL().Exec(
		`INSERT INTO vessel_log
		 (received_at, mmsi, type, body, latitude, longitude,
		  sog, cog, heading, has_position, vessel_name, callsign,
		  destination, ship_type, imo, raw_hex, fcs_ok)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		at.UnixNano(), m.MMSI, m.Type, m.Body,
		m.Latitude, m.Longitude,
		m.SpeedOverGround, m.CourseOverGround, m.Heading, hasPos,
		m.VesselName, m.Callsign, m.Destination, m.ShipType, m.IMO,
		m.RawHex, fcsOK,
	)
	return err
}

// Recent returns the most recent messages, newest first, capped at
// limit. limit ≤ 0 picks 200; limit > 5000 caps at 5000.
func (v *VesselLog) Recent(limit int) ([]AISMessage, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 5000 {
		limit = 5000
	}
	rows, err := v.db.SQL().Query(
		`SELECT id, received_at, mmsi, type, body,
		        latitude, longitude, sog, cog, heading, has_position,
		        vessel_name, callsign, destination, ship_type, imo,
		        raw_hex, fcs_ok
		 FROM vessel_log ORDER BY received_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("storage/vessellog: query: %w", err)
	}
	defer rows.Close()
	var out []AISMessage
	for rows.Next() {
		var (
			m      AISMessage
			ns     int64
			fcsOK  int
			hasPos int
		)
		if err := rows.Scan(&m.ID, &ns, &m.MMSI, &m.Type, &m.Body,
			&m.Latitude, &m.Longitude, &m.SpeedOverGround,
			&m.CourseOverGround, &m.Heading, &hasPos,
			&m.VesselName, &m.Callsign, &m.Destination,
			&m.ShipType, &m.IMO, &m.RawHex, &fcsOK); err != nil {
			return nil, fmt.Errorf("storage/vessellog: scan: %w", err)
		}
		m.ReceivedAt = time.Unix(0, ns)
		m.FCSOK = fcsOK != 0
		m.HasPosition = hasPos != 0
		out = append(out, m)
	}
	return out, rows.Err()
}
