package panels

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

func TestImportPanelAddPath(t *testing.T) {
	dir := t.TempDir()
	csv := filepath.Join(dir, "x.csv")
	if err := os.WriteFile(csv, []byte("# csv"), 0o644); err != nil {
		t.Fatal(err)
	}

	p := NewImport()
	shared := &state.SharedState{Runtime: client.RuntimeDTO{ConfigPath: "/etc/cfg.yaml"}}

	// Type the path one rune at a time so the textinput's parser
	// sees real KeyMsg values.
	for _, r := range csv {
		_, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}, shared)
	}
	// Press enter to add.
	_, _ = p.Update(tea.KeyMsg{Type: tea.KeyEnter}, shared)
	if len(p.paths) != 1 {
		t.Fatalf("paths=%v want 1 entry", p.paths)
	}
	if p.paths[0] != csv {
		t.Errorf("paths[0]=%q want %q", p.paths[0], csv)
	}

	// Press Ctrl-X to clear.
	_, _ = p.Update(tea.KeyMsg{Type: tea.KeyCtrlX}, shared)
	if len(p.paths) != 0 {
		t.Errorf("after clear, paths=%v want empty", p.paths)
	}
}

func TestImportPanelUploadEmits(t *testing.T) {
	p := NewImport()
	p.paths = []string{"/tmp/a.csv", "/tmp/b.pdf"}

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyCtrlU}, nil)
	if cmd == nil {
		t.Fatal("expected upload cmd, got nil")
	}
	msg := cmd()
	upload, ok := msg.(ImportUploadMsg)
	if !ok {
		t.Fatalf("got %T, want ImportUploadMsg", msg)
	}
	if len(upload.Paths) != 2 {
		t.Errorf("paths=%v want 2 entries", upload.Paths)
	}
}

func TestImportPanelPreviewToCommit(t *testing.T) {
	p := NewImport()
	// Inject a preview result.
	updated, _ := p.Update(ImportPreviewArrivedMsg{Preview: client.ImportPreview{
		ID: "abc123",
		Systems: []client.ParsedSystemDTO{
			{Name: "Sys1", Protocol: "p25", SiteCount: 1, TalkgroupCt: 2},
		},
	}}, nil)
	p = updated.(*ImportPanel)
	if p.view != importPreviewView {
		t.Fatalf("view=%v want importPreviewView", p.view)
	}

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}, nil)
	if cmd == nil {
		t.Fatal("expected commit cmd")
	}
	commit, ok := cmd().(ImportCommitMsg)
	if !ok {
		t.Fatalf("got %T, want ImportCommitMsg", cmd())
	}
	if commit.ID != "abc123" {
		t.Errorf("commit ID=%q want abc123", commit.ID)
	}
}

func TestImportPanelResultViewRendersAddedSystems(t *testing.T) {
	p := NewImport()
	updated, _ := p.Update(ImportResultArrivedMsg{Result: client.ImportResult{
		SystemsAdded: []string{"Sys1", "Sys2"},
		ConfigPath:   "/etc/cfg.yaml",
	}}, nil)
	p = updated.(*ImportPanel)
	if p.view != importResultView {
		t.Fatalf("view=%v want importResultView", p.view)
	}
	out := p.View(80, 30, true, &state.SharedState{})
	if !strings.Contains(out, "Sys1") || !strings.Contains(out, "Sys2") {
		t.Errorf("output missing added systems: %q", out)
	}
}
