---
title: "Recording, Composition & Streaming, Part 13: Outbound Streaming — The Broadcast Manager"
description: How GopherTrunk ships a finished call to the outside world — one bus subscriber that gates on stream/duration, encodes MP3 exactly once, and fans out through a bounded worker pool with drop-on-full and exponential-backoff retry so a wedged aggregator can never back-pressure the decode path.
category: deep-dives
keywords: broadcast manager gophertrunk, call complete subscriber, bounded worker pool go, drop on full queue, exponential backoff retry, lazy mp3 encode once, stream gate min duration, aggregator upload fanout, non blocking event loop
tags: [streaming, broadcast, concurrency, go, backpressure, events]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Recording, Composition & Streaming"
series_part: 13
---

*Part 13 of **Recording, Composition & Streaming**, following one call — the 3
p.m. dispatch on talkgroup 101 — from PCM to a Broadcastify upload. By now the
recorder has flushed its WAV, the call log has its row, and the engine has moved
on. The last durable thing left to do is **share** the call: push it to
Broadcastify, RdioScanner, OpenMHz, a webhook, or a live Icecast feed. This post
is the machine that does the pushing — the **broadcast Manager**. Part 14 tours
each destination's wire protocol; here we build the pump that feeds all of them.*

> **TL;DR:** The `broadcast.Manager` is a single `KindCallComplete` subscriber.
> On each completed call it applies two filters — the talkgroup **stream gate**
> and a **minimum-duration** floor — then enqueues the call onto a bounded
> channel. A fixed pool of **worker goroutines** drains that channel, encodes the
> MP3 **once** (lazily, shared across all backends), and calls each backend's
> `Send` with **exponential-backoff** retry. If the queue is full, the call is
> **dropped**, never blocked — the decode path is sacred and the Manager will
> lose an upload before it ever back-pressures the recorder.

**Key takeaways**

- The Manager's event loop does almost nothing: filter, then a **non-blocking
  send** onto a buffered `jobs` channel. A wedged backend fills the queue and the
  next call is dropped — the loop never stalls.
- MP3 encoding is **lazy and once-per-call**. `Call.MP3()` memoises the encoded
  bytes behind a mutex, so five backends streaming the same call pay for **one**
  encode, and a metadata-only webhook pays for **zero**.
- Retry is **per-backend**, bounded by `MaxRetries`, with a delay that starts at
  `RetryBase` (2s) and **doubles** each attempt. After the budget is spent the
  call is abandoned and counted as failed — it is never retried forever.
- Two independent gates decide whether a call streams at all: the talkgroup's
  `Stream` flag (operator opt-out) and `MinDuration` (kerchunk filter). Both live
  in `dispatch`, before anything touches the network.

## Cheat sheet

| Thing | What it does | Where in code |
|---|---|---|
| `broadcast.Manager` | Subscribes to `KindCallComplete`, filters, fans out | `internal/broadcast/manager.go` |
| `Options` | Bus, backends, `MinDuration`, `Workers`, `MaxRetries`, `RetryBase` | `internal/broadcast/manager.go` |
| `Manager.dispatch` | Stream gate + min-duration filter + non-blocking enqueue | `internal/broadcast/manager.go` |
| `Manager.worker` | Drains `jobs`, calls `sendWithRetry` per accepting backend | `internal/broadcast/manager.go` |
| `Manager.sendWithRetry` | `MaxRetries+1` attempts, backoff doubles from `RetryBase` | `internal/broadcast/manager.go` |
| `Call` / `Call.MP3()` | Completed call + lazy, memoised MP3 accessor | `internal/broadcast/broadcast.go` |
| `Backend` | `Name` / `Accepts` / `Send` — one outbound destination | `internal/broadcast/broadcast.go` |
| `buildBroadcastManager` | Builds the Manager from config; returns `nil` when no feed | `cmd/gophertrunk/broadcast.go` |

## In this post

- **The event seam** — why the Manager subscribes rather than being called.
- **The two gates** in `dispatch` — stream opt-out and minimum duration.
- **Drop-on-full** — the one decision that keeps the decode path unblockable.
- **Encode once** — how `Call.MP3()` shares a single encode across N backends.
- **Bounded backoff retry** — the `sendWithRetry` timeline, and when to give up.

