---
slug: convolutional-code
title: Convolutional code
entry_type: algorithm
category: algorithms
description: A convolutional code is a forward-error-correction code that encodes data as a sliding function of recent input bits, typically decoded with the Viterbi algorithm.
keywords: convolutional code, FEC, constraint length, Viterbi, trellis, puncturing
aka: [convolutional code]
autolink: true
infobox:
  - { label: Type, value: Error-correction code }
  - { label: Encodes, value: Sliding window of input bits }
  - { label: Decoded by, value: Viterbi algorithm }
see_also: [viterbi-algorithm, forward-error-correction, trellis-coded-modulation, m17]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
external:
  - { title: "Convolutional code (Wikipedia)", url: https://en.wikipedia.org/wiki/Convolutional_code }
---

A **convolutional code** is a [forward-error-correction](/reference/forward-error-correction/)
code in which each output depends on a sliding window of recent input bits, set by the
*constraint length*.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A shift register of memory cells whose taps are XOR-combined to produce two output bits per input bit." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.3" fill="none"><rect x="60" y="40" width="40" height="30"/><rect x="110" y="40" width="40" height="30"/><rect x="160" y="40" width="40" height="30"/></g>
  <text x="35" y="59" font-size="9" fill="currentColor">in</text><line x1="44" y1="55" x2="59" y2="55" stroke="currentColor"/>
  <circle cx="260" cy="35" r="10" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="260" y="39" text-anchor="middle" font-size="10" fill="currentColor">⊕</text>
  <circle cx="260" cy="95" r="10" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="260" y="99" text-anchor="middle" font-size="10" fill="currentColor">⊕</text>
  <g stroke="currentColor" stroke-width="1" stroke-opacity="0.7"><line x1="80" y1="40" x2="80" y2="35" /><line x1="80" y1="35" x2="250" y2="35"/><line x1="180" y1="40" x2="180" y2="35"/><line x1="130" y1="70" x2="130" y2="95"/><line x1="130" y1="95" x2="250" y2="95"/><line x1="180" y1="70" x2="180" y2="95"/></g>
  <line x1="270" y1="35" x2="320" y2="35" stroke="currentColor"/><text x="330" y="39" font-size="9" fill="currentColor">out1</text>
  <line x1="270" y1="95" x2="320" y2="95" stroke="currentColor"/><text x="330" y="99" font-size="9" fill="currentColor">out2</text>
</svg>
<figcaption>A convolutional code outputs parity bits computed from the current and recent input bits via a shift register.</figcaption>
</figure>

## How it works

The encoder adds structured redundancy; the receiver uses the
[Viterbi algorithm](/reference/viterbi-algorithm/) to find the most likely original
sequence. Puncturing can raise the code rate by omitting some output bits.

## Relevance to SDR

Convolutional coding (K=5) protects [M17](/reference/m17/) and appears in other digital
radio links to recover from bit errors.
