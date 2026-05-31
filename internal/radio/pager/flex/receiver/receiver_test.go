package receiver

import (
	"context"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

func TestReceiverNewRejectsBadOptions(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()

	if _, err := New(Options{}); err == nil {
		t.Error("New without Bus: want error")
	}
	if _, err := New(Options{Bus: bus}); err == nil {
		t.Error("New without InputRateHz: want error")
	}
}

func TestReceiverNewSetsUpDecoder(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	r, err := New(Options{InputRateHz: 128_000, Bus: bus})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if r.decoder == nil {
		t.Error("decoder = nil, want non-nil")
	}
}

func TestReceiverPropagatesContextCancel(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	r, err := New(Options{InputRateHz: 128_000, Bus: bus})
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

func TestReceiverNilInputErrors(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	r, _ := New(Options{InputRateHz: 128_000, Bus: bus})
	if err := r.Process(context.Background(), nil); err == nil {
		t.Error("Process with nil input: want error")
	}
}
