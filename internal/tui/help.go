package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"

	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
	"github.com/MattCheramie/GopherTrunk/internal/tui/theme"
)

// renderHelpOverlay draws the full keybinding card the operator
// sees on `?`. Two columns: "Global" from the root model's
// globalKeys, "Panel" from the active panel's Keys(). Bottom strip
// repeats the universal escape hatch ("? to close").
func (m *Model) renderHelpOverlay(width, height int, behind string) string {
	p := theme.Theme()
	header := p.Title().Render("Keyboard reference")

	globalCol := renderHelpColumn("Global", m.keys.helpEntries())
	panelKeys := m.panels[m.active].Keys()
	panelCol := renderHelpColumn(m.panels[m.active].Title(), keyEntriesFromBindings(panelKeys))

	cols := lipgloss.JoinHorizontal(lipgloss.Top, globalCol, "    ", panelCol)
	footer := p.Hint().Render("? to close")

	box := p.ModalBox("info").Render(header + "\n\n" + cols + "\n\n" + footer)
	return lipgloss.Place(width, height,
		lipgloss.Center, lipgloss.Center,
		box,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(p.BgAlt),
	)
}

type helpEntry struct{ key, desc string }

func renderHelpColumn(title string, entries []helpEntry) string {
	p := theme.Theme()
	if len(entries) == 0 {
		return p.Header().Render(title) + "\n" + p.Hint().Render("  (no keys)")
	}
	maxK := 0
	for _, e := range entries {
		if l := len([]rune(e.key)); l > maxK {
			maxK = l
		}
	}
	var b strings.Builder
	b.WriteString(p.Header().Render(title) + "\n")
	for _, e := range entries {
		k := e.key + strings.Repeat(" ", maxK-len([]rune(e.key)))
		b.WriteString("  " + p.Accented().Render(k) + "  " + e.desc + "\n")
	}
	return b.String()
}

func keyEntriesFromBindings(bs []key.Binding) []helpEntry {
	out := make([]helpEntry, 0, len(bs))
	for _, b := range bs {
		h := b.Help()
		out = append(out, helpEntry{key: h.Key, desc: h.Desc})
	}
	return out
}

// helpEntries lists every global binding the operator can fire from
// the root model. Order matches the natural keyboard scan: nav first,
// then jumps, then commands.
func (k globalKeys) helpEntries() []helpEntry {
	return []helpEntry{
		{"tab", "next panel"},
		{"shift+tab", "prev panel"},
		{"1-9, 0", "jump to panel by index"},
		{"ctrl+p", "command palette"},
		{"?", "toggle help"},
		{"r", "refresh"},
		{"q / ctrl+c", "quit"},
	}
}

// activeHelp is true when the help overlay is currently up. Used by
// activeModal() so the help overlay takes input precedence.
func (m *Model) helpVisible() bool { return m.help.ShowAll }

// helpKindMatches catches whether a key event is what activates the
// help overlay. Exposed so the centralized Update routing can flip
// state.
func (m *Model) helpKindMatches(_ state.PanelKind) bool { return false }
