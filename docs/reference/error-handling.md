---
slug: error-handling
title: Error handling
entry_type: concept
category: principles-quality
description: Error handling is the part of a program's design that deals with failures — exceptions, error returns, or Result types — treating errors as normal events to be validated, propagated, and recovered from rather than exceptional afterthoughts.
keywords: error handling, exceptions, error returns, Result type, defensive programming, fail fast, graceful degradation, input validation, idempotency, robustness
aka: []
autolink: true
infobox:
  - { label: Category, value: "Robustness / program design" }
  - { label: Mindset, value: "Errors are normal, not exceptional" }
  - { label: Styles, value: "Exceptions, error returns, Result/Option" }
  - { label: Strategies, value: "Fail fast vs fail soft" }
  - { label: Anti-pattern, value: "Silently swallowing errors" }
see_also: [api, clean-code, solid, unit-testing, rest, type-system]
related_lessons:
  - { title: "Errors, edge cases & defensive programming", url: /learn/intro-software-dev/robustness-and-errors/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Exception_handling
---

**Error handling** is the part of a program's design that deals with failures, treating
errors as a normal, first-class part of the system rather than an exceptional
afterthought bolted on at the end.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="An operation branches into a success path and an error path; the error path either fails fast or fails soft." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="50" width="80" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="60" y="69">operation</text>
    <line x1="100" y1="60" x2="150" y2="35" stroke="currentColor" stroke-width="1.1"/><line x1="100" y1="70" x2="150" y2="95" stroke="currentColor" stroke-width="1.1"/>
    <rect x="150" y="20" width="90" height="28" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="195" y="38">success path</text>
    <rect x="150" y="80" width="90" height="28" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="195" y="98">error path</text>
    <line x1="240" y1="94" x2="290" y2="72" stroke="currentColor" stroke-width="1.1"/><line x1="240" y1="94" x2="290" y2="116" stroke="currentColor" stroke-width="1.1"/>
    <rect x="290" y="58" width="80" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="330" y="75">fail fast</text>
    <rect x="290" y="100" width="80" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="330" y="117">fail soft</text>
  </g>
</svg>
<figcaption>Every operation has a success and an error path; the error path either stops loudly or degrades gracefully.</figcaption>
</figure>

## Styles of handling

Languages represent and propagate errors differently, and each style has trade-offs:

- **Exceptions** (Java, Python, C#) are thrown and unwind the stack until caught.
  Convenient, but easy to ignore — an uncaught exception crashes the program, and a
  signature doesn't reveal what it might throw.[^wiki]
- **Explicit error returns** (Go) return an error value alongside the result; the
  caller must decide what to do. Verbose, but the error path is impossible to overlook.
- **Result / Option types** (Rust, many functional languages) make errors part of the
  return type, and the [type system](/reference/type-system/) forces the caller to
  handle both cases.

The modern lean is toward explicit, visible handling. Whatever the style, the universal
anti-pattern is swallowing an error silently — an empty `catch {}` or an ignored return
turns a recoverable problem into a mysterious one later.

## Strategies and good practice

Robust error handling rests on a few habits. **Validate at the boundary** — distrust
input the moment it enters the system, so interior code can assume well-formed values.
Choose deliberately between **fail fast** (stop loudly on programmer bugs and broken
invariants) and **fail soft** / graceful degradation (carry on with reduced
functionality on expected, recoverable trouble). Make retry-able operations
**idempotent** so a retry after an ambiguous failure can't cause duplicate work. These
ideas shape how an [API](/reference/api/) or a [REST](/reference/rest/) service reports
problems to its callers, and they are only trustworthy when proven under bad inputs —
which is the job of [tests](/reference/unit-testing/).

## Sources

[^wiki]: [Exception handling](https://en.wikipedia.org/wiki/Exception_handling) — Wikipedia, on error and exception handling styles and the trade-offs between them.
