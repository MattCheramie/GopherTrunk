---
slug: ppm-frequency-correction
title: PPM frequency correction
entry_type: term
category: sdr-dsp
description: PPM frequency correction compensates an SDR's reference-oscillator error, measured in parts per million, so signals appear at their true frequency and digital modes lock.
keywords: PPM, parts per million, frequency error, oscillator drift, calibration, rotating constellation, crystal error, TCXO
aka: [PPM correction, PPM]
autolink: true
infobox:
  - { label: Type, value: Calibration parameter }
  - { label: Corrects, value: Reference-oscillator error }
  - { label: Symptom if wrong, value: Rotating constellation, no lock }
see_also: [local-oscillator, frequency, costas-loop, constellation-diagram, automatic-frequency-control, frequency-stability, phase-noise]
related_lessons:
  - { title: "Calibration & troubleshooting", url: /learn/rf-sdr/calibration-troubleshooting/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Clock_drift
  - https://en.wikipedia.org/wiki/Frequency_drift
---

**PPM frequency correction** compensates for the small error in an SDR's reference
oscillator, measured in **parts per million**, so signals land on their **true
frequency**.[^wiki] The unit is proportional: one PPM is one millionth of the tuned frequency,
so the same crystal error grows in absolute terms with frequency. At UHF a 30 PPM error is
several kilohertz — more than a channel's width — while the same 30 PPM at the HF broadcast band
is only a few hundred hertz.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A signal sitting off the channel centre due to oscillator error, and the same signal centred after PPM correction." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="220" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="125" y1="30" x2="125" y2="78" stroke="currentColor" stroke-dasharray="3 3" stroke-opacity="0.6"/>
  <path d="M150 70 L160 38 L170 70 Z" fill="currentColor" fill-opacity="0.25" stroke="currentColor"/>
  <text x="125" y="95" text-anchor="middle" font-size="9" fill="currentColor">off-centre (uncorrected)</text>
  <line x1="245" y1="55" x2="285" y2="55" stroke="currentColor" marker-end="url(#ppar)"/><text x="265" y="48" font-size="8" fill="currentColor" text-anchor="middle">+PPM</text>
  <line x1="300" y1="70" x2="450" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="375" y1="30" x2="375" y2="78" stroke="currentColor" stroke-dasharray="3 3" stroke-opacity="0.6"/>
  <path d="M365 70 L375 38 L385 70 Z" fill="currentColor" fill-opacity="0.25" stroke="currentColor"/>
  <text x="375" y="95" text-anchor="middle" font-size="9" fill="currentColor">centred (corrected)</text>
  <defs><marker id="ppar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>PPM correction compensates the dongle's oscillator error so signals land on their true frequency.</figcaption>
</figure>

## How it works

Every SDR derives its tuning from a reference crystal that is never exactly on its nominal
frequency. Setting a PPM value scales the [local oscillator](/reference/local-oscillator/) by the
same proportion, so a signal known to be on a particular frequency actually appears there rather
than a few kHz away. In practice you find the number by tuning a station whose frequency is known
precisely — a broadcast pilot, a GSM control channel, a trunking control channel — and adjusting
PPM until the signal sits dead centre.

Because the correction is a single multiplicative factor, it fixes a **static** error across the
whole band at once: dial in the right PPM and every channel lands correctly. What it cannot fix
is *change* over time, which is the job of downstream tracking.

## In practice

- **Warm-up drift.** A crystal's frequency shifts as it heats from cold start to operating
  temperature, so PPM measured on a cold radio will be wrong once it stabilises. Let the dongle
  warm up ten to thirty minutes before calibrating.
- **Temperature sensitivity.** Plain crystals drift with ambient temperature; a
  [TCXO](/reference/tcxo/)-equipped SDR holds its frequency far more tightly (often within 1
  PPM), reducing how often calibration matters — this is a question of the reference's
  [frequency stability](/reference/frequency-stability/).
- **PPM is not AFC.** PPM correction sets a one-time offset; where the error keeps moving,
  [automatic frequency control](/reference/automatic-frequency-control/) tracks and cancels the
  residual continuously. The two work together: PPM removes the bulk static error, AFC and the
  demodulator's [Costas loop](/reference/costas-loop/) mop up the rest.
- **Residual phase noise.** Even a perfectly centred oscillator has short-term
  [phase noise](/reference/phase-noise/); PPM addresses the mean frequency, not this jitter.

## Relevance to SDR

A wrong PPM produces the classic *rotating [constellation](/reference/constellation-diagram/)*
that won't lock — the whole symbol pattern spinning because the carrier sits at a constant offset
from where the receiver expects it. It is fixed by calibration, not by a better antenna or more
gain, and it is one of the first things to check when a strong, clean-looking digital signal
still fails to decode. GopherTrunk applies a configured PPM offset to its tuning and relies on
per-channel frequency tracking to follow any remaining drift.

## Sources

[^wiki]: [Clock drift](https://en.wikipedia.org/wiki/Clock_drift) — Wikipedia, on oscillator frequency error and drift measured in parts per million.
[^drift]: [Frequency drift](https://en.wikipedia.org/wiki/Frequency_drift) — Wikipedia, on the temperature- and age-driven change in oscillator frequency that calibration and tracking address.