## The seam: one subscriber, nothing calls it

[Part 1]({{ '/blog/deep-dives/recording-streaming-01-the-output-half/' | relative_url }})
established the rule the whole output half lives by: subsystems **subscribe**,
they don't call each other. The broadcast Manager is the purest example. It
imports neither the recorder nor the call log; its only tie to the rest of the
daemon is a subscription to one event kind.

```go
// internal/broadcast/manager.go (shape)
func (m *Manager) Run(ctx context.Context) error {
    defer close(m.runDone)
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case ev, ok := <-m.sub.C:
            if !ok {
                return nil
            }
            if ev.Kind != events.KindCallComplete {
                continue
            }
            cc, ok := ev.Payload.(trunking.CallComplete)
            if !ok {
                continue
            }
            m.dispatch(cc)
        }
    }
}
```

`Run` is a plain event loop. It filters the bus down to `KindCallComplete`,
type-asserts the payload to a `trunking.CallComplete` (the struct from
[Part 9]({{ '/blog/deep-dives/recording-streaming-09-call-complete-seam/' | relative_url }})
that carries `AudioPath`, `StartedAt`, `EndedAt`, and the grant), and hands it to
`dispatch`. One subtle guarantee lives in the constructor: `NewManager`
**subscribes to the bus before it returns**, and starts the workers immediately —
so a call that completes in the window between construction and the first `Run`
iteration is buffered in the subscription, not lost.

The important property is what `Run` does *not* do: it never blocks on a network
socket. Everything after `dispatch` happens on a separate goroutine, so a
Broadcastify server that takes thirty seconds to answer stalls a worker, not the
event loop — and never the recorder that published the event.

## Two gates in `dispatch`

`dispatch` is where a completed call earns — or fails to earn — a spot in the
upload queue. It applies two independent filters before it does any work:

```go
// internal/broadcast/manager.go (shape)
func (m *Manager) dispatch(cc trunking.CallComplete) {
    if len(m.backends) == 0 {
        return
    }
    if cc.Talkgroup != nil && !cc.Talkgroup.Stream {
        return // talkgroup opted out of all feeds
    }
    if m.minDuration > 0 && cc.Duration() < m.minDuration {
        return // too short — a kerchunk, not a real transmission
    }
    call := callFromEvent(cc)
    call.normalize = m.normalize
    // …enqueue (below)
}
```

The **stream gate** is the operator's opt-out. Every `TalkGroup` carries a
`Stream bool` (`internal/trunking/talkgroup.go`) that defaults to `true` — legacy
CSV/JSON talkgroup files without a `Stream` column keep streaming everything, so
the gate is backward-compatible. Flag a sensitive talkgroup `Stream=false` and
its completed calls are dropped here, before any backend sees them. It is a
single decision point covering **all** feeds at once.

The **minimum-duration** floor drops calls shorter than `MinDuration`. This is
the kerchunk filter: a 300 ms key-up with no real audio is noise on an
aggregator feed, and `MinDuration` (set from `broadcast.min_duration_ms` in
config) lets an operator suppress them. Zero means "stream calls of any length".
Both gates run *before* `callFromEvent`, so a rejected call costs nothing beyond
two comparisons.

Once a call passes both gates, `callFromEvent` (`internal/broadcast/broadcast.go`)
projects the `CallComplete` payload into a `*Call` — the Manager's own value type
carrying system, protocol, talkgroup, source, frequency, the P25 site identity,
timestamps, and the on-disk `AudioPath`. The recorder's engine types never cross
into the backends; the `Call` is the Manager's stable interface to them.

