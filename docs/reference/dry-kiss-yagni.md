---
slug: dry-kiss-yagni
title: DRY, KISS & YAGNI
entry_type: concept
category: principles-quality
description: DRY, KISS and YAGNI are three software-design heuristics — don't repeat yourself, keep it simple, and you aren't gonna need it — that keep code lean, simple, and free of speculative complexity.
keywords: DRY, KISS, YAGNI, don't repeat yourself, keep it simple, you aren't gonna need it, separation of concerns, over-engineering, wrong abstraction, design heuristics
aka: [DRY, KISS, YAGNI]
autolink: true
infobox:
  - { label: Category, value: "Software design heuristics" }
  - { label: DRY, value: "Don't Repeat Yourself" }
  - { label: KISS, value: "Keep It Simple" }
  - { label: YAGNI, value: "You Aren't Gonna Need It" }
  - { label: Prevents, value: "Drift, complexity, over-engineering" }
see_also: [solid, clean-code, refactoring, abstraction, coupling-and-cohesion, design-patterns]
related_lessons:
  - { title: "DRY, KISS, YAGNI & separation of concerns", url: /learn/intro-software-dev/dry-kiss-yagni/ }
external:
  - { title: "Don't repeat yourself — Wikipedia", url: https://en.wikipedia.org/wiki/Don%27t_repeat_yourself }
---

**DRY, KISS and YAGNI** are three of software's most-quoted design heuristics — short
slogans that pack hard-won judgement about keeping code lean, simple, and free of
speculative complexity.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="Three heuristics: DRY one home for each fact, KISS simplest solution, YAGNI build only what you need." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="35" width="120" height="50" rx="6" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="80" y="55" font-size="12">DRY</text><text x="80" y="72">one home per fact</text>
    <rect x="170" y="35" width="120" height="50" rx="6" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="230" y="55" font-size="12">KISS</text><text x="230" y="72">simplest that works</text>
    <rect x="320" y="35" width="120" height="50" rx="6" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="380" y="55" font-size="12">YAGNI</text><text x="380" y="72">build only what's needed</text>
  </g>
</svg>
<figcaption>Three heuristics that together hold back drift, complexity, and over-engineering.</figcaption>
</figure>

## The three heuristics

- **DRY — Don't Repeat Yourself.** Every piece of *knowledge* should have a single,
  authoritative home. If a rule or constant appears in five places, a change means
  editing all five — and missing one is a bug. The crucial nuance: DRY is about
  knowledge, not characters. Two snippets that look alike but encode different facts
  are not a violation, and merging them creates the "wrong abstraction," which Sandi
  Metz noted is far costlier than duplication.
- **KISS — Keep It Simple.** The simplest design that solves the *actual* problem is
  almost always best. Complexity is a cost you pay forever in comprehension and bugs.
  Match the complexity of the solution to the complexity of the problem, and not a
  notch more.
- **YAGNI — You Aren't Gonna Need It.** Build things when you actually need them, not
  when you merely foresee needing them. Speculative machinery for an imagined future is
  usually dead weight — extra code to read, test, and maintain, often guessing wrong.

## Why they matter

The three reinforce each other and a fourth idea, **separation of concerns**: skipping
speculative features (YAGNI) keeps today's design simple (KISS), and clean seams
between responsibilities let you deduplicate (DRY) correctly. They are the everyday
antidote to over-engineering and the foundation beneath [SOLID](/reference/solid/),
[clean code](/reference/clean-code/), and disciplined [refactoring](/reference/refactoring/).
Like all heuristics they are guidelines, not laws — the skill is knowing when *not* to
apply them, such as resisting a premature [abstraction](/reference/abstraction/) on the
first duplication.
