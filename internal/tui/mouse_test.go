package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/panels"
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

func TestHitTestTab_BodyClickDelegatesToActivePanel(t *testing.T) {
	m := newTestModel(t)
	m.width = 200
	m.height = 24
	_ = m.renderTabs()
	// Land on Systems so the click feeds into a MouseAware panel.
	m.active = state.PanelSystems
	m.shared.Systems = []client.SystemDTO{
		{Name: "A", Protocol: "p25"},
		{Name: "B", Protocol: "dmr"},
		{Name: "C", Protocol: "nxdn"},
	}
	// One Update tick populates the systems table.
	_, _ = m.panels[state.PanelSystems].Update(tea.WindowSizeMsg{Width: m.width, Height: m.height}, m.shared)

	// Click on body row 1: global Y == 1 (tab strip) + 3 (border, title, col header) + 1 = 5.
	msg := tea.MouseMsg{
		X: 5, Y: 5,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}
	_, consumed := m.handleMouseMsg(msg)
	if !consumed {
		t.Fatal("body click on MouseAware panel should be consumed")
	}
	if got := m.panels[state.PanelSystems].(*panels.SystemsPanel); got.Cursor() != 1 {
		t.Errorf("systems cursor = %d, want 1", got.Cursor())
	}
}

func TestHitTestTab_BodyClickWithoutMouseAware(t *testing.T) {
	m := newTestModel(t)
	m.width = 200
	m.height = 24
	_ = m.renderTabs()
	// Dashboard isn't MouseAware — body clicks should not be consumed.
	m.active = state.PanelDashboard
	msg := tea.MouseMsg{
		X: 5, Y: 8,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}
	_, consumed := m.handleMouseMsg(msg)
	if consumed {
		t.Errorf("dashboard body click should be left unconsumed")
	}
}

func TestHandleMouseMsg_WheelForwardsToActivePanel(t *testing.T) {
	m := newTestModel(t)
	m.width = 200
	m.height = 24
	_ = m.renderTabs()
	m.active = state.PanelSystems
	m.shared.Systems = []client.SystemDTO{
		{Name: "A"}, {Name: "B"}, {Name: "C"},
	}
	_, _ = m.panels[state.PanelSystems].Update(tea.WindowSizeMsg{Width: m.width, Height: m.height}, m.shared)

	wheelDown := tea.MouseMsg{
		X: 5, Y: 5,
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	}
	_, consumed := m.handleMouseMsg(wheelDown)
	if !consumed {
		t.Fatal("wheel event should be consumed by MouseAware panel")
	}
	if got := m.panels[state.PanelSystems].(*panels.SystemsPanel).Cursor(); got != 1 {
		t.Errorf("systems cursor = %d after wheel-down, want 1", got)
	}
}

func TestHandleMouseMsg_ReleaseAndMotionIgnored(t *testing.T) {
	m := newTestModel(t)
	m.width = 200
	m.height = 24
	_ = m.renderTabs()
	m.active = state.PanelSystems

	for _, action := range []tea.MouseAction{tea.MouseActionRelease, tea.MouseActionMotion} {
		msg := tea.MouseMsg{X: 5, Y: 5, Button: tea.MouseButtonLeft, Action: action}
		_, consumed := m.handleMouseMsg(msg)
		if consumed {
			t.Errorf("action %v should not be consumed", action)
		}
	}
}
