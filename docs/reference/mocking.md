---
slug: mocking
title: Mocking
entry_type: concept
category: testing-delivery
description: Mocking is the use of test doubles — mocks, fakes, and stubs — that stand in for a unit's real dependencies so it can be tested in isolation, deterministically and without hitting real networks, databases, or hardware.
keywords: mocking, mock, test double, fake, stub, isolation, dependency injection, deterministic tests, unit testing, software testing
aka: []
autolink: true
infobox:
  - { label: Category, value: "Software testing technique" }
  - { label: Purpose, value: "Isolate a unit from its dependencies" }
  - { label: Doubles, value: "Mock, fake, stub" }
  - { label: Enabled by, value: "Dependency inversion / interfaces" }
  - { label: Benefit, value: "Fast, deterministic tests" }
see_also: [unit-testing, integration-testing, end-to-end-testing, test-driven-development, code-coverage, solid, ci-cd]
related_lessons:
  - { title: "Testing — unit, integration & beyond", url: /learn/intro-software-dev/testing/ }
external:
  - { title: "Mock object — Wikipedia", url: https://en.wikipedia.org/wiki/Mock_object }
---

**Mocking** is the use of *test doubles* — stand-ins for a unit's real dependencies — so
the unit can be [tested](/reference/unit-testing/) in isolation, deterministically and
without touching real networks, databases, or hardware.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="Code under test calls a test double standing in for a real dependency such as a database or radio." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="48" width="100" height="40" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="70" y="65">code under</text><text x="70" y="78">test</text>
    <line x1="120" y1="68" x2="180" y2="68" stroke="currentColor" stroke-width="1.1"/><text x="150" y="60" font-size="8">calls</text>
    <rect x="180" y="48" width="100" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.4" stroke-dasharray="4 3"/><text x="230" y="65">test double</text><text x="230" y="78" font-size="8">mock / fake / stub</text>
    <line x1="280" y1="68" x2="340" y2="68" stroke="currentColor" stroke-width="1.1" stroke-dasharray="2 3"/>
    <rect x="340" y="48" width="100" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.1" stroke-opacity="0.5"/><text x="390" y="65" fill-opacity="0.55">real DB /</text><text x="390" y="78" font-size="8" fill-opacity="0.55">radio (skipped)</text>
  </g>
</svg>
<figcaption>A test double replaces the real dependency so the unit runs fast and deterministically.</figcaption>
</figure>

## Kinds of test double

"Mocking" is the everyday umbrella term, but there are distinct kinds of double:

- **Fake** — a lightweight working implementation, for example a sample source that
  replays a file instead of reading a live device.
- **Stub** — simply returns canned, pre-programmed responses, with no logic of its own.
- **Mock** — records *how* it was called and lets the test assert that the code
  interacted with it correctly (it called `send()` once, with these arguments).

The right double depends on what you're testing: a stub or fake when you only need the
dependency to be *present*, a mock when the interaction itself is the behavior under test.

## Why it works

Mocking depends on the [dependency inversion](/reference/solid/) idea: if your code
talks to an *interface* rather than a concrete class, a test can inject a double in place
of the real thing. That is what makes [unit tests](/reference/unit-testing/) fast and
deterministic — no flaky network, no slow database, no hardware in the loop. Good seams
make code mockable, and mockable code tends to be well-designed code, which is why
mocking pairs so naturally with [test-driven development](/reference/test-driven-development/).

## When not to mock

Doubles have a cost: a test that mocks everything verifies your code against your
*assumptions* about a dependency, not the real one. That is exactly the gap
[integration tests](/reference/integration-testing/) and
[end-to-end tests](/reference/end-to-end-testing/) fill by using real collaborators. Over-
mocking also inflates [code coverage](/reference/code-coverage/) without real assurance.
Mock at the boundaries of slow or external systems; let higher layers, run in
[CI/CD](/reference/ci-cd/), exercise the real seams.
