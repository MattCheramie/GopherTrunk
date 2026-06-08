package configtui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/configbuilder"
	"github.com/MattCheramie/GopherTrunk/internal/radioreference"
)

func newTestModel() Model {
	m := New([]string{"/tmp"}, radioreference.Auth{}, nil, "")
	m.width, m.height = 120, 40
	return m
}

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func send(m Model, keys ...string) Model {
	for _, k := range keys {
		tm, _ := m.Update(key(k))
		m = tm.(Model)
	}
	return m
}

// gotoSection moves the section cursor to the section with the given key and
// focuses the form.
func gotoSection(m Model, sectionKey string) Model {
	// reset to top
	for m.sectionIdx > 0 {
		m = send(m, "up")
	}
	for _, want := range sectionKeysOrder() {
		if want == sectionKey {
			break
		}
		m = send(m, "down")
	}
	return send(m, "enter") // focus form
}

func sectionKeysOrder() []string {
	var ks []string
	for _, s := range configbuilder.Sections() {
		ks = append(ks, s.Key)
	}
	return ks
}

func TestModelEditScalar(t *testing.T) {
	m := newTestModel()
	// Logging section: edit Level via select-cycle (space on the row).
	m = gotoSection(m, "log")
	// First row is Level (a select). Activate cycles it off the default.
	before := m.cfg.Log.Level
	m = send(m, "enter") // cycle select once
	if m.cfg.Log.Level == before {
		t.Fatalf("expected Log.Level to change from %q", before)
	}
	if !m.dirty {
		t.Fatalf("expected dirty after edit")
	}
}

func TestModelListAddAndValidate(t *testing.T) {
	m := newTestModel()
	m = gotoSection(m, "trunking")
	// Trunking form: Systems (list) is the first row.
	m = send(m, "enter") // drill into Systems list (empty)
	m = send(m, "a")     // add a system
	if len(m.cfg.Trunking.Systems) != 1 {
		t.Fatalf("expected 1 system, got %d", len(m.cfg.Trunking.Systems))
	}
	// A nameless system is invalid → trunking section has an error.
	if len(m.validation["trunking"]) == 0 {
		t.Fatalf("expected a trunking validation error for the nameless system")
	}
	// Drill into the system and confirm reflection produced rows.
	m = send(m, "enter")
	rows := structRows("SystemConfig", m.cur())
	if len(rows) == 0 {
		t.Fatalf("expected SystemConfig rows")
	}
}

func TestModelQuitGuard(t *testing.T) {
	m := newTestModel()
	m = gotoSection(m, "log")
	m = send(m, "enter") // make a change (dirty)
	// q while in the form zone shouldn't quit; go back to sections then q.
	m = send(m, "esc") // back to sections
	tm, _ := m.Update(key("q"))
	m = tm.(Model)
	if m.modal == nil {
		t.Fatalf("expected an unsaved-changes confirm modal on quit")
	}
}
