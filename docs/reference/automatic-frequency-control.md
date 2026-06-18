---
slug: automatic-frequency-control
title: Automatic frequency control (AFC)
entry_type: term
category: sdr-dsp
description: Automatic frequency control (AFC) continuously measures and cancels a carrier-frequency offset, keeping a demodulator centred on the signal as oscillators drift.
keywords: AFC, automatic frequency control, carrier offset, frequency tracking, PPM, drift
aka: [AFC, "automatic frequency control"]
autolink: true
see_also: [ppm-frequency-correction, costas-loop, demodulation, constellation-diagram]
related_lessons:
  - { title: "Tuning for a clean lock", url: /learn/rf-sdr/tuning-with-scopes/ }
external:
  - { title: "Automatic frequency control (Wikipedia)", url: https://en.wikipedia.org/wiki/Automatic_frequency_control }
---

**Automatic frequency control** (**AFC**) continuously measures a residual
carrier-frequency offset and nudges the receiver to cancel it, keeping the demodulator
centred on the signal as oscillators drift. Where [PPM correction](/reference/ppm-frequency-correction/)
fixes a static error, AFC **tracks** a changing one.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A signal drifting off centre over time being pulled back to the centre frequency by automatic frequency control." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="20" x2="40" y2="100" stroke="currentColor" stroke-opacity="0.4"/><line x1="40" y1="100" x2="430" y2="100" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="40" y1="55" x2="430" y2="55" stroke="currentColor" stroke-dasharray="4 3" stroke-opacity="0.5"/><text x="44" y="50" font-size="8" fill="currentColor">centre</text>
  <path d="M50 80 C 130 80 150 30 220 40 C 290 50 300 56 420 55" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="240" y="115" text-anchor="middle" font-size="8.5" fill="currentColor">AFC pulls the offset back toward centre</text>
</svg>
<figcaption>AFC continuously corrects carrier-frequency offset, keeping the demodulator locked as the signal drifts.</figcaption>
</figure>

## Overview

AFC often works alongside a [Costas loop](/reference/costas-loop/) (for phase) and
appears in GopherTrunk's receiver telemetry as a carrier-error reading; a steadily
**rotating [constellation](/reference/constellation-diagram/)** is the symptom AFC is
there to remove.
