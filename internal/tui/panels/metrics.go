package panels

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

// MetricsPanel surfaces a curated subset of /metrics. Showing every
// Prometheus series produced by the daemon would be noisy; this
// whitelist captures the headline counters operators usually care
// about.
type MetricsPanel struct {
	tbl table.Model
}

// curatedMetrics is the order metrics are displayed in. Names that
// are missing from the daemon's output are rendered as "—".
var curatedMetrics = []string{
	"gophertrunk_calls_active",
	"gophertrunk_calls_total",
	"gophertrunk_grants_total",
	"gophertrunk_cc_locked",
	"gophertrunk_sse_clients",
	"gophertrunk_devices_attached",
	"gophertrunk_tone_alerts_total",
}

func NewMetrics() *MetricsPanel {
	t := table.New(table.WithFocused(true), table.WithColumns([]table.Column{
		{Title: "Metric", Width: 40},
		{Title: "Value", Width: 12},
	}))
	t.SetStyles(tableStyles())
	return &MetricsPanel{tbl: t}
}

func (MetricsPanel) Title() string { return "Metrics" }

var metricsSweepKey = key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "retention sweep"))

func (MetricsPanel) Keys() []key.Binding { return []key.Binding{metricsSweepKey} }

func (p *MetricsPanel) Update(msg tea.Msg, s *state.SharedState) (Panel, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, metricsSweepKey) {
		req := state.WriteRequest{
			Confirm:        "Run a retention sweep now?",
			Label:          "retention sweep",
			Kind:           state.WriteKindSweepRetention,
			SweepRetention: &state.SweepRetentionReq{},
		}
		return p, Emit(req)
	}
	p.refresh(s.Metrics)
	var cmd tea.Cmd
	p.tbl, cmd = p.tbl.Update(msg)
	return p, cmd
}

func (p *MetricsPanel) refresh(m map[string]float64) {
	rows := make([]table.Row, 0, len(curatedMetrics)+8)
	seen := map[string]bool{}
	for _, name := range curatedMetrics {
		seen[name] = true
		val, ok := m[name]
		if !ok {
			rows = append(rows, table.Row{name, "—"})
			continue
		}
		rows = append(rows, table.Row{name, formatMetric(val)})
	}
	// Trailing extras: any gophertrunk_* metric we didn't whitelist,
	// sorted, so unfamiliar additions still surface.
	extras := make([]string, 0)
	for k := range m {
		if seen[k] {
			continue
		}
		if strings.HasPrefix(k, "gophertrunk_") {
			extras = append(extras, k)
		}
	}
	sort.Strings(extras)
	for _, k := range extras {
		rows = append(rows, table.Row{k, formatMetric(m[k])})
	}
	p.tbl.SetRows(rows)
}

func formatMetric(v float64) string {
	if v == float64(int64(v)) {
		return fmt.Sprintf("%d", int64(v))
	}
	return fmt.Sprintf("%.3f", v)
}

func (p *MetricsPanel) View(width, height int, focused bool, s *state.SharedState) string {
	cols := []table.Column{{Title: "Metric", Width: width * 3 / 4}, {Title: "Value", Width: width / 4}}
	p.tbl.SetColumns(cols)
	p.tbl.SetWidth(width)
	if height > 4 {
		p.tbl.SetHeight(height - 2)
	}
	return panelFrame("Metrics", width, height, focused, p.tbl.View())
}
