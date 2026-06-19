---
slug: c-language
title: C
entry_type: language
category: programming-languages
description: C is a small, fast, statically typed compiled language with manual memory management that sits at the core of nearly every operating system, language runtime and embedded device.
keywords: C, C programming, compiled, manual memory management, pointers, systems programming, embedded, Unix, K&R
aka: [C]
autolink: true
infobox:
  - { label: Paradigm, value: "Imperative, procedural, structured" }
  - { label: Typing, value: "Static, weak" }
  - { label: Appeared, value: "1972 (Bell Labs)" }
  - { label: Designed by, value: "Dennis Ritchie" }
  - { label: Compilation, value: "Ahead-of-time to native machine code" }
  - { label: Memory, value: "Manual (malloc / free)" }
  - { label: Notable uses, value: "Operating systems, drivers, embedded, runtimes" }
see_also: [compiler, memory-management, cpp-language, rust-language, go-language, static-binary, type-system]
related_lessons:
  - { title: "A tour of today's major languages", url: /learn/intro-software-dev/language-tour/ }
  - { title: "Memory management across languages", url: /learn/intro-software-dev/memory-management/ }
related_reading:
  - { title: "Build in the Open, Part 2: Choosing your language, platforms & stack", url: /blog/tutorials/build-in-the-open-02-choosing-language-platforms-stack/ }
cite_urls:
  - https://en.cppreference.com/w/c
  - https://en.wikipedia.org/wiki/C_(programming_language)
---

**C** is a small, fast, statically typed compiled language created at Bell Labs in the
1970s.[^wiki] It is the bedrock of modern computing: almost every operating system, language
runtime and embedded device has C at its core.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="C source is compiled and linked ahead of time into a native executable; the programmer manages memory by hand with malloc and free." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="50" width="64" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="52" y="74">.c source</text>
    <line x1="84" y1="70" x2="124" y2="70" stroke="currentColor" stroke-width="1.1"/><text x="104" y="62" font-size="8">compile</text>
    <rect x="124" y="50" width="70" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="159" y="74">object</text>
    <line x1="194" y1="70" x2="234" y2="70" stroke="currentColor" stroke-width="1.1"/><text x="214" y="62" font-size="8">link</text>
    <rect x="234" y="50" width="86" height="40" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="277" y="68">native</text><text x="277" y="80">executable</text>
    <line x1="277" y1="90" x2="277" y2="112" stroke="currentColor" stroke-width="1.1" stroke-dasharray="3 3"/>
    <text x="277" y="128" font-size="8" fill-opacity="0.7">manual malloc / free</text>
  </g>
</svg>
<figcaption>C compiles straight to native code; memory is allocated and freed by hand.</figcaption>
</figure>

## Overview

C [compiles](/reference/compiler/) ahead of time directly to native machine code, with a
tiny runtime and almost no abstraction between the code and the hardware.[^ref] It is
[statically typed](/reference/type-system/), gives the programmer direct access to memory
through pointers, and is portable across virtually every platform. That minimalism is the
point: C is small enough to learn the whole language, and fast and predictable enough to
build everything else on top of.

## Strengths and trade-offs

C's strengths are speed, ubiquity, portability and a tiny footprint — it runs where
nothing else will. The cost is **safety**: [memory management](/reference/memory-management/)
is fully manual, so buffer overflows, use-after-free and dangling pointers are easy to
write and account for whole categories of security bugs that C does nothing to prevent.
It is also low-level, with few of the conveniences modern languages offer. These risks
are exactly what later languages set out to address —
[C++](/reference/cpp-language/) adds higher-level abstractions over the same model, while
[Rust](/reference/rust-language/) keeps the speed but enforces memory safety at compile
time.

## Where it's used

C remains the default for operating-system kernels, device drivers, language runtimes,
and tight embedded work where every byte and cycle counts. Its stable, minimal interface
also makes it the common tongue that other languages bind to when they need to call
native code.

## Sources

[^ref]: [C reference](https://en.cppreference.com/w/c) — cppreference, the standard-tracking reference for the C language and library.
[^wiki]: [C (programming language)](https://en.wikipedia.org/wiki/C_(programming_language)) — Wikipedia, for history and design background.
