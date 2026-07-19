---
title: "Trunking Engine, Part 12: Control-Channel Hunting, the Call Watchdog & Testing With a Fake Bus"
description: How GopherTrunk finds a control channel and backs off when it can't, how a 500 ms watchdog reaps calls that go silent, and how the whole engine is tested by publishing synthetic grants to a fake bus with no radio attached.
category: deep-dives
keywords: control channel hunting, cc lock, hunt backoff, call watchdog, end reason timeout, end reason normal, fake event bus testing, synthetic grant, no radio testing, gophertrunk trunking
tags: [trunking, go, event-bus, testing, watchdog, architecture]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Trunking Engine"
series_part: 12
---

*Part 12 of **Trunking Engine**, the finale. Eleven parts followed a grant from
the control channel into a recorded call and out to every subscriber. This one
covers the three things that make the whole machine dependable: how it finds a
control channel to begin with, how it decides a call is over when nothing says so,
and how all of it is tested without a radio.*

> **TL;DR:** Before any grant exists, the **hunter** scans a system's candidate
> control-channel frequencies, parks on the first that produces a `cc.locked`
> event within a dwell window, and caches the winner for next time; when the list
> exhausts it returns `ErrNoControlChannel` and the supervisor backs off. Once
> calls are running, a **500 ms watchdog** tick reaps any call with no recent
> frames — `EndReasonNormal` if it ever decoded (P25's only end-of-call signal is
> silence), `EndReasonTimeout` if it never did. And because the engine's only
> input is a channel of events, the **entire thing is tested with a fake bus**:
> publish a synthetic `Grant`, assert a `KindCallStart` comes back. No SDR.

**Key takeaways**

- **Hunting is search with memory**: the cache biases the hunt toward the
  last-locked CC, so a restart re-locks fast instead of rescanning the whole list.
- The watchdog is the engine's clock for **call teardown** — P25 has no explicit
  channel-release, so a graceful timeout *is* the end-of-call mechanism.
