---
slug: code-coverage
title: Code coverage
entry_type: concept
category: testing-delivery
description: Code coverage measures the percentage of a codebase exercised by its test suite — which lines, branches, or functions ran — a useful signal for finding untested code but not a guarantee of correctness.
keywords: code coverage, test coverage, line coverage, branch coverage, coverage percentage, untested code, false confidence, software testing
aka: []
autolink: true
infobox:
  - { label: Category, value: "Software testing metric" }
  - { label: Measures, value: "Code exercised by the test suite" }
  - { label: Granularity, value: "Line, branch, function" }
  - { label: Good for, value: "Finding untested code" }
  - { label: Caveat, value: "A floor, not a proof of correctness" }
see_also: [unit-testing, integration-testing, end-to-end-testing, test-driven-development, mocking, ci-cd, refactoring]
related_lessons:
  - { title: "Testing — unit, integration & beyond", url: /learn/intro-software-dev/testing/ }
related_reading:
  - { title: "Build in the Open, Part 8: Testing — how to build and write tests", url: /blog/tutorials/build-in-the-open-08-testing-how-to-write-tests/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Code_coverage
---

**Code coverage** measures the percentage of a codebase exercised by its test suite —
which lines, branches, or functions ran — a useful signal for finding untested code, but
not a guarantee of correctness.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A bar of code lines, most shaded as covered by tests and a few left uncovered, summing to a coverage percentage." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="40" width="50" height="34" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.2"/>
    <rect x="70" y="40" width="50" height="34" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.2"/>
    <rect x="120" y="40" width="50" height="34" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.2"/>
    <rect x="170" y="40" width="50" height="34" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.2"/>
    <rect x="220" y="40" width="50" height="34" fill="none" stroke="currentColor" stroke-width="1.2" stroke-dasharray="3 3"/>
    <text x="145" y="90" font-size="8">covered</text><text x="245" y="90" font-size="8" fill-opacity="0.6">untested</text>
    <text x="370" y="62" font-size="13">80%</text>
  </g>
</svg>
<figcaption>Coverage reports how much code ran during tests — here 80% — but not whether the tests checked the right things.</figcaption>
</figure>

## What it measures

Coverage tools instrument the code and record which parts executed while the
[tests](/reference/unit-testing/) ran, reported as a percentage. Common granularities are
**line coverage** (which lines ran), **branch coverage** (which sides of each
conditional ran), and **function coverage**.[^wiki] The metric is genuinely useful for *finding
untested code*: a module sitting at 0% is a red flag, and watching coverage shows where to
aim the next tests. Coverage is typically computed in a [CI/CD](/reference/ci-cd/)
pipeline so the number is tracked on every change.

## Its limits

Coverage is widely misread. It measures whether a line *ran*, not whether your test
*checked* anything meaningful about it. You can reach 100% with tests that assert
nothing, miss every edge case, or compare against wrong expected values — and heavy
[mocking](/reference/mocking/) can inflate the number without real assurance. High
coverage with weak assertions is false confidence. Treat coverage as a **floor** — a
signal for gaps — never as a target to chase or a proof of correctness. A smaller suite
of sharp, well-asserted tests beats a sprawling one written to game a number.

## In context

Coverage spans every layer of the test pyramid — [unit](/reference/unit-testing/),
[integration](/reference/integration-testing/), and
[end-to-end](/reference/end-to-end-testing/) tests all contribute. It is a by-product of
disciplines like [test-driven development](/reference/test-driven-development/), not their
purpose, and it gives [refactoring](/reference/refactoring/) confidence by showing which
code a change's safety net actually protects.

## Sources

[^wiki]: [Code coverage](https://en.wikipedia.org/wiki/Code_coverage) — Wikipedia, for the metric, its granularities (line, branch, function), and its limits.
