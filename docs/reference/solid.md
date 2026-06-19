---
slug: solid
title: SOLID
entry_type: concept
category: principles-quality
description: SOLID is a set of five object-oriented design principles — single responsibility, open/closed, Liskov substitution, interface segregation, and dependency inversion — aimed at code that is easy to change without breaking.
keywords: SOLID principles, single responsibility, open closed, Liskov substitution, interface segregation, dependency inversion, object-oriented design, software design
aka: [SOLID]
autolink: true
infobox:
  - { label: Category, value: "Object-oriented design principles" }
  - { label: Popularized by, value: "Robert C. Martin" }
  - { label: Letters, value: "SRP, OCP, LSP, ISP, DIP" }
  - { label: Goal, value: "Code that's easy to change and test" }
  - { label: Nature, value: "Heuristics, not laws" }
see_also: [dry-kiss-yagni, clean-code, refactoring, abstraction, coupling-and-cohesion, object-oriented-programming, design-patterns]
related_lessons:
  - { title: "SOLID & object-oriented design", url: /learn/intro-software-dev/solid/ }
external:
  - { title: "SOLID — Wikipedia", url: https://en.wikipedia.org/wiki/SOLID }
---

**SOLID** is a set of five [object-oriented](/reference/object-oriented-programming/)
design principles, popularized by Robert C. Martin, that all aim at one outcome: code
that is easy to change without breaking.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="The five SOLID letters: single responsibility, open/closed, Liskov substitution, interface segregation, dependency inversion." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="14" y="40" width="80" height="40" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="54" y="56" font-size="14">S</text><text x="54" y="72">single resp.</text>
    <rect x="102" y="40" width="80" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="142" y="56" font-size="14">O</text><text x="142" y="72">open/closed</text>
    <rect x="190" y="40" width="80" height="40" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="230" y="56" font-size="14">L</text><text x="230" y="72">Liskov sub.</text>
    <rect x="278" y="40" width="80" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="318" y="56" font-size="14">I</text><text x="318" y="72">interface seg.</text>
    <rect x="366" y="40" width="80" height="40" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="406" y="56" font-size="14">D</text><text x="406" y="72">dependency inv.</text>
  </g>
</svg>
<figcaption>The five SOLID principles, each a heuristic for one-job, loosely coupled design.</figcaption>
</figure>

## The five principles

- **S — Single Responsibility Principle.** A unit of code should have one reason to
  change — it answers to a single concern. Mixing parsing, business rules, and
  storage in one class means a change to one risks breaking the others.
- **O — Open/Closed Principle.** Software should be open for extension but closed for
  modification: add new behavior by writing a new implementation, not by editing
  existing, tested code. An abstraction with multiple implementations is the usual
  mechanism.
- **L — Liskov Substitution Principle.** A subtype must be usable anywhere its base
  type is expected, honoring the parent's contract — its expectations, guarantees, and
  invariants — not just its method signatures.
- **I — Interface Segregation Principle.** Clients shouldn't be forced to depend on
  methods they don't use. Many small, role-focused interfaces beat one fat one.
- **D — Dependency Inversion Principle.** High-level logic should depend on
  [abstractions](/reference/abstraction/), not concrete details, with implementations
  supplied from outside (dependency injection).

## Why it matters

Together the principles reduce [coupling](/reference/coupling-and-cohesion/) and make
code flexible and [testable](/reference/unit-testing/): when logic depends on an
interface rather than a concrete class, a test can pass in a fake. SOLID overlaps
heavily with [DRY, KISS and YAGNI](/reference/dry-kiss-yagni/) and underpins many
[design patterns](/reference/design-patterns/) — but it is a toolbox of heuristics,
not a law. Applied dogmatically it causes as much over-engineering as it prevents;
reach for these principles to diagnose code that resists change or testing, not to
satisfy a checklist. They are a frequent target of [refactoring](/reference/refactoring/).
