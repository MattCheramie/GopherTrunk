---
slug: costas-loop
title: Costas loop
entry_type: algorithm
category: sdr-dsp
description: A Costas loop is a feedback circuit that recovers a suppressed carrier's phase and frequency, enabling coherent demodulation of PSK and other phase-modulated signals.
keywords: Costas loop, carrier recovery, phase locked loop, PSK, coherent demodulation, John Costas
aka: [Costas loop]
autolink: true
infobox:
  - { label: Type, value: Carrier-recovery loop }
  - { label: Recovers, value: Carrier phase and frequency }
  - { label: Used for, value: PSK/QAM coherent demodulation }
see_also: [phase-shift-keying, phase, demodulation, cma-equalizer, ppm-frequency-correction]
related_lessons:
  - { title: "Tuning for a clean lock", url: /learn/rf-sdr/tuning-with-scopes/ }
external:
  - { title: "Costas loop (Wikipedia)", url: https://en.wikipedia.org/wiki/Costas_loop }
---

A **Costas loop** is a phase-locked feedback structure that recovers the
[phase](/reference/phase/) and frequency of a suppressed carrier, enabling **coherent**
[demodulation](/reference/demodulation/) of [PSK](/reference/phase-shift-keying/) and
related signals.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A feedback loop: input to a phase detector, to a loop filter, to a controlled oscillator that feeds back to the phase detector." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="60" y="40" width="70" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="95" y="54">phase</text><text x="95" y="65">detector</text>
    <rect x="180" y="40" width="70" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="215" y="54">loop</text><text x="215" y="65">filter</text>
    <rect x="300" y="40" width="80" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="340" y="54">controlled</text><text x="340" y="65">oscillator</text>
    <line x1="20" y1="56" x2="59" y2="56" stroke="currentColor" stroke-width="1.1"/><text x="38" y="48">in</text>
    <line x1="130" y1="56" x2="179" y2="56" stroke="currentColor" stroke-width="1.1" marker-end="url(#clar)"/>
    <line x1="250" y1="56" x2="299" y2="56" stroke="currentColor" stroke-width="1.1" marker-end="url(#clar)"/>
    <path d="M340 72 V 105 H 95 V 73" fill="none" stroke="currentColor" stroke-width="1.1" marker-end="url(#clar)"/>
  </g>
  <defs><marker id="clar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A Costas loop recovers a PSK carrier by feeding a phase-error estimate back to a controlled oscillator.</figcaption>
</figure>

## How it works

It compares in-phase and quadrature error to drive an oscillator that locks to the
carrier, removing residual frequency offset so the
[constellation](/reference/constellation-diagram/) stops rotating. It is named for John P.
Costas.

## Relevance to SDR

Carrier recovery via a Costas loop is essential to decoding phase-modulated systems and
to stabilising a constellation that would otherwise spin.
