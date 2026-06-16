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
  - { title: "SDR hardware — RTL-SDR, HackRF, Airspy", url: /learn/sdr-hardware/ }
external:
  - { title: "Airspy (Wikipedia)", url: https://en.wikipedia.org/wiki/Software-defined_radio }
---

**Airspy** is a line of high-performance VHF/UHF
[software-defined radio](/reference/software-defined-radio/) receivers (the R2 and the
smaller Mini) offering better sensitivity, dynamic range, and wider
[bandwidth](/reference/bandwidth/) than an [RTL-SDR](/reference/rtl-sdr/).

## Overview

Airspy R2 captures up to ~10 MHz, useful when a system's channels are spread across a
band or in tough RF environments. For the lower bands, the
[Airspy HF+](/reference/airspy-hf-plus/) is the specialised choice.

## Relevance to SDR

GopherTrunk supports Airspy receivers for demanding reception where an RTL-SDR's
bandwidth or sensitivity falls short.