<figure class="lab-figure">
<svg viewBox="0 0 660 220" width="660" height="220" role="img" aria-label="Pipeline: a call-complete event enters the Manager's Run loop, passes through the stream gate and minimum-duration filter in dispatch, then either drops when the bounded jobs channel is full or enqueues onto it; a pool of N worker goroutines drains the channel and runs a per-backend retry loop that ends at Broadcastify, RdioScanner, and other destinations.">
  <rect x="8" y="88" width="104" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="60" y="108" text-anchor="middle" fill="var(--accent)" font-size="10">call.complete</text>
  <text x="60" y="122" text-anchor="middle" fill="var(--fg-muted)" font-size="8">bus event</text>
  <line x1="112" y1="110" x2="140" y2="110" stroke="currentColor"/><polygon points="140,106 150,110 140,114" fill="currentColor"/>
  <rect x="150" y="82" width="118" height="56" rx="6" fill="none" stroke="currentColor"/>
  <text x="209" y="102" text-anchor="middle" fill="currentColor" font-size="10">dispatch</text>
  <text x="209" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="8">Stream gate ·</text>
  <text x="209" y="127" text-anchor="middle" fill="var(--fg-muted)" font-size="8">MinDuration</text>
  <line x1="268" y1="110" x2="296" y2="110" stroke="currentColor"/><polygon points="296,106 306,110 296,114" fill="currentColor"/>
  <line x1="209" y1="138" x2="209" y2="196" stroke="var(--fg-muted)" stroke-dasharray="4 3"/><polygon points="205,196 209,206 213,196" fill="var(--fg-muted)"/>
  <text x="209" y="200" text-anchor="middle" fill="var(--fg-muted)" font-size="8">rejected → return</text>
  <rect x="306" y="86" width="96" height="48" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="354" y="104" text-anchor="middle" fill="var(--accent)" font-size="10">jobs chan</text>
  <text x="354" y="117" text-anchor="middle" fill="var(--fg-muted)" font-size="8">buffered (64)</text>
  <text x="354" y="128" text-anchor="middle" fill="var(--fg-muted)" font-size="8">full → drop</text>
  <line x1="402" y1="110" x2="430" y2="110" stroke="currentColor"/><polygon points="430,106 440,110 430,114" fill="currentColor"/>
  <rect x="440" y="70" width="96" height="26" rx="5" fill="none" stroke="currentColor"/>
  <text x="488" y="87" text-anchor="middle" fill="currentColor" font-size="9">worker 1</text>
  <rect x="440" y="102" width="96" height="26" rx="5" fill="none" stroke="currentColor"/>
  <text x="488" y="119" text-anchor="middle" fill="currentColor" font-size="9">worker N</text>
  <text x="488" y="142" text-anchor="middle" fill="var(--fg-muted)" font-size="8">sendWithRetry</text>
  <line x1="536" y1="90" x2="566" y2="90" stroke="currentColor"/><polygon points="566,86 576,90 566,94" fill="currentColor"/>
  <line x1="536" y1="114" x2="566" y2="114" stroke="currentColor"/><polygon points="566,110 576,114 566,118" fill="currentColor"/>
  <rect x="576" y="76" width="80" height="26" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="616" y="93" text-anchor="middle" fill="var(--fg-muted)" font-size="8">Broadcastify</text>
  <rect x="576" y="106" width="80" height="26" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="616" y="123" text-anchor="middle" fill="var(--fg-muted)" font-size="8">RdioScanner…</text>
</svg>
<figcaption>The Manager's event loop only filters and enqueues; the network work happens on worker goroutines, so a slow backend can never stall <code>dispatch</code>.</figcaption>
</figure>

## Drop-on-full: the decision that protects decode

Here is the single most important line in the subsystem — the enqueue:

```go
// internal/broadcast/manager.go (shape)
select {
case m.jobs <- call:
    m.mu.Lock(); m.queued++; m.mu.Unlock()
default:
    // Queue full — a backend is wedged. Drop rather than block the
    // event loop (and the recorder behind it).
    m.mu.Lock(); m.dropped++; m.mu.Unlock()
    m.log.Warn("broadcast: upload queue full, dropping call",
        "system", call.System, "tg", call.Talkgroup)
}
```

The `jobs` channel is buffered (depth 64). The `select` with a `default` case is
a **non-blocking send**: if there is room, the call is queued; if the buffer is
full, the `default` fires and the call is **dropped** and counted. There is no
third option where `dispatch` waits.

