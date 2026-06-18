---
slug: frequency
title: Frequency
entry_type: term
category: rf-fundamentals
description: Frequency is the number of cycles a wave completes per second, measured in hertz (Hz); for radio it sets the tuning point and, inversely, the wavelength.
keywords: frequency, hertz, Hz, cycles per second, kHz MHz GHz
infobox:
  - { label: Symbol, value: f }
  - { label: Unit, value: Hertz (Hz) }
  - { label: Relation, value: "wavelength = c / frequency" }
see_also: [wavelength, frequency-bands, electromagnetic-spectrum, radio-wave]
related_lessons:
  - { title: "What is a radio wave?", url: /learn/rf-sdr/radio-waves/ }
  - { title: "Frequency, bands & the spectrum", url: /learn/rf-sdr/frequency-and-spectrum/ }
external:
  - { title: "Frequency (Wikipedia)", url: https://en.wikipedia.org/wiki/Frequency }
---

**Frequency** is the number of cycles a periodic wave completes each second, measured
in **hertz (Hz)**. For a [radio wave](/reference/radio-wave/) it is the quantity you
tune to, and it is inversely related to [wavelength](/reference/wavelength/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Two sine waves: a low-frequency wave with few cycles on top and a high-frequency wave with many cycles below." xmlns="http://www.w3.org/2000/svg">
  <text x="20" y="24" font-size="10" fill="currentColor">low frequency</text>
  <path d="M20 50 q30 -28 60 0 t60 0 t60 0 t60 0 t60 0 t60 0 t40 0" fill="none" stroke="currentColor" stroke-width="2"/>
  <text x="20" y="100" font-size="10" fill="currentColor">high frequency</text>
  <path d="M20 122 q12 -26 24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0" fill="none" stroke="currentColor" stroke-width="2"/>
</svg>
<figcaption>Frequency is how many cycles pass each second; higher frequency packs more cycles into the same time (and means a shorter wavelength).</figcaption>
</figure>

## How it works

One hertz is one cycle per second. Radio frequencies are large, so they are scaled in
kilohertz (kHz), megahertz (MHz), and gigahertz (GHz). Because all radio waves travel
at the speed of light *c*, frequency and wavelength satisfy *wavelength = c /
frequency*.

## Relevance to SDR

Tuning an SDR sets the centre frequency its [local oscillator](/reference/local-oscillator/)
mixes down to baseband. The chosen frequency, within a [band](/reference/frequency-bands/),
determines what signal you receive.
