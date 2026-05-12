package panels

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

// historyMsg drives Update with a non-key, non-refresh message so we
// exercise the hash-diff path without triggering reload-on-keypress.
type historyMsg struct{}

func TestHistoryPanel_AsyncRefresh_HappyPath(t *testing.T) {
	p := NewHistory()
	s := &state.SharedState{History: []client.CallRow{
		{ID: 1, GroupID: 100, System: "Alpha", TalkgroupAlpha: "DISPATCH",
			StartedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		{ID: 2, GroupID: 200, System: "Bravo",
			StartedAt: time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC)},
	}}

	// First Update on a fresh panel: hash mismatch → spawn a refresh
	// Cmd. Table is still empty (commit only happens when the msg
	// returns).
	_, cmd := p.Update(historyMsg{}, s)
	if cmd == nil {
		t.Fatal("expected a refresh Cmd on first Update")
	}
	if got := len(p.tbl.Rows()); got != 0 {
		t.Errorf("table rebuilt synchronously: %d rows", got)
	}
	if p.pendingAt == 0 {
		t.Errorf("pendingAt not stamped")
	}

	// Run the Cmd to completion (tests run synchronously; the Cmd is
	// just a function returning a tea.Msg).
	msg := cmd()
	got, ok := msg.(HistoryRefreshedMsg)
	if !ok {
		// Cmd may have been batched with the table's own Update cmd.
		// Drain a tea.BatchMsg.
		if bm, isBatch := msg.(tea.BatchMsg); isBatch {
			for _, c := range bm {
				if c == nil {
					continue
				}
				if m, ok := c().(HistoryRefreshedMsg); ok {
					got = m
					ok = true
					break
				}
			}
		}
		if !ok {
			t.Fatalf("Cmd produced %T, want HistoryRefreshedMsg", msg)
		}
	}
	if len(got.Rows) != 2 {
		t.Errorf("HistoryRefreshedMsg.Rows = %d, want 2", len(got.Rows))
	}

	// Feed the msg back into Update — table commits.
	_, _ = p.Update(got, s)
	if got := len(p.tbl.Rows()); got != 2 {
		t.Errorf("after commit table has %d rows, want 2", got)
	}
	if p.pendingAt != 0 {
		t.Errorf("pendingAt should reset after commit, got %d", p.pendingAt)
	}
}

func TestHistoryPanel_AsyncRefresh_StaleResultDropped(t *testing.T) {
	p := NewHistory()
	stale := HistoryRefreshedMsg{Hash: 42, Rows: nil}
	// Feeding a msg whose hash doesn't match pendingAt (== 0 on a
	// fresh panel) must be a no-op.
	_, _ = p.Update(stale, &state.SharedState{})
	if got := len(p.tbl.Rows()); got != 0 {
		t.Errorf("stale msg committed %d rows", got)
	}
}

func TestHistoryPanel_AsyncRefresh_NoDuplicateDispatch(t *testing.T) {
	p := NewHistory()
	s := &state.SharedState{History: []client.CallRow{
		{ID: 1, GroupID: 100, System: "A",
			StartedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
	}}
	_, cmd1 := p.Update(historyMsg{}, s)
	if cmd1 == nil {
		t.Fatal("first Update should dispatch a refresh")
	}
	// Second Update with the same snapshot: pendingAt is set, so we
	// must not dispatch a second goroutine for the same hash.
	_, cmd2 := p.Update(historyMsg{}, s)
	if cmd2 != nil {
		// cmd2 may carry the table's own bubbles/table Cmd, but it
		// must not contain a buildHistoryRowsCmd. The simplest check:
		// only one refresh msg arrives across both Cmds.
		msg2 := cmd2()
		if _, ok := msg2.(HistoryRefreshedMsg); ok {
			t.Errorf("second Update dispatched a duplicate refresh")
		}
	}
}
