package cryptolab

import (
	"io"
	"log/slog"
)

// Env carries the shared run context a Mode needs: where to log, where to
// write artifacts, the structured-output format, and the stream for
// human-facing text. It is built once by the CLI from the global flags and
// passed to whichever mode runs.
type Env struct {
	// Logger receives structured progress. Never nil inside a mode (the CLI
	// substitutes a discard logger when logging is off).
	Logger *slog.Logger
	// OutDir is the directory for append-only *.jsonl survivor logs and
	// resumable checkpoints. Empty means "no artifacts, stdout only".
	OutDir string
	// ResumePath points at a prior checkpoint to continue from (cell solver
	// and any other resumable mode). Empty starts fresh.
	ResumePath string
	// Format selects how the returned Result is rendered to Stdout.
	Format Format
	// Stdout is where the rendered Result and human text go.
	Stdout io.Writer
}

// Logf is a convenience that logs at info level when a logger is present.
func (e Env) Logf(msg string, args ...any) {
	if e.Logger != nil {
		e.Logger.Info(msg, args...)
	}
}
