// Package adsb decodes Mode S extended-squitter messages
// transmitted by aircraft transponders on 1090 MHz. ADS-B
// (Automatic Dependent Surveillance — Broadcast) is the
// ICAO-mandated cooperative aviation surveillance protocol that
// every commercial passenger flight, most general-aviation, and
// all military aircraft over US / EU airspace continuously
// broadcasts. The data — ICAO 24-bit address, callsign, position,
// altitude, ground speed, vertical rate, heading — feeds every
// public flight-tracking service (FlightRadar24, FlightAware,
// adsb.lol, OpenSky) and is free wide-area aircraft data
// GopherTrunk now has the protocol layer to decode.
//
// Each ADS-B message is a 56- or 112-bit Mode S frame. This
// package handles the 112-bit extended-squitter formats
// (DF 17 = ADS-B from ICAO-addressed transponder, DF 18 = non-
// transponder ADS-B equipment) — the operator-visible majority —
// plus the simpler 56-bit DF 11 all-call reply, and tags the
// less-common DFs (4, 5, 20, 21) with the raw payload preserved.
//
// Spec references:
//   - ICAO Annex 10 Volume IV (Aeronautical Telecommunications —
//     Surveillance and Collision Avoidance Systems), Chapter 3
//     (Mode-S).
//   - RTCA DO-260B / EUROCAE ED-102A — ADS-B 1090 ES Minimum
//     Operational Performance Standards.
//   - https://mode-s.org/decode/ — the de-facto reference parser
//     documentation, cross-checked against real on-the-air
//     payloads.
//
// Mode-S bit ordering is MSB-first throughout: bits[0] is the
// MSB of byte 0, fields read MSB-first from absolute bit
// positions. The CRC-24 polynomial is 0xFFF409 and is
// XOR-overlaid with the ICAO address for DF 17 / 18 frames
// (the polynomial is computed over the message bits and the
// trailing 24 bits hold message_crc XOR icao_address — so a
// receiver can recover the ICAO address by computing the CRC
// over the message and XORing with the trailing 24 bits).
package adsb

import "fmt"

// DownlinkFormat is the 5-bit DF code at the start of every
// Mode-S transmission. Identifies the message family.
type DownlinkFormat uint8

const (
	DFShortAirAir         DownlinkFormat = 0  // Short air-air surveillance reply (ACAS)
	DFAltitudeReply       DownlinkFormat = 4  // Surveillance altitude reply
	DFIdentityReply       DownlinkFormat = 5  // Surveillance identity reply
	DFAllCallReply        DownlinkFormat = 11 // All-call reply (ICAO address + capability)
	DFLongAirAir          DownlinkFormat = 16 // Long air-air surveillance
	DFExtendedSquitter    DownlinkFormat = 17 // ADS-B (ICAO-addressed transponder)
	DFExtendedSquitterAlt DownlinkFormat = 18 // ADS-B (non-transponder equipment)
	DFMilitaryExtSquitter DownlinkFormat = 19 // Military extended squitter
	DFCommBAltitudeReply  DownlinkFormat = 20 // Comm-B altitude reply
	DFCommBIdentityReply  DownlinkFormat = 21 // Comm-B identity reply
	DFCommDExtendedLength DownlinkFormat = 24 // Comm-D (extended length, DF 24-31 all map here)
)

// TypeCode is the 5-bit type field at the start of an extended-
// squitter ME payload (bits 32..36 of a DF 17/18 frame). Picks
// the payload format.
type TypeCode uint8

const (
	TCAircraftID1      TypeCode = 1 // Identification, Category D
	TCAircraftID2      TypeCode = 2 // Identification, Category C
	TCAircraftID3      TypeCode = 3 // Identification, Category B
	TCAircraftID4      TypeCode = 4 // Identification, Category A
	TCSurfacePos1      TypeCode = 5 // Surface position
	TCSurfacePos2      TypeCode = 6
	TCSurfacePos3      TypeCode = 7
	TCSurfacePos4      TypeCode = 8
	TCAirbornePos1     TypeCode = 9 // Airborne position, barometric altitude
	TCAirbornePos2     TypeCode = 10
	TCAirbornePos3     TypeCode = 11
	TCAirbornePos4     TypeCode = 12
	TCAirbornePos5     TypeCode = 13
	TCAirbornePos6     TypeCode = 14
	TCAirbornePos7     TypeCode = 15
	TCAirbornePos8     TypeCode = 16
	TCAirbornePos9     TypeCode = 17
	TCAirbornePos10    TypeCode = 18
	TCAirborneVel      TypeCode = 19 // Airborne velocity (ground speed + track or air speed + heading)
	TCAirbornePosGNSS  TypeCode = 20 // Airborne position, GNSS height
	TCAirbornePosGNSS2 TypeCode = 21
	TCAirbornePosGNSS3 TypeCode = 22
	TCAircraftStatus   TypeCode = 28 // Aircraft status (emergency / 1090 transmit)
	TCTargetState      TypeCode = 29 // Target state and status
	TCAircraftOpStatus TypeCode = 31 // Aircraft operational status
)

