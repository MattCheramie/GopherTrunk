// Package aprs decodes the APRS info-field payload that rides on
// top of AX.25 (the link layer in package aprs/ax25). APRS is a
// huge, semi-formal protocol — the spec covers position, weather,
// message, telemetry, status, object, item, query, and bulletin
// formats, plus all the trailing-comment compressed variants.
//
// This package handles the operator-visible majority: position
// reports (uncompressed lat/lon-NMEA-style), messages, status, and
// the catch-all printable-info case. Each decoder produces a
// strongly-typed Go struct so the storage + UI layers can render
// it without re-parsing.
//
// Spec references:
//
//   - APRS Protocol Reference 1.0.1 (1998-08-07) — the canonical
//     text. http://www.aprs.org/doc/APRS101.PDF
//   - aprs.fi parser source code as cross-check for the messy
//     real-world variants the spec doesn't quite pin down.
package aprs

import (
	"fmt"
	"strconv"
	"strings"
)

// PacketType is what the info field's first byte / leading
// pattern identifies the payload as. APRS uses an inline data-type
// indicator (DTI) byte for everything except the comment-only
// case; the parser dispatches on it.
type PacketType uint8

const (
	TypeUnknown   PacketType = iota
	TypePosition             // "!" (no msg) / "=" (with msg) / "/" / "@" prefixed lat/lon
	TypeStatus               // ">" prefixed free-text
	TypeMessage              // ":" addressee + ":" body
	TypeObject               // ";" — like position but for a named object
	TypeMicE                 // 0x1C / 0x1D / "'" / "`" — Mic-E compressed (decoder is in the follow-up)
	TypeWeather              // "_" / "#" / "*" — full weather report
	TypeTelemetry            // "T#" prefix
	TypeBulletin             // ":BLNxxxxx:" — addressed bulletin board
)

// Position is the decoded latitude / longitude payload from an
// uncompressed position report. APRS encodes latitude as DDMM.mmH
// where H is N/S, longitude as DDDMM.mmH where H is E/W.
type Position struct {
	Latitude  float64
	Longitude float64
	// Symbol is the APRS symbol table + char (e.g. "/>" for a car,
	// "/k" for a truck, "\>" for a vehicle from the alternate
	// table). Two ASCII bytes.
	SymbolTable byte
	SymbolCode  byte
	// Comment is everything after the symbol char.
	Comment string
	// HasTimestamp is true when the report carried a leading
	// 7-character timestamp (formats "@" and "/"). Stored
	// unparsed in TimestampRaw to avoid the timezone heuristics
	// — the bus event carries received-time anyway.
	HasTimestamp  bool
	TimestampRaw  string
	WithMessaging bool // true for "=" and "@" (messaging-capable)
}

// Message is the decoded payload for the message format:
// ":ADDRESSEE :body{seqno}".
type Message struct {
	Addressee string // 1..9 chars, trimmed of trailing spaces
	Body      string
	// SeqNo is the optional ack number when the body ends with
	// "{NNN}". Empty when not present.
	SeqNo string
	// Ack reports whether this is an "ack" / "rej" rather than a
	// message body. The body shape is "ack123" / "rej123".
	Ack bool
	Rej bool
}

// Status is the decoded payload for the status format: ">body".
type Status struct {
	Text string
}

// Bulletin is the decoded payload for the bulletin format:
// ":BLNxxxxxx:body".
type Bulletin struct {
	ID   string
	Body string
}

// Packet is the decoded result of one APRS info field.
type Packet struct {
	Type     PacketType
	Position *Position
	Message  *Message
	Status   *Status
	Bulletin *Bulletin
	// Raw is the original info field; always populated so the UI
	// can show the literal bytes when the decoder picks Unknown.
	Raw string
}

