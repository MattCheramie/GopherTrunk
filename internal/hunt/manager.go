package hunt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

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
}

// Manager owns the daemon's single live-hunt run: it acquires an SDR, runs the
// sweep→identify→map pipeline in the background, publishes hunt.* bus events,
// and holds the latest discovered system for export/commit. Safe for
// concurrent use by the REST/TUI cockpit.
type Manager struct {
	acquire Acquirer
	bus     *events.Bus
	log     *slog.Logger

	mu         sync.Mutex
	runID      int
	state      RunState
	runSurvey  bool // the active/last run is a survey (drives RunStatus.Mode)
	progress   LiveHuntProgress
	sys        *DiscoveredSystem
	survey     *SignalSurvey
	reports    []CaptureReport
	err        string
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
	return &Manager{acquire: opts.Acquire, bus: opts.Bus, log: log, state: StateRunIdle}, nil
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
	m.startedAt = time.Now()
	m.finishedAt = time.Time{}
	m.mu.Unlock()

	// Wire progress events onto the bus + the live snapshot.
	opts.Log = m.log
	opts.OnProgress = func(p LiveHuntProgress) {
		m.mu.Lock()
		m.progress = p
		m.mu.Unlock()
		m.publish(events.KindHuntLiveProgress, p)
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

	if opts.Survey {
		sv, reports, serr := RunLiveSurvey(ctx, opts)
		m.finish(id, surveySystem(sv), sv, reports, serr)
		return
	}
	sys, reports, err := RunLiveHunt(ctx, opts)
	m.finish(id, sys, nil, reports, err)
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
