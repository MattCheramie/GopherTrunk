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

// ScannerPanel is the police-scanner cockpit. Renders three sections:
//
//  1. Systems (trunked) — per-system CC hunt state + last grant.
//  2. Conventional — fixed-frequency analog channels + current dwell.
//  3. TG scan summary — global scan_mode + scan-list size.
//
// Operator mutations (hold / resume / retune / dwell / cycle scan
// mode) ride the existing WriteRequest machinery so the daemon's
// allow_mutations gate + the TUI's --write flag govern them.
type ScannerPanel struct {
	// cursor selects one of two enumerable rows: a trunked system
	// (Systems section) or a conventional channel (Conv section).
	// We treat both sections as one virtual list keyed by
	// (section, index); cursorAt yields the selected slot.
	cursor int
}

// NewScanner returns a new read+write scanner cockpit.
func NewScanner() *ScannerPanel {
	return &ScannerPanel{}
}

func (ScannerPanel) Title() string { return "Scanner" }

var (
	scanHoldKey   = key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "hold/resume"))
	scanRetuneKey = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "force re-hunt"))
	scanDwellKey  = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "dwell on conv channel"))
	scanModeKey   = key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "cycle scan mode"))
	scanUpKey     = key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("k/↑", "row up"))
	scanDownKey   = key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("j/↓", "row down"))
	scanVolUpKey  = key.NewBinding(key.WithKeys("+", "="), key.WithHelp("+", "volume up"))
	scanVolDnKey  = key.NewBinding(key.WithKeys("-", "_"), key.WithHelp("-", "volume down"))
	scanMuteKey   = key.NewBinding(key.WithKeys("M"), key.WithHelp("M", "mute toggle"))
	scanRecKey    = key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "record toggle"))
)

func (ScannerPanel) Keys() []key.Binding {
	return []key.Binding{
		scanHoldKey, scanRetuneKey, scanDwellKey, scanModeKey,
		scanUpKey, scanDownKey,
		scanVolUpKey, scanVolDnKey, scanMuteKey, scanRecKey,
	}
}

// volumeStep is the increment per +/− press. 0.05 maps to 20 presses
// edge-to-edge — matches the analog volume-knob muscle memory of a
// handheld scanner without overshooting on a keyboard.
const volumeStep = 0.05

func (p *ScannerPanel) rowCount(s *state.SharedState) int {
	return len(s.Scanner.Systems) + len(s.Scanner.Conventional.Channels)
}

