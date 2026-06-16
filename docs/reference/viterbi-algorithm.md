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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A trellis of states over time with many candidate paths and one highlighted most-likely surviving path." xmlns="http://www.w3.org/2000/svg">
  <g fill="currentColor"><circle cx="40" cy="40" r="3"/><circle cx="40" cy="100" r="3"/><circle cx="150" cy="40" r="3"/><circle cx="150" cy="100" r="3"/><circle cx="260" cy="40" r="3"/><circle cx="260" cy="100" r="3"/><circle cx="370" cy="40" r="3"/><circle cx="370" cy="100" r="3"/></g>
  <g stroke="currentColor" stroke-opacity="0.3" stroke-width="1"><line x1="40" y1="40" x2="150" y2="40"/><line x1="40" y1="40" x2="150" y2="100"/><line x1="40" y1="100" x2="150" y2="40"/><line x1="40" y1="100" x2="150" y2="100"/><line x1="150" y1="40" x2="260" y2="40"/><line x1="150" y1="40" x2="260" y2="100"/><line x1="150" y1="100" x2="260" y2="40"/><line x1="150" y1="100" x2="260" y2="100"/><line x1="260" y1="40" x2="370" y2="40"/><line x1="260" y1="40" x2="370" y2="100"/><line x1="260" y1="100" x2="370" y2="40"/><line x1="260" y1="100" x2="370" y2="100"/></g>
  <polyline points="40,100 150,40 260,40 370,100" fill="none" stroke="currentColor" stroke-width="2.4"/>
  <text x="205" y="130" text-anchor="middle" font-size="9" fill="currentColor">most-likely path through the trellis</text>
</svg>
<figcaption>The Viterbi algorithm finds the most-likely sequence through a trellis, decoding convolutional codes.</figcaption>
</figure>

## How it works

By keeping only the best path into each trellis state at each step, it avoids an
exponential search while achieving maximum-likelihood decoding — recovering the
transmitted bits even with errors.

## Relevance to SDR

Viterbi decoding is used in error-corrected digital systems such as
[M17](/reference/m17/) and various trunked-radio components to drive down the error rate.
