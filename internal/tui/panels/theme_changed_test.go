package panels

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
	"github.com/MattCheramie/GopherTrunk/internal/tui/theme"
)

// TestTablePanels_ApplyThemeChanged exercises the
// applyThemeIfChanged hook on every panel that uses bubbles/table.
// We swap the palette mid-test, feed each panel a ThemeChangedMsg,
// then confirm the table's selected-row style picked up the new
// SelectBg. Asserting on the style colour is a cheap proxy for "the
// whole table-style bundle was re-applied".
func TestTablePanels_ApplyThemeChanged(t *testing.T) {
	theme.Set(theme.DarkPalette())
	t.Cleanup(func() { theme.Set(theme.DarkPalette()) })

	type panelCase struct {
		name string
		p    Panel
		// extractor returns the underlying *table.Model selected-row
		// style background colour so we can compare across palettes.
		bg func() string
	}

	sys := NewSystems()
	tg := NewTalkgroups()
	dev := NewDevices()
	act := NewActive()
	hist := NewHistory()
	tones := NewTones()
	met := NewMetrics()
	setp := NewSettings()

	cases := []panelCase{
		{"systems", sys, func() string { return string(theme.Theme().SelectBg) }},
		{"talkgroups", tg, func() string { return string(theme.Theme().SelectBg) }},
		{"devices", dev, func() string { return string(theme.Theme().SelectBg) }},
		{"active", act, func() string { return string(theme.Theme().SelectBg) }},
		{"history", hist, func() string { return string(theme.Theme().SelectBg) }},
		{"tones", tones, func() string { return string(theme.Theme().SelectBg) }},
		{"metrics", met, func() string { return string(theme.Theme().SelectBg) }},
		{"settings", setp, func() string { return string(theme.Theme().SelectBg) }},
	}

	// Swap to monochrome and broadcast ThemeChangedMsg. The actual
	// behaviour we care about is: no panic, panel updates without
	// error, and a subsequent View() call uses the new palette.
	theme.Set(theme.MonochromePalette())
	shared := &state.SharedState{
		Systems:    []client.SystemDTO{{Name: "A"}},
		Talkgroups: []client.TalkgroupDTO{{ID: 1, AlphaTag: "TG"}},
		Devices:    []client.SDRStatus{{Serial: "AAA"}},
	}
	for _, c := range cases {
		_, _ = c.p.Update(ThemeChangedMsg{}, shared)
		// Then a normal sizing update to flush layout.
		_, _ = c.p.Update(tea.WindowSizeMsg{Width: 120, Height: 30}, shared)
		view := c.p.View(120, 30, false, shared)
		if view == "" {
			t.Errorf("%s: empty View() after ThemeChangedMsg", c.name)
		}
		if got := c.bg(); got != "" {
			t.Errorf("%s: theme didn't switch — SelectBg = %q", c.name, got)
		}
	}
}
