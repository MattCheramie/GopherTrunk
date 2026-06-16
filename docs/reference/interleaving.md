---
slug: interleaving
title: Interleaving
entry_type: algorithm
category: algorithms
description: Interleaving reorders bits or symbols before transmission so that a burst of errors is spread across multiple codewords, letting error-correction codes recover it.
keywords: interleaving, de-interleaving, burst error, FEC, fading, robustness
aka: [interleaving]
autolink: true
infobox:
  - { label: Type, value: Error-resilience technique }
  - { label: Spreads, value: Burst errors across codewords }
  - { label: Paired with, value: FEC codes }
see_also: [forward-error-correction, reed-solomon-code, bch-code, multipath-propagation]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/demodulation-pipeline/ }
external:
  - { title: "Interleaving (Wikipedia)", url: https://en.wikipedia.org/wiki/Interleaving_(data) }
---

**Interleaving** reorders bits or symbols before transmission and restores their order on
receive, so that a **burst** of errors (from fading or interference) is spread thinly
across many codewords rather than overwhelming one.

## How it works

Because [FEC](/reference/forward-error-correction/) codes correct only a limited number of
errors per codeword, distributing a burst lets each codeword's correction cope. The
receiver de-interleaves before decoding.

## Relevance to SDR

Interleaving makes digital radio robust to [multipath](/reference/multipath-propagation/)
fading and short interference bursts.
