package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

// HuntPanel renders the live system-discovery ("hunt") run — its state, sweep/
// identify progress, and the discovered system summary — and lets the operator
// start a run (bands to sweep + an optional name) or stop the active one.
type HuntPanel struct {
	bandsInput textinput.Model
	nameInput  textinput.Model
	editing    bool
	focusName  bool // false ⇒ bands input focused, true ⇒ name input
	inputErr   string
}

func NewHunt() *HuntPanel {
	bands := textinput.New()
	bands.Placeholder = "bands MHz e.g. 851:869,769:775"
	bands.CharLimit = 64
	bands.Width = 30
	bands.Prompt = "bands> "
	name := textinput.New()
	name.Placeholder = "name (optional)"
	name.CharLimit = 48
	name.Width = 30
	name.Prompt = "name>  "
	return &HuntPanel{bandsInput: bands, nameInput: name}
}

func (HuntPanel) Title() string { return "Hunt" }

var (
	huntStartKey = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "start run"))
	huntStopKey  = key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "stop run"))
	huntEscKey   = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel"))
)

func (HuntPanel) Keys() []key.Binding { return []key.Binding{huntStartKey, huntStopKey} }

func (p *HuntPanel) Update(msg tea.Msg, s *state.SharedState) (Panel, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return p, nil
	}

	// While the start form is open, keys drive the inputs. Enter submits,
	// Esc cancels, Tab switches focus.
	if p.editing {
		switch {
		case key.Matches(km, huntEscKey):
			p.closeForm()
			return p, nil
		case km.String() == "tab":
			p.toggleFocus()
			return p, nil
		case km.String() == "enter":
			return p, p.submit()
		}
		var cmd tea.Cmd
		if p.focusName {
			p.nameInput, cmd = p.nameInput.Update(msg)
		} else {
			p.bandsInput, cmd = p.bandsInput.Update(msg)
		}
		return p, cmd
	}

	switch {
	case key.Matches(km, huntStartKey) && !s.Hunt.Running:
		p.editing = true
		p.focusName = false
		p.inputErr = ""
		p.bandsInput.Focus()
		return p, textinput.Blink
	case key.Matches(km, huntStopKey) && s.Hunt.Running:
		return p, Emit(state.WriteRequest{Label: "stop hunt run", Kind: state.WriteKindHuntStop})
	}
	return p, nil
}

func (p *HuntPanel) toggleFocus() {
	p.focusName = !p.focusName
	if p.focusName {
		p.bandsInput.Blur()
		p.nameInput.Focus()
	} else {
		p.nameInput.Blur()
		p.bandsInput.Focus()
	}
}

func (p *HuntPanel) closeForm() {
	p.editing = false
	p.inputErr = ""
	p.bandsInput.SetValue("")
	p.nameInput.SetValue("")
	p.bandsInput.Blur()
	p.nameInput.Blur()
}

// submit validates the bands input and emits a WriteKindHuntStart.
func (p *HuntPanel) submit() tea.Cmd {
	bands, err := parseBandList(p.bandsInput.Value())
	if err != nil {
		p.inputErr = err.Error()
		return nil
	}
	name := strings.TrimSpace(p.nameInput.Value())
	p.closeForm()
	return Emit(state.WriteRequest{
		Label: "start hunt",
		Kind:  state.WriteKindHuntStart,
		Hunt:  &state.HuntStartReq{Bands: bands, Name: name},
	})
}

// parseBandList parses a comma-separated list of "low:high" MHz specs, lightly
// validating each so an obviously-malformed entry is caught before dispatch.
func parseBandList(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("enter at least one band (low:high MHz)")
	}
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		lo, hi, ok := strings.Cut(part, ":")
		if !ok || strings.TrimSpace(lo) == "" || strings.TrimSpace(hi) == "" {
			return nil, fmt.Errorf("band %q: want low:high in MHz", part)
		}
		out = append(out, part)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("enter at least one band (low:high MHz)")
	}
	return out, nil
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
	switch {
	case p.editing:
		b.WriteString(p.bandsInput.View() + "\n")
		b.WriteString(p.nameInput.View() + "\n")
		b.WriteString("[enter] start   [tab] next field   [esc] cancel\n")
		if p.inputErr != "" {
			fmt.Fprintf(&b, "! %s\n", p.inputErr)
		}
	case h.Running:
		b.WriteString("[x] stop run\n")
	default:
		b.WriteString("[s] start a sweep   (or POST /api/v1/hunt/start)\n")
	}
	if s.HuntErr != nil {
		fmt.Fprintf(&b, "\n(status unavailable: %v)\n", s.HuntErr)
	}

	return panelFrame("Hunt", width, height, focused, b.String())
}
