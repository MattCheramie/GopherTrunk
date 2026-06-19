---
slug: jit-compilation
title: JIT vs AOT compilation
entry_type: concept
category: language-internals
description: JIT (just-in-time) compilation translates code to native machine code while a program runs, while AOT (ahead-of-time) compilation does all the translation before the program ships.
keywords: JIT compilation, AOT compilation, just in time, ahead of time, hot path, HotSpot, V8, warm up, native code
aka: ["JIT", "AOT"]
autolink: true
infobox:
  - { label: Category, value: Compilation strategy }
  - { label: AOT, value: "Translate to native code before shipping" }
  - { label: JIT, value: "Translate hot paths to native code while running" }
  - { label: JIT trade-off, value: "Near-native speed vs slow warm-up & memory" }
  - { label: AOT examples, value: "C, Rust, Go" }
  - { label: JIT examples, value: "JVM, V8, .NET" }
see_also: [compiler, interpreter, bytecode, static-binary, java-language, javascript-language, go-language]
related_lessons:
  - { title: "Compiled vs interpreted languages", url: /learn/intro-software-dev/compiled-vs-interpreted/ }
  - { title: "Performance vs productivity", url: /learn/intro-software-dev/performance-vs-productivity/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Just-in-time_compilation
---

**Just-in-time (JIT) compilation** translates code to native machine code *while the
program runs*, optimizing the parts it observes executing most; **ahead-of-time (AOT)
compilation** does all of that translation up front, before the program ever ships.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A JIT runs bytecode in a VM, detects a frequently-run hot path, and compiles it to native code on the fly." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="50" width="70" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="55" y="74">bytecode</text>
    <line x1="90" y1="70" x2="140" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="140" y="50" width="70" height="40" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="175" y="74">VM runs</text>
    <line x1="210" y1="70" x2="260" y2="40" stroke="currentColor" stroke-width="1.1"/><text x="240" y="30" font-size="8">hot path</text>
    <rect x="260" y="20" width="90" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="305" y="39">JIT compile</text>
    <line x1="350" y1="35" x2="400" y2="60" stroke="currentColor" stroke-width="1.1"/>
    <rect x="360" y="58" width="80" height="30" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="400" y="77">native code</text>
  </g>
</svg>
<figcaption>A JIT runs bytecode, spots a frequently-executed "hot" path, and compiles it to native code on the fly.</figcaption>
</figure>

## How it works

An [AOT](/reference/compiler/) compiler produces a finished native binary — the
approach [C](/reference/c-language/), [Rust](/reference/rust-language/), and
[Go](/reference/go-language/) take. A JIT instead ships portable
[bytecode](/reference/bytecode/) that a virtual machine begins
[interpreting](/reference/interpreter/); as it runs, the JIT spots frequently
executed ("hot") sections, compiles them to native machine code, and can optimize
based on what it actually observes — types, branch directions, inlining
opportunities. The result is often near-native steady-state speed.[^wiki]

## Trade-offs

AOT gives **instant startup** and predictable performance, and can produce a
self-contained [static binary](/reference/static-binary/). A JIT trades that for
**slower startup** (the engine must warm up before hot paths are optimized) and
**higher memory use** (it holds both bytecode and generated native code). A
long-running server happily pays the warm-up to gain peak throughput; a short-lived
command-line tool may exit before warm-up pays off, so AOT often suits it better.[^wiki]

## In practice

The [JVM](/reference/java-language/) JIT-compiles bytecode via HotSpot, V8 powers
[JavaScript](/reference/javascript-language/) in Chrome and Node.js, and .NET
JIT-compiles its intermediate language. The lines blur: Java is AOT-compiled to
bytecode *and* JIT-compiled to native code, and tools like GraalVM can AOT-compile
traditionally JIT'd languages for instant startup.

## Sources

[^wiki]: [Just-in-time compilation](https://en.wikipedia.org/wiki/Just-in-time_compilation) — Wikipedia, on JIT translating hot paths to native code at run time and how it contrasts with ahead-of-time compilation.
