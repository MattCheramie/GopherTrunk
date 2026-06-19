---
slug: clean-code
title: Clean code
entry_type: concept
category: principles-quality
description: Clean code is the discipline of writing programs optimized for the reader — clear names, small focused functions, why-not-what comments, and consistent style — because code is read far more often than it is written.
keywords: clean code, readable code, code readability, intention-revealing names, small functions, code comments, code style, maintainable code, naming
aka: []
autolink: true
infobox:
  - { label: Category, value: "Code quality / readability" }
  - { label: Core idea, value: "Code is read more than written" }
  - { label: Pillars, value: "Names, small functions, why-comments, style" }
  - { label: Tools, value: "Formatters & linters (gofmt, Prettier, Black)" }
  - { label: Enemy, value: "Cleverness" }
see_also: [refactoring, dry-kiss-yagni, solid, abstraction, coupling-and-cohesion, unit-testing]
related_lessons:
  - { title: "Writing readable code", url: /learn/intro-software-dev/clean-code/ }
external:
  - { title: "Robert C. Martin — Wikipedia", url: https://en.wikipedia.org/wiki/Robert_C._Martin }
---

**Clean code** is the discipline of writing programs optimized for the reader rather
than the writer, because code is read far more often than it is written — by reviewers,
future maintainers, and you, months later.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="Code written once is read many times by reviewer, debugger, and future self." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="45" width="80" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="60" y="60">written</text><text x="60" y="72" font-size="8">once</text>
    <line x1="100" y1="62" x2="150" y2="40" stroke="currentColor" stroke-width="1.1"/><line x1="100" y1="62" x2="150" y2="62" stroke="currentColor" stroke-width="1.1"/><line x1="100" y1="62" x2="150" y2="84" stroke="currentColor" stroke-width="1.1"/>
    <rect x="150" y="26" width="120" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="210" y="42">read: reviewer</text>
    <rect x="150" y="52" width="120" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="210" y="68">read: debugger</text>
    <rect x="150" y="78" width="120" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="210" y="94">read: future you</text>
    <text x="360" y="66" font-size="11">read &gt;&gt; written</text>
  </g>
</svg>
<figcaption>A line is written once and read many times — so optimize for the reader.</figcaption>
</figure>

## Core habits

The everyday habits of clean code all flow from one fact: the reader's time is what you
are optimizing.

- **Intention-revealing names.** A good name says *what* something is and *why* it
  exists without forcing a detour into the implementation. `sampleRateHz` cannot be
  misread; `sr` can. Encode units, avoid cryptic abbreviations and noise words, and
  pick one word per concept.
- **Small, focused functions.** A function should do one thing at one level of
  [abstraction](/reference/abstraction/). If you can't name it without "and," or it
  doesn't fit on a screen, it's probably hiding a smaller function. Extracting steps
  turns stale comments into self-documenting names.
- **Comments explain *why*, not *what*.** Names and structure already show the *what*;
  reserve comments for intent, trade-offs, and non-obvious constraints. A comment that
  merely narrates the next line will drift and eventually lie.
- **Consistent style.** Lean on formatters and linters (`gofmt`, Prettier, Black,
  `rustfmt`) so attention goes to logic, not layout, and diffs stay small.

## The cost of cleverness

Clever, compact code shifts cost from the writer, who understood it in the moment, to
every future reader who must reverse-engineer it. As Brian Kernighan warned, debugging
is harder than writing, so code written as cleverly as possible is code you can't debug.
The genuinely hard skill is making complex logic *look* simple. Clean code is closely
tied to [DRY, KISS and YAGNI](/reference/dry-kiss-yagni/) and [SOLID](/reference/solid/),
and improving it after the fact is the work of [refactoring](/reference/refactoring/).