// Decode parses one APRS info field and returns a typed Packet.
// Never returns an error — unknown / malformed payloads come
// back as Type=TypeUnknown with the raw bytes preserved on the
// Raw field. This matches operator expectations (APRS is messy;
// we surface what we can and pass through the rest).
func Decode(info []byte) Packet {
	raw := string(info)
	p := Packet{Raw: raw}
	if len(info) == 0 {
		return p
	}
	dti := info[0]
	switch dti {
	case '!':
		// Uncompressed position without timestamp, no messaging.
		if pos, ok := parseUncompressedPosition(string(info[1:]), false, false); ok {
			p.Type = TypePosition
			p.Position = &pos
			return p
		}
	case '=':
		// Uncompressed position without timestamp, messaging-capable.
		if pos, ok := parseUncompressedPosition(string(info[1:]), false, true); ok {
			p.Type = TypePosition
			p.Position = &pos
			return p
		}
	case '/':
		// Uncompressed position with timestamp, no messaging.
		if len(info) >= 8 {
			ts := string(info[1:8])
			if pos, ok := parseUncompressedPosition(string(info[8:]), true, false); ok {
				pos.HasTimestamp = true
				pos.TimestampRaw = ts
				p.Type = TypePosition
				p.Position = &pos
				return p
			}
		}
	case '@':
		// Uncompressed position with timestamp, messaging-capable.
		if len(info) >= 8 {
			ts := string(info[1:8])
			if pos, ok := parseUncompressedPosition(string(info[8:]), true, true); ok {
				pos.HasTimestamp = true
				pos.TimestampRaw = ts
				p.Type = TypePosition
				p.Position = &pos
				return p
			}
		}
	case ':':
		// Message — including bulletins and acks.
		if msg, bln, isBulletin, ok := parseMessage(string(info[1:])); ok {
			if isBulletin {
				p.Type = TypeBulletin
				p.Bulletin = bln
			} else {
				p.Type = TypeMessage
				p.Message = msg
			}
			return p
		}
	case '>':
		p.Type = TypeStatus
		p.Status = &Status{Text: string(info[1:])}
		return p
	case ';':
		p.Type = TypeObject
		return p
	case 0x1C, 0x1D, '`', '\'':
		// Mic-E — the decoder is involved enough that it lives in
		// its own file. For now, surface the type so the UI can
		// render "Mic-E (compressed)" rather than dropping the
		// packet.
		p.Type = TypeMicE
		return p
	case '_':
		p.Type = TypeWeather
		return p
	case 'T':
		if len(info) >= 2 && info[1] == '#' {
			p.Type = TypeTelemetry
			return p
		}
	}
	return p
}

// parseUncompressedPosition decodes the "DDMM.mmH/DDDMM.mmH"
// section that follows the data-type indicator (and optional
// timestamp). On success returns the populated Position and
// true; on malformed input returns ok=false so the caller can
// surface the packet as TypeUnknown.
func parseUncompressedPosition(s string, _ bool, withMsg bool) (Position, bool) {
	// Minimum length: 8 (lat) + 1 (sym table) + 9 (lon) + 1 (sym) = 19.
	if len(s) < 19 {
		return Position{}, false
	}
	latStr := s[0:8] // DDMM.mmH
	symTable := s[8]
	lonStr := s[9:18] // DDDMM.mmH
	symCode := s[18]
	comment := s[19:]

	lat, ok := parseLatLon(latStr, true)
	if !ok {
		return Position{}, false
	}
	lon, ok := parseLatLon(lonStr, false)
	if !ok {
		return Position{}, false
	}
	return Position{
		Latitude:      lat,
		Longitude:     lon,
		SymbolTable:   symTable,
		SymbolCode:    symCode,
		Comment:       comment,
		WithMessaging: withMsg,
	}, true
}

