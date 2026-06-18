---
slug: solid
title: SOLID & object-oriented design
description: The five SOLID principles explained plainly with tiny examples — single responsibility, open/closed, Liskov, interface segregation, dependency inversion.
keywords: SOLID principles, object-oriented design, single responsibility principle, open closed principle, Liskov substitution, interface segregation, dependency inversion, OOP design
level: intermediate
status: full
faq:
  - q: "Do SOLID principles only apply to object-oriented languages?"
    a: "The classic formulations are OO-flavored — they talk about classes and interfaces. But the underlying ideas (one job per unit, depend on abstractions, don't force fat interfaces on callers) translate to any paradigm, including functional and procedural code. The vocabulary changes; the goal of changeable, decoupled code does not."
  - q: "What does \"depend on abstractions, not concretions\" actually mean?"
    a: "It means your high-level code should talk to an interface (a contract describing *what* it needs) rather than to a specific implementation (a concrete *how*). That way you can swap implementations — a real radio decoder, a test stub, a file replay — without changing the code that uses them."
  - q: "Is it bad to violate a SOLID principle?"
    a: "Not necessarily. SOLID is a set of heuristics, not laws. Applied dogmatically they cause as much over-engineering as they prevent. Use them to diagnose pain — code that's hard to change or test — rather than as a checklist to satisfy for its own sake."
---
# SOLID & object-oriented design

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**SRP & OCP** — one reason to change; open to extension, closed to modification. **LSP & ISP** — subtypes must honor the contract; keep interfaces small. **DIP** — depend on abstractions, not concrete details.
</div>

SOLID is a set of five design principles, popularized by Robert C. Martin, that aim at one outcome: code that's **easy to change without breaking**. They're framed in object-oriented terms — classes, interfaces, inheritance — but the spirit generalizes. Each principle is really a refinement of ideas from the [previous lesson](/learn/intro-software-dev/dry-kiss-yagni/): give each unit one job, and depend on contracts rather than concrete details. We'll take them one at a time with a tiny example each, then tie them together with a radio-decoder design that foreshadows [architectural patterns](/learn/intro-software-dev/architectural-patterns/).

A caution up front: SOLID is a toolbox of heuristics, not a law to satisfy. Reach for these when code resists change or testing — not to score points.

## S — Single Responsibility Principle

A unit of code should have **one reason to change** — it should answer to a single concern or stakeholder. When a class mixes responsibilities, a change to one drags in the others and risks breaking them.

```go
// Violates SRP: parsing, business logic, and persistence in one place.
type FrameHandler struct{}
func (h FrameHandler) Handle(raw []byte) {
    frame := parse(raw)      // parsing concern
    validate(frame)          // domain rule concern
    saveToDisk(frame)        // storage concern
}
```

Split into a parser, a validator, and a store, and each can change for its own reason — a new disk format doesn't touch parsing logic. SRP is separation of concerns applied at the level of a single class or module.

## O — Open/Closed Principle

Software should be **open for extension but closed for modification.** You should be able to add new behavior without editing existing, tested code — because editing working code risks breaking it.

The usual mechanism is an abstraction with multiple implementations:

```go
type Demodulator interface {
    Demod(samples []float64) []byte
}

// Adding FM support means writing a new type, not editing the old ones.
type AMDemod struct{}
type FMDemod struct{}
```

Code that consumes a `Demodulator` never changes when you add a new one. Contrast that with a giant `switch mode { case "AM": ...; case "FM": ... }` that you must reopen and re-test for every new mode. OCP is what lets a system grow at its edges instead of churning its core.

## L — Liskov Substitution Principle

Named after Barbara Liskov, this one is about inheritance done honestly: **a subtype must be usable anywhere its base type is expected, without surprising the caller.** If code works with a base type, swapping in a derived type shouldn't change correctness.

The classic violation is the rectangle/square trap: a `Square` that inherits from `Rectangle` but overrides `setWidth` to also change the height breaks any code that assumed width and height move independently. The subtype technically *is* a rectangle but doesn't *behave* like one, so substitution fails.

