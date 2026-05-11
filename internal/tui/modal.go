package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

// confirmModal is the modal overlay shown when a panel issues a
// WriteRequest with Confirm != "". It captures keyboard focus until
// the operator presses y/Enter (run the action) or n/Esc (cancel).
type confirmModal struct {
	prompt string
	label  string
	req    state.WriteRequest
}

// detailModal is the read-only modal opened when a panel asks for a
// drill-in. Dismissed with esc/q/enter; all other keys are swallowed
// so the modal stays the focused surface until explicitly closed.
type detailModal struct {
	title string
	body  string
}

// activeModal returns true when any modal (confirm or detail) is up.
func (m *Model) activeModal() bool { return m.confirm != nil || m.detail != nil }

// requestConfirm stores the request and triggers the modal overlay,
// or runs the request immediately when Confirm is empty.
func (m *Model) requestConfirm(r state.WriteRequest) tea.Cmd {
	if r.Confirm == "" {
		return m.dispatchWrite(r)
	}
	m.confirm = &confirmModal{prompt: r.Confirm, label: r.Label, req: r}
	return nil
}

// openDetail shows a read-only drill-in card. Subsequent calls
// replace the current detail (e.g. user opens a system, then a
// talkgroup — the second one wins).
func (m *Model) openDetail(title, body string) {
	m.detail = &detailModal{title: title, body: body}
}

// renderModal draws a centered modal box over the body. Confirm
// modals win over detail modals when both happen to be set, since the
// confirm flow is a stronger interrupt.
func (m *Model) renderModal(width, height int, behind string) string {
	if m.confirm != nil {
		return m.renderConfirmModal(width, height, behind)
	}
	if m.detail != nil {
		return m.renderDetailModal(width, height, behind)
	}
	return behind
}

func (m *Model) renderConfirmModal(width, height int, _ string) string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(1, 2).
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("231"))

	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")).Render("Confirm action")
	body := m.confirm.prompt
	footer := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("y/enter: confirm   n/esc: cancel")
	contents := strings.Join([]string{header, "", body, "", footer}, "\n")
	rendered := box.Render(contents)

	return lipgloss.Place(width, height,
		lipgloss.Center, lipgloss.Center,
		rendered,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("236")),
	)
}

func (m *Model) renderDetailModal(width, height int, _ string) string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("231"))

	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render(m.detail.title)
	footer := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("esc/q/enter: close")
	contents := strings.Join([]string{header, "", m.detail.body, "", footer}, "\n")
	rendered := box.Render(contents)

	return lipgloss.Place(width, height,
		lipgloss.Center, lipgloss.Center,
		rendered,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("236")),
	)
}
