package hunt

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/survey"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// p25Source builds a FileIQSource over a synthesized P25 control channel plus
// the run dwell that captures the whole buffer once.
func p25Source(t *testing.T) (*FileIQSource, float64) {
	t.Helper()
	iq, meta, err := siglab.Synthesize(siglab.SynthOptions{
		Protocol: trunking.ProtocolP25,
		Format:   siglab.FormatF32,
	})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	rate := uint32(meta.SampleRateHz)
	return NewFileIQSource(iq, rate), float64(len(iq)) / float64(rate)
}

func waitUntil(t *testing.T, d time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met within deadline")
}

func TestManager_SurveyModeReportsSignals(t *testing.T) {
	bus := events.NewBus(256)
	sub := bus.Subscribe()
	defer sub.Close()

	src, dwell := p25Source(t)
	mgr, err := NewManager(ManagerOptions{
		Acquire: func(context.Context, LiveHuntOptions) (IQSource, func(), error) { return src, func() {}, nil },
		Bus:     bus,
	})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	if _, err := mgr.Start(LiveHuntOptions{
		Survey:        true,
		Candidates:    []uint32{851_000_000},
		DwellSeconds:  dwell,
		MinConfidence: 0.3,
	}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitUntil(t, 15*time.Second, func() bool { return !mgr.Status().Running })

	st := mgr.Status()
	if st.State != StateRunDone {
		t.Fatalf("state = %q, want done", st.State)
	}
	if st.Mode != "survey" {
		t.Errorf("mode = %q, want survey", st.Mode)
	}
	if len(st.Signals) != 1 {
		t.Fatalf("len(Signals) = %d, want 1", len(st.Signals))
	}
	// The single P25 candidate should be mapped as a trunk-control carrier.
	if st.Signals[0].Class != survey.ClassTrunkControl {
		t.Errorf("signal class = %q, want trunk-control", st.Signals[0].Class)
	}
	if sv, ok := mgr.CurrentSurvey(); !ok || sv.System == nil {
		t.Error("CurrentSurvey() returned no survey/system")
	}

	// A per-signal candidate event should have been published to the bus.
	sawCandidate := false
	for drain := true; drain; {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindHuntLiveCandidate {
				sawCandidate = true
			}
		default:
			drain = false
		}
	}
	if !sawCandidate {
		t.Error("expected a KindHuntLiveCandidate event on the bus")
	}
}

// TestManager_SurveyPersistAndResume runs a survey with PersistSurvey into a
// SurveyDir, confirms an NDJSON file is written, then runs again with Resume and
// confirms the already-surveyed candidate is skipped.
func TestManager_SurveyPersistAndResume(t *testing.T) {
	dir := t.TempDir()
	src, dwell := p25Source(t)
	mgr, err := NewManager(ManagerOptions{
		Acquire:   func(context.Context, LiveHuntOptions) (IQSource, func(), error) { return src, func() {}, nil },
		Bus:       events.NewBus(256),
		SurveyDir: dir,
	})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	opts := LiveHuntOptions{
		Survey: true, PersistSurvey: true,
		Candidates: []uint32{851_000_000}, DwellSeconds: dwell, MinConfidence: 0.3,
	}
	if _, err := mgr.Start(opts); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitUntil(t, 15*time.Second, func() bool { return !mgr.Status().Running })

	files, _ := filepath.Glob(filepath.Join(dir, "hunt-survey-*.ndjson"))
	if len(files) != 1 {
		t.Fatalf("ndjson files = %v, want exactly 1", files)
	}
	done, err := LoadSurveyedFreqs(files[0])
	if err != nil {
		t.Fatalf("LoadSurveyedFreqs: %v", err)
	}
	if _, ok := done[851_000_000]; !ok {
		t.Fatalf("surveyed freqs = %v, want 851000000 recorded", done)
	}

	// Resume: the same band/candidate set keys the same file; the lone candidate
	// is already surveyed, so the run skips it and reports no signals.
	src2, _ := p25Source(t)
	mgr.acquire = func(context.Context, LiveHuntOptions) (IQSource, func(), error) { return src2, func() {}, nil }
	opts.Resume = true
	if _, err := mgr.Start(opts); err != nil {
		t.Fatalf("Start resume: %v", err)
	}
	waitUntil(t, 15*time.Second, func() bool { return !mgr.Status().Running })
	if n := len(mgr.Status().Signals); n != 0 {
		t.Errorf("resumed run surveyed %d signal(s), want 0 (candidate already done)", n)
	}
}