- The `EndReasonNormal` vs `EndReasonTimeout` distinction is a real bug fix:
  healthy calls were being reported as timeouts and scaring operators (#356).
- The one-way "publish, never call outward" rule from Part 1 pays off here: the
  engine is fully testable by driving a fake bus and asserting emitted events.

## Cheat sheet

| Concept | Where it lives | One-line role |
|---|---|---|
| `Hunter.Hunt(ctx)` | `cchunt.go` | scan candidate CCs, park on the first lock, cache it |
| `ErrNoControlChannel` | `cchunt.go` | every candidate exhausted its dwell — triggers backoff |
| `HuntProgress` / `HuntFailed` | `cchunt.go` | telemetry events; `HuntFailed.BackoffMs` is the next sleep |
| `Cache` | `cache.go` | per-system last-known CC frequency, persisted atomically |
| `runWatchdog()` | `engine.go` | 500 ms tick; reaps silent calls, releases encrypted holds |
| `EndReasonNormal` / `EndReasonTimeout` | `grant.go` | decoded-then-dropped vs never-decoded teardown |

## In this post

- **Finding the control channel** — hunting, dwell, and the cache that remembers.
- **Backoff** — what happens when nothing locks.
- **The watchdog** — reaping silent calls and the two end reasons.
- **Testing with a fake bus** — driving the engine with synthetic grants, no
  radio. Then we wrap the series.

## Finding the control channel

Everything in this series started with a grant, but a grant only exists once the
receiver is locked to a control channel. Getting there is the **hunter's** job. A
`System` carries a list of candidate CC frequencies; the hunter tunes each in turn
and waits up to a **dwell** window (default 3 s) for the demod pipeline to publish
a `cc.locked` event on that frequency. The first match wins — the hunter parks
there and returns.

The hunter is deliberately protocol-agnostic at the wiring level. It retunes an
SDR and watches the bus; it does not import a single radio package. It talks to
two tiny interfaces — a `Tuner` (just `SetCenterFreq`) and a `LockedPayload`
(`LockedFrequencyHz` + `LockedNAC`) — so each protocol's lock state satisfies them
by method, and the hunter stays out of the import graph that would otherwise cycle
back through the `trunking` package the radios publish `Grant`s to.

```go
// internal/trunking/cchunt.go (shape)
func (h *Hunter) Hunt(ctx context.Context) (LockResult, error) {
    lastKnown := h.cachedFrequency()          // bias toward the last lock
    candidates := h.system.HuntOrder(lastKnown)
    sub := h.bus.Subscribe()
    defer sub.Close()
    for i, freq := range candidates {
        h.publishProgress(freq, i, len(candidates)) // TUI: "trying 2/3"
        if err := h.tuner.SetCenterFreq(freq); err != nil {
            continue
        }
        drainSubscription(sub) // discard stale events from the prior candidate
        if locked, ok := waitForLock(ctx, sub, time.After(h.dwell), freq); ok {
            h.cache.Set(h.system.Name, /* freq + NAC + timestamp */)
            return LockResult{System: h.system.Name, Frequency: freq, NAC: locked.LockedNAC()}, nil
        }
    }
    return LockResult{}, ErrNoControlChannel
}
```

The **cache** is what makes a restart fast. `HuntOrder(lastKnown)` moves the
last-locked frequency to the front of the candidate list, so after a daemon
restart the hunter re-locks on the first try instead of rescanning. The cache is a
small per-system JSON file written atomically via a temp-file rename, so a crash
mid-write can never corrupt it. On every successful lock the winning frequency,
NAC, and timestamp are persisted for next time.

<figure class="lab-figure">
<svg viewBox="0 0 680 175" width="680" height="175" role="img" aria-label="The hunter tries each candidate control channel in turn, publishing progress; a lock within the dwell window parks and caches the frequency, while exhausting the list returns no-control-channel and the supervisor backs off before retrying">
  <rect x="14" y="20" width="120" height="30" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="74" y="39" text-anchor="middle" fill="var(--fg-muted)" font-size="10">cache: last CC</text>
  <line x1="74" y1="50" x2="74" y2="72" stroke="var(--fg-muted)"/>
  <polygon points="70,72 74,82 78,72" fill="var(--fg-muted)"/>
  <rect x="14" y="82" width="120" height="34" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="74" y="103" text-anchor="middle" fill="var(--accent)" font-size="10">HuntOrder()</text>
  <line x1="134" y1="99" x2="180" y2="99" stroke="currentColor"/>
  <polygon points="180,95 190,99 180,103" fill="currentColor"/>
  <rect x="190" y="70" width="130" height="58" rx="6" fill="none" stroke="currentColor"/>
  <text x="255" y="92" text-anchor="middle" fill="currentColor" font-size="11">try freq[i]</text>
  <text x="255" y="108" text-anchor="middle" fill="var(--fg-muted)" font-size="9">tune + wait dwell</text>
  <text x="255" y="121" text-anchor="middle" fill="var(--fg-muted)" font-size="9">for cc.locked</text>
  <line x1="320" y1="84" x2="366" y2="60" stroke="var(--accent)"/>
  <polygon points="366,56 376,60 365,64" fill="var(--accent)"/>
  <text x="345" y="52" text-anchor="middle" fill="var(--accent)" font-size="9">lock</text>
  <rect x="376" y="42" width="150" height="34" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="451" y="58" text-anchor="middle" fill="var(--accent)" font-size="10">park + cache winner</text>
  <text x="451" y="70" text-anchor="middle" fill="var(--fg-muted)" font-size="9">grants start flowing</text>
  <line x1="255" y1="128" x2="255" y2="150" stroke="currentColor"/>
  <polygon points="251,150 255,160 259,150" fill="currentColor"/>
  <text x="255" y="145" text-anchor="middle" fill="var(--fg-muted)" font-size="9">no lock → next candidate</text>
  <line x1="320" y1="112" x2="366" y2="130" stroke="var(--fg-muted)"/>
  <polygon points="366,126 376,131 365,134" fill="var(--fg-muted)"/>
  <text x="345" y="122" text-anchor="middle" fill="var(--fg-muted)" font-size="9">list done</text>
  <rect x="376" y="112" width="180" height="36" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="466" y="128" text-anchor="middle" fill="currentColor" font-size="10">ErrNoControlChannel</text>
  <text x="466" y="141" text-anchor="middle" fill="var(--fg-muted)" font-size="9">HuntFailed{BackoffMs} → sleep, retry</text>
</svg>
<figcaption>Hunting is a search biased by the cache. A lock parks and persists; an exhausted list returns a sentinel that drives the supervisor's backoff.</figcaption>
</figure>

## When nothing locks

Sometimes no candidate locks — the antenna is unplugged, the system is off the
air, the gain is wrong. `Hunt` returns `ErrNoControlChannel`, and the supervisor
publishes a `HuntFailed` event whose `BackoffMs` field is the next sleep window,
so the TUI can render "retry in 5 s" without scraping logs. Backoff keeps a dead
system from pegging the CPU in a tight rescan loop while still recovering
automatically when it comes back.

`HuntFailed` also carries a best-effort `HuntDiagnostics` snapshot filled from the
live control-channel decoder, so the failure answers the operator's first question
instead of forcing a round-trip. It distinguishes a **silent SDR** (no IQ observed
— USB/driver binding, a dead handle, no antenna) from **front-end overload**
(power near 0 dBFS with a high clip ratio), an **on-channel DC spike**, or a clean
signal that simply never carried a decodable control channel — and boils it down to
a one-line human-readable `Diagnosis`.

## The watchdog

Once calls are running, the engine's other clock takes over. P25 trunking has **no
explicit channel-release message** on the control channel for most calls — the
system just stops repeating the grant and the voice carrier drops. So the only way
to know a call ended is to notice it went quiet. That's the **watchdog**: the
500 ms ticker in the engine's `select` loop (from
[Part 1]({{ '/blog/deep-dives/trunking-engine-01-grant-to-call/' | relative_url }})).

