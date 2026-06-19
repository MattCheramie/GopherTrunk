---
slug: integration-testing
title: Integration testing
entry_type: concept
category: testing-delivery
description: Integration testing checks that several components work correctly together — exercising the real seams between them, such as a parser feeding a decoder or code talking to a database.
keywords: integration testing, integration test, test pyramid, component interaction, seams, database testing, software testing, middle layer
aka: []
autolink: true
infobox:
  - { label: Category, value: "Software testing" }
  - { label: Scope, value: "Several components working together" }
  - { label: Pyramid layer, value: "Middle — some, medium speed" }
  - { label: Exercises, value: "Real seams between components" }
  - { label: Run by, value: "CI pipelines" }
see_also: [unit-testing, end-to-end-testing, mocking, code-coverage, test-driven-development, ci-cd, rest]
related_lessons:
  - { title: "Testing — unit, integration & beyond", url: /learn/intro-software-dev/testing/ }
external:
  - { title: "Integration testing — Wikipedia", url: https://en.wikipedia.org/wiki/Integration_testing }
---

**Integration testing** checks that several components work correctly *together*,
exercising the real seams between them — a parser feeding a decoder, or code talking to a
database.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="Three components connected in sequence, with the joins between them highlighted as what integration tests exercise." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="45" width="90" height="36" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="65" y="66">component A</text>
    <line x1="110" y1="63" x2="160" y2="63" stroke="currentColor" stroke-width="1.6"/><circle cx="135" cy="63" r="4" fill="currentColor"/>
    <rect x="160" y="45" width="90" height="36" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="205" y="66">component B</text>
    <line x1="250" y1="63" x2="300" y2="63" stroke="currentColor" stroke-width="1.6"/><circle cx="275" cy="63" r="4" fill="currentColor"/>
    <rect x="300" y="45" width="90" height="36" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="345" y="66">database</text>
    <text x="205" y="100" font-size="8">tests the joins, not just each box</text>
  </g>
</svg>
<figcaption>Integration tests exercise the seams where components meet, not just each component alone.</figcaption>
</figure>

## What it is

Where a [unit test](/reference/unit-testing/) isolates one piece, an integration test
deliberately wires several pieces together and checks that the *combination* behaves
correctly. The bugs it catches live in the joins — mismatched assumptions about data
formats, wrong ordering, broken contracts between an [API](/reference/api/) and its
caller, or a query that doesn't match the schema. These tests often use real
collaborators (a real database, a real file format) precisely because the seam is the
thing under test, though they may still stub out slow or external systems via
[mocking](/reference/mocking/).

## Where it sits

Integration tests form the *middle* layer of the test pyramid: fewer than unit tests and
slower, because they exercise real interactions rather than one isolated function. That
trade-off is deliberate — they give more end-to-end-like confidence than a unit test
while remaining faster and more targeted than a full
[end-to-end test](/reference/end-to-end-testing/). A healthy suite has a wide base of
unit tests, a smaller band of integration tests, and only a thin tip of E2E.

## In practice

Like all tests, integration tests only protect you if they run automatically. A
[CI/CD](/reference/ci-cd/) pipeline builds the code and runs the full suite on every
push, so a broken seam is caught within minutes. They contribute to
[code coverage](/reference/code-coverage/) and complement the
[test-driven development](/reference/test-driven-development/) habit of specifying
behavior before building it.
