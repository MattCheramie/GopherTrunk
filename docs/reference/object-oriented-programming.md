---
slug: object-oriented-programming
title: Object-oriented programming
entry_type: concept
category: paradigms-design
description: Object-oriented programming (OOP) is a paradigm that bundles state and the behaviour that acts on it into objects, organised around encapsulation, inheritance, and polymorphism.
keywords: object-oriented programming, OOP, objects, classes, encapsulation, inheritance, polymorphism, composition, methods, paradigm
aka: [object-oriented programming "OOP"]
autolink: true
infobox:
  - { label: Type, value: "Programming paradigm" }
  - { label: Key idea, value: "Bundle state + behaviour into objects" }
  - { label: Pillars, value: "Encapsulation, inheritance, polymorphism" }
  - { label: Prefer, value: "Composition over deep inheritance" }
  - { label: Strong examples, value: "Java, C#, Ruby, Python" }
  - { label: Contrast with, value: "Functional, imperative" }
see_also: [functional-programming, imperative-programming, abstraction, coupling-and-cohesion, design-patterns, java-language, csharp-language, ruby-language]
related_lessons:
  - { title: "Paradigms & language families", url: /learn/intro-software-dev/language-families/ }
external:
  - { title: "Object-oriented programming — Wikipedia", url: https://en.wikipedia.org/wiki/Object-oriented_programming }
---

**Object-oriented programming** (**OOP**) is a paradigm that bundles state and the
behaviour that acts on it into **objects** — units that combine *fields* (data) with
*methods* (functions that operate on that data).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="An object encapsulates private data behind public methods, and several objects share one interface so callers can treat them uniformly." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="30" width="120" height="80" rx="6" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="80" y="24" font-size="8">object</text>
    <circle cx="80" cy="62" r="16" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.2"/><text x="80" y="65">data</text>
    <text x="80" y="98" font-size="8">methods (public)</text>
    <line x1="140" y1="70" x2="200" y2="70" stroke="currentColor" stroke-width="1.1"/><text x="170" y="62" font-size="8">interface</text>
    <rect x="200" y="20" width="100" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="250" y="35">Shape.area()</text>
    <rect x="200" y="58" width="100" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="250" y="73">Circle.area()</text>
    <rect x="200" y="96" width="100" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="250" y="111">Square.area()</text>
    <line x1="300" y1="70" x2="350" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="350" y="56" width="90" height="28" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="395" y="73">caller</text>
  </g>
</svg>
<figcaption>An object hides its data behind methods, and shared interfaces let callers treat different types uniformly.</figcaption>
</figure>

## Core ideas

OOP rests on a few pillars. **Encapsulation** hides an object's internal data behind a
clean interface, the foundation of [abstraction](/reference/abstraction/) and information
hiding. **Inheritance** derives specialised types from general ones, and
**polymorphism** lets you treat different types uniformly through a shared interface —
calling `area()` without knowing whether the object is a circle or a square. These
combine to keep [coupling](/reference/coupling-and-cohesion/) low: callers depend on a
contract, not a concrete class.

## When it fits

OOP shines when a problem decomposes naturally into "things" with identity and lifecycle
— a `User`, a `Connection`, a tuned `Receiver`. Languages such as
[Java](/reference/java-language/), [C#](/reference/csharp-language/),
[Ruby](/reference/ruby-language/), and [Python](/reference/python-language/) support it
strongly. Most of the Gang of Four [design patterns](/reference/design-patterns/) are
expressed in object-oriented terms.

## Trade-offs

Critics note that overusing inheritance creates rigid, tangled hierarchies, which is why
modern OOP leans on **composition** ("has-a") over deep inheritance ("is-a"). Many
languages are multi-paradigm: the same codebase can mix object-oriented structure with
[functional](/reference/functional-programming/) transformations and
[imperative](/reference/imperative-programming/) inner loops, choosing the style that
fits each part of the problem.
