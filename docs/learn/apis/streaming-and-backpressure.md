---
slug: streaming-and-backpressure
title: Streaming & backpressure
description: What happens when the producer is faster than the consumer — buffering, blocking, and dropping compared, why every real stream needs an explicit plan, and how to choose per kind of data.
keywords: backpressure, streaming data, producer consumer, bounded buffer, drop oldest, flow control, slow consumer problem
level: advanced
status: full
prereq:
  - polling-vs-push
  - websockets
---

# Streaming & backpressure

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Every stream has a **producer** and a **consumer**, and sooner or later the
producer runs faster. Exactly three things can happen to the excess: **buffer** it
(borrowing memory against the future), **block** the producer (pushing the
problem upstream — real **flow control**), or **drop** data (sacrificing
completeness for freshness). "Sooner or later" is a law, not a risk — so every
real stream needs an **explicit, chosen** policy with a **bounded buffer**, and
the right choice depends on whether the data is a **must-deliver record** or a
**perishable sample**.
</div>

This is the unit's engineering capstone. Push transports solved *how* events
reach a consumer; this lesson is about the moment the consumer can't keep up —
which, on a long enough timeline, always arrives. A scanner daemon lives this
problem daily: an SDR produces millions of samples per second whether or not
anything downstream is ready.

## The mismatch is guaranteed

Why "always"? Because producer and consumer rates are set by different worlds.
The radio spectrum doesn't slow down because your disk got busy; a burst of
simultaneous calls doesn't wait for a subscriber on hotel Wi-Fi. Even matched
average rates guarantee *momentary* mismatches — a garbage-collection pause, a
network hiccup, one slow browser tab. A stream designed only for the matched
case hasn't been designed; it's been postponed.

## The three options — there is no fourth

When the consumer lags, each unit of excess data must be **buffered**,
**blocked on**, or **dropped**. Everything else is one of these wearing a
costume.

| Policy | What happens | Buys you | Costs you |
|--------|--------------|----------|-----------|
| **Buffer** | Queue the excess in memory/disk | Absorbs bursts; nothing lost | Memory; growing **latency**; only defers the choice |
| **Block** | Producer waits until the consumer drains | Nothing lost; bounded memory | Producer stalls — fatal for real-time sources |
| **Drop** | Discard some data by rule | Bounded memory *and* bounded latency | Data loss — by design, not accident |

Two truths sharpen the table. First, **an unbounded buffer is not a policy** —
it's the decision to crash later instead of deciding now: memory grows, latency
grows with it (a "live" feed running minutes behind is often worse than a lossy
one), and the process eventually dies at the worst time. Every production queue
is bounded, and the real policy is what happens when it fills. Second,
**blocking propagates**: if stage B blocks stage A, and A feeds from real time,
A must itself buffer, block, or drop. Backpressure — the useful jargon — is
exactly this: the consumer's slowness pushed *back* up the pipeline until
something absorbs it. TCP does this natively (its flow-control window is the
network's built-in backpressure — see
[TCP & UDP](/learn/networking/tcp-and-udp/)), which is why a slow WebSocket or
SSE reader shows up in the *server* as writes that won't complete.

## Dropping well is a craft

When drop you must, *which* data matters:

- **Drop-newest** (reject on arrival) keeps the oldest queued items — fine for
  work queues, wrong for live views, which end up showing the stale past.
- **Drop-oldest** keeps the queue fresh — the usual choice for live telemetry.
  The subtlety: each drop mid-queue is a **discontinuity** the consumer sees
  later, and it looks like corruption unless the design says otherwise. A gap in
  an IQ sample stream, for instance, doesn't just lose data — it breaks the
  continuity every decoder downstream assumes, which is why sample drops must be
  *counted and surfaced*, never silent.
- **Conflate** — when only the latest state matters (a spectrum display, a
  gauge), replace the queued update with the new one. Effectively drop-oldest
  with a buffer of one, and often the perfect answer for UI streams.

> Rule of thumb: classify each stream first. **Records** (call events, log
> entries) → bounded buffer + block or persist, because losing one is a lie in
> the history. **Perishable samples** (spectrum frames, audio for live
> listening) → drop-oldest or conflate, because a late sample is worthless and a
> silent stall is worse. Then *surface every drop* — a counter, a log line — so
> "can't keep up" is a fact you observe, not an outage you discover.

## Where you'll meet this

Every transport in this unit hits the wall somewhere: an SSE server whose
subscriber stopped reading holds an ever-filling socket buffer; a WebSocket
server broadcasting spectrum frames must decide per-client what a slow tab
misses; a webhook sender queues undelivered events with a retry budget and a
final "gave up" policy. And inside a scanner daemon, the same shape repeats at
every joint: SDR → DSP → decoder → recorder → subscribers, each stage a
producer to the next. When GopherTrunk's driver logs dropped chunks, that
counter *is* this lesson — the drop policy doing its job and saying so. In Go,
you'll recognise the whole topic as bounded channels and `select` with a
`default:` branch — the [channels lesson](/learn/programming-go/channels/) is
the code-level twin of this one.

<div class="knowledge-check" data-quiz data-correct-msg="Right — an unbounded buffer just converts the problem into growing latency and an eventual crash." markdown="0">
  <p class="knowledge-check__q">Quick check: to "never lose data," a designer gives a live stream an unlimited in-memory queue. What actually happens under sustained overload?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The consumer eventually catches up, so nothing is lost</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Latency and memory grow without bound until the process dies — the decision was deferred, not avoided</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">TCP flow control automatically slows the producer to match</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Producer/consumer rate mismatch is **guaranteed**, not hypothetical — design
  for it up front.
- The complete option space is **buffer, block, or drop**; an unbounded buffer
  is a deferred crash, and blocking **propagates upstream** as backpressure.
- **Bounded buffers** absorb bursts; the real policy is what happens when they
  fill.
- Drop by rule: **drop-oldest** or **conflate** for live views; never drop
  **records** silently.
- Classify streams as **records vs perishable samples**, choose per class, and
  **surface every drop** with counters.

Next up: [What is RPC?](/learn/apis/what-is-rpc/).
