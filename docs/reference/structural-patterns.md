---
slug: structural-patterns
title: Structural patterns
entry_type: concept
category: paradigms-design
description: Structural patterns are the Gang of Four family that composes classes and objects into larger structures, mostly through wrapping, while keeping the parts loosely coupled.
keywords: structural patterns, adapter pattern, facade pattern, decorator pattern, proxy, composite, bridge, wrapper, composition, gang of four, design patterns
aka: []
autolink: true
infobox:
  - { label: Type, value: "Design pattern family (GoF)" }
  - { label: Problem solved, value: "How objects are composed into structures" }
  - { label: Members, value: "Adapter, Facade, Decorator, Proxy, Composite, Bridge" }
  - { label: Common mechanism, value: "One object wraps another" }
  - { label: Benefit, value: "Flexible assembly, loose coupling" }
  - { label: Sibling families, value: "Creational, behavioral" }
see_also: [design-patterns, creational-patterns, behavioral-patterns, object-oriented-programming, coupling-and-cohesion, abstraction, solid]
related_lessons:
  - { title: "Structural patterns: adapter, facade, decorator", url: /learn/intro-software-dev/structural-patterns/ }
external:
  - { title: "Structural pattern — Wikipedia", url: https://en.wikipedia.org/wiki/Structural_pattern }
---

**Structural patterns** are the [Gang of Four](/reference/design-patterns/) family that
answers *how do objects fit together into larger structures?* — mostly by taking objects
that already exist and arranging them so the whole stays flexible.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="Three structural shapes: an adapter wrapping one object to change its interface, a facade fronting many parts, and a decorator stacking wrappers." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="40" width="100" height="60" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><rect x="50" y="58" width="40" height="24" rx="3" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/><text x="70" y="34" font-size="8">adapter</text><text x="70" y="113" font-size="8">wraps 1</text>
    <rect x="180" y="40" width="100" height="60" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="230" y="34" font-size="8">facade</text><rect x="190" y="62" width="22" height="18" rx="2" fill="none" stroke="currentColor" stroke-width="1"/><rect x="219" y="62" width="22" height="18" rx="2" fill="none" stroke="currentColor" stroke-width="1"/><rect x="248" y="62" width="22" height="18" rx="2" fill="none" stroke="currentColor" stroke-width="1"/><text x="230" y="113" font-size="8">fronts many</text>
    <rect x="340" y="40" width="100" height="60" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><rect x="352" y="50" width="76" height="40" rx="3" fill="none" stroke="currentColor" stroke-width="1"/><rect x="364" y="60" width="52" height="20" rx="2" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/><text x="390" y="34" font-size="8">decorator</text><text x="390" y="113" font-size="8">stacks</text>
  </g>
</svg>
<figcaption>Adapter wraps one object to change its interface, Facade fronts many parts, and Decorator stacks wrappers.</figcaption>
</figure>

## What they are for

Real systems are assembled from parts written at different times by different people — a
third-party driver, your own code, a UI toolkit. Getting them to cooperate without welding
them tightly together is the recurring problem these patterns address. They share a family
resemblance: most involve one object **wrapping** another, and all lean on programming to
an [interface](/reference/abstraction/) so dependencies point at the contract rather than
concrete classes, keeping [coupling](/reference/coupling-and-cohesion/) low.

## The main members

- **Adapter** — converts one object's interface into the one a client expects; the
  universal travel plug of software, wrapping a single object to fix *compatibility*.
- **Facade** — a brand-new, simpler interface in front of a whole subsystem, fixing
  *complexity* by offering one easy entry point.
- **Decorator** — wraps an object with the *same* interface to add behavior (logging,
  gain) that stacks in any order at runtime, avoiding a subclass explosion.
- **Proxy** — stands in for another object to control access, add caching, or defer work.
- **Composite** — treats individual objects and groups of them uniformly through one
  interface.
- **Bridge** — separates an abstraction from its implementation so they vary independently.

## A subtle distinction

Adapter and Decorator look structurally identical — both wrap an object — but the *intent*
differs: Adapter changes the interface to make things fit, while Decorator keeps the
interface the same and adds behavior. Knowing the pattern *names* is exactly what lets a
team distinguish them when discussing a design. The sibling families are
[creational patterns](/reference/creational-patterns/) (making objects) and
[behavioral patterns](/reference/behavioral-patterns/) (coordinating them), and all three
support the open/closed principle from [SOLID](/reference/solid/).
