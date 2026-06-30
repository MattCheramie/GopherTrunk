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
	survey     bool // the open form will start a survey rather than a hunt
	focusName  bool // false ⇒ bands input focused, true ⇒ name input
	inputErr   string
	selected   int // cursor into the survey signal inventory (capture target)
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
	huntStartKey  = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "start hunt"))
	huntSurveyKey = key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "start survey"))
	huntStopKey   = key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "stop run"))
	huntEscKey    = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel"))
	huntUpKey     = key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "prev signal"))
	huntDownKey   = key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "next signal"))
	huntCapKey    = key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "capture → SigLab"))
	huntCapCryKey = key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "capture → CryptoLab"))
)

func (HuntPanel) Keys() []key.Binding {
	return []key.Binding{huntStartKey, huntSurveyKey, huntStopKey, huntUpKey, huntDownKey, huntCapKey, huntCapCryKey}
}

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

	n := len(s.Hunt.Signals)
	switch {
	case key.Matches(km, huntStartKey) && !s.Hunt.Running:
		p.openForm(false)
		return p, textinput.Blink
	case key.Matches(km, huntSurveyKey) && !s.Hunt.Running:
		p.openForm(true)
		return p, textinput.Blink
	case key.Matches(km, huntStopKey) && s.Hunt.Running:
		return p, Emit(state.WriteRequest{Label: "stop hunt run", Kind: state.WriteKindHuntStop})
	case key.Matches(km, huntUpKey) && n > 0:
		if p.selected > 0 {
			p.selected--
		}
		return p, nil
	case key.Matches(km, huntDownKey) && n > 0:
		if p.selected < n-1 {
			p.selected++
		}
		return p, nil
	case key.Matches(km, huntCapKey) && n > 0 && !s.Hunt.Running:
		return p, p.captureSelected(s, "siglab")
	case key.Matches(km, huntCapCryKey) && n > 0 && !s.Hunt.Running:
		return p, p.captureSelected(s, "cryptolab")
	}
	return p, nil
}

// captureSelected emits a capture request for the currently-selected survey
// signal, routed to SigLab or CryptoLab.
func (p *HuntPanel) captureSelected(s *state.SharedState, target string) tea.Cmd {
	sigs := s.Hunt.Signals
	if len(sigs) == 0 {
		return nil
	}
	sel := p.selected
	if sel < 0 || sel >= len(sigs) {
		sel = 0
	}
	return Emit(state.WriteRequest{
		Label:       "capture signal",
		Kind:        state.WriteKindHuntCapture,
		HuntCapture: &state.HuntCaptureReq{FreqHz: sigs[sel].FreqHz, Target: target},
	})
}

// openForm opens the start form for either a hunt or a survey run.
func (p *HuntPanel) openForm(survey bool) {
	p.editing = true
	p.survey = survey
	p.focusName = false
	p.inputErr = ""
	p.bandsInput.Focus()
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
	survey := p.survey
	label := "start hunt"
	if survey {
		label = "start survey"
	}
	p.closeForm()
	return Emit(state.WriteRequest{
		Label: label,
		Kind:  state.WriteKindHuntStart,
		Hunt:  &state.HuntStartReq{Bands: bands, Name: name, Survey: survey},
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

	if h.Mode != "" {
		fmt.Fprintf(&b, "Mode:    %s\n", h.Mode)
	}

	b.WriteString("\n")
	if h.SystemName != "" {
		fmt.Fprintf(&b, "Discovered: %s\n", h.SystemName)
		fmt.Fprintf(&b, "  sites: %d   talkgroups: %d\n", h.Sites, h.Talkgroups)
	} else if h.Mode != "survey" {
		b.WriteString("No system discovered yet.\n")
	}

	// Survey inventory: list the classified carriers (most recent first up to
	// a height-bounded cap so the panel doesn't overflow).
	if len(h.Signals) > 0 {
		if p.selected >= len(h.Signals) {
			p.selected = len(h.Signals) - 1
		}
		fmt.Fprintf(&b, "Signals (%d):\n", len(h.Signals))
		max := height - 12
		if max < 1 {
			max = 1
		}
		for i, sig := range h.Signals {
			if i >= max {
				fmt.Fprintf(&b, "  … %d more\n", len(h.Signals)-i)
				break
			}
			cursor := "  "
			if i == p.selected {
				cursor = "> "
			}
			name := sig.Name
			if name == "" {
				name = string(sig.Class)
			}
			enc := ""
			if sig.Encrypted {
				enc = " 🔒"
			}
			fmt.Fprintf(&b, "%s%9.4f MHz  %-16s  %4.1f kHz%s\n",
				cursor, float64(sig.FreqHz)/1e6, name, float64(sig.OccupiedBwHz)/1e3, enc)
		}
		if !h.Running {
			b.WriteString("[↑/↓] select   [c] → SigLab   [C] → CryptoLab\n")
		}
	}

	b.WriteString("\n")
	switch {
	case p.editing:
		mode := "hunt"
		if p.survey {
			mode = "survey"
		}
		fmt.Fprintf(&b, "(%s) ", mode)
		b.WriteString(p.bandsInput.View() + "\n")
		b.WriteString(p.nameInput.View() + "\n")
		b.WriteString("[enter] start   [tab] next field   [esc] cancel\n")
		if p.inputErr != "" {
			fmt.Fprintf(&b, "! %s\n", p.inputErr)
		}
	case h.Running:
		b.WriteString("[x] stop run\n")
	default:
		b.WriteString("[s] hunt   [v] survey\n")
	}
	if s.HuntErr != nil {
		fmt.Fprintf(&b, "\n(status unavailable: %v)\n", s.HuntErr)
	}

	return panelFrame("Hunt", width, height, focused, b.String())
}
