package panels

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

func TestHuntPanel_StartFormEmitsRequest(t *testing.T) {
	p := NewHunt()
	s := &state.SharedState{Hunt: client.HuntStatusDTO{}} // idle

	// Open the form, type a band spec, submit.
	_, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}, s)
	_, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("851:869")}, s)
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter}, s)
	if cmd == nil {
		t.Fatal("enter should emit a start WriteRequest")
	}
	wa, ok := cmd().(WriteActionMsg)
	if !ok {
		t.Fatalf("msg = %T, want WriteActionMsg", cmd())
	}
	if wa.Request.Kind != state.WriteKindHuntStart {
		t.Errorf("kind = %v, want HuntStart", wa.Request.Kind)
	}
	if wa.Request.Hunt == nil || len(wa.Request.Hunt.Bands) != 1 || wa.Request.Hunt.Bands[0] != "851:869" {
		t.Errorf("bands = %+v, want [851:869]", wa.Request.Hunt)
	}
}

func TestHuntPanel_EmptyBandsShowsError(t *testing.T) {
	p := NewHunt()
	s := &state.SharedState{}
	_, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}, s)
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter}, s) // submit empty
	if cmd != nil {
		t.Errorf("empty bands should not emit a request")
	}
	if p.inputErr == "" {
		t.Errorf("expected an input error for empty bands")
	}
}

func TestHuntPanel_StopEmitsWhenRunning(t *testing.T) {
	p := NewHunt()
	s := &state.SharedState{Hunt: client.HuntStatusDTO{Running: true, State: "running"}}
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}, s)
	if cmd == nil {
		t.Fatal("x should emit a stop request while running")
	}
	wa := cmd().(WriteActionMsg)
	if wa.Request.Kind != state.WriteKindHuntStop {
		t.Errorf("kind = %v, want HuntStop", wa.Request.Kind)
	}
}
