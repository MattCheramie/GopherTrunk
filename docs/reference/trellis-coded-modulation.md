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

## How it works

The encoder constrains which symbol sequences are valid; the receiver uses the
[Viterbi algorithm](/reference/viterbi-algorithm/) over the resulting trellis to pick the
most likely sequence. [P25](/reference/project-25/) applies a trellis code to parts of its
data.

## Relevance to SDR

TCM improves decoding robustness on the protocols that use it, recovering data at lower
SNR than uncoded modulation.
