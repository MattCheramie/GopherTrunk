---
slug: goroutines
title: Goroutines
entry_type: concept
category: concurrency-execution
description: A goroutine is a lightweight task managed by the Go runtime, which multiplexes many of them onto a few OS threads and has them communicate over channels.
keywords: goroutine, Go, channel, CSP, communicating sequential processes, lightweight thread, scheduler, concurrency, share by communicating
aka: [goroutines]
autolink: true
infobox:
  - { label: Category, value: Concurrency primitive }
  - { label: Language, value: "Go" }
  - { label: Managed by, value: "The Go runtime, not the OS" }
  - { label: Cost, value: "Tiny stacks — hundreds of thousands are cheap" }
  - { label: Communicate via, value: "Channels (CSP)" }
  - { label: Slogan, value: "Share memory by communicating" }
see_also: [concurrency, threads, async-programming, go-language, garbage-collection]
related_lessons:
  - { title: "Concurrency and parallelism", url: /learn/intro-software-dev/concurrency-models/ }
  - { title: "Concurrency and pipelines", url: /learn/intro-software-dev/concurrency-and-pipelines/ }
related_reading:
  - { title: "SDR Internals, Part 3: The SDR pool, streaming & concurrency", url: /blog/deep-dives/sdr-internals-03-sdr-pool-streaming-concurrency/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Go_(programming_language)#Concurrency
---

**A goroutine** is a lightweight task managed by the [Go](/reference/go-language/)
runtime rather than the operating system. They are so cheap that you can run hundreds of
thousands at once, and they communicate over *channels* instead of sharing memory
directly.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="The Go runtime multiplexes many goroutines onto a few OS threads, and goroutines pass values through a channel." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="18" width="80" height="20" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="60" y="32" font-size="8">goroutine</text>
    <rect x="20" y="44" width="80" height="20" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="60" y="58" font-size="8">goroutine</text>
    <rect x="20" y="70" width="80" height="20" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="60" y="84" font-size="8">goroutine</text>
    <rect x="20" y="96" width="80" height="20" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="60" y="110" font-size="8">goroutine</text>
    <line x1="100" y1="28" x2="160" y2="55" stroke="currentColor" stroke-width="1.1"/><line x1="100" y1="54" x2="160" y2="60" stroke="currentColor" stroke-width="1.1"/><line x1="100" y1="80" x2="160" y2="72" stroke="currentColor" stroke-width="1.1"/><line x1="100" y1="106" x2="160" y2="78" stroke="currentColor" stroke-width="1.1"/>
    <rect x="160" y="48" width="90" height="36" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="205" y="63" font-size="8">runtime</text><text x="205" y="76" font-size="7">few OS threads</text>
    <line x1="250" y1="66" x2="310" y2="66" stroke="currentColor" stroke-width="1.1"/>
    <rect x="310" y="50" width="120" height="32" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="370" y="70" font-size="8">channel</text>
  </g>
</svg>
<figcaption>The Go runtime multiplexes many cheap goroutines onto a few OS threads; channels carry values between them.</figcaption>
</figure>

## How it works

You start a goroutine by prefixing a function call with `go`. Each begins with a tiny
stack that grows as needed, and the Go runtime's scheduler multiplexes many goroutines
onto a small pool of OS [threads](/reference/threads/), so they cost almost nothing
compared with one OS thread per task.[^wiki] Goroutines coordinate through **channels** — typed
pipes where one goroutine sends a value and another receives it. The channel handles
synchronization, so there is no explicit lock and no data race on the data that flows
through it. This is the *Communicating Sequential Processes* (CSP) model, captured by the
slogan: "Do not communicate by sharing memory; share memory by communicating."[^wiki]

## Trade-offs

Goroutines make concurrent **pipelines** natural to express and are cheap enough that you
can spawn one per connection or per work item without worrying about cost. The runtime
can spread them across cores for real [parallelism](/reference/concurrency/) when work is
CPU-bound, and channels make whole categories of race bugs impossible by design. The
trade-offs are the small runtime scheduler overhead and the [garbage
collector](/reference/garbage-collection/) that supports them, plus the discipline of
choosing channels over shared state.

## In practice

Goroutines and channels are central to [Go](/reference/go-language/)'s character and a
big reason it is common in networked and streaming systems. In GopherTrunk they map
cleanly onto the concurrent decode pipelines an SDR scanner runs at once — a capture
stage feeding a DSP stage feeding a decode stage, each a goroutine, glued together by
channels.

## Sources

[^wiki]: [Go (programming language) — Concurrency](https://en.wikipedia.org/wiki/Go_(programming_language)#Concurrency) — Wikipedia, on goroutines, channels, and Go's CSP-based "share memory by communicating" model.
