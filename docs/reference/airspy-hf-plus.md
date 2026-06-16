---
slug: airspy-hf-plus
title: Airspy HF+
entry_type: hardware
category: hardware
description: Airspy HF+ is a software-defined radio optimised for the HF and low-VHF bands, with excellent dynamic range for receiving shortwave and weak low-band signals.
keywords: Airspy HF+, HF SDR, shortwave receiver, dynamic range, Discovery, low VHF
aka: [Airspy HF+, Airspy HF plus]
autolink: true
infobox:
  - { label: Type, value: HF / low-VHF SDR receiver }
  - { label: Strength, value: High dynamic range on low bands }
  - { label: Range, value: ~9 kHz – 31 MHz, 60–260 MHz }
see_also: [airspy, rtl-sdr, upconverter, ionospheric-propagation, frequency-bands]
related_lessons:
  - { title: "SDR hardware — RTL-SDR, HackRF, Airspy", url: /learn/sdr-hardware/ }
external:
  - { title: "Airspy (Wikipedia)", url: https://en.wikipedia.org/wiki/Software-defined_radio }
---

**Airspy HF+** is a [software-defined radio](/reference/software-defined-radio/)
optimised for the **HF and low-VHF** [bands](/reference/frequency-bands/), with
excellent dynamic range for receiving shortwave and weak low-band signals.

## Overview

Where a basic [RTL-SDR](/reference/rtl-sdr/) cannot tune HF directly (needing an
[upconverter](/reference/upconverter/) or direct-sampling mode), the HF+ is purpose-built
for it — ideal for [ionospheric](/reference/ionospheric-propagation/) shortwave
reception.

## Relevance to SDR

Choose the HF+ when HF or low-band is your target; GopherTrunk supports it as a receiver.
