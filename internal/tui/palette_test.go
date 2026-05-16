package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

func TestFuzzyScore_ExactPrefixWins(t *testing.T) {
	if fuzzyScore("settings", "set") <= fuzzyScore("scanner settings", "set") {
		t.Fatalf("prefix match did not outrank middle-of-string match")
	}
}

func TestFuzzyScore_NoMatch(t *testing.T) {
	if fuzzyScore("scanner", "xyz") != 0 {
		t.Fatalf("expected 0 on miss, got >0")
	}
}

func TestPalette_OpensAndDiscoversPanelJumps(t *testing.T) {
	m := newTestModel(t)
	m.openPalette()
	if !m.palette.open {
		t.Fatal("palette did not open")
	}
	// Every panel title must show up as a jump action.
	for i := state.PanelKind(0); i < state.PanelCount; i++ {
		want := "Jump to " + m.panels[i].Title()
		var found bool
		for _, a := range m.palette.all {
			if a.Label == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing palette action %q", want)
		}
	}
}

func TestPalette_FilterMatchesSubstring(t *testing.T) {
	m := newTestModel(t)
	m.openPalette()
	// Type "scan" — should match Scanner-related entries.
	m.palette.input.SetValue("scan")
	m.palette.refilter()
	if len(m.palette.filtered) == 0 {
		t.Fatalf("scan filter yielded no matches")
	}
	if !strings.Contains(strings.ToLower(m.palette.filtered[0].Label), "scan") {
		t.Errorf("top match doesn't contain 'scan': %q", m.palette.filtered[0].Label)
	}
}

func TestPalette_EnterRunsAction(t *testing.T) {
	m := newTestModel(t)
	m.openPalette()
	// Move to first action and Enter — should close the palette.
	cmd := m.handlePaletteKey(tea.KeyMsg{Type: tea.KeyEnter})
	if m.palette.open {
		t.Fatal("Enter did not close the palette")
	}
	// Panel jumps return nil cmd; that's fine. Just assert no panic.
	_ = cmd
}

func TestPalette_EscClosesWithoutAction(t *testing.T) {
	m := newTestModel(t)
	m.openPalette()
	originalActive := m.active
	m.handlePaletteKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.palette.open {
		t.Fatal("Esc did not close the palette")
	}
	if m.active != originalActive {
		t.Errorf("Esc fired an action — active changed from %v to %v", originalActive, m.active)
	}
}
