---
slug: automatic-frequency-control
title: Automatic frequency control (AFC)
entry_type: term
category: sdr-dsp
description: Automatic frequency control (AFC) continuously measures and cancels a carrier-frequency offset, keeping a demodulator centred on the signal as oscillators drift.
keywords: AFC, automatic frequency control, carrier offset, frequency tracking, PPM, drift, frequency-locked loop, carrier recovery
aka: [AFC, "automatic frequency control"]
autolink: true
infobox:
  - { label: Type, value: Frequency-tracking loop }
  - { label: Corrects, value: Residual carrier offset (dynamic) }
  - { label: Symptom it removes, value: Rotating constellation }
see_also: [ppm-frequency-correction, costas-loop, demodulation, constellation-diagram, frequency-locked-loop, frequency-stability, phase-noise, afc-alias-traps]
related_lessons:
  - { title: "Tuning for a clean lock", url: /learn/rf-sdr/tuning-with-scopes/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Automatic_frequency_control
  - https://en.wikipedia.org/wiki/Frequency-locked_loop
---

**Automatic frequency control** (**AFC**) continuously measures a residual
carrier-frequency offset and nudges the receiver to cancel it, keeping the demodulator
centred on the signal as oscillators drift.[^wiki] Where [PPM correction](/reference/ppm-frequency-correction/)
fixes a **static** error once, AFC **tracks** a changing one — it is a closed loop, not a
one-time setting.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A signal drifting off centre over time being pulled back to the centre frequency by automatic frequency control." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="20" x2="40" y2="100" stroke="currentColor" stroke-opacity="0.4"/><line x1="40" y1="100" x2="430" y2="100" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="40" y1="55" x2="430" y2="55" stroke="currentColor" stroke-dasharray="4 3" stroke-opacity="0.5"/><text x="44" y="50" font-size="8" fill="currentColor">centre</text>
  <path d="M50 80 C 130 80 150 30 220 40 C 290 50 300 56 420 55" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="240" y="115" text-anchor="middle" font-size="8.5" fill="currentColor">AFC pulls the offset back toward centre</text>
</svg>
<figcaption>AFC continuously corrects carrier-frequency offset, keeping the demodulator locked as the signal drifts.</figcaption>
</figure>

## How it works

AFC is a feedback loop with three parts, like any control loop: a **frequency-error detector**
that estimates how far the carrier sits from the receiver's centre, a **loop filter** that
smooths the estimate, and a **correction** applied by shifting the tuning — retuning a
numerically controlled oscillator, or adding a phase rotation per sample to the
[IQ](/reference/iq-data/) stream. The error detector can measure offset directly (for example
from the average rate of phase rotation in the demodulated signal, or from the position of a
known pilot or spectral feature) and the loop drives that error toward zero.

The loop's **bandwidth** sets its character: a wide loop acquires and follows fast drift but lets
more noise through and can be pulled off by interference; a narrow loop is steady and noise-immune
but slow to catch a moving carrier. AFC typically handles a larger frequency range than a phase
loop but to coarser precision, so it is often paired with a finer phase tracker.

## Variants

- **Frequency-locked loop (FLL).** A pure AFC that locks *frequency* only, tolerating any phase.
  Robust and wide-range, it is the usual acquisition aid before a phase loop takes over — see
  [frequency-locked loop](/reference/frequency-locked-loop/).
- **AFC + Costas loop.** A [Costas loop](/reference/costas-loop/) recovers *phase* for coherent
  PSK demodulation but has a narrow pull-in range; an AFC/FLL first drags the carrier close
  enough for the Costas loop to lock. This two-stage acquire-then-track pattern is standard in
  digital receivers.
- **Data-aided vs non-data-aided.** The error detector may use known symbols (a preamble or sync
  word) for a clean estimate, or operate blindly on the modulated signal.

## In practice

AFC exists because oscillators are never perfectly stable: the reference drifts with temperature
and age (a matter of [frequency stability](/reference/frequency-stability/)), and short-term
[phase noise](/reference/phase-noise/) jitters the carrier even when its mean frequency is right.
A steadily **rotating [constellation](/reference/constellation-diagram/)** is the visible symptom
AFC is there to remove — the whole symbol pattern spinning at a rate equal to the residual offset.
AFC catches what static PPM correction leaves behind and what changes after calibration, so the
two are complementary rather than alternatives: PPM removes the bulk error, AFC follows the
remainder in real time.

## Relevance to SDR

AFC often works alongside a Costas loop and appears in GopherTrunk's receiver telemetry as a
carrier-error reading; the loop keeps each channel's demodulator centred as the tuner and the
transmitter's own reference drift. For narrowband trunking signals, where a few hundred hertz of
uncorrected offset is enough to break symbol decisions, this continuous tracking is what turns an
intermittent lock into a stable decode.

## Sources

[^wiki]: [Automatic frequency control](https://en.wikipedia.org/wiki/Automatic_frequency_control) — Wikipedia, on tracking and cancelling carrier-frequency offset.
[^fll]: [Frequency-locked loop](https://en.wikipedia.org/wiki/Frequency-locked_loop) — Wikipedia, on the frequency-only tracking loop commonly used to acquire before phase lock.
