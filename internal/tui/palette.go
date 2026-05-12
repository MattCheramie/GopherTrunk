package tui

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/panels"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
	"github.com/MattCheramie/GopherTrunk/internal/tui/theme"
)

// paletteAction is one entry in the command palette. Label is what
// the operator sees + fuzzy-matches against; Run is the side effect
// fired on Enter. The Kind field is a coarse category used only for
// inline tagging in the palette list ("nav" / "audio" / "scanner" /
// "talkgroup" / "system" / "device" / "help").
type paletteAction struct {
	ID    string
	Label string
	Kind  string
	Hint  string
	Run   func(*Model) tea.Cmd
}

// paletteModel is the textinput + ranked list overlay opened with
// Ctrl+P. It rebuilds its action list each time it opens so newly-
// arrived systems / talkgroups / devices appear without a re-launch.
type paletteModel struct {
	open     bool
	cursor   int
	input    textinput.Model
	all      []paletteAction
	filtered []paletteAction
}

func newPalette() *paletteModel {
	in := textinput.New()
	in.Placeholder = "search panels, mutations, systems, talkgroups…"
	in.Prompt = " "
	in.CharLimit = 80
	in.Width = 48
	return &paletteModel{input: in}
}

// openPalette rebuilds the action list against the current shared
// state and focuses the input.
func (m *Model) openPalette() {
	if m.palette == nil {
		m.palette = newPalette()
	}
	m.palette.all = m.discoverActions()
	m.palette.input.SetValue("")
	m.palette.input.Focus()
	m.palette.cursor = 0
	m.palette.open = true
	m.palette.refilter()
}

// closePalette hides the overlay and unfocuses the input.
func (m *Model) closePalette() {
	if m.palette == nil {
		return
	}
	m.palette.open = false
	m.palette.input.Blur()
}

// refilter recomputes filtered using the current query.
func (pm *paletteModel) refilter() {
	q := strings.ToLower(strings.TrimSpace(pm.input.Value()))
	if q == "" {
		pm.filtered = append(pm.filtered[:0], pm.all...)
		if pm.cursor >= len(pm.filtered) {
			pm.cursor = 0
		}
		return
	}
	type scored struct {
		a paletteAction
		s int
	}
	out := make([]scored, 0, len(pm.all))
	for _, a := range pm.all {
		if s := fuzzyScore(strings.ToLower(a.Label), q); s > 0 {
			out = append(out, scored{a: a, s: s})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].s > out[j].s })
	pm.filtered = pm.filtered[:0]
	for _, s := range out {
		pm.filtered = append(pm.filtered, s.a)
	}
	if pm.cursor >= len(pm.filtered) {
		pm.cursor = 0
	}
}

// fuzzyScore returns a small positive integer when query's characters
// appear in order inside label, 0 otherwise. Prefix matches + word-
// start matches earn bonus points so "act" ranks "Active" above
// "Scanner activities".
func fuzzyScore(label, query string) int {
	if query == "" {
		return 1
	}
	if strings.HasPrefix(label, query) {
		return 1000 + len(query)
	}
	score := 0
	li := 0
	prevSep := true
	for _, qc := range query {
		found := false
		for li < len(label) {
			lc := rune(label[li])
			if lc == qc {
				score += 10
				if prevSep {
					score += 20 // word-start bonus
				}
				li++
				prevSep = false
				found = true
				break
			}
			if unicode.IsSpace(lc) || lc == '_' || lc == '-' || lc == ':' {
				prevSep = true
			} else {
				prevSep = false
			}
			li++
		}
		if !found {
			return 0
		}
	}
	return score
}

// handlePaletteKey routes a key event while the palette is open. The
// caller has already established m.palette.open == true.
func (m *Model) handlePaletteKey(km tea.KeyMsg) tea.Cmd {
	switch km.String() {
	case "esc", "ctrl+p":
		m.closePalette()
		return nil
	case "enter":
		if a, ok := m.paletteSelected(); ok {
			m.closePalette()
			if a.Run != nil {
				return a.Run(m)
			}
		}
		return nil
	case "up", "ctrl+k":
		if m.palette.cursor > 0 {
			m.palette.cursor--
		}
		return nil
	case "down", "ctrl+j":
		if m.palette.cursor < len(m.palette.filtered)-1 {
			m.palette.cursor++
		}
		return nil
	}
	var cmd tea.Cmd
	m.palette.input, cmd = m.palette.input.Update(km)
	m.palette.refilter()
	return cmd
}

func (m *Model) paletteSelected() (paletteAction, bool) {
	if m.palette == nil || !m.palette.open || len(m.palette.filtered) == 0 {
		return paletteAction{}, false
	}
	i := m.palette.cursor
	if i < 0 || i >= len(m.palette.filtered) {
		return paletteAction{}, false
	}
	return m.palette.filtered[i], true
}

