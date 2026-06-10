---
layout: page
title: What is GopherTrunk?
description: A concise tour of GopherTrunk and its interfaces and parts
nav_group: Getting Started
---

# What is GopherTrunk?

**GopherTrunk is a software-defined-radio (SDR) scanner that follows digital
trunked-radio voice calls and decodes them to audio.** It is written in pure
Go with **zero CGO** — no `librtlsdr`, `libhackrf`, `libusb`, `libasound2`, or
`libmp3lame` at build or runtime — and ships as a single ~10 MB static binary
for Linux, macOS, and Windows. Drop the binary on a box with a dongle, point a
YAML file at a control channel, and it starts recording calls.

This page is the conceptual map: what GopherTrunk decodes, the interfaces you
drive it with, and how the parts fit together. When you're ready to run it,
jump to **[Get started](getting-started-setup.html)**.

## Supported hardware

GopherTrunk drives a *pool* of SDR dongles — one engine, many radios:

- **RTL-SDR** — every osmocom-compatible tuner (the cheapest way to start).
- **HackRF** — One / Jawbreaker / Rad1o.
- **Airspy** — R2 / Mini, and **Airspy HF+**.
- **Remote backends** — `rtl_tcp` and SoapyRemote, so the radio can live on
  another host.

Full device matrix, antenna notes, and gain tips are in the
[Hardware guide](hardware.html).

## What it decodes

- **Trunked voice** — P25 Phase 1 + Phase 2, DMR Tier II + Tier III, NXDN,
  Motorola Type II / SmartZone, EDACS / GE-Marc, LTR, MPT 1327, dPMR Mode 3,
  TETRA TMO; amateur D-STAR and Yaesu System Fusion.
- **Paging & signaling** — POCSAG + FLEX paging, MDC1200.
- **Position & safety** — APRS / AX.25 packet, AIS marine, ADS-B aircraft,
  DSC marine distress.

A live shipping-vs-pending checklist per protocol lives in
[Status](status.html).

## The interfaces (the parts you drive)

GopherTrunk is **one binary and one config file**. The *daemon* is the engine;
everything else is a view onto the same running daemon.

| Part | What it is | Reach it with |
|------|------------|---------------|
| **Headless daemon** | The engine — owns the SDR pool, decoders, recorder, and APIs. Runs silent (systemd, Docker, Windows service). | `gophertrunk -headless` |
| **TUI cockpit** | In-process [Bubbletea](tui.html) terminal console with 12 panels (dashboard, active calls, scanner, history, …). | `gophertrunk -tui` |
| **Web console** | Bundled browser SPA served by the daemon — the same panels in a browser. | `gophertrunk -web` |
| **Interactive launcher** | What plain `gophertrunk` shows on a TTY: pick **[1] TUI**, **[2] Web**, or **[3] Headless**. | `gophertrunk` |
| **Config Builder** | Standalone guided editor for `config.yaml` — add systems, talkgroups, and feeds without hand-writing YAML. Comes as both a terminal app and a browser app. | `gophertrunk config` / `config serve` |
| **SigLab** | Standalone signal workbench for offline captures — replay, analyze, identify, and grade decodes. Also a terminal app and a browser console. | `gophertrunk siglab` / `siglab serve` |

The first four surfaces are *views onto a running daemon*: quitting the TUI or
closing the browser does **not** stop the daemon — it keeps running in the same
process until you send `Ctrl-C` / `SIGTERM` (see the [Launcher](launcher.html)).
**Config Builder** and **SigLab** are standalone — they run on their own,
without a live daemon, each available as a terminal interface and a browser
interface.

## APIs & integrations

- **HTTP REST + SSE + WebSocket** on `127.0.0.1:8080` (configurable) — drives
  the web console and any custom tooling.
- **gRPC** on `127.0.0.1:50051` for typed clients.
- **Prometheus `/metrics`** for scraping.
- **Hamlib `rigctld`** wire protocol for amateur-radio tooling — see
  [rigctld integration](rigctld.html).
- **Outbound call streaming** to Broadcastify Calls, RdioScanner, OpenMHz, and
  live Icecast / ShoutCast mountpoints, out of the box.

## Offline & analysis tooling

Beyond the live daemon, the same binary carries a workbench of subcommands for
captures and discovery:

- `gophertrunk capture` — record live SDR IQ to a file.
- `gophertrunk replay` / `analyze` / `identify` — decode and inspect a capture
  offline, auto-detect its protocol, export JSON/YAML/CSV.
- `gophertrunk hunt` — discover and map an unknown trunked system
  ([Hunt](hunt.html)).
- `gophertrunk import-pdf` — pull a RadioReference PDF straight into your config
  ([Import](import.html)).

The **SigLab** and **Config Builder** interfaces above wrap this workbench in
terminal and browser front-ends.

## How the parts fit together

One daemon owns the radios and the decoders. The TUI, the web console, and the
REST/gRPC APIs are all just *views and controls* over that one running daemon —
which is why you can start headless and attach a UI later, or run the TUI
locally while a browser on your phone watches the same calls. Edits made in a
UI flow back to `config.yaml` and many apply without a restart
([Live edits](live-edits.html)).

For the full design — the multi-consumer IQ fan-out, the DSP chain, the voice
pipeline — read the [Architecture](architecture.html) reference.

---

**Ready to run it? → [Get started](getting-started-setup.html)**

Once it's running, the guides continue the tour from your first call all the way
to the APIs: [Basics](guide-basics.html) →
[Intermediate](guide-intermediate.html) → [Advanced](guide-advanced.html).
