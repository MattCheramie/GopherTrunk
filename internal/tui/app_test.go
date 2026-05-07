package tui

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

func newTestModel(t *testing.T) *Model {
	t.Helper()
	cli := client.New("http://example.invalid", time.Second, false)
	m := New(cli, Options{NoColor: true})
	// Fake a window-size so View doesn't bail.
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return updated.(*Model)
}

func TestPollActiveMsg_PopulatesShared(t *testing.T) {
	m := newTestModel(t)
	updated, _ := m.Update(pollActiveMsg{
		calls: []client.ActiveCallDTO{{
			Grant: client.GrantDTO{System: "Demo", GroupID: 42, FrequencyHz: 851012500},
			Talkgroup: &client.TalkgroupDTO{ID: 42, AlphaTag: "Dispatch"},
			DeviceSerial: "00000001",
			StartedAt:    time.Now(),
		}},
	})
	m = updated.(*Model)
	if len(m.shared.ActiveCalls) != 1 {
		t.Fatalf("ActiveCalls len = %d", len(m.shared.ActiveCalls))
	}
	out := m.View()
	if !strings.Contains(out, "Dispatch") {
		t.Errorf("Dispatch not in dashboard view")
	}
}

func TestEventMsg_LandsInRingBuffers(t *testing.T) {
	m := newTestModel(t)
	tonePayload, _ := json.Marshal(client.Tone{Profile: "two-tone", DeviceSerial: "abc"})
	ev := client.Event{Kind: "tone.alert", Time: time.Now(), Raw: tonePayload}
	updated, _ := m.Update(eventMsg{ev: ev})
	m = updated.(*Model)
	if m.shared.EventLog.Len() != 1 {
		t.Errorf("EventLog len = %d", m.shared.EventLog.Len())
	}
	if m.shared.ToneAlerts.Len() != 1 {
		t.Errorf("ToneAlerts len = %d", m.shared.ToneAlerts.Len())
	}
}

func TestSSEDownMsg_SchedulesReconnect(t *testing.T) {
	m := newTestModel(t)
	_, cmd := m.Update(sseDownMsg{})
	if cmd == nil {
		t.Fatal("sseDownMsg should yield a reconnect Cmd")
	}
}

func TestPanelSwitch_DigitAndTab(t *testing.T) {
	m := newTestModel(t)
	cases := []struct {
		key  string
		want state.PanelKind
	}{
		{"3", state.PanelTalkgroups},
		{"5", state.PanelHistory},
		{"8", state.PanelMetrics},
	}
	for _, c := range cases {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(c.key)})
		m = updated.(*Model)
		if m.active != c.want {
			t.Errorf("after %q active=%v, want %v", c.key, m.active, c.want)
		}
	}
	// Tab cycles forward
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(*Model)
	if m.active != state.PanelDashboard {
		t.Errorf("Tab from Metrics: active=%v, want Dashboard", m.active)
	}
}
