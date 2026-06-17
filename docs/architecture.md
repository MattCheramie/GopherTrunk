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
│    Driver registry → rtlsdr · hackrf · airspy · airspyhf     │
│    (all pure-Go, shared USB transport at rtlsdr/usb);        │
│    baseband (WAV IQ replay as virtual tuners); mock          │
│    (raw u8/f32 file replay)                                  │
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

## Streaming, recording & telemetry subsystems

Several subsystems hang off the `events.Bus` as independent
subscribers — they never block the decode path, and each is
constructed only when its config section is present:

- **`internal/broadcast`** — outbound call streaming. A `Manager`
  subscribes to `KindCallComplete` (published by the recorder once a
  call's WAV is flushed), encodes the audio to MP3 via the pure-Go
  `internal/voice/mp3` package, and fans it out to the configured
  backends (`broadcastify`, `rdioscanner`, `openmhz`, `icecast`)
  with bounded exponential-backoff retry. Per-feed `systems` filters
  and the per-talkgroup `Stream` flag gate delivery. Counters are at
  `GET /api/v1/broadcast`.
- **`internal/sdr/baseband`** — wideband IQ capture. A
  `RecordingDevice` decorates an `sdr.Device` and tees its IQ stream
  to a two-channel 16-bit WAV; a `FileDriver` registers with the SDR
  driver registry so recorded WAVs mount as virtual tuners. Wired
  through the `baseband:` config section.
- **`internal/radio/location`** + the `location_log` table — a
  strict NMEA-0183 parser feeds `KindLocation` events that
  `storage.LocationLog` persists and `GET /api/v1/locations` serves.
- **`trunking.AffiliationTracker`** — a protocol-agnostic, in-memory
  table of unit→talkgroup activity built from `KindGrant`,
  `KindAffiliation` and `KindUnitRegistration` events, served at
  `GET /api/v1/affiliations`.
- **`trunking.SiteTracker`** — an in-memory table of the P25 sites
  discovered from the control channel, built from `KindSiteUpdate`
  events (published on each RFSS Status Broadcast) and served at
  `GET /api/v1/sites`. Operator-supplied site names from
  `trunking.systems[].sites` in the config are merged onto the
  discovered rows, and every `grant` event also carries `rfss_id` /
  `site_id` so downstream tooling can label calls by site (issue #698).
- **`log.MessageLog`** — a rotating, human-readable text log of
  every trunking event, enabled via `log.message_log`.

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
calls `sdr.Register` from its `init()` so the binary's import set
chooses what hardware it can talk to:

- `internal/sdr/rtlsdr/purego` (`rtlsdr`) — pure-Go RTL2832U +
  every osmocom tuner.
- `internal/sdr/hackrf` (`hackrf`) — pure-Go libhackrf protocol
  port covering HackRF One / Jawbreaker / Rad1o, with BOARD_ID and
  VERSION_STRING readback at open so the operator-visible model
  name and firmware version (including PortaPack / Mayhem
  detection) come from the device itself.
- `internal/sdr/airspy` (`airspy`) — pure-Go libairspy protocol
  port covering Airspy R2 and Airspy Mini, with the R2-vs-Mini
  split surfaced through the `TunerName` field.
- `internal/sdr/airspyhf` (`airspyhf`) — pure-Go libairspyhf
  protocol port covering the Airspy HF+ family (Discovery, Dual
  Port, legacy HF+). HF (9 kHz – 31 MHz) + VHF (60 – 260 MHz);
  HF AGC plus a 0–48 dB attenuator and +6 dB LNA preamp.
- `internal/sdr/baseband` (`baseband-replay`) — mounts recorded
  IQ WAVs as virtual tuners. Registered dynamically by the daemon
  when `baseband.replay` is configured.

`cmd/gophertrunk` blank-imports the four real-hardware drivers
unconditionally; the baseband-replay driver registers at runtime
when the operator points at a recording.

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
