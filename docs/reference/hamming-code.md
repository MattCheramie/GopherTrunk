---
slug: hamming-code
title: Hamming code
entry_type: algorithm
category: algorithms
description: Hamming codes are simple block codes that correct single-bit errors using parity check bits; they are a foundational error-correction scheme named for Richard Hamming.
keywords: Hamming code, single error correction, parity bits, block code, Richard Hamming, FEC
aka: [Hamming code]
autolink: true
infobox:
  - { label: Type, value: Single-error-correcting block code }
  - { label: Uses, value: Parity check bits }
  - { label: Named for, value: Richard Hamming }
see_also: [forward-error-correction, golay-code, bch-code, richard-hamming, cyclic-redundancy-check]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/demodulation-pipeline/ }
external:
  - { title: "Hamming code (Wikipedia)", url: https://en.wikipedia.org/wiki/Hamming_code }
---

**Hamming codes** are simple block [error-correction](/reference/forward-error-correction/)
codes that correct **single-bit** errors (and detect two) using a handful of parity-check
bits. They are named for [Richard Hamming](/reference/richard-hamming/).

## How it works

Parity bits cover overlapping subsets of the data so that the pattern of failed checks (the
syndrome) points directly at the erroneous bit. Variants such as Hamming(20,8) appear in
digital radio link control.

## Relevance to SDR

Hamming coding protects small control fields in several digital protocols, contributing to
robust decoding.
