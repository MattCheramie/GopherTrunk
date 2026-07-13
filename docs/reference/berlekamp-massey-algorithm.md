---
slug: berlekamp-massey-algorithm
title: Berlekamp–Massey algorithm (BM)
entry_type: algorithm
category: error-correction
description: Berlekamp–Massey finds the shortest LFSR (error-locator polynomial) that generates a syndrome sequence, forming the algebraic core of Reed–Solomon and BCH decoding.
keywords: Berlekamp-Massey algorithm, BM algorithm, error-locator polynomial, shortest LFSR, syndrome, Reed-Solomon decoding, BCH decoding, linear feedback shift register
aka: [Berlekamp–Massey, BM algorithm, Berlekamp-Massey]
autolink: true
infobox:
  - { label: Type, value: Algebraic decoding step }
  - { label: Finds, value: Error-locator polynomial (shortest LFSR) }
  - { label: Used by, value: Reed–Solomon, BCH decoders }
see_also: [reed-solomon-code, bch-code, chien-search, forney-algorithm, linear-feedback-shift-register, forward-error-correction]
cite_urls:
  - https://en.wikipedia.org/wiki/Berlekamp%E2%80%93Massey_algorithm
  - https://ieeexplore.ieee.org/document/1054260
---

The **Berlekamp–Massey algorithm** finds the shortest [linear-feedback shift register](/reference/linear-feedback-shift-register/)
(equivalently, the lowest-degree polynomial) that produces a given sequence of field
elements.[^wiki] In algebraic decoding it is the step that turns a block of computed
**syndromes** into the **error-locator polynomial**, the object whose roots reveal where a
[Reed–Solomon](/reference/reed-solomon-code/) or [BCH](/reference/bch-code/) codeword was
damaged.[^massey] It is fast, exact, and works entirely in the code's finite field.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Syndromes feed the Berlekamp–Massey algorithm, which iteratively grows the shortest LFSR and outputs the error-locator polynomial passed to Chien search and Forney." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="bmar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <rect x="20" y="45" width="90" height="34"/>
    <rect x="165" y="45" width="120" height="34"/>
    <rect x="340" y="45" width="100" height="34"/>
  </g>
  <text x="65" y="66" text-anchor="middle" font-size="10" fill="currentColor">syndromes Sₖ</text>
  <text x="225" y="60" text-anchor="middle" font-size="10" fill="currentColor">Berlekamp–Massey</text>
  <text x="225" y="73" text-anchor="middle" font-size="9" fill="currentColor">grow shortest LFSR</text>
  <text x="390" y="60" text-anchor="middle" font-size="10" fill="currentColor">Λ(x)</text>
  <text x="390" y="73" text-anchor="middle" font-size="9" fill="currentColor">error locator</text>
  <g stroke="currentColor" stroke-width="1.4"><line x1="110" y1="62" x2="163" y2="62" marker-end="url(#bmar)"/><line x1="285" y1="62" x2="338" y2="62" marker-end="url(#bmar)"/></g>
  <path d="M225 45 C 225 20, 225 20, 225 20" fill="none"/>
  <text x="225" y="118" text-anchor="middle" font-size="9" fill="currentColor">each step: measure discrepancy, update Λ(x), extend register length only when forced</text>
</svg>
<figcaption>Berlekamp–Massey iteratively builds the shortest recurrence that fits the syndromes, yielding the error-locator polynomial Λ(x).</figcaption>
</figure>

## How it works

A received word is checked against the code's parity structure to produce a short list of
**syndromes** — numbers that are all zero when there are no errors and otherwise encode a
system of nonlinear equations in the unknown error positions and values. The classical way
to solve that system is to first find the **error-locator polynomial** Λ(x), whose roots
point at the corrupted symbols.

Berlekamp–Massey builds Λ(x) incrementally. It processes the syndromes one at a time and
maintains a current candidate LFSR:

- At each step it computes a **discrepancy**: how far the current register's prediction of
  the next syndrome is from the actual value.
- If the discrepancy is zero, the current polynomial still works — nothing changes.
- If not, it corrects Λ(x) using a saved copy of the last polynomial that forced a length
  increase, scaled by the discrepancy. Register length grows **only when unavoidable**,
  which is exactly what makes the result the *shortest* generator.

After all 2*t* syndromes are consumed (for a code that corrects *t* errors), the degree of
Λ(x) equals the number of errors that actually occurred, provided that number is within the
code's capability. The whole procedure is O(*t*²) in the field — dramatically cheaper than
solving the syndrome equations by brute force, and numerically exact because finite-field
arithmetic has no rounding.

## In practice

Berlekamp–Massey is one stage of a three-stage algebraic RS/BCH decoder, and it is almost
always followed by two companions:

- [Chien search](/reference/chien-search/) evaluates Λ(x) at every field element to find
  its roots, giving the **error positions**.
- The [Forney algorithm](/reference/forney-algorithm/) uses Λ(x) together with an
  error-evaluator polynomial to compute the **error values** at those positions.

An equivalent alternative to BM is the extended Euclidean algorithm applied to the syndrome
polynomial; the two produce the same locator and are chosen mostly by implementation
convenience. Massey's key insight was that "synthesise the shortest LFSR for a sequence" and
"decode a BCH code" are the *same* problem, which is why the algorithm is also a staple of
linear-complexity analysis and cryptanalysis of stream ciphers.

## Relevance to SDR

Reed–Solomon and BCH codes appear throughout the digital land-mobile radio world —
[P25](/reference/p25-phase-1/) protects control-channel and header fields with short RS
codes, and BCH guards sync and signalling words in several formats — so a hard-decision
algebraic decoder for those fields runs some equivalent of Berlekamp–Massey to locate errors
before correcting them. The algorithm is a building block of the
[forward error correction](/reference/forward-error-correction/) stage rather than something
a listener sees directly. GopherTrunk performs the FEC checks that these formats require to
validate frames; the BM/Chien/Forney trio is the standard math behind that class of block
code, whether implemented as a full solver or as a table-driven correction for the small
codes GT encounters.

## Sources

[^wiki]: [Berlekamp–Massey algorithm](https://en.wikipedia.org/wiki/Berlekamp%E2%80%93Massey_algorithm) — Wikipedia, for the shortest-LFSR formulation and its use in RS/BCH decoding.
[^massey]: [Shift-register synthesis and BCH decoding](https://ieeexplore.ieee.org/document/1054260) — J. L. Massey, IEEE Trans. Information Theory (1969), the paper linking LFSR synthesis to error correction.
