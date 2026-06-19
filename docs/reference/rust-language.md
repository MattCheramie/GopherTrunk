---
slug: rust-language
title: Rust
entry_type: language
category: programming-languages
description: Rust is a compiled systems language whose borrow checker enforces memory safety at compile time with no garbage collector, delivering C-level performance at the cost of a steep learning curve.
keywords: Rust, borrow checker, ownership, memory safety, systems programming, no garbage collector, WebAssembly, data races, cargo
aka: [Rust]
autolink: true
infobox:
  - { label: Paradigm, value: "Multi-paradigm: imperative, functional, concurrent" }
  - { label: Typing, value: "Static, strong, inferred" }
  - { label: Appeared, value: "2010 (Mozilla; 1.0 in 2015)" }
  - { label: Designed by, value: "Graydon Hoare; Mozilla" }
  - { label: Compilation, value: "Ahead-of-time to native machine code (LLVM)" }
  - { label: Memory, value: "Ownership + borrow checking (no GC)" }
  - { label: Notable uses, value: "Systems, CLIs, WebAssembly, safety-critical components" }
see_also: [c-language, cpp-language, go-language, memory-management, compiler, concurrency, type-system]
related_lessons:
  - { title: "A tour of today's major languages", url: /learn/intro-software-dev/language-tour/ }
  - { title: "Memory management across languages", url: /learn/intro-software-dev/memory-management/ }
external:
  - { title: "Rust (programming language) — Wikipedia", url: https://en.wikipedia.org/wiki/Rust_(programming_language) }
  - { title: "The Rust programming language", url: https://www.rust-lang.org/ }
---

**Rust** is a statically typed, compiled systems language designed to be fast *and*
safe: its borrow checker enforces memory safety at compile time, eliminating whole
classes of bugs without a garbage collector or runtime cost.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="The Rust compiler's borrow checker verifies ownership and borrowing rules at compile time; code that passes is memory-safe with no garbage collector." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="50" width="64" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="52" y="74">.rs source</text>
    <line x1="84" y1="70" x2="124" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="124" y="46" width="96" height="48" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="172" y="66">borrow</text><text x="172" y="78">checker</text>
    <line x1="220" y1="58" x2="262" y2="40" stroke="currentColor" stroke-width="1.1"/><text x="290" y="38" font-size="8">pass</text>
    <line x1="220" y1="82" x2="262" y2="104" stroke="currentColor" stroke-width="1.1" stroke-dasharray="3 3"/><text x="296" y="110" font-size="8" fill-opacity="0.7">reject unsafe</text>
    <rect x="320" y="28" width="120" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="380" y="44">safe native binary</text><text x="380" y="55" font-size="8">no GC</text>
  </g>
</svg>
<figcaption>Rust's borrow checker proves memory safety at compile time, then emits a native binary with no garbage collector.</figcaption>
</figure>

## Overview

Rust [compiles](/reference/compiler/) ahead of time to native machine code, with
performance that rivals [C](/reference/c-language/) and [C++](/reference/cpp-language/).
Its defining idea is **ownership**: each value has a single owner, and a compile-time
borrow checker enforces strict rules about how references may be shared and mutated.
Code that violates those rules simply does not compile, which rules out use-after-free,
double-free and data races — providing [memory](/reference/memory-management/) safety and
safe [concurrency](/reference/concurrency/) without a [garbage collector](/reference/garbage-collection/).

## Strengths and trade-offs

The payoff is memory safety *and* speed at the same time, with no runtime collector and
no GC pauses. The price is a **steep learning curve** — the borrow checker is famously
hard to satisfy at first and forces you to think carefully about lifetimes and ownership.
The ecosystem, while growing fast, is younger than C's or C++'s, and compile times can be
long. The trade is deliberate: more friction up front in exchange for fewer whole classes
of bugs later.

## Where it's used

Rust is increasingly chosen for new systems software, command-line tools, WebAssembly
targets, and safety-critical components — places that want C-level control without C's
memory hazards. It is not so much replacing [C](/reference/c-language/) and
[C++](/reference/cpp-language/) as competing with them for new work, with the three
expected to coexist for a long time.
