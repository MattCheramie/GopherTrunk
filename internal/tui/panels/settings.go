package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

// settingsTab enumerates the inspector's sub-views. Each tab pulls
// exclusively from SharedState.Runtime (the /api/v1/runtime snapshot)
// or SharedState.Systems (already polled) — so the tab switch is a
// pure render, no extra fetches.
type settingsTab int

const (
	tabDaemon settingsTab = iota
	tabStorage
	tabAudio
	tabRecording
	tabTones
	tabAPI
	tabVocoders
	tabSDR
	tabFEC
	tabCount
)

func (t settingsTab) String() string {
	switch t {
	case tabDaemon:
		return "Daemon"
	case tabStorage:
		return "Storage"
	case tabAudio:
		return "Audio"
	case tabRecording:
		return "Recording"
	case tabTones:
		return "Tones"
	case tabAPI:
		return "API"
	case tabVocoders:
		return "Vocoders"
	case tabSDR:
		return "SDR"
	case tabFEC:
		return "FEC"
	}
	return "?"
}

// SettingsPanel is the tabbed inspector that surfaces every config
// knob, output, protocol surface, and runtime fact. Driven by
// SharedState.Runtime; one /api/v1/runtime poll feeds every tab so
// the operator sees a coherent snapshot.
type SettingsPanel struct {
	tab settingsTab

	// FEC tab retains the existing table view; the other tabs
	// render plain text. The tab summarises each system's FEC
	// state — every protocol's chain is on by default, so the
	// table is operationally an opt-out reference.
	tbl      table.Model
	lastHash uint64

	// Editing state. editCursor indexes into editableFieldsForTab
	// for the current tab; editing flips to true while the operator
	// is typing into editInput. editErr surfaces validation failures
	// from the daemon's PATCH response (caught at the root model and
	// passed back via a panel message — see ApplySettingsError).
	editCursor int
	editing    bool
	editInput  textinput.Model
	editErr    string
}

// SettingsErrorMsg is dispatched by the root model when a Settings
// PATCH comes back with an error (validation failure, 503, etc.).
// The panel re-opens the editor on the failing row and surfaces the
// message inline so the operator can correct without re-navigating.
type SettingsErrorMsg struct {
	Field   string
	Message string
}

func NewSettings() *SettingsPanel {
	t := table.New(
		table.WithColumns(settingsColumns(80)),
		table.WithFocused(true),
	)
	t.SetStyles(tableStyles())
	return &SettingsPanel{tab: tabDaemon, tbl: t}
}

func (SettingsPanel) Title() string { return "Settings" }

var (
	settingsNextTab = key.NewBinding(key.WithKeys("]", "l", "right"), key.WithHelp("]/l/→", "next tab"))
	settingsPrevTab = key.NewBinding(key.WithKeys("[", "h", "left"), key.WithHelp("[/h/←", "prev tab"))
)

func (SettingsPanel) Keys() []key.Binding {
	return []key.Binding{
		settingsNextTab, settingsPrevTab,
		editUpKey, editDownKey, editStart, editCancel,
	}
}

