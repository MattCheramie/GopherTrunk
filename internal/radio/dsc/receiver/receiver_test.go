package receiver

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dsc"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

// pushChar shifts one 10-bit DSC character (BCH-encoded from a 7-bit
// symbol) MSB-first into the receiver.
func pushChar(r *Receiver, sym byte) {
	cw := dsc.BCHEncode(uint16(sym))
	for i := 9; i >= 0; i-- {
		r.Push(byte((cw >> uint(i)) & 1))
	}
}

// pushSequence drives a full DSC sequence into the receiver: a phasing
// run (DX = 125 interleaved with RX placeholders on the 10-bit grid),
// then the message symbols transmitted DX-then-RX so the DX grid (every
// 20 bits) carries the data. This mirrors the "drop RX, use DX only"
// path the receiver decodes.
func pushSequence(r *Receiver, syms []byte) {
	// Phasing: 8 DX/RX pairs. RX placeholders count down from 111 — the
	// receiver samples only the DX grid, so their value is immaterial,
	// but using valid characters keeps the stream realistic.
	for i := 0; i < 8; i++ {
		pushChar(r, phasingDX) // DX
		pushChar(r, byte(111-i))
	}
	// Message: each data symbol on the DX grid, its RX twin following.
	for _, s := range syms {
		pushChar(r, s) // DX
		pushChar(r, s) // RX (dropped)
	}
}

func waitForDSC(t *testing.T, sub *events.Subscription) storage.DSCMessage {
	t.Helper()
	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindDSCMessage {
				continue
			}
			m, ok := ev.Payload.(storage.DSCMessage)
			if !ok {
				t.Fatalf("payload type = %T, want storage.DSCMessage", ev.Payload)
			}
			return m
		case <-deadline:
			t.Fatal("timed out waiting for KindDSCMessage")
			return storage.DSCMessage{}
		}
	}
}

func TestNewRequiresBus(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("New(nil bus): want panic")
		}
	}()
	_ = New(Options{})
}

func TestDecodeDistressSequence(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	r := New(Options{Bus: bus})

	// Distress alert mirroring dsc_test.go's TestDecodeDistressMessage:
	// format=112, self-MMSI 366053209, nature=fire, position, time,
	// EOS=127.
	syms := []byte{112, 36, 60, 53, 20, 90, 100, 3, 74, 81, 22, 24, 14, 25, 127}
	pushSequence(r, syms)

	m := waitForDSC(t, sub)
	if m.Format != "distress" {
		t.Errorf("Format = %q, want distress", m.Format)
	}
	if m.SelfMMSI != 366053209 {
		t.Errorf("SelfMMSI = %d, want 366053209", m.SelfMMSI)
	}
	if m.Nature != "fire / explosion" {
		t.Errorf("Nature = %q, want fire / explosion", m.Nature)
	}
	if !m.HasPosition {
		t.Error("HasPosition = false, want true")
	}
	if m.TimeUTC != "14:25" {
		t.Errorf("TimeUTC = %q, want 14:25", m.TimeUTC)
	}
	if got := r.Stats().SequencesEmit; got != 1 {
		t.Errorf("SequencesEmit = %d, want 1", got)
	}
}

func TestDecodeIndividualCall(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	r := New(Options{Bus: bus})

	// format=120 individual, target 003660000, category routine,
	// self-MMSI 366053209, EOS=127.
	syms := []byte{120, 0, 36, 60, 0, 0, 100, 36, 60, 53, 20, 90, 127}
	pushSequence(r, syms)

	m := waitForDSC(t, sub)
	if m.Format != "individual" {
		t.Errorf("Format = %q, want individual", m.Format)
	}
	if m.Category != "routine" {
		t.Errorf("Category = %q, want routine", m.Category)
	}
	if m.TargetMMSI != 3660000 {
		t.Errorf("TargetMMSI = %d, want 3660000", m.TargetMMSI)
	}
	if m.SelfMMSI != 366053209 {
		t.Errorf("SelfMMSI = %d, want 366053209", m.SelfMMSI)
	}
}

// TestDecodeInvertedPolarity feeds the whole sequence with every bit
// complemented; the dual-polarity phasing hunt must still lock and
// recover the true symbols.
func TestDecodeInvertedPolarity(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	r := New(Options{Bus: bus})

	syms := []byte{116, 0, 0, 0, 0, 0, 108, 0, 36, 69, 99, 90, 127}
	// Re-implement pushSequence with inverted bits.
	pushCharInv := func(sym byte) {
		cw := dsc.BCHEncode(uint16(sym))
		for i := 9; i >= 0; i-- {
			r.Push(byte((^(cw >> uint(i))) & 1))
		}
	}
	for i := 0; i < 8; i++ {
		pushCharInv(phasingDX)
		pushCharInv(byte(111 - i))
	}
	for _, s := range syms {
		pushCharInv(s)
		pushCharInv(s)
	}

	m := waitForDSC(t, sub)
	if m.Format != "all-ships" {
		t.Errorf("Format = %q, want all-ships", m.Format)
	}
	if m.Category != "safety" {
		t.Errorf("Category = %q, want safety", m.Category)
	}
	if m.SelfMMSI != 3669999 {
		t.Errorf("SelfMMSI = %d, want 3669999", m.SelfMMSI)
	}
}

// TestNoEOSDoesNotPublish confirms a sequence that never reaches an
// end-of-sequence character is abandoned (no event, no leak).
func TestNoEOSDoesNotPublish(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	r := New(Options{Bus: bus})

	// Phasing then a long run of non-EOS symbols.
	for i := 0; i < 8; i++ {
		pushChar(r, phasingDX)
		pushChar(r, byte(111-i))
	}
	for i := 0; i < maxSeqSyms+5; i++ {
		pushChar(r, 50) // arbitrary non-EOS symbol on the DX grid
		pushChar(r, 50)
	}

	select {
	case ev := <-sub.C:
		if ev.Kind == events.KindDSCMessage {
			t.Error("published a DSC message for a sequence with no EOS")
		}
	case <-time.After(100 * time.Millisecond):
		// expected: nothing published
	}
	if got := r.Stats().SequencesEmit; got != 0 {
		t.Errorf("SequencesEmit = %d, want 0", got)
	}
}
