package adsb

import (
	"fmt"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

// ProcessFrame decodes a raw Mode-S frame (7 or 14 bytes), gates it on
// CRC, updates the CPR tracker for even/odd position pairing, and
// returns the storage.AircraftReport to publish.
//
// ok is false for frames that should be dropped: a CRC failure on
// anything but an all-call reply (DF 11) can't be trusted — extended
// squitters with a bad CRC carry unreliable payloads, while an
// all-call reply carries only the ICAO and is still useful for
// "this aircraft is in range" tracking.
//
// This is the single decode → track → report path shared by the BEAST
// upstream client and the native PPM receiver, so both sources produce
// byte-for-byte identical AircraftReports. Pass a non-nil tracker to
// get global CPR positions; pass nil to skip pairing.
func ProcessFrame(frame []byte, tracker *Tracker, now time.Time) (storage.AircraftReport, bool) {
	m := Decode(frame)
	if !m.CRCValid && m.Kind != KindAllCall {
		return storage.AircraftReport{}, false
	}
	if tracker != nil {
		m, _ = tracker.Update(m, now.UnixNano())
	}
	return BuildReport(m, now), true
}

// BuildReport maps a decoded Message into the storage.AircraftReport
// the bus carries. Exported for callers that have already decoded /
// tracked a Message themselves.
func BuildReport(m Message, now time.Time) storage.AircraftReport {
	rep := storage.AircraftReport{
		ReceivedAt: now,
		ICAO:       m.ICAO,
		ICAOHex:    fmt.Sprintf("%06X", m.ICAO),
		Kind:       KindString(m.Kind),
		Body:       m.String(),
		CRCValid:   m.CRCValid,
		RawHex:     m.RawHex,
	}
	if m.Identification != nil {
		rep.Callsign = m.Identification.Callsign
		rep.Category = m.Identification.Category
	}
	if m.Position != nil {
		rep.HasAltitude = m.Position.HasAltitude
		rep.Altitude = m.Position.Altitude
		if m.Position.HasGlobalPosition {
			rep.HasPosition = true
			rep.Latitude = m.Position.Latitude
			rep.Longitude = m.Position.Longitude
		}
	}
	if m.Velocity != nil {
		if m.Velocity.HasGroundSpeed {
			rep.GroundSpeedKn = m.Velocity.GroundSpeedKn
			rep.TrackDeg = m.Velocity.TrackDeg
		}
		if m.Velocity.HasVerticalRate {
			rep.VerticalRateFPM = m.Velocity.VerticalRateFPM
		}
	}
	return rep
}
