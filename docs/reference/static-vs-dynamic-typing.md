---
slug: static-vs-dynamic-typing
title: Static vs dynamic typing
entry_type: concept
category: language-internals
description: Static typing checks the types of values at compile time before a program runs, while dynamic typing checks them at run time as each operation executes.
keywords: static typing, dynamic typing, compile time, run time, type checking, type inference, gradual typing, duck typing, strong typing
aka: ["static typing", "dynamic typing"]
autolink: true
infobox:
  - { label: Category, value: Typing discipline }
  - { label: The axis, value: "When types are checked" }
  - { label: Static, value: "At compile time, before running" }
  - { label: Dynamic, value: "At run time, per operation" }
  - { label: Static examples, value: "C, Rust, Go, Java" }
  - { label: Dynamic examples, value: "Python, Ruby, JavaScript" }
see_also: [type-system, compiler, interpreter, typescript-language, python-language, go-language, rust-language]
related_lessons:
  - { title: "Type systems and safety", url: /learn/intro-software-dev/type-systems/ }
  - { title: "Performance vs productivity", url: /learn/intro-software-dev/performance-vs-productivity/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Type_system#Static_and_dynamic_type_checking_in_practice
---

**Static versus dynamic typing** is about *when* a language checks the types of its
values: a **statically typed** language checks at compile time, before the program
runs, while a **dynamically typed** language checks at run time, as each operation
executes.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="Static typing checks types during the build step before running; dynamic typing checks each operation while the program runs." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="115" y="20" font-size="8" fill-opacity="0.7">static</text>
    <rect x="20" y="28" width="70" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="55" y="47">source</text>
    <line x1="90" y1="43" x2="130" y2="43" stroke="currentColor" stroke-width="1.1"/>
    <rect x="130" y="28" width="80" height="30" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="170" y="47">check + run</text>
    <text x="115" y="80" font-size="8" fill-opacity="0.7">dynamic</text>
    <rect x="20" y="88" width="70" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="55" y="107">source</text>
    <line x1="90" y1="103" x2="130" y2="103" stroke="currentColor" stroke-width="1.1"/>
    <rect x="130" y="88" width="60" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="160" y="107">run...</text>
    <line x1="190" y1="103" x2="230" y2="103" stroke="currentColor" stroke-width="1.1"/>
    <rect x="230" y="88" width="90" height="30" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="275" y="107">check on use</text>
  </g>
</svg>
<figcaption>Static typing checks before the program runs; dynamic typing checks each operation as it executes.</figcaption>
</figure>

## How they differ

Under **static typing**, the [compiler](/reference/compiler/) knows the type of every
variable and rejects code that misuses it before you can run it — passing a string
where a number is expected is a build error. Under **dynamic typing**, a variable can
hold a string now and a number later; the [interpreter](/reference/interpreter/) only
complains when an actual operation is invalid. This is a separate question from *how
strictly* a language enforces types (strong versus weak), so the two axes combine:
Rust is static and strong, Python dynamic and strong, JavaScript dynamic and weak.[^wiki]

## Trade-offs

Static typing catches **whole classes of bugs at compile time** — wrong types, missing
methods, broken refactors — and makes APIs self-documenting, which is why large,
long-lived codebases drift toward it. The cost is ceremony and some rigidity, though
[type inference](/reference/type-system/) removes most of the annotation burden.
Dynamic typing offers **flexibility and less ceremony** — quick scripts, duck typing,
runtime metaprogramming — at the risk of type errors that hide until that line runs.[^wiki]

## In practice

[C](/reference/c-language/), [Rust](/reference/rust-language/),
[Go](/reference/go-language/), and Java are statically typed;
[Python](/reference/python-language/), [Ruby](/reference/ruby-language/), and
[JavaScript](/reference/javascript-language/) are dynamic. *Gradual typing* bridges the
two: [TypeScript](/reference/typescript-language/) and Python type hints add optional,
checkable types to dynamic languages, keeping flexibility where you want it and adding
guarantees where the code is critical.

## Sources

[^wiki]: [Type system — static and dynamic type checking](https://en.wikipedia.org/wiki/Type_system#Static_and_dynamic_type_checking_in_practice) — Wikipedia, on when type checking happens and how the static/dynamic axis combines with strong/weak typing.
