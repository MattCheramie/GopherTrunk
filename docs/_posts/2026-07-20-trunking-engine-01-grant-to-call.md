---
title: "Trunking Engine, Part 1: From Grant to Recorded Call"
description: A tour of GopherTrunk's trunking engine — the single-goroutine state machine that subscribes to control-channel grants, allocates a voice SDR, starts a call, and publishes events, without ever importing the code that records or displays them.
category: deep-dives
keywords: trunking engine, control channel grant, p25 call flow, voice sdr allocation, event driven architecture, single writer goroutine, select loop, gophertrunk trunking, sdr state machine
tags: [trunking, go, event-bus, architecture, software-design, p25]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Trunking Engine"
series_part: 1
---

*Part 1 of **Trunking Engine**, a 12-part deep dive into the "brain" of
GopherTrunk — the component that turns control-channel grants into recorded
calls. The [SDR Internals]({{ '/blog/series/sdr-internals/' | relative_url }})
series spent one episode here and called it "worth its own series." This is that
series, and it starts with the loop at the center of everything.*

> **TL;DR:** The trunking engine is a **single goroutine** running a `select`
> loop. It subscribes to one thing — the event bus — and reacts to `KindGrant`
> events by looking up the talkgroup, asking the voice pool for an SDR, retuning
> it, and starting a call. It **publishes** `KindCallStart` / `KindCallEnd` and
> never calls the recorder, the database, or the UI directly. That one rule —
> *publish, never call outward* — is what makes the whole engine testable with a
> fake bus and no radio.

**Key takeaways**

- The engine is the **only writer** of call state, so the hot path needs no
  locks — concurrency lives in the bus, not in shared memory.
- It is **ignorant of its consumers**: no import of `internal/api`,
  `internal/storage`, or the recorder. They subscribe to *it*.
- A grant is not a call. Turning one into the other means talkgroup lookup, a
  scarce-resource allocation, retuning, and a watchdog — the subjects of this
  series.
- Everything downstream (recording, the call log, metrics, Broadcastify) is a
  **subscriber to events the engine emits**, added without touching the engine.

## Cheat sheet

| Concept | Where it lives | One-line role |
|---|---|---|
| The engine | `internal/trunking/engine.go` | single-goroutine `select` loop; the only mutator of call state |
| The event bus | `internal/events/bus.go` | typed pub/sub fanout the engine subscribes to and publishes on |
| A grant | `Grant` (`grant.go`) | "talkgroup T is up on frequency F" — the engine's input |
| The voice pool | `VoicePool` (`voicepool.go`) | the scarce set of Voice-role SDRs the engine tunes |
| A call | `ActiveCall` | one talkgroup bound to one voice SDR, watched until it goes silent |
| Key events out | `KindCallStart`, `KindCallEnd` | what subscribers react to |

## In this post

- **What the engine is for** — the gap between a grant and a recorded call.
- **The loop** — one goroutine, one `select`, the classic Go event loop.
- **The one-way rule** — why the engine publishes and never calls outward.
- **The design principle** — single-writer state and observer decoupling, and
  how they shaped the Go.

## The gap between a grant and a call

On a trunked system, voice doesn't live on a fixed frequency. A **control
channel** continuously announces assignments: *talkgroup 1234 is now on 851.0125
MHz.* That announcement is a **grant**. To actually hear the call, something has
to notice the grant, find a spare radio, tune it to 851.0125 MHz, start
capturing audio, and know when the call is over so it can free the radio for the
next one. That "something" is the trunking engine.

The decoders from the [Protocol Decoders]({{ '/blog/series/protocol-decoders/' | relative_url }})
series do the first half — symbols in, a structured `Grant` out. The engine does
the second half. Everything between "a grant arrived" and "a `.wav` is on disk"
is engine work, and it is more subtle than it looks: there are usually **more
talkgroups active than radios to follow them**, grants repeat, calls end without
an explicit "call over" message, and some calls are encrypted. This series is
about how the engine handles all of that. Part 1 is about the shape it handles it
*in*.

