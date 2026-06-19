---
slug: behavioral-patterns
title: Behavioral patterns
entry_type: concept
category: paradigms-design
description: Behavioral patterns are the Gang of Four family that governs how objects interact and divide responsibility at runtime, keeping the communicating parties loosely coupled.
keywords: behavioral patterns, observer pattern, strategy pattern, state pattern, command, iterator, template method, publish subscribe, state machine, gang of four, design patterns
aka: []
autolink: true
infobox:
  - { label: Type, value: "Design pattern family (GoF)" }
  - { label: Problem solved, value: "How objects interact and share work" }
  - { label: Members, value: "Observer, Strategy, State, Command, Iterator, Template Method" }
  - { label: Focus, value: "Communication and flow of control" }
  - { label: Benefit, value: "Loose coupling between collaborators" }
  - { label: Sibling families, value: "Creational, structural" }
see_also: [design-patterns, creational-patterns, structural-patterns, object-oriented-programming, coupling-and-cohesion, solid, abstraction]
related_lessons:
  - { title: "Behavioral patterns: observer, strategy, state", url: /learn/intro-software-dev/behavioral-patterns/ }
external:
  - { title: "Behavioral pattern — Wikipedia", url: https://en.wikipedia.org/wiki/Behavioral_pattern }
---

**Behavioral patterns** are the [Gang of Four](/reference/design-patterns/) family that
handles the question once objects exist and are connected: *how do they talk to each other
and divide up the work?* They are about communication and responsibility at runtime.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A subject publishes an event to several observers at once, while a context delegates work to a swappable strategy or state object." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="52" width="80" height="36" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="60" y="74">subject</text>
    <line x1="100" y1="60" x2="150" y2="30" stroke="currentColor" stroke-width="1.1"/><line x1="100" y1="70" x2="150" y2="70" stroke="currentColor" stroke-width="1.1"/><line x1="100" y1="80" x2="150" y2="110" stroke="currentColor" stroke-width="1.1"/>
    <rect x="150" y="18" width="90" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="195" y="33">observer</text>
    <rect x="150" y="58" width="90" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="195" y="73">observer</text>
    <rect x="150" y="98" width="90" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="195" y="113">observer</text>
    <rect x="300" y="40" width="90" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="345" y="59">FmDemod</text>
    <rect x="300" y="80" width="90" height="30" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="345" y="99">AmDemod</text>
    <text x="345" y="30" font-size="8" fill-opacity="0.7">swappable strategy</text>
  </g>
</svg>
<figcaption>A subject notifies many observers at once, while a context delegates to a swappable strategy or state object.</figcaption>
</figure>

## What they are for

The hard part of a running program is rarely a single object — it is the *interactions*.
When a call is decoded, the UI, the logger, and the recorder all need to know; when the
user picks a mode, the right algorithm must run; when a follower moves from idle to active,
its whole behaviour changes. Handling each with ad-hoc code scatters logic and tightens
[coupling](/reference/coupling-and-cohesion/). Behavioral patterns give these interactions
a clean, named shape that keeps the parties from depending too directly on each other.

## The main members

- **Observer** — a one-to-many dependency where a *subject* notifies all its *observers*
  when it changes, without knowing who is listening; the basis of publish/subscribe and UI
  events.
- **Strategy** — a family of interchangeable algorithms behind a common interface,
  swappable at runtime (pick FM, AM, or SSB demodulation without branching logic).
- **State** — each state is its own object owning its behaviour and transitions, so the
  object appears to change its class; the object-oriented expression of a state machine.
- **Command** — wraps a request as an object so it can be queued, logged, or undone.
- **Iterator** — exposes the elements of a collection one at a time without revealing its
  structure.
- **Template Method** — a base class fixes the skeleton of an algorithm and lets subclasses
  fill in steps.

## A common thread

Observer, Strategy, and State all let you add reactions, algorithms, or states by adding
classes rather than editing existing code — the open/closed principle from
[SOLID](/reference/solid/) in action. They rely on programming to an
[interface](/reference/abstraction/), just like the sibling
[creational](/reference/creational-patterns/) and
[structural](/reference/structural-patterns/) families catalogued under
[design patterns](/reference/design-patterns/).
