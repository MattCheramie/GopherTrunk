---
slug: garbage-collection
title: Garbage collection
entry_type: concept
category: language-internals
description: Garbage collection (GC) is automatic memory management that reclaims heap memory a program can no longer reach, freeing developers from manual allocation and freeing at the cost of some runtime overhead.
keywords: garbage collection, GC, automatic memory management, heap, mark and sweep, generational, reference counting, memory leak, stop-the-world
aka: [garbage collection, GC]
autolink: true
infobox:
  - { label: Category, value: Memory management }
  - { label: Frees developer from, value: Manual allocate / free }
  - { label: Common strategies, value: "Mark-and-sweep, generational, reference counting" }
  - { label: Trade-off, value: "Safety & convenience vs pause time / overhead" }
  - { label: Garbage-collected, value: "Go, Java, Python, C#, JavaScript" }
  - { label: Opposite of, value: "Manual memory management (C, C++)" }
see_also: [memory-management, static-binary, compiler, go-language, java-language, c-language, type-system]
related_lessons:
  - { title: "Memory management across languages", url: /learn/intro-software-dev/memory-management/ }
  - { title: "Performance vs productivity", url: /learn/intro-software-dev/performance-vs-productivity/ }
external:
  - { title: "Garbage collection (computer science) — Wikipedia", url: https://en.wikipedia.org/wiki/Garbage_collection_(computer_science) }
---

**Garbage collection** (**GC**) is a form of automatic [memory management](/reference/memory-management/)
in which the language runtime reclaims memory the program can no longer reach, so the
programmer never has to free it by hand.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="Root references reach some heap objects, which the collector keeps; objects with no path from the roots are unreachable and reclaimed." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="55" width="50" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="45" y="74">roots</text>
    <circle cx="160" cy="40" r="18" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.3"/><text x="160" y="43">live</text>
    <circle cx="160" cy="100" r="18" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.3"/><text x="160" y="103">live</text>
    <circle cx="250" cy="70" r="18" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.3"/><text x="250" y="73">live</text>
    <line x1="70" y1="65" x2="142" y2="45" stroke="currentColor" stroke-width="1.1"/><line x1="70" y1="75" x2="142" y2="98" stroke="currentColor" stroke-width="1.1"/><line x1="178" y1="45" x2="232" y2="65" stroke="currentColor" stroke-width="1.1"/>
    <circle cx="360" cy="40" r="18" fill="none" stroke="currentColor" stroke-width="1.1" stroke-dasharray="3 3"/><text x="360" y="43" fill-opacity="0.6">freed</text>
    <circle cx="360" cy="100" r="18" fill="none" stroke="currentColor" stroke-width="1.1" stroke-dasharray="3 3"/><text x="360" y="103" fill-opacity="0.6">freed</text>
    <text x="360" y="135" font-size="8" fill-opacity="0.7">unreachable → reclaimed</text>
  </g>
</svg>
<figcaption>The collector keeps objects reachable from the program's roots and reclaims the rest.</figcaption>
</figure>

## How it works

The runtime periodically determines which heap objects are still reachable by
following references from a set of *roots* (stack variables, globals, registers).
Anything with no path from the roots is garbage and its memory is reclaimed. Common
strategies include **mark-and-sweep** (mark reachable objects, sweep the rest),
**generational** collection (collect short-lived objects more often), and
**reference counting** (track how many references point at each object).

## Trade-offs

GC eliminates entire classes of bugs that plague manual memory code — use-after-free,
double-free, and many memory leaks — and removes a great deal of boilerplate. The price
is runtime overhead and occasional pauses (historically "stop-the-world" collections),
which is why latency-sensitive and systems languages such as [C](/reference/c-language/)
and [C++](/reference/cpp-language/) leave memory to the programmer, and why
[Rust](/reference/rust-language/) achieves safety at compile time without a collector
at all.

## In practice

Most mainstream languages are garbage-collected, including [Go](/reference/go-language/),
[Java](/reference/java-language/), [Python](/reference/python-language/), and C#. Their
collectors have grown sophisticated enough that GC pauses are a non-issue for the vast
majority of applications.
