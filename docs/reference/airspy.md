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

## Sources

[^wiki]: [Software-defined radio](https://en.wikipedia.org/wiki/Software-defined_radio) — Wikipedia, for background on Airspy-class high-performance VHF/UHF SDR receivers.
