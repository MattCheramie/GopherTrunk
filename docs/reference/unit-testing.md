---
slug: unit-testing
title: Unit testing
entry_type: concept
category: testing-delivery
description: Unit testing checks a single small piece of code — one function or class — in isolation, forming the fast, numerous base of the test pyramid.
keywords: unit testing, unit test, test pyramid, isolation, test doubles, fast tests, assertions, regression, software testing
aka: []
autolink: true
infobox:
  - { label: Category, value: "Software testing" }
  - { label: Scope, value: "One function or class in isolation" }
  - { label: Pyramid layer, value: "Base — many, fast, focused" }
  - { label: Needs, value: "Mocks/fakes for dependencies" }
  - { label: Run by, value: "CI on every push" }
see_also: [integration-testing, end-to-end-testing, test-driven-development, mocking, code-coverage, ci-cd, refactoring]
related_lessons:
  - { title: "Testing — unit, integration & beyond", url: /learn/intro-software-dev/testing/ }
external:
  - { title: "Unit testing — Wikipedia", url: https://en.wikipedia.org/wiki/Unit_testing }
---

**Unit testing** checks a single small piece of code — one function or class — in
isolation, forming the fast, numerous base of the test pyramid.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="The test pyramid with a wide unit-testing base, a narrower integration layer, and a small end-to-end tip." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <path d="M230 20 L290 60 L170 60 Z" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="230" y="50" font-size="8">E2E</text>
    <path d="M170 60 L290 60 L320 95 L140 95 Z" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.3"/><text x="230" y="82">integration</text>
    <path d="M140 95 L320 95 L355 125 L105 125 Z" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.4"/><text x="230" y="115">unit (many, fast)</text>
  </g>
</svg>
<figcaption>Unit tests form the wide base of the test pyramid — many, fast, and focused.</figcaption>
</figure>

## What it is

A unit test exercises the smallest meaningful piece of behavior on its own, asserting
that for a given input the unit produces the expected output. Because each test targets
one unit, a failure points straight at the broken code. Unit tests are *fast* — a good
suite runs thousands in seconds — and that speed is what lets a developer run them
constantly and lets [CI/CD](/reference/ci-cd/) run them on every push. They are also the
safety net that makes [refactoring](/reference/refactoring/) safe: restructure freely,
and a red test catches any accidental behavior change instantly.

## Isolation

The "unit" in unit testing means the piece is tested *in isolation* from its
collaborators — no real network, database, or hardware in the loop. Standing in for
those dependencies is the job of [mocking](/reference/mocking/) and other test doubles
(fakes and stubs). This is where the [dependency inversion](/reference/solid/) idea pays
off: if your code depends on an interface rather than a concrete dependency, a test can
inject a fake and run deterministically. Good seams make code testable, and testable
code tends to be well-designed code.

## In the bigger picture

Unit tests sit at the base of the pyramid beneath fewer
[integration tests](/reference/integration-testing/) and a thin layer of
[end-to-end tests](/reference/end-to-end-testing/). They are the kind of test
[test-driven development](/reference/test-driven-development/) writes first, and the kind
[code coverage](/reference/code-coverage/) most directly measures — though coverage shows
only which lines *ran*, not whether the assertions checked anything meaningful.
