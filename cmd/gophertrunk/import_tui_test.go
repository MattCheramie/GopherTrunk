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

func TestPageBounds(t *testing.T) {
	cases := []struct {
		name   string
		cursor int
		total  int
		page   int
		want   [2]int
	}{
		{"cursor at start", 0, 100, 10, [2]int{0, 10}},
		{"cursor mid", 50, 100, 10, [2]int{45, 55}},
		{"cursor near end clamps to fit", 98, 100, 10, [2]int{90, 100}},
		{"cursor at end clamps to fit", 99, 100, 10, [2]int{90, 100}},
		{"total smaller than page", 2, 5, 10, [2]int{0, 5}},
		{"empty list", 0, 0, 10, [2]int{0, 0}},
		{"page=0 falls back to 1", 0, 10, 0, [2]int{0, 1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gs, ge := pageBounds(tc.cursor, tc.total, tc.page)
			if gs != tc.want[0] || ge != tc.want[1] {
				t.Errorf("pageBounds(%d,%d,%d) = (%d,%d), want (%d,%d)",
					tc.cursor, tc.total, tc.page, gs, ge, tc.want[0], tc.want[1])
			}
		})
	}
}

func TestVisibleRows(t *testing.T) {
	cases := []struct {
		name   string
		height int
		want   int
	}{
		{"no WindowSizeMsg yet", 0, 20},
		{"tiny terminal floors at 5", 8, 5},
		{"normal 30-line terminal", 30, 24},
		{"large 60-line terminal", 60, 54},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := importTUIModel{height: tc.height}
			if got := m.visibleRows(); got != tc.want {
				t.Errorf("visibleRows() with height=%d = %d, want %d", tc.height, got, tc.want)
			}
		})
	}
}

func TestSitesTabNavigation(t *testing.T) {
	sys := manySitesParsedSystem(100)
	m := newImportTUI([]parsedSystem{sys}, dummyWrite)
	m.height = 30 // visibleRows()=24, pageJump=23
	m = step(m, tea.KeyMsg{Type: tea.KeyEnter})

	// down x3 → cursor=3
	for i := 0; i < 3; i++ {
		m = step(m, tea.KeyMsg{Type: tea.KeyDown})
	}
	if m.cursor != 3 {
		t.Errorf("after 3x down: cursor=%d, want 3", m.cursor)
	}

	// pgdown → +23
	m = step(m, tea.KeyMsg{Type: tea.KeyPgDown})
	if m.cursor != 26 {
		t.Errorf("after pgdown: cursor=%d, want 26", m.cursor)
	}

	// end → last site
	m = step(m, tea.KeyMsg{Type: tea.KeyEnd})
	if m.cursor != 99 {
		t.Errorf("after end: cursor=%d, want 99", m.cursor)
	}

	// home → 0
	m = step(m, tea.KeyMsg{Type: tea.KeyHome})
	if m.cursor != 0 {
		t.Errorf("after home: cursor=%d, want 0", m.cursor)
	}

	// G → last site (vim-style)
	m = step(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if m.cursor != 99 {
		t.Errorf("after G: cursor=%d, want 99", m.cursor)
	}

	// g → first site
	m = step(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.cursor != 0 {
		t.Errorf("after g: cursor=%d, want 0", m.cursor)
	}

	// pgup at top stays at 0
	m = step(m, tea.KeyMsg{Type: tea.KeyPgUp})
	if m.cursor != 0 {
		t.Errorf("pgup at top: cursor=%d, want 0", m.cursor)
	}
}

func TestTalkgroupsTabNavigation(t *testing.T) {
	sys := manyTalkgroupsParsedSystem(1000)
	m := newImportTUI([]parsedSystem{sys}, dummyWrite)
	m.height = 30 // visibleRows()=24, pageJump=23
	m = step(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = step(m, tea.KeyMsg{Type: tea.KeyTab})

	m = step(m, tea.KeyMsg{Type: tea.KeyEnd})
	if m.cursor != 999 {
		t.Errorf("end: cursor=%d, want 999", m.cursor)
	}

	m = step(m, tea.KeyMsg{Type: tea.KeyPgUp})
	if m.cursor != 999-23 {
		t.Errorf("pgup from end: cursor=%d, want %d", m.cursor, 999-23)
	}

	m = step(m, tea.KeyMsg{Type: tea.KeyHome})
	if m.cursor != 0 {
		t.Errorf("home: cursor=%d, want 0", m.cursor)
	}
}

func TestPositionLabel(t *testing.T) {
	cases := []struct {
		name                       string
		noun                       string
		cursor, total, start, end  int
		want                       string
	}{
		{"empty", "Site", 0, 0, 0, 0, "(no sites)"},
		{"full list fits", "Site", 1, 5, 0, 5, "Site 2 of 5"},
		{"window in middle", "Talkgroup", 50, 1000, 40, 60, "Talkgroup 51 of 1000  (showing 41-60)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := positionLabel(tc.noun, tc.cursor, tc.total, tc.start, tc.end)
			if got != tc.want {
				t.Errorf("positionLabel = %q, want %q", got, tc.want)
			}
		})
	}
}

// manySitesParsedSystem returns a system stuffed with n synthetic
// sites. Used for navigation tests over large lists.
func manySitesParsedSystem(n int) parsedSystem {
	sys := parsedSystem{
		Name:     "Big",
		Protocol: "p25",
		SysID:    "X",
	}
	for i := 0; i < n; i++ {
		sys.Sites = append(sys.Sites, parsedSite{
			RFSS: 1, SiteID: i + 1, SiteName: "S", Include: true,
			Frequencies: []parsedFreq{{Hz: 851012500, ControlChannel: true}},
		})
	}
	sys.Talkgroups = []parsedTalkgroup{{Dec: 1, Hex: "1", Mode: "D", AlphaTag: "TG", Scan: true}}
	return sys
}

// manyTalkgroupsParsedSystem returns a system with n synthetic
// talkgroups (and one valid site so parseSystem-style invariants hold).
func manyTalkgroupsParsedSystem(n int) parsedSystem {
	sys := parsedSystem{
		Name:     "Big",
		Protocol: "p25",
		SysID:    "X",
		Sites: []parsedSite{{
			RFSS: 1, SiteID: 1, SiteName: "S", Include: true,
			Frequencies: []parsedFreq{{Hz: 851012500, ControlChannel: true}},
		}},
	}
	for i := 0; i < n; i++ {
		sys.Talkgroups = append(sys.Talkgroups, parsedTalkgroup{
			Dec: uint32(i + 1), Hex: "1", Mode: "D", AlphaTag: "TG", Scan: true,
		})
	}
	return sys
}

func step(m importTUIModel, msg tea.Msg) importTUIModel {
	mAny, _ := m.Update(msg)
	return mAny.(importTUIModel)
}

func dummyWrite(_ []parsedSystem) (mergeResult, error) {
	return mergeResult{}, nil
}
