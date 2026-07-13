---
slug: rayleigh-fading
title: Rayleigh fading
entry_type: term
category: propagation
description: Rayleigh fading is the statistical model for signal amplitude in a no-line-of-sight multipath channel, where the envelope follows a Rayleigh distribution.
keywords: Rayleigh fading, multipath fading, non-line-of-sight, NLOS, fading distribution, deep fade, flat fading, mobile channel
aka: [Rayleigh fading, Rayleigh channel]
autolink: true
infobox:
  - { label: Type, value: Small-scale fading model }
  - { label: Condition, value: No dominant line-of-sight path }
  - { label: Envelope PDF, value: Rayleigh distribution }
see_also: [rician-fading, multipath-propagation, radio-propagation, antenna-diversity, fade-margin]
cite_urls:
  - https://en.wikipedia.org/wiki/Rayleigh_fading
  - https://en.wikipedia.org/wiki/Fading
---

**Rayleigh fading** is a statistical model for the rapid amplitude variation of a
radio signal that arrives entirely by scattered and reflected paths, with **no**
dominant direct component.[^wiki] When many equal-strength [multipath](/reference/multipath-propagation/)
copies add with random phases, the resulting envelope follows a Rayleigh probability
distribution — hence the name. It is the worst-case small-scale fading model for a
mobile receiver deep inside urban clutter or dense foliage, and it underpins how
engineers size a [fade margin](/reference/fade-margin/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A time plot of received signal envelope showing frequent deep fades dipping below a decode threshold, characteristic of Rayleigh fading, with a Rayleigh probability density curve at the right." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="20" x2="30" y2="140" stroke="currentColor" stroke-width="1.2"/>
  <line x1="30" y1="140" x2="300" y2="140" stroke="currentColor" stroke-width="1.2" marker-end="url(#rfar)"/>
  <text x="150" y="158" text-anchor="middle" font-size="9" fill="currentColor">time / distance</text>
  <text x="18" y="30" font-size="9" fill="currentColor" transform="rotate(-90 18 80)">envelope</text>
  <line x1="30" y1="95" x2="300" y2="95" stroke="currentColor" stroke-width="0.8" stroke-dasharray="3 3"/>
  <text x="230" y="92" font-size="8" fill="currentColor">decode threshold</text>
  <path d="M30 70 Q45 40 60 75 T90 60 Q100 120 115 80 T140 55 Q152 130 168 90 T200 65 Q214 118 230 82 T262 58 Q276 125 292 78" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <line x1="330" y1="20" x2="330" y2="140" stroke="currentColor" stroke-width="1.2"/>
  <line x1="330" y1="140" x2="450" y2="140" stroke="currentColor" stroke-width="1.2"/>
  <path d="M330 140 Q355 138 372 100 Q388 66 405 78 Q428 96 450 130" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="390" y="158" text-anchor="middle" font-size="9" fill="currentColor">Rayleigh PDF</text>
  <defs><marker id="rfar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A no-line-of-sight channel produces deep, frequent fades; the envelope statistics follow a Rayleigh distribution.</figcaption>
</figure>

## How it works

Model the received signal as a sum of many independent scattered rays. By the central
limit theorem, the in-phase (I) and quadrature (Q) components each become
zero-mean Gaussian random variables. The envelope, `r = √(I² + Q²)`, then follows
the **Rayleigh** distribution:

- Probability density: `p(r) = (r/σ²)·exp(−r²/2σ²)` for `r ≥ 0`.
- The average power is `2σ²`; the phase is uniform over 0–2π.
- Because there is no steady dominant term, the envelope can occasionally cancel
  almost completely, producing **deep fades** tens of decibels below the mean.

The fading is called **flat** when the channel's coherence bandwidth exceeds the signal
bandwidth (all frequencies fade together) and **frequency-selective** otherwise, when
different parts of the spectrum fade independently and cause
[intersymbol interference](/reference/intersymbol-interference/). How fast the envelope
moves is set by the [Doppler shift](/reference/doppler-shift/): a faster mobile crosses
the standing-wave pattern more quickly, giving a higher fade rate.

## Relevance to SDR

Rayleigh fading dominates the land-mobile channels that trunking scanners listen to.
A pedestrian or vehicle in a city receives P25, DMR, TETRA, and NXDN signals with no
clean line of sight, so the instantaneous level swings by tens of dB over fractions of
a wavelength. This is why a talkgroup can be perfectly readable one moment and drop into
a burst of errors the next, even with a strong average signal — a deep fade briefly
pushed the carrier below the demodulator's usable
[signal-to-noise ratio](/reference/signal-to-noise-ratio/).

Cellular and broadcast systems fight Rayleigh fading with
[antenna diversity](/reference/antenna-diversity/), interleaving, and
[forward error correction](/reference/forward-error-correction/) so that a fade that
kills one branch or one symbol span does not kill the whole message.
[GopherTrunk](/reference/software-defined-radio/) is a receiver: it does not implement
diversity combining, but its decode chain relies on the FEC and interleaving already
built into the on-air protocols to ride through the short fades, and it exposes
per-frame [error-vector-magnitude](/reference/error-vector-magnitude/) and SNR estimates
that reveal fading directly.

## In practice

The single most useful consequence is the **fade margin**: because deep fades are
inevitable, a link must be budgeted with headroom above the bare minimum SNR. For a
Rayleigh channel, achieving a given outage probability requires many decibels of margin,
which is why mobile systems close their [link budget](/reference/link-budget/)
conservatively. When a line-of-sight component is present, the channel is better
described by [Rician fading](/reference/rician-fading/), of which Rayleigh is the
limiting case (K-factor = 0).

## Sources

[^wiki]: [Rayleigh fading](https://en.wikipedia.org/wiki/Rayleigh_fading) — Wikipedia, on the no-line-of-sight multipath model and its Rayleigh-distributed envelope.