// discoverActions builds the action list against the current shared
// state. Each call re-enumerates so post-startup arrivals (new
// talkgroup file load, new SDR hotplug) are reachable without
// closing/reopening the palette twice.
func (m *Model) discoverActions() []paletteAction {
	out := make([]paletteAction, 0, 64)

	// 1) Panel jumps.
	for i := state.PanelKind(0); i < state.PanelCount; i++ {
		i := i
		out = append(out, paletteAction{
			ID:    fmt.Sprintf("jump:%d", i),
			Label: fmt.Sprintf("Jump to %s", m.panels[i].Title()),
			Kind:  "nav",
			Hint:  fmt.Sprintf("%d", int(i)+1),
			Run: func(mm *Model) tea.Cmd {
				mm.active = i
				return nil
			},
		})
	}

	// 2) Help / theme toggle / quit.
	out = append(out, paletteAction{
		ID:    "help",
		Label: "Show keyboard reference",
		Kind:  "help",
		Hint:  "?",
		Run: func(mm *Model) tea.Cmd {
			mm.help.ShowAll = true
			return nil
		},
	})
	out = append(out, paletteAction{
		ID:    "theme:toggle",
		Label: "Toggle theme (dark / monochrome)",
		Kind:  "theme",
		Hint:  "ctrl+t",
		Run: func(mm *Model) tea.Cmd {
			return mm.toggleTheme()
		},
	})
	out = append(out, paletteAction{
		ID:    "quit",
		Label: "Quit GopherTrunk TUI",
		Kind:  "help",
		Hint:  "q",
		Run: func(mm *Model) tea.Cmd {
			if mm.sseCancel != nil {
				mm.sseCancel()
			}
			return tea.Quit
		},
	})

	// 3) Audio mutations (only useful when --write is enabled, but
	// listing them unconditionally keeps the palette consistent;
	// the actions short-circuit on m.shared.WriteEnabled).
	addAudio := func(label, hint string, run func(*Model) tea.Cmd) {
		out = append(out, paletteAction{
			ID: "audio:" + label, Label: "Audio: " + label, Kind: "audio", Hint: hint, Run: run,
		})
	}
	addAudio("volume up 5%", "+", paletteAudioVol(+0.05))
	addAudio("volume down 5%", "-", paletteAudioVol(-0.05))
	addAudio("toggle mute", "M", paletteAudioMute())
	addAudio("toggle recording", "R", paletteAudioRecord())

	// 4) Retention sweep.
	out = append(out, paletteAction{
		ID:    "retention:sweep",
		Label: "Retention: run sweep now",
		Kind:  "retention",
		Run: func(mm *Model) tea.Cmd {
			req := state.WriteRequest{
				Confirm:        "Run a retention sweep now?",
				Label:          "retention sweep",
				Kind:           state.WriteKindSweepRetention,
				SweepRetention: &state.SweepRetentionReq{},
			}
			return func() tea.Msg { return panels.WriteActionMsg{Request: req} }
		},
	})

	// 5) Scanner mutations — scan-mode toggle + per-system holds.
	out = append(out, paletteAction{
		ID:    "scanner:toggle-mode",
		Label: "Scanner: toggle scan mode (all/list)",
		Kind:  "scanner",
		Run: func(mm *Model) tea.Cmd {
			next := "list"
			if mm.shared.Scanner.ScanMode == "list" {
				next = "all"
			}
			req := state.WriteRequest{
				Label:       fmt.Sprintf("set scan_mode=%s", next),
				Kind:        state.WriteKindScannerMode,
				ScannerMode: &state.ScannerModeReq{Mode: next},
			}
			return func() tea.Msg { return panels.WriteActionMsg{Request: req} }
		},
	})
	for _, sys := range m.shared.Scanner.Systems {
		sys := sys
		out = append(out, paletteAction{
			ID:    "scanner:hold:" + sys.Name,
			Label: "Scanner: hold/resume hunt — " + sys.Name,
			Kind:  "scanner",
			Run: func(mm *Model) tea.Cmd {
				mm.revealOn(state.PanelScanner, "sys:"+sys.Name)
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
				return func() tea.Msg { return panels.WriteActionMsg{Request: req} }
			},
		})
	}

	// 6) System / talkgroup / device drill-ins — handy nav. Each
	// action jumps to the relevant panel, pre-positions the panel's
	// cursor on the matching row via the Revealer interface, and
	// fires the detail modal where applicable.
	for _, sys := range m.shared.Systems {
		sys := sys
		out = append(out, paletteAction{
			ID:    "system:" + sys.Name,
			Label: "Open system: " + sys.Name + " (" + sys.Protocol + ")",
			Kind:  "system",
			Run: func(mm *Model) tea.Cmd {
				mm.active = state.PanelSystems
				mm.revealOn(state.PanelSystems, sys.Name)
				return func() tea.Msg { return panels.SystemDetailMsg{Name: sys.Name} }
			},
		})
	}
	// Cap talkgroup actions to the first 200 — bigger lists drown
	// out the palette.
	tgLimit := len(m.shared.Talkgroups)
	if tgLimit > 200 {
		tgLimit = 200
	}
	for _, tg := range m.shared.Talkgroups[:tgLimit] {
		tg := tg
		alpha := tg.AlphaTag
		if alpha == "" {
			alpha = "—"
		}
		out = append(out, paletteAction{
			ID:    fmt.Sprintf("tg:%d", tg.ID),
			Label: fmt.Sprintf("Open TG %d  %s", tg.ID, alpha),
			Kind:  "talkgroup",
			Run: func(mm *Model) tea.Cmd {
				mm.active = state.PanelTalkgroups
				mm.revealOn(state.PanelTalkgroups, fmt.Sprintf("%d", tg.ID))
				return func() tea.Msg { return panels.TalkgroupDetailMsg{ID: tg.ID} }
			},
		})
	}
	for _, dev := range m.shared.Devices {
		dev := dev
		out = append(out, paletteAction{
			ID:    "device:" + dev.Serial,
			Label: fmt.Sprintf("SDR: %s  %s  %s", dev.Serial, dev.Driver, dev.TunerName),
			Kind:  "device",
			Run: func(mm *Model) tea.Cmd {
				mm.active = state.PanelDevices
				mm.revealOn(state.PanelDevices, dev.Serial)
				return nil
			},
		})
	}

	return out
}

