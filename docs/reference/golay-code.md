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
  - { title: "The demodulation pipeline", url: /learn/demodulation-pipeline/ }
external:
  - { title: "Binary Golay code (Wikipedia)", url: https://en.wikipedia.org/wiki/Binary_Golay_code }
---

The **Golay code** is a remarkably efficient block
[error-correction](/reference/forward-error-correction/) code. The binary Golay(23,12)
and extended Golay(24,12) correct up to three bit errors in a short codeword.

## How it works

Its mathematical structure (a perfect code in the binary case) packs strong correction
into few bits, making it ideal for small but critical control fields.

## Relevance to SDR

Golay coding protects link-control and signalling fields in [DMR](/reference/dmr/),
[P25](/reference/project-25/), and [M17](/reference/m17/).
