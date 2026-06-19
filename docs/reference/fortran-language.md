---
slug: fortran-language
title: FORTRAN
entry_type: language
category: programming-languages
description: FORTRAN (FORmula TRANslation), introduced in 1957, was the first widely used high-level language; built for scientific and numeric computing, its descendants still dominate high-performance math.
keywords: FORTRAN, Fortran, scientific computing, numerical computing, high-level language, IBM, John Backus, formula translation, HPC
aka: [Fortran]
autolink: true
infobox:
  - { label: Paradigm, value: "Imperative, procedural, array-oriented" }
  - { label: Typing, value: "Static, strong" }
  - { label: Appeared, value: "1957 (IBM)" }
  - { label: Designed by, value: "John Backus and team at IBM" }
  - { label: Compilation, value: "Compiled to native code" }
  - { label: Memory, value: "Manual / static (no garbage collector)" }
  - { label: Notable uses, value: "Scientific computing, HPC, numerical libraries" }
see_also: [cobol-language, lisp-language, c-language, julia-language, compiler, imperative-programming, type-system]
related_lessons:
  - { title: "A tour of today's major languages", url: /learn/intro-software-dev/language-tour/ }
  - { title: "The birth of programming languages", url: /learn/intro-software-dev/birth-of-languages/ }
external:
  - { title: "Fortran — Wikipedia", url: https://en.wikipedia.org/wiki/Fortran }
  - { title: "Fortran programming language", url: https://fortran-lang.org/ }
---

**FORTRAN** (FORmula TRANslation), introduced by IBM in 1957, was the first widely
used high-level language. Built so scientists could write formulas almost as they
appear on paper, its descendants still dominate high-performance numerical computing.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A mathematical formula is written as FORTRAN source, compiled to native code, and run as fast numerical computation." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="50" width="80" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="60" y="74">formula</text>
    <line x1="100" y1="70" x2="150" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="150" y="50" width="90" height="40" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="195" y="74">compiler</text>
    <line x1="240" y1="70" x2="290" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="290" y="50" width="90" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="335" y="68">fast</text><text x="335" y="80">native math</text>
  </g>
</svg>
<figcaption>FORTRAN let scientists write formulas directly; the compiler turned them into fast machine code.</figcaption>
</figure>

## History

Before FORTRAN, fast code meant hand-written assembly, and many doubted a
[compiler](/reference/compiler/) could match a human's tuned machine code. John
Backus's team at IBM proved otherwise, and the productivity gain was so large there
was no going back. FORTRAN focused on fast numerical computation — exactly the
number-crunching behind filtering and demodulating signals — and it established that
high-level languages were practical, opening the door to every language that followed.

## Legacy and use today

FORTRAN is [statically typed](/reference/type-system/), compiled to native code, and
[imperative](/reference/imperative-programming/) and array-oriented in style, with no
[garbage collector](/reference/garbage-collection/). Decades of mature, heavily
optimised numerical libraries are written in it, and it remains in active use across
weather modelling, physics, and high-performance computing where raw numeric
throughput matters. Its drawbacks are those of its age: the language and much existing
code feel dated next to modern tooling, and it is a numeric specialist rather than a
general-purpose language. Newer numeric languages like [Julia](/reference/julia-language/)
court its users, while [C](/reference/c-language/) became the more general
systems-and-DSP workhorse. It shares the early-language stage with
[COBOL](/reference/cobol-language/) and [LISP](/reference/lisp-language/).
