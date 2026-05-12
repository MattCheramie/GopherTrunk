package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/state"
)

// DevicesPanel renders the SDR pool snapshot — every dongle the
// daemon has opened, with role / tuner / configured gain / PPM /
// bias-tee state.
type DevicesPanel struct {
	tbl      table.Model
	lastHash uint64
}

func NewDevices() *DevicesPanel {
	t := table.New(
		table.WithColumns(devicesColumns(80)),
		table.WithFocused(true),
	)
	t.SetStyles(tableStyles())
	return &DevicesPanel{tbl: t}
}

func (DevicesPanel) Title() string       { return "Devices" }
func (DevicesPanel) Keys() []key.Binding { return nil }

func (p *DevicesPanel) Update(msg tea.Msg, s *state.SharedState) (Panel, tea.Cmd) {
	h := hashRows(s.Devices, func(d client.SDRStatus) string {
		return fmt.Sprintf("%s|%s|%s|%s|%v|%d|%d|%v|%v",
			d.Serial, d.Driver, d.TunerName, d.Role,
			d.GainAuto, d.GainTenthDB, d.PPM, d.BiasTee, d.Attached)
	})
	if h != p.lastHash {
		p.refresh(s.Devices)
		p.lastHash = h
	}
	var cmd tea.Cmd
	p.tbl, cmd = p.tbl.Update(msg)
	return p, cmd
}

func (p *DevicesPanel) refresh(devs []client.SDRStatus) {
	rows := make([]table.Row, 0, len(devs))
	for _, d := range devs {
		gain := "auto"
		if !d.GainAuto {
			gain = fmt.Sprintf("%d.%d dB", d.GainTenthDB/10, abs(d.GainTenthDB%10))
		}
		ppm := fmt.Sprintf("%d", d.PPM)
		bias := "off"
		if d.BiasTee {
			bias = "on"
		}
		status := "detached"
		if d.Attached {
			status = "attached"
		}
		rows = append(rows, table.Row{
			d.Serial,
			d.Driver,
			d.TunerName,
			d.Role,
			gain,
			ppm,
			bias,
			status,
		})
	}
	p.tbl.SetRows(rows)
}

func (p *DevicesPanel) View(width, height int, focused bool, s *state.SharedState) string {
	p.tbl.SetColumns(devicesColumns(width))
	p.tbl.SetWidth(width)
	if height > 4 {
		p.tbl.SetHeight(height - 4)
	}
	body := p.tbl.View()
	if len(s.Devices) == 0 {
		body = strings.TrimRight(body, "\n") + "\n\n" + dashDim.Render("no devices opened — check `gophertrunk sdr list` and your config")
	}
	if s.DevicesErr != nil {
		body = strings.TrimRight(body, "\n") + "\n\n" + dashErr.Render("devices: "+s.DevicesErr.Error())
	}
	return panelFrame("Devices", width, height, focused, body)
}

// Reveal positions the cursor on the row whose Serial matches key.
func (p *DevicesPanel) Reveal(key string) {
	for i, row := range p.tbl.Rows() {
		if len(row) > 0 && row[0] == key {
			p.tbl.SetCursor(i)
			return
		}
	}
}

// HandleMouseAt moves the cursor to the clicked data row.
func (p *DevicesPanel) HandleMouseAt(_, localY int) tea.Cmd {
	idx := tableRowFromLocalY(localY, len(p.tbl.Rows()))
	if idx >= 0 {
		p.tbl.SetCursor(idx)
	}
	return nil
}

func devicesColumns(w int) []table.Column {
	if w < 60 {
		w = 60
	}
	serialW := 14
	driverW := 8
	tunerW := 10
	roleW := 8
	gainW := 9
	ppmW := 6
	biasW := 6
	stateW := w - serialW - driverW - tunerW - roleW - gainW - ppmW - biasW - 4
	if stateW < 8 {
		stateW = 8
	}
	return []table.Column{
		{Title: "Serial", Width: serialW},
		{Title: "Driver", Width: driverW},
		{Title: "Tuner", Width: tunerW},
		{Title: "Role", Width: roleW},
		{Title: "Gain", Width: gainW},
		{Title: "PPM", Width: ppmW},
		{Title: "BiasT", Width: biasW},
		{Title: "State", Width: stateW},
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
