---
slug: upconverter
title: Upconverter
entry_type: hardware
category: rf-front-end
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
  - { title: "Frequency, bands & the spectrum", url: /learn/rf-sdr/frequency-and-spectrum/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Frequency_mixer
---

An **upconverter** is an external device that **shifts HF signals up** into the tuning
range of a VHF/UHF SDR such as an [RTL-SDR](/reference/rtl-sdr/), letting radios that
cannot tune [HF](/reference/frequency-bands/) directly receive shortwave.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A low HF input shifted upward by a fixed offset into the RTL-SDR's tunable range." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="430" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <rect x="40" y="55" width="30" height="15" fill="currentColor" fill-opacity="0.3" stroke="currentColor" stroke-width="1.1"/><text x="55" y="90" font-size="8" fill="currentColor" text-anchor="middle">HF</text>
  <line x1="80" y1="48" x2="250" y2="48" stroke="currentColor" stroke-dasharray="4 3" marker-end="url(#upar)"/><text x="165" y="42" font-size="8" fill="currentColor" text-anchor="middle">+125 MHz</text>
  <rect x="260" y="55" width="30" height="15" fill="currentColor" fill-opacity="0.3" stroke="currentColor" stroke-width="1.1"/><text x="275" y="90" font-size="8" fill="currentColor" text-anchor="middle">shifted</text>
  <text x="360" y="62" font-size="8" fill="currentColor">RTL-SDR range</text>
  <defs><marker id="upar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>An upconverter shifts HF signals up into the RTL-SDR's tunable range, since the dongle can't reach HF directly.</figcaption>
</figure>

## How it works

It mixes the incoming HF signal with a fixed [local oscillator](/reference/local-oscillator/)
(commonly 100–125 MHz), so a 7 MHz signal appears around 107–132 MHz where the dongle can
tune. Software subtracts the offset to show true frequencies.

## Relevance to SDR

An upconverter is the budget route to HF on an RTL-SDR; a dedicated
[Airspy HF+](/reference/airspy-hf-plus/) is the higher-performance alternative.

## Sources

[^wiki]: [Frequency mixer](https://en.wikipedia.org/wiki/Frequency_mixer) — Wikipedia, on the mixing principle an upconverter uses to shift HF up into a VHF/UHF tuning range.
