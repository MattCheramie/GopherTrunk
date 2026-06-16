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

## Overview

Its huge range and TX capability make it popular for experimentation, but it uses 8-bit
sampling (less dynamic range than [Airspy](/reference/airspy/)) and transmit is
irrelevant to receive-only scanning.

## Relevance to SDR

For decoding trunked voice, HackRF is overkill; an [RTL-SDR](/reference/rtl-sdr/) or
Airspy is usually the better fit, but GopherTrunk can use it as a receiver.
