//go:build windows

package main

import (
	"context"
	"log/slog"
)

// watchReloadSignal is a no-op on Windows — SIGHUP isn't available
// there. Operators reload by restarting the daemon (or by issuing
// PATCH /api/v1/settings, which is the same hot-reload path under
// the hood).
func watchReloadSignal(_ context.Context, _ *Daemon, _ *slog.Logger) {}
