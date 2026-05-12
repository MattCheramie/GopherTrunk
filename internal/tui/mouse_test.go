package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

func TestHitTestTab_SelectsClickedTab(t *testing.T) {
	m := newTestModel(t)
	m.width = 200 // wide enough for the full strip
	m.height = 24
	_ = m.renderTabs() // populate m.tabRects

	if len(m.tabRects) == 0 {
		t.Fatal("renderTabs populated no rects")
	}
	// Click the Talkgroups tab (index 2 / PanelTalkgroups).
	target := m.tabRects[state.PanelTalkgroups]
	msg := tea.MouseMsg{
		X:      target.xStart + 1,
		Y:      0,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}
	_, consumed := m.handleMouseMsg(msg)
	if !consumed {
		t.Fatal("click on tab strip was not consumed")
	}
	if m.active != state.PanelTalkgroups {
		t.Errorf("active = %v, want %v", m.active, state.PanelTalkgroups)
	}
}

func TestHitTestTab_IgnoresBodyClicks(t *testing.T) {
	m := newTestModel(t)
	m.width = 200
	m.height = 24
	_ = m.renderTabs()
	original := m.active
	msg := tea.MouseMsg{
		X: 5, Y: 8, // below the tab strip
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}
	_, consumed := m.handleMouseMsg(msg)
	if consumed {
		t.Fatal("body click should not be consumed by tab hit-test")
	}
	if m.active != original {
		t.Errorf("active changed unexpectedly: %v → %v", original, m.active)
	}
}
