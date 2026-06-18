---
slug: gmsk
title: GMSK
entry_type: technology
category: modulation
description: GMSK (Gaussian minimum-shift keying) is a continuous-phase FSK variant with a Gaussian pulse-shaping filter, giving a compact spectrum; used by AIS, GSM, and D-STAR.
keywords: GMSK, Gaussian minimum shift keying, continuous phase, AIS, GSM, D-STAR, pulse shaping
aka: [GMSK]
autolink: true
infobox:
  - { label: Type, value: Digital modulation (CPM) }
  - { label: Feature, value: Gaussian-filtered, compact spectrum }
  - { label: Used by, value: AIS, GSM, D-STAR }
see_also: [frequency-shift-keying, ais, d-star, root-raised-cosine-filter]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/rf-sdr/digital-modulation/ }
external:
  - { title: "Minimum-shift keying (Wikipedia)", url: https://en.wikipedia.org/wiki/Minimum-shift_keying }
---

**GMSK** (Gaussian minimum-shift keying) is a continuous-phase
[FSK](/reference/frequency-shift-keying/) variant in which the data is passed through a
Gaussian filter before modulation, smoothing phase transitions for a **compact
spectrum** and constant envelope.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A smooth continuous phase trajectory with no abrupt jumps, illustrating Gaussian-filtered MSK." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="60" x2="440" y2="60" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M20 60 C 60 30, 90 30, 120 60 S 180 90, 220 75 C 260 62, 270 35, 310 35 S 380 80, 440 55" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="20" y="100" font-size="10" fill="currentColor">phase is continuous and smoothly filtered — compact spectrum</text>
</svg>
<figcaption>GMSK is continuous-phase FSK with Gaussian pulse shaping, giving a narrow spectrum; AIS uses it.</figcaption>
</figure>

## How it works

The constant envelope suits efficient non-linear amplifiers, while the Gaussian shaping
limits bandwidth. These traits made GMSK the choice for GSM cellular and for
[AIS](/reference/ais/) and [D-STAR](/reference/d-star/).

## Relevance to SDR

A GMSK demodulator recovers the underlying frequency/phase transitions; GopherTrunk
uses one in its AIS pipeline.
