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
	PanelDevices
	PanelScanner
	PanelHunt
	PanelSettings
	PanelImport
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
	case PanelDevices:
		return "Devices"
	case PanelScanner:
		return "Scanner"
	case PanelHunt:
		return "Hunt"
	case PanelSettings:
		return "Settings"
	case PanelImport:
		return "Import"
	}
	return "?"
}

// Key returns the stable config key for this panel — the same key the
// web SPA uses (route path minus its leading slash) so a single
// web.tabs entry hides the tab in both UIs. Keep in sync with
// config.KnownUITabs and web/src/App.tsx.
func (p PanelKind) Key() string {
	switch p {
	case PanelDashboard:
		return "dashboard"
	case PanelSystems:
		return "systems"
	case PanelTalkgroups:
		return "talkgroups"
	case PanelActive:
		return "active"
	case PanelHistory:
		return "history"
	case PanelEvents:
		return "events"
	case PanelTones:
		return "tones"
	case PanelMetrics:
		return "metrics"
	case PanelDevices:
		return "devices"
	case PanelScanner:
		return "scanner"
	case PanelHunt:
		return "hunt"
	case PanelSettings:
		return "settings"
	case PanelImport:
		return "import"
	}
	return ""
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
	Devices     []client.SDRStatus
	DevicesErr  error
	Scanner     client.ScannerStatusDTO
	ScannerErr  error
	Hunt        client.HuntStatusDTO
	HuntErr     error
	Audio       client.AudioStatusDTO
	AudioErr    error
	Runtime     client.RuntimeDTO
	RuntimeErr  error

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

	EndCall           *EndCallReq
	UpdateTalkgroup   *UpdateTalkgroupReq
	SweepRetention    *SweepRetentionReq
	ResetTone         *ResetToneReq
	ScannerMode       *ScannerModeReq
	ScannerHunt       *ScannerHuntReq
	ScannerConv       *ScannerConvReq
	ScannerManualTune *ScannerManualTuneReq
	Audio             *AudioReq
	Settings          *SettingsReq
	Hunt              *HuntStartReq
}

// HuntStartReq is the payload for WriteKindHuntStart — a live system-discovery
// run started from the TUI. Frequencies are in MHz (matching the operator's
// input); the client converts to the wire shape.
type HuntStartReq struct {
	Bands      []string  // "low:high" MHz
	Candidates []float64 // MHz
	Survey     bool      // classify+decode every carrier, not just trunking CCs
	Name       string
	Serial     string
	Protocol   string
}

// WriteKind discriminates a WriteRequest's payload.
type WriteKind int

const (
	WriteKindUnknown WriteKind = iota
	WriteKindEndCall
	WriteKindUpdateTalkgroup
	WriteKindSweepRetention
	WriteKindResetTone
	WriteKindScannerMode
	WriteKindScannerHuntHold
	WriteKindScannerHuntResume
	WriteKindScannerHuntRetune
	WriteKindScannerConvHold
	WriteKindScannerConvResume
	WriteKindScannerConvDwell
	WriteKindScannerConvLockout
	WriteKindScannerConvUnlockout
	WriteKindAudio
	WriteKindScannerManualTune
	WriteKindSettings
	WriteKindHuntStop
	WriteKindHuntStart
)

// ScannerManualTuneReq adds a temp VFO channel and forces dwell.
type ScannerManualTuneReq struct {
	FrequencyHz uint32
	Label       string
	Mode        string
}

// SettingsReq carries a single settings-panel edit. Field is the
// dotted YAML path (e.g. "audio.volume", "log.level") and Value is
// the operator's typed input; the dispatcher converts to the right
// SettingsPatch shape per field.
type SettingsReq struct {
	Field string
	Value string
}

// AudioReq sets one or more knobs on the audio cockpit. Nil fields
// are left unchanged.
type AudioReq struct {
	Volume    *float32
	Muted     *bool
	Recording *bool
}

// ScannerModeReq sets the engine's global scan_mode at runtime.
type ScannerModeReq struct{ Mode string }

// ScannerHuntReq targets a single trunked system for hold / resume /
// force-retune. The WriteKind discriminates the operation.
type ScannerHuntReq struct{ System string }

// ScannerConvReq is shared by hold / resume / dwell-on-index for the
// conventional scanner. Index is ignored for hold/resume.
type ScannerConvReq struct{ Index int }

// EndCallReq forces the engine to release the active call held on
// the given device.
type EndCallReq struct {
	DeviceSerial string
	Reason       string // optional; "" means "manual"
}

// UpdateTalkgroupReq mutates priority and/or lockout and/or scan on a
// talkgroup. Pointer fields allow "leave unchanged" semantics.
type UpdateTalkgroupReq struct {
	ID       uint32
	Priority *int
	Lockout  *bool
	Scan     *bool
}

// SweepRetentionReq runs one immediate retention sweep.
type SweepRetentionReq struct{}

// ResetToneReq clears tone-out match progress on the given device.
type ResetToneReq struct {
	DeviceSerial string
}
