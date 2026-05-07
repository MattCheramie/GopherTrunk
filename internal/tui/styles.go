package tui

import "github.com/charmbracelet/lipgloss"

// styles bundles the lipgloss styles the panels share. Holding them
// in one place keeps colour decisions in lockstep across panels.
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

// newStyles builds a style set. If noColor is true colours are
// stripped — useful when writing to a non-TTY or when --no-color
// is passed.
func newStyles(noColor bool) styles {
	if noColor {
		lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(nil))
	}
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)
	focus := border.Copy().BorderForeground(lipgloss.Color("39"))
	return styles{
		border:      border,
		focusBorder: focus,
		header:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")),
		tab:         lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("245")),
		activeTab:   lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("231")).Background(lipgloss.Color("39")).Bold(true),
		statusBar:   lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Background(lipgloss.Color("236")).Padding(0, 1),
		dim:         lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		accent:      lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true),
		alert:       lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		ok:          lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
		error:       lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		help:        lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		toast:       lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("88")).Padding(0, 1),
	}
}
