package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// runSiglabTUI is the entry point for `gophertrunk siglab` — a standalone
// (no-daemon) replay/test/analysis terminal UI. It picks a protocol, runs
// the siglab engine against a capture file, streams live decode events, and
// shows a signal-quality dashboard with one-key structured export. It is a
// self-contained bubbletea app (modeled on import_tui.go), not wired into
// the daemon operator TUI, because it runs entirely offline against a file.
func runSiglabTUI(args []string) {
	fs := flag.NewFlagSet("siglab", flag.ExitOnError)
	verboseFlag := fs.Bool("verbose-errors", false, "print full error chain + stack on failures")
	in := fs.String("in", "", "raw IQ capture file (required)")
	format := fs.String("format", "f32", "sample format: u8 | f32")
	sampleRate := fs.Float64("sample-rate", 2_400_000, "IQ sample rate in Hz")
	protocolFlag := fs.String("protocol", "", "protocol to decode (empty ⇒ pick in the TUI)")
	freq := fs.Uint64("freq", 0, "informational: the capture's nominal centre frequency in Hz")
	autoTune := fs.Bool("auto-tune", false, "estimate the carrier offset and tune to 0 Hz before demod")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `gophertrunk siglab — standalone replay/test/analysis TUI.

USAGE:
  gophertrunk siglab -in <path> [-protocol <p>] [-format u8|f32] [-sample-rate Hz]

Pick a protocol (if not given), run the decode, watch live events, view the
signal-quality dashboard, and press e to export a JSON report.

FLAGS:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)
	resolveVerbose(*verboseFlag, false)
	rep := newReporter("siglab")

	if *in == "" {
		fs.Usage()
		rep.Fatalf(2, "-in is required")
	}
	sampleFormat, err := siglab.ParseSampleFormat(*format)
	if err != nil {
		rep.Fatal(2, err)
	}
	var preProto trunking.Protocol
	havePreProto := false
	if *protocolFlag != "" {
		p, perr := siglab.ParseProtocolCLI(*protocolFlag)
		if perr != nil {
			rep.Fatal(2, perr)
		}
		preProto = p
		havePreProto = true
	}

	m := newSiglabTUI(*in, sampleFormat, *sampleRate, uint32(*freq), *autoTune, preProto, havePreProto)
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		rep.Fatal(1, fmt.Errorf("tui: %w", err))
	}
}

type siglabView int

const (
	siglabPickProtocol siglabView = iota
	siglabRunning
	siglabDashboard
)

// siglabEventMsg / siglabDoneMsg carry engine progress back into the bubbletea
// event loop from the worker goroutine.
type siglabEventMsg siglab.EventRecord
type siglabDoneMsg struct {
	res *siglab.Result
	err error
}

type siglabTUIModel struct {
	// inputs
	capture    string
	format     siglab.SampleFormat
	sampleRate float64
	freqHz     uint32
	autoTune   bool

	// protocol selection
	protos   []trunking.Protocol
	cursor   int
	protocol trunking.Protocol

	view   siglabView
	events []string
	vp     viewport.Model
	result *siglab.Result
	runErr error
	status string
	width  int
	height int

	// eventCh is fed by the worker goroutine; waitEvent pumps it into Update.
	eventCh chan siglab.EventRecord
	doneCh  chan siglabDoneMsg
}

func newSiglabTUI(capture string, format siglab.SampleFormat, sampleRate float64, freqHz uint32, autoTune bool, proto trunking.Protocol, havePreProto bool) siglabTUIModel {
	m := siglabTUIModel{
		capture:    capture,
		format:     format,
		sampleRate: sampleRate,
		freqHz:     freqHz,
		autoTune:   autoTune,
		protos:     ccdecoder.RegisteredProtocols(),
		vp:         viewport.New(80, 16),
		eventCh:    make(chan siglab.EventRecord, 256),
		doneCh:     make(chan siglabDoneMsg, 1),
	}
	if havePreProto {
		m.protocol = proto
		m.view = siglabRunning
	}
	return m
}

func (m siglabTUIModel) Init() tea.Cmd {
	if m.view == siglabRunning {
		return tea.Batch(m.startRun(), m.waitEvent())
	}
	return nil
}

// startRun launches the engine in a goroutine, feeding EventRecords to
// eventCh and the final Result to doneCh.
func (m siglabTUIModel) startRun() tea.Cmd {
	cfg := siglab.Config{
		Protocol:      m.protocol,
		SystemName:    "siglab",
		FrequencyHz:   m.freqHz,
		SampleRateHz:  m.sampleRate,
		Format:        m.format,
		AutoTune:      m.autoTune,
		CollectIQDiag: true,
	}
	capture := m.capture
	eventCh := m.eventCh
	doneCh := m.doneCh
	return func() tea.Msg {
		go func() {
			res, err := siglab.RunStream(capture, cfg, func(ev siglab.EventRecord) {
				select {
				case eventCh <- ev:
				default: // drop on a slow UI rather than block the engine
				}
			})
			doneCh <- siglabDoneMsg{res: res, err: err}
		}()
		return nil
	}
}

// waitEvent blocks on the event/done channels and turns the next signal into
// a bubbletea message, re-arming itself until the run completes.
func (m siglabTUIModel) waitEvent() tea.Cmd {
	eventCh := m.eventCh
	doneCh := m.doneCh
	return func() tea.Msg {
		select {
		case ev := <-eventCh:
			return siglabEventMsg(ev)
		case done := <-doneCh:
			return done
		}
	}
}

func (m siglabTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.vp.Width = msg.Width - 2
		if msg.Height > 12 {
			m.vp.Height = msg.Height - 10
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case siglabEventMsg:
		m.events = append(m.events, fmt.Sprintf("%7.2fs  %-16s %s", msg.OffsetSec, msg.Kind, briefFields(siglab.EventRecord(msg))))
		m.vp.SetContent(strings.Join(m.events, "\n"))
		m.vp.GotoBottom()
		return m, m.waitEvent()

	case siglabDoneMsg:
		m.result = msg.res
		m.runErr = msg.err
		m.view = siglabDashboard
		if msg.err != nil {
			m.status = "run failed: " + msg.err.Error()
		} else {
			m.status = "decode complete — e: export JSON   q: quit"
		}
		return m, nil
	}
	return m, nil
}

func (m siglabTUIModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	}
	switch m.view {
	case siglabPickProtocol:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.protos)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.protos) > 0 {
				m.protocol = m.protos[m.cursor]
				m.view = siglabRunning
				return m, tea.Batch(m.startRun(), m.waitEvent())
			}
		}
	case siglabDashboard:
		if msg.String() == "e" && m.result != nil {
			m.status = m.exportJSON()
		}
	}
	return m, nil
}

func (m siglabTUIModel) exportJSON() string {
	path := strings.TrimSuffix(m.capture, ext(m.capture)) + ".siglab.json"
	f, err := os.Create(path)
	if err != nil {
		return "export failed: " + err.Error()
	}
	defer f.Close()
	if err := siglab.WriteResult(f, m.result, siglab.FormatJSON); err != nil {
		return "export failed: " + err.Error()
	}
	return "exported → " + path
}

var (
	siglabTitle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	siglabAccent = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	siglabDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	siglabSelect = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("12"))
)

func (m siglabTUIModel) View() string {
	switch m.view {
	case siglabPickProtocol:
		return m.viewPick()
	case siglabRunning:
		return m.viewRunning()
	default:
		return m.viewDashboard()
	}
}

func (m siglabTUIModel) viewPick() string {
	var b strings.Builder
	b.WriteString(siglabTitle.Render("GopherTrunk Signal Lab") + "  " + siglabDim.Render(m.capture) + "\n\n")
	b.WriteString("Select a protocol to decode:\n\n")
	for i, p := range m.protos {
		line := "  " + p.String()
		if i == m.cursor {
			line = siglabSelect.Render("▸ " + p.String())
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n" + siglabDim.Render("↑/↓ move   enter run   q quit"))
	return b.String()
}

func (m siglabTUIModel) viewRunning() string {
	var b strings.Builder
	b.WriteString(siglabTitle.Render("Decoding "+m.protocol.String()) + "  " + siglabDim.Render(m.capture) + "\n\n")
	b.WriteString(m.vp.View())
	b.WriteString("\n" + siglabDim.Render(fmt.Sprintf("%d events   q quit", len(m.events))))
	return b.String()
}

func (m siglabTUIModel) viewDashboard() string {
	var b strings.Builder
	b.WriteString(siglabTitle.Render("Signal Lab — Results") + "\n\n")
	if m.runErr != nil {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(m.status) + "\n")
		return b.String()
	}
	r := m.result
	lockStr := "NOT LOCKED"
	if r.Locked {
		lockStr = siglabAccent.Render(fmt.Sprintf("LOCKED @ %d Hz (%.2fs)", lockFreq(r), r.LockLatencySec))
	}
	b.WriteString(fmt.Sprintf("Protocol:   %s\n", r.Protocol))
	b.WriteString(fmt.Sprintf("Samples:    %d (%.2fs @ %.0f Hz)\n", r.TotalSamples, r.DurationSec, r.SampleRateHz))
	b.WriteString(fmt.Sprintf("Symbols:    %d   effective baud %.1f (expected %.0f, %+.1f%%)\n",
		r.Symbols, r.EffectiveBaud, r.ExpectedBaud, r.BaudDeviationPct))
	b.WriteString(fmt.Sprintf("Lock:       %s\n", lockStr))
	b.WriteString(fmt.Sprintf("Grants:     %d\n", len(r.Grants)))
	if r.Signal != nil {
		b.WriteString("\n" + siglabTitle.Render("Symbol histogram") + "\n")
		for i := range r.Signal.SymbolHistogram {
			pct := 0.0
			if i < len(r.Signal.SymbolHistogramPct) {
				pct = r.Signal.SymbolHistogramPct[i]
			}
			b.WriteString(fmt.Sprintf("  %d  %s %.1f%%\n", i, bar(pct), pct))
		}
		if r.Signal.IQObserved {
			b.WriteString(fmt.Sprintf("\nIQ imbalance: gain %+.2f dB  phase %+.2f°  image rej %.1f dB\n",
				r.Signal.IQGainImbalanceDB, r.Signal.IQPhaseImbalanceDeg, r.Signal.IQImageRejectionDB))
		}
		b.WriteString(fmt.Sprintf("Decode-error rate: %.2f / 1000 symbols\n", r.Signal.DecodeErrorRate))
		if d := r.Signal.Demod; d != nil {
			b.WriteString(fmt.Sprintf("Demod (%s): EVM %.1f%%  SNR≈%.1f dB\n",
				d.Modulation, d.EVMPct, d.SNREstimateDB))
		}
	}
	b.WriteString("\n" + siglabDim.Render(m.status))
	return b.String()
}

// bar renders a small percentage bar (0..100%) for the dashboard histogram.
func bar(pct float64) string {
	const width = 24
	n := int(pct / 100 * width)
	if n < 0 {
		n = 0
	}
	if n > width {
		n = width
	}
	return siglabAccent.Render(strings.Repeat("█", n)) + siglabDim.Render(strings.Repeat("·", width-n))
}

// briefFields renders a compact one-line view of an event's salient fields
// for the live log.
func briefFields(ev siglab.EventRecord) string {
	var parts []string
	for _, k := range []string{"NAC", "ColorCode", "SystemID", "FrequencyHz", "GroupID", "SourceID", "Stage"} {
		if v, ok := ev.Fields[k]; ok {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
	}
	return strings.Join(parts, " ")
}
