---
slug: abstraction-coupling
title: Abstraction, coupling & cohesion
description: Hiding detail behind interfaces, measuring how modules depend on each other, and why loose coupling plus high cohesion is the goal of good design.
keywords: abstraction, coupling, cohesion, loose coupling, high cohesion, information hiding, encapsulation, leaky abstraction, interface design, software modularity
level: intermediate
status: full
faq:
  - q: "What's the difference between coupling and cohesion?"
    a: "Coupling measures how much one module depends on another — how entangled they are. Cohesion measures how focused a single module is — how well its parts belong together. The goal is loose coupling (modules barely depend on each other's internals) and high cohesion (each module does one well-defined job)."
  - q: "What is a leaky abstraction?"
    a: "An abstraction is leaky when its hidden details bleed through and the caller has to know about them anyway. A \"simple\" file interface that mysteriously fails on network paths is leaking the underlying network's behavior. Leaks aren't always avoidable, but they undermine the point of hiding detail."
  - q: "How are abstraction and encapsulation related?"
    a: "Abstraction is about exposing a simple *what* — a clean interface — while hiding the complex *how*. Encapsulation (information hiding) is the mechanism that enforces it: keeping internal state and helpers private so callers can't reach in and depend on them. Abstraction is the goal; encapsulation is how you protect it."
---
# Abstraction, coupling & cohesion

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Abstraction** — hide the *how* behind a clean *what*. **Coupling & cohesion** — aim for loose coupling and high cohesion. **Information hiding** — keep internals private so callers can't depend on them.
</div>

The [SOLID principles](/learn/intro-software-dev/solid/) all rest on a few deeper ideas. This lesson names them directly: **abstraction** (hiding detail behind an interface), **coupling** (how much modules depend on each other), and **cohesion** (how focused a module is). Get these right and your code stays flexible — parts can be understood, tested, and changed in isolation. Get them wrong and you end up with a brittle web where every change ripples everywhere. These aren't abstract niceties; they're the difference between a codebase you can evolve and one you're afraid to touch.

## Abstraction: a clean *what* over a messy *how*

Abstraction means **exposing a simple interface that describes what something does, while hiding how it does it.** When you call `sort()`, you don't know or care whether it's quicksort or mergesort — you depend on the promise ("returns sorted output"), not the mechanism. That contract is the abstraction; everything behind it is free to change.

Good abstractions shrink the amount you have to hold in your head. Instead of reasoning about a thousand lines of filter math, you reason about a single operation with a clear name and a clear contract. Consider a DSP block in a radio pipeline:

```go
// The abstraction: a clean contract.
type Stage interface {
    Process(samples []complex128) []complex128
}

// A concrete stage hides a lattice of filter coefficients, gain
// staging, and buffering behind one method.
type BandpassFilter struct {
    coeffs []float64   // hidden internals — callers never see these
    state  []complex128
}
func (f *BandpassFilter) Process(samples []complex128) []complex128 {
    // ... filtering math the caller doesn't need to understand ...
}
```

Any pipeline can chain stages — `Process(samples) -> samples` — without knowing a single thing about each stage's internals. Swap the bandpass filter for a notch filter and the pipeline code doesn't change. That's the power of a well-drawn `what`.

## Information hiding and encapsulation

Abstraction only holds if the internals stay genuinely hidden. **Information hiding** is the discipline of keeping a module's internal state and helper logic private, exposing only the interface. **Encapsulation** is the language mechanism that enforces it — private fields, unexported names, access modifiers.

Why it matters: anything you expose, someone will depend on. The moment callers reach into `filter.coeffs` directly, those coefficients become part of your public contract, and you can no longer change them freely without breaking callers. By keeping them private, you reserve the right to change *how* the filter works without anyone noticing — because all they ever touched was `Process`.

The rule of thumb: **expose as little as possible.** A small public surface is a small set of promises you have to keep. In Go, this is the capitalization convention — `Process` is public, `coeffs` is private — and it's not cosmetic; it's the contract boundary.

