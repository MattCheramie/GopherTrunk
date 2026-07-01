package hunt

import (
	"context"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

func TestManagerCaptureSignal(t *testing.T) {
	const rate = 2_048_000
	src := NewFileIQSource(make([]complex64, rate), rate) // 1 s of zero IQ, cycles
	released := false
	mgr, err := NewManager(ManagerOptions{
		Acquire: func(context.Context, LiveHuntOptions) (IQSource, func(), error) {
			return src, func() { released = true }, nil
		},
		Bus: events.NewBus(256),
	})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	iq, gotRate, err := mgr.CaptureSignal(context.Background(), 460_000_000, 0.1)
	if err != nil {
		t.Fatalf("CaptureSignal: %v", err)
	}
	if gotRate != rate {
		t.Errorf("rate = %d, want %d", gotRate, rate)
	}
	if want := int(0.1 * rate); len(iq) != want {
		t.Errorf("captured %d samples, want %d", len(iq), want)
	}
	if !released {
		t.Error("the acquired SDR was not released")
	}
	// The source was tuned to the requested frequency.
	if calls := src.TuneCalls(); len(calls) == 0 || calls[len(calls)-1] != 460_000_000 {
		t.Errorf("tune calls = %v, want last = 460_000_000", calls)
	}
}

func TestManagerCaptureSignalRefusesDuringRun(t *testing.T) {
	const rate = 2_048_000
	src := NewFileIQSource(make([]complex64, rate), rate)
	mgr, err := NewManager(ManagerOptions{
		Acquire: func(context.Context, LiveHuntOptions) (IQSource, func(), error) { return src, func() {}, nil },
		Bus:     events.NewBus(256),
	})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	// Force the "running" state to model an active sweep.
	mgr.mu.Lock()
	mgr.state = StateRunActive
	mgr.mu.Unlock()

	if _, _, err := mgr.CaptureSignal(context.Background(), 460_000_000, 0.1); err == nil {
		t.Fatal("CaptureSignal during an active run: want a busy error, got nil")
	}
}
