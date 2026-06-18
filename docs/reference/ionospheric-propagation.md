---
slug: ionospheric-propagation
title: Ionospheric propagation
entry_type: term
category: antennas-propagation
description: Ionospheric propagation is the refraction of HF radio waves by charged layers of the upper atmosphere, enabling long-distance "skip" communication around the world.
keywords: ionosphere, skip, skywave, HF propagation, shortwave, refraction
aka: [ionospheric propagation, skywave]
autolink: true
infobox:
  - { label: Type, value: HF propagation mode }
  - { label: Mechanism, value: Refraction by ionosphere }
  - { label: Enables, value: Worldwide HF "skip" }
see_also: [radio-propagation, frequency-bands, airspy-hf-plus]
related_lessons:
  - { title: "Frequency, bands & the spectrum", url: /learn/rf-sdr/frequency-and-spectrum/ }
external:
  - { title: "Skywave (Wikipedia)", url: https://en.wikipedia.org/wiki/Skywave }
---

**Ionospheric propagation** (skywave) is the refraction of
[HF](/reference/frequency-bands/) [radio waves](/reference/radio-wave/) by ionised
layers of the upper atmosphere, allowing signals to "skip" over the horizon for
hundreds or thousands of kilometres.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="An HF signal leaving a transmitter, refracting off the ionosphere layer, and returning to a distant receiver." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="120" x2="440" y2="120" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="20" y1="35" x2="440" y2="35" stroke="currentColor" stroke-opacity="0.4" stroke-dasharray="6 4"/><text x="20" y="28" font-size="9" fill="currentColor">ionosphere</text>
  <line x1="60" y1="118" x2="60" y2="100" stroke="currentColor" stroke-width="2"/><text x="60" y="135" text-anchor="middle" font-size="8" fill="currentColor">TX</text>
  <line x1="400" y1="118" x2="400" y2="100" stroke="currentColor" stroke-width="2"/><text x="400" y="135" text-anchor="middle" font-size="8" fill="currentColor">RX (far)</text>
  <path d="M60 100 L230 38 L400 100" fill="none" stroke="currentColor" stroke-width="1.5" marker-end="url(#ioar)"/>
  <defs><marker id="ioar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>HF signals can refract off the ionosphere and "skip" thousands of kilometres beyond the horizon.</figcaption>
</figure>

## How it works

The ionosphere's electron density varies with the sun, so HF skip changes with time of
day, season, and the solar cycle. Higher bands (VHF and up) generally pass through the
ionosphere rather than reflecting, so they stay line-of-sight.

## Relevance to SDR

Receiving HF skip needs an HF-capable radio such as the
[Airspy HF+](/reference/airspy-hf-plus/) or an [upconverter](/reference/upconverter/),
since a basic [RTL-SDR](/reference/rtl-sdr/) does not tune HF directly.
