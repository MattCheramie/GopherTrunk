package panels

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

func TestTalkgroupsPanel_Filter(t *testing.T) {
	p := NewTalkgroups()
	s := &state.SharedState{
		Talkgroups: []client.TalkgroupDTO{
			{ID: 1, AlphaTag: "Dispatch"},
			{ID: 2, AlphaTag: "Tac1"},
			{ID: 3, AlphaTag: "Fire"},
		},
	}
	// First Update populates the table (3 rows).
	_, _ = p.Update(tea.WindowSizeMsg{Width: 120, Height: 30}, s)
	if p.RowCount() != 3 {
		t.Fatalf("initial rows = %d, want 3", p.RowCount())
	}

	// Apply a filter: should narrow to 1 row.
	p.SetFilterValue("disp")
	_, _ = p.Update(tea.WindowSizeMsg{Width: 120, Height: 30}, s)
	if p.RowCount() != 1 {
		t.Errorf("filtered rows = %d, want 1 (matching 'Dispatch')", p.RowCount())
	}

	// Empty filter restores all rows.
	p.SetFilterValue("")
	_, _ = p.Update(tea.WindowSizeMsg{Width: 120, Height: 30}, s)
	if p.RowCount() != 3 {
		t.Errorf("after clearing filter rows = %d, want 3", p.RowCount())
	}
}
