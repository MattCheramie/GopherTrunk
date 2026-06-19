---
slug: functional-programming
title: Functional programming
entry_type: concept
category: paradigms-design
description: Functional programming (FP) treats computation as the evaluation of pure functions, favouring immutable data, higher-order functions, and composing small transformations into pipelines.
keywords: functional programming, FP, pure functions, immutability, higher-order functions, side effects, map filter reduce, composition, declarative
aka: [functional programming "FP"]
autolink: true
infobox:
  - { label: Type, value: "Programming paradigm" }
  - { label: Key idea, value: "Compose pure, side-effect-free functions" }
  - { label: Favours, value: "Immutable data, higher-order functions" }
  - { label: Benefit, value: "Easy to test and parallelise" }
  - { label: Strong examples, value: "Lisp, Haskell, Clojure, Elixir" }
  - { label: A kind of, value: "Declarative programming" }
see_also: [imperative-programming, declarative-programming, object-oriented-programming, abstraction, concurrency, lisp-language, python-language]
related_lessons:
  - { title: "Paradigms & language families", url: /learn/intro-software-dev/language-families/ }
external:
  - { title: "Functional programming — Wikipedia", url: https://en.wikipedia.org/wiki/Functional_programming }
---

**Functional programming** (**FP**) treats computation as the evaluation of functions,
ideally **pure** ones — functions that always return the same output for the same input
and have no side effects.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="Input data flows through a chain of pure functions — map, filter, reduce — each producing a new value without mutating shared state." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="55" width="55" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="47" y="74">input</text>
    <line x1="75" y1="70" x2="110" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="110" y="52" width="60" height="36" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="140" y="73">map</text>
    <line x1="170" y1="70" x2="200" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="200" y="52" width="60" height="36" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="230" y="73">filter</text>
    <line x1="260" y1="70" x2="290" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="290" y="52" width="60" height="36" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="320" y="73">reduce</text>
    <line x1="350" y1="70" x2="385" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="385" y="55" width="55" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="412" y="74">output</text>
    <text x="230" y="118" font-size="8" fill-opacity="0.7">no shared state mutated</text>
  </g>
</svg>
<figcaption>Data flows through a pipeline of pure functions, each producing a new value rather than mutating shared state.</figcaption>
</figure>

## Core ideas

Pure functions are easy to test, reason about, and run in parallel, because they cannot
interfere with each other. FP favours **immutable data**, **higher-order functions**
(functions that take or return other functions), and composing small transformations
into pipelines — `map`, `filter`, and `reduce` are its everyday vocabulary. Because pure
code has no hidden state to corrupt, it sidesteps an entire class of bugs and meshes well
with [concurrency](/reference/concurrency/).

## The mutable-vs-pure axis

FP contrasts sharply with [imperative programming](/reference/imperative-programming/),
which says "change this variable, then that one." Functional code instead says "produce a
new value from old values." FP is one branch of
[declarative programming](/reference/declarative-programming/): you describe
transformations rather than spelling out step-by-step state changes. This makes it a
natural fit for dataflow problems where each stage is a clean transformation.

## In practice

[Lisp](/reference/lisp-language/) is the original functional language; Haskell is purely
functional, while Clojure, Elixir, and Scala lean functional. Most mainstream languages —
[Python](/reference/python-language/),
[JavaScript](/reference/javascript-language/), and others — now offer functional tools
alongside [object-oriented](/reference/object-oriented-programming/) features, so the
paradigm is usually a style you choose per piece of code rather than a whole language you
adopt.