// PayloadKind summarises what the extended-squitter type code
// resolves to. Used as the column value on the aircraft_log
// SQLite table.
type PayloadKind uint8

const (
	KindUnknown PayloadKind = iota
	KindIdentification
	KindSurfacePosition
	KindAirbornePosition
	KindAirborneVelocity
	KindAircraftStatus
	KindTargetState
	KindOperationalStatus
	KindAllCall
)

// KindString returns the stable lowercase label.
func KindString(k PayloadKind) string {
	switch k {
	case KindIdentification:
		return "ident"
	case KindSurfacePosition:
		return "surface-pos"
	case KindAirbornePosition:
		return "airborne-pos"
	case KindAirborneVelocity:
		return "velocity"
	case KindAircraftStatus:
		return "status"
	case KindTargetState:
		return "target-state"
	case KindOperationalStatus:
		return "op-status"
	case KindAllCall:
		return "all-call"
	}
	return "unknown"
}

// kindFromTC maps a TypeCode to a PayloadKind.
func kindFromTC(tc TypeCode) PayloadKind {
	switch {
	case tc >= 1 && tc <= 4:
		return KindIdentification
	case tc >= 5 && tc <= 8:
		return KindSurfacePosition
	case tc >= 9 && tc <= 18:
		return KindAirbornePosition
	case tc == 19:
		return KindAirborneVelocity
	case tc >= 20 && tc <= 22:
		return KindAirbornePosition
	case tc == 28:
		return KindAircraftStatus
	case tc == 29:
		return KindTargetState
	case tc == 31:
		return KindOperationalStatus
	}
	return KindUnknown
}

// Identification is the decoded callsign payload from a TC 1-4
// extended-squitter message. The callsign is 8 ASCII chars,
// packed 6-bits-per-char in the ME field, decoded via the ICAO
// alphabet (M.1371-style but distinct: A-Z = 1-26, space = 32,
// 0-9 = 48-57). Trailing spaces / underscores are stripped.
type Identification struct {
	Category int    // wake-vortex / size category (A1..D7)
	Callsign string // 8-char airline code + flight number ("UAL123 ", "N12345  ")
}

// Position is the decoded lat/lon for an airborne or surface
// position message. CPR (Compact Position Reporting) gives the
// position in two halves — even and odd encodings; the caller
// pairs them within ~10 s to recover the global position.
// The bit-stream layer surfaces the raw CPR encoded values + the
// format flag; the higher-level Tracker pairs even/odd halves
// and populates Latitude / Longitude / HasGlobalPosition.
type Position struct {
	// Even-format CPR raw lat / lon (17 bits each) — populated
	// when CPRFormat == 0.
	CPRLatEven int
	CPRLonEven int
	// Odd-format CPR raw lat / lon — populated when CPRFormat == 1.
	CPRLatOdd int
	CPRLonOdd int

	// CPRFormat: 0 = even-encoded message, 1 = odd-encoded.
	CPRFormat int

	// Latitude / Longitude in degrees, populated by the Tracker
	// when the per-ICAO even+odd CPR pair completes inside the
	// spec's 10 s window. HasGlobalPosition distinguishes
	// "decoded" from "raw CPR halves preserved but not yet
	// paired".
	Latitude          float64
	Longitude         float64
	HasGlobalPosition bool

	// Altitude in feet (or 0 if the message carried the "altitude
	// not available" sentinel). HasAltitude distinguishes "0 ft"
	// (rare but valid for surface) from "not present".
	Altitude    int
	HasAltitude bool

	// SurveillanceStatus, NICSupplementB are spec fields that
	// affect position-decoding semantics. Surfaced for callers
	// that want to render the spec metadata.
	SurveillanceStatus int
	NICSupplementB     int
}

