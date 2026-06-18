---
slug: creational-patterns
title: "Creational patterns: factory, builder, singleton"
description: Creational patterns control how objects get made. Learn Factory and Factory Method, Builder for complex assembly, and why Singleton is often an anti-pattern.
keywords: creational patterns, factory pattern, factory method, builder pattern, singleton, object creation, abstract factory, design patterns
level: intermediate
status: full
faq:
  - q: "What problem do creational patterns solve?"
    a: "They handle *how objects are created* so the rest of your code does not have to hard-code which concrete class to instantiate or how to assemble it. By isolating creation, you can swap implementations, build complex objects step by step, or control how many instances exist — all without rewriting the callers that use the result."
  - q: "What is the difference between a Factory and a Builder?"
    a: "A **Factory** decides *which* object to create and returns it ready to use, hiding the concrete class from the caller. A **Builder** assembles *one complex* object step by step, letting you set many optional parts before producing the finished result. Use a factory when the choice of type varies; use a builder when construction has many parameters or stages."
  - q: "Why is Singleton considered an anti-pattern?"
    a: "A **Singleton** guarantees one shared instance with a global access point, but that global state hides dependencies, makes code hard to test (you cannot easily substitute a fake), and creates tight coupling to the single instance. It is sometimes justified for genuinely unique resources, but passing a shared object explicitly is usually cleaner than reaching for a global singleton."
---
# Creational patterns: factory, builder, singleton

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Factory** — create objects without naming the concrete class, so callers stay decoupled from implementations. **Builder** — assemble a complex object step by step instead of a giant constructor. **Singleton** — one shared instance, useful occasionally but often an anti-pattern because of global state.
</div>

Every program has to create objects somewhere. The naive way is to write `new ConcreteThing(...)` wherever you need one. That works until the exact type needs to vary, or the constructor grows a dozen parameters, or you need to control how many instances exist. **Creational patterns** are the Gang of Four's answers to those situations: they put a layer between "I need an object" and "here is exactly how it gets built." This lesson covers the three most common — Factory, Builder, and Singleton — using a software-defined radio receiver as the running example. The pseudocode is deliberately tiny; the point is the shape, not the syntax.

## The problem creational patterns solve

When code says `new FskDecoder()` directly, two things are now baked in: the *decision* of which decoder to use, and the *details* of how to construct it. That tight binding becomes painful when:

- The concrete type should depend on runtime input (which protocol did we detect?).
- Construction is complicated (set the sample rate, the gain, the filter, the squelch, then start).
- You want exactly one of something shared across the program.

Creational patterns isolate that creation logic so the rest of the code depends on an **interface** or an abstract step, not a specific class. This is the open/closed principle from [SOLID](/learn/intro-software-dev/solid/) in action: you can add a new decoder without editing the callers that ask for one. It is also a direct way to reduce [coupling](/learn/intro-software-dev/abstraction-coupling/).

## Factory and Factory Method

A **Factory** is code whose job is to create and return objects, hiding the concrete class behind a common interface. The caller asks for "a decoder for this protocol" and receives something it can use, without knowing — or caring — which class it actually got.

Imagine a scanner that supports several digital protocols. Without a factory, every place that needs a decoder has a tangle of `if`/`else` choosing the class. With a factory, that decision lives in exactly one place:

```text
interface Decoder { decode(samples) }

function NewDecoder(protocol) -> Decoder {
  switch protocol {
    case "P25":   return new P25Decoder()
    case "DMR":   return new DmrDecoder()
    case "NXDN":  return new NxdnDecoder()
    default:      throw "unknown protocol"
  }
}

// caller never names a concrete class:
dec = NewDecoder(detectedProtocol)
dec.decode(samples)
```

The caller depends only on the `Decoder` interface. Adding a new protocol means writing one new class and adding one line to `NewDecoder` — no other code changes.

**Factory Method** is the more formal GoF variant. Instead of a standalone function, the creation step is a *method* that subclasses override. A base `ReceiverPipeline` class might declare `createDecoder()` as the step subclasses fill in: a `TrunkedPipeline` overrides it to return a trunking decoder, an `AnalogPipeline` returns an FM decoder. The base class controls the overall flow; subclasses control which object gets created. The intent is the same — defer the choice of concrete class — just expressed through inheritance rather than a function.

There is also **Abstract Factory**, which creates whole *families* of related objects (a matching decoder, filter, and display for a given mode), but Factory Method covers the everyday case.

