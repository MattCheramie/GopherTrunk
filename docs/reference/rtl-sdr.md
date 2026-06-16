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
  - { title: "SDR hardware — RTL-SDR, HackRF, Airspy", url: /learn/sdr-hardware/ }
external:
  - { title: "RTL-SDR (Wikipedia)", url: https://en.wikipedia.org/wiki/Software-defined_radio#RTL-SDR }
  - { title: "GopherTrunk hardware guide", url: /hardware.html }
---

**RTL-SDR** is a family of inexpensive USB [software-defined radio](/reference/software-defined-radio/)
receivers built around the [RTL2832U](/reference/rtl2832u/) chip — originally a DVB-T TV
tuner that hobbyists discovered could stream raw [IQ](/reference/iq-data/) samples.

## Overview

A typical RTL-SDR costs around $30, tunes roughly **24 MHz–1.7 GHz**, and captures about
2.4 MHz of [bandwidth](/reference/bandwidth/). It is receive-only with modest dynamic
range, but more than enough to follow most VHF/UHF trunked systems.

## Relevance to SDR

The RTL-SDR is the ideal entry point and the baseline GopherTrunk targets. For HF, add an
[upconverter](/reference/upconverter/) or use an [Airspy HF+](/reference/airspy-hf-plus/);
see the [hardware guide](/hardware.html).