This is deliberate and it is the whole reason the broadcast path is safe to run
in production. Imagine Broadcastify goes down for a minute. Each worker blocks
inside `Send` waiting on a dead socket; the `jobs` channel fills to 64; and then
`dispatch` starts dropping. What it does **not** do is block — which means the
`Run` loop keeps draining the bus, the bus never fills, and the recorder that
publishes `KindCallComplete` never waits on the broadcaster. **A downstream
outage costs you some uploads; it never costs you a recording.** The dropped
counter surfaces the loss in `Stats()` so an operator can see it happened.

> ⚠ Drop-on-full is a design choice, not a bug. If your feed is losing calls,
> the fix is more `Workers` or a faster network — not an unbounded queue, which
> would trade dropped uploads for unbounded memory growth and, eventually, the
> exact back-pressure this design exists to prevent.

## Encode the MP3 once

A completed call might go to five backends. Encoding its WAV to MP3 five times
would be wasteful — MP3 encoding is the most expensive thing the Manager does.
`Call.MP3()` solves this with lazy, memoised, mutex-guarded encoding:

```go
// internal/broadcast/broadcast.go (shape)
func (c *Call) MP3() ([]byte, error) {
    c.mu.Lock()
    defer c.mu.Unlock()
    if c.mp3Done {
        return c.mp3Data, c.mp3Err // cached bytes (or cached error)
    }
    c.mp3Done = true
    if c.normalize.Enabled {
        c.mp3Data, c.mp3Err = encodeNormalizedMP3(c.AudioPath, c.normalize.Params)
    } else {
        c.mp3Data, _, c.mp3Err = mp3.EncodeWAVFile(c.AudioPath)
    }
    return c.mp3Data, c.mp3Err
}
```

The first backend to call `MP3()` runs the encode; every later caller — including
retries — gets the cached slice back. The mutex makes it safe across the several
worker goroutines that might touch the same call. Two consequences fall out of
this shape:

- **Metadata-only feeds pay nothing.** The webhook backend, unless
  `IncludeAudio` is set, never calls `MP3()`, so a call streamed only to a
  metadata webhook is never encoded at all.
- **Encoding is a black box here.** The Shine MP3 encoder, the Xing/LAME header,
  and the WAV read all live in the voice package and are covered in
  [Voice Coding Part 11]({{ '/blog/deep-dives/voice-coding-11-recording-encoding/' | relative_url }}).
  The Manager treats `mp3.EncodeWAVFile` as a function that turns a path into
  bytes; that boundary is exactly why the two subsystems can evolve independently.

The one wrinkle is optional loudness normalization. When
`normalize.Enabled`, `encodeNormalizedMP3` reads the WAV, applies a single gain
**in memory**, and encodes the adjusted samples — the on-disk file the recorder
wrote is left pristine. The *math* of that gain (BS.1770/R128) belongs to
[Voice Coding Part 10]({{ '/blog/deep-dives/voice-coding-10-enhancement-loudness/' | relative_url }});
what matters here is only *where* it is applied: on the distributed copy, once,
behind the same memoisation.

## The worker and its retry loop

Workers are trivial: each drains `jobs` and, for every backend that `Accepts`
the call's system, runs `sendWithRetry`.

```go
// internal/broadcast/manager.go (shape)
func (m *Manager) worker() {
    defer m.wg.Done()
    for call := range m.jobs {
        for _, b := range m.backends {
            if !b.Accepts(call.System) {
                continue // this backend filters out this system
            }
            m.sendWithRetry(b, call)
        }
    }
}
```

`Accepts` is the per-backend `systemFilter` from
[Part 14]({{ '/blog/deep-dives/recording-streaming-14-aggregator-backends/' | relative_url }}):
a backend configured with a `Systems` list only takes calls from those trunking
systems; an unfiltered backend takes everything. So one Manager can feed
"Metro" calls to one Broadcastify system and "County" calls to another.

`sendWithRetry` is the backoff engine:

