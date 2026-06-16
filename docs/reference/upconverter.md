---
slug: upconverter
title: Upconverter
entry_type: hardware
category: hardware
description: An upconverter shifts HF signals up into the tuning range of a VHF/UHF SDR such as an RTL-SDR, enabling shortwave reception on radios that cannot tune HF directly.
keywords: upconverter, HF converter, Ham It Up, shortwave, RTL-SDR HF, frequency shifting
aka: [upconverter]
autolink: true
infobox:
  - { label: Type, value: External RF converter }
  - { label: Function, value: Shifts HF up into VHF tuning range }
  - { label: Enables, value: HF reception on VHF/UHF SDRs }
see_also: [rtl-sdr, airspy-hf-plus, frequency-bands, local-oscillator]
related_lessons:
  - { title: "Frequency, bands & the spectrum", url: /learn/frequency-and-spectrum/ }
external:
  - { title: "Upconverter (Wikipedia)", url: https://en.wikipedia.org/wiki/Frequency_mixer }
---

An **upconverter** is an external device that **shifts HF signals up** into the tuning
range of a VHF/UHF SDR such as an [RTL-SDR](/reference/rtl-sdr/), letting radios that
cannot tune [HF](/reference/frequency-bands/) directly receive shortwave.

## How it works

It mixes the incoming HF signal with a fixed [local oscillator](/reference/local-oscillator/)
(commonly 100–125 MHz), so a 7 MHz signal appears around 107–132 MHz where the dongle can
tune. Software subtracts the offset to show true frequencies.

## Relevance to SDR

An upconverter is the budget route to HF on an RTL-SDR; a dedicated
[Airspy HF+](/reference/airspy-hf-plus/) is the higher-performance alternative.
