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
  - { title: "Tuning for a clean lock", url: /learn/tuning-with-scopes/ }
external:
  - { title: "Costas loop (Wikipedia)", url: https://en.wikipedia.org/wiki/Costas_loop }
---

A **Costas loop** is a phase-locked feedback structure that recovers the
[phase](/reference/phase/) and frequency of a suppressed carrier, enabling **coherent**
[demodulation](/reference/demodulation/) of [PSK](/reference/phase-shift-keying/) and
related signals.

## How it works

It compares in-phase and quadrature error to drive an oscillator that locks to the
carrier, removing residual frequency offset so the
[constellation](/reference/constellation-diagram/) stops rotating. It is named for John P.
Costas.

## Relevance to SDR

Carrier recovery via a Costas loop is essential to decoding phase-modulated systems and
to stabilising a constellation that would otherwise spin.
