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
  - { title: "Antennas 101", url: /learn/rf-sdr/antennas/ }
external:
  - { title: "Standing wave ratio (Wikipedia)", url: https://en.wikipedia.org/wiki/Standing_wave_ratio }
---

**Standing wave ratio** (**SWR**, often VSWR) measures how well an
[antenna](/reference/antenna/) is impedance-matched to its feedline and radio at a
given [frequency](/reference/frequency/). A perfect match is 1:1.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A forward wave travelling toward an antenna and a smaller reflected wave returning from a mismatch." xmlns="http://www.w3.org/2000/svg">
  <rect x="380" y="45" width="50" height="40" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="405" y="70" text-anchor="middle" font-size="8" fill="currentColor">antenna</text>
  <line x1="30" y1="50" x2="375" y2="50" stroke="currentColor" stroke-width="1.4" marker-end="url(#swf)"/><text x="60" y="42" font-size="9" fill="currentColor">forward</text>
  <line x1="375" y1="80" x2="30" y2="80" stroke="currentColor" stroke-width="1.2" stroke-dasharray="4 3" marker-end="url(#swr)"/><text x="60" y="98" font-size="9" fill="currentColor">reflected (mismatch)</text>
  <defs><marker id="swf" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker><marker id="swr" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>SWR measures how well the antenna is matched; a poor match reflects energy back instead of radiating it.</figcaption>
</figure>

## How it works

When the match is poor, some energy is reflected back rather than transferred,
creating standing waves on the feedline. For transmitting this can damage equipment;
for receive-only SDR use it mainly costs a little signal.

## Relevance to SDR

A reasonably matched, resonant antenna delivers more signal to the SDR than a
mismatched one, helping [SNR](/reference/signal-to-noise-ratio/).
