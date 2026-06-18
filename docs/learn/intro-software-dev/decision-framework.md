---
slug: decision-framework
title: A practical decision framework
description: A repeatable five-step method for choosing a language — eliminate on hard constraints, weight soft criteria, score a short list, prototype the risky part, then commit — with worked examples.
keywords: choosing a programming language, decision framework, language selection, hard constraints, weighted criteria, prototype, SDR scanner, Go, worked examples
level: advanced
status: full
faq:
  - q: "How do I actually decide between two close languages?"
    a: "Stop debating on paper and prototype the riskiest part in each. Build the one piece you're least sure about — the hot DSP loop, the concurrency model, the deployment story — and measure. A half-day spike usually settles an argument that a week of discussion won't, because it tests reality instead of opinion."
  - q: "Why is Go a good fit for an SDR scanner engine, but not a slam dunk?"
    a: "Go fits the *plumbing* — easy concurrency for capture and decode pipelines, fast-enough garbage-collected performance, and a single static binary anyone can download and run. The caveat is the hot DSP loop: C, C++ or Rust give more raw throughput and tighter timing. Go is strong overall, but you weigh that throughput trade-off honestly."
  - q: "Should I always pick the highest-scoring language from the framework?"
    a: "Treat the score as a guide, not a verdict. It organises the trade-offs and exposes your assumptions, but a small score gap can be overturned by team familiarity, a critical library or a deployment requirement. Use it to narrow to a short list and force a prototype, then commit with judgement."
---
# A practical decision framework

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Eliminate, then weigh, then prototype** — hard constraints cut the field, soft criteria rank the rest, a spike tests reality. **Score to think, not to obey** — the number organises trade-offs, judgement decides. **Then commit** — a chosen-and-shipped language beats a perfect one still being debated.
</div>

You've now got the pieces — what the choice decides, how to read requirements, the
core trade-off, the languages and the domains. This lesson assembles them into a
method you can actually run, the same way each time. It's deliberately simple: five
steps that move from facts (which languages are even possible?) to judgement (which
should we commit to?). We then run it on three concrete examples, including
GopherTrunk's own situation.

## The five steps

1. **List the hard constraints, and eliminate.** These are pass/fail requirements — a
   real-time latency budget, a target platform, a memory ceiling, a licence. Anything
   that fails one is out. This is the cheapest, highest-leverage step: it often cuts
   the field from a dozen to three.
2. **Weight the soft criteria.** For what survives, list what *matters and by how much* —
   team familiarity, ecosystem, raw performance, deployment ease, hiring, maintainability.
   Assign weights honestly *before* you score, so you can't fudge them to favour a
   pre-chosen winner.
3. **Score a short list.** Rate each surviving language against each weighted criterion.
   The numbers won't be precise, and that's fine — the point is to make the trade-offs
   explicit and visible, not to compute a winner to three decimal places.
4. **Prototype the risky part.** Don't decide on paper. Build the single thing you're
   *least* sure about in your top one or two candidates — the hot loop, the concurrency
   model, the build-and-ship step — and measure. A half-day spike beats a week of
   argument.
5. **Decide and commit.** Pick, write down *why* (so future-you remembers the
   reasoning), and stop relitigating. Indecision is itself a cost.

```text
constraints → eliminate → weight → score short list → prototype risky part → commit
   (facts)                                                              (judgement)
```

## Worked example 1: an internal CRUD web app

A small team needs an internal tool — forms, a database, some reports, a few hundred
users.

- **Hard constraints:** runs on the company's Linux servers; finished in six weeks.
  These eliminate almost nothing — every mainstream language qualifies.
- **Weights:** team familiarity (high), ecosystem (medium), performance (low — it's
  I/O-bound and small).
- **Score:** the team already knows **Python** and uses Django elsewhere. Go and Java
  would work fine but offer no advantage that matters here.
- **Prototype:** unnecessary — no part is risky.
- **Decide:** Python. The lesson: when nothing's at the extremes, **familiarity and
  ecosystem win**, exactly as the first lesson promised.

## Worked example 2: firmware for a sensor

A battery-powered microcontroller must read a sensor and report over radio, with
kilobytes of RAM.

- **Hard constraints:** runs on a tiny MCU; minimal memory; predictable timing; no
  garbage collector pauses. These eliminate Python, JavaScript, Java, Go — anything
  that needs a heavy runtime or GC.
- **Weights:** footprint (critical), timing predictability (critical), safety (high),
  team experience (medium).
- **Score:** **C** (ubiquitous on MCUs, total control, but unsafe) vs **Rust**
  (same control, memory-safe, steeper ramp and thinner embedded ecosystem).
- **Prototype:** flash a minimal sensor-read loop in both; check binary size, timing
  and toolchain pain.
- **Decide:** C if the team's deep in it and the ecosystem's there; Rust if safety is
  paramount and they can absorb the curve. The lesson: **hard constraints did most of
  the work** — by step one the field was down to two.

