package panels

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/MattCheramie/GopherTrunk/internal/tui/theme"
)

// panelFrame draws the canonical bordered card used by every table
// panel. focused flips the border colour to the accent so the active
// panel is unambiguous regardless of where the operator's eye lands.
func panelFrame(title string, w, h int, focused bool, body string) string {
	p := theme.Theme()
	border := p.Frame(focused)
	if w > 4 {
		border = border.Width(w - 2)
	}
	if h > 4 {
		border = border.Height(h - 2)
	}
	titleStyle := p.Title()
	if focused {
		// Invert the title bar on the focused panel so the eye lands
		// on it immediately even when many panels share a screen.
		titleStyle = titleStyle.Background(p.Accent).Foreground(p.Fg)
	}
	return border.Render(titleStyle.Render(" "+title+" ") + "\n" + body)
}

// tableStyles is the shared bubbles/table theme — bold accent
// header with a muted bottom border, accent-tinted selected row.
func tableStyles() table.Styles {
	p := theme.Theme()
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(p.Border).
		BorderBottom(true).
		Bold(true).
		Foreground(p.Accent)
	s.Selected = s.Selected.
		Foreground(p.Fg).
		Background(p.SelectBg)
	return s
}
