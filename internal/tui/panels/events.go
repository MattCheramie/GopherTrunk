package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

// EventsPanel renders the rolling SSE event log.
type EventsPanel struct {
	vp        viewport.Model
	filter    textinput.Model
	editing   bool
	paused    bool
	lastSize  int
	lastQuery string
}

func NewEvents() *EventsPanel {
	in := textinput.New()
	in.Placeholder = "filter…"
	in.Width = 32
	in.Prompt = "/"
	return &EventsPanel{vp: viewport.New(80, 20), filter: in}
}

func (EventsPanel) Title() string { return "Events" }

var (
	eventsFilterKey = key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter"))
	eventsPauseKey  = key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pause"))
	eventsClearKey  = key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "clear filter"))
	eventsEscKey    = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "exit filter"))
)

func (EventsPanel) Keys() []key.Binding {
	return []key.Binding{eventsFilterKey, eventsPauseKey, eventsClearKey}
}

func (p *EventsPanel) Update(msg tea.Msg, s *state.SharedState) (Panel, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		if p.editing {
			if key.Matches(m, eventsEscKey) {
				p.editing = false
				p.filter.Blur()
				return p, nil
			}
			var cmd tea.Cmd
			p.filter, cmd = p.filter.Update(msg)
			return p, cmd
		}
		switch {
		case key.Matches(m, eventsFilterKey):
			p.editing = true
			p.filter.Focus()
			return p, textinput.Blink
		case key.Matches(m, eventsPauseKey):
			p.paused = !p.paused
			return p, nil
		case key.Matches(m, eventsClearKey):
			p.filter.SetValue("")
			return p, nil
		}
	}
	if !p.paused {
		p.refresh(s)
	}
	var cmd tea.Cmd
	p.vp, cmd = p.vp.Update(msg)
	return p, cmd
}

func (p *EventsPanel) refresh(s *state.SharedState) {
	q := p.filter.Value()
	if s.EventLog == nil {
		p.vp.SetContent("")
		return
	}
	all := s.EventLog.Snapshot()
	var b strings.Builder
	for _, ev := range all {
		line := fmt.Sprintf("%s  %-14s  %s",
			ev.Time.Format("15:04:05.000"),
			ev.Kind,
			summariseEvent(ev),
		)
		if q != "" && !containsFold(line, q) {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	p.vp.SetContent(b.String())
	if !p.paused {
		p.vp.GotoBottom()
	}
}

func (p *EventsPanel) View(width, height int, focused bool, s *state.SharedState) string {
	p.vp.Width = width - 4
	if height > 6 {
		p.vp.Height = height - 5
	}
	header := p.filter.View()
	if p.paused {
		header += "  " + dashAlert.Render("[paused]")
	}
	body := header + "\n" + p.vp.View()
	return panelFrame("Events", width, height, focused, body)
}

// summariseEvent extracts a human-readable single-line summary from
// an Event payload. Falls back to "—" if the payload doesn't match
// any well-known shape.
func summariseEvent(ev client.Event) string {
	switch ev.Kind {
	case "cc.locked", "cc.lost":
		var ls client.LockState
		if jsonUnmarshal(ev.Raw, &ls) == nil {
			return client.FormatFreqMHz(ls.FrequencyHz)
		}
	case "grant":
		var g client.GrantDTO
		if jsonUnmarshal(ev.Raw, &g) == nil {
			return fmt.Sprintf("TG %d  src %d  %s", g.GroupID, g.SourceID, client.FormatFreqMHz(g.FrequencyHz))
		}
	case "call.start", "call.end":
		var s struct {
			Grant     client.GrantDTO    `json:"grant"`
			Talkgroup *client.TalkgroupDTO `json:"talkgroup,omitempty"`
			Reason    string             `json:"reason,omitempty"`
		}
		if jsonUnmarshal(ev.Raw, &s) == nil {
			alpha := ""
			if s.Talkgroup != nil {
				alpha = " " + s.Talkgroup.AlphaTag
			}
			if s.Reason != "" {
				return fmt.Sprintf("TG %d%s  reason=%s", s.Grant.GroupID, alpha, s.Reason)
			}
			return fmt.Sprintf("TG %d%s  %s", s.Grant.GroupID, alpha, client.FormatFreqMHz(s.Grant.FrequencyHz))
		}
	case "tone.alert":
		var t client.Tone
		if jsonUnmarshal(ev.Raw, &t) == nil {
			return fmt.Sprintf("%s  device=%s", t.Profile, t.DeviceSerial)
		}
	}
	if len(ev.Raw) > 0 && len(ev.Raw) < 64 {
		return string(ev.Raw)
	}
	return "—"
}
