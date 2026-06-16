---
slug: hackrf
title: HackRF One
entry_type: hardware
category: hardware
description: HackRF One is an open-source wideband half-duplex software-defined radio transceiver covering 1 MHz to 6 GHz with up to 20 MHz of bandwidth and transmit capability.
keywords: HackRF One, Great Scott Gadgets, wideband SDR, transceiver, 1 MHz 6 GHz, transmit
aka: [HackRF, HackRF One]
autolink: true
infobox:
  - { label: Type, value: Wideband SDR transceiver }
  - { label: Vendor, value: Great Scott Gadgets }
  - { label: Range, value: 1 MHz – 6 GHz }
  - { label: Bandwidth, value: up to ~20 MHz }
  - { label: TX, value: Yes (half-duplex) }
see_also: [rtl-sdr, airspy, software-defined-radio]
related_lessons:
  - { title: "SDR hardware — RTL-SDR, HackRF, Airspy", url: /learn/sdr-hardware/ }
external:
  - { title: "HackRF (Wikipedia)", url: https://en.wikipedia.org/wiki/HackRF_One }
---

**HackRF One** is an open-source, wideband, half-duplex
[software-defined radio](/reference/software-defined-radio/) transceiver from Great Scott
Gadgets, covering **1 MHz to 6 GHz** with up to ~20 MHz bandwidth and the ability to
**transmit**.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A frequency coverage bar for HackRF One (~1 MHz–6 GHz) on an axis from about 0 to 6 gigahertz." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="430" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="30" y="86">0</text><text x="163" y="86">2 GHz</text><text x="296" y="86">4 GHz</text><text x="430" y="86">6 GHz</text></g>
  <rect x="30" y="40" width="400" height="20" rx="3" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.3"/>
  <text x="230" y="28" text-anchor="middle" font-size="10" fill="currentColor">HackRF One (~1 MHz–6 GHz) coverage</text>
</svg>
<figcaption>HackRF spans a huge range and can transmit, but with lower dynamic range — overkill for scanning.</figcaption>
</figure>

## Overview

Its huge range and TX capability make it popular for experimentation, but it uses 8-bit
sampling (less dynamic range than [Airspy](/reference/airspy/)) and transmit is
irrelevant to receive-only scanning.

## Relevance to SDR

For decoding trunked voice, HackRF is overkill; an [RTL-SDR](/reference/rtl-sdr/) or
Airspy is usually the better fit, but GopherTrunk can use it as a receiver.
