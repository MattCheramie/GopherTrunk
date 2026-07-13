---
slug: reed-solomon-code
title: Reed–Solomon code
entry_type: algorithm
category: error-correction
description: Reed–Solomon is a block error-correction code that operates on multi-bit symbols over a Galois field, excelling at correcting bursts of errors; used in P25, DMR, storage, and broadcast.
keywords: Reed-Solomon, RS code, block code, burst error, symbols, Galois field, GF(2^m), syndrome, Berlekamp-Massey, Chien search, Forney, interleaving, FEC
aka: [Reed–Solomon code, Reed-Solomon, RS code]
autolink: true
infobox:
  - { label: Type, value: Block error-correction code }
  - { label: Operates on, value: Symbols over GF(2^m) }
  - { label: Corrects, value: Up to (n−k)/2 symbol errors }
see_also: [berlekamp-massey-algorithm, chien-search, forney-algorithm, bch-code, interleaving, forward-error-correction]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Reed%E2%80%93Solomon_error_correction
  - https://en.wikipedia.org/wiki/BCH_code
---

**Reed–Solomon** (RS) is a block [error-correction](/reference/forward-error-correction/)
code that works on multi-bit **symbols** rather than individual bits, which makes it
especially good at correcting **bursts** of errors — a run of adjacent bad bits usually
damages only a symbol or two.[^wiki] An RS code is written **(n, k)** over the Galois field
GF(2^m): it takes k data symbols, appends n−k parity symbols, and can correct up to
t = (n−k)/2 symbol errors anywhere in the codeword.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A codeword of data symbols followed by parity symbols, with the parity able to correct a burst of damaged symbols." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1">
    <rect x="30" y="40" width="34" height="30" fill="none"/><rect x="64" y="40" width="34" height="30" fill="none"/><rect x="98" y="40" width="34" height="30" fill="currentColor" fill-opacity="0.4"/><rect x="132" y="40" width="34" height="30" fill="currentColor" fill-opacity="0.4"/><rect x="166" y="40" width="34" height="30" fill="none"/><rect x="200" y="40" width="34" height="30" fill="none"/>
    <rect x="250" y="40" width="34" height="30" fill="currentColor" fill-opacity="0.15"/><rect x="284" y="40" width="34" height="30" fill="currentColor" fill-opacity="0.15"/><rect x="318" y="40" width="34" height="30" fill="currentColor" fill-opacity="0.15"/>
  </g>
  <text x="132" y="90" text-anchor="middle" font-size="9" fill="currentColor">data symbols (shaded = damaged)</text>
  <text x="301" y="90" text-anchor="middle" font-size="9" fill="currentColor">parity</text>
</svg>
<figcaption>Reed–Solomon adds parity symbols that correct bursts of damaged symbols — used across P25 and DMR.</figcaption>
</figure>

## How it works

Reed–Solomon treats each block of m bits as one element of the finite field GF(2^m) and a
codeword as a polynomial over that field. The parity symbols are chosen so that a valid
codeword is divisible by a generator polynomial whose roots are consecutive powers of a
primitive element; that structure lets the decoder detect and locate errors purely
algebraically. Classic hard-decision decoding runs in three stages, each of which has its
own page:

1. **Syndromes → error locator.** The decoder evaluates the received polynomial at the code's
   roots to get syndromes, then solves for the *error-locator polynomial* using the
   [Berlekamp–Massey algorithm](/reference/berlekamp-massey-algorithm/) (or the equivalent
   Euclidean method).
2. **Find the error positions.** [Chien search](/reference/chien-search/) evaluates the
   locator polynomial at every field element; its roots point to which symbols are wrong.
3. **Find the error values.** [Forney's algorithm](/reference/forney-algorithm/) then
   computes the magnitude to XOR into each located symbol, completing the correction.

Because the whole pipeline is symbol-oriented, a burst that corrupts many consecutive bits
still counts as only a handful of symbol errors — the source of RS's famous burst
resilience.

## In practice

RS is almost always paired with [interleaving](/reference/interleaving/): the encoder spreads
the symbols of each codeword across time (or across a data frame) so that a long fade or
dropout damages only a few symbols in any single codeword instead of wiping one out
entirely. Concatenating an inner convolutional/Viterbi stage with an outer RS code — the
scheme used by CDs, DVB, and deep-space links — is a classic way to clean up the residual
burst errors the inner decoder leaves behind. Reed–Solomon is also the mathematical cousin
of the [BCH codes](/reference/bch-code/); both are built on the same Galois-field
machinery, and RS is essentially a non-binary BCH code.

## Relevance to SDR

Reed–Solomon protects signalling and data fields in [P25](/reference/project-25/) and
[DMR](/reference/dmr/), and it is ubiquitous in storage (CDs, DVDs, QR codes) and broadcast
(DVB, ATSC). In the scanner path, RS decoding helps GopherTrunk recover trunking control
messages and data payloads on marginal signals where several symbols per block arrive
corrupted.

## Sources

[^wiki]: [Reed–Solomon error correction](https://en.wikipedia.org/wiki/Reed%E2%80%93Solomon_error_correction) — Wikipedia, for the symbol-oriented block code, Galois-field construction, and burst-error correction. See also [BCH code](https://en.wikipedia.org/wiki/BCH_code) for the shared algebraic decoding theory.
