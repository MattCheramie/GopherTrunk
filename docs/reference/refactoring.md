---
slug: refactoring
title: Refactoring
entry_type: concept
category: principles-quality
description: Refactoring is the disciplined practice of restructuring existing code to improve its design, readability, and maintainability without changing its external behavior.
keywords: refactoring, code refactoring, restructuring code, technical debt, code smell, behavior-preserving, extract function, rename, design improvement
aka: []
autolink: true
infobox:
  - { label: Category, value: "Code quality practice" }
  - { label: Definition, value: "Change structure, not behavior" }
  - { label: Safety net, value: "Tests (especially unit tests)" }
  - { label: Triggers, value: "Code smells, technical debt" }
  - { label: Popularized by, value: "Martin Fowler" }
see_also: [clean-code, dry-kiss-yagni, solid, unit-testing, test-driven-development, abstraction, coupling-and-cohesion]
related_lessons:
  - { title: "Writing readable code", url: /learn/intro-software-dev/clean-code/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Code_refactoring
---

**Refactoring** is the disciplined practice of restructuring existing code to improve
its design, readability, and maintainability *without* changing its external behavior.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="Tangled code is restructured into clean code while the observable behavior, the output, stays the same." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="35" width="90" height="60" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="65" y="58">tangled</text><text x="65" y="72">code</text>
    <path d="M30 86 L45 80 L60 90 L75 82 L100 88" fill="none" stroke="currentColor" stroke-width="1"/>
    <line x1="110" y1="65" x2="160" y2="65" stroke="currentColor" stroke-width="1.2"/><text x="135" y="56" font-size="8">refactor</text>
    <rect x="160" y="35" width="90" height="60" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="205" y="58">clean</text><text x="205" y="72">code</text>
    <line x1="250" y1="65" x2="300" y2="65" stroke="currentColor" stroke-width="1.2"/>
    <rect x="300" y="48" width="140" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3" stroke-dasharray="4 3"/><text x="370" y="69">same behavior</text>
  </g>
</svg>
<figcaption>Refactoring changes a program's internal structure while its observable behavior stays fixed.</figcaption>
</figure>

## What it is

The defining constraint is behavior preservation: a refactor reorganizes the *how*
while keeping the *what* identical. Typical moves — popularized by Martin Fowler's
catalog — include renaming for clarity, extracting a function or class, inlining a
needless indirection, removing duplication, and breaking up a class that has taken on
too many responsibilities. Each is a small, safe step rather than a sweeping rewrite.[^wiki]
Refactoring is triggered by *code smells*: duplication, long functions, large classes,
tangled dependencies, and other signals that the code resists change. Left
unaddressed, those smells accumulate as **technical debt** that slows every future
change.

## Why tests matter

Because a refactor must not alter behavior, you need a way to prove it didn't — and that
is the job of [tests](/reference/unit-testing/). A solid suite of fast
[unit tests](/reference/unit-testing/) (run automatically by [CI/CD](/reference/ci-cd/))
lets you restructure boldly and catch any accidental behavior change in seconds. This is
also why refactoring is the third beat of the
[test-driven development](/reference/test-driven-development/) cycle: red, green, then
*refactor* on a green suite.

## How it connects

Refactoring is how the quality principles get applied to code that already exists. You
refactor toward [clean code](/reference/clean-code/), toward
[DRY, KISS and YAGNI](/reference/dry-kiss-yagni/), and toward
[SOLID](/reference/solid/) — improving names, reducing
[coupling](/reference/coupling-and-cohesion/), and introducing the right
[abstraction](/reference/abstraction/) only once it has proven itself. Done
continuously in small steps, it keeps a codebase healthy; deferred indefinitely, it
becomes a costly rewrite.

## Sources

[^wiki]: [Code refactoring](https://en.wikipedia.org/wiki/Code_refactoring) — Wikipedia, for the behavior-preserving definition and Martin Fowler's catalog of refactorings.
