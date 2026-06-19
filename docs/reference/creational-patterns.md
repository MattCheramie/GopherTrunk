---
slug: creational-patterns
title: Creational patterns
entry_type: concept
category: paradigms-design
description: Creational patterns are the Gang of Four family that controls how objects get made, isolating creation logic so callers depend on interfaces rather than concrete classes.
keywords: creational patterns, factory pattern, factory method, builder pattern, singleton, abstract factory, prototype, object creation, gang of four, design patterns
aka: []
autolink: true
infobox:
  - { label: Type, value: "Design pattern family (GoF)" }
  - { label: Problem solved, value: "How objects get created" }
  - { label: Members, value: "Factory Method, Abstract Factory, Builder, Singleton, Prototype" }
  - { label: Benefit, value: "Decouple callers from concrete classes" }
  - { label: Watch for, value: "Singleton (often an anti-pattern)" }
  - { label: Sibling families, value: "Structural, behavioral" }
see_also: [design-patterns, structural-patterns, behavioral-patterns, object-oriented-programming, coupling-and-cohesion, solid, abstraction]
related_lessons:
  - { title: "Creational patterns: factory, builder, singleton", url: /learn/intro-software-dev/creational-patterns/ }
external:
  - { title: "Creational pattern — Wikipedia", url: https://en.wikipedia.org/wiki/Creational_pattern }
---

**Creational patterns** are the [Gang of Four](/reference/design-patterns/) family that
put a layer between "I need an object" and "here is exactly how it gets built," so the
rest of the code does not hard-code which concrete class to instantiate.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A caller asks a factory for an object and receives one through a common interface, without naming any concrete class." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="55" width="70" height="30" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="55" y="74">caller</text>
    <line x1="90" y1="70" x2="130" y2="70" stroke="currentColor" stroke-width="1.1"/><text x="110" y="62" font-size="8">asks</text>
    <rect x="130" y="50" width="90" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="175" y="68">factory</text><text x="175" y="80" font-size="8">decides type</text>
    <line x1="220" y1="70" x2="260" y2="70" stroke="currentColor" stroke-width="1.1"/>
    <rect x="260" y="20" width="100" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="310" y="35">P25Decoder</text>
    <rect x="260" y="58" width="100" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="310" y="73">DmrDecoder</text>
    <rect x="260" y="96" width="100" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="310" y="111">NxdnDecoder</text>
    <text x="310" y="135" font-size="8" fill-opacity="0.7">returned as Decoder</text>
  </g>
</svg>
<figcaption>A factory chooses the concrete type and returns it behind a common interface, so callers stay decoupled.</figcaption>
</figure>

## The problem they solve

When code says `new ConcreteThing(...)` directly, two things get baked in: the *decision*
of which type to use and the *details* of how to construct it. That binding hurts when the
type should depend on runtime input, construction is complicated, or you want exactly one
shared instance. Creational patterns isolate that logic so callers depend on an
[interface](/reference/abstraction/), reducing [coupling](/reference/coupling-and-cohesion/)
and honouring the open/closed principle from [SOLID](/reference/solid/).

## The main members

- **Factory / Factory Method** — create and return an object through a common interface so
  the caller never names the concrete class; Factory Method defers the choice to a
  subclass-overridable step.
- **Abstract Factory** — create whole *families* of related objects (a matching decoder,
  filter, and display for a mode).
- **Builder** — assemble one complex object step by step with named, chainable steps,
  ideal when a single constructor would be an unreadable wall of arguments.
- **Prototype** — create new objects by cloning an existing instance.
- **Singleton** — guarantee one shared instance with global access.

## Notes and caveats

Use a **Factory** when *which* type varies; use a **Builder** when *how* one complex
object is built varies — they compose well, since a factory can use a builder internally.
**Singleton** is often an anti-pattern: it is global mutable state in disguise, with hidden
dependencies and poor testability, so prefer passing a shared instance in explicitly. The
sibling families are [structural patterns](/reference/structural-patterns/) (arranging
objects) and [behavioral patterns](/reference/behavioral-patterns/) (coordinating them).
