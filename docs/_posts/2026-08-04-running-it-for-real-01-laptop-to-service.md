---
title: "Running It For Real, Part 1: From a Laptop Demo to a 24/7 Service"
description: What actually changes when GopherTrunk stops being a laptop demo and becomes an unattended 24/7 service — the daemon lifecycle, dependency-ordered construction and teardown, graceful shutdown, the supervised spawn model, and the operator mindset behind the whole series.
category: deep-dives
keywords: sdr daemon lifecycle, graceful shutdown, gophertrunk service, 24/7 sdr scanner, daemon supervision, essential vs non-essential goroutines, panic recovery, systemd sdr, gophertrunk running it for real
tags: [running-it-for-real, ops, deployment, lifecycle, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Running It For Real"
series_part: 1
---

*Part 1 of **Running It For Real**, a 14-part deep dive into taking one
GopherTrunk daemon from a laptop demo to a hardened service that has been feeding
a public channel for six months without anyone touching it. Every earlier series
ended on something working *once* — a control channel decoded, a call recorded, a
system discovered. This one starts from "it works on my laptop" and asks the
uncomfortable follow-up: **what breaks when you walk away and leave it running?**
This opener is the map of that whole journey, and it plants the thread we follow
the rest of the way — the difference between a process that is *up* and a process
that is *doing its job*.*

> **TL;DR:** Running for real is a **lifecycle problem**, not a feature. The same
> binary that decodes on your desk has to construct its subsystems in dependency
> order, supervise a few dozen background goroutines, tell an essential failure
> apart from a cosmetic one, drain in-flight calls and streams on a signal, and
> never die silently. GopherTrunk factors that into one `Daemon` type
> (`cmd/gophertrunk/daemon.go`) with three phases — **construct** (`NewDaemon`),
> **run** (`Run` blocks until the context cancels), **tear down** (`Close`, in
> reverse order) — plus a `spawn` helper that wraps every component in a panic
> guard. The rest of the series is the operator surface layered on top: auth,
> TLS, metrics, logs, diagnostics, feature flags, and the container/systemd
> plumbing.

**Key takeaways**

- **A demo optimises for the first success; a service optimises for the
  thousandth.** The mindset shift is from "does it decode?" to "when it stops
  decoding at 3am, does the timeline tell me why?"
- **Construction is dependency-ordered and degrades gracefully.** A missing
  storage path, an empty SDR pool, or an unparseable gain becomes a *warning* the
  operator sees, not a crash — the daemon builds what it can and runs the rest.
- **Only one component is essential.** The trunking engine failing unwinds the
  daemon; a paging receiver or a webhook failing is logged and left behind. That
  one bit — `essential bool` — is the whole supervision policy.
- **Nothing is allowed to fail silently.** A `signal.NotifyContext`, a
  reverse-order `Close`, a 30 s connection-drain window, a per-goroutine panic
  guard, and a periodic heartbeat exist so "the log just stops" is never the last
  thing you see.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Construct | build subsystems in dependency order, warn-not-crash | `cmd/gophertrunk/daemon.go` (`NewDaemon`) |
| Run | spawn every component, block until ctx cancels | `daemon.go` (`Daemon.Run`) |
| Supervise | one goroutine per component, essential vs not | `daemon.go` (`Daemon.spawn`) |
| Tear down | close everything in reverse order, drain calls | `daemon.go` (`Daemon.Close`) |
| Escalate | first essential error cancels the run context | `daemon.go` (`recordFatal` / `takeFatal`) |
| Recover | turn a panic into a logged fatal, not a crash | `internal/log/recover.go` (`Recover`) |
| Heartbeat | periodic health line so a stop is never silent | `cmd/gophertrunk/runtime_health.go` |
| Readiness | gate the launcher until the API has bound | `daemon.go` (`Ready` / `markReadyAfter`) |

## In this post

- **What actually changes** going from a laptop demo to an unattended service.
- **The three lifecycle phases** — construct, run, tear down — and why the order matters.
- **The supervision model** — one `spawn`, one `essential` bit, one fatal path.
- **Never failing silently** — graceful shutdown, panic recovery, the heartbeat.
- **The map of the series** — where the next thirteen posts fit.

## What actually changes

On a laptop you run `gophertrunk`, watch a call scroll past, and `Ctrl-C` when
you're bored. Every failure mode is one you're standing in front of: no dongle,
wrong gain, no lock. You fix it and re-run. The feedback loop is seconds long and
you are the supervisor.

A service has none of that. It starts at boot, runs for months, survives a dongle
that browns out and re-enumerates on a different USB path, and keeps a public feed
alive through all of it. The failures you never saw on the laptop — a slow memory
leak the OOM killer eventually reaps, a panic in one obscure decoder goroutine, a
config path that resolves to the wrong place — are now the *only* failures that
matter, because they happen while you're asleep.

So the instinct this whole series leans on is the same one the
[RF Front End]({{ '/blog/series/rf-front-end/' | relative_url }}) series ended
on: **name the failure modes and give each one a surface.** A demo can afford to
crash — you'll re-run it. A service has to convert every silent degradation into
something an operator, a dashboard, or a log line can point at. That conversion
is what "running it for real" means, and the daemon lifecycle is where it starts.

## The three lifecycle phases

The `Daemon` type owns every long-lived thing the binary runs — the event bus,
the SDR pool, the trunking engine, the recorder, the SQLite call log, the
Prometheus collector, the HTTP and gRPC servers — and it moves them through three
phases in a fixed order.

<figure class="lab-figure">
<svg viewBox="0 0 680 168" width="680" height="168" role="img" aria-label="The daemon lifecycle: NewDaemon constructs subsystems in dependency order, Run spawns each as a supervised goroutine and blocks until the context cancels, and Close tears everything down in reverse order while the engine drains active calls">
  <rect x="6" y="52" width="150" height="52" rx="6" fill="none" stroke="currentColor"/>
  <text x="81" y="74" text-anchor="middle" fill="currentColor" font-size="12">NewDaemon</text>
  <text x="81" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="9">construct in order</text>
  <line x1="156" y1="78" x2="182" y2="78" stroke="currentColor"/><polygon points="182,74 192,78 182,82" fill="currentColor"/>
  <rect x="192" y="52" width="150" height="52" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="267" y="74" text-anchor="middle" fill="var(--accent)" font-size="12">Run</text>
  <text x="267" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="9">spawn + block</text>
  <line x1="342" y1="78" x2="368" y2="78" stroke="currentColor"/><polygon points="368,74 378,78 368,82" fill="currentColor"/>
  <rect x="378" y="52" width="150" height="52" rx="6" fill="none" stroke="currentColor"/>
  <text x="453" y="74" text-anchor="middle" fill="currentColor" font-size="12">Close</text>
  <text x="453" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="9">reverse order + drain</text>
  <rect x="564" y="52" width="110" height="52" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="619" y="74" text-anchor="middle" fill="var(--fg-muted)" font-size="11">SIGINT/TERM</text>
  <text x="619" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="9">cancels ctx</text>
  <line x1="564" y1="78" x2="530" y2="78" stroke="var(--fg-muted)"/><polygon points="530,74 520,78 530,82" fill="var(--fg-muted)"/>
  <text x="300" y="142" text-anchor="middle" fill="var(--fg-muted)" font-size="10">construction warns instead of crashing; teardown mirrors construction; the engine drains active calls before the DB closes</text>
</svg>
<figcaption>Construct → run → tear down. Signals cancel the root context; teardown unwinds in reverse so nothing is closed out from under a component that still needs it.</figcaption>
</figure>

**Construct** (`NewDaemon`) is where the graceful-degradation philosophy shows up
most. Almost nothing is fatal. A talkgroup CSV that fails to load becomes a
startup warning ("calls on this system will have no alpha tags"), not an abort. An
SDR pool that fails to open logs "no radios will demodulate; check device
permissions / cabling / kernel modules" and sets the pool to nil so downstream
components fall through gracefully. Even a gain value that looks like a unit
mistake — `32` when the operator meant `320` tenths-of-dB — earns a specific WARN
rather than silent deafness. The pattern is everywhere: build what you can,
collect what you couldn't into `startupWarnings`, and let the launcher and TUI
pin those warnings where a human will see them.

**Run** spawns each constructed component as its own supervised goroutine and
then blocks on a single line — `<-runCtx.Done()` — until something cancels the
context. **Tear down** (`Close`) closes everything in the reverse of construction
order, and it's idempotent via a `sync.Once` so a signal and an essential failure
racing each other can't double-close. The ordering is load-bearing: the HTTP
server closes first (stop accepting work), the engine closes near the end (drain
active calls into the call log), and the database and pool close last, after
everyone who writes to them is done.

## The supervision model

Every component in `Run` goes through one helper, and reading it tells you the
entire failure policy in twenty lines:

```go
// cmd/gophertrunk/daemon.go (shape) — Daemon.spawn
func (d *Daemon) spawn(ctx context.Context, name string, essential bool, fn func(context.Context) error) {
    d.wg.Add(1)
    go func() {
        defer d.wg.Done()
        // A panic here would otherwise crash the whole process with only a
        // stderr stack — the silent "log just stops" failure mode (#492).
        defer gtlog.Recover(d.log, "spawn:"+name, d.recordFatal)
        err := fn(ctx)
        if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
            return // clean exit or an expected shutdown cancel
        }
        if essential {
            d.log.Error("daemon: essential component failed", "component", name, "err", err)
            d.recordFatal(fmt.Errorf("%s: %w", name, err))
            return
        }
        d.log.Warn("daemon: component exited with error", "component", name, "err", err)
    }()
}
```

There are only two tiers. **Essential** is the trunking engine and, effectively,
nothing else — its failure means the daemon can't do the one job it exists for,
so `recordFatal` captures the first error and calls the run context's `cancel`,
which unwinds every sibling. **Non-essential** is everything else: a paging
receiver whose SDR isn't found, a `rigctld` server whose port is already taken by
a real Hamlib daemon, a DMR band-plan learner that hits a snag. Those get a WARN
and the daemon keeps decoding. That single `essential bool` is the whole
supervision contract — no restart storms, no partial-failure ambiguity. A
component either matters enough to take the daemon down with it, or it doesn't and
it's allowed to die alone.

### How that principle shaped the Go code

- **One fatal, first wins.** `recordFatal` guards `fatalErr` with a mutex and
  only stores the *first* error, then cancels. A cascade of secondary failures
  during teardown can't overwrite the real cause; `takeFatal` hands it back to
  `main` as the process exit error.
- **The panic guard escalates regardless of tier.** A panic is never expected, so
  `log.Recover` logs the value and the goroutine stack and calls `recordFatal`
  even for a "non-essential" component — a bug that corrupts state shouldn't be
  swallowed just because the component was cosmetic.
- **`WaitGroup`, not fire-and-forget.** `Run` does `d.Close(); d.wg.Wait()` after
  the context cancels, so shutdown actually *waits* for every goroutine to unwind
  before the process exits — no half-flushed call log, no truncated recording.
- **Readiness is separate from liveness.** `markReadyAfter` closes a `ready`
  channel a beat after spawn so the launcher never prompts against a half-dead
  daemon; the health endpoint (Part 4's neighbour) reports the deeper "actually
  doing work" state.

## Never failing silently

The deepest lesson baked into the lifecycle came from a real bug —
issue #492, where the daemon's log "just stopped" mid-line and the process was
gone. Three separate mechanisms exist so that can't recur, and each is a small
piece of the run/teardown path:

```go
// cmd/gophertrunk/runtime_health.go (shape) — runHeartbeat
func (d *Daemon) runHeartbeat(ctx context.Context, interval time.Duration) error {
    // A periodic health line: a climbing goroutine/heap curve points at a leak,
    // a frozen heartbeat on a live process points at a hang, and the last line
    // before an abrupt log cut pins the pre-kill footprint for an OOM diagnosis.
    var ms runtime.MemStats
    for range ticker.C {
        runtime.ReadMemStats(&ms)
        d.log.Info("runtime: heartbeat", "uptime", …, "goroutines", runtime.NumGoroutine(),
            "heap_alloc_mb", ms.HeapAlloc/(1<<20), "num_gc", ms.NumGC)
    }
}
```

First, a `signal.NotifyContext` for `SIGINT`/`SIGTERM` cancels the root context,
which flows to every component's `Run(ctx)` — the same context every `spawn`
closes over — so a `systemctl stop` unwinds cleanly instead of `SIGKILL` tearing
connections mid-frame. Second, the HTTP server's `Shutdown` gets a 30 s drain
window so in-flight SSE, WebSocket, and per-call audio-stream subscribers see a
clean close, and gRPC uses `GracefulStop`. Third, the daemon installs a **soft
memory limit** (`applyMemoryLimit`, ~70% of physical RAM by default) so the Go GC
keeps RSS bounded instead of letting the OS reap the process with no trace. The
heartbeat above is the fourth: it turns "the log stops" into a diagnosable
timeline, because the *last* heartbeat pins the footprint and goroutine count
right before the end. None of these are features you'd demo. All of them are why
the thing is still running in month six.

## Where this goes next

The lifecycle is the skeleton; the next thirteen posts are the operator surface
bolted onto it. [Part 2]({{ '/blog/deep-dives/running-it-for-real-02-auth-posture/' | relative_url }})
is the first decision you make before the daemon leaves your LAN — the auth
posture, and why the default is `disabled` for closed networks and how you opt
back up to `auto` or `required`. From there: TLS and reverse proxies (Part 3),
the Prometheus metrics and SDR tiles worth alerting on (Part 4), the structured
event/message/power logs and panic recovery (Part 5), the diagnostics reporter
and boot banner (Part 6), the `sdr doctor` preflight that catches a bad dongle
before it costs a call (Part 7), the opt-in feature matrix (Part 8), and onward
through the broadcast backends and the Docker/systemd plumbing. The
[Hardening]({{ '/hardening.html' | relative_url }}) and
[Opt-in features]({{ '/opt-in-features.html' | relative_url }}) docs are the
operator reference; this series is the design behind them.

## FAQ

**Is `gophertrunk run` different from just running `gophertrunk`?**
It's the same binary and the same `Daemon`. The difference is what wraps it: the
service invocation installs the signal-cancelled context, the soft memory limit,
and the heartbeat, and it's launched by systemd/Docker rather than a terminal.
The decode path is identical — that's the point. What changes is the supervision
and observability around it.

**Why is only the trunking engine "essential"?**
Because it's the one component whose job the daemon can't do without. If a paging
receiver or a webhook dies, the daemon is still decoding trunked calls — its
core purpose — so taking the whole process down would be worse than logging the
failure. The engine dying means there's nothing left to supervise.

**What happens to an active call when I restart the daemon?**
The engine drains it. On context cancel, every `ActiveCall` gets a final
`CallEnd` with a normal reason so the call log captures it before the database
closes, and the 30 s HTTP drain window lets any live audio-stream subscriber
finish its frame instead of being cut off mid-word.

**How do I know it's actually working, not just running?**
Liveness (the process is up) and readiness (it's doing work) are different
questions. `GET /api/v1/health` reports `pool_attached_count`, `active_calls`,
and `db_connected` so a probe can distinguish the two — Part 4 and the
[Hardening]({{ '/hardening.html' | relative_url }}) doc cover the readiness recipe.

**Does a panic in one decoder crash everything?**
No. `log.Recover` catches it in the component's goroutine, logs the value and
stack, and routes it through `recordFatal` for a clean, logged shutdown — never a
bare stderr stack and a vanished process.

## Series navigation

**Part 1 of 14** · Next →
[Part 2: Auth Posture — Closed-LAN, Auto, Required]({{ '/blog/deep-dives/running-it-for-real-02-auth-posture/' | relative_url }})