<figure class="lab-figure">
<svg viewBox="0 0 680 150" width="680" height="150" role="img" aria-label="A grant flows from the control-channel decoder through the engine, which allocates a voice SDR and starts a call, then publishes call events to subscribers">
  <rect x="6" y="52" width="128" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="70" y="72" text-anchor="middle" fill="currentColor" font-size="12">CC decoder</text>
  <text x="70" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="10">emits Grant</text>
  <line x1="134" y1="75" x2="182" y2="75" stroke="currentColor"/>
  <polygon points="182,71 192,75 182,79" fill="currentColor"/>
  <rect x="192" y="46" width="150" height="58" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="267" y="70" text-anchor="middle" fill="var(--accent)" font-size="12">trunking engine</text>
  <text x="267" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="10">select loop · 1 goroutine</text>
  <line x1="342" y1="75" x2="390" y2="75" stroke="currentColor"/>
  <polygon points="390,71 400,75 390,79" fill="currentColor"/>
  <rect x="400" y="52" width="120" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="460" y="72" text-anchor="middle" fill="currentColor" font-size="12">voice SDR</text>
  <text x="460" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="10">retuned + call</text>
  <line x1="267" y1="104" x2="267" y2="132" stroke="var(--accent)"/>
  <polygon points="263,132 267,142 271,132" fill="var(--accent)"/>
  <rect x="360" y="120" width="314" height="24" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="517" y="136" text-anchor="middle" fill="var(--fg-muted)" font-size="10">recorder · call log · metrics · UI (subscribers)</text>
  <text x="200" y="136" text-anchor="middle" fill="var(--accent)" font-size="10">publishes CallStart / CallEnd</text>
</svg>
<figcaption>The engine sits between the decoder and the radios. It acts on grants and publishes call events; everything that records or displays a call is a subscriber, not a caller.</figcaption>
</figure>

## The loop

The entire engine is one goroutine running a `select`. It drains the event bus
and ticks a watchdog, and that's the whole control structure:

```go
// internal/trunking/engine.go (shape)
func (e *Engine) Run(ctx context.Context) error {
    tick := time.NewTicker(500 * time.Millisecond)
    defer tick.Stop()
    for {
        select {
        case <-ctx.Done():
            e.shutdown()
            return ctx.Err()
        case ev, ok := <-e.sub.C:
            if !ok {
                return nil
            }
            switch ev.Kind {
            case events.KindGrant:
                if g, ok := ev.Payload.(Grant); ok {
                    e.HandleGrant(g) // look up TG, allocate a voice SDR, start a call
                }
            case events.KindPatch:            // regroup/supergroup (Part 9)
                if p, ok := ev.Payload.(Patch); ok { e.handlePatch(p) }
            case events.KindCallEncryption:    // encrypted-mode policy (Part 11)
                if c, ok := ev.Payload.(CallEncryption); ok { e.handleCallEncryption(c) }
            case events.KindCallSourceUpdate:  // source RID backfill (Part 7)
                if c, ok := ev.Payload.(CallSourceUpdate); ok { e.handleCallSourceUpdate(c) }
            case events.KindTalkerAlias:
                if a, ok := ev.Payload.(TalkerAlias); ok { e.handleTalkerAlias(a) }
            }
        case <-tick.C:
            e.runWatchdog() // reap calls with no recent frames (Part 12)
        }
    }
}
```

Every branch of that `switch` is a future post in this series, which is a good
map of where we're going: grants (Parts 3–6), source RID (Part 7), patches
(Part 9), encryption (Part 11), and the watchdog (Part 12). But notice what the
loop does *not* have: no mutex around the call map on the hot path, no callbacks
into other subsystems, no I/O. It receives an event, mutates its own state, and
publishes. That is the entire discipline.

## The one-way rule

The single most important design decision in the engine is stated in its own doc
comment: **the engine knows nothing about the demod pipeline, and nothing about
its consumers.** It imports `internal/events` and that's essentially it. When a
call starts, it does not call `recorder.Start()`. It publishes:

```go
// internal/trunking/engine.go (shape) — inside HandleGrant, after a voice
// device is allocated and retuned:
e.bus.Publish(events.Event{
    Kind:    events.KindCallStart,
    Payload: call, // *ActiveCall: system, talkgroup, frequency, device
})
```

The recorder is a *subscriber* to `KindCallStart`. So is the SQLite call log. So
is the metrics exporter, the web UI's live feed, and the Broadcastify uploader.
None of them is visible to the engine. This is the observer pattern applied at
the scale of a whole subsystem, and it is what the layered, one-way dependency
diagram from
[SDR Internals Part 1]({{ '/blog/deep-dives/sdr-internals-01-what-is-software-defined-radio/' | relative_url }})
looks like in practice. Part 2 takes the bus itself apart.

### How that principle shaped the Go code

- **One writer, no hot-path locks.** The `select` loop is the sole mutator of
  the active-call map, so the fast path — a grant arriving — never contends on a
  mutex. (There *is* an `mu` guarding the maps, but only so read-only API
  callers like the cockpit can snapshot state safely; the loop itself is the
  lone writer.)
