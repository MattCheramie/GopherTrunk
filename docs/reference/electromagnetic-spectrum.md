---
slug: electromagnetic-spectrum
title: Electromagnetic spectrum
entry_type: term
category: rf-fundamentals
description: The electromagnetic spectrum is the full range of radiation by frequency — from radio waves through microwaves and visible light to X-rays and gamma rays.
keywords: electromagnetic spectrum, EM spectrum, radio waves, frequency range, light, microwaves
aka: [electromagnetic spectrum, EM spectrum]
autolink: true
infobox:
  - { label: Type, value: Physical concept }
  - { label: Radio portion, value: ~3 kHz – 300 GHz }
  - { label: Travels at, value: Speed of light }
see_also: [radio-wave, frequency, wavelength, frequency-bands]
related_lessons:
  - { title: "What is a radio wave?", url: /learn/rf-sdr/radio-waves/ }
external:
  - { title: "Electromagnetic spectrum (Wikipedia)", url: https://en.wikipedia.org/wiki/Electromagnetic_spectrum }
---

The **electromagnetic spectrum** is the full range of electromagnetic radiation
ordered by frequency (or, equivalently, wavelength). It spans from low-frequency
[radio waves](/reference/radio-wave/) through microwaves, infrared, visible light,
ultraviolet, X-rays, and gamma rays — all the same phenomenon vibrating at different
rates.

<figure class="figure" markdown="0">
<svg viewBox="0 0 480 110" role="img" aria-label="The electromagnetic spectrum from radio waves through microwaves, infrared, visible light, ultraviolet, X-rays and gamma rays, with the radio portion highlighted." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="30" width="60" height="26" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.2"/>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <rect x="80" y="30" width="60" height="26"/><rect x="140" y="30" width="60" height="26"/>
    <rect x="200" y="30" width="60" height="26"/><rect x="260" y="30" width="60" height="26"/>
    <rect x="320" y="30" width="60" height="26"/><rect x="380" y="30" width="80" height="26"/>
  </g>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <text x="50" y="47">Radio</text><text x="110" y="47">Micro</text><text x="170" y="47">IR</text>
    <text x="230" y="47">Visible</text><text x="290" y="47">UV</text><text x="350" y="47">X-ray</text><text x="420" y="47">Gamma</text>
  </g>
  <text x="20" y="78" font-size="9" fill="currentColor">lower frequency · longer wavelength</text>
  <text x="460" y="78" font-size="9" fill="currentColor" text-anchor="end">higher frequency</text>
  <line x1="20" y1="66" x2="460" y2="66" stroke="currentColor" stroke-opacity="0.4" marker-end="url(#emar)"/>
  <defs><marker id="emar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Radio occupies the low-frequency, long-wavelength end of the same electromagnetic spectrum as light and X-rays.</figcaption>
</figure>

## Overview

Every part of the spectrum is electromagnetic energy travelling at the speed of light;
only the [frequency](/reference/frequency/) (and thus [wavelength](/reference/wavelength/))
differs. Radio occupies the low-frequency end, conventionally about **3 kHz to 300
GHz**, which is slow enough that electronics can generate and detect it directly.

## Relevance to SDR

Software-defined radios operate within the radio portion of the spectrum. Where a
signal sits in that range — its [band](/reference/frequency-bands/) — determines how it
propagates, what antenna it needs, and what equipment can receive it.
