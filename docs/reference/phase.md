---
slug: phase
title: Phase
entry_type: term
category: rf-fundamentals
description: Phase is the position of a point within a wave's cycle, measured in degrees or radians; shifting it carries information in phase-shift keying and is captured by IQ data.
keywords: phase, phase shift, degrees, radians, PSK, IQ, phase difference, carrier recovery
infobox:
  - { label: Type, value: Wave property }
  - { label: Unit, value: Degrees or radians }
  - { label: Encoded by, value: IQ angle }
see_also: [amplitude, iq-data, phase-shift-keying, constellation-diagram, costas-loop, phase-noise]
related_lessons:
  - { title: "IQ data & complex signals", url: /learn/rf-sdr/iq-data/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Phase_(waves)
  - https://en.wikipedia.org/wiki/Phase-shift_keying
---

**Phase** is the position of a point within the cycle of a wave, expressed in degrees
(0–360°) or radians (0–2π).[^wiki] Two waves of the same
[frequency](/reference/frequency/) can differ in phase, meaning one is shifted in time
relative to the other — a quarter-cycle lag is a 90° phase difference. Together with
[amplitude](/reference/amplitude/) and frequency, phase is one of the three carrier
properties a transmitter can vary, and deliberately jumping it between fixed values is how
digital radios send bits.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="Two identical sine waves offset horizontally, with the gap between them labelled phase difference, illustrating that same-frequency waves can differ in phase." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="70" x2="440" y2="70" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M20 70 q35 -40 70 0 t70 0 t70 0 t70 0 t70 0 t50 0" fill="none" stroke="currentColor" stroke-width="2"/>
  <path d="M55 70 q35 -40 70 0 t70 0 t70 0 t70 0 t70 0 t15 0" fill="none" stroke="currentColor" stroke-width="2" stroke-opacity="0.55" stroke-dasharray="5 3"/>
  <line x1="20" y1="100" x2="55" y2="100" stroke="currentColor"/>
  <text x="60" y="104" font-size="11" fill="currentColor">phase difference</text>
</svg>
<figcaption>Phase is where a wave sits in its cycle; shifting phase is the basis of PSK digital modulation.</figcaption>
</figure>

## How it works

Phase is only meaningful relative to a reference — either another wave or a notional
clock ticking at the carrier frequency. On the [IQ](/reference/iq-data/) plane, which is
how radios represent a signal, a sample's **angle** measured from the positive I axis is
its phase, and its distance from the origin is its amplitude. A rotating point traces out
the wave: constant frequency is steady rotation, and a sudden change of angle is a phase
shift. This geometric picture is why the IQ representation is so powerful — amplitude and
phase, the two independent quantities of a bandpass signal, become the polar coordinates
of a single complex number.

Because phase can be changed instantly and read out precisely, it is prime real estate
for carrying data. [Phase-shift keying](/reference/phase-shift-keying/) assigns bit
patterns to discrete phase positions — BPSK uses two (0° and 180°), QPSK four spaced 90°
apart — and the receiver decides which was sent by measuring the angle of each symbol.
[QAM](/reference/quadrature-amplitude-modulation/) goes further, using both phase and
amplitude at once. Differential schemes such as
[π/4-DQPSK](/reference/pi-4-dqpsk/) encode data in the *change* of phase between
successive symbols, sidestepping the need to know the absolute phase reference.

## In practice

- **Carrier recovery.** The receiver's idea of "zero phase" must be locked to the
  transmitter's, or the whole [constellation](/reference/constellation-diagram/) rotates.
  A [Costas loop](/reference/costas-loop/) or [PLL](/reference/phase-locked-loop/)
  estimates and removes this offset continuously.
- **Phase noise.** Real oscillators jitter, smearing each symbol's angle into a fuzzy
  cloud ([phase noise](/reference/phase-noise/)); too much of it collapses the margin
  between adjacent PSK points and forces errors.
- **Ambiguity.** With symmetric constellations the recovered phase can lock 90° or 180°
  off; differential coding or a known [sync](/reference/frame-synchronization/) pattern
  resolves which rotation is correct.

## Relevance to SDR

Tracking phase is essential to demodulating the PSK and QAM signals GopherTrunk handles.
The π/4-DQPSK used by P25 Phase 1 and the four-level schemes of DMR and NXDN all live in
the angle of the IQ samples, so GopherTrunk's carrier-recovery loop estimates the
incoming phase and rotates each symbol back onto the ideal constellation before slicing
it to bits. Residual phase error, whether from oscillator drift or channel-induced
[intersymbol interference](/reference/intersymbol-interference/), directly raises the
[error-vector magnitude](/reference/error-vector-magnitude/) and, past a point, breaks the
decode.

## Sources

[^wiki]: [Phase (waves)](https://en.wikipedia.org/wiki/Phase_(waves)) — Wikipedia, on the position within a wave's cycle and phase difference.
[^psk]: [Phase-shift keying](https://en.wikipedia.org/wiki/Phase-shift_keying) — Wikipedia, on encoding data by shifting a carrier's phase between discrete states.
