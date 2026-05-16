package panels

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

// ImportPanel is the live-import cockpit. Three views:
//
//  1. Stage — operator adds local file paths via a textinput; pressing
//     'u' uploads them to the daemon and we transition to Preview.
//  2. Preview — table of parsed systems returned by the daemon;
//     'c' commits, 'd' discards, esc returns to Stage.
//  3. Result — list of systems_added / systems_replaced; esc returns
//     to Stage.
type ImportPanel struct {
	view        importView
	input       textinput.Model
	paths       []string
	inputErr    string
	previewID   string
	preview     []client.ParsedSystemDTO
	result      *client.ImportResult
	statusMsg   string
}

type importView int

const (
	importStageView importView = iota
	importPreviewView
	importResultView
)

// ImportUploadMsg is emitted when the operator has confirmed an
// upload via 'u'. The root model resolves it into an HTTP call.
type ImportUploadMsg struct {
	Paths []string
}

// ImportCommitMsg is emitted when the operator presses 'c' on the
// Preview view.
type ImportCommitMsg struct {
	ID    string
	Force bool
}

// ImportDiscardMsg is emitted when the operator presses 'd' on the
// Preview view.
type ImportDiscardMsg struct {
	ID string
}

// ImportPreviewArrivedMsg is consumed by the panel when the root
// model has received the staging response.
type ImportPreviewArrivedMsg struct {
	Preview client.ImportPreview
	Err     error
}

// ImportResultArrivedMsg is consumed by the panel when a commit
// finishes.
type ImportResultArrivedMsg struct {
	Result client.ImportResult
	Err    error
}

// NewImport returns a fresh import panel.
func NewImport() *ImportPanel {
	in := textinput.New()
	in.Placeholder = "/path/to/system.csv or /path/to/system.pdf"
	in.CharLimit = 512
	in.Width = 60
	in.Prompt = "path> "
	in.Focus()
	return &ImportPanel{input: in}
}

func (*ImportPanel) Title() string { return "Import" }

var (
	// "Add a path" rides Enter so it never fights the textinput. The
	// upload / clear shortcuts use control sequences for the same
	// reason — the textinput swallows plain-letter keys.
	importAddKey     = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "add path"))
	importUploadKey  = key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("⌃U", "upload"))
	importClearKey   = key.NewBinding(key.WithKeys("ctrl+x"), key.WithHelp("⌃X", "clear queue"))
	importCommitKey  = key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "commit"))
	importDiscardKey = key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "discard preview"))
	importEscKey     = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back"))
)

func (*ImportPanel) Keys() []key.Binding {
	return []key.Binding{
		importAddKey, importUploadKey, importClearKey,
		importCommitKey, importDiscardKey, importEscKey,
	}
}

func (p *ImportPanel) Update(msg tea.Msg, shared *state.SharedState) (Panel, tea.Cmd) {
	switch m := msg.(type) {
	case ImportPreviewArrivedMsg:
		if m.Err != nil {
			p.inputErr = m.Err.Error()
			return p, nil
		}
		p.previewID = m.Preview.ID
		p.preview = m.Preview.Systems
		p.view = importPreviewView
		p.statusMsg = ""
		return p, nil
	case ImportResultArrivedMsg:
		if m.Err != nil {
			p.inputErr = m.Err.Error()
			return p, nil
		}
		p.result = &m.Result
		p.preview = nil
		p.previewID = ""
		p.paths = nil
		p.view = importResultView
		return p, nil
	}

	switch p.view {
	case importStageView:
		return p.updateStage(msg, shared)
	case importPreviewView:
		return p.updatePreview(msg)
	case importResultView:
		return p.updateResult(msg)
	}
	return p, nil
}

func (p *ImportPanel) updateStage(msg tea.Msg, shared *state.SharedState) (Panel, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(km, importAddKey):
			path := strings.TrimSpace(p.input.Value())
			if path == "" {
				p.inputErr = "empty path"
				return p, nil
			}
			if _, err := os.Stat(path); err != nil {
				p.inputErr = err.Error()
				return p, nil
			}
			p.paths = append(p.paths, path)
			p.input.SetValue("")
			p.inputErr = ""
			return p, nil
		case key.Matches(km, importUploadKey):
			if len(p.paths) == 0 {
				p.inputErr = "no paths queued"
				return p, nil
			}
			p.inputErr = ""
			p.statusMsg = "uploading…"
			return p, func() tea.Msg { return ImportUploadMsg{Paths: append([]string(nil), p.paths...)} }
		case key.Matches(km, importClearKey):
			p.paths = nil
			p.inputErr = ""
			return p, nil
		}
	}
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	return p, cmd
}

