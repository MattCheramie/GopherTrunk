---
slug: phase
title: Phase
entry_type: term
category: rf-fundamentals
description: Phase is the position of a point within a wave's cycle, measured in degrees or radians; shifting it carries information in phase-shift keying and is captured by IQ data.
keywords: phase, phase shift, degrees, radians, PSK, IQ
infobox:
  - { label: Type, value: Wave property }
  - { label: Unit, value: Degrees or radians }
  - { label: Encoded by, value: IQ angle }
see_also: [amplitude, iq-data, phase-shift-keying, constellation-diagram]
related_lessons:
  - { title: "IQ data & complex signals", url: /learn/iq-data/ }
external:
  - { title: "Phase (waves) (Wikipedia)", url: https://en.wikipedia.org/wiki/Phase_(waves) }
---

**Phase** is the position of a point within the cycle of a wave, expressed in degrees
(0–360°) or radians. Two waves of the same [frequency](/reference/frequency/) can
differ in phase, meaning one is shifted in time relative to the other.

## How it works

On the [IQ](/reference/iq-data/) plane, a sample's **angle** is its phase and its
distance from the origin is its [amplitude](/reference/amplitude/). Deliberately
jumping the phase between fixed values encodes data — the basis of
[phase-shift keying](/reference/phase-shift-keying/).

## Relevance to SDR

Tracking phase is essential to demodulating PSK and QAM signals; a
[Costas loop](/reference/costas-loop/) recovers the carrier phase so symbols land
correctly on the [constellation](/reference/constellation-diagram/).
