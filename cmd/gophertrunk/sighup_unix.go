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
// reload on each receipt. signal.Notify is called synchronously
// before the function spawns its background loop so callers can
// rely on signals being captured by the time it returns — important
// for tests that send SIGHUP immediately after kicking off the
// watcher. The returned function tears down the signal handler;
// production callers typically defer it from main.
//
// On Windows there is no SIGHUP (see sighup_windows.go for the
// no-op build).
func watchReloadSignal(ctx context.Context, d *Daemon, log *slog.Logger) {
	ch := make(chan os.Signal, 1)
	// Register the handler synchronously here so a caller that
	// sends SIGHUP immediately after this returns has its signal
	// captured. (The previous in-goroutine signal.Notify raced
	// against fast SIGHUP deliveries — observed as a test process
	// being killed by the default SIGHUP action because the
	// goroutine hadn't installed the handler yet.)
	signal.Notify(ch, syscall.SIGHUP)
	go func() {
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
	}()
}
