---
layout: page
title: Architecture
description: Pipelined goroutines, typed channels, and the registry-based driver model behind GopherTrunk
nav_group: Reference
---

# GopherTrunk Architecture

GopherTrunk is a headless, low-latency trunking-radio engine that manages a
pool of RTL-SDR dongles and decodes every major trunked-radio family (P25
Phase 1 / Phase 2, DMR Tier II / III, NXDN, Motorola Type II, EDACS /
GE-Marc, LTR, MPT 1327, dPMR Mode 3, TETRA TMO) plus the D-STAR + Yaesu
System Fusion amateur modes. The engine is structured as a set of pipelined
goroutines connected by typed channels, with a registry-based driver model
so that mock IQ files and real hardware are interchangeable. A multi-system
scanner subsystem and an analog FM conventional scanner sit on top so the
daemon behaves like a high-end digital-trunking police scanner end-to-end.

## Layered overview

```
              ┌────────────────────────────────────────────┐
              │  cmd/gophertrunk  ── daemon + sdr list CLI │
              │                  ── TUI cockpit (10 panels)│
              └───────────────┬────────────────────────────┘
                              │
       ┌──────────────────────┼──────────────────────────┐
       │                      │                          │
┌──────▼──────┐        ┌──────▼──────┐            ┌──────▼──────┐
│  internal/  │        │  internal/  │            │  internal/  │
│   config    │        │     log     │            │   events    │
└─────────────┘        └─────────────┘            └─────────────┘

┌──────────────────────────────────────────────────────────────┐
│  internal/sdr                                                │
│    Driver registry → rtlsdr (pure-Go), mock (file replay)    │
│    Pool: enumerates, opens, role-assigns, supervises;        │
│    publishes sdr.attached/sdr.detached events with per-      │
│    device SDRStatus payloads                                 │
└──────────────────────────────────────────────────────────────┘
                              │
                              ▼ chan []complex64
┌──────────────────────────────────────────────────────────────┐
│  internal/dsp           filters · channelizer · demod · sync │
│                         · equalizer · diversity · fft        │
└──────────────────────────────────────────────────────────────┘
                              │
                              ▼ symbol streams
┌──────────────────────────────────────────────────────────────┐
│  internal/radio         framing · p25/{phase1,phase2} ·      │
│                         dmr/{tier2,tier3} · nxdn · ysf ·     │
│                         dstar · dpmr · edacs · ltr ·         │
│                         motorola · mpt1327 · tetra           │
└──────────────────────────────────────────────────────────────┘
                              │
                              ▼ control-channel events
┌──────────────────────────────────────────────────────────────┐
│  internal/trunking      engine · grant · priority · site ·   │
│                         ScanMode · HandleSyntheticCall ·     │
│                         cc cache · Hunter primitive          │
└──────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────┐
│  internal/scanner       cchunt/ (multi-system CC supervisor) │
│                         conventional/ (FM scan list w/ IQ-   │
│                         power squelch + hop-on-silence)      │
└──────────────────────────────────────────────────────────────┘
                              │
                              ▼ events.Bus
┌──────────────────────────────────────────────────────────────┐
│  internal/voice         recorder · composer · vocoder plugin │
│                         · imbe (pure-Go) · ambe2 (pure-Go) · │
│                         mbe (shared MBE synthesis) · toneout │
│  internal/api           HTTP/SSE/WebSocket + gRPC servers    │
│                         (mutations gated by allow_mutations) │
│  internal/storage       SQLite call log · retention sweeper  │
│  internal/metrics       Prometheus exporter                  │
│  internal/tui           bubbletea cockpit (10 panels) over   │
│                         REST + SSE                           │
└──────────────────────────────────────────────────────────────┘
```

## Concurrency model

- Each opened device owns one async-read goroutine that pushes
  `[]complex64` chunks (~6 ms each at 2.4 MS/s) onto a buffered channel.
- DSP stages compose as channels-in / channels-out. Each stage runs in its
  own goroutine; back-pressure flows naturally through buffered channels.
- The trunking engine consumes parsed control-channel events and emits
  domain events onto an in-process pub/sub bus (`internal/events`).
- API surfaces (gRPC, WebSocket) subscribe to the bus; they never call into
  the engine directly. This keeps the engine API-agnostic and the API
  testable in isolation.

## Driver registry

`internal/sdr` maintains a process-global registry. Each backend
(pure-Go RTL-SDR under `internal/sdr/rtlsdr/purego`, file-replay
mock, future HackRF/Airspy) calls `sdr.Register` from its `init()`
so the binary's import set chooses what hardware it can talk to.
`cmd/gophertrunk` blank-imports the drivers it ships with.

## Build tags

- *(default)* — fully pure-Go (`CGO_ENABLED=0`). Pure-Go RTL-SDR
  driver, pure-Go IMBE (`internal/voice/imbe`), and pure-Go
  AMBE+2 (`internal/voice/ambe2`).
- `-tags integration` — enables the wired end-to-end daemon test
  under `cmd/gophertrunk` (no real SDR; synthetic call on the bus).
- `-tags dvsi` — *planned* — links a DVSI USB-3000 / AMBE-3003
  hardware backend through the same `Vocoder` interface.

See `docs/hardware.md` for the hardware setup checklist,
`docs/hardening.md` for the operations playbook, and
`docs/vocoders.md` for the IMBE / AMBE+2 licensing situation.
