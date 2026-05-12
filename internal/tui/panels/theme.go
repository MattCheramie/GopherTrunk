package panels

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/MattCheramie/GopherTrunk/internal/tui/theme"
)

// Package-level inline-text styles consumed by every panel. Held as
// var rather than func so call sites stay terse (`dashHeader.Render`
// instead of `dashHeader().Render`). RefreshTheme rebuilds them after
// a runtime palette swap (--no-color, future light-mode toggle, etc.)
// so the root model can flip the whole panel package's colours in
// one call.
var (
	dashHeader lipgloss.Style
	dashOK     lipgloss.Style
	dashErr    lipgloss.Style
	dashDim    lipgloss.Style
	dashAlert  lipgloss.Style
	dashWarn   lipgloss.Style
	dashAccent lipgloss.Style
)

func init() { RefreshTheme() }

// RefreshTheme rebuilds the package-level styles from the active
// theme.Theme(). Safe to call any time; mutates module-level vars so
// don't call concurrently with a render.
func RefreshTheme() {
	p := theme.Theme()
	dashHeader = p.Title()
	dashOK = p.OK()
	dashErr = p.Error()
	dashDim = p.Dim()
	dashAlert = p.Alert()
	dashWarn = p.Warn()
	dashAccent = p.Accented()
}
