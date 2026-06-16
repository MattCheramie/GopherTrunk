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
  - { title: "Digital modulation & constellations", url: /learn/digital-modulation/ }
external:
  - { title: "Minimum-shift keying (Wikipedia)", url: https://en.wikipedia.org/wiki/Minimum-shift_keying }
---

**GMSK** (Gaussian minimum-shift keying) is a continuous-phase
[FSK](/reference/frequency-shift-keying/) variant in which the data is passed through a
Gaussian filter before modulation, smoothing phase transitions for a **compact
spectrum** and constant envelope.

## How it works

The constant envelope suits efficient non-linear amplifiers, while the Gaussian shaping
limits bandwidth. These traits made GMSK the choice for GSM cellular and for
[AIS](/reference/ais/) and [D-STAR](/reference/d-star/).

## Relevance to SDR

A GMSK demodulator recovers the underlying frequency/phase transitions; GopherTrunk
uses one in its AIS pipeline.