func (p *SettingsPanel) Update(msg tea.Msg, s *state.SharedState) (Panel, tea.Cmd) {
	applyThemeIfChanged(msg, &p.tbl)

	// SettingsErrorMsg comes back from the root model when a
	// PATCH /api/v1/settings dispatch failed — re-focus the row +
	// surface the error inline.
	if em, ok := msg.(SettingsErrorMsg); ok {
		p.editErr = em.Message
		return p, nil
	}

	fields := editableFieldsForTab(p.tab)
	editable := len(fields) > 0 && s.Runtime.ConfigPath != ""

	if km, ok := msg.(tea.KeyMsg); ok {
		// Tab switching happens regardless of edit state — the
		// existing UX guarantees []/h/l always navigate tabs. Fall
		// through so the FEC-tab refresh below still runs when the
		// operator lands on that tab.
		switched := false
		switch {
		case key.Matches(km, settingsNextTab):
			p.editing = false
			p.editErr = ""
			p.tab = (p.tab + 1) % tabCount
			p.editCursor = 0
			switched = true
		case key.Matches(km, settingsPrevTab):
			p.editing = false
			p.editErr = ""
			p.tab = (p.tab + tabCount - 1) % tabCount
			p.editCursor = 0
			switched = true
		}
		if switched {
			// Skip the edit-mode handling but keep the FEC-tab
			// refresh further down so the table populates on
			// landing.
			goto fecRefresh
		}

		if editable {
			if p.editing {
				// Edit mode: route everything to the textinput
				// except enter (save) and esc (cancel).
				switch {
				case key.Matches(km, editStart):
					return p, p.commitEdit(fields)
				case key.Matches(km, editCancel):
					p.editing = false
					p.editErr = ""
					return p, nil
				}
				var cmd tea.Cmd
				p.editInput, cmd = p.editInput.Update(msg)
				return p, cmd
			}
			// Navigation mode.
			switch {
			case key.Matches(km, editUpKey):
				if p.editCursor > 0 {
					p.editCursor--
				}
				return p, nil
			case key.Matches(km, editDownKey):
				if p.editCursor < len(fields)-1 {
					p.editCursor++
				}
				return p, nil
			case key.Matches(km, editStart):
				p.startEdit(fields, s.Runtime)
				return p, nil
			}
		}
	}

fecRefresh:
	// Refresh the FEC table whenever we're on (or have just switched
	// to) the FEC tab — the hash gate keeps the cost negligible.
	if p.tab == tabFEC {
		h := hashRows(s.Systems, func(sys client.SystemDTO) string {
			return fmt.Sprintf("%s|%s|%d|%s|%s|%s|%s|%s|%s|%s|%s|%.1f|%s",
				sys.Name, sys.Protocol,
				sys.TETRAColourCode, sys.TETRAChannel,
				sys.TETRAChannelCoding,
				sys.LTRFCSMode, sys.LTRManchesterMode,
				sys.P25Phase2TrellisMode, sys.P25Phase2RSMode,
				sys.P25Phase2ScramblerMode,
				sys.NXDNViterbiMode,
				sys.NXDNDeviationHz,
				sys.EDACSBCHMode)
		})
		if h != p.lastHash {
			p.refresh(s.Systems)
			p.lastHash = h
		}
		var cmd tea.Cmd
		p.tbl, cmd = p.tbl.Update(msg)
		return p, cmd
	}
	return p, nil
}

func (p *SettingsPanel) refresh(sys []client.SystemDTO) {
	rows := make([]table.Row, 0, len(sys))
	for _, s := range sys {
		rows = append(rows, table.Row{
			s.Name,
			s.Protocol,
			fecSummary(s),
		})
	}
	p.tbl.SetRows(rows)
}

func (p *SettingsPanel) View(width, height int, focused bool, s *state.SharedState) string {
	header := p.renderTabBar(width)
	banner := p.renderEditBanner(s)
	var body string
	if p.tab == tabFEC {
		p.tbl.SetColumns(settingsColumns(width))
		p.tbl.SetWidth(width)
		if height > 6 {
			p.tbl.SetHeight(height - 6)
		}
		body = p.tbl.View() + "\n\n" + dashDim.Render("Edit config.yaml + restart daemon to change; see the FEC opt-outs section in README.md.")
	} else {
		// Editable rows first (only when the daemon has a config
		// file backing it — empty ConfigPath means PATCH returns
		// 503 and we shouldn't lead the operator on).
		fields := editableFieldsForTab(p.tab)
		var head string
		if len(fields) > 0 && s.Runtime.ConfigPath != "" {
			head = p.renderEditableRows(fields, s.Runtime)
		}
		body = head + p.renderTab(width, s)
	}
	return panelFrame("Settings", width, height, focused, header+"\n"+banner+body)
}

// renderEditBanner surfaces the daemon's live-edit capability: when
// the daemon ran with -config, settings can be edited via
// PATCH /api/v1/settings (curl / web SPA); without a config file the
// endpoint returns 503 and the panel stays purely read-only.
func (p *SettingsPanel) renderEditBanner(s *state.SharedState) string {
	if s == nil {
		return ""
	}
	if s.Runtime.ConfigPath == "" {
		return dashDim.Render(
			"  daemon running without -config — Settings are read-only (PATCH /api/v1/settings returns 503)") + "\n\n"
	}
	return dashDim.Render(
		"  live edits supported — PATCH /api/v1/settings (config @ "+s.Runtime.ConfigPath+")") + "\n\n"
}

func (p *SettingsPanel) renderTabBar(width int) string {
	parts := make([]string, 0, tabCount)
	for i := settingsTab(0); i < tabCount; i++ {
		label := fmt.Sprintf(" %s ", i.String())
		if i == p.tab {
			parts = append(parts, dashAccent.Render(">"+label+"<"))
		} else {
			parts = append(parts, dashDim.Render(label))
		}
	}
	bar := strings.Join(parts, " ")
	hint := dashDim.Render("  [/]  to switch  •  config.yaml owns these values")
	return bar + hint
}

