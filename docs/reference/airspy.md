---
slug: airspy
title: Airspy
entry_type: hardware
category: hardware
description: Airspy is a line of high-performance VHF/UHF software-defined radio receivers (R2 and Mini) offering better sensitivity and wider bandwidth than RTL-SDR.
keywords: Airspy, Airspy R2, Airspy Mini, high performance SDR, VHF UHF receiver
aka: [Airspy]
autolink: true
infobox:
  - { label: Type, value: VHF/UHF SDR receiver }
  - { label: Models, value: Airspy R2, Airspy Mini }
  - { label: Range, value: ~24 MHz – 1.8 GHz }
  - { label: Bandwidth, value: up to ~10 MHz (R2) }
see_also: [rtl-sdr, airspy-hf-plus, hackrf, software-defined-radio]
related_lessons:
  - { title: "SDR hardware — RTL-SDR, HackRF, Airspy", url: /learn/rf-sdr/sdr-hardware/ }
related_reading:
  - { title: "RF Front End, Part 10: Airspy — real to complex", url: /blog/deep-dives/rf-front-end-10-airspy-real-to-complex/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Software-defined_radio
---

**Airspy** is a line of high-performance VHF/UHF
[software-defined radio](/reference/software-defined-radio/) receivers (the R2 and the
smaller Mini) offering better sensitivity, dynamic range, and wider
[bandwidth](/reference/bandwidth/) than an [RTL-SDR](/reference/rtl-sdr/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A frequency coverage bar for Airspy R2/Mini (~24 MHz–1.8 GHz) on an axis from about 0 to 6 gigahertz." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="430" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="30" y="86">0</text><text x="163" y="86">2 GHz</text><text x="296" y="86">4 GHz</text><text x="430" y="86">6 GHz</text></g>
  <rect x="32" y="40" width="120" height="20" rx="3" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.3"/>
  <text x="230" y="28" text-anchor="middle" font-size="10" fill="currentColor">Airspy R2/Mini (~24 MHz–1.8 GHz) coverage</text>
</svg>
<figcaption>Airspy adds sensitivity and bandwidth over RTL-SDR across VHF/UHF.</figcaption>
</figure>

## Overview

Airspy R2 captures up to ~10 MHz, useful when a system's channels are spread across a
band or in tough RF environments. For the lower bands, the
[Airspy HF+](/reference/airspy-hf-plus/) is the specialised choice.

## Relevance to SDR

GopherTrunk supports Airspy receivers for demanding reception where an RTL-SDR's
bandwidth or sensitivity falls short.

## Wideband multi-site monitoring

An Airspy pinned to `role: wideband` can channelize several control channels —
including multiple **sites** of one P25 system — out of a single IQ capture, all
decoded in parallel. List each site's control channel as its own `channels:`
entry (see `config.example.yaml`).

Every tap shares one antenna, one centre frequency and one gain, and the
channelizer is **gain-flat across taps**. So if one site decodes cleanly while
the others sit at the noise floor, the cause is RF, not the DDC:

- **Front-end overload.** A strong (often hilltop) site can drive the shared ADC
  into clipping, raising the noise floor and burying weaker sites. Gain is in
  **tenths of a dB** — `gain: 600` means 60 dB, very high for a wideband capture.
  If `gophertrunk_sdr_wideband_input_clip_ratio` is non-zero (a throttled WARN
  also fires), **lower the gain or add attenuation** — do not raise it.
- **A genuinely weak/distant site** may not survive a capture optimised for a
  stronger one. Give it a dedicated dongle if it matters.

Diagnostics: each tap's level is on `gophertrunk_sdr_iq_power_dbfs` labelled
`<system> @ <freq> MHz`; the whole capture is on
`gophertrunk_sdr_wideband_input_iq_power_dbfs{serial}` and
`gophertrunk_sdr_wideband_input_clip_ratio{serial}`. Compare a tap against the
whole-capture power to tell a weak site apart from a decode problem, and watch
the clip ratio for overload.

## Troubleshooting

### Stream goes silent after a few seconds (macOS)

On macOS the pure-Go USB backend reaps the Airspy's bulk-IN endpoint with
blocking reads. If the device silently halts its endpoint — no USB error, no
disconnect — those reads never return, and older builds wedged with the process
alive but decoding nothing (only the periodic `runtime: heartbeat` kept
logging). A **stall watchdog** now guards this: if the stream delivers no data
for a couple of seconds while the device is still enumerated, GopherTrunk aborts
the pipe, surfaces the death as a real end-of-stream, and the daemon reacquires
the dongle and restarts the wideband decoder automatically.

Knobs for diagnosing and tuning it:

- `RTLSDR_DEBUG_USB=1` — emits a periodic bulk-stream telemetry line (URBs,
  bytes, throughput, per-slot spread, idle gap) plus a one-shot **`bulk-IN
  stalled`** line at the moment the stream freezes. Capture this to pin down
  *when* and *how* a freeze happens.
- `GT_USB_BULK_STALL_MS` — the stall window in milliseconds (default `2000`).
  Set `0` to disable the watchdog.
- `GT_USB_READPIPE_TIMEOUT_MS` — opt-in: switch the reaper to IOKit's
  `ReadPipeTO` with this per-read no-data timeout so a halted endpoint returns a
  timeout directly instead of relying on the watchdog. Off by default; try e.g.
  `200` if the watchdog alone doesn't recover cleanly.

## Sources

[^wiki]: [Software-defined radio](https://en.wikipedia.org/wiki/Software-defined_radio) — Wikipedia, for background on Airspy-class high-performance VHF/UHF SDR receivers.
