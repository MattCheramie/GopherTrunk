package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/panels"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

// tabRect is the bounding box of one tab in the rendered tab strip,
// cached every renderTabs() call so hitTestTab can map a mouse click
// back to a state.PanelKind without re-measuring the strip.
type tabRect struct {
	kind         state.PanelKind
	xStart, xEnd int
}

// bodyTopY records the row at which the active panel's frame begins —
// always immediately under the tab strip. Cached at render time so
// body clicks can be translated into panel-local coordinates without
// re-measuring.
//
// The tab strip is currently a single line in every rendered layout
// (renderTabs returns one row regardless of wide vs. compact mode),
// so this defaults to 1. The field is kept on the Model so future
// multi-line chrome (e.g. a session banner) can update it from one
// place.
//
// We don't measure lipgloss.Height(tabs) here because that would
// re-invoke renderTabs from a non-View path and force a duplicate
// render. Storing the height during View() is simpler.

// handleMouseMsg routes a tea.MouseMsg to the right tabbable
// surface. Left-clicks on the tab strip switch panels; left-clicks
// and wheel events in the body are delegated to the active panel
// when it implements panels.MouseAware. Returns true when the event
// was consumed.
func (m *Model) handleMouseMsg(msg tea.MouseMsg) (tea.Cmd, bool) {
	// Tab-strip clicks are left-press only — wheel ticks over the
	// strip still belong to the active panel's body.
	if msg.Y == 0 && msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
		for _, r := range m.tabRects {
			if msg.X >= r.xStart && msg.X < r.xEnd {
				m.active = r.kind
				return nil, true
			}
		}
		return nil, false
	}
	// Drop button releases and motion events — they're noisy and no
	// panel cares about them today.
	if msg.Action == tea.MouseActionRelease || msg.Action == tea.MouseActionMotion {
		return nil, false
	}
	if int(m.active) < 0 || int(m.active) >= len(m.panels) {
		return nil, false
	}
	aware, ok := m.panels[m.active].(panels.MouseAware)
	if !ok {
		return nil, false
	}
	// Local Y is global Y minus the tab strip height (one row). For
	// wheel events that arrive over the tab strip we still want the
	// body to receive them, hence localY may be negative — panels
	// must guard against it via tableRowFromLocalY.
	localY := msg.Y - 1
	cmd := aware.HandleMouse(msg, localY)
	return cmd, true
}
