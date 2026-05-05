# GopherTrunk Architecture

GopherTrunk is a headless, low-latency trunking-radio engine that manages a
pool of RTL-SDR dongles and decodes P25, DMR, and NXDN trunked systems. The
engine is structured as a set of pipelined goroutines connected by typed
channels, with a registry-based driver model so that mock IQ files and real
hardware are interchangeable.

## Layered overview

```
              ┌────────────────────────────────────────────┐
              │  cmd/gophertrunk  ── daemon + sdr list CLI │
              └───────────────┬────────────────────────────┘
                              │
       ┌──────────────────────┼──────────────────────────┐
       │                      │                          │
┌──────▼──────┐        ┌──────▼──────┐            ┌──────▼──────┐
│  internal/  │        │  internal/  │            │  internal/  │
│   config    │        │     log     │            │   events    │
└─────────────┘        └─────────────┘            └─────────────┘

┌──────────────────────────────────────────────────────────────┐
│  internal/sdr (Phase 1)                                      │
│    Driver registry → rtlsdr (CGO), mock (file replay)        │
│    Pool: enumerates, opens, role-assigns, supervises         │
└──────────────────────────────────────────────────────────────┘
                              │
                              ▼ chan []complex64
┌──────────────────────────────────────────────────────────────┐
│  internal/dsp (Phase 2)  filters · channelizer · demod · sync│
└──────────────────────────────────────────────────────────────┘
                              │
                              ▼ symbol streams
┌──────────────────────────────────────────────────────────────┐
│  internal/radio (Phases 3–5)  framing · p25 · dmr · nxdn     │
└──────────────────────────────────────────────────────────────┘
                              │
                              ▼ control-channel events
┌──────────────────────────────────────────────────────────────┐
│  internal/trunking (Phase 6) engine · grant · priority · site│
└──────────────────────────────────────────────────────────────┘
                              │
                              ▼ events.Bus
┌──────────────────────────────────────────────────────────────┐
│  internal/voice (Phase 7) recorder · imbe · mbelib (-tags)   │
│  internal/api    (Phase 8) gRPC server · WebSocket bridge    │
│  internal/storage (Phase 9) SQLite call log · audio files    │
│  internal/metrics (Phase 10) Prometheus exporter             │
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

`internal/sdr` maintains a process-global registry. Each backend (CGO
librtlsdr, file-replay mock, future HackRF/Airspy) calls `sdr.Register` from
its `init()` so the binary's import set chooses what hardware it can talk
to. `cmd/gophertrunk` blank-imports the drivers it ships with.

## Build tags

- *(default)* — links librtlsdr, no AMBE+2 vocoder.
- `-tags mbelib` — links libmbe via CGO for AMBE+2 (P25 P2 / DMR / NXDN
  voice). Off by default for distribution-license clarity.

See `docs/phases.md` for the phased build roadmap and `docs/hardware.md`
for the hardware setup checklist.
