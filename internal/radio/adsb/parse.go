package adsb

import (
	"encoding/hex"
	"math"
	"strings"
)

// parseIdentification decodes a TC 1-4 callsign payload from an
// 8-byte ME field. The callsign is 8 chars × 6 bits = 48 bits,
// stored in ME bits 8..55 (bytes 1..6 with the 6-bit alphabet
// described in DO-260B §2.2.3.2.5.2).
//
// The ICAO 6-bit alphabet: 1-26 = 'A'-'Z', 32 = space, 48-57
// = '0'-'9'. Unused / reserved code points fall back to '?'.
func parseIdentification(me []byte) (Identification, bool) {
	if len(me) < 7 {
		return Identification{}, false
	}
	id := Identification{Category: int(me[0] & 0x07)}
	// Pull the 48 bits from me[1..6] into one uint64 and chunk
	// out 8 × 6-bit characters.
	var bits uint64
	for i := 1; i <= 6; i++ {
		bits = bits<<8 | uint64(me[i])
	}
	var sb strings.Builder
	sb.Grow(8)
	for i := 7; i >= 0; i-- {
		code := (bits >> uint(i*6)) & 0x3F
		sb.WriteByte(callsignAlphabet[code])
	}
	id.Callsign = strings.TrimRight(sb.String(), " _")
	return id, true
}

// callsignAlphabet maps the 6-bit ICAO code to ASCII. Index 0 is
// the "unused" sentinel — surfaces as "_" so trim picks it up.
var callsignAlphabet = [64]byte{
	'_', 'A', 'B', 'C', 'D', 'E', 'F', 'G',
	'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O',
	'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W',
	'X', 'Y', 'Z', '_', '_', '_', '_', '_',
	' ', '_', '_', '_', '_', '_', '_', '_',
	'_', '_', '_', '_', '_', '_', '_', '_',
	'0', '1', '2', '3', '4', '5', '6', '7',
	'8', '9', '_', '_', '_', '_', '_', '_',
}

// parseAirbornePosition decodes a TC 9-18 (barometric altitude) or
// TC 20-22 (GNSS altitude) airborne-position ME field. Spec
// DO-260B §2.2.3.2.3 / ICAO Annex 10 IV §3.1.2.8.5.
//
// ME layout (8 bytes, 56 bits):
//
//	bits 0..4   = TC
//	bits 5..6   = surveillance status
//	bit  7      = NIC supplement-B (single-antenna flag)
//	bits 8..19  = encoded altitude (12 bits)
//	bit  20     = time-synchronisation flag
//	bit  21     = CPR format (0 = even, 1 = odd)
//	bits 22..38 = CPR latitude (17 bits)
//	bits 39..55 = CPR longitude (17 bits)
func parseAirbornePosition(me []byte) (Position, bool) {
	if len(me) < 7 {
		return Position{}, false
	}
	pos := Position{
		SurveillanceStatus: int((me[0] >> 1) & 0x03),
		NICSupplementB:     int(me[0] & 0x01),
	}
	// Altitude: 12 bits at bit positions 8..19.
	altRaw := (uint16(me[1]) << 4) | (uint16(me[2]) >> 4)
	if altRaw != 0 {
		pos.Altitude = decodeAltitude(altRaw)
		pos.HasAltitude = true
	}
	// CPR format flag (bit 21) — same byte that holds the
	// time-sync flag at bit 20.
	pos.CPRFormat = int((me[2] >> 2) & 0x01)
	// CPR latitude: 17 bits starting at bit 22.
	latRaw := (uint32(me[2]&0x03) << 15) |
		(uint32(me[3]) << 7) |
		(uint32(me[4]) >> 1)
	// CPR longitude: 17 bits starting at bit 39.
	lonRaw := (uint32(me[4]&0x01) << 16) |
		(uint32(me[5]) << 8) |
		uint32(me[6])

	if pos.CPRFormat == 0 {
		pos.CPRLatEven = int(latRaw)
		pos.CPRLonEven = int(lonRaw)
	} else {
		pos.CPRLatOdd = int(latRaw)
		pos.CPRLonOdd = int(lonRaw)
	}
	return pos, true
}

// parseSurfacePosition decodes a TC 5-8 surface-position ME field.
// Layout is similar to airborne-position but the 12-bit altitude
// slot holds a 7-bit movement (ground speed) and a 7-bit ground
// track instead. We surface the CPR values for now and leave the
// movement / track decode to a follow-up.
func parseSurfacePosition(me []byte) (Position, bool) {
	if len(me) < 7 {
		return Position{}, false
	}
	pos := Position{}
	pos.CPRFormat = int((me[2] >> 2) & 0x01)
	latRaw := (uint32(me[2]&0x03) << 15) |
		(uint32(me[3]) << 7) |
		(uint32(me[4]) >> 1)
	lonRaw := (uint32(me[4]&0x01) << 16) |
		(uint32(me[5]) << 8) |
		uint32(me[6])
	if pos.CPRFormat == 0 {
		pos.CPRLatEven = int(latRaw)
		pos.CPRLonEven = int(lonRaw)
	} else {
		pos.CPRLatOdd = int(latRaw)
		pos.CPRLonOdd = int(lonRaw)
	}
	return pos, true
}

