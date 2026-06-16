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
