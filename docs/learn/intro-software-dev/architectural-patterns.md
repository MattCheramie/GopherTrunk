---
slug: architectural-patterns
title: "Architecture patterns: layered, event-driven & plugins"
description: Architecture patterns shape whole systems. Learn layered, client-server, event-driven, MVC, and plugin (microkernel) architecture — and monolith vs microservices.
keywords: architecture patterns, layered architecture, client server, event-driven, microkernel, plugin architecture, MVC, monolith, microservices, software architecture
level: advanced
status: full
faq:
  - q: "What is the difference between a design pattern and an architecture pattern?"
    a: "A **design pattern** solves a local problem inside a program — how a few classes are created, composed, or coordinated. An **architecture pattern** describes the shape of a *whole system* — how its major parts are divided and how they communicate. Layered, event-driven, and plugin architectures are about overall structure; Factory or Observer are about pieces within it."
  - q: "What is a plugin (microkernel) architecture?"
    a: "A **plugin** or **microkernel** architecture has a small, stable core that provides the essential framework, plus separate plugin modules that add features through a defined extension interface. The core knows nothing about specific plugins. A scanner engine uses this so a new protocol decoder can be added as a plugin without modifying the core — a direct application of the open/closed principle."
  - q: "Should I choose a monolith or microservices?"
    a: "A **monolith** is one deployable unit — simpler to build, test, and run, and the right default for most projects. **Microservices** split the system into independently deployable services, which helps large teams scale and deploy parts separately but adds network, operational, and data-consistency complexity. Start with a well-structured monolith; move to services only when real scaling or team pressures justify the overhead."
---
# Architecture patterns: layered, event-driven & plugins

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Layered** — stack UI, logic, and data so each layer depends only on the one below. **Event-driven** — components communicate through events rather than direct calls, for loose coupling. **Plugin (microkernel)** — a small core plus pluggable modules, so new features (like a protocol decoder) drop in without touching the core.
</div>

The patterns so far operated *inside* a program — how classes are made, composed, coordinated, and streamed. **Architecture patterns** zoom all the way out: they describe the shape of a *whole system* — its major parts and how they communicate. Choosing an architecture is one of the earliest and most consequential decisions in a project, because it sets the boundaries everything else lives within. This lesson surveys the big shapes — layered, client-server, event-driven, plugin/microkernel, and MVC — and the monolith-versus-microservices question. Several map cleanly onto how a scanner engine like GopherTrunk is built; you can see the real thing on the [architecture](/architecture.html) page.

## Layered architecture

A **layered** (or *n-tier*) architecture organizes the system into horizontal layers, each with a clear responsibility, where each layer depends only on the layer directly beneath it. The classic three:

| Layer | Responsibility | Example |
|-------|----------------|---------|
| **Presentation (UI)** | Display information and take user input | The web dashboard, the CLI |
| **Logic (business)** | The rules and processing | Trunk-following logic, talkgroup filtering |
| **Data** | Storage and retrieval | The call database, config files |

The rule that makes it work is *directional dependency*: the UI calls the logic, the logic calls the data, and nothing calls upward. Because each layer talks only to its neighbour through a defined interface, you can replace one (swap a web UI for a CLI, swap one database for another) without disturbing the rest. This is the system-scale version of the [abstraction and coupling](/learn/intro-software-dev/abstraction-coupling/) ideas: clear boundaries, dependencies pointing one way. The cost is that a request often passes through every layer, which adds some ceremony for trivial operations — but the structure keeps a growing codebase comprehensible.

## Client-server

In a **client-server** architecture, work is split between **clients** that request services and a **server** that provides them, communicating over a network. The server centralizes shared resources and logic; many clients connect to it. The web is the giant example: browsers (clients) request pages from web servers.

A networked scanner fits naturally. A headless engine runs on a machine near the antenna, doing the capture and decoding; browser clients connect from anywhere to view live calls and change settings. Centralizing the heavy work on the server means clients can be thin and numerous. The trade-offs are the usual networked ones: you must handle latency, connection loss, authentication, and the server becoming a bottleneck or single point of failure. Client-server often combines with the layered style — the server itself is internally layered.

## Event-driven architecture

An **event-driven** architecture has components communicate by producing and reacting to **events** rather than calling each other directly. A component announces that something happened ("call decoded," "signal lost"); other components that care react. An event bus or broker typically routes events between producers and consumers.

