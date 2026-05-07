// Package tui is the GopherTrunk TUI — a read-only operator view
// over the daemon's REST + SSE API. The root Model dispatches
// keystrokes to one of eight panels and runs a fan of polling Cmds
// + a long-lived SSE pump to keep its SharedState fresh.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/panels"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

// Options controls the TUI's startup behaviour.
type Options struct {
	NoColor bool
}

// Model is the root bubbletea model.
type Model struct {
	cli    *client.Client
	opts   Options
	styles styles
	keys   globalKeys
	help   help.Model

	width, height int
	active        state.PanelKind
	panels        []panels.Panel
	shared        *state.SharedState

	eventCh    <-chan client.Event
	sseCancel  func()
	sseRetries int

	historyLoaded bool
	toastUntil    time.Time
}

// New constructs a Model pointed at cli.
func New(cli *client.Client, opts Options) *Model {
	st := newStyles(opts.NoColor)
	shared := &state.SharedState{
		EventLog:   NewRingBuf[client.Event](500),
		ToneAlerts: NewRingBuf[client.Event](100),
		Server:     cli.Base(),
		Metrics:    map[string]float64{},
	}
	m := &Model{
		cli:    cli,
		opts:   opts,
		styles: st,
		keys:   newGlobalKeys(),
		help:   help.New(),
		shared: shared,
		panels: []panels.Panel{
			panels.NewDashboard(),
			panels.NewSystems(),
			panels.NewTalkgroups(),
			panels.NewActive(),
			panels.NewHistory(),
			panels.NewEvents(),
			panels.NewTones(),
			panels.NewMetrics(),
		},
	}
	return m
}

// Init kicks off the initial polling fan + SSE connect.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		cmdPollHealth(m.cli),
		cmdPollVersion(m.cli),
		cmdPollSystems(m.cli),
		cmdPollTalkgroups(m.cli),
		cmdPollActive(m.cli),
		cmdPollMetrics(m.cli),
		cmdPollHistory(m.cli, client.HistoryFilter{Limit: 100}),
		connectSSE(m.cli),
	)
}

