---
slug: standing-wave-ratio
title: Standing wave ratio (SWR)
entry_type: term
category: antennas-propagation
description: SWR measures how well an antenna is impedance-matched to its feedline and radio at a frequency; a poor match reflects power instead of transferring it.
keywords: SWR, VSWR, standing wave ratio, impedance match, reflection, return loss
aka: [standing wave ratio, SWR, VSWR]
autolink: true
infobox:
  - { label: Type, value: Impedance-match metric }
  - { label: Ideal, value: 1:1 }
  - { label: Effect of mismatch, value: Reflected power, lost signal }
see_also: [antenna, dipole-antenna, attenuation]
related_lessons:
  - { title: "Antennas 101", url: /learn/antennas/ }
external:
  - { title: "Standing wave ratio (Wikipedia)", url: https://en.wikipedia.org/wiki/Standing_wave_ratio }
---

**Standing wave ratio** (**SWR**, often VSWR) measures how well an
[antenna](/reference/antenna/) is impedance-matched to its feedline and radio at a
given [frequency](/reference/frequency/). A perfect match is 1:1.

## How it works

When the match is poor, some energy is reflected back rather than transferred,
creating standing waves on the feedline. For transmitting this can damage equipment;
for receive-only SDR use it mainly costs a little signal.

## Relevance to SDR

A reasonably matched, resonant antenna delivers more signal to the SDR than a
mismatched one, helping [SNR](/reference/signal-to-noise-ratio/).