```go
// internal/broadcast/manager.go (shape)
func (m *Manager) sendWithRetry(b Backend, call *Call) {
    backoff := m.retryBase // 2s by default
    for attempt := 0; attempt <= m.maxRetries; attempt++ {
        ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
        err := b.Send(ctx, call)
        cancel()
        if err == nil {
            m.mu.Lock(); m.sent[b.Name()]++; m.mu.Unlock()
            return // delivered
        }
        m.log.Warn("broadcast: upload failed", "backend", b.Name(),
            "attempt", attempt+1, "of", m.maxRetries+1, "err", err)
        if attempt < m.maxRetries {
            time.Sleep(backoff)
            backoff *= 2 // 2s → 4s → 8s …
        }
    }
    m.mu.Lock(); m.failed[b.Name()]++; m.mu.Unlock()
    m.log.Error("broadcast: giving up on call", "backend", b.Name())
}
```

The loop runs `MaxRetries+1` attempts total (the first try plus the retry
budget). Each `Send` gets a fresh 60-second context, so a hung request can't wedge
a worker forever. Between attempts the worker **sleeps** for `backoff`, then
**doubles** it: 2s, 4s, 8s with the defaults. That doubling is what turns a
transient blip (a 502, a dropped TCP connection) into a recovered upload while
keeping a persistent outage from hammering a dead server. When the budget is
spent, the call is counted as `failed` for that backend and abandoned — the
Manager never retries a call forever, and a failure on one backend never affects
the others, because each gets its own independent `sendWithRetry` call.

<figure class="lab-figure">
<svg viewBox="0 0 660 176" width="660" height="176" role="img" aria-label="Timeline for sendWithRetry with three retries: attempt 1 fails, wait 2 seconds; attempt 2 fails, wait 4 seconds; attempt 3 fails, wait 8 seconds; attempt 4 fails and the Manager gives up and counts the call as failed. A green branch shows any attempt succeeding leads to sent and an immediate return.">
  <line x1="20" y1="70" x2="600" y2="70" stroke="var(--fg-muted)"/>
  <polygon points="600,66 610,70 600,74" fill="var(--fg-muted)"/>
  <text x="610" y="60" fill="var(--fg-muted)" font-size="8">time</text>
  <rect x="24" y="56" width="70" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="59" y="74" text-anchor="middle" fill="currentColor" font-size="9">attempt 1</text>
  <text x="120" y="60" text-anchor="middle" fill="var(--fg-muted)" font-size="8">wait 2s</text>
  <line x1="94" y1="70" x2="146" y2="70" stroke="currentColor" stroke-dasharray="3 2"/>
  <rect x="146" y="56" width="70" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="181" y="74" text-anchor="middle" fill="currentColor" font-size="9">attempt 2</text>
  <text x="248" y="60" text-anchor="middle" fill="var(--fg-muted)" font-size="8">wait 4s</text>
  <line x1="216" y1="70" x2="290" y2="70" stroke="currentColor" stroke-dasharray="3 2"/>
  <rect x="290" y="56" width="70" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="325" y="74" text-anchor="middle" fill="currentColor" font-size="9">attempt 3</text>
  <text x="404" y="60" text-anchor="middle" fill="var(--fg-muted)" font-size="8">wait 8s</text>
  <line x1="360" y1="70" x2="458" y2="70" stroke="currentColor" stroke-dasharray="3 2"/>
  <rect x="458" y="56" width="70" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="493" y="74" text-anchor="middle" fill="currentColor" font-size="9">attempt 4</text>
  <line x1="528" y1="70" x2="556" y2="70" stroke="currentColor"/><polygon points="556,66 566,70 556,74" fill="currentColor"/>
  <rect x="566" y="54" width="80" height="32" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="606" y="70" text-anchor="middle" fill="var(--fg-muted)" font-size="8">give up</text>
  <text x="606" y="80" text-anchor="middle" fill="var(--fg-muted)" font-size="8">failed++</text>
  <line x1="325" y1="84" x2="325" y2="128" stroke="var(--accent)"/><polygon points="321,128 325,138 329,128" fill="var(--accent)"/>
  <text x="325" y="118" text-anchor="middle" fill="var(--accent)" font-size="8">Send err == nil</text>
  <rect x="270" y="138" width="110" height="28" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="325" y="156" text-anchor="middle" fill="var(--accent)" font-size="9">sent++ · return</text>
