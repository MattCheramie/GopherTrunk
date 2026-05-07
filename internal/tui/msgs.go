package tui

import (
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
)

// tea.Msg types used by the root model. Panels also receive these
// messages and may consume them.

type tickMsg struct{ at time.Time }

type pollHealthMsg struct {
	h   client.Health
	err error
}
type pollVersionMsg struct {
	v   string
	err error
}
type pollSystemsMsg struct {
	s   []client.SystemDTO
	err error
}
type pollTalkgroupsMsg struct {
	tg  []client.TalkgroupDTO
	err error
}
type pollActiveMsg struct {
	calls []client.ActiveCallDTO
	err   error
}
type pollHistoryMsg struct {
	rows []client.CallRow
	err  error
}
type pollMetricsMsg struct {
	m   map[string]float64
	err error
}

// eventMsg carries one decoded SSE event. The root model fans this
// out into its event log + tone alert ring buffer, then forwards
// to whichever panel displays the event live.
type eventMsg struct{ ev client.Event }

// sseDownMsg signals the SSE pump terminated. The root model will
// schedule a reconnect Cmd with backoff.
type sseDownMsg struct{ err error }

// sseUpMsg signals a fresh SSE channel. The root model swaps it in
// and re-arms the listenSSE Cmd.
type sseUpMsg struct {
	ch     <-chan client.Event
	cancel func()
}

// errToastMsg displays a transient error in the status bar.
type errToastMsg struct{ msg string }

// writeResultMsg carries the outcome of a write Cmd.
type writeResultMsg struct {
	Label string
	Err   error
}

// pollMutationStatusMsg fetches /api/v1/mutations once at startup
// to set shared.Mutations + WriteEnabled.
type pollMutationStatusMsg struct {
	s   client.MutationStatus
	err error
}
