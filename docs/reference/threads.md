---
slug: threads
title: Threads
entry_type: concept
category: concurrency-execution
description: A thread is an independent flow of execution within a process; multiple threads share the process's memory and can run on different CPU cores at once.
keywords: thread, OS thread, process, shared memory, lock, mutex, data race, deadlock, parallelism, context switch
aka: ["thread", "OS thread"]
autolink: true
infobox:
  - { label: Category, value: Execution unit }
  - { label: Lives in, value: "A process" }
  - { label: Shares, value: "The process's memory" }
  - { label: Managed by, value: "The operating system" }
  - { label: Parallel, value: "Yes, across CPU cores" }
  - { label: Main hazards, value: "Data races, deadlocks" }
see_also: [concurrency, async-programming, goroutines, memory-management, c-language, rust-language]
related_lessons:
  - { title: "Concurrency and parallelism", url: /learn/intro-software-dev/concurrency-models/ }
  - { title: "Concurrency and pipelines", url: /learn/intro-software-dev/concurrency-and-pipelines/ }
external:
  - { title: "Thread (computing) — Wikipedia", url: https://en.wikipedia.org/wiki/Thread_(computing) }
---

**A thread** is an independent flow of execution inside a process. The operating system
can give one process several threads that share the same memory and run on different
CPU cores at the same time — the oldest and most direct route to
[parallelism](/reference/concurrency/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="One process contains shared memory and three threads of execution running within it." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="30" y="20" width="400" height="90" rx="6" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="70" y="35" font-size="8" fill-opacity="0.7">process</text>
    <rect x="50" y="48" width="100" height="44" rx="4" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.2"/><text x="100" y="66">shared</text><text x="100" y="78">memory</text>
    <line x1="150" y1="60" x2="200" y2="55" stroke="currentColor" stroke-width="1.1"/><line x1="150" y1="70" x2="200" y2="75" stroke="currentColor" stroke-width="1.1"/><line x1="150" y1="80" x2="200" y2="95" stroke="currentColor" stroke-width="1.1"/>
    <rect x="200" y="44" width="100" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="250" y="59" font-size="8">thread 1</text>
    <rect x="200" y="69" width="100" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="250" y="84" font-size="8">thread 2</text>
    <rect x="320" y="56" width="100" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="370" y="71" font-size="8">thread 3</text>
  </g>
</svg>
<figcaption>One process, several threads — all sharing the same memory, which is exactly where the danger lies.</figcaption>
</figure>

## How it works

A process starts with one thread and can spawn more, and the OS scheduler hands each
thread time on a core. Because threads in a process share its
[memory](/reference/memory-management/), they communicate simply by reading and writing
the same variables — fast and powerful, but the source of the danger. When two threads
touch the same data and at least one writes, with no coordination, you have a **data
race**, and the result is undefined: a half-updated value, a counter that loses
increments, a crash that appears once a week.

## Trade-offs

Threads are genuinely parallel and map directly onto hardware cores, so they suit
CPU-bound work. The traditional fix for races is a **lock** (mutex): only one thread may
hold the lock and touch the shared data at a time. Locks work but bring their own
hazards — **deadlock** (two threads each waiting on a lock the other holds), forgotten
locks, and contention that erodes performance. OS threads are also relatively heavy, so
running hundreds of thousands of them is impractical.

## In practice

[C](/reference/c-language/), [C++](/reference/cpp-language/),
[Rust](/reference/rust-language/), and Java expose OS threads directly, with Rust's type
system preventing many data races at compile time. To avoid heavy threads and explicit
locks, other models build on top of them: [async/await](/reference/async-programming/)
multiplexes tasks onto a single thread, and [goroutines](/reference/goroutines/)
multiplex many lightweight tasks onto a few OS threads.