Each tick, `runWatchdog` does three things in order. It releases any encrypted
call whose metadata-follow window has elapsed (the machinery from
[Part 11]({{ '/blog/deep-dives/trunking-engine-11-encrypted-mode/' | relative_url }}),
done first so an encrypted release isn't double-counted as an inactivity timeout).
It reaps any active call whose `LastHeardAt` is older than the call timeout
(default 30 s). And it ages out observed-only calls the control channel has stopped
repeating.

The subtle part is the **end reason**, and it comes straight from a field bug
(#356). A call that went silent could mean two very different things, and reporting
them the same way misled operators into thinking a working decode was broken:

```go
// internal/trunking/engine.go (shape) — inside runWatchdog
if ac.LastHeardAt.Before(cutoff) {
    reason := EndReasonTimeout          // never decoded a single frame
    if ac.LastHeardAt.After(ac.StartedAt) {
        reason = EndReasonNormal        // decoded, then the carrier dropped
    }
    e.endCall(ac, reason)
}
```

If `LastHeardAt` ever advanced past `StartedAt`, the call *received frames* and
then the carrier dropped — that's the natural P25 end-of-call, `EndReasonNormal`.
If `LastHeardAt` never moved off `StartedAt`, the call never decoded a single frame
— a genuine silent failure (wrong band plan, an LSM site decoded as C4FM), and that
is the real `EndReasonTimeout` worth surfacing. Before this split, three healthy
calls in a field log all reported `reason=timeout`, and the operator concluded the
decode was still broken when it was only a terminology problem.

`endCall` is the single teardown path: release the device back to the pool, delete
the call from the map, and publish `KindCallEnd` with the reason and the call's
final signal metrics. Shutdown reuses it — on `ctx.Done()`, `shutdown` walks every
active call and ends it `EndReasonNormal`, so subscribers see clean `CallEnd`
events even on a hard stop.

## Testing the whole engine with a fake bus

Here is where the design principle that opened the series — *publish, never call
outward* — cashes out completely. The engine's only input is a channel of events
and its only output is published events. It imports the recorder, the database, and
the UI exactly zero times. So a test needs none of them: construct a real (in-memory)
bus, build the engine on it with a fake voice pool, publish a synthetic `Grant`,
and assert the engine publishes back `KindCallStart`.

```go
// internal/trunking/engine_test.go (shape)
bus := events.NewBus()
eng, _ := NewEngine(EngineOptions{Bus: bus, VoicePool: fakePool, /* ... */})
go eng.Run(ctx)

sub := bus.Subscribe()
bus.Publish(events.Event{Kind: events.KindGrant, Payload: Grant{
    System: "test", GroupID: 1234, FrequencyHz: 851_012_500, SourceID: 0x4A21,
}})
// assert a KindCallStart for tg 1234 arrives on sub within a deadline
```

No SDR, no HTTP server, no `.wav` on disk, no network. Every branch of the engine's
behaviour is reachable this way: publish two grants 20 ms apart and assert only one
device binds (duplicate suppression); publish a `KindCallEncryption` and assert an
`EndReasonEncrypted` `CallEnd` (encrypted policy); publish a source-carrying grant
on an active call's channel and assert the RID is folded (Part 9); drive the fake
clock past the timeout and assert the right end reason. The fake bus *is* the test
harness for the entire subsystem, and it exists only because the engine never
reaches outward — it just reacts and emits.

<figure class="lab-figure">
<svg viewBox="0 0 680 165" width="680" height="165" role="img" aria-label="A test publishes a synthetic grant to an in-memory bus, the engine under test reacts on its select loop and publishes a call-start event, and the test asserts on that emitted event with no radio, recorder, or network involved">
  <rect x="14" y="60" width="130" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="79" y="80" text-anchor="middle" fill="currentColor" font-size="11">test</text>
  <text x="79" y="96" text-anchor="middle" fill="var(--fg-muted)" font-size="9">publish Grant</text>
  <line x1="144" y1="83" x2="196" y2="83" stroke="currentColor"/>
  <polygon points="196,79 206,83 196,87" fill="currentColor"/>
  <rect x="206" y="58" width="120" height="50" rx="6" fill="none" stroke="currentColor"/>
  <text x="266" y="80" text-anchor="middle" fill="currentColor" font-size="11">in-memory bus</text>
  <text x="266" y="96" text-anchor="middle" fill="var(--fg-muted)" font-size="9">events.NewBus()</text>
  <line x1="326" y1="83" x2="378" y2="83" stroke="var(--accent)"/>
  <polygon points="378,79 388,83 378,87" fill="var(--accent)"/>
  <rect x="388" y="52" width="140" height="62" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="458" y="76" text-anchor="middle" fill="var(--accent)" font-size="12">Engine under test</text>
  <text x="458" y="92" text-anchor="middle" fill="var(--fg-muted)" font-size="9">select loop · fake pool</text>
  <text x="458" y="105" text-anchor="middle" fill="var(--fg-muted)" font-size="9">no SDR, no recorder</text>
  <path d="M 458 114 L 458 134 L 266 134 L 266 108" fill="none" stroke="var(--accent)" stroke-dasharray="4 3"/>
  <polygon points="262,116 266,106 270,116" fill="var(--accent)"/>
  <text x="360" y="149" text-anchor="middle" fill="var(--accent)" font-size="10">publishes KindCallStart → test asserts on it</text>
  <text x="590" y="80" text-anchor="middle" fill="var(--fg-muted)" font-size="9">recorder · DB · UI</text>
  <text x="590" y="94" text-anchor="middle" fill="var(--fg-muted)" font-size="9">(not in the test)</text>
  <rect x="540" y="60" width="126" height="46" rx="6" fill="none" stroke="var(--fg-muted)" stroke-dasharray="2 3"/>
</svg>
<figcaption>The engine's only input and output are events, so a test drives it with a fake bus and asserts on what it emits — the whole subsystem tested with no radio in the loop.</figcaption>
</figure>

### How that principle held for twelve parts

Every part of this series was a branch of one `select` loop, and every one obeyed
the same two rules: a **single writer** of call state, so the hot path needs no
locks, and **observer decoupling**, so the engine publishes and never calls
outward. The event bus (Part 2) made the fanout cheap; the voice pool (Part 4) and
priority policy (Part 5) rationed the scarce tuners; the derived-state trackers
(Parts 8–10) bolted on as pure subscribers; the source-RID recovery (Parts 7, 9)
and encrypted-mode policy (Part 11) folded new behaviour into the loop without ever
importing a consumer. And because of that discipline, the whole thing is testable
with a fake bus and a synthetic grant — which is the strongest evidence the
architecture was right.

## Where this goes next

This is the end of **Trunking Engine**. If you want to see the decode side that
produces the grants this engine consumes, that's the
[Protocol Decoders]({{ '/blog/series/protocol-decoders/' | relative_url }})
series; for the DSP and radio plumbing beneath both, see
[SDR Internals]({{ '/blog/series/sdr-internals/' | relative_url }}). Or start over
at [Part 1]({{ '/blog/deep-dives/trunking-engine-01-grant-to-call/' | relative_url }})
and re-read the loop now that you know every branch of the `switch`. The full
series lives on the [Trunking Engine]({{ '/blog/series/trunking-engine/' | relative_url }})
landing page.

## FAQ

**How does GopherTrunk find a control channel?**
The hunter tunes each candidate frequency in a system's list and waits a dwell
window (default 3 s) for a `cc.locked` event on that frequency. The first lock wins
and is cached, so a restart re-locks on the last-known channel first instead of
rescanning the whole list.

**Why does a call end from a timeout instead of a "call over" message?**
Most P25 trunked calls have no explicit channel-release on the control channel — the
carrier just drops. So the engine's watchdog treats a graceful timeout after the
last decoded frame as the natural end of the call. That's why the timeout exists,
and why its default is a generous 30 s.

**What's the difference between EndReasonNormal and EndReasonTimeout?**
`EndReasonNormal` means the call decoded frames and then the carrier dropped — a
healthy end. `EndReasonTimeout` means the call never decoded a single frame — a
silent failure worth investigating. The watchdog picks by whether `LastHeardAt`
ever advanced past `StartedAt`.

**How can the engine be tested without a radio?**
Because its only input is a channel of events and its only output is published
events. A test builds an in-memory bus, publishes a synthetic `Grant`, and asserts
the engine publishes `KindCallStart` back — no SDR, recorder, database, or network.
Every behaviour, including duplicate suppression and encrypted-call policy, is
reachable this way.

## Series navigation

**Part 12 of 12** · ←
[Part 11: Encrypted-Mode Handling]({{ '/blog/deep-dives/trunking-engine-11-encrypted-mode/' | relative_url }})
