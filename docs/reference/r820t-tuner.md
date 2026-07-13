---
slug: r820t-tuner
title: R820T / R820T2 tuner
entry_type: hardware
category: sdr-devices
description: The Rafael Micro R820T and R820T2 are the most common tuner chips paired with the RTL2832U in RTL-SDR dongles, providing the RF front-end and mixer up to ~1.7 GHz.
keywords: R820T, R820T2, R828D, Rafael Micro, tuner chip, RTL-SDR front end, mixer
aka: [R820T, R820T2]
autolink: true
infobox:
  - { label: Type, value: RF tuner IC }
  - { label: Vendor, value: Rafael Micro }
  - { label: Role, value: Front-end + mixer/LO }
  - { label: Range, value: ~24 MHz – 1.7 GHz }
see_also: [rtl-sdr, rtl2832u, superheterodyne-receiver, local-oscillator]
related_lessons:
  - { title: "How an SDR receiver works", url: /learn/rf-sdr/sdr-receiver/ }
related_reading:
  - { title: "RF Front End, Part 7: RTL-SDR / RTL2832U bring-up", url: /blog/deep-dives/rf-front-end-07-rtlsdr-rtl2832u-bringup/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Software-defined_radio#RTL-SDR
---

The **R820T** and improved **R820T2** (and related R828D) from Rafael Micro are the most
common tuner chips paired with the [RTL2832U](/reference/rtl2832u/) in
[RTL-SDR](/reference/rtl-sdr/) dongles. They provide the RF front-end and
mixer/[local oscillator](/reference/local-oscillator/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="The tuner's role: RF in, mixed with a local oscillator, low-IF out to the ADC." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="35" y="55">RF in</text>
    <rect x="80" y="36" width="120" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="140" y="50">R820T/R820T2</text><text x="140" y="62" font-size="7.5">LNA · mixer · LO</text>
    <rect x="240" y="38" width="80" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="280" y="57">to ADC</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="60" y1="53" x2="79" y2="53"/><line x1="200" y1="53" x2="239" y2="53"/></g>
  </g>
</svg>
<figcaption>The R820T/R820T2 is the most common RTL-SDR tuner chip — it amplifies and mixes RF down for the ADC.</figcaption>
</figure>

## Overview

The tuner amplifies and shifts the selected band down to a low frequency the RTL2832U
can digitise, covering roughly 24 MHz–1.7 GHz. Tuner quality affects sensitivity and
overload behaviour.

## Relevance to SDR

The tuner sets the dongle's frequency range and much of its noise performance, important
when chasing weak signals.

## Sources

[^wiki]: [RTL-SDR](https://en.wikipedia.org/wiki/Software-defined_radio#RTL-SDR) — Wikipedia, on the R820T/R820T2 tuners commonly paired with the RTL2832U.
