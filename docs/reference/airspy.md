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

When a reaper dies, the daemon's `IQ stream died; retrying` log line now names
the **concrete USB cause** — e.g. `... closed unexpectedly: usb: bulk-IN stream
stalled ...` (the stall watchdog fired) versus `usb: device disconnected` (an
unplug) versus a wrapped per-URB error. That distinction tells a genuinely
stalling endpoint apart from a disconnect or an overrun without needing any env
var. The wideband/control retry loops also **self-heal indefinitely** across a
dongle that keeps recovering — a stream that dies, reacquires, streams for a few
seconds and dies again no longer accumulates to a process-killing fatal; only a
device that re-dies immediately on every reopen (truly gone) still escalates.

Knobs for diagnosing and tuning it:

- `RTLSDR_DEBUG_USB=1` — emits a periodic bulk-stream telemetry line (URBs,
  bytes, throughput, per-slot spread, idle gap) plus a one-shot **`bulk-IN
  stalled`** line at the moment the stream freezes. It now **also traces the
  Airspy's vendor control transfers** (`SET_SAMPLERATE` / `SET_FREQ` /
  `RECEIVER_MODE` / gain) — previously only the RTL-SDR driver wrapped its
  transport for this, so Airspy control setup was invisible. Capture this to pin
  down *when* and *how* a freeze happens.
- `GT_USB_BULK_STALL_MS` — the stall window in milliseconds (default `2000`).
  Set `0` to disable the watchdog.
- `GT_USB_READPIPE_TIMEOUT_MS` — opt-in: switch the reaper to IOKit's
  `ReadPipeTO` with this per-read no-data timeout so a halted endpoint returns a
  timeout directly instead of relying on the watchdog. Off by default; try e.g.
  `200` if the watchdog alone doesn't recover cleanly.

### Stream aborts at 10 MS/s (macOS): `usb: ReadPipe: 0xe00002eb`

A raw `usb: ReadPipe: 0xe00002eb` in the `IQ stream died` cause line is macOS's
`kIOReturnAborted` — the host controller aborting the bulk-IN pipe. It is a
distinct failure from the two above: the stall watchdog reports `bulk-IN stream
stalled` and an overrun reports `dropping live IQ chunks`; this is neither. At
10 MS/s the Airspy streams ~40 MB/s (the real ADC at 2× the IQ rate), near the
USB 2.0 ceiling, and the host must keep bulk transfers continuously outstanding
or the controller aborts.

GopherTrunk now runs the real→IQ conversion on a dedicated goroutine rather than
inline on the USB reapers, so a reaper re-posts its next read immediately and the
outstanding-transfer queue no longer collapses under the conversion (the earlier
cause of this abort on macOS). If you still hit `ReadPipe: 0xe00002eb` at 10 MS/s,
the host genuinely can't sustain the rate on that machine/port — the surest fix is
a lower `sdr.sample_rate` (2.5 MS/s on an R2), which usually covers the channel
plan anyway (watch for the *oversampled for the channel plan* warning below).

> **10 MS/s note.** If the daemon warns that the capture is *oversampled for the
> channel plan* (`sdr.sample_rate` far wider than the carriers span), a lower
> rate cuts DSP + USB load and overrun pressure for no loss of coverage. A run
> at 10 MS/s that logs `dropping live IQ chunks; consumer can't keep up` is
> shedding samples on the host — that is an overrun (degraded decode), *not* the
> reaper death above, which the `IQ stream died` cause line identifies
> separately.

## Sources

[^wiki]: [Software-defined radio](https://en.wikipedia.org/wiki/Software-defined_radio) — Wikipedia, for background on Airspy-class high-performance VHF/UHF SDR receivers.
