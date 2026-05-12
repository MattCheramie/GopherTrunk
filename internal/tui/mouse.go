package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

// tabRect is the bounding box of one tab in the rendered tab strip,
// cached every renderTabs() call so hitTestTab can map a mouse click
// back to a state.PanelKind without re-measuring the strip.
type tabRect struct {
	kind        state.PanelKind
	xStart, xEnd int
}

// handleMouseMsg routes a tea.MouseMsg to the right tabbable
// surface. Currently only the tab strip (top row) and the "more"
// pill (also top row) are wired. Returns true when the click was
// consumed.
func (m *Model) handleMouseMsg(msg tea.MouseMsg) (tea.Cmd, bool) {
	if msg.Button != tea.MouseButtonLeft || msg.Action != tea.MouseActionPress {
		return nil, false
	}
	// Tab strip is always row 0. If the click is below the strip
	// pass it through to the active panel (delegated below in
	// Update once panel-side mouse hooks land).
	if msg.Y != 0 {
		return nil, false
	}
	for _, r := range m.tabRects {
		if msg.X >= r.xStart && msg.X < r.xEnd {
			m.active = r.kind
			return nil, true
		}
	}
	return nil, false
}
