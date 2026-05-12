package tui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/MattCheramie/GopherTrunk/internal/tui/theme"
)

// styles bundles the lipgloss styles the root model and modals
// share. Every style is derived from theme.Theme() so a future
// light-mode palette swap is a one-line change here.
type styles struct {
	border      lipgloss.Style
	focusBorder lipgloss.Style
	header      lipgloss.Style
	tab         lipgloss.Style
	activeTab   lipgloss.Style
	statusBar   lipgloss.Style
	dim         lipgloss.Style
	accent      lipgloss.Style
	alert       lipgloss.Style
	ok          lipgloss.Style
	error       lipgloss.Style
	help        lipgloss.Style
	toast       lipgloss.Style
}

// newStyles builds a style set. If noColor is true the active theme
// is swapped to the monochrome palette so every accessor across the
// process collapses to default fg/bg simultaneously.
func newStyles(noColor bool) styles {
	if noColor {
		lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(nil))
		theme.Set(theme.MonochromePalette())
	}
	p := theme.Theme()
	return styles{
		border:      p.Frame(false),
		focusBorder: p.Frame(true),
		header:      p.Header(),
		tab:         p.Tab(),
		activeTab:   p.ActiveTab(),
		statusBar:   p.StatusBar(),
		dim:         p.Dim(),
		accent:      p.Accented(),
		alert:       p.Alert(),
		ok:          p.OK(),
		error:       p.Error(),
		help:        p.Hint(),
		toast:       p.Toast(),
	}
}
