//go:build !windows

package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

// watchReloadSignal listens for SIGHUP and triggers a daemon config
// reload on each receipt. Cancellation tears down the signal
// handler. On Windows there is no SIGHUP (see sighup_windows.go for
// the no-op build).
func watchReloadSignal(ctx context.Context, d *Daemon, log *slog.Logger) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP)
	defer signal.Stop(ch)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ch:
			summary, err := d.Reload()
			if err != nil {
				log.Warn("sighup: reload failed", "err", err)
				continue
			}
			log.Info(summary)
		}
	}
}
