---
slug: cpp-language
title: C++
entry_type: language
category: programming-languages
description: C++ extends C with objects, templates, generics and RAII, offering C-level performance alongside high-level abstractions at the cost of considerable complexity.
keywords: C++, cpp, RAII, templates, generics, object-oriented, systems programming, game engines, DSP, STL
aka: [C++, cpp, "C plus plus"]
autolink: true
infobox:
  - { label: Paradigm, value: "Multi-paradigm: procedural, object-oriented, generic" }
  - { label: Typing, value: "Static, strong (with weak corners from C)" }
  - { label: Appeared, value: "1985 (Bell Labs)" }
  - { label: Designed by, value: "Bjarne Stroustrup" }
  - { label: Compilation, value: "Ahead-of-time to native machine code" }
  - { label: Memory, value: "Manual, but managed via RAII / smart pointers" }
  - { label: Notable uses, value: "Game engines, browsers, trading, DSP" }
see_also: [c-language, rust-language, object-oriented-programming, memory-management, compiler, type-system, go-language]
related_lessons:
  - { title: "A tour of today's major languages", url: /learn/intro-software-dev/language-tour/ }
  - { title: "Memory management across languages", url: /learn/intro-software-dev/memory-management/ }
related_reading:
  - { title: "Build in the Open, Part 2: Choosing your language, platforms & stack", url: /blog/tutorials/build-in-the-open-02-choosing-language-platforms-stack/ }
cite_urls:
  - https://en.cppreference.com/w/cpp
  - https://en.wikipedia.org/wiki/C%2B%2B
---

**C++** is a statically typed, compiled language that extends [C](/reference/c-language/)
with objects, templates, generics, RAII and a large standard library.[^wiki] It can match C's
performance while offering far higher-level abstractions.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="C++ adds object-oriented and generic abstractions and RAII-managed resources on top of C's native compilation model." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="48" width="80" height="44" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="60" y="66">C core</text><text x="60" y="80" font-size="8">native, fast</text>
    <line x1="100" y1="70" x2="140" y2="70" stroke="currentColor" stroke-width="1.1"/><text x="120" y="62" font-size="8">adds</text>
    <rect x="140" y="22" width="92" height="22" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="186" y="37">classes / OOP</text>
    <rect x="140" y="58" width="92" height="22" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="186" y="73">templates</text>
    <rect x="140" y="94" width="92" height="22" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="186" y="109">RAII</text>
    <line x1="232" y1="33" x2="272" y2="69" stroke="currentColor" stroke-width="1.1"/><line x1="232" y1="69" x2="272" y2="69" stroke="currentColor" stroke-width="1.1"/><line x1="232" y1="105" x2="272" y2="69" stroke="currentColor" stroke-width="1.1"/>
    <rect x="272" y="52" width="92" height="36" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="318" y="74">compiled C++</text>
  </g>
</svg>
<figcaption>C++ layers objects, generics and RAII over C's fast native model — power at the price of complexity.</figcaption>
</figure>

## Overview

C++ [compiles](/reference/compiler/) ahead of time to native code like
[C](/reference/c-language/), but adds [object-oriented](/reference/object-oriented-programming/)
features, compile-time generics through templates, and the Standard Template Library.[^ref]
Its signature idiom is RAII — tying resource lifetimes to object scope — which gives a
disciplined approach to [memory management](/reference/memory-management/) and, with
smart pointers, much of the safety of automatic management without a garbage collector.

## Strengths and trade-offs

The strength is reach: C++ delivers C-level performance while letting you build large,
abstracted systems, which is why it dominates game engines, browsers, high-frequency
trading and serious DSP. The drawback is **complexity** — the language is vast, with
decades of accumulated features and sharp edges, and it inherits C's memory-safety risks
unless you are disciplined. It is genuinely hard to master. For new systems work where
safety is paramount, [Rust](/reference/rust-language/) is increasingly chosen as an
alternative that keeps the speed without the same footguns.

## Where it's used

C++ is the workhorse of performance-critical software with rich logic: game engines,
web browsers, trading systems, scientific and signal-processing code, and other domains
where both raw speed and high-level structure are required at once.

## Sources

[^ref]: [C++ reference](https://en.cppreference.com/w/cpp) — cppreference, the standard-tracking reference for the C++ language and library.
[^wiki]: [C++](https://en.wikipedia.org/wiki/C%2B%2B) — Wikipedia, for history and design background.
