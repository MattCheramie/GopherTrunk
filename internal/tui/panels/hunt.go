package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

// HuntPanel renders the live system-discovery ("hunt") run: its state, sweep/
// identify progress, and the discovered system summary. A blind discovery is
// started from the web/API (where band/candidate parameters are entered); the
// TUI monitors the run and can stop it.
type HuntPanel struct{}

func NewHunt() *HuntPanel { return &HuntPanel{} }

func (HuntPanel) Title() string { return "Hunt" }

var huntStopKey = key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "stop run"))

func (HuntPanel) Keys() []key.Binding { return []key.Binding{huntStopKey} }

func (p *HuntPanel) Update(msg tea.Msg, s *state.SharedState) (Panel, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return p, nil
	}
	if key.Matches(km, huntStopKey) && s.Hunt.Running {
		return p, Emit(state.WriteRequest{
			Label: "stop hunt run",
			Kind:  state.WriteKindHuntStop,
		})
	}
	return p, nil
}

func (p *HuntPanel) View(width, height int, focused bool, s *state.SharedState) string {
	if width < 30 || height < 6 {
		return panelFrame("Hunt", width, height, focused, "")
	}
	h := s.Hunt
	var b strings.Builder

	runState := h.State
	if runState == "" {
		runState = "idle"
	}
	fmt.Fprintf(&b, "State:   %s", runState)
	if h.Running {
		b.WriteString("  ●")
	}
	b.WriteString("\n")
	if h.Phase != "" {
		fmt.Fprintf(&b, "Phase:   %s", h.Phase)
		if h.Detail != "" {
			fmt.Fprintf(&b, " — %s", h.Detail)
		}
		b.WriteString("\n")
	}
	if h.Error != "" {
		fmt.Fprintf(&b, "Error:   %s\n", h.Error)
	}

	b.WriteString("\n")
	if h.SystemName != "" {
		fmt.Fprintf(&b, "Discovered: %s\n", h.SystemName)
		fmt.Fprintf(&b, "  sites: %d   talkgroups: %d\n", h.Sites, h.Talkgroups)
	} else {
		b.WriteString("No system discovered yet.\n")
	}

	b.WriteString("\n")
	if h.Running {
		b.WriteString("[x] stop run\n")
	} else {
		b.WriteString("Start a run from the web console or POST /api/v1/hunt/start.\n")
	}
	if s.HuntErr != nil {
		fmt.Fprintf(&b, "\n(status unavailable: %v)\n", s.HuntErr)
	}

	return panelFrame("Hunt", width, height, focused, b.String())
}
