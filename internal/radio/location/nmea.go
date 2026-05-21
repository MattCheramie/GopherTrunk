// Package location decodes geographic positions reported over the air
// by trunked-radio subscriber units — P25 Motorola Unit GPS, L3Harris
// Talker GPS, and DMR LRRP all ultimately carry a latitude/longitude
// pair, and several (Tait CCDI, many MOTOTRBO GPS profiles) transport
// it as a verbatim NMEA-0183 sentence.
//
// This file implements a strict NMEA-0183 parser for the two sentence
// types that carry a fix — GGA and RMC. It is the protocol-agnostic
// core the per-protocol decoders feed once they have extracted the
// text/data payload of a GPS message.
package location

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Position is a decoded geographic fix. Latitude is positive north,
// Longitude positive east, both in decimal degrees.
type Position struct {
	Latitude   float64
	Longitude  float64
	SpeedKnots float64
	HeadingDeg float64
	HasSpeed   bool
	HasHeading bool
}

// Valid reports whether the fix is within the geographic coordinate
// range. A (0,0) fix is treated as invalid — it is the overwhelmingly
// common "no fix yet" placeholder, and Null Island is in open ocean.
func (p Position) Valid() bool {
	if p.Latitude == 0 && p.Longitude == 0 {
		return false
	}
	return p.Latitude >= -90 && p.Latitude <= 90 &&
		p.Longitude >= -180 && p.Longitude <= 180
}

// ErrNoFix is returned when a sentence parses cleanly but carries no
// usable position (empty coordinate fields, or an RMC "void" status).
var ErrNoFix = errors.New("location: sentence carries no position fix")

// ParseNMEA decodes a single NMEA-0183 sentence. GGA and RMC sentences
// (any talker ID — GP, GN, GL, …) yield a Position; the optional
// trailing "*hh" checksum is verified when present.
func ParseNMEA(sentence string) (Position, error) {
	s := strings.TrimSpace(sentence)
	if !strings.HasPrefix(s, "$") {
		return Position{}, errors.New("location: NMEA sentence must start with '$'")
	}
	s = s[1:]

	if star := strings.IndexByte(s, '*'); star >= 0 {
		body := s[:star]
		sum := strings.TrimSpace(s[star+1:])
		if len(sum) >= 2 {
			want, err := strconv.ParseUint(sum[:2], 16, 8)
			if err != nil {
				return Position{}, fmt.Errorf("location: bad NMEA checksum %q", sum)
			}
			if got := nmeaChecksum(body); byte(want) != got {
				return Position{}, fmt.Errorf("location: NMEA checksum mismatch (got %02X, want %02X)", got, want)
			}
		}
		s = body
	}

	fields := strings.Split(s, ",")
	if len(fields) == 0 || len(fields[0]) < 5 {
		return Position{}, errors.New("location: malformed NMEA sentence")
	}
	switch fields[0][2:] {
	case "GGA":
		return parseGGA(fields)
	case "RMC":
		return parseRMC(fields)
	default:
		return Position{}, fmt.Errorf("location: unsupported NMEA sentence %q", fields[0])
	}
}

// parseGGA decodes a GGA (fix data) sentence: lat in field 2, N/S in
// 3, lon in 4, E/W in 5.
func parseGGA(f []string) (Position, error) {
	if len(f) < 6 {
		return Position{}, errors.New("location: short GGA sentence")
	}
	lat, lon, err := decodeLatLon(f[2], f[3], f[4], f[5])
	if err != nil {
		return Position{}, err
	}
	return Position{Latitude: lat, Longitude: lon}, nil
}

// parseRMC decodes an RMC (recommended minimum) sentence: status in
// field 2 ('A' valid / 'V' void), lat 3, N/S 4, lon 5, E/W 6, speed
// (knots) 7, course 8.
func parseRMC(f []string) (Position, error) {
	if len(f) < 9 {
		return Position{}, errors.New("location: short RMC sentence")
	}
	if strings.EqualFold(f[2], "V") {
		return Position{}, ErrNoFix
	}
	lat, lon, err := decodeLatLon(f[3], f[4], f[5], f[6])
	if err != nil {
		return Position{}, err
	}
	p := Position{Latitude: lat, Longitude: lon}
	if f[7] != "" {
		if spd, err := strconv.ParseFloat(f[7], 64); err == nil {
			p.SpeedKnots = spd
			p.HasSpeed = true
		}
	}
	if f[8] != "" {
		if hdg, err := strconv.ParseFloat(f[8], 64); err == nil {
			p.HeadingDeg = hdg
			p.HasHeading = true
		}
	}
	return p, nil
}

// decodeLatLon converts the NMEA ddmm.mmmm / dddmm.mmmm coordinate
// pair plus hemisphere letters into signed decimal degrees.
func decodeLatLon(latStr, ns, lonStr, ew string) (float64, float64, error) {
	if latStr == "" || lonStr == "" {
		return 0, 0, ErrNoFix
	}
	lat, err := nmeaDegrees(latStr, 2)
	if err != nil {
		return 0, 0, fmt.Errorf("location: latitude: %w", err)
	}
	lon, err := nmeaDegrees(lonStr, 3)
	if err != nil {
		return 0, 0, fmt.Errorf("location: longitude: %w", err)
	}
	if strings.EqualFold(ns, "S") {
		lat = -lat
	}
	if strings.EqualFold(ew, "W") {
		lon = -lon
	}
	return lat, lon, nil
}

// nmeaDegrees converts an NMEA "(d)ddmm.mmmm" string to decimal
// degrees. degDigits is the count of leading degree digits (2 for
// latitude, 3 for longitude).
func nmeaDegrees(s string, degDigits int) (float64, error) {
	if len(s) < degDigits {
		return 0, fmt.Errorf("coordinate %q too short", s)
	}
	deg, err := strconv.ParseFloat(s[:degDigits], 64)
	if err != nil {
		return 0, err
	}
	min, err := strconv.ParseFloat(s[degDigits:], 64)
	if err != nil {
		return 0, err
	}
	return deg + min/60, nil
}

// nmeaChecksum XORs every byte of an NMEA sentence body (the text
// between '$' and '*').
func nmeaChecksum(body string) byte {
	var c byte
	for i := 0; i < len(body); i++ {
		c ^= body[i]
	}
	return c
}