func (p *ImportPanel) updatePreview(msg tea.Msg) (Panel, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(km, importCommitKey):
			id := p.previewID
			p.statusMsg = "committing…"
			return p, func() tea.Msg { return ImportCommitMsg{ID: id} }
		case key.Matches(km, importDiscardKey):
			id := p.previewID
			p.preview = nil
			p.previewID = ""
			p.view = importStageView
			p.statusMsg = "discarded"
			return p, func() tea.Msg { return ImportDiscardMsg{ID: id} }
		case key.Matches(km, importEscKey):
			p.view = importStageView
			return p, nil
		}
	}
	return p, nil
}

func (p *ImportPanel) updateResult(msg tea.Msg) (Panel, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(km, importEscKey), key.Matches(km, importAddKey),
			km.String() == "n":
			p.view = importStageView
			p.result = nil
			return p, nil
		}
	}
	return p, nil
}

func (p *ImportPanel) View(width, height int, focused bool, shared *state.SharedState) string {
	switch p.view {
	case importPreviewView:
		return panelFrame("Import", width, height, focused, p.renderPreview(width))
	case importResultView:
		return panelFrame("Import", width, height, focused, p.renderResult(width))
	}
	return panelFrame("Import", width, height, focused, p.renderStage(width, shared))
}

func (p *ImportPanel) renderStage(width int, shared *state.SharedState) string {
	var b strings.Builder

	if shared != nil && shared.Runtime.ConfigPath == "" {
		fmt.Fprintln(&b, dashDim.Render(
			"daemon running without a -config file — import is disabled (server returns 503)"))
		fmt.Fprintln(&b)
	}

	fmt.Fprintln(&b, dashHeader.Render("Stage files to upload"))
	fmt.Fprintln(&b, dashDim.Render("  Type a path → 'a' to add. Repeat for more. Then 'u' to upload."))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "  "+p.input.View())
	if p.inputErr != "" {
		fmt.Fprintln(&b, "  "+lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("! "+p.inputErr))
	}
	fmt.Fprintln(&b)

	fmt.Fprintln(&b, dashHeader.Render(fmt.Sprintf("Queued (%d)", len(p.paths))))
	if len(p.paths) == 0 {
		fmt.Fprintln(&b, dashDim.Render("  (empty)"))
	} else {
		for _, q := range p.paths {
			fmt.Fprintln(&b, "  "+filepath.Base(q)+dashDim.Render("   "+filepath.Dir(q)))
		}
	}
	fmt.Fprintln(&b)
	if p.statusMsg != "" {
		fmt.Fprintln(&b, dashDim.Render("  "+p.statusMsg))
	}
	fmt.Fprintln(&b, dashDim.Render("  Keys: enter add  •  ⌃U upload  •  ⌃X clear queue"))
	return b.String()
}

func (p *ImportPanel) renderPreview(width int) string {
	var b strings.Builder
	fmt.Fprintln(&b, dashHeader.Render("Parsed systems (preview)"))
	if len(p.preview) == 0 {
		fmt.Fprintln(&b, dashDim.Render("  (no systems parsed)"))
		return b.String()
	}
	for _, s := range p.preview {
		fmt.Fprintf(&b, "  %s\n", dashAccent.Render(s.Name))
		fmt.Fprintf(&b, "    protocol  %s\n", s.Protocol)
		fmt.Fprintf(&b, "    sites     %d\n", s.SiteCount)
		fmt.Fprintf(&b, "    tgs       %d\n", s.TalkgroupCt)
		if s.Location != "" {
			fmt.Fprintf(&b, "    location  %s\n", s.Location)
		}
	}
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, dashDim.Render("  staging id "+p.previewID))
	if p.statusMsg != "" {
		fmt.Fprintln(&b, dashDim.Render("  "+p.statusMsg))
	}
	fmt.Fprintln(&b, dashDim.Render("  Keys: c commit  •  d discard  •  esc back"))
	return b.String()
}

func (p *ImportPanel) renderResult(width int) string {
	var b strings.Builder
	fmt.Fprintln(&b, dashHeader.Render("Import committed"))
	if p.result == nil {
		fmt.Fprintln(&b, dashDim.Render("  (no result)"))
		return b.String()
	}
	if len(p.result.SystemsAdded) > 0 {
		fmt.Fprintln(&b, "  "+dashAccent.Render("Added: ")+strings.Join(p.result.SystemsAdded, ", "))
	}
	if len(p.result.SystemsReplaced) > 0 {
		fmt.Fprintln(&b, "  "+dashAccent.Render("Replaced: ")+strings.Join(p.result.SystemsReplaced, ", "))
	}
	if len(p.result.CSVPaths) > 0 {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, dashHeader.Render("Talkgroup CSVs"))
		for _, p := range p.result.CSVPaths {
			fmt.Fprintln(&b, "  "+p)
		}
	}
	if p.result.ConfigPath != "" {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, dashDim.Render("  config.yaml @ "+p.result.ConfigPath))
	}
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, dashDim.Render("  esc/a back to stage"))
	return b.String()
}
