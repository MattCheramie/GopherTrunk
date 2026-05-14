package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestTUIToggleScan asserts the `s` key on the Talkgroups tab flips
// the Scan flag on the cursored talkgroup. Drives the model directly
// without a real terminal program.
func TestTUIToggleScan(t *testing.T) {
	sys := sampleParsedSystem()
	m := newImportTUI([]parsedSystem{sys}, dummyWrite)

	// Enter system view, then switch to Talkgroups tab.
	m = step(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = step(m, tea.KeyMsg{Type: tea.KeyTab})

	// Toggle Scan twice; first flip false, second flip back true.
	if !m.systems[0].Talkgroups[0].Scan {
		t.Fatalf("initial Scan should be true")
	}
	m = step(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if m.systems[0].Talkgroups[0].Scan {
		t.Errorf("after first 's': Scan should be false")
	}
	m = step(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if !m.systems[0].Talkgroups[0].Scan {
		t.Errorf("after second 's': Scan should be true again")
	}
}

func TestTUITogglePriority(t *testing.T) {
	sys := sampleParsedSystem()
	m := newImportTUI([]parsedSystem{sys}, dummyWrite)
	m = step(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = step(m, tea.KeyMsg{Type: tea.KeyTab})

	m = step(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	if got := m.systems[0].Talkgroups[0].Priority; got != 5 {
		t.Errorf("Priority = %d, want 5", got)
	}
	m = step(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	if got := m.systems[0].Talkgroups[0].Priority; got != 0 {
		t.Errorf("Priority = %d, want 0 after '0' key", got)
	}
}

func TestTUIToggleLockout(t *testing.T) {
	sys := sampleParsedSystem()
	m := newImportTUI([]parsedSystem{sys}, dummyWrite)
	m = step(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = step(m, tea.KeyMsg{Type: tea.KeyTab})

	m = step(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	if !m.systems[0].Talkgroups[0].Lockout {
		t.Errorf("after 'L': Lockout should be true")
	}
}

func TestTUIToggleSiteInclude(t *testing.T) {
	sys := sampleParsedSystem()
	m := newImportTUI([]parsedSystem{sys}, dummyWrite)
	m = step(m, tea.KeyMsg{Type: tea.KeyEnter}) // enter system view (sites tab default)

	if !m.systems[0].Sites[0].Include {
		t.Fatalf("initial Include should be true")
	}
	m = step(m, tea.KeyMsg{Type: tea.KeySpace})
	if m.systems[0].Sites[0].Include {
		t.Errorf("after space: site Include should be false")
	}
}

func TestTUIWriteCallsWriteFn(t *testing.T) {
	called := false
	writeFn := func(_ []parsedSystem) (mergeResult, error) {
		called = true
		return mergeResult{}, nil
	}
	m := newImportTUI([]parsedSystem{sampleParsedSystem()}, writeFn)

	// 'w' triggers writeFn.
	mAny, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	m = mAny.(importTUIModel)
	if !called {
		t.Errorf("writeFn not invoked on 'w'")
	}
	if !m.wrote {
		t.Errorf("model.wrote should be true after successful write")
	}
}

func step(m importTUIModel, msg tea.Msg) importTUIModel {
	mAny, _ := m.Update(msg)
	return mAny.(importTUIModel)
}

func dummyWrite(_ []parsedSystem) (mergeResult, error) {
	return mergeResult{}, nil
}
