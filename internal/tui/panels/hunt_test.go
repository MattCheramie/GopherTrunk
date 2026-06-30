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

func TestHuntPanel_CaptureSelectedSignal(t *testing.T) {
	p := NewHunt()
	s := &state.SharedState{Hunt: client.HuntStatusDTO{
		State: "done",
		Signals: []client.HuntSignalDTO{
			{FreqHz: 462_562_500, Class: "nbfm", Name: "FRS"},
			{FreqHz: 751_000_000, Class: "wideband", Name: "LTE/5G NR (10 MHz)"},
		},
	}}

	// Move the cursor to the second signal, then capture it to CryptoLab.
	_, _ = p.Update(tea.KeyMsg{Type: tea.KeyDown}, s)
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")}, s)
	if cmd == nil {
		t.Fatal("C should emit a capture request")
	}
	wa := cmd().(WriteActionMsg)
	if wa.Request.Kind != state.WriteKindHuntCapture {
		t.Fatalf("kind = %v, want HuntCapture", wa.Request.Kind)
	}
	if wa.Request.HuntCapture == nil || wa.Request.HuntCapture.FreqHz != 751_000_000 {
		t.Errorf("capture = %+v, want freq 751_000_000 (the selected row)", wa.Request.HuntCapture)
	}
	if wa.Request.HuntCapture.Target != "cryptolab" {
		t.Errorf("target = %q, want cryptolab", wa.Request.HuntCapture.Target)
	}
}

func TestHuntPanel_CaptureSuppressedWhileRunning(t *testing.T) {
	p := NewHunt()
	s := &state.SharedState{Hunt: client.HuntStatusDTO{
		Running: true, State: "running",
		Signals: []client.HuntSignalDTO{{FreqHz: 462_562_500, Class: "nbfm"}},
	}}
	if _, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")}, s); cmd != nil {
		t.Error("capture must not fire while a run is active (SDR busy)")
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