## Coupling: how much modules depend on each other

**Coupling** measures how entangled two modules are. Tightly coupled modules know each other's internals, so a change in one forces a change in the other. Loosely coupled modules interact only through narrow, stable interfaces, so each can change behind its contract without disturbing its neighbors.

A rough spectrum from worse to better:

- **Tight (content coupling)** — one module reaches into another's internal state directly.
- **Shared global state** — modules communicate through global variables, creating invisible dependencies.
- **Data coupling** — modules pass exactly the data they need through a clean interface. This is the goal.

The aim is **loose coupling**: minimize what each module needs to know about any other. Loose coupling is what makes code testable (you can substitute a fake for a dependency), reusable (a module isn't welded to its original neighbors), and safe to change (effects stay local). The [Dependency Inversion Principle](/learn/intro-software-dev/solid/) is precisely a technique for loosening coupling — depend on an interface, not a concrete type.

## Cohesion: how focused a module is

**Cohesion** is the flip side. Where coupling is *between* modules, cohesion is *within* one: how well do its parts belong together? A highly cohesive module does one well-defined job, and everything in it serves that job. A low-cohesion module is a grab-bag — a `Utils` class that validates emails, parses dates, and resizes images has no coherent reason to exist as a unit.

The target is **high cohesion**: each module focused on a single responsibility (you'll recognize this as SRP from the SOLID lesson). High cohesion makes a module easy to name, easy to understand, and easy to change for one reason at a time.

The two ideas combine into the central slogan of modular design:

> **Aim for loose coupling and high cohesion.**

| | Loose coupling | High cohesion |
|---|---|---|
| **Scope** | Between modules | Within a module |
| **Means** | Few, narrow dependencies | One focused responsibility |
| **Payoff** | Local changes, easy testing | Clear purpose, simple to reason about |

When both hold, your modules behave like well-designed components: self-contained, with clear connectors. When coupling is tight or cohesion is low, you get the dreaded "big ball of mud" where nothing can move without everything moving.

## Leaky abstractions

Abstractions are powerful, but they're never perfect — and pretending otherwise causes bugs. A **leaky abstraction** is one whose hidden details surface anyway, forcing the caller to know what the abstraction claimed to hide. Joel Spolsky's "Law of Leaky Abstractions" puts it bluntly: all non-trivial abstractions leak to some degree.

Examples:

- A network file path behind a "looks like a local file" interface — fast until the network hiccups and the abstraction stalls or fails in ways a real local file never would.
- An ORM that lets you ignore SQL — until a query is slow and you must understand the SQL it generates.
- A radio `Stage` whose `Process` silently assumes a specific sample rate — callers feeding a different rate get garbage, because an assumption leaked through the clean signature.

You can't eliminate leaks, but you can manage them: document the assumptions and edge behavior, make failure modes explicit (return an error rather than misbehave silently), and keep the abstraction honest about what it promises. An abstraction that hides *less* but never lies is often better than one that hides everything and occasionally deceives.

<div class="knowledge-check" data-quiz data-correct-msg="Right — loose coupling between modules and high cohesion within them is the central goal of modular design." markdown="0">
  <p class="knowledge-check__q">Quick check: what combination of coupling and cohesion are you aiming for?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Tight coupling and low cohesion</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Loose coupling and high cohesion</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Tight coupling and high cohesion</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Abstraction** — expose a simple *what* (a contract) and hide the messy *how* behind it.
- **Information hiding** — keep internals private; anything you expose becomes a promise you must keep.
- **Coupling** — minimize how much modules depend on each other; loose coupling keeps changes local.
- **Cohesion** — keep each module focused on one job; high cohesion makes purpose clear.
- **The goal** — loose coupling plus high cohesion, the backbone of modular, changeable design.
- **Leaky abstractions** — no abstraction is perfect; document assumptions and surface failures honestly.

Next up: what happens when the world doesn't cooperate — [errors, edge cases and defensive programming](/learn/intro-software-dev/robustness-and-errors/).
