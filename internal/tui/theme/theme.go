// Package theme owns the TUI's semantic colour palette and the
// derived high-level lipgloss styles. Panels read from theme.Theme()
// instead of hard-coding lipgloss.Color("…") literals so colour
// decisions stay in one place and a future light-mode palette is a
// one-line swap.
//
// The package lives outside internal/tui so internal/tui/panels can
// import it without an import cycle.
package theme

import (
	"sync/atomic"

	"github.com/charmbracelet/lipgloss"
)

// Palette is the semantic colour set. Field names describe *role*,
// not hue — Accent is the canonical highlight colour, Danger is
// reserved for destructive/error surfaces, etc.
type Palette struct {
	Fg          lipgloss.Color
	FgMuted     lipgloss.Color
	BgAlt       lipgloss.Color
	Accent      lipgloss.Color
	Success     lipgloss.Color
	Warning     lipgloss.Color
	Danger      lipgloss.Color
	Info        lipgloss.Color
	Border      lipgloss.Color
	BorderFocus lipgloss.Color
	SelectBg    lipgloss.Color
	ToastBg     lipgloss.Color
}

// DarkPalette is the default 256-colour palette tuned for dark
// terminals. Numbers map to xterm-256: 39 (dodger blue), 42 (green),
// 196 (red), 220 (yellow), 245 (light grey), 240 (mid grey), 236
// (charcoal), 231 (white), 88 (dark red — for toast bg), 57 (purple
// — for selected row bg).
func DarkPalette() Palette {
	return Palette{
		Fg:          "231",
		FgMuted:     "245",
		BgAlt:       "236",
		Accent:      "39",
		Success:     "42",
		Warning:     "220",
		Danger:      "196",
		Info:        "39",
		Border:      "240",
		BorderFocus: "39",
		SelectBg:    "57",
		ToastBg:     "88",
	}
}

// MonochromePalette is used when --no-color is passed or stdout
// isn't a TTY. Every role collapses to default fg/bg so lipgloss
// emits no ANSI.
func MonochromePalette() Palette {
	return Palette{
		Fg:          "",
		FgMuted:     "",
		BgAlt:       "",
		Accent:      "",
		Success:     "",
		Warning:     "",
		Danger:      "",
		Info:        "",
		Border:      "",
		BorderFocus: "",
		SelectBg:    "",
		ToastBg:     "",
	}
}

// Title returns the bold accent style used for panel titles and
// section headers.
func (p Palette) Title() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(p.Accent)
}

// Header is the table-header style — bold accent with a bottom
// border in the muted colour.
func (p Palette) Header() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(p.Accent)
}

// Hint is the dim helper-text style used in footers and inline
// usage hints.
func (p Palette) Hint() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(p.FgMuted)
}

// Selected is the highlight applied to the active row inside a
// table-style panel.
func (p Palette) Selected() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(p.Fg).Background(p.SelectBg)
}

// Frame returns the rounded border style, accenting the border
// colour when focused. Named Frame rather than Border so it doesn't
// collide with the Border colour field.
func (p Palette) Frame(focused bool) lipgloss.Style {
	s := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(p.Border).
		Padding(0, 1)
	if focused {
		s = s.BorderForeground(p.BorderFocus)
	}
	return s
}

// OK returns the success-coloured inline style for "● ok" indicators.
func (p Palette) OK() lipgloss.Style { return lipgloss.NewStyle().Foreground(p.Success) }

// Error is the danger-coloured inline style for error indicators.
func (p Palette) Error() lipgloss.Style { return lipgloss.NewStyle().Foreground(p.Danger) }

// Alert is the bold danger style used for tone-alert-style attention
// callouts. Same hue as Error but with weight to differentiate from
// a passive error indicator.
func (p Palette) Alert() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(p.Danger)
}

// Warn is the warning-coloured inline style — used for in-progress /
// transient states like "hunting".
func (p Palette) Warn() lipgloss.Style { return lipgloss.NewStyle().Foreground(p.Warning) }

// Dim is the muted-foreground style for de-emphasised text.
func (p Palette) Dim() lipgloss.Style { return lipgloss.NewStyle().Foreground(p.FgMuted) }

// Accented returns a bold-accent inline style.
func (p Palette) Accented() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(p.Accent)
}

// StatusBar returns the status-bar background style.
func (p Palette) StatusBar() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(p.FgMuted).Background(p.BgAlt).Padding(0, 1)
}

// Tab returns an unselected-tab style.
func (p Palette) Tab() lipgloss.Style {
	return lipgloss.NewStyle().Padding(0, 1).Foreground(p.FgMuted)
}

// ActiveTab returns the selected-tab style — inverse fg/bg in the
// accent colour.
func (p Palette) ActiveTab() lipgloss.Style {
	return lipgloss.NewStyle().Padding(0, 1).Foreground(p.Fg).Background(p.Accent).Bold(true)
}

// Toast returns the inline notification style — white on dark-red.
func (p Palette) Toast() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(p.Fg).Background(p.ToastBg).Padding(0, 1)
}

// ModalBox returns the modal-window styling shared by the confirm and
// detail modals. kind is one of "danger" / "info"; kind == "danger"
// uses the Danger border colour (red), anything else uses Accent.
func (p Palette) ModalBox(kind string) lipgloss.Style {
	border := p.Accent
	if kind == "danger" {
		border = p.Danger
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(1, 2).
		Background(p.BgAlt).
		Foreground(p.Fg)
}

// currentTheme is the process-wide singleton. Read with Theme(),
// swapped with Set(). atomic.Pointer keeps the swap goroutine-safe
// without taking a lock on every read.
var currentTheme atomic.Pointer[Palette]

func init() {
	p := DarkPalette()
	currentTheme.Store(&p)
}

// Theme returns the active palette. Always non-nil.
func Theme() Palette {
	return *currentTheme.Load()
}

// Set swaps the active palette. Pass MonochromePalette() to strip
// colour, DarkPalette() to restore.
func Set(p Palette) {
	currentTheme.Store(&p)
}