// revealOn calls Reveal(key) on the target panel if it implements
// panels.Revealer. Silent no-op otherwise — Revealer is opt-in.
func (m *Model) revealOn(kind state.PanelKind, key string) {
	if int(kind) < 0 || int(kind) >= len(m.panels) {
		return
	}
	if r, ok := m.panels[kind].(panels.Revealer); ok {
		r.Reveal(key)
	}
}

func paletteAudioVol(delta float32) func(*Model) tea.Cmd {
	return func(mm *Model) tea.Cmd {
		v := mm.shared.Audio.Volume + delta
		if v < 0 {
			v = 0
		}
		if v > 1 {
			v = 1
		}
		req := state.WriteRequest{
			Label: fmt.Sprintf("volume %d%%", int(v*100+0.5)),
			Kind:  state.WriteKindAudio,
			Audio: &state.AudioReq{Volume: &v},
		}
		return func() tea.Msg { return panels.WriteActionMsg{Request: req} }
	}
}

func paletteAudioMute() func(*Model) tea.Cmd {
	return func(mm *Model) tea.Cmd {
		next := !mm.shared.Audio.Muted
		req := state.WriteRequest{
			Label: ifElse(next, "mute", "unmute"),
			Kind:  state.WriteKindAudio,
			Audio: &state.AudioReq{Muted: &next},
		}
		return func() tea.Msg { return panels.WriteActionMsg{Request: req} }
	}
}

func paletteAudioRecord() func(*Model) tea.Cmd {
	return func(mm *Model) tea.Cmd {
		next := !mm.shared.Audio.RecordingEnabled
		req := state.WriteRequest{
			Label: ifElse(next, "recording on", "recording off"),
			Kind:  state.WriteKindAudio,
			Audio: &state.AudioReq{Recording: &next},
		}
		return func() tea.Msg { return panels.WriteActionMsg{Request: req} }
	}
}

func ifElse(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

// renderPalette draws the overlay. Width is clamped so the overlay
// stays readable on terminals as narrow as 40 cols. Up to 12 rows
// shown.
func (m *Model) renderPalette(width, height int, _ string) string {
	pm := m.palette
	p := theme.Theme()
	w := width * 3 / 4
	if w < 50 {
		w = 50
	}
	if w > 100 {
		w = 100
	}
	header := p.Title().Render("Command palette")
	hint := p.Hint().Render("type to filter • ↑/↓ to move • Enter to run • Esc to close")
	box := p.ModalBox("info").Width(w)

	var rows []string
	rows = append(rows, header, "", pm.input.View(), hint, "")
	// Render up to 12 hits.
	const maxRows = 12
	end := len(pm.filtered)
	if end > maxRows {
		end = maxRows
	}
	for i := 0; i < end; i++ {
		a := pm.filtered[i]
		line := a.Label
		if a.Hint != "" {
			line += "  " + p.Hint().Render("("+a.Hint+")")
		}
		if a.Kind != "" {
			line = p.Hint().Render(fmt.Sprintf("[%-9s]", a.Kind)) + " " + line
		}
		if i == pm.cursor {
			line = p.Selected().Render(" ▶ " + line)
		} else {
			line = "   " + line
		}
		rows = append(rows, line)
	}
	if end == 0 {
		rows = append(rows, p.Hint().Render("  (no matches)"))
	} else if len(pm.filtered) > end {
		rows = append(rows, p.Hint().Render(fmt.Sprintf("  +%d more — refine your search", len(pm.filtered)-end)))
	}
	body := strings.Join(rows, "\n")
	rendered := box.Render(body)
	return lipgloss.Place(width, height,
		lipgloss.Center, lipgloss.Center,
		rendered,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(p.BgAlt),
	)
}

// ifElseClient and clientShortRef keep the imports honest — the
// palette references client types via the shared state but Go's
// import-pruner will complain otherwise. (No-op at runtime.)
var _ = client.Event{}
