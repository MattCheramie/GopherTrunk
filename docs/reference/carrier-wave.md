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
  - { title: "Anatomy of a signal", url: /learn/signal-anatomy/ }
external:
  - { title: "Carrier wave (Wikipedia)", url: https://en.wikipedia.org/wiki/Carrier_wave }
---

A **carrier wave** is a steady [radio-frequency](/reference/radio-wave/) signal at a
single [frequency](/reference/frequency/) that conveys no information on its own. It
becomes useful only when [modulation](/reference/modulation/) varies one of its
properties in step with a message.

## How it works

An unmodulated carrier appears on a spectrum display as a single narrow spike.
Modulation spreads energy into sidebands around it, and the width of those sidebands is
essentially the signal's [bandwidth](/reference/bandwidth/).

## Relevance to SDR

Receivers tune to a carrier's frequency and then demodulate the variations around it.
A residual carrier at zero frequency after downconversion is the familiar "DC spike"
seen on SDR spectra.