- **Testable without a radio.** Because the engine's only input is a channel of
  events and its only output is published events, a test constructs a fake bus,
  publishes a synthetic `Grant`, and asserts a `KindCallStart` comes back — no
  SDR, no HTTP server, no database. Part 12 shows exactly that harness.
- **Subsystems bolt on, never in.** Adding Broadcastify streaming was writing a
  new subscriber to `KindCallEnd`, not editing the engine. The same is true of
  the call log, the affiliation tracker (Part 8), and per-call metrics.
- **Back-pressure is contained.** Delivery is asynchronous with overflow
  handling (Part 2), so a stalled WebSocket client degrades only its own feed;
  the decode-and-record path keeps running.

<figure class="lab-figure">
<svg viewBox="0 0 620 168" width="620" height="168" role="img" aria-label="The engine imports only the events package and publishes outward; consumers depend on the engine's events, the engine never depends on the consumers">
  <rect x="230" y="14" width="160" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="310" y="34" text-anchor="middle" fill="var(--accent)" font-size="12">trunking engine</text>
  <text x="310" y="48" text-anchor="middle" fill="var(--fg-muted)" font-size="10">imports: internal/events</text>
  <line x1="310" y1="56" x2="310" y2="86" stroke="var(--accent)"/>
  <polygon points="306,86 310,96 314,86" fill="var(--accent)"/>
  <rect x="250" y="96" width="120" height="30" rx="6" fill="none" stroke="currentColor"/>
  <text x="310" y="115" text-anchor="middle" fill="currentColor" font-size="11">event bus</text>
  <g stroke="var(--fg-muted)">
    <line x1="250" y1="126" x2="120" y2="150"/><polygon points="120,146 111,151 122,155" fill="var(--fg-muted)"/>
    <line x1="290" y1="126" x2="250" y2="150"/><polygon points="252,146 243,151 254,155" fill="var(--fg-muted)"/>
    <line x1="330" y1="126" x2="380" y2="150"/><polygon points="378,145 389,150 378,155" fill="var(--fg-muted)"/>
    <line x1="370" y1="126" x2="510" y2="150"/><polygon points="508,145 519,150 508,155" fill="var(--fg-muted)"/>
  </g>
  <text x="95" y="163" text-anchor="middle" fill="var(--fg-muted)" font-size="10">recorder</text>
  <text x="240" y="163" text-anchor="middle" fill="var(--fg-muted)" font-size="10">call log</text>
  <text x="385" y="163" text-anchor="middle" fill="var(--fg-muted)" font-size="10">metrics</text>
  <text x="520" y="163" text-anchor="middle" fill="var(--fg-muted)" font-size="10">web UI · upload</text>
</svg>
<figcaption>Dependencies point one way: consumers subscribe to the engine's events. The engine has no idea they exist, which is why new consumers never change it.</figcaption>
</figure>

## Where this goes next

[Part 2]({{ '/blog/deep-dives/trunking-engine-02-event-bus/' | relative_url }})
opens up the event bus the engine lives on — how a typed `Kind` + payload is
fanned out to subscribers, how overflow is handled so a slow consumer can't stall
the engine, and why an in-process bus beats direct function calls for this
design. After that we start following a single grant through the machine: the
`Grant` type and the source-less-grant problem (Part 3), the voice pool that
allocates a radio for it (Part 4), and the priority policy that decides what
happens when there are more calls than radios (Part 5).

## FAQ

**What is the difference between a grant and a call?**
A grant is a control-channel *announcement* — "talkgroup T is on frequency F."
A call is what the engine *creates* from a grant: a voice SDR retuned to F, bound
to talkgroup T, and watched until it goes silent. Many grants (repeats of the
same assignment) map to one call.

**Why is the engine a single goroutine instead of one per call?**
So there is exactly one writer of call state. A single `select` loop means the
hot path — reacting to a grant — needs no locks and can't race with itself.
Per-call concurrency would reintroduce the shared-state bugs the design exists to
avoid; the actual audio work happens in the recorder subscribers, off the loop.

**How does the engine record calls if it never calls the recorder?**
It doesn't record them itself. It publishes `KindCallStart` and `KindCallEnd`,
and the recorder is a subscriber that reacts to those events. The engine has no
import of the recorder at all — that decoupling is the whole point.

**Does the control channel say when a call ends?**
Usually not — P25, for instance, has no explicit channel-release message. The
engine infers the end from a **watchdog**: when a call's voice SDR stops
delivering frames for a timeout window, the watchdog reaps it and publishes
`KindCallEnd`. Part 12 covers this.

## Series navigation

**Part 1 of 12** · Next →
[Part 2: The Event Bus]({{ '/blog/deep-dives/trunking-engine-02-event-bus/' | relative_url }})