The practical takeaway: a subtype must honor the **contract** of its parent — its expectations, guarantees, and invariants — not just its method signatures. If a subclass throws where the parent didn't, or tightens what inputs it accepts, it's an LSP violation waiting to bite a caller.

## I — Interface Segregation Principle

**Clients shouldn't be forced to depend on methods they don't use.** A fat interface with a dozen methods couples every implementer to all of them, even when they only care about two.

```go
// Fat — a read-only file source is forced to implement Write and Tune.
type Radio interface {
    Read() []float64
    Write([]float64)
    Tune(freqHz int)
    SetGain(db int)
}

// Segregated — small interfaces clients can compose as needed.
type SampleSource interface { Read() []float64 }
type Tunable      interface { Tune(freqHz int) }
```

Now a file-based replay source implements just `SampleSource`. Small, role-focused interfaces keep implementations honest and make dependencies explicit. Go's standard library leans hard on this — `io.Reader` and `io.Writer` are single-method interfaces precisely so anything can satisfy them.

## D — Dependency Inversion Principle

**High-level code should depend on abstractions, not on concrete low-level details** — and the abstraction shouldn't depend on the details either. In practice: your important logic talks to an interface, and the concrete implementation is supplied from outside (often called dependency injection).

```go
// High-level pipeline depends on the SampleSource interface, not a specific radio.
func Run(src SampleSource, d Demodulator) {
    for {
        d.Demod(src.Read())
    }
}
```

`Run` doesn't know or care whether `src` is a real SDR, a network stream, or a file replaying captured samples — which, not coincidentally, is exactly what makes it [testable](/learn/intro-software-dev/testing/): the test passes in a fake source. DIP is the principle that turns rigid, hard-to-test code into flexible, swappable components.

## Tying it together: a pluggable decoder

Watch how the principles combine into one design. Suppose GopherTrunk must decode several radio protocols and gain more over time. Define a contract:

```go
type Decoder interface {
    Name() string
    Decode(frame []byte) (Message, error)
}
```

- **SRP** — each decoder handles exactly one protocol.
- **OCP** — adding a protocol means writing a new `Decoder`, not editing existing ones.
- **LSP** — every decoder honors the same contract, so the pipeline treats them uniformly.
- **ISP** — the interface is tiny; decoders aren't forced to implement unrelated methods.
- **DIP** — the pipeline depends on `Decoder`, not on any concrete protocol.

The payoff: new radio protocols **slot in without touching existing code**, and you can register decoders dynamically — the seed of a plugin architecture we'll explore in [architectural patterns](/learn/intro-software-dev/architectural-patterns/) and [what are patterns](/learn/intro-software-dev/what-are-patterns/). The same idea drives the [RF & SDR path](/learn/rf-sdr/), where many modulation schemes share one processing skeleton.

<div class="knowledge-check" data-quiz data-correct-msg="Right — DIP means high-level logic depends on an interface, so implementations can be swapped freely." markdown="0">
  <p class="knowledge-check__q">Quick check: what does the Dependency Inversion Principle ask you to do?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Make every class inherit from a common base class</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Depend on abstractions (interfaces), not concrete implementations</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Invert the order in which dependencies are installed</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **SRP** — one reason to change; don't mix unrelated responsibilities in one unit.
- **OCP** — add behavior by extension, not by editing tested code; abstractions over big switches.
- **LSP** — subtypes must honor the parent's contract so substitution never surprises callers.
- **ISP** — keep interfaces small and role-focused; don't force clients to depend on unused methods.
- **DIP** — high-level logic depends on abstractions; inject concrete implementations from outside.
- **SOLID is OO-flavored but general** — the deeper ideas (one job, depend on contracts) apply to any paradigm.

Next up: the concepts that underpin all of SOLID — [abstraction, coupling and cohesion](/learn/intro-software-dev/abstraction-coupling/).
