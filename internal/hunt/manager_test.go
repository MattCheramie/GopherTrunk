package hunt

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
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

func TestManager_StartRunsAndMapsSystem(t *testing.T) {
	bus := events.NewBus(256)
	sub := bus.Subscribe()
	defer sub.Close()

	src, dwell := p25Source(t)
	mgr, err := NewManager(ManagerOptions{
		Acquire: func(context.Context) (IQSource, func(), error) { return src, func() {}, nil },
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
		Acquire: func(context.Context) (IQSource, func(), error) {
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
		Acquire: func(context.Context) (IQSource, func(), error) { return nil, nil, errors.New("x") },
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

// release must run on completion so a borrowed SDR is always returned.
func TestManager_ReleaseRunsOnCompletion(t *testing.T) {
	src, dwell := p25Source(t)
	released := make(chan struct{})
	mgr, _ := NewManager(ManagerOptions{
		Acquire: func(context.Context) (IQSource, func(), error) {
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
