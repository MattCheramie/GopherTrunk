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
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
external:
  - { title: "Hamming code (Wikipedia)", url: https://en.wikipedia.org/wiki/Hamming_code }
---

**Hamming codes** are simple block [error-correction](/reference/forward-error-correction/)
codes that correct **single-bit** errors (and detect two) using a handful of parity-check
bits. They are named for [Richard Hamming](/reference/richard-hamming/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="Data bits interspersed with parity bits, with brackets showing each parity bit covering a set of data bits." xmlns="http://www.w3.org/2000/svg">
  <g font-family="monospace" font-size="11" fill="currentColor" text-anchor="middle">
    <text x="60" y="55">P</text><text x="100" y="55">P</text><text x="140" y="55">D</text><text x="180" y="55">P</text><text x="220" y="55">D</text><text x="260" y="55">D</text><text x="300" y="55">D</text>
  </g>
  <path d="M60 65 q40 18 80 0" fill="none" stroke="currentColor" stroke-opacity="0.6"/>
  <path d="M100 70 q60 22 120 0" fill="none" stroke="currentColor" stroke-opacity="0.6"/>
  <text x="230" y="100" text-anchor="middle" font-size="9" fill="currentColor">each parity bit (P) checks a set of data bits (D)</text>
</svg>
<figcaption>A Hamming code interleaves parity bits that locate and correct a single bit error per block.</figcaption>
</figure>

## How it works

Parity bits cover overlapping subsets of the data so that the pattern of failed checks (the
syndrome) points directly at the erroneous bit. Variants such as Hamming(20,8) appear in
digital radio link control.

## Relevance to SDR

Hamming coding protects small control fields in several digital protocols, contributing to
robust decoding.