## Worked example 3: GopherTrunk's SDR scanner engine

Now the real one. The task: an engine that captures samples from an SDR device,
runs trunk-tracking and decode pipelines across many channels at once, and ships to
non-technical users who just want to download and run it on Linux, macOS or Windows.
See [GopherTrunk's architecture](/architecture.html) and
[hardware support](/hardware.html) for the full picture.

- **Hard constraints:** real-time-ish stream handling without dropping samples;
  cross-platform; **easy for non-technical users to install and run**. That last one
  matters more than it looks.
- **Weights:** concurrency (high — many channels and clients), deployment simplicity
  (high — users aren't developers), raw DSP throughput (medium), team familiarity
  (medium), safety (medium).
- **Score the short list:**

| Criterion | Go | C++ | Rust | Python |
|---|---|---|---|---|
| Concurrency for capture/decode pipelines | strong | manual, error-prone | strong but harder | GIL hampers CPU threads |
| Deployment to non-technical users | one static binary | per-platform build/link pain | single binary, harder build | needs runtime + deps |
| Raw DSP throughput (hot loop) | good, not the fastest | fastest | fastest, safe | slow unless delegated |
| Learning curve / iteration speed | gentle, fast compiles | steep, slow builds | steep | gentle |
| Memory safety | GC, safe-ish | unsafe by default | safe, no GC | safe |

**Why Go is a strong fit here.** The engine is mostly *concurrent coordination* —
pulling a sample stream, fanning it to many decoders, juggling channels and network
clients. Go's goroutines and channels express that cleanly. It's garbage-collected
but **fast enough** for this coordination work, compiles in seconds, and — the
clincher for GopherTrunk's audience — **cross-compiles to a single static binary** so
a non-technical user downloads one file and runs it on Linux, macOS or Windows, no
runtime, no dependency hunt. That deployment story is a genuine product advantage
when your users aren't programmers.

**Where Go costs you, honestly.** For the *tightest DSP inner loops*, **C, C++ and
Rust deliver more raw throughput and tighter timing control** — no GC pauses, full
control over memory layout and vectorisation. If a particular demodulator becomes the
bottleneck, Go's garbage collector and somewhat lower numeric ceiling are real
trade-offs. The mitigations are the mixed-language patterns from the previous lesson:
keep Go for the concurrent plumbing and, where a hot loop demands it, drop to C via
cgo or a native library for that piece. This is a *trade-off*, not a free win — Go is
chosen here because the balance of concurrency, deployment and "good enough" speed
fits *this* product, not because it beats C on raw DSP. A different SDR project with a
brutal single-channel throughput requirement might well land on C++ or Rust instead.

- **Prototype:** build the capture-to-decode pipeline for a couple of channels in Go,
  measure whether it keeps up with the sample rate on target hardware, and confirm the
  cross-compiled binary runs cleanly on all three OSes.
- **Decide:** Go for the engine, with the door open to native code for any proven hot
  loop — and write down that reasoning.

The packaging side of "one downloadable binary" is its own topic — see
[packaging and distribution](/learn/intro-software-dev/packaging-and-distribution/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — the first step is eliminating languages that fail a hard, pass/fail constraint; it cheaply cuts the field before any scoring." markdown="0">
  <p class="knowledge-check__q">Quick check: what's the first and highest-leverage step of the framework?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Score every popular language against weighted criteria</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Eliminate any language that fails a hard, pass/fail constraint</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Pick whichever language the team likes most and move on</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Using the framework well

A few habits keep it honest:

- **Set weights before scoring.** Otherwise you'll quietly tune them to justify a
  decision you already made.
- **Treat the score as a thinking aid, not a verdict.** A narrow gap can be overturned
  by a critical library, the team's skills or a deployment need.
- **Always prototype the risk.** The framework's real power is forcing you to test the
  scary part for real instead of arguing about it.
- **Commit and record why.** Future maintainers — including you — will thank you for a
  paragraph explaining the choice. And remember the first lesson: switching later is
  costly, so commit deliberately, then build.

## Recap

- **Five steps** — eliminate on hard constraints, weight soft criteria, score a short
  list, prototype the risky part, decide and commit.
- **Hard constraints do the heavy lifting** — they cheaply cut the field before any
  scoring.
- **Score to think, not to obey** — the number organises trade-offs; judgement,
  ecosystem and team can override a close call.
- **Prototype the risk** — a short spike settles arguments that paper debate won't.
- **GopherTrunk's engine fits Go** — easy concurrency, fast-enough GC'd performance and
  a single cross-platform binary for non-technical users — while C/C++/Rust still win
  raw DSP throughput, an honest trade-off, not a free lunch.

Next up: putting a chosen language to work as a one-person team — the
[solo developer mindset](/learn/intro-software-dev/solo-developer-mindset/) that opens
Module 7.
