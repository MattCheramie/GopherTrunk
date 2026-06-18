---
slug: concurrency-and-pipelines
title: "Concurrency & streaming patterns"
description: Pipelines, producer/consumer with bounded buffers, and pub/sub fan-out are the patterns behind real-time signal software. Learn backpressure, queues, and why this shape dominates DSP.
keywords: pipeline pattern, dataflow, producer consumer, bounded buffer, backpressure, publish subscribe, fan-out, actor model, channels, concurrency, DSP pipeline
level: advanced
status: full
faq:
  - q: "What is backpressure?"
    a: "**Backpressure** is the mechanism by which a slow consumer signals a fast producer to slow down. With a bounded buffer between them, once the buffer fills the producer blocks (or drops data) until the consumer catches up. Without backpressure, a fast SDR feeding a slow decoder would grow an unbounded queue until memory runs out — so backpressure is what keeps a real-time pipeline stable."
  - q: "Why do pipelines dominate real-time signal software?"
    a: "Digital signal processing is naturally a sequence of stages — capture, filter, decimate, demodulate, decode — where each stage transforms a stream and hands it on. The pipeline pattern maps directly onto that, lets each stage run concurrently on its own core, and uses bounded queues between stages to balance speed. It matches both the math and the hardware, which is why it is the default shape."
  - q: "What is the difference between producer/consumer and publish/subscribe?"
    a: "In **producer/consumer**, items flow through a shared queue and each item is handled by *one* consumer — work is divided up. In **publish/subscribe (fan-out)**, each event is delivered to *every* subscriber — the same data is broadcast to many independent handlers. One splits a workload; the other distributes copies of events."
---
# Concurrency & streaming patterns

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Pipeline** — stages connected by queues, each transforming a stream and passing it on, exactly like a DSP chain. **Producer/consumer with bounded buffers** — backpressure stops a fast source from overrunning a slow sink. **Pub/sub fan-out** — one event delivered to many independent subscribers.
</div>

The GoF patterns so far described single-threaded object designs. Real-time signal software has a different dominant problem: a relentless stream of data arriving faster than any one step can fully process, which must flow through several stages *at once*. A software radio captures millions of samples per second and must filter, demodulate, and decode them continuously, without falling behind. The patterns that solve this are about **concurrency and dataflow** — pipelines, producers and consumers, fan-out, and the idea of backpressure. This lesson builds on [concurrency models](/learn/intro-software-dev/concurrency-models/) and maps directly onto the [demodulation pipeline](/learn/rf-sdr/demodulation-pipeline/) from the [RF & SDR path](/learn/rf-sdr/).

## The pipeline (dataflow) pattern

A **pipeline** decomposes processing into a series of **stages**, each of which takes a stream of items, transforms them, and passes the result to the next stage. The stages are connected by **queues** (often called channels). Each stage runs concurrently, so while stage 3 works on item 10, stage 1 is already pulling in item 12.

This is not a metaphor for DSP — it *is* DSP. A receive chain is literally a pipeline:

```text
capture  →[queue]→  filter/decimate  →[queue]→  demodulate  →[queue]→  decode
 (SDR)                (lower the rate)            (to audio)            (to data)
```

Each arrow is a queue; each box is a stage running on its own thread or task. Why this shape wins:

- **Concurrency for free.** Stages run in parallel on different cores, so total throughput is set by the *slowest* stage, not the sum of all of them.
- **Separation of concerns.** Each stage does one job and knows nothing about its neighbours except the data format on the queue — very low coupling.
- **Composability.** Insert, remove, or swap a stage (add a new filter, change the demodulator) by rewiring queues, not rewriting the whole flow.

A stage is, in effect, both a consumer of its input queue and a producer for its output queue — which brings us to the core relationship.

## Producer/consumer and bounded buffers

The fundamental two-party version is **producer/consumer**: one part (the producer) generates items and puts them on a queue; another part (the consumer) takes items off and processes them. The queue between them decouples their timing — the producer does not have to wait for the consumer to finish each item, and vice versa.

The critical design choice is making that queue a **bounded buffer** — a queue with a fixed maximum size. This is the single most important decision in a streaming system, because it is what creates **backpressure**:

> **Backpressure** is the feedback that makes a fast producer slow down to match a slow consumer. When the bounded buffer fills, the producer blocks until space frees up — automatically pacing the whole pipeline to its slowest stage.

Consider an SDR producing samples far faster than a CPU-bound P25 decoder can consume them. With an **unbounded** queue, the producer never waits, so the backlog grows without limit until the program exhausts memory and crashes — latency climbing the whole time. With a **bounded** buffer:

```text
buffer = BoundedQueue(capacity = 1024)

producer:  loop { samples = sdr.read(); buffer.put(samples) }  // blocks when full
consumer:  loop { samples = buffer.get(); decode(samples) }    // blocks when empty
```

When the decoder lags, the buffer fills, `put` blocks, and the producer is forced to wait — the system self-regulates. The bound also forces an honest design conversation: if the producer *cannot* be slowed (a real radio will not stop emitting samples for you), you must choose a strategy when the buffer is full — **block**, **drop oldest/newest**, or **expand capacity** — and for real-time radio the answer is often to drop samples deliberately rather than fall behind forever. Either way, the bound makes the decision explicit instead of letting an unbounded queue hide a slow crash.

## Channels and how stages communicate

The queues between stages are usually implemented as **channels**: thread-safe conduits you send to and receive from, with the locking and signalling handled for you. Channels are the backbone of the message-passing style of concurrency (covered in [concurrency models](/learn/intro-software-dev/concurrency-models/)), and many languages provide them directly — Go's `chan`, for instance, is a bounded or unbounded channel built into the language.

The reason streaming systems favour channels over shared mutable memory is that **sharing data by communicating** sidesteps most concurrency bugs. If a sample buffer is only ever owned by one stage at a time and handed off through a channel, two threads never touch it simultaneously, so there is no data race to guard with locks. The queue *is* the synchronization. This is why "don't communicate by sharing memory; share memory by communicating" became a guiding slogan for pipeline-heavy software.

## Fan-out, fan-in, and publish/subscribe

Straight-line pipelines are the start; real systems branch.

- **Fan-out** — one stage's output is split across *several* consumers running in parallel, so a CPU-heavy step (like decoding) can be scaled across multiple workers pulling from the same queue. Each item still goes to exactly *one* worker; this is producer/consumer with many consumers, used to parallelize a bottleneck.
- **Fan-in** — multiple producers' outputs are merged into a single queue for a downstream stage (for example, several tuned channels feeding one logger).
- **Publish/subscribe** — distinct from fan-out: here each event is delivered to *every* subscriber, not just one. This is the concurrent cousin of the Observer pattern. When a call is decoded, the same event is broadcast to the UI, the recorder, and a webhook — each gets its own copy and reacts independently.

The difference matters: **fan-out divides a workload** (one item, one worker), while **pub/sub broadcasts events** (one event, all subscribers). Both are everyday tools in a signal pipeline — fan-out to keep up with the sample rate, pub/sub to notify everything that cares about a decoded result.

## The actor idea

A complementary model is the **actor**. An actor is an independent unit that owns its private state and communicates *only* by sending and receiving messages through its own mailbox (a queue). It never shares memory; it just processes one message at a time and may send messages to other actors.

Actors fit signal software well because each stage — a tuner, a decoder, a channel follower — can be an actor with its own state and inbox, wired together by message passing. Since an actor handles one message at a time, its internal state needs no locks, and the system scales by adding more actors. Whether you call them stages, goroutines, or actors, the underlying recipe is the same and is why this shape dominates real-time signal software: **independent units, connected by bounded queues, paced by backpressure.** It matches the math (a chain of transforms), the hardware (many cores), and the timing reality (data never stops arriving).

<div class="knowledge-check" data-quiz data-correct-msg="Right — a bounded buffer creates backpressure, forcing the fast producer to wait for the slow consumer." markdown="0">
  <p class="knowledge-check__q">Quick check: A fast SDR feeds a slow decoder through a bounded queue. What happens when the queue fills?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The queue grows without limit until memory runs out</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The producer blocks (backpressure) until the consumer frees space</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The consumer automatically speeds up to match the producer</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Pipeline** — processing as concurrent stages connected by queues; this *is* the shape of a DSP chain (capture → filter → demodulate → decode).
- **Producer/consumer** — a queue decouples a producer's timing from a consumer's; the foundational two-party streaming relationship.
- **Bounded buffers and backpressure** — a fixed-size queue forces a fast producer to wait for a slow consumer, keeping the system stable instead of growing an unbounded backlog.
- **Channels** — thread-safe queues that let stages share data by communicating, avoiding data races without manual locks.
- **Fan-out vs pub/sub** — fan-out divides one workload across many workers (one item, one worker); pub/sub broadcasts each event to all subscribers.
- **Actors** — independent message-driven units with private state; another expression of the same "units connected by queues" recipe that dominates real-time signal software.

Next up: zooming out from individual processes to whole-system shapes — layered, client-server, event-driven, and plugin architectures. See [Architecture patterns](/learn/intro-software-dev/architectural-patterns/).