// parseLatLon decodes a DDMM.mmH (latitude) or DDDMM.mmH
// (longitude) string into decimal degrees. Negative south /
// west. APRS supports "ambiguity spaces" — bytes that should be
// digits but are space, indicating reduced precision — which
// this implementation tolerates by treating spaces as 0.
func parseLatLon(s string, isLat bool) (float64, bool) {
	wantLen := 8 // DDMM.mmH
	degDigits := 2
	if !isLat {
		wantLen = 9 // DDDMM.mmH
		degDigits = 3
	}
	if len(s) != wantLen {
		return 0, false
	}
	// Replace ambiguity spaces with 0 — they only ever appear in
	// the minutes' fractional digits, but conservatively treat any
	// of the numeric positions.
	clean := []byte(s)
	for i := 0; i < degDigits+4; i++ { // DD or DDD then MM.mm
		if clean[i] == ' ' {
			clean[i] = '0'
		}
	}
	// Skip the "." separator — it lives at index degDigits+2.
	if clean[degDigits+2] != '.' {
		return 0, false
	}
	deg, err := strconv.ParseFloat(string(clean[:degDigits]), 64)
	if err != nil {
		return 0, false
	}
	min, err := strconv.ParseFloat(string(clean[degDigits:degDigits+2])+"."+string(clean[degDigits+3:degDigits+5]), 64)
	if err != nil {
		return 0, false
	}
	val := deg + min/60
	hemi := clean[wantLen-1]
	if isLat {
		switch hemi {
		case 'N':
		case 'S':
			val = -val
		default:
			return 0, false
		}
	} else {
		switch hemi {
		case 'E':
		case 'W':
			val = -val
		default:
			return 0, false
		}
	}
	return val, true
}

// parseMessage decodes the ":ADDRESSEE :body{seqno}" payload.
// Returns (message, bulletin, isBulletin, ok). The bulletin
// branch fires when the addressee starts with "BLN".
func parseMessage(s string) (*Message, *Bulletin, bool, bool) {
	// AX.25 message format: 9-char addressee + ':' + body.
	if len(s) < 10 || s[9] != ':' {
		return nil, nil, false, false
	}
	addr := strings.TrimRight(s[0:9], " ")
	body := s[10:]
	// Bulletin?
	if strings.HasPrefix(addr, "BLN") {
		return nil, &Bulletin{ID: addr, Body: body}, true, true
	}
	msg := &Message{Addressee: addr, Body: body}
	// Strip optional "{seq}" suffix.
	if i := strings.LastIndex(body, "{"); i >= 0 && strings.HasSuffix(body, "}") {
		msg.SeqNo = body[i+1 : len(body)-1]
		msg.Body = body[:i]
	}
	// Detect ack / rej.
	if strings.HasPrefix(msg.Body, "ack") {
		msg.Ack = true
		msg.SeqNo = msg.Body[3:]
		msg.Body = ""
	} else if strings.HasPrefix(msg.Body, "rej") {
		msg.Rej = true
		msg.SeqNo = msg.Body[3:]
		msg.Body = ""
	}
	return msg, nil, false, true
}

// String renders the packet for log / panel display. Always
// non-empty.
func (p Packet) String() string {
	switch p.Type {
	case TypePosition:
		if p.Position == nil {
			break
		}
		return fmt.Sprintf("POSITION %.4f,%.4f %q",
			p.Position.Latitude, p.Position.Longitude, p.Position.Comment)
	case TypeMessage:
		if p.Message == nil {
			break
		}
		if p.Message.Ack {
			return fmt.Sprintf("ACK to %s seq=%s", p.Message.Addressee, p.Message.SeqNo)
		}
		if p.Message.Rej {
			return fmt.Sprintf("REJ to %s seq=%s", p.Message.Addressee, p.Message.SeqNo)
		}
		return fmt.Sprintf("MSG to %s: %q", p.Message.Addressee, p.Message.Body)
	case TypeStatus:
		if p.Status == nil {
			break
		}
		return fmt.Sprintf("STATUS %q", p.Status.Text)
	case TypeBulletin:
		if p.Bulletin == nil {
			break
		}
		return fmt.Sprintf("BULLETIN %s: %q", p.Bulletin.ID, p.Bulletin.Body)
	case TypeMicE:
		return "MIC-E (compressed; decoder pending)"
	case TypeWeather:
		return "WEATHER " + p.Raw
	case TypeTelemetry:
		return "TELEMETRY " + p.Raw
	case TypeObject:
		return "OBJECT " + p.Raw
	}
	return "UNKNOWN " + p.Raw
}
