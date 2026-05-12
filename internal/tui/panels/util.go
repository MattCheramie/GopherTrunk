package panels

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/maphash"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

// hashSeed is the per-process maphash seed. Held module-wide so
// successive hashRows calls on the same payload return the same
// digest across an entire daemon run.
var hashSeed = maphash.MakeSeed()

// hashRows produces a stable digest over a slice of rows. Panels
// call it on the latest snapshot from SharedState; if the digest
// hasn't changed since the last refresh, the bubbles/table rebuild
// is skipped — the user-flagged "rebuilt every tick" path.
//
// keyFn must return a deterministic string for each row covering
// every field the table renders. Order matters; reordering a slice
// counts as a change even if the contents are identical.
func hashRows[T any](rows []T, keyFn func(T) string) uint64 {
	var h maphash.Hash
	h.SetSeed(hashSeed)
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(len(rows)))
	_, _ = h.Write(buf[:])
	for _, r := range rows {
		_, _ = h.WriteString(keyFn(r))
		_, _ = h.Write([]byte{0})
	}
	return h.Sum64()
}

// hashStringMap is the map-flavoured sibling of hashRows. metrics.go
// holds map[string]float64; we sort keys so the digest is stable
// across iteration order.
func hashStringMap(m map[string]float64) uint64 {
	var h maphash.Hash
	h.SetSeed(hashSeed)
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// stable sort without importing sort package for one call site —
	// insertion sort over short maps is fine.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j-1] > keys[j]; j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
	for _, k := range keys {
		_, _ = h.WriteString(k)
		_, _ = h.Write([]byte{0})
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], uint64(int64(m[k]*1000)))
		_, _ = h.Write(buf[:])
	}
	return h.Sum64()
}

// jsonUnmarshal centralises the json.Unmarshal call so panels don't
// each pull in encoding/json directly. Returns nil on empty raw.
func jsonUnmarshal(raw json.RawMessage, out any) error {
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}

// truncate clips s to n runes, appending an ellipsis if truncated.
func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}

// since formats t as "5s", "2m", "1h" relative to now.
func since(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	d := time.Since(t)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// containsFold is a case-insensitive substring check for table
// filtering.
func containsFold(haystack, needle string) bool {
	if needle == "" {
		return true
	}
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

// tableRowFromLocalY maps a panel-local Y (0 = panel's top border) to
// a bubbles/table row index, accounting for the canonical chrome that
// panelFrame draws: top border, title line, table column header. The
// returned index is clamped to [0, rowCount-1]; -1 means the click
// landed on chrome, not a row.
func tableRowFromLocalY(localY, rowCount int) int {
	// panelFrame: row 0 border, row 1 title, row 2 column header.
	// First data row sits at local Y == 3.
	idx := localY - 3
	if idx < 0 {
		return -1
	}
	if rowCount == 0 {
		return -1
	}
	if idx >= rowCount {
		return rowCount - 1
	}
	return idx
}

// applyThemeIfChanged re-applies tableStyles() to the supplied table
// when msg is a ThemeChangedMsg. Table panels call this at the top
// of Update so a runtime palette swap (Ctrl+T) takes effect on the
// next render without the operator restarting the TUI.
func applyThemeIfChanged(msg tea.Msg, tbl *table.Model) {
	if _, ok := msg.(ThemeChangedMsg); ok {
		tbl.SetStyles(tableStyles())
	}
}

// handleTableMouse is the canonical MouseAware implementation for the
// bubbles/table-backed panels. Translates a press-left into a row
// cursor and forwards wheel ticks one-row-at-a-time. chromeRows is
// the number of body lines drawn above the table (always 0 for
// systems/devices/active/history/tones/metrics; 1 for talkgroups
// which renders a filter row first).
//
// Bubbles v1.0.0's table.Update is KeyMsg-only — it ignores MouseMsg
// entirely — so wheel forwarding has to live in the host panel. We
// centralise it here so every table panel handles wheel scroll the
// same way without each one importing tea constants separately.
func handleTableMouse(tbl interface {
	Cursor() int
	Rows() []table.Row
	SetCursor(int)
	MoveUp(int)
	MoveDown(int)
}, msg tea.MouseMsg, localY, chromeRows int) {
	switch msg.Button {
	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress {
			return
		}
		idx := tableRowFromLocalY(localY-chromeRows, len(tbl.Rows()))
		if idx >= 0 {
			tbl.SetCursor(idx)
		}
	case tea.MouseButtonWheelUp:
		tbl.MoveUp(1)
	case tea.MouseButtonWheelDown:
		tbl.MoveDown(1)
	}
}
