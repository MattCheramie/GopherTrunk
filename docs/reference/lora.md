---
slug: lora
title: LoRa
entry_type: protocol
category: wireless-data-iot
description: LoRa is a low-power, long-range modulation that encodes data as chirps (frequency sweeps), giving excellent sensitivity for IoT telemetry in ISM bands.
keywords: LoRa, chirp spread spectrum, CSS, LoRaWAN, IoT, long range, ISM band, dechirp
aka: [LoRa, "chirp spread spectrum"]
autolink: true
see_also: [frequency-shift-keying, bandwidth, signal-to-noise-ratio]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/rf-sdr/other-signals/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/LoRa
---

**LoRa** is a low-power, long-range modulation that encodes data as **chirps** —
frequency sweeps that ramp across the channel. Chirp spread spectrum gives LoRa
excellent sensitivity (it decodes well below the [noise floor](/reference/noise-floor/)),
making it popular for low-rate IoT telemetry in ISM bands.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A spectrogram showing repeated rising frequency sweeps (chirps), the signature of LoRa." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="20" width="380" height="80" fill="none" stroke="currentColor" stroke-opacity="0.4"/>
  <g stroke="currentColor" stroke-width="1.8" fill="none"><path d="M50 95 L110 25"/><path d="M120 95 L180 25"/><path d="M190 95 L250 25"/><path d="M260 95 L320 25"/><path d="M330 95 L390 25"/></g>
  <text x="30" y="60" font-size="8" fill="currentColor" transform="rotate(-90 30 60)">freq</text>
  <text x="230" y="114" text-anchor="middle" font-size="8.5" fill="currentColor">time → · each ramp is one chirp symbol</text>
</svg>
<figcaption>LoRa encodes symbols as rising frequency chirps; the diagonal sweeps are its waterfall signature.</figcaption>
</figure>

## Overview

A receiver "de-chirps" by multiplying with a reference sweep, collapsing each chirp to a
tone whose frequency encodes the symbol. LoRa is unrelated to the trunked-voice systems
GopherTrunk targets, but is a common sight on the [waterfall](/reference/fast-fourier-transform/)
in the 433/868/915 MHz ISM bands.

## Sources

[^wiki]: [LoRa](https://en.wikipedia.org/wiki/LoRa) — Wikipedia, for the chirp-spread-spectrum modulation, its sensitivity, and ISM-band use.
