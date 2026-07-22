---
title: "SDR in Pure Go, Part 14: APIs, Testing & the Pure-Go Story"
description: How GopherTrunk exposes the engine through gRPC, REST, WebSocket and a TUI, observes it with Prometheus and SQLite, and proves it works with synthesized-IQ integration tests — a ports-and-adapters finale.
category: deep-dives
tags: [sdr, go, api, testing, architecture, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "SDR Internals"
series_part: 14
---

*Part 14 of **SDR Internals**, the finale. We've gone from RF to audio. Now: how
the engine is exposed to the outside world, how it's observed, and how the whole
pure-Go stack is proven correct without any hardware.*

## In this post

- The **output surfaces**: gRPC, REST, WebSocket/SSE, the TUI, the web console,
  Prometheus metrics, and SQLite storage.
- The **ports-and-adapters** design that lets all of them coexist.
- The **synthesized-IQ integration tests** that exercise the real code path —
  and the single static binary that ships it.

## What the integration layer does

The engine produces a stream of events; everything users actually touch is a
*consumer* of that stream:

- **`internal/api`** — gRPC (typed, streaming), REST/JSON, and WebSocket/SSE for
  live data (spectrum, constellation, call feed). Mutations are gated behind auth;
  read endpoints are public by default.
- **`internal/tui`** — a Bubbletea cockpit with ~10 panels, driven entirely over
  REST + SSE.
- **`internal/metrics`** — a Prometheus exporter (`/metrics`).
- **`internal/storage`** — a SQLite call log and per-protocol message logs.
- **`web/`** — a React/TypeScript single-page console.

Every one of them subscribes to the event bus from
[Part 11]({{ '/blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/' | relative_url }}).
None of them calls into the engine.

<figure class="lab-figure">
<svg viewBox="0 0 660 190" width="660" height="190" role="img" aria-label="Ports-and-adapters output surface: the engine core and its event bus sit on the left, and five adapters — internal/api for gRPC, REST and WebSocket; internal/tui; internal/metrics; internal/storage; and the web single-page console — each subscribe to the bus, while the engine imports none of them.">
  <rect x="8" y="68" width="140" height="52" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="78" y="90" text-anchor="middle" fill="var(--accent)" font-size="11">engine core</text>
  <text x="78" y="106" text-anchor="middle" fill="var(--fg-muted)" font-size="9">event bus (Part 11)</text>
  <line x1="148" y1="94" x2="290" y2="22" stroke="currentColor"/><polygon points="286,18 296,20 290,29" fill="currentColor"/>
  <line x1="148" y1="94" x2="290" y2="58" stroke="currentColor"/><polygon points="286,54 296,56 289,65" fill="currentColor"/>
  <line x1="148" y1="94" x2="290" y2="94" stroke="currentColor"/><polygon points="290,90 300,94 290,98" fill="currentColor"/>
  <line x1="148" y1="94" x2="290" y2="130" stroke="currentColor"/><polygon points="289,123 296,132 286,134" fill="currentColor"/>
  <line x1="148" y1="94" x2="290" y2="166" stroke="currentColor"/><polygon points="288,159 296,168 284,168" fill="currentColor"/>
  <rect x="300" y="8" width="340" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="470" y="26" text-anchor="middle" fill="currentColor" font-size="10">internal/api · gRPC / REST / WebSocket · SSE</text>
  <rect x="300" y="44" width="340" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="470" y="62" text-anchor="middle" fill="currentColor" font-size="10">internal/tui · Bubbletea cockpit</text>
  <rect x="300" y="80" width="340" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="470" y="98" text-anchor="middle" fill="currentColor" font-size="10">internal/metrics · Prometheus /metrics</text>
  <rect x="300" y="116" width="340" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="470" y="134" text-anchor="middle" fill="currentColor" font-size="10">internal/storage · SQLite call log</text>
  <rect x="300" y="152" width="340" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="470" y="170" text-anchor="middle" fill="currentColor" font-size="10">web/ · React SPA console</text>
</svg>
<figcaption>Every output is an adapter over one event stream. Adapters depend on the core through the bus and the core depends on none of them, so any surface can be added, removed, or tested alone.</figcaption>
</figure>

## How GopherTrunk implements it in Go

This is **ports and adapters** (a.k.a. hexagonal architecture): the engine is the
core; each output is an *adapter* that translates the core's events into a
particular protocol — gRPC messages, JSON, SSE frames, terminal cells, Prometheus
counters, SQL rows. Because adapters depend on the core (via the bus) and never
the reverse, you can add, remove, or test any of them in isolation.

That same design is what makes the **test strategy** possible. The mock SDR driver
from
[Part 2]({{ '/blog/deep-dives/sdr-internals-02-sdr-device-driver-registry/' | relative_url }})
plus the modulators in `internal/dsp/demod` let a test **synthesize spec-correct
IQ**, register it as a device, boot a full daemon, and assert the entire chain —
demod → receiver → FEC → engine → bus → API/metrics — recovers the signal:

```go
// cmd/gophertrunk integration test (shape)
dibits := buildNXDNSpecEncodedDibits(...)
iq := demod.ModulateC4FM(dibits, sps, span, alpha, sampleRateHz, deviationHz)
// write IQ to a temp file, register it as a mock SDR, boot the daemon,
// then assert KindCCLocked appears on the bus and metrics agree.
```

These run under `-tags integration` so they're separate from fast unit tests,
which are table-driven (the DSP and FEC suites inject known inputs and assert
exact outputs).

<figure class="lab-figure">
<svg viewBox="0 0 680 130" width="680" height="130" role="img" aria-label="The synthesized-IQ integration test path: a modulator and mock SDR are the only fake — the sample source — feeding real demodulator, receiver and FEC code, then the engine and event bus, ending in assertions that KindCCLocked appears and the metrics agree.">
  <rect x="8" y="42" width="152" height="52" rx="6" fill="none" stroke="var(--accent)" stroke-dasharray="4 3"/>
  <text x="84" y="64" text-anchor="middle" fill="var(--accent)" font-size="10">synth IQ · mock SDR</text>
  <text x="84" y="80" text-anchor="middle" fill="var(--fg-muted)" font-size="9">the only fake</text>
  <line x1="160" y1="68" x2="180" y2="68" stroke="currentColor"/><polygon points="180,64 190,68 180,72" fill="currentColor"/>
  <rect x="190" y="42" width="172" height="52" rx="6" fill="none" stroke="currentColor"/>
  <text x="276" y="64" text-anchor="middle" fill="currentColor" font-size="10">demod → receiver → FEC</text>
  <text x="276" y="80" text-anchor="middle" fill="var(--fg-muted)" font-size="9">production code</text>
  <line x1="362" y1="68" x2="382" y2="68" stroke="currentColor"/><polygon points="382,64 392,68 382,72" fill="currentColor"/>
  <rect x="392" y="42" width="120" height="52" rx="6" fill="none" stroke="currentColor"/>
  <text x="452" y="64" text-anchor="middle" fill="currentColor" font-size="10">engine + bus</text>
  <text x="452" y="80" text-anchor="middle" fill="var(--fg-muted)" font-size="9">real path</text>
  <line x1="512" y1="68" x2="532" y2="68" stroke="currentColor"/><polygon points="532,64 542,68 532,72" fill="currentColor"/>
  <rect x="542" y="42" width="130" height="52" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="607" y="64" text-anchor="middle" fill="var(--accent)" font-size="10">assert CCLocked</text>
  <text x="607" y="80" text-anchor="middle" fill="var(--fg-muted)" font-size="9">+ metrics agree</text>
</svg>
<figcaption>Under <code>-tags integration</code>, only the sample source is faked: the test synthesizes spec-correct IQ and runs the genuine DSP, protocol, engine, and bus code end to end.</figcaption>
</figure>

## The design principle: ports & adapters + testability

The whole architecture exists to serve two goals at once: **let many interfaces
consume the engine** and **let the engine be tested without any of them.** Ports
and adapters delivers both — and pure Go makes the result shippable as one file.

### How that principle shaped the Go code

- **The core never imports an adapter.** `internal/api`, `internal/tui`, and
  `internal/storage` import the engine's events; the engine imports none of them.
  That one-way dependency, set in
  [Part 1]({{ '/blog/deep-dives/sdr-internals-01-what-is-software-defined-radio/' | relative_url }}),
  holds all the way to the edges.
- **Tests drive the real path, not mocks of it.** Instead of mocking the decoder,
  the integration suite synthesizes real IQ and runs the actual DSP and protocol
  code. The only fake is the *source* of samples — everything downstream is
  production code.
- **Build tags separate concerns.** `-tags integration` gates the wired daemon
  tests; `-tags dvsi` would link a hardware vocoder. The default build stays pure
  Go, `CGO_ENABLED=0`.
- **One binary, every surface.** Because there's no CGO, `go build` produces a
  single static binary — daemon, CLI, TUI, and embedded web console — that
  cross-compiles to Linux, macOS, and Windows. Simple interfaces and the registry
  pattern mean fakes need no framework; a struct with the right methods is enough.

## Where this goes next — and a series recap

That's the full pipeline: **RF → IQ → DSP → symbols → FEC → protocol → events →
audio → output**, every block pure Go, every block built on a clear software-design
principle:

- Layered architecture, the registry and driver model, CSP concurrency
  (Parts 1–3)
- Stateful zero-allocation DSP, the Strategy-pattern channelizer,
  single-responsibility demodulators, feedback timing loops, decorator
  equalizers (Parts 4–8)
- Pure-function FEC, adapter-based protocol decoders, the event-driven engine,
  plugin vocoders, config-driven recording/streaming, and ports-and-adapters
  output (Parts 9–14)

Each of these is a doorway to a deeper series. From here, the per-component deep
dives — pure-Go SDR drivers, DSP math, individual protocols, and digital voice —
pick up where this overview leaves off. Start the journey again from
[Part 1]({{ '/blog/deep-dives/sdr-internals-01-what-is-software-defined-radio/' | relative_url }}),
or grab a build and watch the pipeline run for real.

## FAQ

**Why expose gRPC, REST, and WebSocket all at once?**
Different clients want different things — gRPC for typed streaming integrations,
REST for simple queries, WebSocket/SSE for live spectrum and call feeds. As
adapters over one event stream, they add surface without adding coupling.

**How can you test a radio with no radio?**
By synthesizing the IQ a real signal would produce, registering it as a mock SDR,
and running the genuine DSP/protocol code against it. The test controls the input
samples; everything else is production code.

**What does "pure Go" buy the end user?**
One static binary with no shared-library dependencies to install. `go build`
cross-compiles it for Linux, macOS, and Windows, and the same code is readable
from antenna to audio without leaving the language.

## Series navigation

**Part 14 of 14** · ←
[Part 13]({{ '/blog/deep-dives/sdr-internals-13-recording-composition-streaming/' | relative_url }})
· Back to
[Part 1: What is software-defined radio?]({{ '/blog/deep-dives/sdr-internals-01-what-is-software-defined-radio/' | relative_url }})
