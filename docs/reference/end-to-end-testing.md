---
slug: end-to-end-testing
title: End-to-end testing
entry_type: concept
category: testing-delivery
description: End-to-end (E2E) testing drives the whole system the way a user would, from input to final output, giving the highest confidence that the product works at the cost of being slow and few.
keywords: end-to-end testing, E2E testing, system testing, test pyramid, user flow, full stack, slow tests, high confidence, software testing
aka: [end-to-end testing "E2E"]
autolink: true
infobox:
  - { label: Category, value: "Software testing" }
  - { label: Scope, value: "The whole system, input to output" }
  - { label: Pyramid layer, value: "Tip — few, slow, high confidence" }
  - { label: Perspective, value: "Exercises the system as a user" }
  - { label: Risk, value: "Slow, brittle, costly to maintain" }
see_also: [unit-testing, integration-testing, mocking, code-coverage, test-driven-development, ci-cd, rest]
related_lessons:
  - { title: "Testing — unit, integration & beyond", url: /learn/intro-software-dev/testing/ }
related_reading:
  - { title: "Build in the Open, Part 8: Testing — how to build and write tests", url: /blog/tutorials/build-in-the-open-08-testing-how-to-write-tests/ }
cite_urls:
  - https://en.wikipedia.org/wiki/System_testing
---

**End-to-end (E2E) testing** drives the whole system the way a user would, from input
all the way to final output — the highest-confidence test, but slow and kept few.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A test exercises the full path from user input through the running system to the final output." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="14" y="40" width="70" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="49" y="61">user input</text>
    <line x1="84" y1="57" x2="124" y2="57" stroke="currentColor" stroke-width="1.1"/>
    <rect x="124" y="40" width="90" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="169" y="61">frontend</text>
    <line x1="214" y1="57" x2="254" y2="57" stroke="currentColor" stroke-width="1.1"/>
    <rect x="254" y="40" width="90" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="299" y="61">backend + DB</text>
    <line x1="344" y1="57" x2="384" y2="57" stroke="currentColor" stroke-width="1.1"/>
    <rect x="384" y="40" width="62" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="415" y="61">output</text>
    <text x="230" y="95" font-size="8">one test spans the whole path</text>
  </g>
</svg>
<figcaption>An E2E test exercises the entire system from real input to final output, as a user experiences it.</figcaption>
</figure>

## What it is

An end-to-end test treats the system as a black box and verifies a complete user-facing
flow: it submits real input, lets the request travel through every layer — frontend,
backend, [API](/reference/api/), database, external services — and asserts on the final
result. Because nothing is stubbed, a passing E2E test is strong evidence the product
*actually works* the way a user will experience it, including the integration points that
unit and integration tests can miss.[^wiki]

## The trade-off

That confidence is expensive. E2E tests are slow (they spin up the whole system), brittle
(any layer changing can break them), and costly to maintain. They also localize poorly:
a failure tells you *something* broke, not *where*. For all these reasons they sit at the
narrow *tip* of the test pyramid — you keep only a thin layer of them, covering the most
important flows. The classic anti-pattern is the "inverted pyramid," a suite made mostly
of slow E2E tests, which becomes slow and flaky and tells you little about the cause of
a failure.

## How it fits

E2E complements the faster layers beneath it: many [unit tests](/reference/unit-testing/)
at the base, fewer [integration tests](/reference/integration-testing/) in the middle, a
handful of E2E on top. They are usually run in a [CI/CD](/reference/ci-cd/) pipeline
before release rather than on every commit, and even an E2E suite may use
[mocking](/reference/mocking/) for genuinely external dependencies. Like all tests, they
contribute to [code coverage](/reference/code-coverage/) but are no substitute for sharp,
well-asserted checks lower down, and they can be written test-first in the
[TDD](/reference/test-driven-development/) spirit.

## Sources

[^wiki]: [System testing](https://en.wikipedia.org/wiki/System_testing) — Wikipedia, on testing a complete, integrated system end to end as a user would.
