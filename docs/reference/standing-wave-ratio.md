---
slug: standing-wave-ratio
title: Standing wave ratio (SWR)
entry_type: term
category: antennas
description: SWR measures how well an antenna is impedance-matched to its feedline and radio at a frequency; a poor match reflects power instead of transferring it.
keywords: SWR, VSWR, standing wave ratio, impedance match, reflection, return loss, reflection coefficient, feedpoint impedance
aka: [standing wave ratio, SWR, VSWR]
autolink: true
infobox:
  - { label: Type, value: Impedance-match metric }
  - { label: Ideal, value: 1:1 }
  - { label: Effect of mismatch, value: Reflected power, lost signal }
see_also: [antenna, dipole-antenna, feedpoint-impedance, return-loss, reflection-coefficient, impedance, attenuation]
related_lessons:
  - { title: "Antennas 101", url: /learn/rf-sdr/antennas/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Standing_wave_ratio
  - https://en.wikipedia.org/wiki/Reflection_coefficient
---

**Standing wave ratio** (**SWR**, often VSWR for *voltage* standing wave ratio) measures
how well an [antenna](/reference/antenna/) is impedance-matched to its feedline and radio at
a given [frequency](/reference/frequency/).[^wiki] A perfect match is **1:1**; larger
numbers mean a worse match and more reflected energy.

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

When the antenna's [feedpoint impedance](/reference/feedpoint-impedance/) does not equal the
feedline's characteristic impedance (usually 50 Ω), part of the wave travelling down the
line is reflected back at the junction. The forward and reflected waves overlap and, because
they travel in opposite directions, add and cancel at fixed points along the cable to form a
stationary interference pattern — a **standing wave**. SWR is defined as the ratio of the
maximum voltage on that pattern to the minimum:

- SWR = (1 + |Γ|) / (1 − |Γ|), where **Γ** is the
  [reflection coefficient](/reference/reflection-coefficient/) — the fraction of the wave's
  amplitude that bounces back.[^refl]
- A matched load reflects nothing, |Γ| = 0, so SWR = 1:1.
- A total mismatch (open, short, or pure reactance) reflects everything, |Γ| = 1, and SWR
  runs to infinity.

SWR is thus a single dial that summarizes the same mismatch that
[return loss](/reference/return-loss/) expresses in decibels and Γ expresses as a complex
number. A 2:1 SWR corresponds to roughly 11% of the *power* reflected, about 9.5 dB return
loss — noticeable but rarely fatal for receive.

## In practice

- **Frequency dependence** — SWR is only meaningful at a stated frequency. A resonant
  antenna is well-matched over a band and rises sharply outside it; an SWR sweep across the
  band shows the usable [bandwidth](/reference/bandwidth/).
- **Measurement** — an SWR meter, directional wattmeter, or a
  [NanoVNA](/reference/nanovna/) reads it directly. On receive-only setups a VNA is the
  practical tool.
- **Transmit vs receive stakes** — for a *transmitter*, high SWR sends reflected power back
  into the amplifier and can damage it, so a good match is safety-critical. For a
  *receive-only* SDR, the reflected energy is small and the real cost is a modest loss of
  signal reaching the radio.

## Relevance to SDR

For a scanner, SWR matters less than it does for a transmitter, but a reasonably matched,
resonant antenna still delivers more signal to the SDR than a badly mismatched one, helping
[SNR](/reference/signal-to-noise-ratio/) on weak [control channels](/reference/control-channel/).
A wildly high SWR usually points to a fault worth fixing — a wrong-band antenna, a broken
element, or water in the coax — rather than a subtle tuning issue. GopherTrunk sees only the
signal that survives the feedline; it has no way to correct a poor match, so keeping SWR
sensible is an antenna-and-feedline task done before the [ADC](/reference/analog-to-digital-converter/).

## Sources

[^wiki]: [Standing wave ratio](https://en.wikipedia.org/wiki/Standing_wave_ratio) — Wikipedia, on impedance matching, reflected power, and VSWR.
[^refl]: [Reflection coefficient](https://en.wikipedia.org/wiki/Reflection_coefficient) — Wikipedia, for Γ and the SWR = (1+|Γ|)/(1−|Γ|) relationship.
