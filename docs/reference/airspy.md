---
slug: airspy
title: Airspy
entry_type: hardware
category: sdr-devices
description: "Airspy is a line of high-performance VHF/UHF software-defined radio receivers (R2 and Mini) offering better sensitivity, dynamic range and wider bandwidth than an RTL-SDR."
keywords: Airspy, Airspy R2, Airspy Mini, high performance SDR, VHF UHF receiver, 12-bit ADC, real-to-complex, oversampling, wideband capture
aka: [Airspy]
autolink: true
infobox:
  - { label: Type, value: VHF/UHF SDR receiver }
  - { label: Vendor/Chip, value: "Airspy; R820T2 tuner + LPC4370" }
  - { label: Models, value: Airspy R2, Airspy Mini }
  - { label: ADC, value: 12-bit }
  - { label: Range, value: ~24 MHz – 1.8 GHz }
  - { label: Bandwidth, value: up to ~10 MHz (R2) }
  - { label: TX, value: No (receive only) }
see_also: [rtl-sdr, rtl2832u, r820t-tuner, airspy-hf-plus, hackrf, sdrplay-rsp1a, dynamic-range, software-defined-radio]
related_lessons:
  - { title: "SDR hardware — RTL-SDR, HackRF, Airspy", url: /learn/rf-sdr/sdr-hardware/ }
related_reading:
  - { title: "RF Front End, Part 10: Airspy — real to complex", url: /blog/deep-dives/rf-front-end-10-airspy-real-to-complex/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Software-defined_radio
  - https://airspy.com/airspy-r2/
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
[Airspy HF+](/reference/airspy-hf-plus/) is the specialised choice. Where an
[RTL-SDR](/reference/rtl-sdr/) is deliberately the cheapest thing that works, an Airspy
is what you reach for when the RTL-SDR's 8-bit ADC or ~2.4 MHz of usable bandwidth is
the thing standing between you and a decode.

## How it works

An Airspy R2 shares the front-end tuner with an RTL-SDR — a Rafael Micro
[R820T2](/reference/r820t-tuner/) — but replaces everything behind it. Instead of the
[RTL2832U](/reference/rtl2832u/)'s 8-bit ADC and USB bridge, the Airspy digitises the
tuner's IF with a **12-bit** [ADC](/reference/analog-to-digital-converter/) driven by an
NXP LPC4370 microcontroller, then streams the samples over USB 2.0. Two design choices
give it its edge:

- **More bits.** A 12-bit ADC carries roughly **72 dB** of theoretical
  [dynamic range](/reference/dynamic-range/) against the RTL2832U's ~48 dB — the
  headroom that lets a weak signal survive next to a strong one without the front end
  being pushed into clipping.
- **Oversampling and real-to-complex conversion.** The Airspy samples the IF at a high
  real rate (e.g. 20 MS/s) and digitally converts it to complex baseband on the way
  out, decimating to the requested rate. Averaging many high-rate samples into each
  output sample adds effective bits (process gain), so the delivered stream is quieter
  than the raw ADC alone.

The R2 delivers up to about **10 MS/s** of complex [bandwidth](/reference/bandwidth/);
the smaller **Mini** tops out around 6 MS/s.[^airspy] Both are **receive-only** — there
is no transmit path. The [HydraSDR RFOne](/reference/hydrasdr/) is an independent successor
to the R2 built on the same 12-bit R820T2 architecture.

## Variants

- **Airspy R2** — the flagship VHF/UHF receiver, ~24 MHz–1.8 GHz, up to ~10 MS/s, with a
  clock output for chaining and a 4.5 V [bias tee](/reference/bias-tee/) for powering an
  inline LNA.
- **Airspy Mini** — the same architecture in a dongle form factor, up to ~6 MS/s;
  cheaper and more portable, with slightly less capture width.
- **[Airspy HF+](/reference/airspy-hf-plus/)** — a different design entirely, optimised
  for HF and low-VHF with very high dynamic range rather than wide VHF/UHF coverage.

Compared with a [HackRF One](/reference/hackrf/), an Airspy trades the HackRF's 6 GHz
reach and transmit capability for a quieter, higher-resolution receive path in the bands
scanners actually use. An [SDRplay RSP1A](/reference/sdrplay-rsp1a/) is the closest
14-bit alternative in the same niche.

## Relevance to SDR

GopherTrunk supports Airspy receivers for demanding reception where an RTL-SDR's
bandwidth or sensitivity falls short — a congested band, a distant control channel, or a
multi-site system whose control channels are spread too far apart to fit one RTL-SDR
capture. The extra ADC bits and wider capture are exactly what a wideband, multi-tap
channelizer wants, which is why the Airspy is GopherTrunk's recommended device for the
`role: wideband` use below. GopherTrunk drives it over USB with a pure-Go backend (no
`libairspy` needed); it remains a receiver only, so it decodes clear and scrambled
traffic, never keyed encryption.

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
  also fires), **lower the gain or add attenuation** — do not raise it. In a metro
  area the usual culprit is broadcast FM; an [FM broadcast notch
  filter](/reference/fm-broadcast-filter/) inline is often the cheapest fix.
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

### Stream aborts with `usb: ReadPipe: 0xe00002eb` (macOS)

A raw `usb: ReadPipe: 0xe00002eb` (or `ReadPipeAsync: 0xe00002eb`) in the `IQ
stream died` cause line is macOS's `kIOReturnAborted` — the host controller
aborting the bulk-IN pipe mid-stream. It is distinct from the two failures above:
the stall watchdog reports `bulk-IN stream stalled` and an overrun reports
`dropping live IQ chunks`; this is neither. It is **not** a bandwidth problem — it
reproduces at 2.5 MS/s (~10 MB/s, well under USB 2.0) just as at 10 MS/s, always
after a second or two of healthy decoding.

The cause was the transport model. The macOS backend used to issue 32 *concurrent
synchronous* `ReadPipe` calls on a single pipe, which is not a supported IOUSBLib
usage and macOS aborts intermittently under sustained streaming, at any rate.
GopherTrunk now defaults to an **asynchronous** bulk-IN path (`ReadPipeAsync`
serviced by one CFRunLoop thread, re-arming each transfer on completion) — the
same model libusb (and hence SDR#/SDRtrunk) use, which stream this hardware
cleanly.

- `GT_USB_SYNC_BULK=1` forces the legacy synchronous reapers (the pre-async path)
  if you need to compare behaviour.
- If the abort persists on the async path, capture a run with `RTLSDR_DEBUG_USB=1`
  and open an issue — that points at something host- or hardware-specific rather
  than the transfer model.

> **10 MS/s note.** If the daemon warns that the capture is *oversampled for the
> channel plan* (`sdr.sample_rate` far wider than the carriers span), a lower
> rate cuts DSP + USB load and overrun pressure for no loss of coverage. A run
> at 10 MS/s that logs `dropping live IQ chunks; consumer can't keep up` is
> shedding samples on the host — that is an overrun (degraded decode), *not* the
> reaper death above, which the `IQ stream died` cause line identifies
> separately.

## Sources

[^wiki]: [Software-defined radio](https://en.wikipedia.org/wiki/Software-defined_radio) — Wikipedia, for background on Airspy-class high-performance VHF/UHF SDR receivers.
[^airspy]: [Airspy R2](https://airspy.com/airspy-r2/) — Airspy, on the R2's R820T2 front end, 12-bit oversampling architecture and up-to-10 MS/s capture.
