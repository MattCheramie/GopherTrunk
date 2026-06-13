package hunt

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"log/slog"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
	"github.com/MattCheramie/GopherTrunk/internal/survey"
)

// surveyRunKey derives a stable filename key from a run's band set / candidate
// list / forced protocol so a resumed survey reuses the same NDJSON file.
func surveyRunKey(opts LiveHuntOptions) string {
	h := fnv.New64a()
	for _, b := range opts.Bands {
		fmt.Fprintf(h, "b%d:%d;", b.LowHz, b.HighHz)
	}
	for _, c := range opts.Candidates {
		fmt.Fprintf(h, "c%d;", c)
	}
	fmt.Fprintf(h, "p%d;", opts.Protocol)
	return strconv.FormatUint(h.Sum64(), 16)
}

// RunState is the lifecycle of a live hunt run, surfaced to the cockpit/REST.
type RunState string

const (
	StateRunIdle    RunState = "idle"    // no run has started yet
	StateRunActive  RunState = "running" // a sweep/identify run is in progress
	StateRunDone    RunState = "done"    // last run finished and produced a map
	StateRunStopped RunState = "stopped" // last run was cancelled by the operator
	StateRunFailed  RunState = "failed"  // last run errored (e.g. SDR acquisition)
)

// Acquirer obtains an IQSource for one run plus a release callback (resume the
// control-channel hunter, close the SDR subscription, …). The daemon supplies
// this so the hunt package stays free of SDR/pool dependencies; release is
// invoked exactly once when the run ends (success, error, or stop).
// Acquirer obtains an IQSource for one run plus a release callback. opts
// carries the run's parameters — notably opts.Serial, so the daemon can honor
// an operator-requested SDR — and is passed by the Manager when a run starts.
type Acquirer func(ctx context.Context, opts LiveHuntOptions) (src IQSource, release func(), err error)

// ManagerOptions configure a Manager.
type ManagerOptions struct {
	// Acquire obtains the run's IQSource. Required.
	Acquire Acquirer
	Bus     *events.Bus
	Log     *slog.Logger
	// SurveyDir, when set, is the directory the manager streams a survey run's
	// classified carriers into as crash-safe NDJSON (one file per band set), so a
	// web survey can resume after an interruption. Empty disables persistence.
	SurveyDir string
}

// Manager owns the daemon's single live-hunt run: it acquires an SDR, runs the
// sweep→identify→map pipeline in the background, publishes hunt.* bus events,
// and holds the latest discovered system for export/commit. Safe for
// concurrent use by the REST/TUI cockpit.
type Manager struct {
	acquire   Acquirer
	bus       *events.Bus
	log       *slog.Logger
	surveyDir string

	mu         sync.Mutex
	sink       *NDJSONSink // active survey persistence sink (nil when off)
	runID      int
	state      RunState
	runSurvey  bool // the active/last run is a survey (drives RunStatus.Mode)
	progress   LiveHuntProgress
	sys        *DiscoveredSystem
	survey     *SignalSurvey
	reports    []CaptureReport
	err        string
	gainRecs   []GainRecommendation
	gainNote   string
	startedAt  time.Time
	finishedAt time.Time
	cancel     context.CancelFunc

	// history retains the last maxRunHistory finished runs that produced a
	// system, keyed implicitly by id, so /api/v1/hunt/{id}/export|commit can
	// reach a prior run after a newer one starts.
	history []runRecord
}

// maxRunHistory bounds the retained finished-run records.
const maxRunHistory = 10

// runRecord is one finished run's discovered system + reports, kept in the
// Manager's bounded history.
type runRecord struct {
	id      int
	sys     *DiscoveredSystem
	survey  *SignalSurvey
	reports []CaptureReport
}

// ErrNoSuchRun is returned by id-addressed lookups when the run id is unknown
// or has been evicted from the bounded history.
var ErrNoSuchRun = errors.New("hunt: no such run id")

// NewManager builds a Manager. Acquire is required.
func NewManager(opts ManagerOptions) (*Manager, error) {
	if opts.Acquire == nil {
		return nil, errors.New("hunt: Manager requires an Acquirer")
	}
	log := opts.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &Manager{acquire: opts.Acquire, bus: opts.Bus, log: log, surveyDir: opts.SurveyDir, state: StateRunIdle}, nil
}