func (p *SettingsPanel) renderTab(width int, s *state.SharedState) string {
	r := s.Runtime
	switch p.tab {
	case tabDaemon:
		return rows([][2]string{
			{"Version", or(r.Version, "—")},
			{"Log level", or(r.LogLevel, "info")},
			{"Log format", or(r.LogFormat, "text")},
			{"Metrics enabled", boolText(r.MetricsEnabled)},
		})
	case tabStorage:
		return rows([][2]string{
			{"Call log path", or(r.StorageDBPath, "—  (call log disabled)")},
			{"CC cache file", or(r.StorageCCCache, "—  (cache disabled)")},
			{"Retention (call log)", fmt.Sprintf("%d days", r.RetentionCallLogDays)},
			{"Retention (files)", fmt.Sprintf("%d days", r.RetentionFilesDays)},
			{"Retention interval", durText(r.RetentionInterval)},
		})
	case tabAudio:
		out := rows([][2]string{
			{"Enabled", boolText(r.AudioEnabled)},
			{"Device", or(r.AudioDevice, "default (system sink)")},
			{"Sample rate", fmt.Sprintf("%d Hz", r.AudioSampleRate)},
			{"Buffer", fmt.Sprintf("%d ms", r.AudioBufferMs)},
			{"Disable auto-fallback (Linux)", boolText(r.AudioDisableFallbk)},
		})
		if len(r.AudioBackends) > 0 {
			out += "\n\n" + dashHeader.Render("Available outputs") + "\n  " + strings.Join(r.AudioBackends, "\n  ")
		}
		return out
	case tabRecording:
		return rows([][2]string{
			{"Output directory", or(r.RecordingDir, "—  (recording disabled)")},
			{"Sample rate", fmt.Sprintf("%d Hz", r.RecordingSampleRate)},
			{"Write raw vocoder frames", boolText(r.RecordingWriteRaw)},
			{"CMA equalizer", boolText(r.RecordingEQEnabled)},
			{"  EQ taps", intText(r.RecordingEQTaps)},
			{"  EQ step size", or(r.RecordingEQStepSize, "—")},
		})
	case tabTones:
		if len(r.ToneProfiles) == 0 {
			return dashDim.Render("  no tone-out profiles configured")
		}
		var b strings.Builder
		for _, t := range r.ToneProfiles {
			fmt.Fprintf(&b, "  %s\n", dashAccent.Render(t.Name))
			if t.AlphaTag != "" {
				fmt.Fprintf(&b, "    alpha    %s\n", t.AlphaTag)
			}
			fmt.Fprintf(&b, "    tones    %d\n", t.ToneCount)
			fmt.Fprintf(&b, "    cooldown %s\n", durText(t.Cooldown))
		}
		return b.String()
	case tabAPI:
		return rows([][2]string{
			{"HTTP REST + SSE + WS", or(r.HTTPAddr, "—  (HTTP API disabled)")},
			{"  SSE path", "/api/v1/events"},
			{"  WS path", "/api/v1/events/ws"},
			{"  Metrics path", "/metrics  (if metrics.enabled)"},
			{"gRPC", or(r.GRPCAddr, "—  (gRPC disabled)")},
			{"Allow mutations", boolText(r.AllowMutations)},
		})
	case tabVocoders:
		if len(r.VocoderMap) == 0 {
			return dashDim.Render("  vocoder map unavailable")
		}
		// Render the map sorted for deterministic output.
		keys := make([]string, 0, len(r.VocoderMap))
		for k := range r.VocoderMap {
			keys = append(keys, k)
		}
		// inline insertion sort over short slices
		for i := 1; i < len(keys); i++ {
			for j := i; j > 0 && keys[j-1] > keys[j]; j-- {
				keys[j-1], keys[j] = keys[j], keys[j-1]
			}
		}
		out := make([][2]string, 0, len(keys))
		for _, k := range keys {
			out = append(out, [2]string{k, r.VocoderMap[k]})
		}
		return rows(out)
	case tabSDR:
		out := rows([][2]string{
			{"Sample rate", fmt.Sprintf("%d Hz", r.SDRSampleRate)},
		})
		if len(r.SDRBackends) > 0 {
			out += "\n\n" + dashHeader.Render("Linked backends") + "\n  " + strings.Join(r.SDRBackends, ", ")
		}
		return out
	}
	return ""
}