// Update is the bubbletea reducer. Order matters: window/quit/help
// are handled at the root, then SSE/poll msgs update SharedState,
// then the active panel gets the message.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		_ = m // handled below
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		// Quit & global navigation always win.
		switch {
		case key.Matches(msg, m.keys.Quit):
			if m.sseCancel != nil {
				m.sseCancel()
			}
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		case key.Matches(msg, m.keys.NextPanel):
			m.active = (m.active + 1) % state.PanelCount
			return m, nil
		case key.Matches(msg, m.keys.PrevPanel):
			m.active = (m.active + state.PanelCount - 1) % state.PanelCount
			return m, nil
		case key.Matches(msg, m.keys.JumpPanel1):
			m.active = state.PanelDashboard
			return m, nil
		case key.Matches(msg, m.keys.JumpPanel2):
			m.active = state.PanelSystems
			return m, nil
		case key.Matches(msg, m.keys.JumpPanel3):
			m.active = state.PanelTalkgroups
			return m, nil
		case key.Matches(msg, m.keys.JumpPanel4):
			m.active = state.PanelActive
			return m, nil
		case key.Matches(msg, m.keys.JumpPanel5):
			m.active = state.PanelHistory
			return m, nil
		case key.Matches(msg, m.keys.JumpPanel6):
			m.active = state.PanelEvents
			return m, nil
		case key.Matches(msg, m.keys.JumpPanel7):
			m.active = state.PanelTones
			return m, nil
		case key.Matches(msg, m.keys.JumpPanel8):
			m.active = state.PanelMetrics
			return m, nil
		}

	case pollHealthMsg:
		m.shared.Health = msg.h
		m.shared.HealthErr = msg.err
		if msg.err != nil {
			m.toast(fmt.Sprintf("health: %v", msg.err))
		}
		cmds = append(cmds, scheduleAfter(pollHealthEvery, cmdPollHealth(m.cli)))

	case pollVersionMsg:
		if msg.err == nil {
			m.shared.Version = msg.v
		}

	case pollSystemsMsg:
		if msg.err == nil {
			m.shared.Systems = msg.s
		} else {
			m.toast(fmt.Sprintf("systems: %v", msg.err))
		}
		cmds = append(cmds, scheduleAfter(pollSystemsEvery, cmdPollSystems(m.cli)))

	case pollTalkgroupsMsg:
		if msg.err == nil {
			m.shared.Talkgroups = msg.tg
		} else {
			m.toast(fmt.Sprintf("talkgroups: %v", msg.err))
		}
		cmds = append(cmds, scheduleAfter(pollTalkgroupsEvery, cmdPollTalkgroups(m.cli)))

	case pollActiveMsg:
		if msg.err == nil {
			m.shared.ActiveCalls = msg.calls
			m.shared.LastPoll = time.Now()
		} else {
			m.toast(fmt.Sprintf("active: %v", msg.err))
		}
		cmds = append(cmds, scheduleAfter(pollActiveEvery, cmdPollActive(m.cli)))

	case pollMetricsMsg:
		if msg.err == nil {
			m.shared.Metrics = msg.m
		}
		cmds = append(cmds, scheduleAfter(pollMetricsEvery, cmdPollMetrics(m.cli)))

	case pollHistoryMsg:
		m.shared.History = msg.rows
		m.shared.HistoryErr = msg.err
		m.historyLoaded = true

	case sseUpMsg:
		if m.sseCancel != nil {
			m.sseCancel()
		}
		m.eventCh = msg.ch
		m.sseCancel = msg.cancel
		m.sseRetries = 0
		cmds = append(cmds, listenSSE(m.eventCh))

	case eventMsg:
		ring, _ := m.shared.EventLog.(*RingBuf[client.Event])
		if ring != nil {
			ring.Push(msg.ev)
		}
		if msg.ev.Kind == "tone.alert" {
			tones, _ := m.shared.ToneAlerts.(*RingBuf[client.Event])
			if tones != nil {
				tones.Push(msg.ev)
			}
		}
		cmds = append(cmds, listenSSE(m.eventCh))

	case sseDownMsg:
		m.sseRetries++
		backoff := time.Duration(1<<m.sseRetries) * time.Second
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
		m.toast("event stream disconnected — reconnecting in " + backoff.Truncate(time.Second).String())
		cmds = append(cmds, tea.Tick(backoff, func(time.Time) tea.Msg { return connectSSE(m.cli)() }))
	}

	// Forward to active panel.
	updated, cmd := m.panels[m.active].Update(msg, m.shared)
	m.panels[m.active] = updated
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	// History panel reload-on-demand.
	if m.active == state.PanelHistory {
		if hp, ok := m.panels[state.PanelHistory].(*panels.HistoryPanel); ok && hp.ReloadRequested() {
			cmds = append(cmds, cmdPollHistory(m.cli, client.HistoryFilter{Limit: 200}))
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the chrome (tabs, status bar) and delegates the body
// to the active panel.
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "starting…"
	}
	tabs := m.renderTabs()
	status := m.renderStatusBar()
	bodyH := m.height - lipgloss.Height(tabs) - lipgloss.Height(status)
	if bodyH < 4 {
		bodyH = 4
	}
	body := m.panels[m.active].View(m.width, bodyH, true, m.shared)
	return lipgloss.JoinVertical(lipgloss.Left, tabs, body, status)
}

func (m *Model) renderTabs() string {
	parts := make([]string, 0, state.PanelCount)
	for i := state.PanelKind(0); i < state.PanelCount; i++ {
		label := fmt.Sprintf("%d %s", int(i)+1, m.panels[i].Title())
		if i == m.active {
			parts = append(parts, m.styles.activeTab.Render(label))
		} else {
			parts = append(parts, m.styles.tab.Render(label))
		}
	}
	return strings.Join(parts, " ")
}

func (m *Model) renderStatusBar() string {
	left := fmt.Sprintf("server=%s", m.shared.Server)
	if m.shared.HealthErr != nil {
		left = m.styles.error.Render("● ") + left
	} else if m.shared.Health.Status != "" {
		left = m.styles.ok.Render("● ") + left
	}
	right := fmt.Sprintf("active=%d  events=%d  tones=%d",
		len(m.shared.ActiveCalls),
		m.shared.EventLog.Len(),
		m.shared.ToneAlerts.Len())
	help := m.styles.help.Render("tab:next  ?:help  q:quit")
	toast := ""
	if m.shared.Toast != "" && time.Now().Before(m.toastUntil) {
		toast = m.styles.toast.Render(m.shared.Toast)
	}
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - lipgloss.Width(help) - lipgloss.Width(toast) - 4
	if gap < 1 {
		gap = 1
	}
	bar := left + strings.Repeat(" ", gap) + toast + "  " + right + "  " + help
	return m.styles.statusBar.Width(m.width).Render(bar)
}

func (m *Model) toast(s string) {
	m.shared.Toast = s
	m.toastUntil = time.Now().Add(4 * time.Second)
}
