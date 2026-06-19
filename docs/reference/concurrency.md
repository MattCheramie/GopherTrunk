---
slug: concurrency
title: Concurrency vs parallelism
entry_type: concept
category: concurrency-execution
description: Concurrency is structuring a program so multiple tasks can make independent progress, while parallelism is actually running multiple computations at the same instant on multiple cores.
keywords: concurrency, parallelism, threads, async, goroutines, channels, data race, event loop, multiple cores
aka: ["concurrency", "parallelism"]
autolink: true
infobox:
  - { label: Category, value: Execution model }
  - { label: Concurrency, value: "Dealing with many tasks at once (structure)" }
  - { label: Parallelism, value: "Doing many things at once (execution)" }
  - { label: Needs multiple cores, value: "Parallelism yes; concurrency no" }
  - { label: Common models, value: "Threads, async/await, channels, actors" }
  - { label: The core hazard, value: "Data races" }
see_also: [threads, async-programming, goroutines, go-language, rust-language, javascript-language]
related_lessons:
  - { title: "Concurrency and parallelism", url: /learn/intro-software-dev/concurrency-models/ }
  - { title: "Concurrency and pipelines", url: /learn/intro-software-dev/concurrency-and-pipelines/ }
external:
  - { title: "Concurrency (computer science) — Wikipedia", url: https://en.wikipedia.org/wiki/Concurrency_(computer_science) }
---

**Concurrency** is structuring a program so that multiple tasks can be in progress and
advance independently; **parallelism** is actually running multiple computations at the
same instant. Concurrency is about structure; parallelism is about execution, and the
two are related but not the same.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="Concurrency interleaves two tasks on one core; parallelism runs them at the same time on two cores." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="115" y="18" font-size="8" fill-opacity="0.7">concurrency (1 core)</text>
    <rect x="20" y="28" width="40" height="22" rx="3" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="40" y="43" font-size="7">A</text>
    <rect x="62" y="28" width="40" height="22" rx="3" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="82" y="43" font-size="7">B</text>
    <rect x="104" y="28" width="40" height="22" rx="3" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="124" y="43" font-size="7">A</text>
    <rect x="146" y="28" width="40" height="22" rx="3" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="166" y="43" font-size="7">B</text>
    <text x="115" y="80" font-size="8" fill-opacity="0.7">parallelism (2 cores)</text>
    <rect x="20" y="88" width="120" height="22" rx="3" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="80" y="103" font-size="7">task A</text>
    <rect x="20" y="112" width="120" height="14" rx="3" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="80" y="123" font-size="7">task B</text>
    <text x="300" y="70" font-size="8" fill-opacity="0.7">structure vs execution</text>
  </g>
</svg>
<figcaption>Concurrency interleaves tasks (even on one core); parallelism runs them at the same instant on several cores.</figcaption>
</figure>

## The distinction

Even on a single CPU core, a concurrent program can interleave work — run task A, pause
it while it waits for I/O, switch to task B — so the tasks *deal with* progress
together. Parallelism *does* several things at once and therefore requires multiple
cores. You can have concurrency without parallelism (one core juggling tasks), and a
good concurrency model makes tasks the runtime can *also* run in parallel when cores
are available. As Rob Pike put it: concurrency is a way to structure things; if it
works, parallelism may be a free bonus.

## Models and the shared hazard

Languages offer different concurrency models: OS [threads](/reference/threads/) with
locks, single-threaded [async/await](/reference/async-programming/) event loops, Go's
[goroutines](/reference/goroutines/) and channels, and the share-nothing actor model.
What they are all really fighting is the **data race** — two tasks touching the same
memory with at least one writing, with no synchronization, producing undefined results.
Each model is, at heart, a different strategy for avoiding unsynchronized shared writes.

## In practice

The right model follows the workload. **I/O-bound** work — waiting on networks or disks
— favors async or lightweight tasks like [goroutines](/reference/goroutines/);
**CPU-bound** work like DSP favors real parallelism across cores; fault-tolerant
systems favor actors. A streaming radio pipeline in [Go](/reference/go-language/) runs
capture, DSP, and decode stages concurrently and lets the runtime spread them across
cores in parallel.