// RunStatus is the read snapshot the cockpit/REST renders.
type RunStatus struct {
	RunID      int              `json:"run_id"`
	State      RunState         `json:"state"`
	Running    bool             `json:"running"`
	Mode       string           `json:"mode"` // "hunt" | "survey"
	Progress   LiveHuntProgress `json:"progress"`
	Sites      int              `json:"sites"`
	Talkgroups int              `json:"talkgroups"`
	SystemName string           `json:"system_name,omitempty"`
	// Signals is the classified-carrier inventory of a survey run (nil for a
	// plain hunt). It is the survey's primary result, rendered by the cockpit.
	Signals []DetectedSignal `json:"signals,omitempty"`
	Error   string           `json:"error,omitempty"`

	// GainRecommendations is the per-control-channel result of an -auto-gain run
	// (nil when auto-gain wasn't requested or no CC locked). GainNote explains
	// why auto-gain produced nothing (e.g. the SDR is shared).
	GainRecommendations []GainRecommendation `json:"gain_recommendations,omitempty"`
	GainNote            string               `json:"gain_note,omitempty"`

	StartedAt  time.Time `json:"started_at,omitempty"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
}

// Start launches a live hunt with opts. It returns the new run id, or an error
// if a run is already active. The run executes in the background; observe it via
// Status and the hunt.* bus events.
func (m *Manager) Start(opts LiveHuntOptions) (int, error) {
	m.mu.Lock()
	if m.state == StateRunActive {
		m.mu.Unlock()
		return 0, errors.New("hunt: a run is already in progress")
	}
	m.runID++
	id := m.runID
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.state = StateRunActive
	m.progress = LiveHuntProgress{Phase: PhaseSweeping}
	m.runSurvey = opts.Survey
	m.sys = nil
	m.survey = nil
	m.reports = nil
	m.err = ""
	m.gainRecs = nil
	m.gainNote = ""
	m.startedAt = time.Now()
	m.finishedAt = time.Time{}
	m.sink = nil
	m.mu.Unlock()

	// Survey persistence + resume: stream classified carriers to a per-band-set
	// NDJSON file, and (when resuming) skip frequencies already recorded there.
	var sink *NDJSONSink
	if opts.PersistSurvey && opts.Survey && m.surveyDir != "" {
		path := filepath.Join(m.surveyDir, "hunt-survey-"+surveyRunKey(opts)+".ndjson")
		if opts.Resume {
			if done, err := LoadSurveyedFreqs(path); err != nil {
				m.log.Warn("hunt: survey resume load failed", "path", path, "err", err)
			} else if len(done) > 0 {
				if opts.SkipFreqs == nil {
					opts.SkipFreqs = done
				} else {
					for f := range done {
						opts.SkipFreqs[f] = struct{}{}
					}
				}
				m.log.Info("hunt: resuming survey", "path", path, "skip", len(done))
			}
		}
		if s, err := OpenNDJSONSink(path); err != nil {
			m.log.Warn("hunt: survey persistence disabled", "path", path, "err", err)
		} else {
			sink = s
			m.mu.Lock()
			m.sink = s
			m.mu.Unlock()
		}
	}

	// Wire progress events onto the bus + the live snapshot.
	opts.Log = m.log
	opts.OnProgress = func(p LiveHuntProgress) {
		m.mu.Lock()
		m.progress = p
		m.mu.Unlock()
		m.publish(events.KindHuntLiveProgress, p)
	}
	// Publish each classified carrier as it is found, and persist any pages the
	// survey decoded onto the same bus the live pager receivers use — so a
	// survey's POCSAG/FLEX catches land in the pager log and stream to clients
	// exactly like a dedicated paging receiver's.
	opts.OnSignal = func(ds DetectedSignal) {
		m.publish(events.KindHuntLiveCandidate, ds)
		if sink != nil {
			if err := sink.Write(ds); err != nil {
				m.log.Warn("hunt: survey NDJSON write failed", "err", err)
			}
		}
		for _, pg := range ds.Pages {
			m.publish(events.KindPagerMessage, storage.PagerMessage{
				ReceivedAt: time.Now(),
				Protocol:   pg.Protocol,
				RIC:        pg.Capcode,
				Encoding:   pg.Encoding,
				Body:       pg.Text,
				Corrected:  pg.Corrected,
			})
		}
	}

	go m.run(ctx, id, opts)
	return id, nil
}

func (m *Manager) run(ctx context.Context, id int, opts LiveHuntOptions) {
	src, release, err := m.acquire(ctx, opts)
	if err != nil {
		m.finish(id, nil, nil, nil, fmt.Errorf("acquire SDR: %w", err))
		return
	}
	defer release()
	opts.Source = src

	var (
		sys     *DiscoveredSystem
		sv      *SignalSurvey
		reports []CaptureReport
		runErr  error
	)
	if opts.Survey {
		sv, reports, runErr = RunLiveSurvey(ctx, opts)
		sys = surveySystem(sv)
	} else {
		sys, reports, runErr = RunLiveHunt(ctx, opts)
	}

	// Optional post-run gain sweep on the locked control channels (advisory).
	if opts.AutoGain && runErr == nil && ctx.Err() == nil {
		m.runAutoGain(ctx, src, opts, lockedControlChannels(sys, sv))
	}

	m.finish(id, sys, sv, reports, runErr)
}

// runAutoGain sweeps front-end gains on the locked control channels and stores
// the recommendations (or a note explaining why none were produced). It needs a
// gain-controllable, exclusively-held SDR; a shared SDR's source reports gain
// control unavailable, which becomes a user-facing note rather than an error.
func (m *Manager) runAutoGain(ctx context.Context, src IQSource, opts LiveHuntOptions, ccs []uint32) {
	note := func(s string) {
		m.mu.Lock()
		m.gainNote = s
		m.mu.Unlock()
	}
	if len(ccs) == 0 {
		note("auto-gain: no locked control channel to sweep")
		return
	}
	gc, ok := src.(GainSettable)
	if !ok {
		note("auto-gain: gain control unavailable on this SDR")
		return
	}
	recs, err := AutoGainSweep(ctx, src, gc, ccs, AutoGainOptions{
		Gains:        opts.AutoGainSet,
		Protocol:     opts.Protocol,
		SampleRateHz: float64(src.SampleRateHz()),
		Log:          m.log,
	})
	if err != nil {
		note(fmt.Sprintf("auto-gain unavailable: %v", err))
		return
	}
	m.mu.Lock()
	m.gainRecs = recs
	m.mu.Unlock()
}

// lockedControlChannels collects the control-channel frequencies a run locked,
// from the discovered system's sites and (for a survey) the trunk-control
// signals — protocol-neutral, the input to the auto-gain sweep.
func lockedControlChannels(sys *DiscoveredSystem, sv *SignalSurvey) []uint32 {
	seen := map[uint32]struct{}{}
	var out []uint32
	add := func(hz uint32) {
		if hz == 0 {
			return
		}
		if _, ok := seen[hz]; !ok {
			seen[hz] = struct{}{}
			out = append(out, hz)
		}
	}
	if sys != nil {
		for _, st := range sys.Sites {
			for _, ch := range st.ControlChannels {
				if ch.IsControl {
					add(ch.FrequencyHz)
				}
			}
		}
	}
	if sv != nil {
		for _, s := range sv.Signals {
			if s.Class == survey.ClassTrunkControl {
				add(s.FreqHz)
			}
		}
	}
	return out
}

// surveySystem safely extracts the discovered system from a (possibly nil)
// survey, so the export/commit tail sees the same *DiscoveredSystem a hunt run
// produces.
func surveySystem(sv *SignalSurvey) *DiscoveredSystem {
	if sv == nil {
		return nil
	}
	return sv.System
}

// finish records the terminal state of a run and publishes hunt.done.
func (m *Manager) finish(id int, sys *DiscoveredSystem, sv *SignalSurvey, reports []CaptureReport, err error) {
	m.mu.Lock()
	if id != m.runID {
		m.mu.Unlock()
		return // a newer run superseded this one
	}
	m.finishedAt = time.Now()
	m.cancel = nil
	if m.sink != nil {
		_ = m.sink.Close()
		m.sink = nil
	}
	switch {
	case err != nil && errors.Is(err, context.Canceled):
		m.state = StateRunStopped
	case err != nil:
		m.state = StateRunFailed
		m.err = err.Error()
	default:
		m.state = StateRunDone
	}
	m.survey = sv
	if sys != nil || sv != nil {
		m.sys = sys
		m.reports = reports
		// Retain in the bounded history (newest last; drop oldest over cap).
		m.history = append(m.history, runRecord{id: id, sys: sys, survey: sv, reports: reports})
		if len(m.history) > maxRunHistory {
			m.history = m.history[len(m.history)-maxRunHistory:]
		}
	}
	st := m.statusLocked()
	m.mu.Unlock()
	m.publish(events.KindHuntLiveDone, st)
}

// Stop cancels the active run. Returns false if no run is active.
func (m *Manager) Stop() bool {
	m.mu.Lock()
	cancel := m.cancel
	active := m.state == StateRunActive
	m.mu.Unlock()
	if !active || cancel == nil {
		return false
	}
	cancel()
	return true
}

// Status returns the current run snapshot.
func (m *Manager) Status() RunStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.statusLocked()
}

func (m *Manager) statusLocked() RunStatus {
	st := RunStatus{
		RunID:      m.runID,
		State:      m.state,
		Running:    m.state == StateRunActive,
		Mode:       "hunt",
		Progress:   m.progress,
		Error:      m.err,
		StartedAt:  m.startedAt,
		FinishedAt: m.finishedAt,
	}
	if m.sys != nil {
		st.Sites = len(m.sys.Sites)
		st.Talkgroups = len(m.sys.Talkgroups)
		st.SystemName = m.sys.DisplayName()
	}
	if m.runSurvey {
		st.Mode = "survey"
	}
	if m.survey != nil {
		st.Signals = m.survey.Signals
	}
	st.GainRecommendations = m.gainRecs
	st.GainNote = m.gainNote
	return st
}

// CurrentSurvey returns the latest run's signal inventory, or (nil, false) when
// the latest run was a plain hunt or none has produced one yet.
func (m *Manager) CurrentSurvey() (*SignalSurvey, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.survey == nil {
		return nil, false
	}
	return m.survey, true
}

// SurveyRun returns a specific run's signal inventory (id 0 = latest). ok=false
// for an unknown/evicted id or a run that produced no survey (a plain hunt).
func (m *Manager) SurveyRun(id int) (*SignalSurvey, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id == 0 || id == m.runID {
		if m.survey == nil {
			return nil, false
		}
		return m.survey, true
	}
	for i := len(m.history) - 1; i >= 0; i-- {
		if m.history[i].id == id {
			return m.history[i].survey, m.history[i].survey != nil
		}
	}
	return nil, false
}

// ExportSurvey writes a run's signal inventory to w in the given format (id 0 =
// latest). Returns ErrNoSuchRun for an unknown/evicted id, or a "no survey"
// error when the run exists but was a plain hunt.
func (m *Manager) ExportSurvey(id int, w io.Writer, f SurveyFormat) error {
	sv, ok := m.SurveyRun(id)
	if !ok {
		if id != 0 && !m.KnownRun(id) {
			return ErrNoSuchRun
		}
		return errors.New("hunt: no signal survey for this run")
	}
	return WriteSurvey(w, sv, f)
}

// Current returns the latest discovered system and its per-candidate reports,
// or (nil, nil, false) when no run has produced a map yet. The returned system
// is the live pointer; callers must not mutate it.
func (m *Manager) Current() (*DiscoveredSystem, []CaptureReport, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sys == nil {
		return nil, nil, false
	}
	return m.sys, m.reports, true
}

// Run returns a specific run's discovered system + reports. id 0 means the
// latest (same as Current). Returns ok=false for an unknown/evicted id, or for
// a run that produced no system.
func (m *Manager) Run(id int) (*DiscoveredSystem, []CaptureReport, bool) {
	if id == 0 {
		return m.Current()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	// The live current run may not be in history yet (still in flight, or it's
	// the most recent finished one) — check it first.
	if id == m.runID && m.sys != nil {
		return m.sys, m.reports, true
	}
	for i := len(m.history) - 1; i >= 0; i-- {
		if m.history[i].id == id {
			return m.history[i].sys, m.history[i].reports, true
		}
	}
	return nil, nil, false
}

// KnownRun reports whether id refers to any run the Manager has seen (active or
// in history), so callers can distinguish "unknown id" (404) from "run exists
// but produced nothing" (409).
func (m *Manager) KnownRun(id int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id == 0 || id == m.runID {
		return m.runID > 0
	}
	for _, r := range m.history {
		if r.id == id {
			return true
		}
	}
	return false
}

// Export writes the latest discovered system to w in the given format. Returns
// an error when no system has been discovered yet.
func (m *Manager) Export(w io.Writer, f Format, hints []DuplicateHint) error {
	return m.ExportRun(0, w, f, hints)
}

// ExportRun writes a specific run's discovered system (id 0 = latest). Returns
// ErrNoSuchRun for an unknown/evicted id, or a "no system" error when the run
// exists but produced nothing.
func (m *Manager) ExportRun(id int, w io.Writer, f Format, hints []DuplicateHint) error {
	sys, _, ok := m.Run(id)
	if !ok {
		if id != 0 && !m.KnownRun(id) {
			return ErrNoSuchRun
		}
		return errors.New("hunt: no discovered system to export yet")
	}
	return Write(w, sys, f, hints)
}

func (m *Manager) publish(kind events.Kind, payload any) {
	if m.bus == nil {
		return
	}
	m.bus.Publish(events.Event{Kind: kind, Timestamp: time.Now(), Payload: payload})
}
