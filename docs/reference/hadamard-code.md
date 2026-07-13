---
slug: hadamard-code
title: Hadamard code (Walsh–Hadamard)
entry_type: algorithm
category: error-correction
description: A Hadamard code uses the rows of a Hadamard matrix as codewords, giving enormous minimum distance at very low rate; its Walsh codes provide the orthogonal spreading sequences of CDMA.
keywords: Hadamard code, Walsh code, Walsh-Hadamard, Hadamard matrix, orthogonal codes, fast Walsh-Hadamard transform, CDMA spreading, biorthogonal, minimum distance, deep space
aka: [Hadamard code, Walsh–Hadamard code, Walsh code, Walsh function]
autolink: true
infobox:
  - { label: Type, value: Linear block code / orthogonal set }
  - { label: Min distance, value: n/2 (very strong) }
  - { label: Used by, value: CDMA Walsh spreading, deep space }
see_also: [reed-muller-code, cdma, forward-error-correction, maximal-length-sequence, direct-sequence-spread-spectrum, gold-code]
cite_urls:
  - https://en.wikipedia.org/wiki/Hadamard_code
  - https://en.wikipedia.org/wiki/Walsh_function
---

A **Hadamard code** takes the rows of a **Hadamard matrix** — a square ±1 matrix whose
rows are mutually orthogonal — and uses them (mapped to 0/1) as its
[codewords](/reference/forward-error-correction/).[^wiki] The payoff is a huge
**minimum distance of `n/2`** for length `n`, meaning any two codewords differ in half
their bits, at the cost of a very **low rate**: only `log₂ n` message bits per `n`
transmitted bits. In its signal-processing guise the same rows are the **Walsh
functions**, and the orthogonal **Walsh codes** they define are what let many
[CDMA](/reference/cdma/) users share one channel without interfering.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Recursive Sylvester construction of a Hadamard matrix, where an H matrix expands into a two-by-two block pattern of H and negated H, and a small four-by-four sign checkerboard whose rows are the orthogonal Walsh codewords." xmlns="http://www.w3.org/2000/svg">
  <g font-size="12" fill="currentColor">
    <g stroke="currentColor" fill="none" stroke-width="1.1"><rect x="30" y="45" width="60" height="60"/></g>
    <line x1="60" y1="45" x2="60" y2="105" stroke="currentColor" stroke-width="0.6" stroke-opacity="0.5"/>
    <line x1="30" y1="75" x2="90" y2="75" stroke="currentColor" stroke-width="0.6" stroke-opacity="0.5"/>
    <text x="45" y="68" text-anchor="middle">H</text><text x="75" y="68" text-anchor="middle">H</text>
    <text x="45" y="98" text-anchor="middle">H</text><text x="75" y="98" text-anchor="middle">−H</text>
    <text x="120" y="80" font-size="16">→</text>
  </g>
  <g stroke="currentColor" stroke-width="0.8">
    <g fill="currentColor">
      <rect x="160" y="45" width="16" height="16"/><rect x="176" y="45" width="16" height="16" fill="none"/><rect x="192" y="45" width="16" height="16"/><rect x="208" y="45" width="16" height="16" fill="none"/>
      <rect x="160" y="61" width="16" height="16"/><rect x="176" y="61" width="16" height="16"/><rect x="192" y="61" width="16" height="16" fill="none"/><rect x="208" y="61" width="16" height="16" fill="none"/>
      <rect x="160" y="77" width="16" height="16"/><rect x="176" y="77" width="16" height="16" fill="none"/><rect x="192" y="77" width="16" height="16" fill="none"/><rect x="208" y="77" width="16" height="16"/>
      <rect x="160" y="93" width="16" height="16"/><rect x="176" y="93" width="16" height="16"/><rect x="192" y="93" width="16" height="16"/><rect x="208" y="93" width="16" height="16"/>
    </g>
    <rect x="160" y="45" width="64" height="64" fill="none"/>
  </g>
  <text x="192" y="125" text-anchor="middle" font-size="9" fill="currentColor">each row = one orthogonal Walsh codeword</text>
  <text x="330" y="70" font-size="9" fill="currentColor">any two rows agree in</text>
  <text x="330" y="84" font-size="9" fill="currentColor">exactly half their bits</text>
  <text x="330" y="98" font-size="9" fill="currentColor">→ minimum distance n/2</text>
</svg>
<figcaption>Doubling a Hadamard matrix by the Sylvester rule yields a sign checkerboard whose orthogonal rows are the Walsh codewords underpinning CDMA and low-rate deep-space codes.</figcaption>
</figure>

## How it works

The simplest Hadamard matrices are built by **Sylvester's recursion**: start from
`H₁ = [1]` and repeatedly form `[[H, H], [H, −H]]`, doubling the size each step to
`n = 2^m`. Every pair of rows is orthogonal — their dot product is zero — which in
0/1 terms means they differ in exactly `n/2` positions. Encoding maps a `k`-bit
message (`k = log₂ n`) to the corresponding matrix row.

Decoding is where Hadamard codes shine. Rather than compare the received word to each
codeword one at a time, the receiver runs a **fast Walsh–Hadamard transform** — the
same butterfly structure as an FFT but with only additions and subtractions — which
correlates against *all* `n` codewords simultaneously in `O(n log n)`. The largest
transform coefficient names the most likely codeword. This makes an otherwise
brute-force nearest-codeword search cheap, and it is exactly the soft-decision decoder
used for the biorthogonal codes in deep-space links.

## Variants

- **Augmented / biorthogonal code**: adding the complements of the Hadamard rows gives
  `2n` codewords of length `n`, which is precisely the
  [Reed–Muller](/reference/reed-muller-code/) code RM(1, m). This is the form flown on
  Mariner deep-space missions.
- **Walsh codes**: in [CDMA](/reference/cdma/) the rows are used not as an
  error-correcting code but as **orthogonal spreading sequences** — each user is
  assigned a distinct Walsh row so that, under perfect timing, their signals separate
  cleanly at the receiver. Walsh codes provide orthogonality but poor autocorrelation,
  so they are layered over a [maximal-length](/reference/maximal-length-sequence/) or
  [Gold](/reference/gold-code/) scrambling code that supplies the
  [spread-spectrum](/reference/direct-sequence-spread-spectrum/) timing properties.

## Relevance to SDR

Hadamard/Walsh sequences are everywhere in **spread-spectrum radio**: IS-95 and CDMA2000
use 64-ary Walsh codes to separate forward-link channels, WCDMA uses variable-length
orthogonal (OVSF) Walsh codes for channelisation, and GPS-style systems mix Walsh-like
orthogonality with pseudorandom codes. As error-correcting codes, the low-rate
biorthogonal Hadamard codes belong to the deep-space and beacon world where coding gain
outweighs bandwidth. **GopherTrunk** targets narrowband land-mobile trunking (P25, DMR,
NXDN, TETRA), which does not use Walsh spreading, so GT does not implement a Hadamard
decoder — but the fast Walsh–Hadamard transform is a general SDR tool, useful for
sequence correlation and for understanding how [CDMA](/reference/cdma/) packs many
callers onto one frequency.

## Sources

[^wiki]: [Hadamard code](https://en.wikipedia.org/wiki/Hadamard_code) — Wikipedia, for the Hadamard-matrix construction, minimum distance n/2, the fast Walsh–Hadamard decoder, the biorthogonal/Reed–Muller link, and the Walsh-code CDMA application.
