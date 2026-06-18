---
slug: carrier-wave
title: Carrier wave
entry_type: term
category: rf-fundamentals
description: A carrier wave is a steady radio-frequency signal that carries no information by itself until it is modulated, varying its amplitude, frequency, or phase to convey a message.
keywords: carrier wave, carrier, modulation, unmodulated, RF
aka: [carrier wave, carrier]
autolink: true
infobox:
  - { label: Type, value: Reference signal }
  - { label: Carries info via, value: Modulation }
  - { label: Appears as, value: Single spectral spike (unmodulated) }
see_also: [modulation, radio-wave, frequency, amplitude-modulation]
related_lessons:
  - { title: "Anatomy of a signal", url: /learn/rf-sdr/signal-anatomy/ }
external:
  - { title: "Carrier wave (Wikipedia)", url: https://en.wikipedia.org/wiki/Carrier_wave }
---

A **carrier wave** is a steady [radio-frequency](/reference/radio-wave/) signal at a
single [frequency](/reference/frequency/) that conveys no information on its own. It
becomes useful only when [modulation](/reference/modulation/) varies one of its
properties in step with a message.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A steady unmodulated carrier wave on top, and the same carrier amplitude-modulated by a message below." xmlns="http://www.w3.org/2000/svg">
  <text x="20" y="20" font-size="10" fill="currentColor">unmodulated carrier</text>
  <path d="M20 45 q10 -18 20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <text x="20" y="95" font-size="10" fill="currentColor">modulated (carries information)</text>
  <path d="M20 120 q10 -8 20 0 t20 0 q10 -22 20 0 t20 0 q10 -22 20 0 t20 0 q10 -8 20 0 t20 0 q10 -4 20 0 t20 0 q10 -8 20 0 t20 0 q10 -22 20 0 t20 0 q10 -22 20 0 t20 0 q10 -8 20 0 t20 0" fill="none" stroke="currentColor" stroke-width="1.6"/>
</svg>
<figcaption>A bare carrier carries no information until modulation varies its amplitude, frequency, or phase.</figcaption>
</figure>

## How it works

An unmodulated carrier appears on a spectrum display as a single narrow spike.
Modulation spreads energy into sidebands around it, and the width of those sidebands is
essentially the signal's [bandwidth](/reference/bandwidth/).

## Relevance to SDR

Receivers tune to a carrier's frequency and then demodulate the variations around it.
A residual carrier at zero frequency after downconversion is the familiar "DC spike"
seen on SDR spectra.
