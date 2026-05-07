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

	// Write capability — populated at startup from the daemon's
	// /api/v1/mutations endpoint AND-ed with the TUI's --write
	// flag. Panels read this to decide whether write keybindings
	// should fire actions or show a "mutations disabled" toast.
	WriteEnabled bool
	// Mutations exposes per-subsystem capability so panels can
	// show finer-grained tooltips (e.g. "tone-out detector not
	// wired" vs "mutations disabled at daemon").
	Mutations client.MutationStatus
}

// WriteRequest is a plain-value command emitted by panels when the
// operator presses a mutation keybinding. The root model unwraps
// the embedded request type, optionally pops a confirmation modal,
// and dispatches the matching client method.
//
// Panels can't build the write Cmds themselves (they don't hold a
// client.Client), so the request shape is a typed value the root
// resolves. Each request kind has its own struct so the root's
// type switch is exhaustive.
type WriteRequest struct {
	Confirm string
	Label   string
	Kind    WriteKind

	EndCall         *EndCallReq
	UpdateTalkgroup *UpdateTalkgroupReq
	SweepRetention  *SweepRetentionReq
	ResetTone       *ResetToneReq
}

// WriteKind discriminates a WriteRequest's payload.
type WriteKind int

const (
	WriteKindUnknown WriteKind = iota
	WriteKindEndCall
	WriteKindUpdateTalkgroup
	WriteKindSweepRetention
	WriteKindResetTone
)

// EndCallReq forces the engine to release the active call held on
// the given device.
type EndCallReq struct {
	DeviceSerial string
	Reason       string // optional; "" means "manual"
}

// UpdateTalkgroupReq mutates priority and/or lockout on a talkgroup.
// Pointer fields allow "leave unchanged" semantics.
type UpdateTalkgroupReq struct {
	ID       uint32
	Priority *int
	Lockout  *bool
}

// SweepRetentionReq runs one immediate retention sweep.
type SweepRetentionReq struct{}

// ResetToneReq clears tone-out match progress on the given device.
type ResetToneReq struct {
	DeviceSerial string
}
