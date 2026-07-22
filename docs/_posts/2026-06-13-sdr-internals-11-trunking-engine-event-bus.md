---
title: "SDR in Pure Go, Part 11: The Trunking Engine & Event Bus"
description: How GopherTrunk's trunking engine turns channel grants into recorded calls, and how an in-process pub/sub event bus decouples the core from the API, storage, and streaming subsystems.
category: deep-dives
tags: [sdr, go, trunking, event-bus, pubsub, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "SDR Internals"
series_part: 11
---

*Part 11 of **SDR Internals**. A grant arrives: talkgroup 1234 is on 851.0125
MHz. Something has to tune a radio there, start a recording, and tell the world.
This post is about the trunking engine and the event bus that keeps it
decoupled.*

## In this post

- What the **trunking engine** does: grant handling, voice allocation, call
  watchdog.
- The **in-process event bus** that fans domain events out to subscribers.
- The **observer / event-driven** principle that keeps the engine ignorant of —
  and testable without — the API, storage, and UI.

## What the engine and bus do

The decoders from
[Part 10]({{ '/blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/' | relative_url }})
emit grants. The **trunking engine** (`internal/trunking`) acts on them: it asks
the SDR pool for a voice device, retunes it to the granted frequency, starts a
call, and runs a watchdog that reaps calls that have gone silent. It also handles
priority and preemption when more calls are active than there are radios.

But many other parts of the system care about these events too — the recorder,
the web UI, the call-log database, the Prometheus metrics, the Broadcastify
uploader. Rather than have the engine call each of them, it publishes to an
**event bus** (`internal/events`) and they subscribe.

<figure class="lab-figure">
<svg viewBox="0 0 660 190" width="660" height="190" role="img" aria-label="A protocol decoder publishes a grant onto the in-process event bus; the bus fans domain events out to five independent subscribers — the trunking engine, the recorder, the SQLite call log, the web UI over WebSocket, and the Broadcastify uploader — none of which the engine ever calls directly.">
  <rect x="8" y="76" width="112" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="64" y="96" text-anchor="middle" fill="var(--accent)" font-size="11">decoder</text>
  <text x="64" y="111" text-anchor="middle" fill="var(--fg-muted)" font-size="9">publishes grant</text>
  <line x1="120" y1="98" x2="158" y2="98" stroke="currentColor"/><polygon points="158,94 168,98 158,102" fill="currentColor"/>
  <rect x="168" y="66" width="98" height="64" rx="6" fill="none" stroke="currentColor"/>
  <text x="217" y="90" text-anchor="middle" fill="currentColor" font-size="11">event bus</text>
  <text x="217" y="105" text-anchor="middle" fill="var(--fg-muted)" font-size="9">internal/events</text>
  <text x="217" y="118" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Kind + payload</text>
  <line x1="266" y1="90" x2="300" y2="26" stroke="currentColor"/><polygon points="296,22 306,24 300,33" fill="currentColor"/>
  <line x1="266" y1="94" x2="300" y2="62" stroke="currentColor"/><polygon points="296,58 306,60 299,69" fill="currentColor"/>
  <line x1="266" y1="98" x2="300" y2="98" stroke="currentColor"/><polygon points="300,94 310,98 300,102" fill="currentColor"/>
  <line x1="266" y1="102" x2="300" y2="134" stroke="currentColor"/><polygon points="299,127 306,136 296,138" fill="currentColor"/>
  <line x1="266" y1="106" x2="300" y2="170" stroke="currentColor"/><polygon points="298,163 306,172 294,172" fill="currentColor"/>
  <rect x="310" y="12" width="180" height="28" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="400" y="30" text-anchor="middle" fill="var(--accent)" font-size="10">engine · HandleGrant</text>
  <rect x="310" y="48" width="180" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="400" y="66" text-anchor="middle" fill="currentColor" font-size="10">recorder</text>
  <rect x="310" y="84" width="180" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="400" y="102" text-anchor="middle" fill="currentColor" font-size="10">call log · SQLite</text>
  <rect x="310" y="120" width="180" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="400" y="138" text-anchor="middle" fill="currentColor" font-size="10">web UI · WebSocket</text>
  <rect x="310" y="156" width="180" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="400" y="174" text-anchor="middle" fill="currentColor" font-size="10">Broadcastify uploader</text>
</svg>
<figcaption>A grant published by the decoder fans out through the bus; the engine, recorder, call log, web UI, and uploader each subscribe independently — the engine never calls them.</figcaption>
</figure>

## How GopherTrunk implements it in Go

The engine is a single goroutine running a `select` loop — the classic Go event
loop — draining the bus and ticking a watchdog:

```go
// internal/trunking/engine.go (shape)
func (e *Engine) Run(ctx context.Context) error {
    tick := time.NewTicker(500 * time.Millisecond)
    defer tick.Stop()
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case ev := <-e.sub.C:
            if g, ok := ev.Payload.(Grant); ok && ev.Kind == events.KindGrant {
                e.HandleGrant(g) // allocate a voice device, start a call
            }
        case <-tick.C:
            e.runWatchdog()      // reap calls with no recent frames
        }
    }
}
```

The bus is a typed pub/sub fanout. Events are tagged with a `Kind` —
`KindCCLocked`, `KindGrant`, `KindCallStart`, `KindCallComplete`,
`KindAffiliation`, and dozens more — and carry a payload. Subscribers register a
channel and receive the kinds they care about. Delivery is asynchronous with
overflow protection, so a slow subscriber can't stall the engine.

<figure class="lab-figure">
<svg viewBox="0 0 680 120" width="680" height="120" role="img" aria-label="The engine's grant-to-call state flow: a KindGrant event triggers HandleGrant, which allocates a voice SDR and publishes KindCallStart; the call stays active while voice frames arrive; the 500-millisecond watchdog reaps a silent call and it ends as KindCallComplete.">
  <rect x="8" y="40" width="92" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="54" y="60" text-anchor="middle" fill="var(--accent)" font-size="10">KindGrant</text>
  <text x="54" y="73" text-anchor="middle" fill="var(--fg-muted)" font-size="8">tg + freq</text>
  <line x1="100" y1="61" x2="118" y2="61" stroke="currentColor"/><polygon points="118,57 128,61 118,65" fill="currentColor"/>
  <rect x="128" y="40" width="118" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="187" y="60" text-anchor="middle" fill="currentColor" font-size="10">HandleGrant</text>
  <text x="187" y="73" text-anchor="middle" fill="var(--fg-muted)" font-size="8">allocate voice SDR</text>
  <line x1="246" y1="61" x2="264" y2="61" stroke="currentColor"/><polygon points="264,57 274,61 264,65" fill="currentColor"/>
  <rect x="274" y="40" width="104" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="326" y="60" text-anchor="middle" fill="currentColor" font-size="10">KindCallStart</text>
  <text x="326" y="73" text-anchor="middle" fill="var(--fg-muted)" font-size="8">retune + record</text>
  <line x1="378" y1="61" x2="396" y2="61" stroke="currentColor"/><polygon points="396,57 406,61 396,65" fill="currentColor"/>
  <rect x="406" y="40" width="104" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="458" y="60" text-anchor="middle" fill="currentColor" font-size="10">active call</text>
  <text x="458" y="73" text-anchor="middle" fill="var(--fg-muted)" font-size="8">voice frames</text>
  <line x1="510" y1="61" x2="528" y2="61" stroke="currentColor"/><polygon points="528,57 538,61 528,65" fill="currentColor"/>
  <text x="605" y="34" text-anchor="middle" fill="var(--fg-muted)" font-size="8">watchdog: silent</text>
  <rect x="538" y="40" width="134" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="605" y="60" text-anchor="middle" fill="var(--accent)" font-size="10">KindCallComplete</text>
  <text x="605" y="73" text-anchor="middle" fill="var(--fg-muted)" font-size="8">reaped after 500ms</text>
</svg>
<figcaption>Inside the engine's <code>select</code> loop, one grant becomes a call: allocate a radio, start recording, then let the 500&nbsp;ms watchdog reap the call once its frames stop.</figcaption>
</figure>

```go
// internal/events — kinds (excerpt)
const (
    KindCCLocked     Kind = "cc.locked"
    KindGrant        Kind = "grant"
    KindCallStart    Kind = "call.start"
    KindCallComplete Kind = "call.complete"
)
```

## The design principle: observer / event-driven decoupling

The defining rule: **the engine publishes, it never calls outward.** It has no
import of `internal/api`, `internal/storage`, or `internal/broadcast`. Those
subsystems are **observers** that subscribe to the bus. This is the observer
pattern at architecture scale, and it's what makes the layered dependency
direction from
[Part 1]({{ '/blog/deep-dives/sdr-internals-01-what-is-software-defined-radio/' | relative_url }})
real.

### How that principle shaped the Go code

- **The core is testable in isolation.** You can run the engine with a fake bus
  and assert it emits the right events for a given grant — no database, no HTTP
  server, no SDR required.
- **Subsystems bolt on without touching the engine.** Adding Broadcastify
  streaming meant writing a subscriber to `KindCallComplete`, not editing the
  engine. The same is true for metrics, the call log, and the affiliation tracker.
- **One writer per piece of state.** The engine's `select` loop is the sole
  mutator of call state, so there are no locks around the hot logic — concurrency
  is handled by the bus, not by sharing.
- **Back-pressure is contained.** Because delivery is async with overflow
  handling, a stalled WebSocket client degrades only its own feed; the decode and
  recording path keeps running.

## Where this goes next

The engine has rich behavior worth its own series — priority/preemption policy,
multi-site roaming, control-channel hunting and backoff, and the affiliation/patch
tracking built from the event stream. A future deep dive will trace a single call
from grant to `KindCallComplete`. Next, we follow the voice frames that a grant
unlocks into the vocoders that turn them into audio.

## FAQ

**Why an in-process bus instead of just function calls?**
Function calls would make the engine depend on every consumer, breaking the
one-way dependency rule and making the core impossible to test in isolation. The
bus inverts that: consumers depend on the engine's events, not the reverse.

**What happens when there are more calls than radios?**
The engine applies priority and preemption — higher-priority talkgroups can take
a voice device from a lower-priority active call. The policy lives entirely in the
engine, decided from the event stream.

**Can two subsystems react to the same event?**
Yes — that's the point. A single `KindCallComplete` can simultaneously trigger a
database write, a metrics increment, and an upload, each in its own subscriber.

## Series navigation

**Part 11 of 14** · ←
[Part 10]({{ '/blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/' | relative_url }})
· Next →
[Part 12: Voice coding — IMBE, AMBE+2 & MBE]({{ '/blog/deep-dives/sdr-internals-12-voice-coding-vocoders/' | relative_url }})
