// Package panels contains the eight read-only panels rendered by
// the TUI. Each panel is a self-contained bubbletea sub-model that
// renders against the shared state owned by the root model.
package panels

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

// Panel is the contract every visible panel implements. Update may
// return a replacement Panel — bubbletea's Elm-style "return next
// state" pattern. Panels never call the network; they read from
// shared state populated by the root model's polling Cmds.
type Panel interface {
	Title() string
	Keys() []key.Binding
	Update(msg tea.Msg, shared *state.SharedState) (Panel, tea.Cmd)
	View(width, height int, focused bool, shared *state.SharedState) string
}
