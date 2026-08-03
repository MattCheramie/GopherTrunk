---
title: "Running It For Real, Part 14: Staying Up — Health, Watchdogs & the Ops Payoff"
description: The finale — the health endpoint that tells a probe the daemon is doing work not just running, the heartbeat that makes a stop never silent, the soft memory limit that dodges the OOM killer, and the USB watchdog that reacquires a dropped dongle without a restart.
category: deep-dives
keywords: health endpoint sdr, readiness probe daemon, usb watchdog reacquire, soft memory limit gomemlimit, oom killer, heartbeat logging, uptime monitoring, staying up, gophertrunk running it for real
tags: [running-it-for-real, monitoring, health, watchdog, operations, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Running It For Real"
series_part: 14
---

*Part 14 — the finale — of **Running It For Real**. Thirteen posts ago the daemon
was a laptop demo. Since then we've picked its auth posture, wired its metrics and
logs, taught it to catch a failing dongle, gated its optional subsystems, fanned
its calls out to five destinations, and hardened where it runs on three surfaces.
This last post is the promise all of that was for: **staying up**. Not "it started"
— any process starts — but running for months, surviving a USB blip at 3 a.m.
without you, dodging the OOM killer, and telling a monitor the difference between
"the process is alive" and "the process is actually doing work." This is what
turns a binary you launched into a service you trust.*

> **TL;DR:** Three mechanisms keep GopherTrunk up unattended. `GET /api/v1/health`
> returns not just `status: "ok"` but the facts a probe needs to tell *running*
> from *working* — attached SDR count, active calls, DB connectivity, auth mode —
> so a k8s/Nomad readiness probe fails a daemon that's up but blind. A periodic
> **heartbeat** logs uptime, goroutine count, and heap so a stop is never silent —
> a climbing curve is a leak, a frozen line a hang, the last line before a cut the
> pre-kill footprint. A **soft memory limit** keeps RSS bounded so the OS never
> SIGKILLs the process. And the **USB watchdog** re-enumerates every 30 seconds and
> reacquires a dropped dongle on the transition back, without a restart.

**Key takeaways**

- **Health means "doing work," not "process up."** The extended `HealthDTO` fields
  let a probe distinguish a serving daemon from a serving-but-blind one — zero
  attached SDRs is a failing probe, not a 200 with a shrug.
- **A stop is never silent.** The heartbeat's uptime/goroutine/heap line turns three
  different failure modes — leak, hang, OOM kill — into readable signatures in the
  log.
- **The daemon bounds its own memory.** A soft heap limit (from `GOMEMLIMIT`, config,
  or ~70% of RAM) keeps the GC from letting RSS balloon into an OOM/jetsam kill that
  leaves no trace.
- **A USB blip self-heals.** The watchdog acts only on serial-level *transitions* —
  detach surfaces an event, re-appear triggers a reacquire — so flaky USB doesn't
  need a human or a restart.

## Cheat sheet

| Mechanism | Where it lives | What it catches |
|---|---|---|
| Health probe | `handleHealth` → `HealthDTO` | up vs actually-working |
| Heartbeat | `runHeartbeat` (`runtime_health.go`) | leak / hang / pre-kill footprint |
| Soft memory limit | `applyMemoryLimit` | OOM-killer / jetsam SIGKILL (issue #492) |
| USB watchdog | `Pool.RunWatchdog` (`internal/sdr/watchdog.go`) | dropped dongle re-acquire (issue #345) |
| Runtime snapshot | `handleRuntime` → `RuntimeDTO` | effective config + startup warnings |
| Status reference | [status.html]({{ '/status.html' | relative_url }}) | what ships vs what's pending |

## In this post

- **Two kinds of health** — why `status: "ok"` alone lies.
- **The heartbeat** — making every stop legible in the log.
- **Bounding memory** — the soft limit that dodges the OOM killer.
- **The USB watchdog** — self-healing a dropped dongle on the transition.
- **The payoff** — what months of uptime actually look like.

## Two kinds of health

The naive health check answers one question: is the process listening? That's the
question that lies. A daemon can serve `200 OK` while every SDR has detached and it
hasn't decoded a call in an hour — up, and useless. So `GET /api/v1/health` returns
the facts a probe needs to tell the difference:

```go
// internal/api/handlers.go (shape) — HealthDTO
type HealthDTO struct {
    Status            string    `json:"status"` // always "ok" for a serving daemon
    Now               time.Time `json:"now"`    // offset-bearing; a probe can spot clock skew
    Version           string    `json:"version,omitempty"`
    PoolAttachedCount int       `json:"pool_attached_count"` // 0 = blind → actionable
    ActiveCalls       int       `json:"active_calls"`
    DBConnected       bool      `json:"db_connected"`
    MetricsEnabled    bool      `json:"metrics_enabled"`
    AuthMode          string    `json:"auth_mode,omitempty"` // flag a misconfigured prod
}
```

The struct comment names the intent: the extended fields "let k8s / Nomad
readiness probes and operator dashboards distinguish 'the daemon process is up'
from 'the daemon process is actually doing work'." `PoolAttachedCount` is the
sharpest of them — zero means either no SDR provider is wired *or* every device
detached, and both are operator-actionable, so a probe that keys on it fails a
blind daemon instead of passing it. `AuthMode` lets a probe catch a production box
that came up with auth disabled. And every field is **best-effort**: a missing
collaborator (no engine, no history DB) leaves its field at the zero value rather
than failing the request, so the probe always gets a stable shape to read. This is
the endpoint the Part 12 container `healthcheck` and any external monitor point
at — health that means something.

## The heartbeat: no silent stops

A probe tells an *outside* observer the daemon is working. The heartbeat tells the
*log* what happened when it stops. Its whole reason for existing is that the worst
production failure is the one that leaves no trace — the log just ends. So a ticker
writes a runtime line at a configurable cadence:

```go
// cmd/gophertrunk/runtime_health.go (shape)
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
                "num_gc", ms.NumGC)
        }
    }
}
```

That one line makes three different failures legible, as its doc comment lays out.
A **climbing** goroutine or heap curve across heartbeats points at a leak. A
heartbeat that **freezes** on a still-live process points at a hang — the ticker is
fine but something upstream stopped feeding it. And the **last line before an abrupt
log cut** pins the pre-kill footprint, which is the single most useful datum for
diagnosing an OOM/jetsam kill (issue #492) — you see exactly how much memory the
process held the instant before the OS took it. The cadence is a config knob:
negative disables it, zero uses the 60-second default, positive is taken verbatim.
A stop you can read is a stop you can fix.

## Bounding memory: dodging the OOM killer

The heartbeat *diagnoses* an OOM kill; the soft memory limit *prevents* it. Under
sustained high-allocation load, Go's GC will happily let the resident footprint
grow until the OS memory-pressure killer (Linux OOM-killer, macOS jetsam) SIGKILLs
the process — the silent "the log just stops" failure. So the daemon installs a
soft heap limit at startup, with a clear precedence:

```go
// cmd/gophertrunk/runtime_health.go (shape) — applyMemoryLimit
// Precedence: GOMEMLIMIT env (already applied by the runtime) wins; then
// diagnostics.memory_limit_mb; then ~70% of physical RAM when known;
// otherwise unbounded, with a hint to set a limit.
switch {
case cfg.Diagnostics.MemoryLimitMB > 0:
    limit = int64(cfg.Diagnostics.MemoryLimitMB) * 1024 * 1024
default:
    if total := diag.CollectSysInfo().MemTotalMB; total > 0 {
        limit = int64(total) * 1024 * 1024 * 7 / 10
    }
}
debug.SetMemoryLimit(limit)
```

`GOMEMLIMIT` in the environment always wins (the runtime applies it before `main`,
so the code just reads it back and logs it). Failing that, an explicit
`diagnostics.memory_limit_mb` config value; failing *that*, an auto-default of ~70%
of physical RAM when the daemon can detect it. Only if physical RAM is genuinely
unknown does it run unbounded — and even then it logs a hint to set a limit. The
effect is that the GC works harder to keep RSS under the ceiling rather than
letting it balloon past what the host will tolerate. On a memory-constrained box,
this and the systemd `MemoryMax` from Part 13 are belt and braces: the soft limit
keeps the process *inside* the budget, and the hard limit is the backstop if it
ever escapes.

## The USB watchdog: self-healing a dropped dongle

The failure a 24/7 receiver hits most is not a crash — it's a USB blip. A dongle
browns out, a hub cycles, autosuspend nips a device mid-idle, and now the pool
holds a dead handle. The watchdog turns that from an outage into a non-event. It
ticks every 30 seconds, re-enumerates every driver, and — the key design choice —
**acts only on the transition**, never on the steady state:

```go
// internal/sdr/watchdog.go (shape) — watchdogTick
for _, serial := range expected {
    _, here := present[serial]
    if !here {
        if !missing[serial] { // first tick it's gone
            missing[serial] = true
            p.log.Warn("sdr: watchdog: device missing from USB enumerate", "serial", serial)
            if entry := p.FindBySerial(serial); entry != nil {
                p.publish(events.KindSDRDetached, entry.Snapshot(false))
            }
        }
        continue
    }
    if missing[serial] { // was gone, now back
        delete(missing, serial)
        p.log.Info("sdr: watchdog: device reappeared; reacquiring", "serial", serial)
        if _, err := p.Reacquire(serial, sampleRateHz); err != nil {
            p.log.Warn("sdr: watchdog: reacquire failed", "serial", serial, "err", err)
        }
    }
}
```

The transition discipline is what makes it safe to run forever. A device that was
always present and still is gets left completely alone — no spurious reacquires on
healthy hardware. A device that vanishes flips to `missing` and surfaces a
`KindSDRDetached` event **once**, so the API/TUI/web snapshots show the gap without
log spam. And a device that *comes back* triggers exactly one `Pool.Reacquire`,
swapping the dead USB handle for the freshly-enumerated one **before the next
consumer touches it** — so a decoder or voice-pool bind that resumes after the blip
finds a live handle, not a stale one. The 30-second cadence (issue #345) is tuned
to catch a transient drop within one failure cycle without the periodic
re-enumerate showing up as background load on a slow hub. A reacquire failure isn't
terminal either — the in-stream retry paths and the next consumer will try again —
so one bad tick can't wedge the watchdog. The operator does nothing; the dongle
comes back on its own.

<figure class="lab-figure">
<svg viewBox="0 0 660 168" width="660" height="168" role="img" aria-label="A state timeline of one SDR under the watchdog. It starts present and healthy, with the watchdog leaving it alone tick after tick. A USB blip drops it: on the first missing tick the watchdog marks it missing and publishes one SDR-detached event. It stays missing across several ticks with no repeated action. When it re-appears in the enumerate, the watchdog fires exactly one reacquire, swapping the dead handle for a live one, and the SDR returns to present and healthy.">
  <line x1="30" y1="90" x2="630" y2="90" stroke="var(--fg-muted)"/>
  <polygon points="630,86 640,90 630,94" fill="var(--fg-muted)"/>
  <rect x="40" y="72" width="120" height="36" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="100" y="88" text-anchor="middle" fill="var(--accent)" font-size="9">present</text>
  <text x="100" y="101" text-anchor="middle" fill="var(--fg-muted)" font-size="8">left alone</text>
  <line x1="180" y1="60" x2="180" y2="120" stroke="currentColor" stroke-dasharray="3 3"/>
  <text x="180" y="52" text-anchor="middle" fill="currentColor" font-size="8">USB blip</text>
  <rect x="210" y="72" width="200" height="36" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="310" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="9">missing (marked once)</text>
  <text x="310" y="101" text-anchor="middle" fill="var(--fg-muted)" font-size="8">→ 1 SDRDetached event, then quiet</text>
  <line x1="430" y1="60" x2="430" y2="120" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <text x="430" y="52" text-anchor="middle" fill="var(--accent)" font-size="8">re-appears</text>
  <rect x="440" y="66" width="90" height="24" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="485" y="82" text-anchor="middle" fill="var(--accent)" font-size="8">1 Reacquire</text>
  <rect x="540" y="72" width="86" height="36" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="583" y="88" text-anchor="middle" fill="var(--accent)" font-size="9">present</text>
  <text x="583" y="101" text-anchor="middle" fill="var(--fg-muted)" font-size="8">live handle</text>
  <text x="330" y="146" text-anchor="middle" fill="var(--fg-muted)" font-size="9">the watchdog acts only on the two transitions — never on the steady state</text>
</svg>
<figcaption>Detach and re-appear each fire exactly one action; the long stretches of unchanged state fire none. Flaky USB becomes a logged blip instead of an outage.</figcaption>
</figure>

### How these principles shaped the Go code

- **Health is best-effort, never fatal.** Every `HealthDTO` field degrades to its
  zero value when a collaborator is absent, so the probe endpoint can't be taken down
  by a missing subsystem — the one thing a health check must never do is fail because
  something it *reports on* failed.
- **The watchdog owns its state map alone.** The `missing` map is touched by only the
  watchdog goroutine, and the pool snapshot is read under a lock, so re-enumeration
  never races a concurrent open/close/reacquire — a supervisor that corrupted the pool
  it supervises would be worse than none.
- **Every self-healing action is idempotent-under-transition.** Publishing detach
  once, reacquiring once, leaving steady state alone — the design is that running the
  tick a million times on healthy hardware does nothing, so it's safe at any cadence
  for any duration.

## The payoff: what months of uptime look like

Put it together and the picture of a healthy long-running GopherTrunk is quiet in a
very specific way. The heartbeat line repeats every minute with a flat heap and a
steady goroutine count — no drift, no freeze. `GET /api/v1/health` returns a stable
`pool_attached_count` matching your dongles and a live `active_calls` that ebbs and
flows with traffic. The broadcast counters from Parts 9–11 climb; `failed` and
`dropped` stay flat. When a dongle blips at 3 a.m., the log shows one detach warning
and, seconds to a minute later, one reacquire line — and nothing else changed. The
[status page]({{ '/status.html' | relative_url }}) and `GET /api/v1/runtime` (with
its `StartupWarnings` surfaced) tell you what's shipping and what's pending, so
there are no silent gaps in your mental model of the box.

That's the whole arc of **Running It For Real**. We started with a binary that
decoded on a laptop and ended with a service that picks its own auth posture,
reports its own health in terms that mean *working* not merely *running*, bounds
its own memory, heals its own USB, hardens its own filesystem, and streams to the
wider world — all in pure Go, one static binary, no CGO, on Linux, in Docker, or on
Windows. The engineering that got here wasn't glamorous: health DTOs, a heartbeat
ticker, a memory ceiling, a 30-second re-enumerate. But that's exactly the point.
The unglamorous machinery is what lets you stop watching, walk away, and find the
feed still live six months later. A hardened, always-on receiver is the deliverable
— and now it's yours.

## FAQ

**Why does `/api/v1/health` return more than `status: "ok"`?**
Because `status: "ok"` alone can't distinguish a working daemon from one that's up but
blind. The extended fields — `pool_attached_count`, `active_calls`, `db_connected`,
`auth_mode` — let a readiness probe fail a daemon that has zero SDRs attached or came up
with auth disabled, which a bare status string would pass. Health should mean "doing
work," and those fields are how.

**What is the heartbeat actually for?**
Making a stop legible. The per-minute line records uptime, goroutine count, and heap, so
a climbing curve reads as a leak, a frozen line as a hang, and the last line before an
abrupt log cut as the pre-kill memory footprint for an OOM diagnosis. Without it, a
process that dies to the OS killer leaves no trace but a truncated log.

**Do I need to set a memory limit myself?**
Usually not — the daemon auto-defaults to ~70% of physical RAM when it can detect it. Set
`GOMEMLIMIT` or `diagnostics.memory_limit_mb` to override, and pair it with the systemd
`MemoryMax` from Part 13 on a constrained host. The soft limit keeps RSS inside the
budget so the GC works harder instead of letting the OS SIGKILL the process.

**Will a USB glitch take the daemon down?**
No. The watchdog re-enumerates every 30 seconds and acts only on transitions: a
disappearance surfaces one detach event, and a re-appearance triggers exactly one
reacquire that swaps the dead handle for a live one before the next consumer touches it.
Flaky USB becomes a logged blip, not an outage, and needs no restart.

**How do I know what's shipping versus still pending?**
The [status page]({{ '/status.html' | relative_url }}) is the long-form reference for
what runs end-to-end versus what's a documented follow-up, and `GET /api/v1/runtime`
reports the live daemon's effective config plus any `StartupWarnings` it collected at
boot. Between them you never have a silent gap between what you think is on and what
actually is.

## Series navigation

**Part 14 of 14** · ←
[Part 13: systemd Hardening & the Windows Installer]({{ '/blog/deep-dives/running-it-for-real-13-systemd-windows/' | relative_url }})
· This is the finale — back to the
[series index]({{ '/blog/series/running-it-for-real/' | relative_url }}).

*Where to next? A hardened, always-on receiver is only as useful as what it can find — head back out to [The Hunt]({{ '/blog/series/the-hunt/' | relative_url }}) to keep discovering new systems for it to watch.*
