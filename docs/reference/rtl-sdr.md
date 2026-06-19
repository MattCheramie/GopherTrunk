---
slug: rtl-sdr
title: RTL-SDR
entry_type: hardware
category: hardware
description: RTL-SDR is a family of low-cost USB software-defined radio receivers based on the RTL2832U chip, repurposed from DVB-T TV tuners, covering roughly 24 MHz to 1.7 GHz.
keywords: RTL-SDR, RTL2832U, cheap SDR, DVB-T dongle, R820T, 24 MHz 1.7 GHz, receive only
aka: [RTL-SDR, RTL SDR]
autolink: true
infobox:
  - { label: Type, value: USB SDR receiver }
  - { label: Bridge chip, value: RTL2832U }
  - { label: Range, value: ~24 MHz – 1.7 GHz }
  - { label: Bandwidth, value: ~2.4 MHz usable }
  - { label: TX, value: No (receive only) }
see_also: [rtl2832u, r820t-tuner, hackrf, airspy, upconverter, software-defined-radio]
related_lessons:
  - { title: "SDR hardware — RTL-SDR, HackRF, Airspy", url: /learn/rf-sdr/sdr-hardware/ }
related_reading:
  - { title: "RF Front End, Part 7: RTL-SDR / RTL2832U bring-up", url: /blog/deep-dives/rf-front-end-07-rtlsdr-rtl2832u-bringup/ }
external:
  - { title: "GopherTrunk hardware guide", url: /hardware.html }
cite_urls:
  - https://en.wikipedia.org/wiki/Software-defined_radio#RTL-SDR
---

**RTL-SDR** is a family of inexpensive USB [software-defined radio](/reference/software-defined-radio/)
receivers built around the [RTL2832U](/reference/rtl2832u/) chip — originally a DVB-T TV
tuner that hobbyists discovered could stream raw [IQ](/reference/iq-data/) samples.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A frequency coverage bar for RTL-SDR (~24 MHz–1.7 GHz) on an axis from about 0 to 6 gigahertz." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="430" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="30" y="86">0</text><text x="163" y="86">2 GHz</text><text x="296" y="86">4 GHz</text><text x="430" y="86">6 GHz</text></g>
  <rect x="32" y="40" width="113" height="20" rx="3" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.3"/>
  <text x="230" y="28" text-anchor="middle" font-size="10" fill="currentColor">RTL-SDR (~24 MHz–1.7 GHz) coverage</text>
</svg>
<figcaption>RTL-SDR covers most VHF/UHF scanning at low cost — receive only.</figcaption>
</figure>

## Overview

A typical RTL-SDR costs around $30, tunes roughly **24 MHz–1.7 GHz**, and captures about
2.4 MHz of [bandwidth](/reference/bandwidth/). It is receive-only with modest dynamic
range, but more than enough to follow most VHF/UHF trunked systems.

## Relevance to SDR

The RTL-SDR is the ideal entry point and the baseline GopherTrunk targets. For HF, add an
[upconverter](/reference/upconverter/) or use an [Airspy HF+](/reference/airspy-hf-plus/);
see the [hardware guide](/hardware.html).

## Sources

[^wiki]: [RTL-SDR](https://en.wikipedia.org/wiki/Software-defined_radio#RTL-SDR) — Wikipedia, on the DVB-T-dongle origins and capabilities of RTL-SDR.
