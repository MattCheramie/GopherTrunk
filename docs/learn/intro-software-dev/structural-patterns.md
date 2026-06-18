---
slug: structural-patterns
title: "Structural patterns: adapter, facade, decorator"
description: Structural patterns compose objects into larger structures. Learn Adapter for incompatible interfaces, Facade for hiding complexity, and Decorator for wrapping behavior.
keywords: structural patterns, adapter pattern, facade pattern, decorator pattern, wrapper, interface, composition, design patterns
level: intermediate
status: full
faq:
  - q: "What do structural patterns do?"
    a: "Structural patterns describe how to *compose* classes and objects into larger structures while keeping them flexible. Instead of changing what objects do, they arrange how objects fit together — making mismatched interfaces work (Adapter), hiding a complex subsystem behind a simple front (Facade), or wrapping an object to add behavior (Decorator)."
  - q: "What is the difference between Adapter and Facade?"
    a: "An **Adapter** converts one existing interface into another the client expects, usually wrapping a single object so it fits where it otherwise would not. A **Facade** provides a brand-new, simpler interface in front of a whole subsystem of many parts. Adapter is about *compatibility*; Facade is about *simplicity*."
  - q: "How is Decorator different from just editing the class?"
    a: "A **Decorator** wraps an object and adds behavior from the outside, sharing the same interface, so you can stack features (logging, gain, buffering) in any combination at runtime without modifying the original class. Editing the class changes it for everyone and cannot be combined or removed per use — decoration keeps the core untouched and the additions optional."
---
# Structural patterns: adapter, facade, decorator

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Adapter** — wrap an object so an incompatible interface fits where the client expects. **Facade** — put a simple front over a complex subsystem so callers see one easy entry point. **Decorator** — wrap an object to add behavior while keeping the same interface, so features stack.
</div>

The creational patterns answered *how objects get made*. **Structural patterns** answer the next question: *how do objects fit together into larger structures?* They are mostly about composition and wrapping — taking objects that already exist and arranging them so the whole is flexible and the parts stay loosely coupled. Three structural patterns come up constantly: **Adapter** makes mismatched things work together, **Facade** hides complexity behind a simple front, and **Decorator** adds behavior by wrapping. As before, the examples lean on a software radio, where you routinely connect hardware drivers, hide DSP chains, and stack processing stages.

## What structural patterns are for

Real systems are assembled from parts written at different times by different people: a third-party hardware driver, your own DSP code, a UI toolkit. Getting those parts to cooperate without welding them tightly together is a recurring design problem. Structural patterns are templates for that assembly. They share a family resemblance — most of them involve one object **wrapping** another — but each solves a distinct problem:

- **Adapter** — the interfaces do not match, and you need them to.
- **Facade** — the subsystem is complex, and you want a simple way in.
- **Decorator** — the object is fine, but you want to add behavior around it.

All three lean on the same principle: program to an **interface**, and keep dependencies pointing at that interface rather than at concrete classes. That keeps [coupling](/learn/intro-software-dev/abstraction-coupling/) low.

## Adapter

An **Adapter** converts the interface of one class into another interface that the client expects, letting classes work together that otherwise could not because of incompatible interfaces. It is the universal travel plug of software.

The classic SDR example is hardware. An RTL-SDR dongle, an Airspy, and a HackRF each ship with their own driver, and the function names and data formats differ: one offers `read_samples()`, another `rxStream()`, another a callback. Your decode pipeline should not care which radio is plugged in. So you define one interface your code talks to, and write a thin adapter per device that translates:

```text
interface SampleSource { read() -> samples }

class RtlSdrAdapter implements SampleSource {
  rtl  // the vendor driver object
  read() {
    raw = rtl.read_samples()      // vendor's call
    return toComplex(raw)         // translate to our format
  }
}

class HackRfAdapter implements SampleSource {
  hrf
  read() { return hrf.rxStream().asComplex() }
}

// pipeline depends only on SampleSource:
process(source: SampleSource) { loop { handle(source.read()) } }
```

Now the pipeline works with any device, present or future. Supporting a new radio means writing one new adapter — the pipeline never changes. The adapter absorbs the differences so the rest of the system sees a single, clean interface.

There are two flavours worth knowing. An **object adapter** holds a reference to the wrapped object (composition) and delegates to it — that is what the example above does, and it is the more flexible choice because one adapter can wrap any object that offers the vendor calls. A **class adapter** instead inherits from both interfaces, which ties it to one concrete class at compile time. In modern code, the object adapter is almost always preferred, because composition keeps the coupling loose and lets you wrap objects you do not control. Either way, the adapter's whole value is that it is the *only* place in the codebase that knows the vendor's quirks: a units mismatch, a different sample format, an awkward callback model — all of it is quarantined behind the clean interface so it cannot leak into the rest of the system.

