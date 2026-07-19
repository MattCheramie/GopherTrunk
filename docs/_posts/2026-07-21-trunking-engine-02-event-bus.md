---
title: "Trunking Engine, Part 2: The Event Bus — Typed Pub/Sub That Decouples Everything"
description: How GopherTrunk's in-process event bus fans typed events from the trunking engine to every subscriber, with async delivery and overflow protection so one slow consumer can never stall the decode path.
category: deep-dives
keywords: event bus, in-process pub sub, typed events, observer pattern go, trunking engine events, non-blocking channel fanout, overflow protection, gophertrunk event bus, kindgrant kindcallstart
tags: [trunking, go, event-bus, architecture, pub-sub, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Trunking Engine"
series_part: 2
---

*Part 2 of **Trunking Engine**, a 12-part deep dive into the "brain" of
GopherTrunk. [Part 1]({{ '/blog/deep-dives/trunking-engine-01-grant-to-call/' | relative_url }})
introduced the single-goroutine `select` loop and its one rule — publish, never
call outward. This post takes apart the thing it publishes *on*: the event bus,
the 60-line file that lets a dozen subsystems watch the engine without the engine
knowing any of them exist.*

> **TL;DR:** `internal/events.Bus` is an in-process, typed pub/sub bus. Every
> event is an `Event{Kind, Timestamp, Payload}` where `Kind` is a string type and
> `Payload` is `any`. Subscribers get a buffered Go channel; `Publish` does a
> **non-blocking send** to each, and a subscriber whose buffer is full **drops the
> event** rather than stalling the publisher. That one design choice — drop, don't
> block — is what lets the recorder, the web UI, the metrics exporter, and the
> Broadcastify uploader all hang off the engine without any of them being able to
> wedge the decode path.

**Key takeaways**

- The bus is **typed by convention**: a `Kind` constant tells a subscriber which
  concrete struct to type-assert `Payload` into. Add an event kind and its
  payload type; nothing else changes.
- Delivery is **asynchronous and lossy under back-pressure**. A slow subscriber
  drops its own events; it can never apply back-pressure to the publisher or the
  other subscribers.
- Subscribers are fully **decoupled** — the engine holds no reference to any of
  them, only to the `Bus`. This is the observer pattern applied at subsystem
  scale.
- An in-process bus beats direct calls here because it makes the fanout **1-to-N,
  reconfigurable, and testable** without turning the engine into a registry of
  callbacks.

## Cheat sheet

| Symbol | Where | One-line role |
|---|---|---|
| `Kind` | `bus.go` | string type naming an event class (`"grant"`, `"call.start"`, …) |
| `Event{Kind, Timestamp, Payload}` | `bus.go` | one message; `Payload any` is asserted by `Kind` |
| `Bus` | `bus.go` | the fanout — a map of subscriber id → buffered channel |
| `NewBus(buffer int)` | `bus.go` | construct; per-subscriber buffer defaults to 64 |
| `(*Bus).Subscribe() *Subscription` | `bus.go` | returns a read-only `C <-chan Event` |
| `(*Bus).Publish(e Event) int` | `bus.go` | non-blocking fanout; returns the drop count |
| `(*Subscription).Close()` | `bus.go` | unregister and close the channel |

## In this post

- **The `Kind` string type** and the big `const` block that enumerates the
  system's vocabulary of events.
- **The `Event` shape** — `Kind` + `Payload any` — and why the payload is
  untyped.
- **Subscribe / Publish** — how the fanout is wired with a map of channels.
- **Overflow protection** — the non-blocking `select` that makes a slow
  subscriber drop instead of stall.
- **Why a bus at all** — the design principle, and how it shaped the Go.

## The vocabulary: `Kind` and the const block

The bus does not know anything about trunking. It knows about *events*, and an
event's class is a plain string:

```go
// internal/events/bus.go
type Kind string

const (
    KindCallStart        Kind = "call.start"
    KindCallEnd          Kind = "call.end"
    KindCallComplete     Kind = "call.complete"
    KindCallSegment      Kind = "call.segment"
    KindGrant            Kind = "grant"
    KindPatch            Kind = "patch"
    KindTalkerAlias      Kind = "talker.alias"
    KindCallEncryption   Kind = "call.encryption"
    KindCallSourceUpdate Kind = "call.source"
    KindAffiliation      Kind = "affiliation"
    // …and dozens more: cc.locked, sdr.attached, pager.message, adsb.aircraft…
)
```

That const block is the whole system's event vocabulary in one place — from the
control-channel life-cycle (`KindCCLocked`, `KindCCLost`) through the call
life-cycle (`KindGrant` → `KindCallStart` → `KindCallEnd` → `KindCallComplete`)
to the long tail of decoder outputs that have nothing to do with trunking at all
(`KindPagerMessage`, `KindAISMessage`, `KindAircraftReport`). They all share one
bus because they all share one shape.

A few of these are the spine of this series and worth naming now. `KindGrant`
([Part 3]({{ '/blog/deep-dives/trunking-engine-03-grants/' | relative_url }})) is
the engine's only *input*. `KindCallStart` / `KindCallEnd` are its primary
*outputs*. `KindPatch`
([Part 9]({{ '/blog/deep-dives/trunking-engine-09-patches-supergroups/' | relative_url }})),
`KindCallEncryption`
([Part 11]({{ '/blog/deep-dives/trunking-engine-11-encrypted-mode/' | relative_url }})),
and `KindCallSourceUpdate`
([Part 7]({{ '/blog/deep-dives/trunking-engine-07-source-rid-recovery/' | relative_url }}))
are the enrichment events the engine both subscribes to *and* republishes — the
loop reads them off the bus, backfills the bound call, and publishes an enriched
copy so downstream consumers see the update live.

<figure class="lab-figure">
<svg viewBox="0 0 680 176" width="680" height="176" role="img" aria-label="The trunking engine publishes call events onto the event bus, which fans each event out to independent subscribers: recorder, SQLite call log, metrics exporter, web UI feed, and Broadcastify uploader">
  <rect x="250" y="12" width="180" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="340" y="32" text-anchor="middle" fill="var(--accent)" font-size="12">trunking engine</text>
  <text x="340" y="47" text-anchor="middle" fill="var(--fg-muted)" font-size="10">Publish(Event{Kind, Payload})</text>
  <line x1="340" y1="54" x2="340" y2="80" stroke="var(--accent)"/>
  <polygon points="336,80 340,90 344,80" fill="var(--accent)"/>
  <rect x="210" y="90" width="260" height="30" rx="6" fill="none" stroke="currentColor"/>
  <text x="340" y="109" text-anchor="middle" fill="currentColor" font-size="11">events.Bus — map[id]chan Event</text>
  <g stroke="var(--fg-muted)">
    <line x1="230" y1="120" x2="70" y2="146"/><polygon points="70,142 61,147 72,151" fill="var(--fg-muted)"/>
    <line x1="290" y1="120" x2="210" y2="146"/><polygon points="212,142 203,147 214,151" fill="var(--fg-muted)"/>
    <line x1="340" y1="120" x2="340" y2="146"/><polygon points="336,146 340,152 344,146" fill="var(--fg-muted)"/>
    <line x1="390" y1="120" x2="470" y2="146"/><polygon points="468,141 479,146 468,151" fill="var(--fg-muted)"/>
    <line x1="450" y1="120" x2="610" y2="146"/><polygon points="608,141 619,146 608,151" fill="var(--fg-muted)"/>
  </g>
  <text x="55" y="164" text-anchor="middle" fill="var(--fg-muted)" font-size="10">recorder</text>
  <text x="195" y="164" text-anchor="middle" fill="var(--fg-muted)" font-size="10">call log</text>
  <text x="340" y="164" text-anchor="middle" fill="var(--fg-muted)" font-size="10">metrics</text>
  <text x="475" y="164" text-anchor="middle" fill="var(--fg-muted)" font-size="10">web UI</text>
  <text x="618" y="164" text-anchor="middle" fill="var(--fg-muted)" font-size="10">upload</text>
</svg>
<figcaption>One publish, N deliveries. Each subscriber owns a buffered channel; the bus never knows what any of them does with an event.</figcaption>
</figure>

## The `Event` shape

An event is deliberately small:

```go
// internal/events/bus.go
type Event struct {
    Kind      Kind
    Timestamp time.Time
    Payload   any
}
```

`Payload` is `any` (Go's alias for `interface{}`), and that is a considered
trade-off. A statically-typed bus — one channel per event type — would give the
compiler a say, but it would also mean the engine, the API layer, and every
decoder shared a growing zoo of channel types and a fanout struct that had to be
edited for every new event. Instead, the contract is *by convention*: a
`KindCallStart` event's `Payload` is always a `trunking.CallStart`, a
`KindGrant`'s is always a `trunking.Grant`. Subscribers assert it, exactly as
the engine's own loop does in Part 1:

```go
// a subscriber's receive side (shape)
case ev := <-sub.C:
    switch ev.Kind {
    case events.KindCallStart:
        cs, ok := ev.Payload.(trunking.CallStart) // Kind tells you the type
        if !ok { continue }
        recorder.Begin(cs)
    }
```

The `Kind` *is* the type tag. The pairing is documented right on each constant in
`bus.go` — the doc comment for `KindCallSourceUpdate`, for instance, names its
payload as `trunking.CallSourceUpdate` and explains exactly when the voice
composer emits it. That discipline — every `Kind` names its payload type in a
comment — is what keeps an untyped `any` from becoming a guessing game.

Note also that `events` imports nothing from `trunking`. Payload structs that
both packages need — `DecodeError`, `ChannelPower`, the DMR band-plan types —
are declared in `events` with primitive fields precisely so the bus stays free
of a `trunking` import and no import cycle forms. The bus is the bottom of the
dependency graph, and it stays there.

## Subscribe and Publish

The `Bus` itself is a map of subscriber id to channel, under an `RWMutex`:

```go
// internal/events/bus.go
type Bus struct {
    mu     sync.RWMutex
    subs   map[uint64]chan Event
    nextID atomic.Uint64
    buffer int
    closed bool
}

func (b *Bus) Subscribe() *Subscription {
    b.mu.Lock()
    defer b.mu.Unlock()
    id := b.nextID.Add(1)
    ch := make(chan Event, b.buffer) // buffered, default 64
    b.subs[id] = ch
    return &Subscription{id: id, C: ch, b: b}
}
```

`Subscribe` mints a fresh buffered channel and hands back a `Subscription` whose
`C` field is a *receive-only* `<-chan Event` — a subscriber can read but never
send or close it. `Close()` deletes the entry and closes the channel, so a
consumer's `for ev := range sub.C` naturally terminates. The `RWMutex` lets many
`Publish` calls fan out concurrently (read lock) while `Subscribe`/`Close` take
the write lock — subscription churn is rare, publishes are hot.

## Overflow protection: drop, don't block

Here is the load-bearing code — the entire reason the bus exists in this form:

```go
// internal/events/bus.go
// Publish delivers e to every subscriber. Slow subscribers drop the event
// rather than blocking the publisher; we count drops via the returned int.
func (b *Bus) Publish(e Event) int {
    if e.Timestamp.IsZero() {
        e.Timestamp = time.Now()
    }
    b.mu.RLock()
    defer b.mu.RUnlock()
    dropped := 0
    for _, ch := range b.subs {
        select {
        case ch <- e:       // room in the buffer → deliver
        default:            // buffer full → drop, count it
            dropped++
        }
    }
    return dropped
}
```

The `select` with a `default` is a **non-blocking send**. If a subscriber's
64-deep buffer has room, the event lands. If it's full — because that subscriber
is slow — the `default` arm fires, the event is dropped for *that subscriber
only*, and `Publish` moves on. `Publish` returns the number of drops so a caller
can surface it (the engine logs a metric when a publish drops).

Why this matters concretely: the web UI's live feed goes over a WebSocket to a
browser that might be on hotel Wi-Fi. Without overflow protection, a blocked
WebSocket write would back up its subscriber channel, block `Publish`, and block
the engine's `select` loop — and now a laggy browser has stalled call *recording*
on the host. With it, the slow WebSocket simply misses a few live-view updates
while the recorder, the call log, and the decode path run at full speed. Each
subscriber's back-pressure is contained entirely within its own buffer.

This is a real trade-off, not a free lunch: the bus is **lossy under load**. It
is the right default for a *live-view/telemetry* fanout where missing an update
is survivable, and it is explicitly the wrong tool for anything that must not
drop — durable call metadata is written by a subscriber that persists to SQLite,
and if that subscriber ever fell behind, the fix is a bigger buffer or a
dedicated queue, not making the bus block.

<figure class="lab-figure">
<svg viewBox="0 0 680 180" width="680" height="180" role="img" aria-label="Publish sends into each subscriber's buffered channel with a non-blocking select: the fast subscriber accepts the event, the slow subscriber whose buffer is full drops it, and the publisher never blocks">
  <rect x="14" y="72" width="120" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="74" y="90" text-anchor="middle" fill="var(--accent)" font-size="12">Publish(e)</text>
  <text x="74" y="105" text-anchor="middle" fill="var(--fg-muted)" font-size="10">non-blocking send</text>
  <line x1="134" y1="80" x2="196" y2="46" stroke="currentColor"/>
  <polygon points="196,50 205,44 195,42" fill="currentColor"/>
  <line x1="134" y1="104" x2="196" y2="138" stroke="currentColor"/>
  <polygon points="196,134 205,140 195,142" fill="currentColor"/>
  <rect x="206" y="26" width="220" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="316" y="44" text-anchor="middle" fill="currentColor" font-size="11">fast subscriber — buffer has room</text>
  <text x="316" y="59" text-anchor="middle" fill="var(--accent)" font-size="10">event delivered</text>
  <rect x="206" y="118" width="220" height="40" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="316" y="136" text-anchor="middle" fill="currentColor" font-size="11">slow subscriber — buffer full (64)</text>
  <text x="316" y="151" text-anchor="middle" fill="var(--fg-muted)" font-size="10">event dropped (counted)</text>
  <line x1="426" y1="46" x2="486" y2="46" stroke="currentColor"/>
  <polygon points="486,42 496,46 486,50" fill="currentColor"/>
  <rect x="496" y="26" width="176" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="584" y="50" text-anchor="middle" fill="currentColor" font-size="11">recorder keeps up</text>
  <line x1="426" y1="138" x2="486" y2="138" stroke="var(--fg-muted)"/>
  <polygon points="486,134 496,138 486,142" fill="var(--fg-muted)"/>
  <rect x="496" y="118" width="176" height="40" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="584" y="142" text-anchor="middle" fill="var(--fg-muted)" font-size="11">stale live feed only</text>
</svg>
<figcaption>The <code>default</code> arm of the send <code>select</code> is the whole design: a full buffer drops one subscriber's event instead of stalling the publisher and everyone else.</figcaption>
</figure>

## Why a bus instead of direct calls

### How that principle shaped the Go code

The engine could, in principle, hold a `recorder`, a `callLog`, a `metrics`, and
a `wsHub`, and call each in turn when a call starts. It deliberately doesn't, and
the bus is why:

- **One-to-N without a registry.** Direct calls make the engine own a list of
  every consumer and the order it calls them. The bus inverts that: consumers
  register themselves with `Subscribe()`, and the engine's fanout is a single
  `Publish`. Adding the Broadcastify uploader (a `KindCallComplete` subscriber)
  touched zero lines of engine code.
- **Fault isolation.** A direct call means a panic or a slow write in one
  consumer is on the engine's goroutine. A bus means each subscriber drains its
  own channel on its own goroutine, and the non-blocking `Publish` means even a
  wedged one can't reach back.
- **Testability.** Because the engine's only output is `Publish`, a test
  subscribes a fake channel, feeds a synthetic `Grant`, and asserts a
  `KindCallStart` comes back — the harness we build in
  [Part 12]({{ '/blog/deep-dives/trunking-engine-12-cc-hunting-watchdog-testing/' | relative_url }}).
  No mocks of four subsystems, just one channel.
- **Uniformity.** The pager decoder, the ADS-B decoder, and the trunking engine
  all publish onto the same bus with the same `Event` shape, so the API layer's
  SSE/WebSocket bridge is *one* subscriber that forwards everything, not one
  adapter per producer.

The cost is the untyped `Payload` and the by-convention type contract. In a
Go 1.25 codebase without sum types, that is the pragmatic price for a fanout that
every subsystem can share.

## Where this goes next

With the bus understood, we can follow a single message through it.
[Part 3]({{ '/blog/deep-dives/trunking-engine-03-grants/' | relative_url }})
opens the `Grant` — the payload of the one event the engine treats as *input* —
and the duplicate-grant guard that keeps a repeated control-channel TSBK from
binding a second radio. From there the series follows that grant into the voice
pool ([Part 4]({{ '/blog/deep-dives/trunking-engine-04-voice-pool/' | relative_url }}))
and the priority policy
([Part 5]({{ '/blog/deep-dives/trunking-engine-05-priority-preemption/' | relative_url }})).

## FAQ

**Is the event bus thread-safe?**
Yes. The subscriber map is guarded by an `RWMutex`: `Publish` takes the read lock
so many publishes fan out concurrently, while `Subscribe` and `Close` take the
write lock. Each subscriber's channel is single-producer (the bus) and
single-consumer (the subscriber), so no further synchronization is needed on the
channel itself.

**What happens if a subscriber is too slow?**
It drops events. `Publish` does a non-blocking send into each subscriber's
buffered channel; if the buffer (default 64) is full, that event is dropped for
that subscriber and counted in `Publish`'s return value. A slow subscriber
degrades only its own feed — it can never block the publisher or the other
subscribers.

**Why is `Payload` typed as `any` instead of a concrete type?**
So one bus can carry every event class in the system without the `events`
package depending on `trunking` (or any decoder). The `Kind` constant names the
concrete payload type by convention, documented on each constant, and
subscribers type-assert it. It trades compile-time checking for a fanout every
subsystem can share.

**Does publishing block until subscribers process the event?**
No. Delivery is asynchronous: `Publish` hands the event to each subscriber's
buffer and returns immediately. Subscribers process on their own goroutines. This
is what keeps the engine's `select` loop from ever waiting on a consumer.

**How do I add a new event type?**
Add a `Kind` constant to the const block in `bus.go` (with a doc comment naming
its payload type), define the payload struct wherever it naturally lives, publish
it from the producer, and subscribe where you need it. No change to `Bus`,
`Publish`, or any existing subscriber.

## Series navigation

**Part 2 of 12** · ←
[Part 1: From Grant to Recorded Call]({{ '/blog/deep-dives/trunking-engine-01-grant-to-call/' | relative_url }})
· Next →
[Part 3: Grants — The Engine's Only Input]({{ '/blog/deep-dives/trunking-engine-03-grants/' | relative_url }})