This is the Observer and publish/subscribe idea (from the [behavioral patterns](/learn/intro-software-dev/behavioral-patterns/) and [concurrency](/learn/intro-software-dev/concurrency-and-pipelines/) lessons) promoted to a whole-system principle. Its great strength is **loose coupling**: an event producer does not know or care which components consume its events, so you can add new reactions — a new logger, a notifier, an analytics module — without modifying the producer. That makes the system easy to extend and naturally suited to asynchronous, real-time work like a stream of decoded calls. The cost is that flow of control is harder to follow: there is no single call stack to trace, so debugging "who reacted to what, in what order" takes more effort and good tooling.

## Plugin (microkernel) architecture

A **plugin** architecture — also called **microkernel** — separates a minimal **core** from a set of **plugin** modules that extend it through a defined interface. The core provides only the essential, stable framework and knows *nothing* about any specific plugin. Plugins register themselves and add features.

This is the pattern that matters most for a scanner engine. The core handles the universal machinery — capturing samples, managing channels, routing decoded output — and each **protocol decoder is a plugin** conforming to a common decoder interface:

```text
interface ProtocolPlugin {
  name() -> string
  canHandle(signal) -> bool
  decode(samples) -> calls
}

// the core discovers and registers plugins, knowing only the interface:
core.register(P25Plugin())
core.register(DmrPlugin())
core.register(NxdnPlugin())
```

Adding support for a new protocol means writing one new plugin and registering it — **the core is never modified**. That is the [open/closed principle](/learn/intro-software-dev/solid/) at the architectural scale: the system is open to extension (new plugins) but closed to modification (the core stays put). It also keeps responsibilities cleanly separated and lets third parties extend the system without access to its internals. Notice how the smaller patterns reappear here: a [factory](/learn/intro-software-dev/creational-patterns/) chooses which plugin to instantiate, and an [adapter](/learn/intro-software-dev/structural-patterns/) can wrap a quirky decoder to fit the plugin interface. Architecture is patterns composed at the largest scale.

## Model-View-Controller

**Model-View-Controller (MVC)** is a pattern for structuring the parts of a system that have a user interface, by splitting responsibilities three ways:

- **Model** — the data and the rules that govern it (the calls, talkgroups, system state).
- **View** — how that data is presented to the user (the dashboard, the waterfall display).
- **Controller** — handles user input and mediates between view and model.

Separating these means the same model can drive different views (a web view and a CLI view), and you can change how something looks without touching the logic. MVC and its relatives (MVVM, MVP) are the standard organizing principle inside the presentation layer of countless applications, and they pair naturally with event-driven communication — the model publishes a change, and the views, as observers, update.

## Monolith versus microservices

A final, much-debated axis is *how many deployable pieces* the system is.

- A **monolith** is a single deployable application — all the layers and modules ship and run together. It is simpler to develop, test, debug, and deploy, and it is the right default for most projects, including a well-structured scanner engine.
- **Microservices** split the system into many small, independently deployable services that communicate over the network. This lets large teams develop and deploy parts separately and scale hot components independently — but it adds substantial complexity: network calls, distributed data, partial failures, deployment orchestration, and harder debugging.

The honest guidance is to **start with a clean monolith** and adopt microservices only when concrete pressures — team size, independent scaling, separate release cadences — outweigh the operational overhead. A monolith built with clear layers and plugin boundaries can be split later if it must; a premature mesh of services is far harder to undo. Choosing among all of these is exactly the kind of trade-off the [decision framework](/learn/intro-software-dev/decision-framework/) lesson exists to weigh, and you can see one real set of choices on GopherTrunk's [architecture](/architecture.html) page.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a plugin/microkernel core stays unchanged while new decoders are added as plugins through a defined interface." markdown="0">
  <p class="knowledge-check__q">Quick check: Which architecture lets a scanner add a new protocol decoder without modifying its core?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Client-server architecture</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Plugin (microkernel) architecture</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Layered architecture</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Architecture patterns shape whole systems** — they define the major parts and how they communicate, unlike design patterns that work inside a program.
- **Layered** — UI, logic, and data layers each depend only on the one below, so any layer can be replaced behind its interface.
- **Client-server** — clients request, a server provides; centralizes shared work (a headless engine serving browser clients) at the cost of network concerns.
- **Event-driven** — components react to events instead of calling each other, giving loose coupling and easy extension at the cost of harder-to-trace control flow.
- **Plugin (microkernel)** — a small stable core plus pluggable modules; new protocol decoders drop in without touching the core (open/closed at scale).
- **MVC and monolith vs microservices** — MVC separates model, view, and controller in UI code; prefer a clean monolith and adopt microservices only when real scaling or team pressures justify the overhead.

Next up: Module 5 zooms out to the whole development process — how teams plan, build, and ship software. See [The software development lifecycle](/learn/intro-software-dev/sdlc-and-methodologies/).