// resolveCursor returns (section, indexWithinSection). section is
// "sys" or "conv". Returns ("", 0) when there are no rows.
func (p *ScannerPanel) resolveCursor(s *state.SharedState) (string, int) {
	nSys := len(s.Scanner.Systems)
	nConv := len(s.Scanner.Conventional.Channels)
	if nSys+nConv == 0 {
		return "", 0
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
	if p.cursor >= nSys+nConv {
		p.cursor = nSys + nConv - 1
	}
	if p.cursor < nSys {
		return "sys", p.cursor
	}
	return "conv", p.cursor - nSys
}

func (p *ScannerPanel) Update(msg tea.Msg, s *state.SharedState) (Panel, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return p, nil
	}
	section, idx := p.resolveCursor(s)
	switch {
	case key.Matches(km, scanUpKey):
		if p.cursor > 0 {
			p.cursor--
		}
		return p, nil
	case key.Matches(km, scanDownKey):
		if p.cursor < p.rowCount(s)-1 {
			p.cursor++
		}
		return p, nil
	case key.Matches(km, scanModeKey):
		next := "list"
		if s.Scanner.ScanMode == "list" {
			next = "all"
		}
		req := state.WriteRequest{
			Label:       fmt.Sprintf("set scan_mode=%s", next),
			Kind:        state.WriteKindScannerMode,
			ScannerMode: &state.ScannerModeReq{Mode: next},
		}
		return p, Emit(req)
	case key.Matches(km, scanHoldKey):
		switch section {
		case "sys":
			sys := s.Scanner.Systems[idx]
			kind := state.WriteKindScannerHuntHold
			verb := "hold"
			if sys.State == "held" {
				kind = state.WriteKindScannerHuntResume
				verb = "resume"
			}
			req := state.WriteRequest{
				Label:       fmt.Sprintf("%s hunt %s", verb, sys.Name),
				Kind:        kind,
				ScannerHunt: &state.ScannerHuntReq{System: sys.Name},
			}
			return p, Emit(req)
		case "conv":
			kind := state.WriteKindScannerConvHold
			verb := "hold"
			if s.Scanner.Conventional.State == "held" {
				kind = state.WriteKindScannerConvResume
				verb = "resume"
			}
			req := state.WriteRequest{
				Label:       fmt.Sprintf("%s conventional scanner", verb),
				Kind:        kind,
				ScannerConv: &state.ScannerConvReq{},
			}
			return p, Emit(req)
		}
	case key.Matches(km, scanRetuneKey):
		if section == "sys" {
			sys := s.Scanner.Systems[idx]
			req := state.WriteRequest{
				Confirm:     fmt.Sprintf("Force re-hunt for system %s?", sys.Name),
				Label:       fmt.Sprintf("force re-hunt %s", sys.Name),
				Kind:        state.WriteKindScannerHuntRetune,
				ScannerHunt: &state.ScannerHuntReq{System: sys.Name},
			}
			return p, Emit(req)
		}
	case key.Matches(km, scanDwellKey):
		if section == "conv" {
			ch := s.Scanner.Conventional.Channels[idx]
			req := state.WriteRequest{
				Label:       fmt.Sprintf("dwell on %s (%d Hz)", ch.Label, ch.FrequencyHz),
				Kind:        state.WriteKindScannerConvDwell,
				ScannerConv: &state.ScannerConvReq{Index: idx},
			}
			return p, Emit(req)
		}
	case key.Matches(km, scanVolUpKey):
		v := clampVolume(s.Audio.Volume + volumeStep)
		req := state.WriteRequest{
			Label: fmt.Sprintf("volume %d%%", int(v*100+0.5)),
			Kind:  state.WriteKindAudio,
			Audio: &state.AudioReq{Volume: &v},
		}
		return p, Emit(req)
	case key.Matches(km, scanVolDnKey):
		v := clampVolume(s.Audio.Volume - volumeStep)
		req := state.WriteRequest{
			Label: fmt.Sprintf("volume %d%%", int(v*100+0.5)),
			Kind:  state.WriteKindAudio,
			Audio: &state.AudioReq{Volume: &v},
		}
		return p, Emit(req)
	case key.Matches(km, scanMuteKey):
		next := !s.Audio.Muted
		label := "unmute"
		if next {
			label = "mute"
		}
		req := state.WriteRequest{
			Label: label,
			Kind:  state.WriteKindAudio,
			Audio: &state.AudioReq{Muted: &next},
		}
		return p, Emit(req)
	case key.Matches(km, scanRecKey):
		next := !s.Audio.RecordingEnabled
		label := "recording off"
		if next {
			label = "recording on"
		}
		req := state.WriteRequest{
			Label: label,
			Kind:  state.WriteKindAudio,
			Audio: &state.AudioReq{Recording: &next},
		}
		return p, Emit(req)
	}
	return p, nil
}

