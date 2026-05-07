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