</svg>
<figcaption>Per-backend backoff: the delay doubles from <code>RetryBase</code> each attempt, capped by <code>MaxRetries</code>; any successful <code>Send</code> records the delivery and returns immediately.</figcaption>
</figure>

## Where the Manager comes from

The daemon never constructs a `Manager` unconditionally. `buildBroadcastManager`
(`cmd/gophertrunk/broadcast.go`) walks each backend config section, builds the
enabled ones, and — critically — **returns `nil, nil` when the backend list is
empty**:

```go
// cmd/gophertrunk/broadcast.go (shape)
func buildBroadcastManager(cfg config.BroadcastConfig, /* … */) (*broadcast.Manager, error) {
    var backends []broadcast.Backend
    // …append one Backend per enabled Broadcastify/RdioScanner/OpenMHz/Webhook/Icecast entry
    if len(backends) == 0 {
        return nil, nil // no feed configured → no Manager, no workers, no encoder
    }
    return broadcast.NewManager(broadcast.Options{
        Bus: bus, Log: log, Backends: backends,
        MinDuration: time.Duration(cfg.MinDurationMs) * time.Millisecond,
        Workers:     cfg.Workers,
        Normalize:   normalize,
    })
}
```

This is the config-driven lazy init rule from
[Part 1]({{ '/blog/deep-dives/recording-streaming-01-the-output-half/' | relative_url }})
applied to the broadcaster: a scanner with no feed configured allocates no
worker goroutines, no MP3 encoder, and no bus subscription. The subsystem isn't
disabled at runtime — it doesn't exist. And because `Workers` and `MinDurationMs`
flow straight from config, the pool size and kerchunk floor are the operator's to
tune without touching code.

## Where this goes next

The Manager fans a `*Call` out to a set of `Backend`s, but it treats every
backend as an opaque `Send(ctx, call) error`.
[Part 14]({{ '/blog/deep-dives/recording-streaming-14-aggregator-backends/' | relative_url }}),
the finale, opens each of those boxes: Broadcastify's two-step metadata-then-PUT
upload, the multipart POSTs for RdioScanner and OpenMHz, Icecast's paced source
connection topped up with pre-encoded silence, and the generic JSON webhook — and
it finally lands our 3 p.m. dispatch in a real Broadcastify feed.

## FAQ

**What happens to a call when every upload worker is busy?**
`dispatch` does a non-blocking send onto the buffered `jobs` channel. If the
buffer (depth 64) is full because the workers are wedged on a slow backend, the
`default` branch of the `select` fires and the call is **dropped** and counted in
`Stats().Dropped`. The event loop never blocks, so the recorder is never
back-pressured.

**Is the MP3 encoded once or once per backend?**
Once per call. `Call.MP3()` memoises the encoded bytes behind a mutex, so the
first backend that needs audio pays for the encode and all others — including
retries and additional backends — reuse the cached slice. A metadata-only webhook
that never requests audio triggers no encode at all.

**How long does the Manager keep retrying a failed upload?**
`MaxRetries+1` attempts per backend (default 4 total), sleeping `RetryBase`
between attempts and doubling it each time — 2s, 4s, 8s by default. After the
budget is exhausted the call is counted as failed for that backend and abandoned.
It is never retried indefinitely, and a failure on one backend doesn't affect the
others.

**Can I stop a specific talkgroup from ever being uploaded?**
Yes. Set `Stream=false` on the talkgroup (a column in the CSV/JSON talkgroup file
or the `stream` field in JSON). `dispatch` checks `cc.Talkgroup.Stream` before
enqueuing, so a `Stream=false` talkgroup's calls are dropped ahead of every
backend at once. The flag defaults to `true` for backward compatibility.

## Series navigation

**Part 13 of 14** · ←
[Part 12: Live Listening]({{ '/blog/deep-dives/recording-streaming-12-live-listening/' | relative_url }})
· Next →
[Part 14: The Aggregator Backends]({{ '/blog/deep-dives/recording-streaming-14-aggregator-backends/' | relative_url }})
