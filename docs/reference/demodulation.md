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
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
related_reading:
  - { title: "SDR Internals, Part 6: Demodulation", url: /blog/deep-dives/sdr-internals-06-demodulation/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Demodulation
---

**Demodulation** recovers the original modulating information from a
[carrier](/reference/carrier-wave/) — the inverse of
[modulation](/reference/modulation/).[^wiki] For FM/[FSK](/reference/frequency-shift-keying/) it
tracks instantaneous frequency; for [PSK](/reference/phase-shift-keying/) it tracks
[phase](/reference/phase/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A modulated waveform entering a demodulator block and the recovered message waveform leaving it." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 55 q5 -16 10 0 q5 -22 10 0 q5 -16 10 0 q5 -8 10 0 q5 -16 10 0 q5 -22 10 0 q5 -16 10 0 q5 -8 10 0 q5 -16 10 0 q5 -22 10 0" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <rect x="200" y="38" width="74" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="237" y="59" font-size="9" fill="currentColor" text-anchor="middle">demod</text>
  <line x1="120" y1="55" x2="199" y2="55" stroke="currentColor" stroke-width="1.1"/>
  <path d="M290 55 Q 330 30 370 55 T 440 55" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <line x1="274" y1="55" x2="289" y2="55" stroke="currentColor" stroke-width="1.1" marker-end="url(#dmar)"/>
  <text x="365" y="92" font-size="9" fill="currentColor">recovered message</text>
  <defs><marker id="dmar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Demodulation recovers the original modulating signal from the carrier — the step before decoding bits.</figcaption>
</figure>

## How it works

The demodulator outputs a continuous, noisy stream that *contains* the
[symbols](/reference/symbol-rate/); [clock recovery](/reference/clock-recovery/) then
slices it into discrete symbols, which decoding turns into bits. Demodulation handles the
waveform; decoding handles the data.

## Relevance to SDR

Choosing the matching demodulator for a signal's modulation is the core of recovering it;
the [constellation](/reference/constellation-diagram/) visualises this stage.

## Sources

[^wiki]: [Demodulation](https://en.wikipedia.org/wiki/Demodulation) — Wikipedia, on recovering the modulating signal as the inverse of modulation.
