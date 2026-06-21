---
slug: imperative-programming
title: Imperative programming
entry_type: concept
category: paradigms-design
description: Imperative programming is the paradigm of telling the computer how to do something as an ordered sequence of statements that change state, closely mirroring how the hardware works.
keywords: imperative programming, procedural programming, statements, mutable state, control flow, loops, assignment, paradigm, procedures
aka: []
autolink: true
infobox:
  - { label: Type, value: "Programming paradigm" }
  - { label: Key idea, value: "Step-by-step statements that change state" }
  - { label: Defining trait, value: "Mutable state, ordered execution" }
  - { label: Sub-style, value: "Procedural (organised into procedures)" }
  - { label: Strong examples, value: "C, Go, Fortran, COBOL" }
  - { label: Contrast with, value: "Declarative programming" }
see_also: [declarative-programming, functional-programming, object-oriented-programming, memory-management, c-language, go-language, fortran-language]
related_lessons:
  - { title: "Paradigms & language families", url: /learn/intro-software-dev/language-families/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Imperative_programming
---

**Imperative programming** is the oldest and most direct paradigm: you tell the computer
*how* to do something as an ordered sequence of statements that change state — assign a
value, loop, increment a counter, branch on a condition.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A counter variable is initialised then mutated step by step through ordered statements until a loop completes." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="30" y="55" width="90" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="75" y="74">total = 0</text>
    <line x1="120" y1="70" x2="150" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="150" y="55" width="110" height="30" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="205" y="74">total += i</text>
    <line x1="260" y1="70" x2="290" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="290" y="55" width="90" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="335" y="74">i &lt; n ?</text>
    <path d="M335 85 L335 110 L205 110 L205 85" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="270" y="124" font-size="8">loop back (mutate again)</text>
    <line x1="380" y1="70" x2="420" y2="70" stroke="currentColor" stroke-width="1.1"/><text x="420" y="62" font-size="8">done</text>
  </g>
</svg>
<figcaption>State is initialised then mutated step by step; the order of statements is what determines the result.</figcaption>
</figure>

## Core ideas

The defining trait of imperative code is **mutable state**: variables change over time,
and the order of statements matters. This mirrors how the hardware actually works — the
CPU executes one instruction after another, updating registers and memory — which is why
it feels natural and why early languages were imperative. That same power is the risk:
many subtle bugs come from state changing when or where you did not expect.

## Procedural programming

**Procedural** programming is imperative code organised into reusable *procedures*
(functions). Instead of one long script, you factor work into named routines that call
each other.[^wiki] [C](/reference/c-language/) is the classic procedural language, and
[Fortran](/reference/fortran-language/) and [COBOL](/reference/cobol-language/) are early
examples. Procedural style pairs naturally with manual
[memory management](/reference/memory-management/) and tight performance loops.

## When it fits

Reach for imperative or procedural style when you are close to the hardware or in a tight
performance loop, where you want exact control over each operation.
[Go](/reference/go-language/) is mostly procedural with lightweight interfaces. It
contrasts with [declarative programming](/reference/declarative-programming/), which
describes *what* you want rather than *how*, and with
[functional programming](/reference/functional-programming/), which avoids mutable state
altogether. Most languages are multi-paradigm, so an imperative inner loop often sits
inside [object-oriented](/reference/object-oriented-programming/) or functional code.[^wiki]

## Sources

[^wiki]: [Imperative programming](https://en.wikipedia.org/wiki/Imperative_programming) — Wikipedia, for the definition, the role of mutable state and ordered execution, and the procedural sub-style.
