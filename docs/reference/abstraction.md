---
slug: abstraction
title: Abstraction
entry_type: concept
category: paradigms-design
description: Abstraction is exposing a simple interface that describes what something does while hiding how it does it, enforced through encapsulation and information hiding.
keywords: abstraction, encapsulation, information hiding, interface, contract, leaky abstraction, what vs how, modularity, software design
aka: []
autolink: true
infobox:
  - { label: Type, value: "Software design principle" }
  - { label: Key idea, value: "Expose a clean what, hide the messy how" }
  - { label: Enforced by, value: "Encapsulation / information hiding" }
  - { label: Payoff, value: "Less to hold in your head; freedom to change internals" }
  - { label: Failure mode, value: "Leaky abstraction" }
  - { label: Related, value: "Coupling, cohesion, OOP" }
see_also: [coupling-and-cohesion, object-oriented-programming, design-patterns, solid, declarative-programming, type-system]
related_lessons:
  - { title: "Abstraction, coupling & cohesion", url: /learn/intro-software-dev/abstraction-coupling/ }
external:
  - { title: "Abstraction (computer science) — Wikipedia", url: https://en.wikipedia.org/wiki/Abstraction_(computer_science) }
---

**Abstraction** means exposing a simple interface that describes *what* something does
while hiding *how* it does it. When you call `sort()`, you depend on the promise ("returns
sorted output"), not on whether it uses quicksort or mergesort.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A caller depends only on a clean interface contract, while the messy implementation details behind it stay hidden and free to change." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="52" width="80" height="36" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="60" y="74">caller</text>
    <line x1="100" y1="70" x2="160" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="160" y="30" width="40" height="80" rx="4" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="180" y="22" font-size="8">interface</text><text x="180" y="74">what</text>
    <rect x="240" y="20" width="180" height="100" rx="6" fill="none" stroke="currentColor" stroke-width="1.1" stroke-dasharray="4 3"/><text x="330" y="14" font-size="8">hidden how</text>
    <circle cx="285" cy="50" r="10" fill="none" stroke="currentColor" stroke-width="1"/><circle cx="330" cy="80" r="10" fill="none" stroke="currentColor" stroke-width="1"/><circle cx="380" cy="48" r="10" fill="none" stroke="currentColor" stroke-width="1"/>
    <line x1="285" y1="50" x2="330" y2="80" stroke="currentColor" stroke-width="0.9"/><line x1="330" y1="80" x2="380" y2="48" stroke="currentColor" stroke-width="0.9"/>
    <text x="330" y="115" font-size="8" fill-opacity="0.7">free to change</text>
  </g>
</svg>
<figcaption>The caller depends on a clean contract; everything behind it stays hidden and free to change.</figcaption>
</figure>

## What it buys you

Good abstractions shrink the amount you have to hold in your head: instead of reasoning
about a thousand lines of filter math, you reason about a single operation with a clear
name and contract. They also keep [coupling](/reference/coupling-and-cohesion/) low —
callers depend on the *what*, so the *how* is free to change without breaking them. This is
the foundation of [object-oriented](/reference/object-oriented-programming/) interfaces,
[declarative](/reference/declarative-programming/) styles, and most
[design patterns](/reference/design-patterns/), and it underpins the dependency-inversion
idea in [SOLID](/reference/solid/).

## Information hiding and encapsulation

Abstraction only holds if the internals stay genuinely hidden. **Information hiding** is
the discipline of keeping a module's internal state and helpers private, exposing only the
interface; **encapsulation** is the language mechanism that enforces it — private fields,
unexported names, access modifiers. Anything you expose, someone will depend on, so the
rule of thumb is to **expose as little as possible**. A small public surface is a small set
of promises you must keep, closely tied to the language's
[type system](/reference/type-system/) and visibility rules.

## Leaky abstractions

No abstraction is perfect. A **leaky abstraction** is one whose hidden details surface
anyway, forcing the caller to know what was supposed to be hidden — a "local file"
interface that mysteriously stalls on a network path, or an ORM you can ignore until a
query is slow. Joel Spolsky's "Law of Leaky Abstractions" holds that all non-trivial
abstractions leak to some degree. You cannot eliminate leaks, but you can manage them:
document assumptions, surface failures explicitly rather than misbehaving silently, and
keep the abstraction honest about what it promises.
