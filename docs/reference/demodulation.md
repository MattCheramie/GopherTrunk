---
slug: demodulation
title: Demodulation
entry_type: term
category: sdr-dsp
description: Demodulation recovers the original modulating information from a carrier; for digital signals it produces the symbol stream that decoding then turns into bits.
keywords: demodulation, demodulator, FM PSK FSK, symbol recovery, decoding, pipeline
aka: [demodulation]
autolink: true
infobox:
  - { label: Type, value: DSP stage }
  - { label: Recovers, value: Modulating signal from carrier }
  - { label: Followed by, value: Symbol recovery, decoding }
see_also: [modulation, clock-recovery, costas-loop, constellation-diagram, software-defined-radio]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/demodulation-pipeline/ }
external:
  - { title: "Demodulation (Wikipedia)", url: https://en.wikipedia.org/wiki/Demodulation }
---

**Demodulation** recovers the original modulating information from a
[carrier](/reference/carrier-wave/) — the inverse of
[modulation](/reference/modulation/). For FM/[FSK](/reference/frequency-shift-keying/) it
tracks instantaneous frequency; for [PSK](/reference/phase-shift-keying/) it tracks
[phase](/reference/phase/).

## How it works

The demodulator outputs a continuous, noisy stream that *contains* the
[symbols](/reference/symbol-rate/); [clock recovery](/reference/clock-recovery/) then
slices it into discrete symbols, which decoding turns into bits. Demodulation handles the
waveform; decoding handles the data.

## Relevance to SDR

Choosing the matching demodulator for a signal's modulation is the core of recovering it;
the [constellation](/reference/constellation-diagram/) visualises this stage.
