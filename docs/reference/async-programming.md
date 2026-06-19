---
slug: async-programming
title: Asynchronous programming
entry_type: concept
category: concurrency-execution
description: Asynchronous programming runs many tasks concurrently on a single thread using an event loop, where a task yields control whenever it waits on something slow.
keywords: asynchronous programming, async await, event loop, non-blocking, I/O bound, coroutine, future, promise, concurrency
aka: ["async", "async/await"]
autolink: true
infobox:
  - { label: Category, value: Concurrency model }
  - { label: Mechanism, value: "Event loop + cooperative yielding" }
  - { label: Best for, value: "I/O-bound work (waiting)" }
  - { label: Weak for, value: "CPU-bound work" }
  - { label: Keyword pair, value: "async / await" }
  - { label: Examples, value: "JavaScript, Python asyncio, C#, Rust" }
see_also: [concurrency, threads, goroutines, javascript-language, python-language, csharp-language]
related_lessons:
  - { title: "Concurrency and parallelism", url: /learn/intro-software-dev/concurrency-models/ }
  - { title: "Concurrency and pipelines", url: /learn/intro-software-dev/concurrency-and-pipelines/ }
external:
  - { title: "Asynchronous I/O — Wikipedia", url: https://en.wikipedia.org/wiki/Asynchronous_I/O }
---

**Asynchronous programming** runs many tasks [concurrently](/reference/concurrency/) on
a single thread using an *event loop*: whenever a task waits on something slow, it
yields control back to the loop, which runs other ready tasks until the slow thing
finishes.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="An event loop dispatches a task, which yields on await and lets another ready task run, then resumes when its result arrives." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <circle cx="90" cy="65" r="34" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="90" y="62" font-size="8">event</text><text x="90" y="74" font-size="8">loop</text>
    <line x1="124" y1="55" x2="180" y2="40" stroke="currentColor" stroke-width="1.1"/><text x="155" y="34" font-size="8">run</text>
    <rect x="180" y="28" width="90" height="24" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="225" y="44" font-size="8">task A</text>
    <line x1="270" y1="40" x2="320" y2="40" stroke="currentColor" stroke-width="1.1" stroke-dasharray="3 3"/><text x="300" y="33" font-size="7">await</text>
    <rect x="320" y="28" width="110" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="375" y="44" font-size="7">waiting (I/O)</text>
    <line x1="124" y1="78" x2="180" y2="95" stroke="currentColor" stroke-width="1.1"/>
    <rect x="180" y="83" width="90" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="225" y="99" font-size="8">task B runs</text>
  </g>
</svg>
<figcaption>When a task hits an await, it yields to the loop, which runs another ready task until the result arrives.</figcaption>
</figure>

## How it works

Code marks slow operations with `await`. When a task reaches one, it suspends and hands
the single thread back to the event loop, which picks another task that is ready to make
progress; when the awaited operation completes, the loop resumes the original task where
it left off. No operating-system [thread](/reference/threads/) is blocked sitting idle,
so one thread can keep thousands of in-flight operations moving. Because there is only
one thread of execution, there is no shared-memory data race between the tasks.

## Trade-offs

Async excels at **I/O-bound** work — thousands of concurrent network connections or
disk reads on a single thread — because such workloads spend most of their time waiting,
not computing. It does little for **CPU-bound** work: a long computation still hogs the
single thread and blocks every other task, since cooperative scheduling only switches at
`await` points. Async code can also be harder to reason about, and a blocking call
slipped into an async path stalls everything.

## In practice

[JavaScript](/reference/javascript-language/) (Node.js and the browser),
[Python](/reference/python-language/) (`asyncio`), [C#](/reference/csharp-language/),
and Rust all offer `async`/`await`. For CPU-bound parallelism you reach instead for
[threads](/reference/threads/) or lightweight tasks like
[goroutines](/reference/goroutines/), which can spread work across cores.