// Velocity is the decoded airborne velocity payload from a TC 19
// message. Subtypes 1 + 2 carry ground speed + track; subtypes 3
// + 4 carry air speed + heading. The struct flattens both — the
// HasGroundSpeed / HasAirSpeed flags say which is populated.
type Velocity struct {
	// Ground-speed subtype (sub 1 / 2). Speed in knots.
	GroundSpeedKn  int
	TrackDeg       float64
	HasGroundSpeed bool

	// Air-speed subtype (sub 3 / 4). Speed in knots, heading
	// in degrees (0..359.x with 0.5 resolution).
	AirSpeedKn  int
	HeadingDeg  float64
	HasAirSpeed bool

	// Vertical rate in feet per minute. Sign: positive = climb,
	// negative = descent.
	VerticalRateFPM int
	HasVerticalRate bool

	// VerticalRateSource: 0 = GNSS, 1 = barometric.
	VerticalRateSource int
}

// Message is one decoded Mode-S frame.
type Message struct {
	DF       DownlinkFormat
	ICAO     uint32      // 24-bit aircraft address (DF 11 / 17 / 18 always carry it; computable for DF 4 / 5 / 20 / 21 via CRC overlay)
	TypeCode TypeCode    // extended-squitter type code (DF 17 / 18 only); 0 otherwise
	Kind     PayloadKind // semantic label
	CRCValid bool        // CRC matched (DF 11 / 17 / 18), or — for an address/parity frame — the recovered ICAO was recently seen in one that did (ProcessFrame)
	// ICAOInferred is set on DF 0 / 4 / 5 / 16 / 20 / 21, whose trailing
	// 24 bits are CRC XOR ICAO: the address is recovered by assuming the
	// frame is error-free, so there is no check at all in the frame
	// itself — any 56/112 bits of noise "decode" to some address. Such a
	// frame is only trustworthy when its ICAO was recently announced by a
	// CRC-checked DF 11 / 17 / 18 (the dump1090 / readsb rule), which is
	// the gate ProcessFrame applies (#1316).
	ICAOInferred bool
	// IID is the interrogator identifier a DF 11 all-call reply was
	// addressed to (the low 7 bits of its parity field); 0 for the
	// spontaneous acquisition squitter.
	IID uint8

	// Per-kind decoded payload — populated based on Kind.
	Identification *Identification
	Position       *Position
	Velocity       *Velocity

	// Raw 112-bit (or 56-bit) frame as hex. Round-trips into
	// raw_hex on the aircraft_log row for debugging types this
	// package doesn't fully decode.
	RawHex string
}

