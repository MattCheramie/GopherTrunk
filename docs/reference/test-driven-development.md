---
slug: test-driven-development
title: Test-driven development (TDD)
entry_type: concept
category: testing-delivery
description: Test-driven development (TDD) is a practice in which you write a failing test first, then write just enough code to make it pass, then refactor — the "red, green, refactor" cycle.
keywords: test-driven development, TDD, red green refactor, test first, failing test, design through tests, Kent Beck, software testing
aka: [test-driven development "TDD"]
autolink: true
infobox:
  - { label: Category, value: "Development practice" }
  - { label: Cycle, value: "Red → Green → Refactor" }
  - { label: Order, value: "Test first, then code" }
  - { label: Popularized by, value: "Kent Beck" }
  - { label: Benefit, value: "Clarifies intent before building" }
see_also: [unit-testing, integration-testing, end-to-end-testing, mocking, code-coverage, refactoring, ci-cd]
related_lessons:
  - { title: "Testing — unit, integration & beyond", url: /learn/intro-software-dev/testing/ }
related_reading:
  - { title: "Build in the Open, Part 8: Testing — how to build and write tests", url: /blog/tutorials/build-in-the-open-08-testing-how-to-write-tests/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Test-driven_development
---

**Test-driven development (TDD)** flips the usual order of work: you write a failing test
*first*, then write just enough code to make it pass, then refactor — the rhythm known as
"red, green, refactor."[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="The TDD cycle: write a failing red test, write code to make it green, then refactor, and repeat." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <circle cx="90" cy="65" r="34" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="90" y="62" font-size="11">RED</text><text x="90" y="76" font-size="8">failing test</text>
    <line x1="124" y1="65" x2="196" y2="65" stroke="currentColor" stroke-width="1.2"/>
    <circle cx="230" cy="65" r="34" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.4"/><text x="230" y="62" font-size="11">GREEN</text><text x="230" y="76" font-size="8">make it pass</text>
    <line x1="264" y1="65" x2="336" y2="65" stroke="currentColor" stroke-width="1.2"/>
    <circle cx="370" cy="65" r="34" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="370" y="62" font-size="10">REFACTOR</text><text x="370" y="76" font-size="8">clean up</text>
    <path d="M370 99 C300 130 160 130 90 99" fill="none" stroke="currentColor" stroke-width="1.1" stroke-dasharray="4 3"/><text x="230" y="126" font-size="8">repeat</text>
  </g>
</svg>
<figcaption>The red-green-refactor loop: fail first, pass minimally, then improve the design.</figcaption>
</figure>

## The cycle

TDD proceeds in tight loops, popularized by Kent Beck:[^wiki]

- **Red** — write a small test for behavior that doesn't exist yet, and watch it fail.
  A test that passes *before* you write the code is testing nothing, so the failure is
  proof the test is real.
- **Green** — write the simplest code that makes the test pass, resisting the urge to
  build more than the test demands.
- **Refactor** — with a green suite as a safety net, clean up the code —
  [refactoring](/reference/refactoring/) toward better names and structure without
  changing behavior, confident any regression turns the suite red.

Then repeat for the next slice of behavior.

## Why it helps

Writing the test first forces you to clarify *what* the code should do — its contract and
edge cases — before deciding *how* to build it, which tends to produce simpler, more
[testable](/reference/unit-testing/) designs. The discipline also guarantees that every
piece of code is covered by a test that genuinely exercises it, and it pairs naturally
with continuous [CI/CD](/reference/ci-cd/) since the suite is always meant to be green.

## In practice

TDD is most associated with [unit tests](/reference/unit-testing/) but the same test-first
rhythm applies to [integration](/reference/integration-testing/) and
[end-to-end](/reference/end-to-end-testing/) levels, and it often relies on
[mocking](/reference/mocking/) to isolate the unit under test. It is a valuable
discipline rather than a mandate — even teams that don't follow it strictly benefit from
specifying behavior up front. Note that the [code coverage](/reference/code-coverage/) it
produces is a by-product, not the goal: well-asserted tests matter more than the number.

## Sources

[^wiki]: [Test-driven development](https://en.wikipedia.org/wiki/Test-driven_development) — Wikipedia, for the red-green-refactor cycle and Kent Beck's role in popularizing it.
