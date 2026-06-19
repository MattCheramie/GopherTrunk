---
slug: memory-management
title: Memory management
entry_type: concept
category: language-internals
description: Memory management is how a program hands out memory when it needs it and reclaims it when done, ranging from manual allocation to garbage collection to compile-time ownership.
keywords: memory management, stack and heap, manual memory, malloc free, garbage collection, ownership, borrow checker, memory leak, use after free
aka: [memory management]
autolink: true
infobox:
  - { label: Category, value: Runtime resource handling }
  - { label: Regions, value: "Stack (auto) and heap (managed)" }
  - { label: Strategies, value: "Manual, garbage collection, ownership" }
  - { label: Key axis, value: "Safety vs control vs determinism" }
  - { label: Manual, value: "C, C++" }
  - { label: Automatic, value: "Go, Java, Python (GC); Rust (ownership)" }
see_also: [garbage-collection, type-system, static-binary, c-language, cpp-language, rust-language, go-language]
related_lessons:
  - { title: "Memory management across languages", url: /learn/intro-software-dev/memory-management/ }
  - { title: "Performance vs productivity", url: /learn/intro-software-dev/performance-vs-productivity/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Memory_management
---

**Memory management** is how a running program obtains memory to hold its data and
reclaims it once that data is no longer needed — a defining characteristic of a
language that shapes its safety, speed, and fitness for real-time work.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A program uses a fast auto-freed stack and a flexible heap that must be reclaimed manually, by garbage collection, or by ownership." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="30" y="30" width="110" height="80" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="85" y="24" font-size="8">stack</text>
    <line x1="40" y1="50" x2="130" y2="50" stroke="currentColor" stroke-width="1"/><line x1="40" y1="68" x2="130" y2="68" stroke="currentColor" stroke-width="1"/><line x1="40" y1="86" x2="130" y2="86" stroke="currentColor" stroke-width="1"/><text x="85" y="104" font-size="7" fill-opacity="0.7">auto-freed</text>
    <rect x="320" y="30" width="110" height="80" rx="5" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.3"/><text x="375" y="24" font-size="8">heap</text>
    <circle cx="355" cy="58" r="10" fill="none" stroke="currentColor" stroke-width="1.1"/><circle cx="395" cy="50" r="10" fill="none" stroke="currentColor" stroke-width="1.1"/><circle cx="378" cy="84" r="10" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="375" y="104" font-size="7" fill-opacity="0.7">must be reclaimed</text>
    <line x1="140" y1="70" x2="320" y2="70" stroke="currentColor" stroke-width="1.1"/><text x="230" y="62" font-size="8">allocate</text>
  </g>
</svg>
<figcaption>The stack is fast and freed automatically; the flexible heap is where the hard reclamation problems live.</figcaption>
</figure>

## Stack vs heap

A program's memory splits into two regions. The **stack** holds local variables and
function call frames; a frame is pushed on a call and popped on return, so its memory
is freed automatically and allocation is nearly free — but everything on it must have a
size known at compile time and lives only as long as its function. The **heap** is a
flexible pool for data whose size or lifetime is not known up front; it must be
allocated and eventually released, and **all the hard memory problems live on the
heap** because something must decide when each piece is no longer needed.[^wiki]

## Three strategies

Languages differ chiefly in how they reclaim heap memory.
**Manual management** — [C](/reference/c-language/) and [C++](/reference/cpp-language/)
— gives you `malloc`/`free` and total control, but risks leaks, use-after-free, double
frees, and buffer overflows.[^wiki] **[Garbage collection](/reference/garbage-collection/)** —
[Go](/reference/go-language/), [Java](/reference/java-language/),
[Python](/reference/python-language/) — automatically frees unreachable memory,
eliminating whole bug classes at the cost of occasional, non-deterministic pauses.
**Ownership** — [Rust](/reference/rust-language/) — uses a compile-time borrow checker
to free memory at known scope exits, achieving safety with no garbage collector and
deterministic timing.

## Why it matters

The key axis for real-time work is *determinism*. Manual and ownership models reclaim
memory at predictable moments, so a software-defined radio pipeline never stalls
mid-buffer; garbage collection can pause at an unpredictable
moment and drop samples. This is why tight DSP gravitates toward C or Rust, while a
GC'd language like Go is fine for the control and orchestration layers around it.

## Sources

[^wiki]: [Memory management](https://en.wikipedia.org/wiki/Memory_management) — Wikipedia, on stack versus heap, manual allocation, and automatic reclamation strategies.