## Facade

A **Facade** provides a unified, simplified interface to a set of interfaces in a subsystem, making the subsystem easier to use. Where an adapter wraps *one* object to fix compatibility, a facade fronts a *whole* collection of objects to fix complexity.

Decoding a signal involves many steps: tune, set gain, decimate, filter, demodulate, recover the clock, decode symbols. Each is its own component with its own settings — a genuine pipeline (the next lesson and the [demodulation pipeline](/learn/rf-sdr/demodulation-pipeline/) lesson cover that chain in depth). Most callers do not want to orchestrate all of that. A facade collapses it to one call:

```text
class RadioFacade {
  tuneAndDecode(freq, mode) -> stream {
    src   = openDevice()
    tuned = tuner.tune(src, freq)
    base  = decimator.run(filter.run(tuned))
    sym   = demod(mode).run(base)
    return decoder(mode).run(clockRecover(sym))
  }
}

// the UI just says:
calls = radio.tuneAndDecode(854_000_000, "P25")
```

The complexity still exists — the facade does not delete the DSP chain — it just hides it behind one friendly entry point. Two payoffs: callers are simpler, and they are *decoupled* from the subsystem's internals. If you re-order the DSP stages or swap a filter, only the facade changes; every caller keeps working. A facade does not forbid access to the inner parts for the rare caller who needs them — it just offers an easy default path for the common case.

Facades also help when a subsystem is *legacy or messy*. You may not be able to clean up a tangled set of components right away, but you can put a sane facade in front of them, point all new code at the facade, and refactor the mess behind it later without breaking those callers. In that role the facade becomes a stable seam — a boundary you can hold steady while everything behind it changes. This is the same boundary-drawing instinct that runs through the whole [SOLID](/learn/intro-software-dev/solid/) lesson: decide where one part ends and another begins, then let dependencies cross only through a narrow, deliberate interface.

## Decorator

A **Decorator** attaches additional responsibilities to an object dynamically by wrapping it, providing a flexible alternative to subclassing for extending behavior. The decorator implements the *same interface* as the thing it wraps, so from the outside it looks identical — but it does something extra before or after delegating to the inner object.

Suppose you have a `SampleSource` (perhaps via the adapter above) and you want optional extra stages: a logging tap that counts samples, and a gain stage that scales them. You could bake these into the source, but then every source needs them and you cannot turn them off. Instead, wrap:

```text
class GainStage implements SampleSource {
  inner: SampleSource
  factor
  read() { return scale(inner.read(), factor) }   // add behavior, then delegate
}

class LoggingStage implements SampleSource {
  inner: SampleSource
  read() {
    s = inner.read()
    metrics.count(s.length)
    return s
  }
}

// stack them however you like, at runtime:
src = LoggingStage(GainStage(RtlSdrAdapter(rtl), 1.5))
```

Because every layer is a `SampleSource`, they nest in any order and any combination. You can add logging without gain, gain without logging, or both — decided at runtime, not compile time. This is the open/closed principle from [SOLID](/learn/intro-software-dev/solid/) again: you extend behavior by adding wrappers, never by editing the core class. It also avoids a subclass explosion — without decorators you would need `LoggedSource`, `GainSource`, `LoggedGainSource`, and so on for every combination.

A subtle but important point: Decorator looks structurally like Adapter (both wrap an object), but the *intent* differs. Adapter changes an interface to make things fit; Decorator keeps the interface the same and adds behavior. Same shape, different purpose — which is exactly why knowing the pattern *names* matters when you discuss a design.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a facade puts one simple interface in front of a whole complex subsystem." markdown="0">
  <p class="knowledge-check__q">Quick check: Which structural pattern hides a whole complex subsystem behind one simple entry point?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Adapter</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Facade</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Decorator</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Structural patterns compose objects** — they arrange how parts fit together, mostly through wrapping, keeping the whole flexible and loosely coupled.
- **Adapter** — converts one object's interface into the one a client expects; wrap each SDR driver so the pipeline sees a single `SampleSource`.
- **Facade** — a simple front over a complex subsystem; one `tuneAndDecode()` call hides the whole DSP chain and decouples callers from it.
- **Decorator** — wraps an object with the same interface to add behavior (gain, logging) that stacks in any order at runtime, instead of subclassing.
- **Adapter vs Decorator** — same wrapping shape, different intent: Adapter changes the interface, Decorator preserves it and adds behavior.
- **Common thread** — program to interfaces so wrappers and fronts can be swapped without touching the code that uses them.

Next up: behavioral patterns — Observer, Strategy, and State — which govern how objects *interact and share responsibility* at runtime. See [Behavioral patterns](/learn/intro-software-dev/behavioral-patterns/).
