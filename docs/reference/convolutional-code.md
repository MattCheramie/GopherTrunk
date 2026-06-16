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
  - { title: "The demodulation pipeline", url: /learn/demodulation-pipeline/ }
external:
  - { title: "Convolutional code (Wikipedia)", url: https://en.wikipedia.org/wiki/Convolutional_code }
---

A **convolutional code** is a [forward-error-correction](/reference/forward-error-correction/)
code in which each output depends on a sliding window of recent input bits, set by the
*constraint length*.

## How it works

The encoder adds structured redundancy; the receiver uses the
[Viterbi algorithm](/reference/viterbi-algorithm/) to find the most likely original
sequence. Puncturing can raise the code rate by omitting some output bits.

## Relevance to SDR

Convolutional coding (K=5) protects [M17](/reference/m17/) and appears in other digital
radio links to recover from bit errors.
