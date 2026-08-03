---
title: "The Hunt, Part 6: Control-Channel Hunting — The Supervisor"
description: How GopherTrunk's cchunt supervisor multiplexes per-system control-channel hunts over one shared SDR, dwelling on candidates, parking on locks, backing off failures exponentially, and publishing HuntProgress so the cockpit can watch every system at once.
category: deep-dives
keywords: control channel hunting, cchunt supervisor, round robin scanning, exponential backoff, hunt progress event, dwell and park, cc locked lost, streaming monitor, gophertrunk the hunt
tags: [the-hunt, trunking, scanner, supervisor, events, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Hunt"
series_part: 6
---

*Part 6 of **The Hunt**, a 14-part deep dive into how GopherTrunk finds trunked
systems you didn't know were there. [Part 5]({{ '/blog/deep-dives/the-hunt-05-autogain-autotune/' | relative_url }})
settled the front end so a candidate control channel gets the cleanest signal the
radio can give. Now we have candidate control frequencies — including the one under
our 851 MHz carrier — and one SDR. This part is the loop that dwells on each
candidate long enough to lock it, parks the ones that succeed, backs off the ones
that don't, and tells the cockpit what it's doing the whole time.*

> **TL;DR:** The `cchunt.Supervisor` multiplexes many systems' control-channel
> hunts over **one shared control SDR**. It walks the systems **round-robin**, runs
> a per-system `Hunter` that dwells on each candidate CC, and reacts to bus events:
> a `cc.locked` **parks** the system (stop re-hunting something that works), a
> `cc.lost` un-parks it, and a failed round **backs off exponentially**. It never
> produces `cc.locked` itself — the IQ-domain decoders do; the supervisor only
> orchestrates and publishes state. A separate **streaming monitor** exists for the
> long dwell needed to settle a system's full identity without recording gigabytes.

**Key takeaways**

- **One SDR, many systems, round-robin.** The supervisor owns a single control
  tuner and time-shares it across every configured system in a stable order, so a
  daemon watches a whole county's systems on one dongle.
- **Lock means park, not re-hunt.** Once a `cc.locked` edge lands, the system is
  parked until `cc.lost` — re-hunting a locked edge-triggered decoder would exhaust
  the dwell and falsely report failure while it's still decoding calls.
- **Failure backs off, and explains itself.** A round with no lock doubles the
  system's backoff window (capped) and publishes `KindHuntFailed` enriched with an
  IQ-health diagnosis, so "constantly failing" becomes a self-triaging log line.
- **State is observable, not internal.** Every retune updates a `HuntProgress`, and
  `Snapshot` renders per-system status for the REST cockpit and TUI.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Supervisor loop | round-robin hunt over one SDR | `internal/scanner/cchunt/supervisor.go` (`Run`) |
| Per-system state | hunting / locked / failed / held | `internal/scanner/cchunt/types.go` (`HuntState`) |
| Event listener | drain cc.locked/lost/grant/progress | `internal/scanner/cchunt/supervisor.go` (`listen`) |
| Failure + backoff | double the window, diagnose IQ | `internal/scanner/cchunt/supervisor.go` (`markFailed`) |
| Offline live hunt | sweep → identify → map on air | `internal/hunt/livehunt.go` (`RunLiveHunt`) |
| Streaming monitor | long dwell, bounded memory | `internal/hunt/monitor.go` (`monitorCandidate`) |

## In this post

- **The orchestration boundary** — what the supervisor owns and what it refuses to.
- **The round-robin loop** — dwell, park, back off, advance.
- **Reacting to the bus** — how lock/lost/grant/progress drive the state machine.
- **When failure explains itself** — backoff plus the IQ-health diagnosis.
- **The streaming monitor** — the long dwell without the gigabytes.

## The orchestration boundary

The single most important thing to understand about the supervisor is what it
*doesn't* do. It does not decode. It does not produce `cc.locked` events. It is pure
orchestration:

```go
// internal/scanner/cchunt/types.go (shape) — package doc
// The supervisor owns one control SDR and multiplexes trunking.Hunter runs across
// every configured trunked system. On success a system parks until cc.lost; on
// failure the supervisor exponentially backs off, publishes KindHuntFailed, and
// advances to the next system.
//
// This package is the orchestration layer only. It does not produce cc.locked
// events itself — those must come from the IQ-domain protocol decoders. Without
// those upstream feeders the supervisor will always exhaust the candidate list and
// report failed; that's intentional, not a bug.
```

That boundary is the same decoupling the rest of GopherTrunk leans on: the
supervisor tunes the SDR and listens to the bus, while the decoder (`ccdecoder`,
from the Protocol Decoders series) watches the IQ and publishes `cc.locked` when a
control channel actually locks. The two never call each other directly. The
supervisor's `Tuner` is a one-method interface (`SetCenterFreq`) so tests substitute
a fake, and its optional `IQHealthProvider` hook is likewise an interface, so the
supervisor can *ask* the decoder "did you see any IQ?" on failure without importing
it and creating a cycle.

## The round-robin loop

`Run` blocks until its context cancels and walks the configured systems in a stable
round-robin. Each iteration picks the next system and decides what to do with it —
and the *order* of those checks is the state machine:

```go
// internal/scanner/cchunt/supervisor.go (shape) — Run per-iteration guards
name := s.order[cursor%len(s.order)]
cursor++
// Skip held systems — the operator owns their state.
if s.isHeld(name) { s.waitOrSleep(ctx, 500*time.Millisecond); continue }
// Skip systems still in backoff.
if remaining := s.backoffRemaining(name); remaining > 0 { /* sleep, continue */ }
// Park an already-locked system instead of re-hunting it.
if s.isLocked(name) { s.parkUntilUnlocked(ctx, name); continue }
// Otherwise: hunt it.
s.startRound(name)
hunter, _ := trunking.NewHunter(trunking.HunterOptions{ System: rt.sys, Tuner: s.currentTuner(), Bus: s.bus, /* … */ Dwell: s.dwell })
_, herr := hunter.Hunt(hctx)
```

The park check is subtler than it looks and was hard-won. A `cc.locked` edge can
land *after* a hunt round already failed and backed off — a TETRA control channel
whose cold acquisition (BSCH sync + colour code) took longer than the dwell. Without
the `isLocked` park, the loop would re-hunt a system it already knows is locked, and
because an edge-triggered decoder never re-emits `cc.locked` on the same locked
pipeline, every such re-hunt exhausts the dwell and reports "candidates exhausted"
while the decoder is happily decoding calls. So a locked system is *parked* —
`parkUntilUnlocked` waits until the state leaves `StateLocked`, which the protocol's
own lock-loss watchdog (`cc.lost`) drives on real loss.

When a hunt does run, it can be interrupted: the supervisor arms a retune channel so
an operator forcing a retune (or a `SwapTuner` after a USB reacquisition) cancels the
in-flight `Hunter` and the loop advances immediately.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="The per-system hunt state machine: idle transitions to hunting, which on a cc.locked event becomes locked and parks until cc.lost returns it to hunting; a hunting round that exhausts without a lock becomes failed and backs off before returning to hunting; an operator hold pins any state">
  <rect x="30" y="90" width="90" height="40" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="75" y="114" text-anchor="middle" fill="var(--fg-muted)" font-size="10">idle</text>
  <line x1="120" y1="110" x2="160" y2="110" stroke="currentColor"/><polygon points="160,106 170,110 160,114" fill="currentColor"/>
  <rect x="170" y="90" width="100" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="220" y="114" text-anchor="middle" fill="var(--accent)" font-size="10">hunting</text>
  <line x1="270" y1="102" x2="340" y2="70" stroke="currentColor"/><polygon points="337,66 347,66 341,75" fill="currentColor"/>
  <text x="300" y="78" fill="var(--fg-muted)" font-size="9">cc.locked</text>
  <rect x="348" y="50" width="100" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="398" y="70" text-anchor="middle" fill="var(--accent)" font-size="10">locked</text>
  <text x="398" y="83" text-anchor="middle" fill="var(--fg-muted)" font-size="9">parked</text>
  <line x1="360" y1="90" x2="255" y2="120" stroke="currentColor" stroke-dasharray="3 3"/><polygon points="260,116 250,122 262,124" fill="currentColor"/>
  <text x="300" y="128" fill="var(--fg-muted)" font-size="9">cc.lost</text>
  <line x1="270" y1="120" x2="340" y2="155" stroke="currentColor"/><polygon points="337,150 347,155 336,159" fill="currentColor"/>
  <text x="290" y="150" fill="var(--fg-muted)" font-size="9">exhausted</text>
  <rect x="348" y="140" width="120" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="408" y="160" text-anchor="middle" fill="currentColor" font-size="10">failed</text>
  <text x="408" y="173" text-anchor="middle" fill="var(--fg-muted)" font-size="9">backoff ×2</text>
  <line x1="440" y1="140" x2="255" y2="126" stroke="currentColor" stroke-dasharray="3 3"/><polygon points="260,122 250,128 261,131" fill="currentColor"/>
  <text x="500" y="120" fill="var(--fg-muted)" font-size="9">held: operator pins any state (PauseAll/Hold)</text>
  <text x="340" y="202" text-anchor="middle" fill="var(--fg-muted)" font-size="10">one SDR, round-robin across systems; the decoders (not the supervisor) emit cc.locked</text>
</svg>
<figcaption>Per-system states over one shared SDR. Lock parks; loss re-hunts; exhaustion backs off. The supervisor drives transitions from bus events it doesn't itself produce.</figcaption>
</figure>

## Reacting to the bus

A dedicated `listen` goroutine drains the bus so the hunt loop never has to multiplex
between hunting and listening on the same channel. It translates four event kinds
into state transitions:

```go
// internal/scanner/cchunt/supervisor.go (shape) — listen dispatch
switch ev.Kind {
case events.KindCCLocked:
    lp, _ := ev.Payload.(trunking.LockedPayload)
    s.recordLock(lp.LockedFrequencyHz(), lp.LockedNAC())
case events.KindCCLost:
    s.recordLost(freq)
case events.KindGrant:
    g, _ := ev.Payload.(trunking.Grant)
    s.recordGrant(g.System, ev.Timestamp)
case events.KindHuntProgress:
    p, _ := ev.Payload.(trunking.HuntProgress)
    s.recordProgress(p)
}
```

`recordLock` matches the locked frequency against every system's configured control
channels, flips the owner to `StateLocked`, records the NAC and timestamp, and resets
its backoff window. `recordProgress` is how the cockpit sees *live* activity: every
retune the `Hunter` performs publishes a `HuntProgress` (which candidate, index N of
M), and the supervisor stores the latest per system. That progress surfaces through
`Snapshot`, the concurrent-safe status read the REST handler and TUI poll:

```go
// internal/scanner/cchunt/supervisor.go (shape) — Snapshot row
st := SystemStatus{
    Name:            rt.sys.Name,
    State:           rt.state,
    AttemptedFreqHz: rt.progress.AttemptedFreqHz,
    AttemptIndex:    rt.progress.AttemptIndex,
    TotalCandidates: rt.progress.TotalCandidates,
    LockedFreqHz:    rt.lockedFreqHz,
    NAC:             rt.nac,
    // …LastFailedAt, LastGrantAt, BackoffMs when failed
}
```

So a cockpit can render "System X: hunting, candidate 2 of 5, 851.0125 MHz" while it
happens, and "System Y: failed, retry in 5 s" — all from one shared SDR.

### How that principle shaped the Go code

- **The tuner is an interface.** `Tuner` is just `SetCenterFreq`, so the supervisor
  is testable against a fake and its live handle can be hot-swapped (`SwapTuner`)
  after a USB reacquisition without a process restart or a race with the hunt loop.
- **Mutation surface is explicit.** `Hold`/`Resume`, `PauseAll`/`ResumeAll`, and
  `ForceRetune` are the *only* ways an operator perturbs the loop, each lock-guarded.
  `PauseAll` is how the live-hunt sweep borrows the control SDR: quiesce cchunt, sweep,
  then `ResumeAll`.
- **State lives in `systemRuntime`, guarded by one mutex.** Every accessor takes
  `s.mu`; the loop, the listener, and `Snapshot` all read the same guarded state, so
  there's exactly one source of truth for "what is this system doing right now."

## When failure explains itself

A round that exhausts its candidates without a lock isn't just marked failed — it's
*diagnosed*. `markFailed` doubles the backoff window (capped at `MaxBackoff`), then,
outside the lock, asks the optional IQ-health provider why:

```go
// internal/scanner/cchunt/supervisor.go (shape) — markFailed diagnosis
diag := diagnoseFailure(provider, name) // queried OUTSIDE s.mu — foreign lock
s.log.Warn("cchunt: hunt failed — no control-channel lock",
    "system", name, "backoff_ms", int(wait/time.Millisecond),
    "iq_observed", diag.IQObserved, "iq_power_dbfs", diag.IQPowerDbFS,
    "iq_clip_ratio", diag.IQClipRatio, "pipeline_active", diag.PipelineActive,
    "diagnosis", diag.Diagnosis)
s.bus.Publish(events.Event{Kind: events.KindHuntFailed, Payload: trunking.HuntFailed{ /* … */ Diagnostics: diag }})
```

The single most common failure mode — the control SDR delivering *no* IQ at all
(USB unbound, dead handle, nothing tuned) — is exactly the one that's hardest to
diagnose from "hunt failed," so when the decoder reports no observations, that
absence *is* the diagnosis: a one-line pointer to `sdr list --probe`, `sdr doctor`,
and the antenna. The WARN is the line operators paste into bug reports, and the
structured IQ fields turn "constantly getting cchunt.failed" into a self-triaging log
entry. The diagnosis query deliberately runs outside `s.mu`, because the provider (the
decoder) takes its own lock and the supervisor must never hold its mutex across a
foreign call.

## The streaming monitor

The supervisor's hunt is a *lock* problem — get a control channel decoding. But
Part 7's job, settling a system's full identity (WACN, neighbours, band plan),
needs a much *longer* dwell, and buffering minutes of IQ would be gigabytes. That's
what `monitorCandidate` in the hunt package solves: it streams live IQ straight into
the decoder and stops early once the topology settles:

```go
// internal/hunt/monitor.go (shape) — streaming dwell with convergence
dwellCtx, cancel := context.WithTimeout(ctx, time.Duration(monitorSeconds*float64(time.Second)))
live := &streamReader{ctx: dwellCtx, src: src, chunk: prefixSamples, format: p.Format}
r := io.MultiReader(bytes.NewReader(prefixBytes), live)
res, err := siglab.RunReaderMonitor(r, source, cfg, monitorTick, convergeAndStop())
```

The `streamReader` adapts an `IQSource` into an `io.Reader` of encoded samples,
serving live capture chunks and returning `io.EOF` when the dwell cap hits — so the
decoder assembles whatever topology it accumulated. `convergeAndStop` watches the
neighbour count, secondary CCs, and band-plan slots; once identity is present and
those stop growing for `monitorStability` (30 s) past a `monitorMinDwell` floor
(20 s), it declares the picture complete and stops early. Bounded memory, and no
watching a converged system for the full cap when it settled in the first minute.
This is the on-air path `RunLiveHunt` dispatches to when `MonitorSeconds > 0`; the
buffered fixed-dwell path handles the short 3-second probes.

## Where this goes next

The supervisor gets a control channel *locked*. [Part 7]({{ '/blog/deep-dives/the-hunt-07-locking-a-p25-system/' | relative_url }})
is what turns that lock into a *confirmed system* — folding the decoded WACN and
System ID into an identity, recovering the band plan, and handling the honest case
where a P25 system locks but never broadcasts the message that carries its WACN.
Our 851 MHz carrier is about to become a named entry in a `DiscoveredSystem`.

## FAQ

**Why doesn't the supervisor decode the control channel itself?**
Separation of concerns. Decoding is the IQ-domain protocol decoders' job (the
Protocol Decoders series); the supervisor only tunes the SDR and reacts to the
`cc.locked`/`cc.lost` events those decoders publish. The decoupling means the
supervisor is small, testable against fakes, and never duplicates decode logic.

