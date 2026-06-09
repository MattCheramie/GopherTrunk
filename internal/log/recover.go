package log

import (
	"fmt"
	"log/slog"
	"runtime/debug"
)

// Recover is a deferred panic guard for long-lived goroutines. Installed as
// `defer log.Recover(logger, "component", onPanic)`, it converts a panic —
// which would otherwise crash the entire process with only a stderr stack
// trace, leaving operators with a log that simply stops mid-line (issue
// #492) — into a logged ERROR carrying the recovered value and the
// goroutine stack.
//
// onPanic is optional. When non-nil it is called with the recovered error so
// the caller can escalate (e.g. the daemon records a fatal error and cancels
// its run context for a clean, logged shutdown). When nil the goroutine
// simply unwinds: its other deferred calls still run (e.g. closing an output
// channel), so a panicked IQ/stream worker surfaces downstream as a logged,
// recoverable stream-closed event instead of a silent process death.
func Recover(logger *slog.Logger, component string, onPanic func(err error)) {
	r := recover()
	if r == nil {
		return
	}
	err, ok := r.(error)
	if !ok {
		err = fmt.Errorf("panic: %v", r)
	}
	if logger == nil {
		logger = slog.Default()
	}
	logger.Error("goroutine panic recovered",
		"component", component,
		"panic", r,
		"stack", string(debug.Stack()))
	if onPanic != nil {
		onPanic(err)
	}
}
