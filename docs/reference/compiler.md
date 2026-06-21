---
slug: compiler
title: Compiler
entry_type: concept
category: language-internals
description: A compiler is a program that translates source code into another form — usually machine code or bytecode — ahead of time, before the program runs.
keywords: compiler, compilation, ahead of time, AOT, machine code, optimization, native binary, source code, build step
aka: [compiler, AOT compiler]
autolink: true
infobox:
  - { label: Category, value: Language tooling }
  - { label: Input, value: "Source code" }
  - { label: Output, value: "Machine code or bytecode" }
  - { label: When it runs, value: "Ahead of time, before execution" }
  - { label: Trade-off, value: "Build step & per-platform output vs fast, optimized code" }
  - { label: Compiled languages, value: "C, C++, Rust, Go" }
see_also: [interpreter, bytecode, jit-compilation, static-binary, cross-compilation, type-system, c-language, rust-language, go-language]
related_lessons:
  - { title: "Compiled vs interpreted languages", url: /learn/intro-software-dev/compiled-vs-interpreted/ }
  - { title: "Performance vs productivity", url: /learn/intro-software-dev/performance-vs-productivity/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Compiler
---

**A compiler** is a program that translates source code written for humans into a
form a machine can run — typically native machine code — doing the work *ahead of
time*, before the program is ever executed.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="Source code passes through a compiler once, producing machine code the CPU runs directly each time." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="45" width="80" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="60" y="69">source code</text>
    <line x1="100" y1="65" x2="160" y2="65" stroke="currentColor" stroke-width="1.1"/><text x="130" y="57" font-size="8">translate</text>
    <rect x="160" y="45" width="80" height="40" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="200" y="69">compiler</text>
    <line x1="240" y1="65" x2="300" y2="65" stroke="currentColor" stroke-width="1.1"/>
    <rect x="300" y="45" width="80" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="340" y="63">machine</text><text x="340" y="75">code</text>
    <line x1="380" y1="65" x2="430" y2="65" stroke="currentColor" stroke-width="1.1"/><text x="410" y="57" font-size="8">CPU</text>
  </g>
</svg>
<figcaption>A compiler does the translation once, up front; the resulting machine code runs directly on the CPU.</figcaption>
</figure>

## How it works

A compiler reads the entire source program, checks it (including its
[types](/reference/type-system/)), and emits an equivalent program in a lower-level
language. Classic *ahead-of-time* (AOT) compilers such as those for
[C](/reference/c-language/), [C++](/reference/cpp-language/),
[Rust](/reference/rust-language/), and [Go](/reference/go-language/) produce native
machine code packaged as an executable. Other compilers target
[bytecode](/reference/bytecode/) for a virtual machine instead. Along the way the
compiler optimizes — inlining calls, vectorizing loops, eliminating dead code — so
the output often runs far faster than a naive translation would.[^wiki]

## Trade-offs

Because the translation happens once, the CPU runs native instructions with no
per-operation overhead, and performance is predictable from run to run. The price is
a **build step** between editing and running, and output that is usually tied to one
platform — a binary built for x86-64 Linux will not run on an ARM Mac without
recompiling (see [cross-compilation](/reference/cross-compilation/)).[^wiki] This contrasts
with an [interpreter](/reference/interpreter/), which skips the build step but pays
overhead on every run, and with [JIT compilation](/reference/jit-compilation/), which
moves the translation to run time.

## In practice

Compiled languages dominate systems software, where speed and a self-contained
[static binary](/reference/static-binary/) matter. GopherTrunk is compiled by the Go
toolchain into a single native executable, which is why it ships without requiring any
runtime on the target machine.

## Sources

[^wiki]: [Compiler](https://en.wikipedia.org/wiki/Compiler) — Wikipedia, on how compilers translate source ahead of time, optimize, and target machine code or bytecode.
