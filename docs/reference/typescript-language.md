---
slug: typescript-language
title: TypeScript
entry_type: language
category: programming-languages
description: TypeScript is JavaScript with a static type system layered on top; it compiles to plain JavaScript and has become the default for large front-end and Node.js codebases.
keywords: TypeScript, TS, static typing, type system, JavaScript, transpile, compile, Microsoft, type safety
aka: [TypeScript, TS]
autolink: true
infobox:
  - { label: Paradigm, value: "Multi-paradigm: object-oriented, functional, event-driven" }
  - { label: Typing, value: "Static, structural, gradual (over dynamic JS)" }
  - { label: Appeared, value: "2012 (Microsoft)" }
  - { label: Designed by, value: "Anders Hejlsberg, Microsoft" }
  - { label: Compilation, value: "Compiled (transpiled) to JavaScript; types erased" }
  - { label: Memory, value: "Garbage-collected (via the JS runtime)" }
  - { label: Notable uses, value: "Front-end apps, Node.js backends, large codebases" }
see_also: [javascript-language, type-system, static-vs-dynamic-typing, compiler, csharp-language, jit-compilation, error-handling]
related_lessons:
  - { title: "A tour of today's major languages", url: /learn/intro-software-dev/language-tour/ }
  - { title: "Type systems", url: /learn/intro-software-dev/type-systems/ }
external:
  - { title: "TypeScript — Wikipedia", url: https://en.wikipedia.org/wiki/TypeScript }
  - { title: "The TypeScript programming language", url: https://www.typescriptlang.org/ }
---

**TypeScript** is a statically typed superset of [JavaScript](/reference/javascript-language/):
it adds an optional [type system](/reference/type-system/) on top of JavaScript and
compiles down to plain JavaScript that runs anywhere JavaScript does.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="TypeScript source is type-checked and compiled to plain JavaScript, which runs in the browser or Node.js." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="52" width="78" height="38" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="59" y="68">.ts source</text><text x="59" y="80" font-size="8">+ types</text>
    <line x1="98" y1="71" x2="140" y2="71" stroke="currentColor" stroke-width="1.1"/><text x="119" y="63" font-size="8">tsc</text>
    <rect x="140" y="52" width="92" height="38" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="186" y="68">type check</text><text x="186" y="80" font-size="8">+ erase types</text>
    <line x1="232" y1="71" x2="274" y2="71" stroke="currentColor" stroke-width="1.1"/>
    <rect x="274" y="52" width="78" height="38" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="313" y="74">.js output</text>
    <line x1="352" y1="71" x2="394" y2="71" stroke="currentColor" stroke-width="1.1"/>
    <rect x="394" y="52" width="50" height="38" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="419" y="74">runs</text>
  </g>
</svg>
<figcaption>TypeScript type-checks at build time, then erases the types and emits plain JavaScript.</figcaption>
</figure>

## Overview

TypeScript exists to fix JavaScript's biggest weakness — its loose
[dynamic typing](/reference/static-vs-dynamic-typing/). You annotate code with types,
the `tsc` [compiler](/reference/compiler/) checks them at build time, and then the types
are *erased*: the emitted output is ordinary JavaScript with no runtime type system of
its own. Memory and execution are still handled by the JavaScript runtime, so TypeScript
adds safety without changing how the program runs.

## Strengths and trade-offs

The static type layer catches a large class of bugs before the code runs, improves
editor tooling and autocompletion, and makes large codebases far easier to refactor and
maintain. The costs are a build/compile step that plain JavaScript does not need, time
spent writing and maintaining type annotations, and the fact that types are only checked
at compile time — they cannot guarantee anything about untyped data arriving at runtime,
so [error handling](/reference/error-handling/) at boundaries still matters. The
gradual, structural design (its lead designer also created [C#](/reference/csharp-language/))
lets teams adopt it incrementally.

## Where it's used

TypeScript has become the default for serious front-end frameworks and Node.js backends,
and for any JavaScript codebase large enough that loose typing becomes a liability.
Because it compiles to plain [JavaScript](/reference/javascript-language/), it runs
everywhere JavaScript runs and interoperates freely with existing JS libraries.
