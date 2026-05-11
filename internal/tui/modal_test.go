package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/panels"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

// newWriteModel returns a model with --write enabled and a daemon
// that has reported allow_mutations=true. Used by the modal-flow
// tests below.
func newWriteModel(t *testing.T) *Model {
	t.Helper()
	cli := client.New("http://example.invalid", time.Second, false)
	m := New(cli, Options{NoColor: true, Write: true})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(*Model)
	updated, _ = m.Update(pollMutationStatusMsg{
		s: client.MutationStatus{AllowMutations: true, EngineWritable: true},
	})
	return updated.(*Model)
}

func TestWriteAction_DisabledShowsToast(t *testing.T) {
	cli := client.New("http://example.invalid", time.Second, false)
	m := New(cli, Options{NoColor: true, Write: false})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(*Model)
	// Daemon disagrees too; WriteEnabled stays false.
	updated, _ = m.Update(pollMutationStatusMsg{s: client.MutationStatus{AllowMutations: false}})
	m = updated.(*Model)

	req := state.WriteRequest{
		Confirm: "test",
		Label:   "test action",
		Kind:    state.WriteKindEndCall,
		EndCall: &state.EndCallReq{DeviceSerial: "abc"},
	}
	updated, cmd := m.Update(panels.WriteActionMsg{Request: req})
	m = updated.(*Model)
	if cmd != nil {
		t.Errorf("disabled write should not yield a Cmd")
	}
	if m.confirm != nil {
		t.Errorf("disabled write should not open modal")
	}
	if !strings.Contains(m.shared.Toast, "mutations disabled") {
		t.Errorf("toast = %q", m.shared.Toast)
	}
}

func TestWriteAction_OpensConfirmModal(t *testing.T) {
	m := newWriteModel(t)
	req := state.WriteRequest{
		Confirm: "End call on device abc?",
		Label:   "ended call on abc",
		Kind:    state.WriteKindEndCall,
		EndCall: &state.EndCallReq{DeviceSerial: "abc"},
	}
	updated, cmd := m.Update(panels.WriteActionMsg{Request: req})
	m = updated.(*Model)
	if m.confirm == nil {
		t.Fatal("confirm modal not opened")
	}
	if !strings.Contains(m.confirm.prompt, "abc") {
		t.Errorf("prompt = %q", m.confirm.prompt)
	}
	if cmd != nil {
		t.Errorf("modal-pending request should defer Cmd, got non-nil")
	}
}

func TestModal_EscCancels(t *testing.T) {
	m := newWriteModel(t)
	// Open a modal.
	updated, _ := m.Update(panels.WriteActionMsg{Request: state.WriteRequest{
		Confirm: "Sweep?",
		Label:   "sweep",
		Kind:    state.WriteKindSweepRetention,
		SweepRetention: &state.SweepRetentionReq{},
	}})
	m = updated.(*Model)
	if m.confirm == nil {
		t.Fatal("modal not opened")
	}
	// ESC closes without firing.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(*Model)
	if m.confirm != nil {
		t.Errorf("esc should clear modal")
	}
	if cmd != nil {
		t.Errorf("esc should not yield a Cmd, got %T", cmd)
	}
	if !strings.Contains(m.shared.Toast, "cancel") {
		t.Errorf("toast = %q", m.shared.Toast)
	}
}

func TestModal_YConfirmsAndDispatches(t *testing.T) {
	m := newWriteModel(t)
	// Open a modal that resolves to an EndCall request — the
	// dispatcher will build a Cmd for it. The Cmd will fail at
	// runtime (no daemon at example.invalid) but we only assert
	// that a Cmd is returned, not its result.
	updated, _ := m.Update(panels.WriteActionMsg{Request: state.WriteRequest{
		Confirm: "End?",
		Label:   "ended",
		Kind:    state.WriteKindEndCall,
		EndCall: &state.EndCallReq{DeviceSerial: "abc"},
	}})
	m = updated.(*Model)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(*Model)
	if m.confirm != nil {
		t.Errorf("enter should clear modal")
	}
	if cmd == nil {
		t.Fatal("enter should yield a Cmd to fire the action")
	}
}

func TestModal_SwallowsOtherKeys(t *testing.T) {
	m := newWriteModel(t)
	updated, _ := m.Update(panels.WriteActionMsg{Request: state.WriteRequest{
		Confirm: "?", Label: "x",
		Kind:           state.WriteKindSweepRetention,
		SweepRetention: &state.SweepRetentionReq{},
	}})
	m = updated.(*Model)
	before := m.active
	// Pressing a digit (which would normally switch panels) is swallowed.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("5")})
	m = updated.(*Model)
	if m.active != before {
		t.Errorf("modal should swallow panel-switch key; active changed %v->%v", before, m.active)
	}
	if m.confirm == nil {
		t.Errorf("modal closed unexpectedly on '5'")
	}
}

func TestWriteResult_TogglesPollRefresh(t *testing.T) {
	m := newWriteModel(t)
	updated, cmd := m.Update(writeResultMsg{Label: "test"})
	m = updated.(*Model)
	if cmd == nil {
		t.Fatal("writeResultMsg should schedule refresh Cmds")
	}
	if !strings.Contains(m.shared.Toast, "test ok") {
		t.Errorf("toast = %q", m.shared.Toast)
	}
}

func TestDetailModal_OpensFromSystemResult(t *testing.T) {
	cli := client.New("http://example.invalid", time.Second, false)
	m := New(cli, Options{NoColor: true})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(*Model)

	updated, _ = m.Update(systemDetailResultMsg{
		s: client.SystemDTO{Name: "Demo", Protocol: "p25", ControlChannels: []uint32{851012500}},
	})
	m = updated.(*Model)
	if m.detail == nil {
		t.Fatal("expected detail modal to open")
	}
	if !strings.Contains(m.detail.title, "Demo") {
		t.Errorf("title = %q, want Demo", m.detail.title)
	}
	if !strings.Contains(m.detail.body, "p25") {
		t.Errorf("body missing protocol: %q", m.detail.body)
	}
	// Esc closes it.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(*Model)
	if m.detail != nil {
		t.Errorf("esc should close detail modal")
	}
	if cmd != nil {
		t.Errorf("esc should not yield a Cmd")
	}
}

func TestDetailModal_ErrorShowsToast(t *testing.T) {
	cli := client.New("http://example.invalid", time.Second, false)
	m := New(cli, Options{NoColor: true})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(*Model)

	updated, _ = m.Update(systemDetailResultMsg{err: &client.HTTPError{Status: 404, Method: "GET", URL: "/api/v1/systems/missing"}})
	m = updated.(*Model)
	if m.detail != nil {
		t.Errorf("error result should not open modal")
	}
	if !strings.Contains(m.shared.Toast, "system:") {
		t.Errorf("toast = %q", m.shared.Toast)
	}
}
