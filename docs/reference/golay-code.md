---
slug: golay-code
title: Golay code
entry_type: algorithm
category: error-correction
description: The Golay code is a highly efficient perfect binary block code; the (23,12) Golay and extended (24,12) Golay correct up to three bit errors and protect control fields in P25, DMR, and M17.
keywords: Golay code, Golay(23,12), Golay(24,12), perfect code, extended Golay, three-error correcting, block code, P25, DMR, M17
aka: [Golay code, binary Golay code]
autolink: true
infobox:
  - { label: Type, value: Perfect binary block code }
  - { label: Corrects, value: Up to 3 errors (binary Golay) }
  - { label: Used by, value: P25, DMR, M17 }
see_also: [hamming-code, reed-muller-code, reed-solomon-code, forward-error-correction, project-25, m17]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Binary_Golay_code
  - https://en.wikipedia.org/wiki/Perfect_code
---

The **Golay code** is a remarkably efficient binary block
[error-correction](/reference/forward-error-correction/) code.[^wiki] The binary
**Golay(23,12)** packs 12 data bits into a 23-bit codeword and corrects up to three bit
errors, and the **extended Golay(24,12)** adds one overall parity bit to reach a convenient
24-bit word that can additionally *detect* a fourth error. It is prized for squeezing strong
correction into a very short block, which is exactly what small but critical control fields
need.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="A Golay (24,12) codeword: twelve data bits and twelve parity bits of equal size." xmlns="http://www.w3.org/2000/svg">
  <rect x="60" y="40" width="170" height="30" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="145" y="59" text-anchor="middle" font-size="9" fill="currentColor">12 data bits</text>
  <rect x="230" y="40" width="170" height="30" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.2"/><text x="315" y="59" text-anchor="middle" font-size="9" fill="currentColor">12 parity bits</text>
  <text x="230" y="90" text-anchor="middle" font-size="9" fill="currentColor">Golay(24,12) corrects up to 3 bit errors</text>
</svg>
<figcaption>The Golay(24,12) code protects critical fields (e.g. in DMR, P25, and M17) against several bit errors.</figcaption>
</figure>

## How it works

The binary Golay(23,12) is one of the very few **perfect codes**: its 2¹² codewords are
arranged so that the Hamming spheres of radius 3 around them exactly tile the whole 23-bit
space with no gaps and no overlap. That perfect packing is what guarantees every received
word with three or fewer errors sits closest to exactly one codeword, so correction is
unambiguous and no redundancy is wasted. Encoding multiplies the 12 data bits by a generator
matrix; decoding computes a 11-bit syndrome and, because the code is so small and highly
structured, the error pattern can be looked up directly or found with a compact algebraic
procedure — fast enough to run on the tiniest processors. The extended (24,12) form trades
the perfect property for a tidy byte-aligned length and the bonus of four-error detection.

## Variants

Golay sits in the same family of classic linear block codes as the
[Hamming codes](/reference/hamming-code/) (perfect single-error correctors) and the
[Reed–Muller codes](/reference/reed-muller-code/); in fact the extended Golay is closely
related to a shortened Reed–Muller construction and to the mathematics of the Leech lattice.
Where [Reed–Solomon](/reference/reed-solomon-code/) shines on long bursts, Golay is the tool
of choice for a *short* field that must survive a handful of scattered bit errors.

## Relevance to SDR

Golay coding protects link-control and signalling fields across the digital voice systems
GopherTrunk decodes. [P25](/reference/project-25/) uses Golay codes on parts of its Network
ID and link-control words, [DMR](/reference/dmr/) protects several control fields with
Golay(20,8) and (24,12) variants, and [M17](/reference/m17/) uses Golay for its link setup.
Correcting these words is often what makes the difference between reading a call's metadata
and losing the frame entirely, so the decoder leans on Golay heavily on marginal signals.

## Sources

[^wiki]: [Binary Golay code](https://en.wikipedia.org/wiki/Binary_Golay_code) — Wikipedia, for the perfect (23,12) and extended (24,12) three-error-correcting codes. See also [Perfect code](https://en.wikipedia.org/wiki/Perfect_code) for the sphere-packing property.
