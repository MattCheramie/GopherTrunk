package ffsk

import (
	"context"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dsc"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
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

func TestNewSetsUpInner(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	r, err := New(Options{InputRateHz: 96_000, Bus: bus})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if r.Inner() == nil {
		t.Error("Inner() = nil, want non-nil orchestrator")
	}
}

func TestProcessPropagatesContextCancel(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	r, err := New(Options{InputRateHz: 96_000, Bus: bus})
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
	r, _ := New(Options{InputRateHz: 96_000, Bus: bus})
	if err := r.Process(context.Background(), nil); err == nil {
		t.Error("Process with nil input: want error")
	}
}

// charBits / phasingDX are duplicated from the receiver package's
// unexported constants for the test's bit-builder.
const (
	charBits  = 10
	phasingDX = 125
)

// appendChar appends one 10-bit DSC character (BCH-encoded from a
// 7-bit symbol), MSB-first, as 0/1 bytes.
func appendChar(bits []byte, sym byte) []byte {
	cw := dsc.BCHEncode(uint16(sym))
	for i := charBits - 1; i >= 0; i-- {
		bits = append(bits, byte((cw>>uint(i))&1))
	}
	return bits
}

// buildWireBits assembles a complete DSC bit stream: a dotting
// preamble (for symbol-timing lock), a phasing run, then the message
// symbols transmitted DX-then-RX on the 10-bit grid.
func buildWireBits(syms []byte) []byte {
	var bits []byte
	// Dotting: alternating bits give the Mueller-Müller loop and the
	// FFSK discriminator time to settle before the phasing run.
	for i := 0; i < 240; i++ {
		bits = append(bits, byte(i&1))
	}
	for i := 0; i < 12; i++ {
		bits = appendChar(bits, phasingDX)       // DX
		bits = appendChar(bits, byte(111-(i%8))) // RX placeholder
	}
	for _, s := range syms {
		bits = appendChar(bits, s) // DX
		bits = appendChar(bits, s) // RX twin
	}
	return bits
}

// TestEndToEndDistressDecode modulates a DSC distress sequence to an
// IQ stream, runs it through the full frontend, and asserts the
// decoded message lands on the bus. This validates the FM → resample
// → FFSK → symbol-timing → slicer → BCH-sync → parser chain together.
func TestEndToEndDistressDecode(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	const inputRate = 96_000
	r, err := New(Options{InputRateHz: inputRate, Bus: bus})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	syms := []byte{112, 36, 60, 53, 20, 90, 100, 3, 74, 81, 22, 24, 14, 25, 127}
	bits := buildWireBits(syms)
	iq := demod.ModulateFFSK(bits, float64(inputRate), float64(BaudHz), MarkHz, SpaceHz)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	in := make(chan []complex64, 1)
	done := make(chan struct{})
	go func() {
		_ = r.Process(ctx, in)
		close(done)
	}()
	in <- iq
	close(in)
	<-done

	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindDSCMessage {
				continue
			}
			m := ev.Payload.(storage.DSCMessage)
			if m.Format != "distress" {
				t.Errorf("Format = %q, want distress", m.Format)
			}
			if m.SelfMMSI != 366053209 {
				t.Errorf("SelfMMSI = %d, want 366053209", m.SelfMMSI)
			}
			if !m.HasPosition {
				t.Error("HasPosition = false, want true")
			}
			return
		case <-deadline:
			t.Fatalf("no DSC message decoded; frontend stats = %+v, inner = %+v",
				r.Stats(), r.Inner().Stats())
		}
	}
}
