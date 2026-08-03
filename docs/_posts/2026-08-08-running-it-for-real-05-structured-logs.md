---
title: "Running It For Real, Part 5: Structured Logs — Event, Message & Power"
description: How GopherTrunk writes three purpose-built logs off one event bus — the JSONL event log for machines, the human-readable decoded-message log, and the decode-gated power log — plus the panic-recovery guard that turns a crashing goroutine into a logged, survivable event.
category: deep-dives
keywords: structured logging, jsonl event log, decoded message log, power log, sdr signal level, panic recovery goroutine, log rotation, event bus sink, gophertrunk running it for real
tags: [running-it-for-real, logging, observability, ops, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Running It For Real"
series_part: 5
---

*Part 5 of **Running It For Real**, the series taking one GopherTrunk daemon from
a laptop demo to a hardened 24/7 service. Part 4 gave us metrics — the numbers
that say *something* is wrong. This post gives us the timeline that says *what* —
and, crucially, keeps that timeline intact when a goroutine panics. On a laptop
you read the console. As a service, the logs are the only witness to what happened
while you were asleep, so they have to be structured for machines, readable for
humans, and impossible to silence with a single crash.*

> **TL;DR:** GopherTrunk writes three logs, each a bus subscriber with the same
> shape — subscribe, drain events in a `Run(ctx)` loop, format, append, rotate at
> a size cap, drain-and-close cleanly. The **event log** (`eventlog.go`) is
> newline-delimited JSON of *every* bus event in the same envelope the SSE/WS
> streams emit — the machine-readable session recorder you hand to support. The
> **message log** (`messagelog.go`) is a human-readable, one-line-per-event
> decoded-message log (GopherTrunk's answer to SDRtrunk's channel log). The
> **power log** (`powerlog.go`) records per-channel IQ level *gated on decode
> activity*, so it captures the "decoding but weak" diagnostic without a flood of
> idle noise. Underneath all of it, `log.Recover` (`recover.go`) turns a panicking
> goroutine into a logged ERROR instead of the silent "log just stops" death.

**Key takeaways**

- **One bus, three sinks, three audiences.** The event log is for machines
  (JSONL, all kinds), the message log is for humans (formatted, trunking subset),
  the power log is for RF triage (level, decode-gated). They share plumbing and
  differ only in what they keep and how they render it.
- **The event log matches the live stream byte-for-byte.** The daemon injects the
  API's wire encoder, so a recorded session file is exactly what a live SSE/WS
  client would have seen — replayable, diffable, shareable.
- **The power log is gated on decode, not on time.** It only writes a line for a
  channel that decoded at least one frame that window, and by default only when
  the level is *low* — the "why is a live channel marginal" signal, not a
  full-time series.
- **A panic is a logged event, not a process death.** `log.Recover` on every
  spawned goroutine captures the value and stack; the timeline survives the crash
  that would otherwise end mid-line.

## Cheat sheet

| Log | Audience | Content | Where it lives |
|---|---|---|---|
| Event log | machines / support | all bus events as JSONL, wire-identical | `internal/log/eventlog.go` |
| Message log | humans | trunking events, one formatted line each | `internal/log/messagelog.go` |
| Power log | RF triage | per-channel IQ level, decode-gated | `internal/log/powerlog.go` |
| Panic guard | everyone | recover → logged ERROR + stack | `internal/log/recover.go` |

All three rotate to `<path>.1` at a 16 MB default cap and render displayed timestamps in the operator's `display.timezone`.

## In this post

- **Three logs, one bus** — the shared sink shape and why there are three.
- **The event log** — JSONL that matches the live stream.
- **The message log** — human-readable, and the neighbour-block dedup trick.
- **The power log** — decode-gated RF level, the "weak but live" diagnostic.
- **Panic recovery** — keeping the timeline alive through a crash.

## Three logs, one bus

Every log GopherTrunk writes is a subscriber to the one internal
[event bus]({{ '/blog/deep-dives/trunking-engine-02-event-bus/' | relative_url }}).
They share an identical skeleton — open a file, `bus.Subscribe()`, run a drain
loop, append formatted output, rotate at a size cap, and on `Close` release the
subscription and wait for the loop to finish. The drain loop is the same five
lines in all three:

```go
// internal/log/eventlog.go (shape) — the shared sink loop
func (e *EventLog) Run(ctx context.Context) error {
    defer close(e.runDone)
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case ev, ok := <-e.sub.C:
            if !ok {
                return nil // bus closed
            }
            b, err := e.encode(ev)
            if err != nil {
                continue // one unencodable event never tears down the sink
            }
            e.write(append(b, '\n'))
        }
    }
}
```

That `continue` on an encode error is a small resilience decision worth naming: a
single malformed event drops that line and the sink keeps recording the rest of
the session. A log that dies on the first surprise is worse than useless. So why
three logs off one bus instead of one log with three filters? Because they have
genuinely different audiences and lifetimes. You hand the event log to a machine
(or to Claude); you read the message log with your eyes; you grep the power log
when a channel is misbehaving. Different rendering, different retention instinct,
different noise tolerance — one sink each is cleaner than one sink with modes.

<figure class="lab-figure">
<svg viewBox="0 0 640 200" width="640" height="200" role="img" aria-label="One event bus fans out to three log sinks: a JSONL event log for machines, a formatted message log for humans, and a decode-gated power log for RF triage, with a panic-recovery guard wrapping every goroutine">
  <rect x="8" y="82" width="120" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="68" y="100" text-anchor="middle" fill="var(--accent)" font-size="12">event bus</text>
  <text x="68" y="115" text-anchor="middle" fill="var(--fg-muted)" font-size="9">every Kind</text>
  <line x1="128" y1="103" x2="196" y2="42" stroke="currentColor"/><polygon points="192,44 202,38 195,50" fill="currentColor"/>
  <line x1="128" y1="103" x2="196" y2="103" stroke="currentColor"/><polygon points="196,99 206,103 196,107" fill="currentColor"/>
  <line x1="128" y1="103" x2="196" y2="164" stroke="currentColor"/><polygon points="192,162 202,168 195,156" fill="currentColor"/>
  <rect x="206" y="18" width="200" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="306" y="37" text-anchor="middle" fill="currentColor" font-size="11">event log — JSONL, all kinds</text>
  <text x="306" y="52" text-anchor="middle" fill="var(--fg-muted)" font-size="9">machines · wire-identical</text>
  <rect x="206" y="82" width="200" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="306" y="100" text-anchor="middle" fill="currentColor" font-size="11">message log — formatted</text>
  <text x="306" y="115" text-anchor="middle" fill="var(--fg-muted)" font-size="9">humans · trunking subset</text>
  <rect x="206" y="144" width="200" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="306" y="163" text-anchor="middle" fill="currentColor" font-size="11">power log — level</text>
  <text x="306" y="178" text-anchor="middle" fill="var(--fg-muted)" font-size="9">RF triage · decode-gated</text>
  <rect x="440" y="70" width="192" height="66" rx="6" fill="none" stroke="var(--accent)" stroke-dasharray="4 3"/>
  <text x="536" y="94" text-anchor="middle" fill="var(--accent)" font-size="11">log.Recover</text>
  <text x="536" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="9">panic → logged ERROR</text>
  <text x="536" y="124" text-anchor="middle" fill="var(--fg-muted)" font-size="9">wraps every goroutine</text>
</svg>
<figcaption>One bus, three sinks, three audiences — and a recovery guard around every drain loop so a panic becomes a logged event instead of a half-written line.</figcaption>
</figure>

## The event log

The event log is the machine-readable session recorder. It captures **every** bus
event kind — not just the trunking subset — as newline-delimited JSON, one
self-contained object per line. The detail that makes it valuable is the encoder:

```go
// internal/log/eventlog.go (shape) — EventLogOptions.Encode
// Renders one event to a single JSON line. The daemon injects the api package's
// wire encoder so the file matches the live SSE/WS stream byte-for-byte; when nil
// a minimal {kind,timestamp,payload} envelope is used.
Encode func(events.Event) ([]byte, error)
```

Because the daemon passes the same encoder the API uses for its SSE and WebSocket
streams, a recorded event-log file is *exactly* what a live client would have
received over the wire. That's a stronger property than "a log of what happened."
It means you can replay a session offline, diff two runs, or hand a support case a
file that a developer can feed straight back through the SSE-consuming tooling. The
default envelope (`{kind, timestamp, payload}`) mirrors the API's `EventDTO` field
names, so even without the injected encoder a consumer sees the same keys. It's
the honest, complete record — the message and power logs are curated views; this
is the source.

## The message log

The message log is the one you actually read. It's GopherTrunk's analogue of
SDRtrunk's per-channel decoded-message log: one timestamped line per trunking
event — grants, CC lock/loss, affiliations, registrations, patches, talker
aliases, locations, tone alerts, decode errors — formatted for a human scanning
for the interesting one:

```
2026-06-25T14:03:05.250-05:00 GRANT        system=Metro proto=p25 tg=1042 src=90210 freq=851287500 enc=false emer=false
2026-06-25T14:03:05.310-05:00 CALL-START   system=Metro tg=1042 src=90210 dev=00000001
2026-06-25T14:03:11.880-05:00 CALL-END     system=Metro tg=1042 dur=6.57s reason=normal
```

Two design choices make it pleasant to live with. First, the timestamp renders in
the operator's configured `display.timezone`, not UTC — the `stampLayout` uses an
explicit numeric offset so "Z" appears only when the zone really is UTC and every
other zone is unambiguous. When you're reading a log at your desk, wall-clock is
the right clock. Second, the noisy events are deduplicated. A `SiteUpdate` is
republished on *every* status broadcast, which would spam the log with identical
neighbour-site blocks; the formatter remembers the last neighbour set per system
and only writes a block when the adjacent-site list actually changes:

```go
// internal/log/messagelog.go (shape) — formatNeighborBlock
joined := strings.Join(trunking.RenderNeighborLines(su.Topology), "\n")
if m.lastNeighbors[su.System] == joined {
    return "" // unchanged neighbour set — don't re-log it
}
m.lastNeighbors[su.System] = joined
```

Events with no useful textual form return `""` and are skipped, so the message log
stays a curated, readable stream — the trunking story, not the raw firehose.

### How that principle shaped the Go code

- **Rotation is uniform.** All three sinks rotate to `<path>.1` at the same 16 MB
  default (`MaxSizeMB`), closing the current file, renaming, and opening fresh
  under the write lock — so a long-running daemon can't fill a disk with one log.
- **Close drains, bounded.** Each `Close` releases the subscription, then waits on
  the `runDone` channel with a one-second timeout — a clean flush in the normal
  case, but never a hang if the loop is wedged.
- **Timezone is injected, not global.** The daemon passes `cfg.Display.Location()`
  so displayed timestamps match the TUI, the API payloads, and each other; a nil
  location falls back to `time.Local`.
- **The event log stays neutral.** It formats nothing — it only encodes — so it
  can carry event kinds the message log has no opinion about, which is exactly why
  it's the right thing to hand to a tool.

## The power log

The power log is the RF-triage log, and its cleverness is entirely in *when* it
writes. The wideband engine emits a `KindChannelPower` event per diagnostics
window for each channel that decoded at least one frame that window — so idle and
off-band channels never generate events at all. On top of that gating, the power
log by default writes *only* the low-power windows:

```go
// internal/log/powerlog.go (shape) — Run
if ev.Kind != events.KindChannelPower {
    continue
}
cp, ok := ev.Payload.(events.ChannelPower)
if !ok {
    continue
}
if !p.allWindows && !cp.LowPower {
    continue // default: only the "decoding but weak" windows
}
p.write(formatChannelPower(ev.Timestamp, cp, p.loc))
```

A line looks like:

```
2026-06-25T14:07:41.500-05:00 POWER        system=Metro proto=p25 freq=851287500 dbfs=-38.4 decoded=3 low=true
```

This is the "why is a channel that *is* decoding still marginal" diagnostic — the
branch the log is named for. A channel that's decoding cleanly at a healthy level
generates no line; a channel that's decoding but sitting near the floor
(`decoded=3 low=true`) writes one, and now you have a timestamped record of the
weak windows to correlate against gain changes, weather, or time of day. Flip
`AllWindows` on and it becomes a full per-window level time-series for deeper
profiling. Either way it's decode-gated, so it never floods with channels that
aren't doing anything — the opposite instinct from the event log's completeness,
and the right one for its job.

## Panic recovery

All three logs are only useful if they're still being written. The failure that
haunts an unattended service is the one from Part 1: a goroutine panics, the
process dies with a bare stderr stack, and the last thing in every log is a
half-written line. `log.Recover` is the guard against that, installed on every
goroutine the daemon spawns:

```go
// internal/log/recover.go (shape)
func Recover(logger *slog.Logger, component string, onPanic func(err error)) {
    r := recover()
    if r == nil {
        return
    }
    err, ok := r.(error)
    if !ok {
        err = fmt.Errorf("panic: %v", r)
    }
    logger.Error("goroutine panic recovered",
        "component", component, "panic", r, "stack", string(debug.Stack()))
    if onPanic != nil {
        onPanic(err) // daemon escalates: record fatal, cancel run ctx, clean shutdown
    }
}
```

The subtlety is `onPanic`. When it's non-nil — as the daemon wires it — a panic
becomes a *logged* fatal that cancels the run context for a clean, orderly
shutdown, so the other logs get their final flush. When it's nil, the goroutine
simply unwinds: its own deferred cleanups still run, so a panicked IQ/stream
worker closes its output channel and surfaces downstream as a recoverable
"stream closed" event rather than a silent process death. Either path, the panic
value and the goroutine stack land in the log first. That's the difference between
a 3am incident you can reconstruct and one where the evidence died with the
process.

## Where this goes next

Logs and metrics are what a *running* daemon produces. But some failures happen
before the daemon is even up, or belong to the host rather than the daemon — a
missing library, a dongle bound to the wrong kernel driver, a config path that
resolves nowhere. [Part 6]({{ '/blog/deep-dives/running-it-for-real-06-diagnostics-reporter/' | relative_url }})
is the diagnostics reporter and the boot banner: the host/SDR snapshot GopherTrunk
prepends to every error so whoever's triaging has the macro context without asking
the operator to run a separate command. The [status]({{ '/status.html' | relative_url }})
reference covers the health surfaces these logs complement.

## FAQ

**Which log do I hand to support?**
The event log. It's the complete JSONL record of every bus event in the same
format the live streams use, so it's replayable and diffable. The message log is
for your own eyes; the power log is for RF triage.

**Won't three logs plus the event log fill my disk?**
Each rotates to `<path>.1` at a 16 MB default cap, so each log family is bounded at
roughly twice that. The power log is decode-gated and the message log skips
textless events, so both stay far smaller than the raw event log in practice.

**Why is my power log empty?**
Because nothing is decoding weakly. By default it only writes windows where a
channel decoded a frame *and* the level was low — a healthy, clean channel
produces no lines. Enable `AllWindows` for a full per-window time-series.

**What happens to the logs when a goroutine panics?**
`log.Recover` writes the panic value and stack to the log, then (as the daemon
wires it) triggers a clean shutdown that lets the other sinks flush. The timeline
survives the crash instead of ending mid-line.

**Do log timestamps match the API and TUI?**
Yes — displayed timestamps in the message and power logs render in the operator's
`display.timezone`, the same location the TUI and opt-in API payloads use. The
event log carries the raw event timestamp for machine consumption.

## Series navigation

**Part 5 of 14** · ←
[Part 4: Metrics That Matter — Prometheus & SDR Tiles]({{ '/blog/deep-dives/running-it-for-real-04-metrics-that-matter/' | relative_url }})
· Next →
[Part 6: The Diagnostics Reporter]({{ '/blog/deep-dives/running-it-for-real-06-diagnostics-reporter/' | relative_url }})
