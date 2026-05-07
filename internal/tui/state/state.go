// Package state holds the SharedState struct and PanelKind enum so
// the root tui package and panels sub-package can both import it
// without an import cycle.
package state

import (
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
)

// PanelKind enumerates the visible panels. The root model owns the
// active selection; panels themselves don't know their index.
type PanelKind int

const (
	PanelDashboard PanelKind = iota
	PanelSystems
	PanelTalkgroups
	PanelActive
	PanelHistory
	PanelEvents
	PanelTones
	PanelMetrics
	PanelCount
)

func (p PanelKind) String() string {
	switch p {
	case PanelDashboard:
		return "Dashboard"
	case PanelSystems:
		return "Systems"
	case PanelTalkgroups:
		return "Talkgroups"
	case PanelActive:
		return "Active"
	case PanelHistory:
		return "History"
	case PanelEvents:
		return "Events"
	case PanelTones:
		return "Tones"
	case PanelMetrics:
		return "Metrics"
	}
	return "?"
}

// RingReader is the read-side interface RingBuf satisfies. Panels
// only need read access to the event/tone buffers.
type RingReader[T any] interface {
	Len() int
	Snapshot() []T
	Latest(n int) []T
}

// SharedState is the snapshot of daemon-derived data that all panels
// read from. The root model owns it and passes a pointer into each
// panel's Update.
type SharedState struct {
	Health      client.Health
	HealthErr   error
	Version     string
	Systems     []client.SystemDTO
	Talkgroups  []client.TalkgroupDTO
	ActiveCalls []client.ActiveCallDTO
	History     []client.CallRow
	HistoryErr  error
	Metrics     map[string]float64

	EventLog   RingReader[client.Event]
	ToneAlerts RingReader[client.Event]

	LastPoll time.Time
	Toast    string
	Server   string
}
