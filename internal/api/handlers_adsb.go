package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

// ADSBProvider is the read surface the adsb endpoint consumes.
// The daemon implements it on top of storage.AircraftLog; tests
// substitute a fake.
type ADSBProvider interface {
	RecentAircraftReports(limit int) ([]storage.AircraftReport, error)
	// CurrentAircraft returns the coalesced latest state per ICAO seen
	// within maxAge (≤ 0 → provider default).
	CurrentAircraft(maxAge time.Duration) ([]storage.AircraftReport, error)
}

// AircraftReportDTO is the JSON wire shape for the adsb endpoint.
// Position fields, identification fields, and velocity fields
// stay omitted from the JSON when zero / empty so the wire stays
// compact for the kind-specific common case.
type AircraftReportDTO struct {
	ID         int64     `json:"id"`
	ReceivedAt time.Time `json:"received_at"`
	ICAO       uint32    `json:"icao"`
	ICAOHex    string    `json:"icao_hex"`
	Kind       string    `json:"kind"`
	Body       string    `json:"body,omitempty"`
	CRCValid   bool      `json:"crc_valid"`

	Callsign string `json:"callsign,omitempty"`
	Category int    `json:"category,omitempty"`

	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
	Altitude    int     `json:"altitude_ft,omitempty"`
	HasPosition bool    `json:"has_position"`
	HasAltitude bool    `json:"has_altitude"`

	GroundSpeedKn   int     `json:"ground_speed_kn,omitempty"`
	TrackDeg        float64 `json:"track_deg,omitempty"`
	VerticalRateFPM int     `json:"vertical_rate_fpm,omitempty"`

	RawHex string `json:"raw_hex,omitempty"`
}

func aircraftReportToDTO(r storage.AircraftReport) AircraftReportDTO {
	return AircraftReportDTO{
		ID:              r.ID,
		ReceivedAt:      r.ReceivedAt,
		ICAO:            r.ICAO,
		ICAOHex:         r.ICAOHex,
		Kind:            r.Kind,
		Body:            r.Body,
		CRCValid:        r.CRCValid,
		Callsign:        r.Callsign,
		Category:        r.Category,
		Latitude:        r.Latitude,
		Longitude:       r.Longitude,
		Altitude:        r.Altitude,
		HasPosition:     r.HasPosition,
		HasAltitude:     r.HasAltitude,
		GroundSpeedKn:   r.GroundSpeedKn,
		TrackDeg:        r.TrackDeg,
		VerticalRateFPM: r.VerticalRateFPM,
		RawHex:          r.RawHex,
	}
}

// handleADSBAircraft answers GET /api/v1/adsb/aircraft. Optional
// ?limit= (default 200, max 5000). 503 when the storage layer
// isn't wired.
func (s *Server) handleADSBAircraft(w http.ResponseWriter, r *http.Request) {
	if s.adsb == nil {
		s.writeError(w, http.StatusServiceUnavailable, "adsb subsystem not enabled")
		return
	}
	limit := 200
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	rows, err := s.adsb.RecentAircraftReports(limit)
	if err != nil {
		s.log.Error("api: adsb aircraft", "err", err)
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	out := make([]AircraftReportDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, aircraftReportToDTO(r))
	}
	writeJSON(w, http.StatusOK, out)
}

// handleADSBAircraftCurrent answers GET /api/v1/adsb/aircraft/current —
// the coalesced latest state of each aircraft seen recently, one row
// per ICAO (the "currently visible aircraft" view). Optional
// ?max_age_s= bounds how long since last-seen an aircraft stays listed
// (default 300 s, max 3600 s).
func (s *Server) handleADSBAircraftCurrent(w http.ResponseWriter, r *http.Request) {
	if s.adsb == nil {
		s.writeError(w, http.StatusServiceUnavailable, "adsb subsystem not enabled")
		return
	}
	maxAge := 300 * time.Second
	if v := r.URL.Query().Get("max_age_s"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 3600 {
				n = 3600
			}
			maxAge = time.Duration(n) * time.Second
		}
	}
	rows, err := s.adsb.CurrentAircraft(maxAge)
	if err != nil {
		s.log.Error("api: adsb aircraft current", "err", err)
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	out := make([]AircraftReportDTO, 0, len(rows))
	for _, rr := range rows {
		out = append(out, aircraftReportToDTO(rr))
	}
	writeJSON(w, http.StatusOK, out)
}
