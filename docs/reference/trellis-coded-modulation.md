---
slug: trellis-coded-modulation
title: Trellis-coded modulation
entry_type: algorithm
category: algorithms
description: Trellis-coded modulation combines convolutional coding with modulation so error correction is built into the symbol mapping; P25 uses a trellis code on parts of its data.
keywords: trellis coded modulation, TCM, convolutional, Viterbi, P25, coding gain
aka: [trellis-coded modulation, TCM]
autolink: true
infobox:
  - { label: Type, value: Coded-modulation scheme }
  - { label: Combines, value: Coding + modulation mapping }
  - { label: Decoded by, value: Viterbi over a trellis }
see_also: [convolutional-code, viterbi-algorithm, forward-error-correction, project-25]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/demodulation-pipeline/ }
external:
  - { title: "Trellis modulation (Wikipedia)", url: https://en.wikipedia.org/wiki/Trellis_modulation }
---

**Trellis-coded modulation** (**TCM**) integrates
[convolutional coding](/reference/convolutional-code/) with the modulation symbol mapping,
so coding gain is achieved without extra bandwidth.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A small trellis whose transitions map onto points of a constellation, combining coding and modulation." xmlns="http://www.w3.org/2000/svg">
  <g fill="currentColor"><circle cx="40" cy="40" r="3"/><circle cx="40" cy="90" r="3"/><circle cx="150" cy="40" r="3"/><circle cx="150" cy="90" r="3"/></g>
  <g stroke="currentColor" stroke-opacity="0.5"><line x1="40" y1="40" x2="150" y2="40"/><line x1="40" y1="40" x2="150" y2="90"/><line x1="40" y1="90" x2="150" y2="40"/><line x1="40" y1="90" x2="150" y2="90"/></g>
  <text x="95" y="115" text-anchor="middle" font-size="8" fill="currentColor">trellis</text>
  <line x1="175" y1="65" x2="215" y2="65" stroke="currentColor" marker-end="url(#tcmar)"/>
  <line x1="240" y1="65" x2="430" y2="65" stroke="currentColor" stroke-opacity="0.3"/><line x1="335" y1="25" x2="335" y2="105" stroke="currentColor" stroke-opacity="0.3"/>
  <g fill="currentColor"><circle cx="300" cy="45" r="3"/><circle cx="370" cy="45" r="3"/><circle cx="300" cy="90" r="3"/><circle cx="370" cy="90" r="3"/></g>
  <text x="335" y="120" text-anchor="middle" font-size="8" fill="currentColor">constellation</text>
  <defs><marker id="tcmar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Trellis-coded modulation combines error coding with the symbol mapping; P25 uses it on Phase 1 traffic.</figcaption>
</figure>

## How it works

The encoder constrains which symbol sequences are valid; the receiver uses the
[Viterbi algorithm](/reference/viterbi-algorithm/) over the resulting trellis to pick the
most likely sequence. [P25](/reference/project-25/) applies a trellis code to parts of its
data.

## Relevance to SDR

TCM improves decoding robustness on the protocols that use it, recovering data at lower
SNR than uncoded modulation.
