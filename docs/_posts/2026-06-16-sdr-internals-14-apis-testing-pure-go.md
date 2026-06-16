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
