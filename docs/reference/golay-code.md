---
slug: golay-code
title: Golay code
entry_type: algorithm
category: algorithms
description: The Golay code is a highly efficient perfect/near-perfect block code; the binary Golay(23,12) and extended Golay(24,12) correct up to three errors and appear in DMR, P25, and M17.
keywords: Golay code, Golay(23,12), Golay(24,12), block code, three-error correcting, DMR, M17
aka: [Golay code]
autolink: true
infobox:
  - { label: Type, value: Perfect/near-perfect block code }
  - { label: Corrects, value: Up to 3 errors (binary Golay) }
  - { label: Used by, value: DMR, P25, M17 }
see_also: [forward-error-correction, hamming-code, reed-solomon-code, dmr, m17]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
external:
  - { title: "Binary Golay code (Wikipedia)", url: https://en.wikipedia.org/wiki/Binary_Golay_code }
---

The **Golay code** is a remarkably efficient block
[error-correction](/reference/forward-error-correction/) code. The binary Golay(23,12)
and extended Golay(24,12) correct up to three bit errors in a short codeword.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="A Golay (24,12) codeword: twelve data bits and twelve parity bits of equal size." xmlns="http://www.w3.org/2000/svg">
  <rect x="60" y="40" width="170" height="30" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="145" y="59" text-anchor="middle" font-size="9" fill="currentColor">12 data bits</text>
  <rect x="230" y="40" width="170" height="30" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.2"/><text x="315" y="59" text-anchor="middle" font-size="9" fill="currentColor">12 parity bits</text>
  <text x="230" y="90" text-anchor="middle" font-size="9" fill="currentColor">Golay(24,12) corrects up to 3 bit errors</text>
</svg>
<figcaption>The Golay(24,12) code protects critical fields (e.g. in DMR, P25, and M17) against several bit errors.</figcaption>
</figure>

## How it works

Its mathematical structure (a perfect code in the binary case) packs strong correction
into few bits, making it ideal for small but critical control fields.

## Relevance to SDR

Golay coding protects link-control and signalling fields in [DMR](/reference/dmr/),
[P25](/reference/project-25/), and [M17](/reference/m17/).