**What happens if a system never locks?**
Each failed round doubles its backoff window (up to `MaxBackoff`, default 60 s), so
a dead system is retried less and less often instead of hammering the SDR — while
still being retried, in case it was a transient. The failure publishes
`KindHuntFailed` with an IQ-health diagnosis explaining the likely cause.

**Why park a locked system instead of confirming it every round?**
Edge-triggered decoders (like TETRA) emit `cc.locked` once and then just decode; they
don't re-emit it on an already-locked pipeline. Re-hunting such a system would
exhaust the dwell and falsely report failure. Parking until `cc.lost` respects how
the decoders actually signal state.

**How does the cockpit know what's happening?**
Two channels. Live, the `Hunter` publishes a `HuntProgress` on every retune, which
the supervisor stores and `Snapshot` exposes (attempted frequency, candidate index,
total). And failures publish `KindHuntFailed`. The REST cockpit and TUI poll
`Snapshot`; the bus events drive anything subscribed.

**What's the difference between the supervisor's hunt and the streaming monitor?**
The supervisor's per-system `Hunter` is a short-dwell *lock* loop over one shared SDR
for live scanning. The streaming monitor (`monitorCandidate`) is a long-dwell,
bounded-memory *identity settling* pass used by an offline/deliberate live hunt — it
streams rather than buffers and stops early on convergence.

## Series navigation

**Part 6 of 14** · ←
[Part 5: Autogain & Autotune — Settling the Front End]({{ '/blog/deep-dives/the-hunt-05-autogain-autotune/' | relative_url }})
· Next →
[Part 7: Locking a P25 System — Candidate to Confirmed]({{ '/blog/deep-dives/the-hunt-07-locking-a-p25-system/' | relative_url }})
