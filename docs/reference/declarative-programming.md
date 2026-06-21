---
slug: declarative-programming
title: Declarative programming
entry_type: concept
category: paradigms-design
description: Declarative programming describes the result you want and lets the system figure out the steps, covering functional, logic, query, and configuration styles.
keywords: declarative programming, declarative, SQL, logic programming, Prolog, what not how, functional programming, configuration, markup, paradigm
aka: []
autolink: true
infobox:
  - { label: Type, value: "Programming paradigm" }
  - { label: Key idea, value: "Describe what you want, not how" }
  - { label: Families, value: "Functional, logic, query, config/markup" }
  - { label: Examples, value: "SQL, Prolog, HTML/CSS, REST APIs" }
  - { label: Trade-off, value: "Brevity vs control over execution" }
  - { label: Contrast with, value: "Imperative programming" }
see_also: [imperative-programming, functional-programming, object-oriented-programming, rest, abstraction]
related_lessons:
  - { title: "Paradigms & language families", url: /learn/intro-software-dev/language-families/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Declarative_programming
---

**Declarative programming** flips the question from *how* to *what*: you describe the
result you want and let the system figure out the steps needed to produce it.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A declarative request describing a desired result is handed to an engine, which plans and executes the hidden steps to produce the result." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="48" width="120" height="44" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="80" y="66">what you want</text><text x="80" y="80" font-size="8">SELECT ... WHERE</text>
    <line x1="140" y1="70" x2="190" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="190" y="40" width="110" height="60" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="245" y="64">engine plans</text><text x="245" y="78" font-size="8">the how</text>
    <line x1="300" y1="70" x2="350" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="350" y="48" width="90" height="44" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="395" y="74">result</text>
  </g>
</svg>
<figcaption>You declare the desired result; the engine chooses and runs the steps to achieve it.</figcaption>
</figure>

## Families of declarative code

Declarative is a broad umbrella. [Functional programming](/reference/functional-programming/)
is one family — you describe transformations, not state changes.[^wiki] Beyond it:

- **Query languages** — SQL declares *what* rows you want; the database plans *how* to
  fetch them, never telling the engine which index to scan.
- **Logic programming** — Prolog states facts and rules, then poses queries, and the
  engine searches for solutions that satisfy them.
- **Markup and config** — HTML, CSS, and infrastructure tools declare a desired structure
  or end-state rather than a procedure.
- **APIs** — a [REST](/reference/rest/) request declares the resource you want, leaving
  the server to fulfil it.

## Trade-offs

Declarative code is often shorter and harder to get subtly wrong, because there are fewer
moving parts of explicit state to manage. The cost is giving up fine control over
execution: when a SQL query is slow or a constraint solver picks a poor path, you may need
to peek behind the [abstraction](/reference/abstraction/) to understand what it generated.

## Versus imperative

The clean contrast is with [imperative programming](/reference/imperative-programming/),
which spells out an ordered sequence of state changes. Reach for declarative style when
*what* you want is clearer than *how* to get it — querying data, describing
infrastructure, or expressing rules — and reach for imperative when you need precise
control over each step.[^wiki]

## Sources

[^wiki]: [Declarative programming](https://en.wikipedia.org/wiki/Declarative_programming) — Wikipedia, for the what-not-how definition and the functional, logic, query, and configuration families.