// Decode parses one Mode-S frame and returns a typed Message. The
// input is 7 or 14 bytes (56-bit short or 112-bit long Mode-S).
//
// Never returns an error — short / malformed frames come back
// with Kind = KindUnknown and the raw bytes hex-encoded on
// RawHex. ADS-B receivers see a flood of half-frames at low
// SNR; surface what we can and pass through the rest.
func Decode(frame []byte) Message {
	m := Message{RawHex: bytesToHex(frame)}
	if len(frame) != 7 && len(frame) != 14 {
		return m
	}
	m.DF = DownlinkFormat(frame[0] >> 3)

	// CRC + ICAO recovery — the trailing 24 bits hold the CRC
	// for DF 11 / 17 / 18 (no XOR overlay; CRC checks directly).
	// For DF 0 / 4 / 5 / 20 / 21 the trailing 24 bits = CRC XOR
	// ICAO, so we can XOR our computed CRC with the trailing 24
	// bits to recover the ICAO address (no way to verify
	// correctness without a separate ICAO whitelist).
	if len(frame) == 14 {
		computed := crc24(frame[:11])
		stored := uint32(frame[11])<<16 | uint32(frame[12])<<8 | uint32(frame[13])
		switch m.DF {
		case DFExtendedSquitter, DFExtendedSquitterAlt:
			m.CRCValid = computed == stored
			if m.CRCValid {
				m.ICAO = uint32(frame[1])<<16 | uint32(frame[2])<<8 | uint32(frame[3])
			}
		case DFLongAirAir, DFCommBAltitudeReply, DFCommBIdentityReply:
			m.ICAO = computed ^ stored
			m.ICAOInferred = true
		}
	} else { // 56-bit (DF 0, 4, 5, 11)
		computed := crc24(frame[:4])
		stored := uint32(frame[4])<<16 | uint32(frame[5])<<8 | uint32(frame[6])
		switch m.DF {
		case DFAllCallReply:
			// A DF 11 reply to a selective interrogation carries the
			// interrogator's 7-bit IID XORed into the low bits of its
			// parity (Annex 10 Vol IV §3.1.2.5.2.1.3); the spontaneous
			// squitter has IID 0. A straight compare dropped every
			// addressed reply (#1316).
			if diff := computed ^ stored; diff&^0x7F == 0 {
				m.CRCValid = true
				m.IID = uint8(diff)
				m.ICAO = uint32(frame[1])<<16 | uint32(frame[2])<<8 | uint32(frame[3])
				m.Kind = KindAllCall
			}
		case DFShortAirAir, DFAltitudeReply, DFIdentityReply:
			m.ICAO = computed ^ stored
			m.ICAOInferred = true
		}
	}

	// Extended-squitter ME payload dispatch (DF 17 / 18 only).
	if (m.DF == DFExtendedSquitter || m.DF == DFExtendedSquitterAlt) &&
		len(frame) == 14 && m.CRCValid {
		// ME = bits 32..87 (8 bytes, frame[4..11], with the type
		// code in the high 5 bits of frame[4]).
		me := frame[4:11]
		m.TypeCode = TypeCode(me[0] >> 3)
		m.Kind = kindFromTC(m.TypeCode)
		switch m.Kind {
		case KindIdentification:
			if id, ok := parseIdentification(me); ok {
				m.Identification = &id
			}
		case KindAirbornePosition:
			if pos, ok := parseAirbornePosition(me); ok {
				m.Position = &pos
			}
		case KindSurfacePosition:
			if pos, ok := parseSurfacePosition(me); ok {
				m.Position = &pos
			}
		case KindAirborneVelocity:
			if vel, ok := parseAirborneVelocity(me); ok {
				m.Velocity = &vel
			}
		}
	}
	return m
}

// String renders the message for log / panel display.
func (m Message) String() string {
	switch m.Kind {
	case KindIdentification:
		if m.Identification == nil {
			return fmt.Sprintf("IDENT %06X (parse failed)", m.ICAO)
		}
		return fmt.Sprintf("IDENT %06X %q cat=%d",
			m.ICAO, m.Identification.Callsign, m.Identification.Category)
	case KindAirbornePosition:
		if m.Position == nil {
			return fmt.Sprintf("AIRBORNE-POS %06X (parse failed)", m.ICAO)
		}
		alt := "alt=?"
		if m.Position.HasAltitude {
			alt = fmt.Sprintf("alt=%dft", m.Position.Altitude)
		}
		return fmt.Sprintf("AIRBORNE-POS %06X CPR-%s %s",
			m.ICAO, cprLabel(m.Position.CPRFormat), alt)
	case KindSurfacePosition:
		return fmt.Sprintf("SURFACE-POS %06X", m.ICAO)
	case KindAirborneVelocity:
		if m.Velocity == nil {
			return fmt.Sprintf("VELOCITY %06X (parse failed)", m.ICAO)
		}
		var s string
		if m.Velocity.HasGroundSpeed {
			s = fmt.Sprintf("VELOCITY %06X GS=%dkn track=%.1f°",
				m.ICAO, m.Velocity.GroundSpeedKn, m.Velocity.TrackDeg)
		} else if m.Velocity.HasAirSpeed {
			s = fmt.Sprintf("VELOCITY %06X TAS=%dkn hdg=%.1f°",
				m.ICAO, m.Velocity.AirSpeedKn, m.Velocity.HeadingDeg)
		} else {
			s = fmt.Sprintf("VELOCITY %06X", m.ICAO)
		}
		if m.Velocity.HasVerticalRate {
			s += fmt.Sprintf(" VR=%+dfpm", m.Velocity.VerticalRateFPM)
		}
		return s
	case KindAllCall:
		return fmt.Sprintf("ALL-CALL %06X", m.ICAO)
	}
	if m.ICAO != 0 {
		return fmt.Sprintf("%s %06X (DF=%d)", KindString(m.Kind), m.ICAO, m.DF)
	}
	return fmt.Sprintf("%s (DF=%d)", KindString(m.Kind), m.DF)
}

func cprLabel(format int) string {
	if format == 0 {
		return "even"
	}
	return "odd"
}
