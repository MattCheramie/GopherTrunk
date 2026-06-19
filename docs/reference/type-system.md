---
slug: type-system
title: Type system
entry_type: concept
category: language-internals
description: A type system is the set of rules a language uses to track the types of values and decide which operations are allowed, catching certain mistakes before or during execution.
keywords: type system, type checking, static typing, dynamic typing, strong typing, type inference, generics, type safety, null safety
aka: [type system]
autolink: true
infobox:
  - { label: Category, value: Language semantics }
  - { label: Purpose, value: "Track value types, allow only valid operations" }
  - { label: When checked, value: "Compile time (static) or run time (dynamic)" }
  - { label: Strictness, value: "Strong vs weak coercion" }
  - { label: Reduces ceremony via, value: "Type inference" }
  - { label: Trade-off, value: "Safety up front vs flexibility & brevity" }
see_also: [static-vs-dynamic-typing, compiler, interpreter, memory-management, rust-language, typescript-language, go-language]
related_lessons:
  - { title: "Type systems and safety", url: /learn/intro-software-dev/type-systems/ }
  - { title: "Performance vs productivity", url: /learn/intro-software-dev/performance-vs-productivity/ }
external:
  - { title: "Type system — Wikipedia", url: https://en.wikipedia.org/wiki/Type_system }
---

**A type system** is the set of rules a language uses to track the type of every value
— integer, string, list, a `Receiver` object — and decide which operations are allowed,
catching whole classes of mistakes that would otherwise surface as bugs.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A type checker accepts an operation on matching types and rejects one that mixes incompatible types." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="25" width="60" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="50" y="42">int + int</text>
    <line x1="80" y1="38" x2="150" y2="50" stroke="currentColor" stroke-width="1.1"/>
    <rect x="20" y="75" width="60" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="50" y="92">int + str</text>
    <line x1="80" y1="88" x2="150" y2="62" stroke="currentColor" stroke-width="1.1"/>
    <rect x="150" y="40" width="90" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="195" y="61">type check</text>
    <line x1="240" y1="50" x2="310" y2="35" stroke="currentColor" stroke-width="1.1"/>
    <rect x="310" y="20" width="90" height="28" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="355" y="38">accepted</text>
    <line x1="240" y1="64" x2="310" y2="82" stroke="currentColor" stroke-width="1.1"/>
    <rect x="310" y="68" width="90" height="28" rx="4" fill="none" stroke="currentColor" stroke-width="1.2" stroke-dasharray="3 3"/><text x="355" y="86" fill-opacity="0.7">rejected</text>
  </g>
</svg>
<figcaption>The type checker permits operations on compatible types and rejects ones that mix incompatible types.</figcaption>
</figure>

## How it works

The type system assigns a type to each value and to the inputs and outputs of every
operation, then verifies that uses line up. *When* it checks is the
[static-versus-dynamic](/reference/static-vs-dynamic-typing/) axis: a
[compiled](/reference/compiler/) language usually checks at compile time, while a
[dynamic](/reference/interpreter/) language checks at run time as each operation
executes. A separate axis is *how strictly* it enforces types — **strong** systems
refuse to silently mix a string and a number, **weak** ones coerce. **Type inference**
lets the compiler deduce types from context, so you get checking without spelling out
every annotation.

## Trade-offs

A richer type system pays for itself by turning run-time surprises into compile errors:
wrong-type arguments, calling a method that does not exist, forgetting to handle a
null, and broken refactors all get caught for free. Expressive features — generics,
sum types, and Option/Result types — let you "make illegal states unrepresentable."
The cost is ceremony, some rigidity, and slower prototyping. A throwaway script barely
benefits; a large, long-lived, multi-author system benefits enormously.

## In practice

[Rust](/reference/rust-language/), [Go](/reference/go-language/), and Java pair static
checking with inference; [TypeScript](/reference/typescript-language/) adds a checkable
type layer over JavaScript. In domains like radio software, wrapping a frequency and a
sample rate in distinct types makes unit-confusion bugs impossible to compile rather
than merely unlikely.
