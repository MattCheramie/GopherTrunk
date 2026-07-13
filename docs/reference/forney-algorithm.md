---
slug: forney-algorithm
title: Forney algorithm
entry_type: algorithm
category: error-correction
description: The Forney algorithm computes error magnitudes at known error positions using the error-evaluator polynomial, completing Reed–Solomon and BCH decoding.
keywords: Forney algorithm, error magnitude, error value, error-evaluator polynomial, Reed-Solomon decoding, BCH decoding, key equation, Omega polynomial
aka: [Forney algorithm, Forney's formula]
autolink: true
infobox:
  - { label: Type, value: Error-value computation step }
  - { label: Finds, value: Error magnitudes at known positions }
  - { label: Used by, value: Reed–Solomon (symbol) decoders }
see_also: [reed-solomon-code, bch-code, berlekamp-massey-algorithm, chien-search, forward-error-correction, interleaving]
cite_urls:
  - https://en.wikipedia.org/wiki/Forney_algorithm
  - https://ieeexplore.ieee.org/document/1054010
---

The **Forney algorithm** computes the **magnitude** of each error once its position is
known, finishing the correction of a [Reed–Solomon](/reference/reed-solomon-code/) or
[BCH](/reference/bch-code/) codeword.[^wiki] Where
[Chien search](/reference/chien-search/) answers *which* symbols are wrong, Forney answers
*by how much* — for each error position it evaluates a simple rational formula built from the
error-locator and error-evaluator polynomials, giving the exact value to subtract from the
received symbol.[^forney]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 155" role="img" aria-label="Error positions from Chien search plus the error-evaluator and locator-derivative polynomials feed Forney's formula, which outputs the error magnitude subtracted from each damaged symbol." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="fnar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <rect x="20" y="20" width="120" height="30"/>
    <rect x="20" y="70" width="120" height="30"/>
    <rect x="200" y="45" width="110" height="34"/>
    <rect x="370" y="45" width="80" height="34"/>
  </g>
  <text x="80" y="39" text-anchor="middle" font-size="9" fill="currentColor">Ω(x) evaluator</text>
  <text x="80" y="89" text-anchor="middle" font-size="9" fill="currentColor">Λ′(x) locator deriv.</text>
  <text x="255" y="60" text-anchor="middle" font-size="10" fill="currentColor">Forney:</text>
  <text x="255" y="73" text-anchor="middle" font-size="9" fill="currentColor">eᵢ = −Ω(X⁻¹)/Λ′(X⁻¹)</text>
  <text x="410" y="60" text-anchor="middle" font-size="9" fill="currentColor">error value</text>
  <text x="410" y="73" text-anchor="middle" font-size="9" fill="currentColor">eᵢ</text>
  <g stroke="currentColor" stroke-width="1.4">
    <line x1="140" y1="35" x2="198" y2="55" marker-end="url(#fnar)"/>
    <line x1="140" y1="85" x2="198" y2="68" marker-end="url(#fnar)"/>
    <line x1="310" y1="62" x2="368" y2="62" marker-end="url(#fnar)"/>
  </g>
  <text x="230" y="120" text-anchor="middle" font-size="9" fill="currentColor">evaluated once per error position Xᵢ found by Chien search</text>
</svg>
<figcaption>Forney's rational formula turns each known error position into an exact error value, using the evaluator polynomial over the locator's derivative.</figcaption>
</figure>

## How it works

Algebraic decoding factors the problem into "where" and "how much." The
[Berlekamp–Massey algorithm](/reference/berlekamp-massey-algorithm/) produces the
error-locator polynomial Λ(x); Chien search finds its roots to give the error positions Xᵢ.
That leaves the error values eᵢ still unknown. In principle you could set up and solve a
system of linear equations from the syndromes, but Forney found a closed-form shortcut.

The key is a second polynomial, the **error-evaluator** (or "omega") polynomial Ω(x),
obtained from the syndrome polynomial and Λ(x) via the *key equation*
Ω(x) = S(x)·Λ(x) mod x^(2t). Forney's formula then gives each error magnitude directly:

- Take the formal derivative Λ′(x) of the locator polynomial.
- For each error position Xᵢ, evaluate eᵢ = −Ω(Xᵢ⁻¹) / Λ′(Xᵢ⁻¹) (with a scaling factor that
  depends on the code's exact definition).
- Subtract eᵢ from the received symbol at that position to correct it.

Because everything is finite-field arithmetic, the division is a multiply by a precomputed
inverse and the result is exact — no rounding, no iteration. The cost is a couple of
polynomial evaluations per error, similar in structure to Chien search, so the two share
hardware in real decoders.

## In practice

Forney is the final stage of the classic **syndrome → Berlekamp–Massey → Chien → Forney**
pipeline. It is what distinguishes a *symbol* code like Reed–Solomon from a purely *binary*
BCH code: in a binary BCH code every error value is simply 1, so locating the error is enough
and Forney reduces to a no-op. In Reed–Solomon over a byte-sized field the value can be any
nonzero symbol, so Forney's magnitude computation is essential to actually repair the data.
The same three-stage flow underpins the erasure-and-error decoding used with
[interleaving](/reference/interleaving/) to survive burst damage: known erasure positions
are folded into Λ(x), and Forney still supplies the values.

## Relevance to SDR

Reed–Solomon protection appears in digital radio signalling — parts of
[P25](/reference/p25-phase-1/) header and control data, and the RS/BCH-guarded words in other
land-mobile formats — and in nearly all broadcast and storage systems a receiver might touch.
Whenever such a symbol-oriented code is corrected, some equivalent of Forney's formula
computes the repair values; it is an internal step of the
[forward error correction](/reference/forward-error-correction/) stage, not something a user
observes. GopherTrunk performs the FEC these formats mandate to validate frames; Forney names
the standard method for the "how much" half of block-code correction, whether run as a full
solver or, for the very short codes GT meets, a table-driven equivalent.

## Sources

[^wiki]: [Forney algorithm](https://en.wikipedia.org/wiki/Forney_algorithm) — Wikipedia, for the error-value formula and the error-evaluator polynomial.
[^forney]: [On decoding BCH codes](https://ieeexplore.ieee.org/document/1054010) — G. D. Forney, IEEE Trans. Information Theory (1965), the original derivation.
