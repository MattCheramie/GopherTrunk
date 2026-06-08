package diag

import (
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
)

// Chain unwraps the %w error chain and returns one entry per layer,
// outermost first. It also descends into multi-error joins (the
// Unwrap() []error shape produced by errors.Join). A nil error yields
// an empty slice.
//
// The code base wraps with fmt.Errorf("…: %w", err) and never attaches
// stack frames to errors, so this gives the full causal chain that
// today's terse single-line message collapses.
func Chain(err error) []string {
	var out []string
	var walk func(e error)
	walk = func(e error) {
		for e != nil {
			out = append(out, e.Error())
			// Multi-error join: recurse into each branch, then stop
			// the linear walk (Unwrap() error and Unwrap() []error are
			// mutually exclusive on a given value).
			if joined, ok := e.(interface{ Unwrap() []error }); ok {
				for _, sub := range joined.Unwrap() {
					walk(sub)
				}
				return
			}
			e = errors.Unwrap(e)
		}
	}
	walk(err)
	return out
}

// CaptureStack returns a goroutine stack dump for the calling
// goroutine, captured at the call site. Thin wrapper over
// runtime/debug.Stack so callers don't import debug directly.
func CaptureStack() []byte { return debug.Stack() }

// VerboseTrace renders the full error chain followed by the captured
// goroutine stack as a single multi-line string, suitable for printing
// under the banner or embedding in a log/JSON field. stack may be nil
// to omit the dump.
func VerboseTrace(err error, stack []byte) string {
	var b strings.Builder
	chain := Chain(err)
	if len(chain) > 0 {
		b.WriteString("error chain (outermost first):\n")
		for i, layer := range chain {
			fmt.Fprintf(&b, "  [%d] %s\n", i, layer)
		}
	}
	if len(stack) > 0 {
		b.WriteString("\ngoroutine stack:\n")
		b.Write(stack)
	}
	return b.String()
}