// decodeAltitude turns the 12-bit altitude field into a value in
// feet. The "Q" bit (bit 4 of the raw field, position 8 in the
// ME bit layout) chooses between 25-ft Q=1 and 100-ft Q=0
// (Gillham-coded) resolution. For Q=1, the remaining 11 bits
// concatenate around the Q bit and decode as N where altitude =
// 25 * N - 1000.
func decodeAltitude(raw uint16) int {
	qBit := (raw >> 4) & 0x01
	if qBit == 1 {
		// 25 ft resolution: drop the Q bit and concatenate the
		// surrounding 11 bits.
		n := ((raw >> 5) << 4) | (raw & 0x0F)
		return int(n)*25 - 1000
	}
	// Gillham 100-ft resolution is rarely seen above 50,000 ft;
	// decoding it correctly needs the full Gray + Mode-A table.
	// For now return 0 and let the caller treat the Q=0 case as
	// "altitude not decoded" — DO-260B §2.2.3.2.3.4.3.
	return 0
}

// parseAirborneVelocity decodes a TC 19 ME field. Spec DO-260B
// §2.2.3.2.6. The subtype (ST, bits 5..7) picks ground-speed
// (1, 2) vs air-speed (3, 4) and supersonic flags.
func parseAirborneVelocity(me []byte) (Velocity, bool) {
	if len(me) < 7 {
		return Velocity{}, false
	}
	subtype := int(me[0] & 0x07)
	vel := Velocity{}

	switch subtype {
	case 1, 2:
		// Ground-speed subtypes.
		ewDir := int((me[1] >> 2) & 0x01) // 0 = E-W positive (eastward), 1 = westward
		ewVel := int(me[1]&0x03)<<8 | int(me[2])
		nsDir := int((me[3] >> 7) & 0x01) // 0 = northward, 1 = southward
		nsVel := int(me[3]&0x7F)<<3 | int(me[4]>>5)
		if ewVel > 0 && nsVel > 0 {
			ewVel-- // spec: value field is offset by 1 (0 = "not available")
			nsVel--
			if subtype == 2 {
				ewVel *= 4 // supersonic resolution
				nsVel *= 4
			}
			if ewDir == 1 {
				ewVel = -ewVel
			}
			if nsDir == 1 {
				nsVel = -nsVel
			}
			gs := isqrt(ewVel*ewVel + nsVel*nsVel)
			track := atan2Deg(float64(ewVel), float64(nsVel))
			vel.GroundSpeedKn = gs
			vel.TrackDeg = track
			vel.HasGroundSpeed = true
		}
	case 3, 4:
		// Air-speed subtypes (with TAS magnetic heading).
		headingStatus := int((me[1] >> 2) & 0x01) // 1 = heading valid
		if headingStatus == 1 {
			headingRaw := int(me[1]&0x03)<<8 | int(me[2])
			vel.HeadingDeg = float64(headingRaw) * 360.0 / 1024.0
		}
		airSpeedRaw := int(me[3]&0x7F)<<3 | int(me[4]>>5)
		if airSpeedRaw > 0 {
			airSpeedRaw--
			if subtype == 4 {
				airSpeedRaw *= 4
			}
			vel.AirSpeedKn = airSpeedRaw
			vel.HasAirSpeed = true
		}
	default:
		return Velocity{}, false
	}

	// Vertical rate (bits 36..50 of ME) — common to all subtypes.
	vel.VerticalRateSource = int((me[4] >> 4) & 0x01)
	vrDir := int((me[4] >> 3) & 0x01) // 0 = up, 1 = down
	vrRaw := int(me[4]&0x07)<<6 | int(me[5]>>2)
	if vrRaw > 0 {
		vr := (vrRaw - 1) * 64
		if vrDir == 1 {
			vr = -vr
		}
		vel.VerticalRateFPM = vr
		vel.HasVerticalRate = true
	}
	return vel, true
}

// isqrt is the integer square root via Newton's method — used to
// compute ground speed magnitude without pulling math.Sqrt for a
// hot path.
func isqrt(n int) int {
	if n <= 0 {
		return 0
	}
	x := n
	y := (x + 1) / 2
	for y < x {
		x = y
		y = (x + n/x) / 2
	}
	return x
}

// atan2Deg returns the angle of (ew, ns) in degrees 0..360 where
// 0 = North and bearings increase clockwise (standard aviation
// track convention). For ADS-B track precision math.Atan2 is
// well within tolerance.
func atan2Deg(ew, ns float64) float64 {
	if ew == 0 && ns == 0 {
		return 0
	}
	r := math.Atan2(ew, ns) * 180.0 / math.Pi
	if r < 0 {
		r += 360.0
	}
	return r
}

// crc24 computes the Mode-S CRC-24 (polynomial 0xFFF409). The CRC
// is computed over the message bytes (excluding the trailing
// 3-byte CRC field for verification; or over the full frame and
// expected to be 0 for clean reception).
// CRC24 is the Mode-S CRC-24 (polynomial 0xFFF409) over data, exported for
// tests and tools that build frames; Decode applies it itself.
func CRC24(data []byte) uint32 { return crc24(data) }

func crc24(data []byte) uint32 {
	const poly = 0xFFF409
	crc := uint32(0)
	for _, b := range data {
		crc ^= uint32(b) << 16
		for i := 0; i < 8; i++ {
			if crc&0x800000 != 0 {
				crc = ((crc << 1) ^ poly) & 0xFFFFFF
			} else {
				crc = (crc << 1) & 0xFFFFFF
			}
		}
	}
	return crc
}

// bytesToHex hex-encodes the frame for raw_hex logging. Lowercase.
func bytesToHex(frame []byte) string {
	if len(frame) == 0 {
		return ""
	}
	return hex.EncodeToString(frame)
}
