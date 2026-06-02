package main

import (
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/MattCheramie/GopherTrunk/internal/diag"
)

// Shared diagnostics state for the CLI. One Collector is built lazily
// and reused across every reporter so the costly dongle enumeration
// (see internal/diag) runs at most once per process — and only if an
// error actually renders a banner.
var (
	diagOnce      sync.Once
	diagCollector *diag.Collector
	// verboseErrors is the resolved verbose-error setting. Precedence:
	// -verbose-errors flag > GOPHERTRUNK_VERBOSE_ERRORS env >
	// diagnostics.verbose_errors in config. It starts from the env var
	// so the earliest pre-config error sites still honor it, and is
	// upgraded as the flag/config become available.
	verboseErrors = envVerbose()
)

// envVerbose reports whether GOPHERTRUNK_VERBOSE_ERRORS is set to a
// truthy value.
func envVerbose() bool {
	v := strings.TrimSpace(os.Getenv("GOPHERTRUNK_VERBOSE_ERRORS"))
	return v != "" && v != "0" && !strings.EqualFold(v, "false")
}

// resolveVerbose folds a flag value and a config value into the
// package verboseErrors state (env keeps whatever it already set). Call
// once the flag and/or config are known.
func resolveVerbose(flagVal, cfgVal bool) {
	verboseErrors = verboseErrors || flagVal || cfgVal
}

// sharedCollector returns the process-wide diagnostics collector.
func sharedCollector() *diag.Collector {
	diagOnce.Do(func() { diagCollector = diag.NewCollector() })
	return diagCollector
}

// newReporter builds a diag.Reporter for a CLI command, wired to the
// shared collector and the resolved verbose setting.
func newReporter(prefix string) *diag.Reporter {
	return diag.NewReporter(prefix, verboseErrors, sharedCollector())
}

// logErr logs a daemon-side error. The daemon is non-interactive, so
// there's no prompt: when verbose is on it attaches the full wrapped
// chain (and a stack dump), otherwise just the flat error string.
func logErr(l *slog.Logger, verbose bool, msg string, err error) {
	if l == nil {
		l = slog.Default()
	}
	if verbose {
		l.Warn(msg,
			"chain", diag.Chain(err),
			"stack", string(diag.CaptureStack()))
		return
	}
	l.Warn(msg, "err", err)
}
