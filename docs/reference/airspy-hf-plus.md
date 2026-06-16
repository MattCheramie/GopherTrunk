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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A frequency coverage bar for Airspy HF+ (HF + low VHF) on an axis from about 0 to 6 gigahertz." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="430" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="30" y="86">0</text><text x="163" y="86">2 GHz</text><text x="296" y="86">4 GHz</text><text x="430" y="86">6 GHz</text></g>
  <rect x="30" y="40" width="18" height="20" rx="3" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.3"/>
  <text x="230" y="28" text-anchor="middle" font-size="10" fill="currentColor">Airspy HF+ (HF + low VHF) coverage</text>
</svg>
<figcaption>The Airspy HF+ is optimised for the lower bands (HF and low VHF) with excellent dynamic range.</figcaption>
</figure>

## Overview

Where a basic [RTL-SDR](/reference/rtl-sdr/) cannot tune HF directly (needing an
[upconverter](/reference/upconverter/) or direct-sampling mode), the HF+ is purpose-built
for it — ideal for [ionospheric](/reference/ionospheric-propagation/) shortwave
reception.

## Relevance to SDR

Choose the HF+ when HF or low-band is your target; GopherTrunk supports it as a receiver.