// clampVolume keeps the +/− stepping inside the legal 0..1 range.
func clampVolume(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func (p *ScannerPanel) View(width, height int, focused bool, s *state.SharedState) string {
	_, _ = p.resolveCursor(s) // clamp cursor
	if width < 30 || height < 6 {
		return panelFrame("Scanner", width, height, focused, "")
	}
	body := strings.Join([]string{
		p.renderSystems(width, s),
		p.renderConventional(width, s),
		p.renderTGSummary(width, s),
		p.renderAudio(width, s),
	}, "\n")
	return panelFrame("Scanner", width, height, focused, body)
}

func (p *ScannerPanel) renderAudio(width int, s *state.SharedState) string {
	header := dashHeader.Render("Audio")
	a := s.Audio
	state := "off"
	style := dashDim
	if a.BackendEnabled {
		state = "on"
		style = dashOK
		if a.Muted {
			state = "muted"
			style = dashErr
		}
	}
	rec := "off"
	if a.RecordingEnabled {
		rec = "on"
	}
	line := fmt.Sprintf("  output=%s  vol=%d%%  rec=%s   (+/- volume, M mute, R record)",
		style.Render(state),
		int(a.Volume*100+0.5),
		rec,
	)
	return "\n" + header + "\n" + dashDim.Render(line)
}

func (p *ScannerPanel) renderSystems(width int, s *state.SharedState) string {
	header := dashHeader.Render("Trunked systems")
	if len(s.Scanner.Systems) == 0 {
		return header + "\n" + dashDim.Render("  (no trunked systems configured)")
	}
	lines := []string{header}
	section, idx := p.resolveCursor(s)
	for i, sys := range s.Scanner.Systems {
		marker := "  "
		if section == "sys" && idx == i {
			marker = "▶ "
		}
		stateStyle := lipgloss.NewStyle()
		switch sys.State {
		case "locked":
			stateStyle = dashOK
		case "hunting":
			stateStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
		case "failed":
			stateStyle = dashErr
		case "held":
			stateStyle = dashDim
		}
		freq := "—"
		switch sys.State {
		case "locked":
			freq = client.FormatFreqMHz(sys.LockedFreqHz)
		case "hunting":
			freq = fmt.Sprintf("%s (%d/%d)", client.FormatFreqMHz(sys.AttemptedFreqHz),
				sys.AttemptIndex+1, sys.TotalCandidates)
		case "failed":
			freq = fmt.Sprintf("retry in %s", formatBackoff(sys.BackoffMs))
		}
		grantAge := "—"
		if !sys.LastGrantAt.IsZero() {
			grantAge = formatAge(time.Since(sys.LastGrantAt))
		}
		row := fmt.Sprintf("%s%-12s  %-5s  %s  %s  last grant: %s",
			marker, sys.Name, sys.Protocol, stateStyle.Render(padState(sys.State)), freq, grantAge)
		lines = append(lines, row)
	}
	return strings.Join(lines, "\n")
}

func (p *ScannerPanel) renderConventional(width int, s *state.SharedState) string {
	header := dashHeader.Render("Conventional channels")
	cs := s.Scanner.Conventional
	if !cs.Enabled || len(cs.Channels) == 0 {
		return "\n" + header + "\n" + dashDim.Render("  (conventional scanner not configured)")
	}
	lines := []string{"", header}
	stateLine := fmt.Sprintf("  state: %s  device: %s", cs.State, cs.DeviceSerial)
	lines = append(lines, dashDim.Render(stateLine))
	section, idx := p.resolveCursor(s)
	for i, ch := range cs.Channels {
		marker := "  "
		if section == "conv" && idx == i {
			marker = "▶ "
		}
		active := " "
		style := lipgloss.NewStyle()
		if ch.Active {
			active = "●"
			style = dashOK
		}
		break_ := "—"
		if !ch.LastBreakAt.IsZero() {
			break_ = formatAge(time.Since(ch.LastBreakAt))
		}
		row := fmt.Sprintf("%s%-3d %s  %s  %-12s  %-4s  last: %s",
			marker, ch.Index, style.Render(active), padTo(ch.Label, 20),
			client.FormatFreqMHz(ch.FrequencyHz), strings.ToUpper(ch.Mode), break_)
		lines = append(lines, row)
	}
	return strings.Join(lines, "\n")
}

func (p *ScannerPanel) renderTGSummary(width int, s *state.SharedState) string {
	header := dashHeader.Render("Talkgroup scan list")
	mode := s.Scanner.ScanMode
	if mode == "" {
		mode = "all"
	}
	summary := fmt.Sprintf("  mode=%s   enabled=%d / total=%d   (press 'm' to cycle)",
		mode, s.Scanner.TalkgroupScanCount, s.Scanner.TalkgroupTotalCount)
	return "\n" + header + "\n" + dashDim.Render(summary)
}

func padState(s string) string {
	const w = 8
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

// padTo pads s to width n with trailing spaces; truncates with an
// ellipsis if longer.
func padTo(s string, n int) string {
	t := truncate(s, n)
	if len(t) < n {
		return t + strings.Repeat(" ", n-len(t))
	}
	return t
}

func formatAge(d time.Duration) string {
	if d < time.Second {
		return "now"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh ago", int(d.Hours()))
}

func formatBackoff(ms int) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%ds", ms/1000)
}