func settingsColumns(w int) []table.Column {
	if w < 50 {
		w = 50
	}
	nameW := w * 22 / 100
	protoW := 11
	fecW := w - nameW - protoW - 4
	if fecW < 20 {
		fecW = 20
	}
	return []table.Column{
		{Title: "Name", Width: nameW},
		{Title: "Protocol", Width: protoW},
		{Title: "FEC", Width: fecW},
	}
}

// rows renders a two-column key/value list. Keys right-padded to the
// widest key in the slice so the values align without lipgloss table
// machinery — keeps the inspector lightweight.
func rows(pairs [][2]string) string {
	maxK := 0
	for _, p := range pairs {
		if l := len([]rune(p[0])); l > maxK {
			maxK = l
		}
	}
	var b strings.Builder
	for _, p := range pairs {
		k := p[0] + strings.Repeat(" ", maxK-len([]rune(p[0])))
		fmt.Fprintf(&b, "  %s   %s\n", dashDim.Render(k), p[1])
	}
	return b.String()
}

func or(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func boolText(b bool) string {
	if b {
		return dashOK.Render("on")
	}
	return dashDim.Render("off")
}

func intText(n int) string {
	if n == 0 {
		return "—"
	}
	return fmt.Sprintf("%d", n)
}

func durText(d interface{ Nanoseconds() int64 }) string {
	if d.Nanoseconds() == 0 {
		return "—"
	}
	return fmt.Sprintf("%v", d)
}

// fecSummary returns a one-line, protocol-scoped summary of the
// system's FEC state. Only the keys relevant to the system's
// protocol are emitted so the column doesn't drown in N/A noise.
//
// FEC is on by default for every protocol; the connector applies
// the spec-correct chain unless the operator opts out per-protocol
// via the `*_mode: off` keys. Empty config strings render as the
// new on-defaults below; explicit "off" renders as "off".
func fecSummary(s client.SystemDTO) string {
	var parts []string
	switch strings.ToLower(s.Protocol) {
	case "tetra":
		coding := orDefault(s.TETRAChannelCoding, "on")
		if isOff(coding) {
			parts = append(parts, "channel coding: off (opt-out)")
		} else {
			ch := s.TETRAChannel
			if ch == "" {
				ch = "sch/hd"
			}
			parts = append(parts, fmt.Sprintf("channel coding: on (colour=%#x, %s)", s.TETRAColourCode, ch))
		}
	case "ltr":
		parts = append(parts, "fcs: "+orDefault(s.LTRFCSMode, "on"))
		parts = append(parts, "manchester: "+orDefault(s.LTRManchesterMode, "soft"))
	case "p25-phase2":
		parts = append(parts, "trellis: "+orDefault(s.P25Phase2TrellisMode, "on"))
		parts = append(parts, "rs: "+orDefault(s.P25Phase2RSMode, "off"))
		parts = append(parts, "scrambler: "+orDefault(s.P25Phase2ScramblerMode, "off"))
	case "nxdn":
		parts = append(parts, "viterbi: "+orDefault(s.NXDNViterbiMode, "spec"))
		if s.NXDNDeviationHz > 0 {
			parts = append(parts, fmt.Sprintf("deviation: %.0f Hz", s.NXDNDeviationHz))
		} else {
			parts = append(parts, "deviation: 1800 Hz")
		}
	case "edacs":
		parts = append(parts, "bch: "+orDefault(s.EDACSBCHMode, "on"))
	case "mpt1327":
		parts = append(parts, "bch: "+orDefault(s.MPT1327BCHMode, "on"))
		parts = append(parts, "cwsc: "+orDefault(s.MPT1327CWSCTolerance, "2"))
	case "motorola":
		parts = append(parts, "bch: "+orDefault(s.MotorolaBCHMode, "on"))
	default:
		parts = append(parts, "—")
	}
	return strings.Join(parts, "  ·  ")
}

// orDefault returns s when non-empty, otherwise the connector-level
// default for that key.
func orDefault(s, dflt string) string {
	if s == "" {
		return dflt
	}
	return s
}

// isOff reports whether the supplied mode string parses to an
// explicit opt-out across the per-protocol Parse* functions.
func isOff(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "off", "false", "0":
		return true
	}
	return false
}
