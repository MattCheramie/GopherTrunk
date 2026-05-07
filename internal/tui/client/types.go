// Package client is the TUI's network layer. It wraps the daemon's
// HTTP REST + SSE surfaces in typed methods so the bubbletea model
// can stay focused on rendering.
//
// DTO shapes mirror internal/api/types.go and internal/storage's
// CallRow, copied by value rather than imported so that the TUI
// stays a pure client of the wire protocol — `internal/tui` doesn't
// pull in the server-side packages.
package client

import (
	"encoding/json"
	"time"
)

// Health is the response shape of GET /api/v1/health.
type Health struct {
	Status string    `json:"status"`
	Now    time.Time `json:"now"`
}

// Version is the response shape of GET /api/v1/version.
type Version struct {
	Version string `json:"version"`
}

// SystemDTO mirrors api.SystemDTO.
type SystemDTO struct {
	Name            string   `json:"name"`
	Protocol        string   `json:"protocol"`
	ControlChannels []uint32 `json:"control_channels"`
	WACN            uint32   `json:"wacn,omitempty"`
	SystemID        uint16   `json:"system_id,omitempty"`
	RFSS            uint8    `json:"rfss,omitempty"`
	Site            uint8    `json:"site,omitempty"`
}

// TalkgroupDTO mirrors api.TalkgroupDTO.
type TalkgroupDTO struct {
	ID          uint32 `json:"id"`
	AlphaTag    string `json:"alpha_tag"`
	Description string `json:"description,omitempty"`
	Tag         string `json:"tag,omitempty"`
	Group       string `json:"group,omitempty"`
	Mode        string `json:"mode,omitempty"`
	Priority    int    `json:"priority,omitempty"`
	Lockout     bool   `json:"lockout,omitempty"`
}

// GrantDTO mirrors api.GrantDTO.
type GrantDTO struct {
	System        string `json:"system"`
	Protocol      string `json:"protocol"`
	GroupID       uint32 `json:"group_id"`
	SourceID      uint32 `json:"source_id"`
	FrequencyHz   uint32 `json:"frequency_hz"`
	ChannelID     uint8  `json:"channel_id,omitempty"`
	ChannelNumber uint16 `json:"channel_number,omitempty"`
	Encrypted     bool   `json:"encrypted,omitempty"`
	Emergency     bool   `json:"emergency,omitempty"`
	DataCall      bool   `json:"data_call,omitempty"`
}

// ActiveCallDTO mirrors api.ActiveCallDTO.
type ActiveCallDTO struct {
	Grant        GrantDTO      `json:"grant"`
	Talkgroup    *TalkgroupDTO `json:"talkgroup,omitempty"`
	DeviceSerial string        `json:"device_serial"`
	StartedAt    time.Time     `json:"started_at"`
	LastHeardAt  time.Time     `json:"last_heard_at"`
}

// CallRow mirrors storage.CallRow — the shape returned by GET
// /api/v1/calls/history.
type CallRow struct {
	ID             int64     `json:"id"`
	System         string    `json:"system"`
	Protocol       string    `json:"protocol"`
	GroupID        uint32    `json:"group_id"`
	SourceID       uint32    `json:"source_id"`
	FrequencyHz    uint32    `json:"frequency_hz"`
	Encrypted      bool      `json:"encrypted"`
	Emergency      bool      `json:"emergency"`
	DataCall       bool      `json:"data_call"`
	DeviceSerial   string    `json:"device_serial"`
	StartedAt      time.Time `json:"started_at"`
	EndedAt        time.Time `json:"ended_at,omitempty"`
	DurationMs     int64     `json:"duration_ms,omitempty"`
	EndReason      string    `json:"end_reason,omitempty"`
	TalkgroupAlpha string    `json:"talkgroup_alpha,omitempty"`
}

// HistoryFilter is the query-parameter shape for GET
// /api/v1/calls/history. Zero-valued fields are omitted from the
// outgoing query string.
type HistoryFilter struct {
	System    string
	GroupID   uint32
	Since     time.Time
	Until     time.Time
	Limit     int  // 0 → server default (100)
	OnlyEnded bool
}

// Event is one decoded SSE event. Kind matches the events.Kind
// constants ("cc.locked", "grant", "call.start", …); Time is the
// envelope timestamp; Raw is the un-decoded JSON of the kind-
// specific payload, which the caller can decode into a typed
// payload as needed (Grant, Active, Tone, …).
type Event struct {
	Kind string
	Time time.Time
	Raw  json.RawMessage
}

// Tone is the payload shape of a `tone.alert` event.
type Tone struct {
	Profile      string    `json:"profile"`
	AlphaTag     string    `json:"alpha_tag,omitempty"`
	System       string    `json:"system,omitempty"`
	GroupID      uint32    `json:"group_id,omitempty"`
	DeviceSerial string    `json:"device_serial"`
	MatchedAt    time.Time `json:"matched_at"`
	FrequenciesHz []float64 `json:"frequencies_hz"`
}

// LockState is the payload of cc.locked / cc.lost events.
// FrequencyHz is always present; the other fields are protocol-
// specific.
type LockState struct {
	FrequencyHz uint32 `json:"FrequencyHz"`
	NAC         uint16 `json:"NAC,omitempty"`
	SystemID    uint32 `json:"SystemID,omitempty"`
	Repeater    string `json:"Repeater,omitempty"`
	MCC         uint16 `json:"MCC,omitempty"`
	MNC         uint16 `json:"MNC,omitempty"`
}

// HTTPError is returned by the REST methods on a non-2xx response.
// Status carries the HTTP code; Body carries up to a few hundred
// bytes of the response for the toast UI to show.
type HTTPError struct {
	Status int
	Method string
	URL    string
	Body   string
}

func (e *HTTPError) Error() string {
	return e.Method + " " + e.URL + ": " + httpCodeText(e.Status) + " — " + e.Body
}

func httpCodeText(code int) string {
	switch code {
	case 400:
		return "400 Bad Request"
	case 401:
		return "401 Unauthorized"
	case 404:
		return "404 Not Found"
	case 500:
		return "500 Internal Server Error"
	case 502:
		return "502 Bad Gateway"
	case 503:
		return "503 Service Unavailable"
	}
	return shortInt(code)
}

func shortInt(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [11]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
