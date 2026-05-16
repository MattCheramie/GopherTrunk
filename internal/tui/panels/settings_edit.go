package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

// editableField captures everything the Settings panel needs to know
// to render one editable row + dispatch a SettingsReq when the
// operator saves.
type editableField struct {
	// Label is what the operator sees on the left side of the row.
	Label string
	// Field is the dotted YAML path (e.g. "audio.volume"). Used
	// as SettingsReq.Field; the root dispatcher converts to the
	// matching SettingsPatch shape.
	Field string
	// Value extracts the current value from the runtime DTO as a
	// human-readable string.
	Value func(r client.RuntimeDTO) string
	// Restart marks fields that aren't hot-reloadable. The renderer
	// adds a small "restart" badge so operators know the change
	// won't take effect until they bounce the daemon.
	Restart bool
}

// editableFieldsForTab returns the editable rows for the given tab.
// Returning a slice instead of a map keeps row order stable across
// renders.
func editableFieldsForTab(t settingsTab) []editableField {
	switch t {
	case tabDaemon:
		return []editableField{
			{Label: "Log level", Field: "log.level",
				Value: func(r client.RuntimeDTO) string { return or(r.LogLevel, "info") }},
			{Label: "Log format", Field: "log.format", Restart: true,
				Value: func(r client.RuntimeDTO) string { return or(r.LogFormat, "text") }},
			{Label: "Metrics enabled", Field: "metrics.enabled", Restart: true,
				Value: func(r client.RuntimeDTO) string { return boolStr(r.MetricsEnabled) }},
		}
	case tabStorage:
		return []editableField{
			{Label: "Call log path", Field: "storage.path", Restart: true,
				Value: func(r client.RuntimeDTO) string { return or(r.StorageDBPath, "") }},
			{Label: "CC cache file", Field: "storage.cc_cache_file", Restart: true,
				Value: func(r client.RuntimeDTO) string { return or(r.StorageCCCache, "") }},
			{Label: "Retention (call log days)", Field: "retention.call_log_days", Restart: true,
				Value: func(r client.RuntimeDTO) string { return fmt.Sprintf("%d", r.RetentionCallLogDays) }},
			{Label: "Retention (files days)", Field: "retention.files_days", Restart: true,
				Value: func(r client.RuntimeDTO) string { return fmt.Sprintf("%d", r.RetentionFilesDays) }},
			{Label: "Retention interval", Field: "retention.interval", Restart: true,
				Value: func(r client.RuntimeDTO) string {
					if r.RetentionInterval == 0 {
						return ""
					}
					return r.RetentionInterval.String()
				}},
		}
	case tabAudio:
		return []editableField{
			{Label: "Enabled", Field: "audio.enabled", Restart: true,
				Value: func(r client.RuntimeDTO) string { return boolStr(r.AudioEnabled) }},
			{Label: "Device", Field: "audio.device", Restart: true,
				Value: func(r client.RuntimeDTO) string { return r.AudioDevice }},
			{Label: "Buffer (ms)", Field: "audio.buffer_ms", Restart: true,
				Value: func(r client.RuntimeDTO) string { return fmt.Sprintf("%d", r.AudioBufferMs) }},
		}
	case tabRecording:
		return []editableField{
			{Label: "Output directory", Field: "recordings.dir", Restart: true,
				Value: func(r client.RuntimeDTO) string { return r.RecordingDir }},
			{Label: "Sample rate (Hz)", Field: "recordings.sample_rate", Restart: true,
				Value: func(r client.RuntimeDTO) string { return fmt.Sprintf("%d", r.RecordingSampleRate) }},
			{Label: "Write raw vocoder frames", Field: "recordings.write_raw",
				Value: func(r client.RuntimeDTO) string { return boolStr(r.RecordingWriteRaw) }},
		}
	case tabAPI:
		return []editableField{
			{Label: "HTTP addr", Field: "api.http_addr", Restart: true,
				Value: func(r client.RuntimeDTO) string { return r.HTTPAddr }},
			{Label: "gRPC addr", Field: "api.grpc_addr", Restart: true,
				Value: func(r client.RuntimeDTO) string { return r.GRPCAddr }},
		}
	case tabSDR:
		return []editableField{
			{Label: "Sample rate (Hz)", Field: "sdr.sample_rate", Restart: true,
				Value: func(r client.RuntimeDTO) string { return fmt.Sprintf("%d", r.SDRSampleRate) }},
		}
	}
	return nil
}

// startEdit promotes the focused row to an editable textinput,
// seeded with the row's current value so the operator can tweak
// rather than retype.
func (p *SettingsPanel) startEdit(fields []editableField, r client.RuntimeDTO) {
	if p.editCursor < 0 || p.editCursor >= len(fields) {
		return
	}
	f := fields[p.editCursor]
	ti := textinput.New()
	ti.SetValue(f.Value(r))
	ti.Prompt = ""
	ti.Width = 40
	ti.CharLimit = 256
	ti.Focus()
	p.editInput = ti
	p.editing = true
	p.editErr = ""
}

// commitEdit emits a WriteActionMsg with the operator's input and
// resets the editing state so the next Enter starts fresh.
func (p *SettingsPanel) commitEdit(fields []editableField) tea.Cmd {
	if !p.editing || p.editCursor < 0 || p.editCursor >= len(fields) {
		return nil
	}
	val := strings.TrimSpace(p.editInput.Value())
	field := fields[p.editCursor].Field
	p.editing = false
	p.editErr = ""
	return func() tea.Msg {
		return WriteActionMsg{Request: state.WriteRequest{
			Kind:  state.WriteKindSettings,
			Label: "settings " + field,
			Settings: &state.SettingsReq{
				Field: field,
				Value: val,
			},
		}}
	}
}

// renderEditableRows produces the row-by-row view for tabs that
// support editing. Each row gets a cursor marker, label, current
// value (or live textinput when this row is being edited), and a
// 'restart' badge when applicable.
func (p *SettingsPanel) renderEditableRows(fields []editableField, r client.RuntimeDTO) string {
	if len(fields) == 0 {
		return ""
	}
	maxLabel := 0
	for _, f := range fields {
		if l := len([]rune(f.Label)); l > maxLabel {
			maxLabel = l
		}
	}
	var b strings.Builder
	for i, f := range fields {
		cursor := "  "
		if i == p.editCursor {
			cursor = "› "
		}
		label := f.Label + strings.Repeat(" ", maxLabel-len([]rune(f.Label)))
		var value string
		if p.editing && i == p.editCursor {
			value = p.editInput.View()
		} else {
			val := f.Value(r)
			if val == "" {
				val = dashDim.Render("(unset)")
			}
			value = val
		}
		row := cursor + dashDim.Render(label) + "   " + value
		if f.Restart {
			row += "   " + lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("[restart]")
		}
		fmt.Fprintln(&b, row)
	}
	if p.editErr != "" {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("  ! "+p.editErr))
	}
	fmt.Fprintln(&b)
	if p.editing {
		fmt.Fprintln(&b, dashDim.Render("  enter save  •  esc cancel"))
	} else {
		fmt.Fprintln(&b, dashDim.Render("  j/k or ↑/↓ to move  •  enter to edit"))
	}
	return b.String()
}

// boolStr renders a bool as "true" / "false" for an editable field
// (so the wire round-trip with the daemon stays unambiguous).
func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// editableSettingsKeys is the set of bindings the help overlay
// surfaces for the editable rows.
var (
	editUpKey   = key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("k/↑", "row up"))
	editDownKey = key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("j/↓", "row down"))
	editStart   = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "edit / save"))
	editCancel  = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel"))
)
