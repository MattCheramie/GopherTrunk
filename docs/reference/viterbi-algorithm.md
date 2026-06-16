---
slug: viterbi-algorithm
title: Viterbi algorithm
entry_type: algorithm
category: algorithms
description: The Viterbi algorithm efficiently finds the most likely sequence of states in a trellis, used to decode convolutional codes and improve digital reception.
keywords: Viterbi algorithm, maximum likelihood, trellis, convolutional code, Andrew Viterbi, decoding
aka: [Viterbi algorithm, Viterbi]
autolink: true
infobox:
  - { label: Type, value: Maximum-likelihood decoder }
  - { label: Decodes, value: Convolutional / trellis codes }
  - { label: Named for, value: Andrew Viterbi }
see_also: [convolutional-code, trellis-coded-modulation, forward-error-correction, andrew-viterbi, m17]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/demodulation-pipeline/ }
external:
  - { title: "Viterbi algorithm (Wikipedia)", url: https://en.wikipedia.org/wiki/Viterbi_algorithm }
---

The **Viterbi algorithm** efficiently finds the most likely sequence of states through a
trellis, given noisy observations. It is the standard way to decode
[convolutional codes](/reference/convolutional-code/), named for
[Andrew Viterbi](/reference/andrew-viterbi/).

## How it works

By keeping only the best path into each trellis state at each step, it avoids an
exponential search while achieving maximum-likelihood decoding — recovering the
transmitted bits even with errors.

## Relevance to SDR

Viterbi decoding is used in error-corrected digital systems such as
[M17](/reference/m17/) and various trunked-radio components to drive down the error rate.
