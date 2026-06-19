package autotune

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestAverageAndAccumulation(t *testing.T) {
	m := NewManager("test", nil)

	// observed + currentOffset is the stored total.
	m.AddErrorMeasurement(100, 0, 851_000_000) // total 100
	if got := m.AverageError(); got != 100 {
		t.Fatalf("avg after one: got %d want 100", got)
	}
	m.AddErrorMeasurement(200, 0, 851_000_000) // totals {200,100} -> 150
	if got := m.AverageError(); got != 150 {
		t.Fatalf("avg after two: got %d want 150", got)
	}
	// A measurement taken while +150 was already applied: true error 350.
	m.AddErrorMeasurement(200, 150, 851_000_000) // totals {350,200,100} -> 216
	if got := m.AverageError(); got != 216 {
		t.Fatalf("avg with applied offset: got %d want 216", got)
	}
}

func TestHistoryCappedAtMax(t *testing.T) {
	m := NewManager("test", nil)
	// Push MaxHistory zeros, then MaxHistory more of 1000. The oldest zeros
	// must be evicted, leaving the average at 1000.
	for i := 0; i < MaxHistory; i++ {
		m.AddErrorMeasurement(0, 0, 0)
	}
	for i := 0; i < MaxHistory; i++ {
		m.AddErrorMeasurement(1000, 0, 0)
	}
	if got := m.AverageError(); got != 1000 {
		t.Fatalf("avg should have evicted all zeros: got %d want 1000", got)
	}
	if len(m.history) != MaxHistory {
		t.Fatalf("history len: got %d want %d", len(m.history), MaxHistory)
	}
}

func TestReset(t *testing.T) {
	m := NewManager("test", nil)
	m.AddErrorMeasurement(500, 0, 0)
	m.Reset()
	if len(m.history) != 0 {
		t.Fatalf("history not cleared: %v", m.history)
	}
	// Next measurement starts a fresh average, not blended with the old one.
	m.AddErrorMeasurement(20, 0, 0)
	if got := m.AverageError(); got != 20 {
		t.Fatalf("avg after reset: got %d want 20", got)
	}
}

func TestStatusStringSuggestedRounding(t *testing.T) {
	m := NewManager("test", nil)
	m.AddErrorMeasurement(123, 0, 851_000_000) // avg 123

	// initial error 0, total = 0 - 123 = -123, rounded to nearest 10 -> -120.
	s := m.StatusString(851_000_000, 0)
	if !strings.Contains(s, "AutoTune: +123 Hz") {
		t.Errorf("status missing correction: %q", s)
	}
	if !strings.Contains(s, "suggested error: -120 Hz") {
		t.Errorf("status suggested value wrong: %q", s)
	}
	if !strings.Contains(s, "ppm") {
		t.Errorf("status missing ppm hint: %q", s)
	}
}

func TestPPMThresholdWarning(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	m := NewManager("dongleA", log)

	// 3.5 PPM at 100 MHz is 350 Hz. 1000 Hz is ~10 PPM -> must warn.
	m.AddErrorMeasurement(1000, 0, 100_000_000)
	if !strings.Contains(buf.String(), "exceeds PPM threshold") {
		t.Fatalf("expected PPM warning, got: %q", buf.String())
	}

	// A small error well under threshold must not warn.
	buf.Reset()
	m2 := NewManager("dongleB", log)
	m2.AddErrorMeasurement(50, 0, 100_000_000) // 0.5 PPM
	if strings.Contains(buf.String(), "exceeds PPM threshold") {
		t.Fatalf("unexpected PPM warning: %q", buf.String())
	}
}

func TestRegistryDisabledReturnsNil(t *testing.T) {
	r := NewRegistry(false, nil)
	if r.Enabled() {
		t.Fatal("registry should be disabled")
	}
	if r.Get("abc") != nil {
		t.Fatal("disabled registry must return nil manager")
	}
	// nil registry is also safe.
	var nilReg *Registry
	if nilReg.Enabled() || nilReg.Get("x") != nil {
		t.Fatal("nil registry must be safe and return nil")
	}
}

func TestRegistrySharesPerSerial(t *testing.T) {
	r := NewRegistry(true, nil)
	a1 := r.Get("serial-A")
	a2 := r.Get("serial-A")
	b := r.Get("serial-B")
	if a1 == nil || a1 != a2 {
		t.Fatal("same serial must return the same Manager")
	}
	if b == a1 {
		t.Fatal("different serials must return different Managers")
	}
}
