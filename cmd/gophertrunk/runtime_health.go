package main

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/diag"
)

// applyMemoryLimit installs a soft heap limit so the Go GC keeps the resident
// footprint bounded instead of letting it balloon under sustained
// high-allocation load. Without it a long-running daemon can be SIGKILLed by
// the OS memory-pressure killer (Linux OOM-killer / macOS jetsam) after a few
// minutes — leaving no in-process trace, the silent "the log just stops"
// failure mode in issue #492.
//
// Precedence: the GOMEMLIMIT env var (already applied by the runtime before
// main) always wins; then diagnostics.memory_limit_mb; then an auto-default of
// ~70% of physical RAM when that is known; otherwise the runtime is left
// unbounded (with a hint to set a limit).
func applyMemoryLimit(cfg config.Config, log *slog.Logger) {
	if os.Getenv("GOMEMLIMIT") != "" {
		// debug.SetMemoryLimit(-1) returns the limit the runtime already
		// installed from the env var without changing it.
		log.Info("runtime: soft memory limit from GOMEMLIMIT env",
			"limit_bytes", debug.SetMemoryLimit(-1))
		return
	}
	var limit int64
	switch {
	case cfg.Diagnostics.MemoryLimitMB > 0:
		limit = int64(cfg.Diagnostics.MemoryLimitMB) * 1024 * 1024
	default:
		if total := diag.CollectSysInfo().MemTotalMB; total > 0 {
			limit = int64(total) * 1024 * 1024 * 7 / 10
		}
	}
	if limit <= 0 {
		log.Info("runtime: no soft memory limit (physical RAM unknown); set GOMEMLIMIT or diagnostics.memory_limit_mb to bound RSS against OOM/jetsam kills (issue #492)")
		return
	}
	debug.SetMemoryLimit(limit)
	log.Info("runtime: soft memory limit set", "limit_mb", limit/(1024*1024))
}

// heartbeatInterval resolves the diagnostics.heartbeat_seconds config knob:
// negative disables, zero uses the 60 s default, positive is taken verbatim.
func heartbeatInterval(sec int) time.Duration {
	switch {
	case sec < 0:
		return 0
	case sec == 0:
		return 60 * time.Second
	default:
		return time.Duration(sec) * time.Second
	}
}

// runHeartbeat logs a periodic runtime health line so a stop is never silent:
// a climbing goroutine/heap curve points at a leak, a frozen heartbeat on a
// still-live process points at a hang, and the last line before an abrupt log
// cut pins the pre-kill footprint for an OOM/jetsam diagnosis (issue #492).
func (d *Daemon) runHeartbeat(ctx context.Context, interval time.Duration) error {
	start := time.Now()
	t := time.NewTicker(interval)
	defer t.Stop()
	var ms runtime.MemStats
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			runtime.ReadMemStats(&ms)
			d.log.Info("runtime: heartbeat",
				"uptime", time.Since(start).Round(time.Second).String(),
				"goroutines", runtime.NumGoroutine(),
				"heap_alloc_mb", ms.HeapAlloc/(1024*1024),
				"heap_sys_mb", ms.HeapSys/(1024*1024),
				"sys_mb", ms.Sys/(1024*1024),
				"next_gc_mb", ms.NextGC/(1024*1024),
				"num_gc", ms.NumGC)
		}
	}
}
