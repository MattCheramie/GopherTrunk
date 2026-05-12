package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/panels"
	"github.com/MattCheramie/GopherTrunk/internal/tui/theme"
)

func TestToggleTheme_CyclesDarkMonoDark(t *testing.T) {
	// newTestModel constructs with NoColor: true which forces the
	// monochrome palette. Reset to dark *after* constructing so we
	// start the toggle from a known-coloured baseline. Reset on
	// cleanup so sibling tests aren't affected.
	m := newTestModel(t)
	theme.Set(theme.DarkPalette())
	t.Cleanup(func() { theme.Set(theme.DarkPalette()) })
	if got := theme.Theme().Accent; got == "" {
		t.Fatal("expected dark palette accent to be non-empty after fixture")
	}

	// First toggle: dark → mono. The Cmd carries a ThemeChangedMsg.
	cmd := m.toggleTheme()
	if cmd == nil {
		t.Fatal("toggleTheme returned nil cmd")
	}
	if got := theme.Theme().Accent; got != "" {
		t.Errorf("after toggle 1, accent = %q, want empty (monochrome)", got)
	}
	if _, ok := cmd().(panels.ThemeChangedMsg); !ok {
		t.Errorf("cmd did not produce ThemeChangedMsg")
	}

	// Second toggle: mono → dark again.
	_ = m.toggleTheme()
	if got := theme.Theme().Accent; got == "" {
		t.Errorf("after toggle 2, accent collapsed; want dark palette")
	}
}

func TestToggleTheme_BroadcastsThemeChangedMsg(t *testing.T) {
	m := newTestModel(t)
	theme.Set(theme.DarkPalette())
	t.Cleanup(func() { theme.Set(theme.DarkPalette()) })
	// Press Ctrl+T through the Update reducer — exercises the
	// key.Matches path, not the helper directly.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	m = updated.(*Model)
	if cmd == nil {
		t.Fatal("Ctrl+T produced no Cmd")
	}
	msg := cmd()
	if _, ok := msg.(panels.ThemeChangedMsg); !ok {
		t.Errorf("Ctrl+T Cmd produced %T, want ThemeChangedMsg", msg)
	}
	// Toggle should have flipped to monochrome.
	if theme.Theme().Accent != "" {
		t.Errorf("after Ctrl+T, accent = %q, want empty", theme.Theme().Accent)
	}
	// Toast surfaces the new theme so the operator gets feedback.
	if m.shared.Toast == "" {
		t.Errorf("toast not set after toggle")
	}
}
