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

// Revealer is an optional interface for panels that can pre-position
// their internal cursor on a specific row when the operator jumps in
// from the command palette. key is panel-defined: SystemsPanel takes
// a system name, TalkgroupsPanel a decimal ID, DevicesPanel a serial,
// ScannerPanel "sys:<name>" or "conv:<idx>".
type Revealer interface {
	Reveal(key string)
}

// MouseAware is an optional interface for panels that want to react to
// mouse events inside their body. localY is the row offset relative
// to the panel's top-left (0 = panel border row); msg carries the
// full bubbletea mouse payload so the panel can distinguish a
// left-click on a data row (move cursor) from a scroll-wheel tick
// (advance one row) from a button release (no-op). Returning a
// tea.Cmd is permitted but most implementations will return nil.
type MouseAware interface {
	HandleMouse(msg tea.MouseMsg, localY int) tea.Cmd
}

// ThemeChangedMsg is broadcast by the root model after a palette
// swap. Panels that cache lipgloss styles (chiefly the
// bubbles/table-backed ones, which call SetStyles in their
// constructor) handle it by re-applying tableStyles() so the new
// palette takes effect on the next render. Read-only panels that
// fetch theme.Theme() on every View() can ignore it.
type ThemeChangedMsg struct{}
