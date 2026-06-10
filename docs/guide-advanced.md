---
layout: page
title: Advanced
description: The full GopherTrunk surface — the TUI cockpit, signal scopes, offline analysis, every receiver, remote SDRs, and the APIs
nav_group: Getting Started
---

# Advanced & power-user features

This is the rest of GopherTrunk — the parts you reach for once you're running a
serious setup or want to integrate it with other tooling. The same single binary
that's been recording your calls also carries a terminal cockpit, a bench of
signal scopes, an offline analysis workbench, a dozen more decoders, and typed
APIs.

> **Before you start.** This guide assumes [Going further](guide-intermediate.html)
> — comfort with `config.yaml`, the terminal, and a multi-dongle pool. It's
> comfortable with command-line subcommands and a little signal-processing
> vocabulary.

## The terminal cockpit and signal scopes

The web console isn't the only front end. The **[TUI](tui.html)** is a
full-screen terminal cockpit with the same panels — ideal over SSH or on a
headless box. It also exposes GopherTrunk's **signal scopes**, the tools for
seeing *why* a signal does or doesn't decode:

- **[Constellation](constellation.html)** — the IQ scatter plot; tight clusters
  mean a clean signal.
- **[Eye diagram](eye-diagram.html)** and **[Symbol scope](symbol-scope.html)** —
  watch the demodulated symbols and how open the "eye" is.
- **[Plots](plots.html)**, **[Symbol histogram](histogram.html)**, and
  **[Tuning meters](tuning.html)** — spectrum, symbol distribution, and
  receiver health at a glance.

Use these to chase down a marginal site: too much gain, a frequency-error
offset, or a simulcast signal that needs a different demodulator.

## Offline analysis: SigLab and the workbench

Beyond the live daemon, the binary carries a workbench of subcommands for
working with **recorded captures** instead of a live radio:

- `gophertrunk capture` records raw IQ off a live dongle to a file.
- `gophertrunk replay` / `analyze` / `identify` decode a capture offline,
  auto-detect its protocol, and export the results to JSON / YAML / CSV.
- `gophertrunk test` grades a decode against expected results.

**SigLab** wraps this workbench in a terminal and a browser console
(`gophertrunk siglab` / `siglab serve`) so you can replay, inspect, and grade
decodes without a radio attached — the natural home for diagnosing a tricky
capture or building a regression fixture. The
[Architecture reference](architecture.html) describes the decode pipeline these
tools drive.

## Many more receivers

Trunked voice is just the start. GopherTrunk also decodes, each as an opt-in
receiver with its own log and console panel:

- **[ADS-B](adsb.html)** — aircraft transponders, with a live map.
- **[AIS](ais.html)** — marine vessel positions.
- **[APRS / AX.25](aprs.html)** — amateur packet, beacons, and messages.
- **[DSC](dsc.html)** — marine digital selective calling, including distress.
- **[MDC1200](mdc1200.html)** — Motorola in-band signaling (PTT IDs, emergency).
- **[M17](m17.html)** — the open digital-voice link layer.

Which of these are on by default, and how to enable the rest, lives in
[Opt-in features](opt-in-features.html).

## Remote and recorded sources

Your dongle doesn't have to be on the same machine — or even a physical radio:

- **Remote SDRs** — drive a dongle on another host over `rtl_tcp`, or
  professional hardware (USRP, LimeSDR, bladeRF, SDRplay) over SoapyRemote.
- **Baseband IQ record / replay** — capture the wideband IQ stream to disk and
  later mount the recording as a *virtual* dongle, so you can re-run decodes
  against the exact same RF.

## APIs and integrations

The daemon is built to be driven by other tools. Everything the consoles do runs
over public APIs:

- **HTTP REST + Server-Sent Events + WebSocket** — the same API the web console
  uses; point your own scripts and dashboards at it.
- **gRPC** — typed clients for calls, talkgroups, and radio IDs.
- **Prometheus `/metrics`** — scrape GopherTrunk into your monitoring stack.
- **[rigctld integration](rigctld.html)** — speak the Hamlib wire protocol so
  amateur-radio logging tools (Cloudlog, GridTracker, satellite trackers) can
  read and set frequency.

## Squeeze out every decode

When you're chasing the last few percent of a difficult system:

- **[Vocoders](vocoders.html)** and **[Voice calibration](voice-calibration.html)**
  — how GopherTrunk turns digital voice frames into audio, and how to tune it.
- **[DMR encryption](dmr-encryption.html)** — supply keys to decode protected
  DMR traffic you're authorised to monitor.
- **FEC opt-outs** — per-protocol error-correction toggles, documented in
  [Opt-in features](opt-in-features.html), for matching pre-stripped captures.

## You now have the complete picture

From "what is this?" to driving the APIs, you've seen the whole of GopherTrunk:
the conceptual map, a first call, everyday operation, a richer setup, and the
power-user surface. Where to from here:

- **[Architecture](architecture.html)** — the full design: the multi-consumer IQ
  fan-out, the DSP chain, the voice pipeline.
- **[Status](status.html)** and **[Roadmap](roadmap.html)** — what ships today
  and what's coming.
- **[Support](support.html)** — where to get help and how to report issues.

← Back to the start: [What is GopherTrunk?](getting-started.html)
