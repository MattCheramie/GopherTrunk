package panels

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

func TestSettingsPanel_EditableFieldsHidenWithoutConfig(t *testing.T) {
	p := NewSettings()
	s := &state.SharedState{Runtime: client.RuntimeDTO{}}
	// No ConfigPath → render path skips the editable rows entirely.
	view := p.View(120, 30, true, s)
	if strings.Contains(view, "log.level") {
		t.Errorf("editable rows should be hidden when ConfigPath is empty; view:\n%s", view)
	}
}

func TestSettingsPanel_EditableFieldsShownWithConfig(t *testing.T) {
	p := NewSettings()
	s := &state.SharedState{Runtime: client.RuntimeDTO{
		ConfigPath: "/tmp/cfg.yaml",
		LogLevel:   "info",
	}}
	view := p.View(120, 30, true, s)
	if !strings.Contains(view, "Log level") {
		t.Errorf("expected Log level row when ConfigPath is set; view:\n%s", view)
	}
}

func TestSettingsPanel_EditDispatchesWriteAction(t *testing.T) {
	p := NewSettings()
	s := &state.SharedState{Runtime: client.RuntimeDTO{
		ConfigPath: "/tmp/cfg.yaml",
		LogLevel:   "info",
	}}
	// Enter promotes the focused row to edit mode.
	_, _ = p.Update(tea.KeyMsg{Type: tea.KeyEnter}, s)
	if !p.editing {
		t.Fatal("expected editing=true after Enter")
	}
	// Replace seeded value with "warn".
	p.editInput.SetValue("warn")
	// Enter again saves — should return a Cmd that emits a WriteActionMsg.
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter}, s)
	if cmd == nil {
		t.Fatal("expected commit cmd, got nil")
	}
	msg := cmd()
	wa, ok := msg.(WriteActionMsg)
	if !ok {
		t.Fatalf("got %T, want WriteActionMsg", msg)
	}
	if wa.Request.Kind != state.WriteKindSettings {
		t.Errorf("got Kind=%v, want WriteKindSettings", wa.Request.Kind)
	}
	if wa.Request.Settings == nil {
		t.Fatal("Settings req is nil")
	}
	if wa.Request.Settings.Field != "log.level" {
		t.Errorf("got Field=%q want log.level", wa.Request.Settings.Field)
	}
	if wa.Request.Settings.Value != "warn" {
		t.Errorf("got Value=%q want warn", wa.Request.Settings.Value)
	}
}

func TestSettingsPanel_EditCancelEscape(t *testing.T) {
	p := NewSettings()
	s := &state.SharedState{Runtime: client.RuntimeDTO{
		ConfigPath: "/tmp/cfg.yaml",
		LogLevel:   "info",
	}}
	_, _ = p.Update(tea.KeyMsg{Type: tea.KeyEnter}, s)
	if !p.editing {
		t.Fatal("editing should be true")
	}
	_, _ = p.Update(tea.KeyMsg{Type: tea.KeyEsc}, s)
	if p.editing {
		t.Error("editing should be false after esc")
	}
}

func TestSettingsPanel_SettingsErrorMsgSurfacesInline(t *testing.T) {
	p := NewSettings()
	s := &state.SharedState{Runtime: client.RuntimeDTO{
		ConfigPath: "/tmp/cfg.yaml",
		LogLevel:   "info",
	}}
	_, _ = p.Update(SettingsErrorMsg{Field: "log.level", Message: "boom"}, s)
	if p.editErr != "boom" {
		t.Errorf("editErr = %q want boom", p.editErr)
	}
	view := p.View(120, 30, true, s)
	if !strings.Contains(view, "boom") {
		t.Errorf("view should include the inline error; got:\n%s", view)
	}
}

func TestSettingsPanel_DownArrowMovesCursor(t *testing.T) {
	p := NewSettings()
	s := &state.SharedState{Runtime: client.RuntimeDTO{
		ConfigPath: "/tmp/cfg.yaml",
		LogLevel:   "info",
	}}
	if p.editCursor != 0 {
		t.Fatal("editCursor should start at 0")
	}
	_, _ = p.Update(tea.KeyMsg{Type: tea.KeyDown}, s)
	if p.editCursor != 1 {
		t.Errorf("editCursor=%d want 1", p.editCursor)
	}
}
