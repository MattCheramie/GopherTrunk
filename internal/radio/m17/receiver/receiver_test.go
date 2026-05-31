package receiver

import (
	"context"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

func TestNewRejectsBadOptions(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()

	if _, err := New(Options{}); err == nil {
		t.Error("New without Bus: want error")
	}
	if _, err := New(Options{Bus: bus}); err == nil {
		t.Error("New without InputRateHz: want error")
	}
}

func TestNewSetsUpDecoder(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	r, err := New(Options{InputRateHz: 480_000, Bus: bus})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if r.decoder == nil {
		t.Error("decoder = nil, want non-nil")
	}
}

func TestProcessPropagatesContextCancel(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	r, err := New(Options{InputRateHz: 480_000, Bus: bus})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	in := make(chan []complex64)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.Process(ctx, in) }()
	cancel()
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("Process err = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Process did not exit after ctx cancel")
	}
}

func TestProcessNilInputErrors(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	r, _ := New(Options{InputRateHz: 480_000, Bus: bus})
	if err := r.Process(context.Background(), nil); err == nil {
		t.Error("Process with nil input: want error")
	}
}

func TestSymbolToDibit(t *testing.T) {
	cases := map[int]int{1: 0, 3: 1, -1: 2, -3: 3}
	for level, want := range cases {
		if got := symbolToDibit(level); got != want {
			t.Errorf("symbolToDibit(%d) = %d, want %d", level, got, want)
		}
	}
}
