package log

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

// recoverPanicHarness runs fn under `defer Recover(...)` and reports whether
// the goroutine returned normally (panic contained) along with any error
// handed to onPanic.
func recoverPanicHarness(t *testing.T, logger *slog.Logger, fn func()) (survived bool, got error) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer Recover(logger, "test", func(err error) { got = err })
		fn()
		survived = true // only reached if fn did not panic
	}()
	<-done
	return survived, got
}

func TestRecoverContainsPanicAndLogsStack(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	// A panicking goroutine must NOT crash the process: the harness
	// goroutine returns normally because Recover swallowed the panic.
	_, got := recoverPanicHarness(t, logger, func() { panic("boom") })

	if got == nil || !strings.Contains(got.Error(), "boom") {
		t.Fatalf("onPanic err = %v, want one mentioning \"boom\"", got)
	}
	out := buf.String()
	if !strings.Contains(out, "goroutine panic recovered") {
		t.Errorf("log missing recovery line:\n%s", out)
	}
	if !strings.Contains(out, "component=test") {
		t.Errorf("log missing component attr:\n%s", out)
	}
	if !strings.Contains(out, "stack=") {
		t.Errorf("log missing stack attr:\n%s", out)
	}
}

func TestRecoverPreservesErrorValue(t *testing.T) {
	sentinel := errors.New("sentinel")
	_, got := recoverPanicHarness(t, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		func() { panic(sentinel) })
	if !errors.Is(got, sentinel) {
		t.Fatalf("onPanic err = %v, want it to wrap/equal the panicked error", got)
	}
}

func TestRecoverNoPanicIsNoop(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	called := false
	func() {
		defer Recover(logger, "test", func(error) { called = true })
	}()
	if called {
		t.Error("onPanic called with no panic")
	}
	if buf.Len() != 0 {
		t.Errorf("logged on the no-panic path: %s", buf.String())
	}
}