It is worth being precise about *why* this helps, because "hide the constructor" sounds trivial. The real win is that the decision about which class to build is captured in one named place that the whole system shares. When a bug appears in protocol selection, you fix it once in `NewDecoder` rather than hunting through every call site that built a decoder by hand. When you want to log every decoder creation, or pool and reuse decoders instead of allocating fresh ones, the factory is the single hook where that policy lives. Centralizing creation turns a scattered, repeated decision into one maintainable seam — which is the recurring theme of every creational pattern.

## Builder

A **Builder** separates the construction of a complex object from its representation, so the same step-by-step process can produce different configurations. Reach for it when an object has many parts or many optional settings and a single constructor would be an unreadable wall of arguments.

Configuring a receiver is a perfect example. There are many knobs — frequency, sample rate, gain mode, filter bandwidth, squelch — and most have sensible defaults. A telescoping constructor (`Receiver(freq, rate, gain, bw, squelch, agc, ...)`) is error-prone; nobody remembers the order. A builder reads cleanly:

```text
rx = ReceiverBuilder()
       .tuneTo(854_000_000)
       .sampleRate(2_400_000)
       .gain("auto")
       .filterBandwidth(12_500)
       .squelch(-60)
       .build()
```

Each call sets one part and returns the builder, so they chain. Only `build()` produces the finished, validated `Receiver`. Benefits:

- **Readable** — each setting is named, so the call site documents itself.
- **Flexible** — set only what you need; defaults fill the rest.
- **Safe** — `build()` can validate the combination before handing back a half-configured object.

Use a Builder when *construction is complex*; use a Factory when *the type to construct varies*. They compose nicely — a factory can use a builder internally.

## Singleton — and why it is often an anti-pattern

A **Singleton** ensures a class has exactly one instance and provides a global point of access to it. The classic shape:

```text
class HardwareBus {
  static instance = null
  static get() {
    if (instance == null) instance = new HardwareBus()
    return instance
  }
}
// anywhere in the program:
HardwareBus.get().send(command)
```

The appeal is obvious: some things really are unique — a single physical USB radio, one connection to a piece of hardware — and a singleton makes that one instance reachable from anywhere.

But that "reachable from anywhere" is exactly the trap. A singleton is **global mutable state** wearing a class costume, and global state causes recurring problems:

- **Hidden dependencies.** A function that calls `HardwareBus.get()` depends on it, but nothing in its signature says so. You only discover the dependency by reading the body.
- **Hard to test.** You cannot easily swap in a fake hardware bus for a unit test, because the code reaches out to the global directly instead of receiving it as a parameter.
- **Tight coupling.** Every caller is welded to the one concrete instance, which is the opposite of what the other creational patterns work so hard to achieve.

The usual better option is **dependency injection**: create the one instance at the top of your program (the "composition root") and *pass it in* to whatever needs it. You still have one instance, but the dependency is explicit and you can substitute a test double. So treat Singleton as a pattern you should be able to recognize and justify, not a default. When you find yourself reaching for it for convenience rather than a genuine "there can only be one" constraint, that is usually the [decision framework](/learn/intro-software-dev/decision-framework/) telling you to pass the object explicitly instead.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a factory returns an object through a common interface so the caller never names the concrete class." markdown="0">
  <p class="knowledge-check__q">Quick check: What is the main job of a Factory?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Guarantee there is only one instance of a class in the whole program</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Create and return an object without the caller naming the concrete class</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Assemble one complex object step by step from many optional parts</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Creational patterns isolate object creation** — so callers depend on interfaces and abstract steps, not concrete classes or constructors.
- **Factory / Factory Method** — choose and return the right concrete type behind a common interface; add new types without touching callers (open/closed).
- **Builder** — assemble a complex object step by step with named, chainable steps; ideal when there are many parts or optional settings.
- **Factory vs Builder** — factory varies *which* type; builder controls *how* one complex object is built. They compose.
- **Singleton** — one shared instance with global access; occasionally justified, but usually an anti-pattern because of hidden dependencies and poor testability.
- **Prefer injection** — pass a shared instance in explicitly rather than reaching for a global singleton.

Next up: structural patterns — Adapter, Facade, and Decorator — which deal with how objects are *composed* into larger structures. See [Structural patterns](/learn/intro-software-dev/structural-patterns/).
