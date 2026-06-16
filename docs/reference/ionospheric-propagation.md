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
  - { title: "Frequency, bands & the spectrum", url: /learn/frequency-and-spectrum/ }
external:
  - { title: "Skywave (Wikipedia)", url: https://en.wikipedia.org/wiki/Skywave }
---

**Ionospheric propagation** (skywave) is the refraction of
[HF](/reference/frequency-bands/) [radio waves](/reference/radio-wave/) by ionised
layers of the upper atmosphere, allowing signals to "skip" over the horizon for
hundreds or thousands of kilometres.

## How it works

The ionosphere's electron density varies with the sun, so HF skip changes with time of
day, season, and the solar cycle. Higher bands (VHF and up) generally pass through the
ionosphere rather than reflecting, so they stay line-of-sight.

## Relevance to SDR

Receiving HF skip needs an HF-capable radio such as the
[Airspy HF+](/reference/airspy-hf-plus/) or an [upconverter](/reference/upconverter/),
since a basic [RTL-SDR](/reference/rtl-sdr/) does not tune HF directly.
