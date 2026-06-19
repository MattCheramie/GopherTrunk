---
slug: coupling-and-cohesion
title: Coupling & cohesion
entry_type: concept
category: paradigms-design
description: Coupling measures how much one module depends on another and cohesion measures how focused a single module is; good design aims for loose coupling and high cohesion.
keywords: coupling, cohesion, loose coupling, high cohesion, modularity, dependency, single responsibility, big ball of mud, software design
aka: [coupling and cohesion "loose coupling, high cohesion"]
autolink: true
infobox:
  - { label: Type, value: "Software design principle" }
  - { label: Coupling, value: "How much modules depend on each other (between)" }
  - { label: Cohesion, value: "How focused a single module is (within)" }
  - { label: The goal, value: "Loose coupling + high cohesion" }
  - { label: Failure mode, value: "Big ball of mud" }
  - { label: Related, value: "Abstraction, SOLID, design patterns" }
see_also: [abstraction, object-oriented-programming, design-patterns, solid, structural-patterns, behavioral-patterns]
related_lessons:
  - { title: "Abstraction, coupling & cohesion", url: /learn/intro-software-dev/abstraction-coupling/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Coupling_(computer_programming)
---

**Coupling** measures how much one module depends on another — how entangled they are —
while **cohesion** measures how focused a single module is — how well its parts belong
together. The central slogan of modular design is: aim for **loose coupling and high
cohesion**.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="Loose coupling shows modules connected by few narrow lines through a clean interface, while tight coupling shows many tangled connections between internals." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="115" y="16" font-size="8" fill-opacity="0.7">loose coupling</text>
    <rect x="30" y="50" width="60" height="44" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="60" y="76">A</text>
    <rect x="150" y="50" width="60" height="44" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="180" y="76">B</text>
    <line x1="90" y1="72" x2="150" y2="72" stroke="currentColor" stroke-width="1.3"/>
    <text x="350" y="16" font-size="8" fill-opacity="0.7">tight coupling</text>
    <rect x="280" y="50" width="60" height="44" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="310" y="76">C</text>
    <rect x="380" y="50" width="60" height="44" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="410" y="76">D</text>
    <line x1="340" y1="58" x2="380" y2="58" stroke="currentColor" stroke-width="1"/><line x1="340" y1="68" x2="380" y2="78" stroke="currentColor" stroke-width="1"/><line x1="340" y1="78" x2="380" y2="62" stroke="currentColor" stroke-width="1"/><line x1="340" y1="86" x2="380" y2="86" stroke="currentColor" stroke-width="1"/>
  </g>
</svg>
<figcaption>Loose coupling links modules through one narrow interface; tight coupling tangles their internals together.</figcaption>
</figure>

## Coupling: dependence between modules

Tightly coupled modules know each other's internals, so a change in one forces a change in
the other. Loosely coupled modules interact only through narrow, stable interfaces, so each
can change behind its contract without disturbing its neighbours. A rough spectrum from
worse to better: **content coupling** (one module reaches into another's state),
**shared global state** (invisible dependencies through globals), and **data coupling**
(passing exactly the data needed through a clean interface — the goal).[^wiki] Loose coupling is
what makes code testable, reusable, and safe to change, and it leans directly on
[abstraction](/reference/abstraction/) and the dependency-inversion idea in
[SOLID](/reference/solid/).

## Cohesion: focus within a module

Where coupling is *between* modules, cohesion is *within* one: how well do its parts belong
together? A highly cohesive module does one well-defined job, and everything in it serves
that job. A low-cohesion module is a grab-bag — a `Utils` class that validates emails,
parses dates, and resizes images has no coherent reason to exist as a unit. High cohesion
is the same idea as the single-responsibility principle: each module focused on one reason
to change, making it easy to name, understand, and modify.

## Why they go together

| | Loose coupling | High cohesion |
|---|---|---|
| **Scope** | Between modules | Within a module |
| **Means** | Few, narrow dependencies | One focused responsibility |
| **Payoff** | Local changes, easy testing | Clear purpose, simple to reason about |

When both hold, modules behave like well-designed components: self-contained, with clear
connectors. When coupling is tight or cohesion is low, you get the dreaded "big ball of
mud" where nothing can move without everything moving. Most
[design patterns](/reference/design-patterns/) — especially the
[structural](/reference/structural-patterns/) and
[behavioral](/reference/behavioral-patterns/) families — exist precisely to push a design
toward this loose-coupling, high-cohesion ideal.[^wiki]

## Sources

[^wiki]: [Coupling (computer programming)](https://en.wikipedia.org/wiki/Coupling_(computer_programming)) — Wikipedia, for the definition of coupling, its spectrum, and the paired loose-coupling, high-cohesion goal.