// TestManager_AutoGainUnavailableNote: with AutoGain on a source that can't set
// gain (the file source / a borrowed shared SDR), the run completes and records
// a user-facing note rather than failing.
func TestManager_AutoGainUnavailableNote(t *testing.T) {
	src, dwell := p25Source(t)
	mgr, err := NewManager(ManagerOptions{
		Acquire: func(context.Context, LiveHuntOptions) (IQSource, func(), error) { return src, func() {}, nil },
		Bus:     events.NewBus(256),
	})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if _, err := mgr.Start(LiveHuntOptions{
		Survey: true, AutoGain: true,
		Candidates: []uint32{851_000_000}, DwellSeconds: dwell, MinConfidence: 0.3,
	}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitUntil(t, 15*time.Second, func() bool { return !mgr.Status().Running })
	st := mgr.Status()
	if st.State != StateRunDone {
		t.Fatalf("state = %q, want done", st.State)
	}
	if st.GainNote == "" {
		t.Error("expected a gain note when gain control is unavailable")
	}
}

func TestManager_StartRunsAndMapsSystem(t *testing.T) {
	bus := events.NewBus(256)
	sub := bus.Subscribe()
	defer sub.Close()

	src, dwell := p25Source(t)
	mgr, err := NewManager(ManagerOptions{
		Acquire: func(context.Context, LiveHuntOptions) (IQSource, func(), error) { return src, func() {}, nil },
		Bus:     bus,
	})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	id, err := mgr.Start(LiveHuntOptions{
		Candidates:    []uint32{851_000_000},
		DwellSeconds:  dwell,
		MinConfidence: 0.3,
		Name:          "Daemon Hunt",
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if id != 1 {
		t.Errorf("run id = %d, want 1", id)
	}

	// Starting again while active must be refused.
	if _, err := mgr.Start(LiveHuntOptions{Candidates: []uint32{1}}); err == nil {
		t.Error("expected error starting a second run while active")
	}

	waitUntil(t, 15*time.Second, func() bool { return !mgr.Status().Running })

	st := mgr.Status()
	if st.State != StateRunDone {
		t.Fatalf("state = %q, want done", st.State)
	}
	if st.Sites < 1 {
		t.Errorf("sites = %d, want >= 1", st.Sites)
	}
	sys, _, ok := mgr.Current()
	if !ok || sys == nil {
		t.Fatal("Current() returned no system")
	}

	// Bus carried progress + done events.
	var sawProgress, sawDone bool
	for {
		select {
		case ev := <-sub.C:
			switch ev.Kind {
			case events.KindHuntLiveProgress:
				sawProgress = true
			case events.KindHuntLiveDone:
				sawDone = true
			}
		default:
			if !sawProgress {
				t.Error("no hunt.progress event observed")
			}
			if !sawDone {
				t.Error("no hunt.done event observed")
			}
			return
		}
	}
}

func TestManager_AcquireErrorFailsRun(t *testing.T) {
	mgr, err := NewManager(ManagerOptions{
		Acquire: func(context.Context, LiveHuntOptions) (IQSource, func(), error) {
			return nil, nil, errors.New("no spare SDR")
		},
	})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if _, err := mgr.Start(LiveHuntOptions{Candidates: []uint32{1}}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitUntil(t, 5*time.Second, func() bool { return !mgr.Status().Running })
	st := mgr.Status()
	if st.State != StateRunFailed {
		t.Errorf("state = %q, want failed", st.State)
	}
	if st.Error == "" {
		t.Error("expected an error message on a failed run")
	}
}

func TestManager_StopIdleReturnsFalse(t *testing.T) {
	mgr, _ := NewManager(ManagerOptions{
		Acquire: func(context.Context, LiveHuntOptions) (IQSource, func(), error) { return nil, nil, errors.New("x") },
	})
	if mgr.Stop() {
		t.Error("Stop() on an idle manager should return false")
	}
}

func TestNewManager_RequiresAcquirer(t *testing.T) {
	if _, err := NewManager(ManagerOptions{}); err == nil {
		t.Error("expected error when Acquire is nil")
	}
}

// TestManager_PassesOptionsToAcquirer confirms the run's options (notably the
// requested SDR serial) reach the Acquirer, so the daemon can honor it.
func TestManager_PassesOptionsToAcquirer(t *testing.T) {
	src, dwell := p25Source(t)
	gotSerial := make(chan string, 1)
	mgr, _ := NewManager(ManagerOptions{
		Acquire: func(_ context.Context, opts LiveHuntOptions) (IQSource, func(), error) {
			gotSerial <- opts.Serial
			return src, func() {}, nil
		},
	})
	if _, err := mgr.Start(LiveHuntOptions{
		Candidates: []uint32{851_000_000}, DwellSeconds: dwell, MinConfidence: 0.3, Serial: "dongle-7",
	}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	select {
	case s := <-gotSerial:
		if s != "dongle-7" {
			t.Errorf("Acquirer saw serial %q, want dongle-7", s)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Acquirer was not called")
	}
}

// TestManager_RunHistoryAndExportRun covers the bounded run history: a finished
// run is reachable by id, an unknown id yields ErrNoSuchRun, and the oldest is
// evicted past the cap.
func TestManager_RunHistoryAndExportRun(t *testing.T) {
	src, dwell := p25Source(t)
	mgr, _ := NewManager(ManagerOptions{
		Acquire: func(context.Context, LiveHuntOptions) (IQSource, func(), error) {
			return src, func() {}, nil
		},
	})
	runOnce := func() int {
		id, err := mgr.Start(LiveHuntOptions{Candidates: []uint32{851_000_000}, DwellSeconds: dwell, MinConfidence: 0.3})
		if err != nil {
			t.Fatalf("Start: %v", err)
		}
		waitUntil(t, 15*time.Second, func() bool { return !mgr.Status().Running })
		return id
	}

	first := runOnce()
	if _, _, ok := mgr.Run(first); !ok {
		t.Fatalf("run %d not in history", first)
	}
	// Unknown id → ExportRun returns ErrNoSuchRun.
	if err := mgr.ExportRun(99999, io.Discard, FormatBundle, nil); !errors.Is(err, ErrNoSuchRun) {
		t.Errorf("ExportRun(unknown) = %v, want ErrNoSuchRun", err)
	}
	if err := mgr.ExportRun(first, io.Discard, FormatBundle, nil); err != nil {
		t.Errorf("ExportRun(first) = %v, want nil", err)
	}

	// Exceed the cap → the first run is evicted.
	for i := 0; i < maxRunHistory; i++ {
		runOnce()
	}
	if mgr.KnownRun(first) {
		t.Errorf("run %d should have been evicted past cap %d", first, maxRunHistory)
	}
}

// release must run on completion so a borrowed SDR is always returned.
func TestManager_ReleaseRunsOnCompletion(t *testing.T) {
	src, dwell := p25Source(t)
	released := make(chan struct{})
	mgr, _ := NewManager(ManagerOptions{
		Acquire: func(context.Context, LiveHuntOptions) (IQSource, func(), error) {
			return src, func() { close(released) }, nil
		},
	})
	if _, err := mgr.Start(LiveHuntOptions{Candidates: []uint32{851_000_000}, DwellSeconds: dwell, MinConfidence: 0.3}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	select {
	case <-released:
	case <-time.After(15 * time.Second):
		t.Fatal("release was not called after the run finished")
	}
}
