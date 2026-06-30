package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/rfscope"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// runRFScopeCockpit launches the Bubbletea scene cockpit — live RF network
// monitoring (top talkers, protocol hierarchy, per-channel I/O graph,
// conversations, expert info) over a capture file or a live SDR. It is modeled
// on siglab_tui.go: a self-contained tea.Program whose worker goroutine runs the
// rfscope segmentation + analyzers and feeds Scenes back into the event loop.
func runRFScopeCockpit(args []string) {
	fs := flag.NewFlagSet("rfscope cockpit", flag.ExitOnError)
	verboseFlag := fs.Bool("verbose-errors", false, "print full error chain + stack on failures")
	in := fs.String("in", "", "raw IQ capture file (offline cockpit)")
	serial := fs.String("serial", "", "SDR serial for a live cockpit")
	format := fs.String("format", "f32", "sample format: u8 | f32 (file mode)")
	freq := fs.Uint64("freq", 0, "centre frequency in Hz (required for live)")
	sampleRate := fs.Float64("sample-rate", 2_400_000, "IQ sample rate in Hz")
	duration := fs.Float64("duration", 4, "seconds of IQ per live refresh")
	refresh := fs.Duration("refresh", 2*time.Second, "live re-analysis interval")
	af := registerAnalysisFlags(fs)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `gophertrunk rfscope cockpit — live RF scene monitor (TUI).

USAGE:
  gophertrunk rfscope cockpit -in <capture> [-format u8|f32] [-sample-rate Hz]
  gophertrunk rfscope cockpit -serial <sdr> -freq Hz [-duration sec] [-refresh dur]

KEYS: q quit   space freeze/resume   e export scene JSON   ? help

FLAGS:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)
	resolveVerbose(*verboseFlag, false)
	rep := newReporter("rfscope")

	if *in == "" && *serial == "" {
		fs.Usage()
		rep.Fatalf(2, "provide -in <capture> or -serial <sdr>")
	}

	var (
		src   rfscope.Source
		label string
		live  bool
		maxN  int
	)
	if *serial != "" {
		if *freq == 0 {
			rep.Fatalf(2, "-freq is required for a live cockpit")
		}
		dev, info, err := openCaptureDevice(*serial)
		if err != nil {
			rep.Fatal(1, err)
		}
		defer dev.Close()
		iqsrc, err := newDeviceIQSource(context.Background(), dev, uint32(*sampleRate), nil)
		if err != nil {
			rep.Fatal(1, fmt.Errorf("start IQ stream: %w", err))
		}
		ls, err := rfscope.NewLiveSource(iqsrc, uint32(*freq))
		if err != nil {
			rep.Fatal(1, err)
		}
		src, label, live, maxN = ls, fmt.Sprintf("live:%s@%.4fMHz", info.Serial, float64(*freq)/1e6), true, int(*duration*float64(*sampleRate))
	} else {
		sampleFormat, err := siglab.ParseSampleFormat(*format)
		if err != nil {
			rep.Fatal(2, err)
		}
		fsrc, err := rfscope.OpenFile(*in, sampleFormat, uint32(*freq), *sampleRate)
		if err != nil {
			rep.Fatal(1, err)
		}
		defer fsrc.Close()
		src, label, live = fsrc, *in, false
	}

	m := newCockpit(src, af.segConfig(maxN), af.selectedAnalyzers(), label, *refresh, live)
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		rep.Fatal(1, fmt.Errorf("cockpit: %w", err))
	}
}

// cockpitSceneMsg carries a freshly analyzed Scene from the worker goroutine.
type cockpitSceneMsg struct {
	scene *rfscope.Scene
	err   error
}

type cockpitModel struct {
	src       rfscope.Source
	cfg       rfscope.SegmentConfig
	analyzers []string
	label     string
	refresh   time.Duration
	live      bool

	scene   *rfscope.Scene
	err     error
	status  string
	frozen  bool
	showAll bool // help overlay
	width   int
	height  int

	sceneCh chan cockpitSceneMsg
}

func newCockpit(src rfscope.Source, cfg rfscope.SegmentConfig, analyzers []string, label string, refresh time.Duration, live bool) cockpitModel {
	return cockpitModel{
		src: src, cfg: cfg, analyzers: analyzers, label: label, refresh: refresh, live: live,
		status:  "analyzing…",
		sceneCh: make(chan cockpitSceneMsg, 1),
	}
}

func (m cockpitModel) Init() tea.Cmd {
	return tea.Batch(m.analyzeOnce(), m.waitScene())
}

// analyzeOnce runs segmentation + analyzers over the next window in a goroutine.
func (m cockpitModel) analyzeOnce() tea.Cmd {
	src, cfg, analyzers, label, ch := m.src, m.cfg, m.analyzers, m.label, m.sceneCh
	return func() tea.Msg {
		go func() {
			scene, in, err := rfscope.Segment(context.Background(), src, cfg)
			if err == nil {
				scene.Source = label
				err = rfscope.Run(context.Background(), scene, in, analyzers...)
			}
			select {
			case ch <- cockpitSceneMsg{scene: scene, err: err}:
			default:
			}
		}()
		return nil
	}
}

func (m cockpitModel) waitScene() tea.Cmd {
	ch := m.sceneCh
	return func() tea.Msg { return <-ch }
}

func (m cockpitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case " ":
			m.frozen = !m.frozen
			return m, nil
		case "?":
			m.showAll = !m.showAll
			return m, nil
		case "e":
			m.status = m.exportScene()
			return m, nil
		}
		return m, nil
	case cockpitSceneMsg:
		if msg.err != nil {
			m.err = msg.err
			m.status = "analysis error: " + msg.err.Error()
		} else if !m.frozen {
			m.scene = msg.scene
			m.err = nil
			m.status = m.statusLine()
		}
		// Schedule the next pass: live re-analyzes on the refresh tick; offline
		// is a one-shot.
		if m.live {
			return m, tea.Batch(m.waitScene(), tea.Tick(m.refresh, func(time.Time) tea.Msg { return cockpitTickMsg{} }))
		}
		return m, nil
	case cockpitTickMsg:
		if m.frozen {
			return m, m.waitScene()
		}
		return m, tea.Batch(m.analyzeOnce(), m.waitScene())
	}
	return m, nil
}

type cockpitTickMsg struct{}

func (m cockpitModel) statusLine() string {
	froz := ""
	if m.frozen {
		froz = "  [FROZEN]"
	}
	mode := "offline"
	if m.live {
		mode = "live"
	}
	return fmt.Sprintf("%s%s   q quit · space freeze · e export · ? help", mode, froz)
}

func (m cockpitModel) exportScene() string {
	if m.scene == nil {
		return "nothing to export yet"
	}
	path := fmt.Sprintf("rfscope-scene-%d.json", time.Now().Unix())
	f, err := os.Create(path)
	if err != nil {
		return "export failed: " + err.Error()
	}
	defer f.Close()
	if err := rfscope.Write(f, m.scene, rfscope.FormatJSON); err != nil {
		return "export failed: " + err.Error()
	}
	return "exported → " + path
}

var (
	ckTitle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	ckDim   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	ckAlert = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	ckWarn  = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
)

func (m cockpitModel) View() string {
	if m.showAll {
		return cockpitHelp()
	}
	var b strings.Builder
	b.WriteString(ckTitle.Render("GopherTrunk RFScope — Scene Cockpit") + "  " + ckDim.Render(m.label) + "\n")
	if m.scene == nil {
		b.WriteString("\n" + m.status + "\n")
		return b.String()
	}
	b.WriteString(renderSceneHeader(m.scene) + "\n")
	b.WriteString(renderHierarchyPanel(m.scene) + "\n")
	b.WriteString(renderChannelsPanel(m.scene, panelWidth(m.width)) + "\n")
	b.WriteString(renderTalkersPanel(m.scene) + "\n")
	b.WriteString(renderConversationsPanel(m.scene) + "\n")
	b.WriteString(renderExpertPanel(m.scene) + "\n")
	b.WriteString(ckDim.Render(m.status))
	return b.String()
}

func panelWidth(w int) int {
	if w <= 0 {
		return 40
	}
	if w > 60 {
		return 40
	}
	return w - 20
}

func cockpitHelp() string {
	return ckTitle.Render("RFScope Cockpit — keys") + "\n\n" +
		"  q / ctrl+c   quit\n" +
		"  space        freeze / resume the live view\n" +
		"  e            export the current scene as JSON\n" +
		"  ?            toggle this help\n\n" +
		ckDim.Render("press ? to return")
}

// --- pure render helpers (unit-tested without a TTY) ---

func renderSceneHeader(sc *rfscope.Scene) string {
	return fmt.Sprintf("center %.4f MHz  span %.3f MHz  window %.1fs  floor %.0f dBFS  |  %d bursts  %d ch  %d emitters  %d anomalies",
		float64(sc.CenterHz)/1e6, float64(sc.SpanHz)/1e6, sc.WindowSec, sc.NoiseFloorDbFS,
		len(sc.Bursts), len(sc.Channels), len(sc.Emitters), len(sc.Anomalies))
}

func renderHierarchyPanel(sc *rfscope.Scene) string {
	var b strings.Builder
	b.WriteString(ckTitle.Render("Protocol hierarchy"))
	if len(sc.Hierarchy.Nodes) == 0 {
		b.WriteString("  " + ckDim.Render("(none)"))
		return b.String()
	}
	for _, n := range sc.Hierarchy.Nodes {
		indent := strings.Repeat("  ", n.Depth+1)
		b.WriteString(fmt.Sprintf("\n%s%-20s %3d bursts  %4.0f%% time", indent, n.Label, n.Stats.BurstCount, n.Stats.TimePct))
	}
	return b.String()
}

func renderChannelsPanel(sc *rfscope.Scene, width int) string {
	var b strings.Builder
	b.WriteString(ckTitle.Render("Channels (I/O graph)"))
	if len(sc.Channels) == 0 {
		b.WriteString("  " + ckDim.Render("(none)"))
		return b.String()
	}
	for _, c := range sc.Channels {
		b.WriteString(fmt.Sprintf("\n  %10.4f %-6s %s %4.0f%%",
			float64(c.FreqHz)/1e6, string(c.DominantClass), renderSparkline(c.Timeline, width), c.DutyCycle*100))
	}
	return b.String()
}

// renderSparkline compresses a channel's occupancy timeline to a fixed-width
// unicode sparkline: a filled block per occupied bucket, a dot when idle.
func renderSparkline(timeline []rfscope.Bin, width int) string {
	if width < 1 {
		width = 1
	}
	if len(timeline) == 0 {
		return strings.Repeat("·", width)
	}
	out := make([]rune, width)
	for i := 0; i < width; i++ {
		lo := i * len(timeline) / width
		hi := (i + 1) * len(timeline) / width
		if hi <= lo {
			hi = lo + 1
		}
		occupied := false
		for j := lo; j < hi && j < len(timeline); j++ {
			if timeline[j].Occupied {
				occupied = true
				break
			}
		}
		if occupied {
			out[i] = '█'
		} else {
			out[i] = '·'
		}
	}
	return string(out)
}

func renderTalkersPanel(sc *rfscope.Scene) string {
	var b strings.Builder
	b.WriteString(ckTitle.Render("Top talkers"))
	if len(sc.Emitters) == 0 {
		b.WriteString("  " + ckDim.Render("(none)"))
		return b.String()
	}
	n := len(sc.Emitters)
	if n > 6 {
		n = 6
	}
	for _, e := range sc.Emitters[:n] {
		hop := ""
		if len(e.HopSet) > 1 {
			hop = fmt.Sprintf(" hop×%d", len(e.HopSet))
		}
		b.WriteString(fmt.Sprintf("\n  #%-3d %-28s %5.2fs%s", e.ID, e.Label, e.TotalAirtimeSec, hop))
	}
	return b.String()
}

func renderConversationsPanel(sc *rfscope.Scene) string {
	var b strings.Builder
	b.WriteString(ckTitle.Render("Conversations"))
	if len(sc.Conversations) == 0 {
		b.WriteString("  " + ckDim.Render("(none)"))
		return b.String()
	}
	for _, c := range sc.Conversations {
		ids := make([]string, len(c.EmitterIDs))
		for i, id := range c.EmitterIDs {
			ids[i] = fmt.Sprintf("#%d", id)
		}
		b.WriteString(fmt.Sprintf("\n  %-16s %s (%.2f)", c.Kind, strings.Join(ids, "↔"), c.Correlation))
	}
	return b.String()
}

func renderExpertPanel(sc *rfscope.Scene) string {
	var b strings.Builder
	b.WriteString(ckTitle.Render("Expert info"))
	if len(sc.Anomalies) == 0 {
		b.WriteString("  " + ckDim.Render("(none)"))
		return b.String()
	}
	// Already severity-sorted by the expert analyzer; render with color.
	shown := sc.Anomalies
	if len(shown) > 8 {
		shown = shown[:8]
	}
	for _, a := range shown {
		line := fmt.Sprintf("\n  [%s] %-16s %s", a.Severity, a.Kind, a.Detail)
		switch a.Severity {
		case "alert":
			b.WriteString(ckAlert.Render(line))
		case "warn":
			b.WriteString(ckWarn.Render(line))
		default:
			b.WriteString(line)
		}
	}
	return b.String()
}

// sortedAnomalyKinds is a small helper kept for deterministic test assertions.
func sortedAnomalyKinds(sc *rfscope.Scene) []string {
	ks := make([]string, 0, len(sc.Anomalies))
	for _, a := range sc.Anomalies {
		ks = append(ks, a.Kind)
	}
	sort.Strings(ks)
	return ks
}
