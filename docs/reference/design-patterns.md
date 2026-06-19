---
slug: design-patterns
title: Design patterns
entry_type: concept
category: paradigms-design
description: A design pattern is a named, reusable solution to a recurring software design problem, catalogued most famously by the Gang of Four into creational, structural, and behavioral families.
keywords: design patterns, gang of four, GoF, creational structural behavioral, software design, anti-pattern, reusable solution, Christopher Alexander
aka: [design patterns "GoF patterns"]
autolink: true
infobox:
  - { label: Type, value: "Software design concept" }
  - { label: Key idea, value: "Named, reusable solution to a recurring problem" }
  - { label: Canonical source, value: "Gang of Four (1994), 23 patterns" }
  - { label: Three families, value: "Creational, structural, behavioral" }
  - { label: Not, value: "Copy-paste code; it's a template + vocabulary" }
  - { label: Watch for, value: "Golden hammer anti-pattern (overuse)" }
see_also: [creational-patterns, structural-patterns, behavioral-patterns, object-oriented-programming, abstraction, coupling-and-cohesion, solid]
related_lessons:
  - { title: "What is a design pattern?", url: /learn/intro-software-dev/what-are-patterns/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Software_design_pattern
  - https://en.wikipedia.org/wiki/Design_Patterns
---

**Design patterns** are named, reusable descriptions of solutions to problems that recur
in software design — not finished code you paste in, but templates for arranging classes,
objects, and responsibilities so a design stays flexible.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="The Gang of Four catalogue of 23 patterns splits into three families: creational, structural, and behavioral." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="160" y="14" width="140" height="30" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="230" y="33">GoF: 23 patterns</text>
    <line x1="230" y1="44" x2="80" y2="78" stroke="currentColor" stroke-width="1.1"/><line x1="230" y1="44" x2="230" y2="78" stroke="currentColor" stroke-width="1.1"/><line x1="230" y1="44" x2="380" y2="78" stroke="currentColor" stroke-width="1.1"/>
    <rect x="20" y="80" width="120" height="44" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="80" y="100">creational</text><text x="80" y="114" font-size="8">making</text>
    <rect x="170" y="80" width="120" height="44" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="230" y="100">structural</text><text x="230" y="114" font-size="8">arranging</text>
    <rect x="320" y="80" width="120" height="44" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="380" y="100">behavioral</text><text x="380" y="114" font-size="8">coordinating</text>
  </g>
</svg>
<figcaption>The Gang of Four sorted 23 patterns into three families by the kind of problem each solves.</figcaption>
</figure>

## A pattern is a template and a vocabulary

Each pattern has four rough parts: a **name**, a **problem** it applies to, a general
**solution** (an arrangement of classes and objects, not a specific implementation), and
the **consequences** you accept by using it. Because the solution is general, the same
pattern looks different in [Java](/reference/java-language/), [Go](/reference/go-language/),
and [Python](/reference/python-language/) — you reuse the *design*, not the bytes. The
most underrated benefit is communication: saying "put a facade over the DSP chain"
conveys a whole design decision in one word.

## The Gang of Four families

The idea came from architecture (Christopher Alexander), but software's canonical source
is the 1994 book *Design Patterns* by the **"Gang of Four" (GoF)** — Gamma, Helm, Johnson,
and Vlissides — which catalogued **23 patterns** for [object-oriented](/reference/object-oriented-programming/)
design.[^gof] They split into three families:

- [Creational patterns](/reference/creational-patterns/) — how objects get *made* (Factory, Builder, Singleton).
- [Structural patterns](/reference/structural-patterns/) — how objects are *arranged* (Adapter, Facade, Decorator).
- [Behavioral patterns](/reference/behavioral-patterns/) — how objects *coordinate* (Observer, Strategy, State).

## Why and when to use them

Most patterns exist to manage [abstraction](/reference/abstraction/) and keep
[coupling](/reference/coupling-and-cohesion/) low, and many are concrete ways to honour
[SOLID](/reference/solid/), especially the open/closed principle. But every pattern adds
indirection. Forcing one where it is not needed is the "golden hammer" anti-pattern; the
skill is judging when a real problem calls for one rather than memorising all 23.[^wiki]

## Sources

[^wiki]: [Software design pattern](https://en.wikipedia.org/wiki/Software_design_pattern) — Wikipedia, for the definition, the template/vocabulary framing, and the three families.
[^gof]: [Design Patterns](https://en.wikipedia.org/wiki/Design_Patterns) — Wikipedia, on the 1994 Gang of Four book that catalogued the 23 object-oriented patterns.
