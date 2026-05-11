package panels

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

// DashboardPanel is the at-a-glance landing screen. It is a pure
// renderer over SharedState; it owns no local state.
type DashboardPanel struct{}

// NewDashboard returns the default landing panel.
func NewDashboard() *DashboardPanel { return &DashboardPanel{} }

func (DashboardPanel) Title() string       { return "Dashboard" }
func (DashboardPanel) Keys() []key.Binding { return nil }

func (p *DashboardPanel) Update(_ tea.Msg, _ *state.SharedState) (Panel, tea.Cmd) {
	return p, nil
}

var (
	dashHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	dashOK     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	dashErr    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	dashDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	dashAlert  = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
)

func (p *DashboardPanel) View(width, height int, _ bool, s *state.SharedState) string {
	if width < 4 || height < 4 {
		return ""
	}
	colW := (width - 2) / 2
	if colW < 20 {
		colW = width - 2
	}
	rowH := (height - 2) / 2
	if rowH < 4 {
		rowH = height - 2
	}

	tl := dashboardCard("Health", colW, rowH, p.healthBody(s))
	tr := dashboardCard("Active calls", colW, rowH, p.activeBody(s))
	bl := dashboardCard("Recent events", colW, rowH, p.eventsBody(s))
	br := dashboardCard("Tone alerts", colW, rowH, p.tonesBody(s))

	top := lipgloss.JoinHorizontal(lipgloss.Top, tl, tr)
	bot := lipgloss.JoinHorizontal(lipgloss.Top, bl, br)
	return lipgloss.JoinVertical(lipgloss.Left, top, bot)
}

func dashboardCard(title string, w, h int, body string) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(w - 2).
		Height(h - 2)
	header := dashHeader.Render(title)
	return style.Render(header + "\n" + body)
}

func (p *DashboardPanel) healthBody(s *state.SharedState) string {
	var lines []string
	switch {
	case s.HealthErr != nil:
		lines = append(lines, dashErr.Render("daemon unreachable"))
		lines = append(lines, dashDim.Render(s.HealthErr.Error()))
	case s.Health.Status == "":
		lines = append(lines, dashDim.Render("connecting…"))
	default:
		statusStyle := dashOK
		if !strings.EqualFold(s.Health.Status, "ok") {
			statusStyle = dashErr
		}
		lines = append(lines, "Status: "+statusStyle.Render(s.Health.Status))
	}
	if s.Version != "" {
		lines = append(lines, dashDim.Render("Version: "+s.Version))
	}
	if !s.Health.Now.IsZero() {
		lines = append(lines, dashDim.Render("Daemon time: "+s.Health.Now.Format(time.RFC3339)))
	}
	if s.Server != "" {
		lines = append(lines, dashDim.Render("Server: "+s.Server))
	}
	if len(s.Devices) > 0 {
		var control, voice int
		for _, d := range s.Devices {
			switch d.Role {
			case "control":
				control++
			case "voice":
				voice++
			}
		}
		lines = append(lines, dashDim.Render(
			fmt.Sprintf("SDRs: %d (%d control, %d voice)", len(s.Devices), control, voice),
		))
	}
	return strings.Join(lines, "\n")
}

func (p *DashboardPanel) activeBody(s *state.SharedState) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d active\n", len(s.ActiveCalls))
	max := 3
	if len(s.ActiveCalls) < max {
		max = len(s.ActiveCalls)
	}
	for i := 0; i < max; i++ {
		ac := s.ActiveCalls[i]
		alpha := "—"
		if ac.Talkgroup != nil && ac.Talkgroup.AlphaTag != "" {
			alpha = ac.Talkgroup.AlphaTag
		}
		fmt.Fprintf(&b, "%d  %s  %s\n", ac.Grant.GroupID, alpha, ac.Grant.System)
	}
	if len(s.ActiveCalls) == 0 {
		b.WriteString(dashDim.Render("idle"))
	}
	return b.String()
}

func (p *DashboardPanel) eventsBody(s *state.SharedState) string {
	if s.EventLog == nil || s.EventLog.Len() == 0 {
		return dashDim.Render("no events yet")
	}
	latest := s.EventLog.Latest(8)
	var b strings.Builder
	for _, ev := range latest {
		ts := ev.Time.Format("15:04:05")
		fmt.Fprintf(&b, "%s  %s\n", ts, ev.Kind)
	}
	return b.String()
}

func (p *DashboardPanel) tonesBody(s *state.SharedState) string {
	if s.ToneAlerts == nil || s.ToneAlerts.Len() == 0 {
		return dashDim.Render("no tone matches")
	}
	latest := s.ToneAlerts.Latest(5)
	var b strings.Builder
	for _, ev := range latest {
		ts := ev.Time.Format("15:04:05")
		var t client.Tone
		if err := jsonUnmarshal(ev.Raw, &t); err == nil && t.Profile != "" {
			fmt.Fprintln(&b, dashAlert.Render(ts+"  "+t.Profile)+"  "+dashDim.Render(t.DeviceSerial))
			continue
		}
		fmt.Fprintln(&b, dashAlert.Render(ts+"  tone"))
	}
	return b.String()
}
