---
slug: frequency-bands
title: Frequency bands (HF/VHF/UHF)
entry_type: term
category: rf-fundamentals
description: Frequency bands are conventional divisions of the radio spectrum — VLF, LF, MF, HF, VHF, UHF, SHF — each with characteristic propagation and uses.
keywords: frequency bands, HF, VHF, UHF, SHF, band plan, spectrum allocation
aka: [VHF, UHF, SHF]
autolink: true
infobox:
  - { label: Type, value: Spectrum divisions }
  - { label: HF, value: "3–30 MHz" }
  - { label: VHF, value: "30–300 MHz" }
  - { label: UHF, value: "300 MHz – 3 GHz" }
see_also: [electromagnetic-spectrum, frequency, radio-propagation, ionospheric-propagation]
related_lessons:
  - { title: "Frequency, bands & the spectrum", url: /learn/frequency-and-spectrum/ }
external:
  - { title: "Radio spectrum (Wikipedia)", url: https://en.wikipedia.org/wiki/Radio_spectrum }
---

**Frequency bands** are the conventional decade-wide divisions of the radio
[spectrum](/reference/electromagnetic-spectrum/): VLF, LF, MF, **HF** (3–30 MHz),
**VHF** (30–300 MHz), **UHF** (300 MHz–3 GHz), and SHF (3–30 GHz). Each behaves
differently and is allocated to different uses.

<figure class="figure" markdown="0">
<svg viewBox="0 0 480 110" role="img" aria-label="A ruler of radio bands from VLF through SHF, with VHF and UHF highlighted as the main scanner bands." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.2">
    <rect x="20" y="34" width="55" height="24" fill="none"/>
    <rect x="75" y="34" width="55" height="24" fill="none"/>
    <rect x="130" y="34" width="55" height="24" fill="none"/>
    <rect x="185" y="34" width="55" height="24" fill="none"/>
    <rect x="240" y="34" width="80" height="24" fill="currentColor" fill-opacity="0.22"/>
    <rect x="320" y="34" width="80" height="24" fill="currentColor" fill-opacity="0.22"/>
    <rect x="400" y="34" width="60" height="24" fill="none"/>
  </g>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="47" y="50">VLF</text><text x="102" y="50">LF</text><text x="157" y="50">MF</text><text x="212" y="50">HF</text>
    <text x="280" y="50">VHF</text><text x="360" y="50">UHF</text><text x="430" y="50">SHF</text>
  </g>
  <text x="20" y="80" font-size="9" fill="currentColor">3 kHz</text>
  <text x="460" y="80" font-size="9" fill="currentColor" text-anchor="end">300 GHz</text>
  <text x="320" y="94" text-anchor="middle" font-size="9" fill="currentColor">most scanning lives in VHF/UHF</text>
</svg>
<figcaption>The radio spectrum divided into bands; VHF and UHF carry most scanner and trunked-radio traffic.</figcaption>
</figure>

## How it works

HF can refract off the [ionosphere](/reference/ionospheric-propagation/) for worldwide
"skip", while VHF and UHF are essentially line-of-sight. Regulators publish band plans
assigning slices to broadcast, aviation, marine, public safety, and amateur use.

## Relevance to SDR

Most trunked-radio scanning happens in VHF, UHF, and the 700/800 MHz bands. Matching
an SDR's tuning range to the target band is the first hardware decision.
