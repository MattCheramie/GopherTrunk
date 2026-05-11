package panels

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

// WriteActionMsg is what panels emit to ask the root model to run
// a mutation. The root catches this exact type, so panels stay
// decoupled from the root's modal mechanics and HTTP client.
//
// Exported so tests in other packages can assert on it; intended
// to be opaque to consumers.
type WriteActionMsg struct{ Request state.WriteRequest }

// Emit returns a tea.Cmd that delivers the WriteActionMsg to the
// root model.
func Emit(r state.WriteRequest) tea.Cmd {
	return func() tea.Msg { return WriteActionMsg{Request: r} }
}

// SystemDetailMsg is emitted by the Systems panel when the operator
// presses Enter on a row. The root model fetches the detail record
// via GET /api/v1/systems/{name} and renders a read-only modal.
type SystemDetailMsg struct{ Name string }

// TalkgroupDetailMsg is emitted by the Talkgroups panel when the
// operator presses Enter on a row.
type TalkgroupDetailMsg struct{ ID uint32 }

// EmitSystemDetail returns a Cmd delivering a SystemDetailMsg.
func EmitSystemDetail(name string) tea.Cmd {
	return func() tea.Msg { return SystemDetailMsg{Name: name} }
}

// EmitTalkgroupDetail returns a Cmd delivering a TalkgroupDetailMsg.
func EmitTalkgroupDetail(id uint32) tea.Cmd {
	return func() tea.Msg { return TalkgroupDetailMsg{ID: id} }
}
