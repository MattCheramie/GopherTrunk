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
	// Write enables the TUI's mutation keybindings. AND-ed with
	// the daemon's /api/v1/mutations capability — both must agree
	// for write-side keys to fire.
	Write bool
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

	confirm *confirmModal
	detail  *detailModal

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
			panels.NewDevices(),
			panels.NewScanner(),
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
		cmdPollDevices(m.cli),
		cmdPollScanner(m.cli),
		cmdMutationStatus(m.cli),
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
		// Modal capture: when a confirmation is pending, every key
		// goes to the modal — global nav and panels are frozen.
		if m.confirm != nil {
			switch msg.String() {
			case "y", "Y", "enter":
				req := m.confirm.req
				m.confirm = nil
				return m, m.dispatchWrite(req)
			case "n", "N", "esc":
				m.confirm = nil
				m.toast("cancelled")
				return m, nil
			}
			return m, nil // swallow other keys while modal is up
		}
		// Detail modal: dismissable with esc / q / enter; all other
		// keys are swallowed so the modal stays focused.
		if m.detail != nil {
			switch msg.String() {
			case "esc", "q", "enter":
				m.detail = nil
				return m, nil
			}
			return m, nil
		}
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
		case key.Matches(msg, m.keys.JumpPanel9):
			m.active = state.PanelDevices
			return m, nil
		case key.Matches(msg, m.keys.JumpPanel0):
			m.active = state.PanelScanner
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

	case pollDevicesMsg:
		m.shared.Devices = msg.devs
		m.shared.DevicesErr = msg.err
		cmds = append(cmds, scheduleAfter(pollDevicesEvery, cmdPollDevices(m.cli)))

	case pollScannerMsg:
		if msg.err == nil {
			m.shared.Scanner = msg.s
		}
		m.shared.ScannerErr = msg.err
		cmds = append(cmds, scheduleAfter(pollScannerEvery, cmdPollScanner(m.cli)))

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
		switch msg.ev.Kind {
		case "tone.alert":
			tones, _ := m.shared.ToneAlerts.(*RingBuf[client.Event])
			if tones != nil {
				tones.Push(msg.ev)
			}
		case "sdr.attached", "sdr.detached":
			// Refresh the devices snapshot now rather than waiting
			// for the next poll tick.
			cmds = append(cmds, cmdPollDevices(m.cli))
		case "call.start", "call.end":
			cmds = append(cmds, cmdPollActive(m.cli))
			cmds = append(cmds, cmdPollScanner(m.cli))
		case "cchunt.progress", "cchunt.failed", "cc.locked", "cc.lost":
			cmds = append(cmds, cmdPollScanner(m.cli))
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

	case pollMutationStatusMsg:
		// /api/v1/mutations only fails on a network error; an
		// older daemon that doesn't know the route yields a
		// zero-value status. WriteEnabled is the AND of the
		// daemon's allow_mutations and the TUI's --write flag.
		m.shared.Mutations = msg.s
		m.shared.WriteEnabled = m.opts.Write && msg.s.AllowMutations

	case panels.WriteActionMsg:
		if !m.shared.WriteEnabled {
			m.toast("mutations disabled — pass --write to the TUI and api.allow_mutations: true to the daemon")
			return m, nil
		}
		// requestConfirm either fires the request now or stashes
		// it for the modal. The returned Cmd is non-nil only on
		// the immediate-fire path.
		if cmd := m.requestConfirm(msg.Request); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case panels.SystemDetailMsg:
		cmds = append(cmds, cmdFetchSystemDetail(m.cli, msg.Name))

	case panels.TalkgroupDetailMsg:
		cmds = append(cmds, cmdFetchTalkgroupDetail(m.cli, msg.ID))

	case systemDetailResultMsg:
		if msg.err != nil {
			m.toast(fmt.Sprintf("system: %v", msg.err))
		} else {
			m.openDetail("System: "+msg.s.Name, formatSystemDetail(msg.s))
		}

	case talkgroupDetailResultMsg:
		if msg.err != nil {
			m.toast(fmt.Sprintf("talkgroup: %v", msg.err))
		} else {
			m.openDetail(fmt.Sprintf("Talkgroup %d", msg.tg.ID), formatTalkgroupDetail(msg.tg))
		}

	case writeResultMsg:
		if msg.Err != nil {
			m.toast(fmt.Sprintf("%s: %v", msg.Label, msg.Err))
		} else if msg.Label != "" {
			m.toast(msg.Label + " ok")
		}
		// Refresh the dependent surfaces immediately so the UI
		// reflects the mutation without waiting for the next
		// poll tick.
		cmds = append(cmds,
			cmdPollActive(m.cli),
			cmdPollTalkgroups(m.cli),
		)
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
	full := lipgloss.JoinVertical(lipgloss.Left, tabs, body, status)
	if m.activeModal() {
		return m.renderModal(m.width, m.height, full)
	}
	return full
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

// dispatchWrite turns a state.WriteRequest into the matching write
// Cmd. The Cmd's outcome arrives back as writeResultMsg, which the
// Update loop surfaces as a toast and uses to refresh dependent
// reads.
func (m *Model) dispatchWrite(r state.WriteRequest) tea.Cmd {
	switch r.Kind {
	case state.WriteKindEndCall:
		if r.EndCall == nil {
			return nil
		}
		return cmdEndCall(m.cli, r.EndCall.DeviceSerial, r.EndCall.Reason, r.Label)
	case state.WriteKindUpdateTalkgroup:
		if r.UpdateTalkgroup == nil {
			return nil
		}
		return cmdUpdateTalkgroup(m.cli,
			r.UpdateTalkgroup.ID,
			r.UpdateTalkgroup.Priority,
			r.UpdateTalkgroup.Lockout,
			r.UpdateTalkgroup.Scan,
			r.Label)
	case state.WriteKindSweepRetention:
		return cmdSweepRetention(m.cli)
	case state.WriteKindResetTone:
		if r.ResetTone == nil {
			return nil
		}
		return cmdResetTone(m.cli, r.ResetTone.DeviceSerial)
	case state.WriteKindScannerMode:
		if r.ScannerMode == nil {
			return nil
		}
		return cmdScannerSetMode(m.cli, r.ScannerMode.Mode, r.Label)
	case state.WriteKindScannerHuntHold:
		if r.ScannerHunt == nil {
			return nil
		}
		return cmdScannerHuntHold(m.cli, r.ScannerHunt.System, r.Label)
	case state.WriteKindScannerHuntResume:
		if r.ScannerHunt == nil {
			return nil
		}
		return cmdScannerHuntResume(m.cli, r.ScannerHunt.System, r.Label)
	case state.WriteKindScannerHuntRetune:
		if r.ScannerHunt == nil {
			return nil
		}
		return cmdScannerHuntRetune(m.cli, r.ScannerHunt.System, r.Label)
	case state.WriteKindScannerConvHold:
		return cmdScannerConvHold(m.cli, r.Label)
	case state.WriteKindScannerConvResume:
		return cmdScannerConvResume(m.cli, r.Label)
	case state.WriteKindScannerConvDwell:
		if r.ScannerConv == nil {
			return nil
		}
		return cmdScannerConvDwell(m.cli, r.ScannerConv.Index, r.Label)
	}
	return nil
}
