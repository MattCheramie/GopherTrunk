package panels

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

func TestSystemsPanel_RevealPositionsCursor(t *testing.T) {
	p := NewSystems()
	s := &state.SharedState{Systems: []client.SystemDTO{
		{Name: "Alpha", Protocol: "p25"},
		{Name: "Bravo", Protocol: "dmr"},
		{Name: "Charlie", Protocol: "nxdn"},
	}}
	_, _ = p.Update(tea.WindowSizeMsg{Width: 120, Height: 30}, s)

	p.Reveal("Bravo")
	if got := p.tbl.Cursor(); got != 1 {
		t.Errorf("cursor = %d, want 1 after Reveal(Bravo)", got)
	}
	// Unknown key is a no-op.
	p.Reveal("Zulu")
	if got := p.tbl.Cursor(); got != 1 {
		t.Errorf("cursor = %d, want 1 after Reveal(Zulu)", got)
	}
}

func TestTalkgroupsPanel_RevealByID(t *testing.T) {
	p := NewTalkgroups()
	s := &state.SharedState{Talkgroups: []client.TalkgroupDTO{
		{ID: 1001, AlphaTag: "DISPATCH"},
		{ID: 1002, AlphaTag: "TACTICAL"},
		{ID: 1003, AlphaTag: "FIRE"},
	}}
	_, _ = p.Update(tea.WindowSizeMsg{Width: 120, Height: 30}, s)

	p.Reveal("1003")
	if got := p.tbl.Cursor(); got != 2 {
		t.Errorf("cursor = %d, want 2 after Reveal(1003)", got)
	}
}

func TestDevicesPanel_RevealBySerial(t *testing.T) {
	p := NewDevices()
	s := &state.SharedState{Devices: []client.SDRStatus{
		{Serial: "AAA"},
		{Serial: "BBB"},
	}}
	_, _ = p.Update(tea.WindowSizeMsg{Width: 120, Height: 30}, s)
	p.Reveal("BBB")
	if got := p.tbl.Cursor(); got != 1 {
		t.Errorf("cursor = %d, want 1 after Reveal(BBB)", got)
	}
}

func TestScannerPanel_RevealResolvedOnNextUpdate(t *testing.T) {
	p := NewScanner()
	s := &state.SharedState{Scanner: client.ScannerStatusDTO{
		Systems: []client.SystemHuntStatusDTO{
			{Name: "Alpha"},
			{Name: "Bravo"},
		},
		Conventional: client.ConvScannerStatusDTO{Enabled: true,
			Channels: []client.ConvChannelStatusDTO{
				{Index: 0, Label: "PD"},
				{Index: 1, Label: "FD"},
			}},
	}}
	// Reveal stashes — cursor hasn't moved yet.
	p.Reveal("sys:Bravo")
	if p.cursor != 0 {
		t.Errorf("cursor moved before Update: %d", p.cursor)
	}
	// Update resolves the pending reveal against the snapshot.
	_, _ = p.Update(struct{}{}, s)
	if p.cursor != 1 {
		t.Errorf("cursor = %d, want 1 after sys:Bravo resolved", p.cursor)
	}
	// Conventional reveal lands past the systems block.
	p.Reveal("conv:1")
	_, _ = p.Update(struct{}{}, s)
	if p.cursor != 3 {
		t.Errorf("cursor = %d, want 3 (nSys=2 + conv 1)", p.cursor)
	}
	// Malformed key clears without panicking.
	p.Reveal("conv:bogus")
	_, _ = p.Update(struct{}{}, s)
	if p.cursor != 3 {
		t.Errorf("cursor = %d, want 3 (malformed reveal should not move cursor)", p.cursor)
	}
}

func TestSystemsPanel_HandleMouseAtMovesCursor(t *testing.T) {
	p := NewSystems()
	s := &state.SharedState{Systems: []client.SystemDTO{
		{Name: "A"}, {Name: "B"}, {Name: "C"}, {Name: "D"},
	}}
	_, _ = p.Update(tea.WindowSizeMsg{Width: 120, Height: 30}, s)

	// localY 0/1/2 are chrome; first data row is local Y == 3.
	if cmd := p.HandleMouseAt(10, 2); cmd != nil {
		t.Errorf("expected nil cmd on chrome click")
	}
	if got := p.tbl.Cursor(); got != 0 {
		t.Errorf("chrome click moved cursor to %d", got)
	}
	_ = p.HandleMouseAt(10, 4) // row 1
	if got := p.tbl.Cursor(); got != 1 {
		t.Errorf("cursor = %d after click on row 1, want 1", got)
	}
	// Past-end clamps to the last row.
	_ = p.HandleMouseAt(10, 50)
	if got := p.tbl.Cursor(); got != 3 {
		t.Errorf("cursor = %d after out-of-range click, want 3", got)
	}
}

func TestTableRowFromLocalY(t *testing.T) {
	cases := []struct {
		y, rows, want int
	}{
		{0, 5, -1},  // top border
		{1, 5, -1},  // title
		{2, 5, -1},  // column header
		{3, 5, 0},   // first row
		{7, 5, 4},   // last row
		{99, 5, 4},  // past end → clamp
		{3, 0, -1},  // empty table
		{-1, 5, -1}, // negative
	}
	for _, c := range cases {
		got := tableRowFromLocalY(c.y, c.rows)
		if got != c.want {
			t.Errorf("tableRowFromLocalY(%d, %d) = %d, want %d", c.y, c.rows, got, c.want)
		}
	}
}
