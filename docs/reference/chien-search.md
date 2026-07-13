---
slug: chien-search
title: Chien search
entry_type: algorithm
category: error-correction
description: Chien search evaluates the error-locator polynomial at every finite-field element to find its roots, revealing the error positions in Reed–Solomon and BCH decoding.
keywords: Chien search, error-locator polynomial roots, error positions, Reed-Solomon decoding, BCH decoding, finite field, Galois field evaluation
aka: [Chien search, Chien's search]
autolink: true
infobox:
  - { label: Type, value: Root-finding decoding step }
  - { label: Finds, value: Error positions (roots of Λ(x)) }
  - { label: Used by, value: Reed–Solomon, BCH decoders }
see_also: [reed-solomon-code, bch-code, berlekamp-massey-algorithm, forney-algorithm, linear-feedback-shift-register, forward-error-correction]
cite_urls:
  - https://en.wikipedia.org/wiki/Chien_search
  - https://ieeexplore.ieee.org/document/1053907
---

**Chien search** is the step in algebraic decoding that finds the **roots** of the
error-locator polynomial Λ(x) by simply evaluating it at every element of the code's finite
field.[^wiki] Each root corresponds to one error position in the received
[Reed–Solomon](/reference/reed-solomon-code/) or [BCH](/reference/bch-code/) codeword, so
Chien search converts the abstract locator polynomial — produced by the
[Berlekamp–Massey algorithm](/reference/berlekamp-massey-algorithm/) — into a concrete list
of *which symbols* are wrong.[^chien]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="The error-locator polynomial is evaluated at successive powers of the field's primitive element; positions where the sum equals zero are marked as errors." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1"><line x1="30" y1="105" x2="440" y2="105"/></g>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="35" y="120">α⁰</text><text x="75" y="120">α¹</text><text x="115" y="120">α²</text><text x="155" y="120">α³</text><text x="195" y="120">α⁴</text><text x="235" y="120">α⁵</text><text x="275" y="120">α⁶</text><text x="315" y="120">α⁷</text><text x="355" y="120">α⁸</text><text x="395" y="120">α⁹</text><text x="430" y="120">…</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2">
    <line x1="35" y1="105" x2="35" y2="70"/><line x1="75" y1="105" x2="75" y2="55"/><line x1="115" y1="105" x2="115" y2="105"/><line x1="155" y1="105" x2="155" y2="80"/><line x1="195" y1="105" x2="195" y2="105"/><line x1="235" y1="105" x2="235" y2="45"/><line x1="275" y1="105" x2="275" y2="72"/><line x1="315" y1="105" x2="315" y2="60"/><line x1="355" y1="105" x2="355" y2="88"/><line x1="395" y1="105" x2="395" y2="66"/>
  </g>
  <g fill="currentColor"><circle cx="115" cy="105" r="4"/><circle cx="195" cy="105" r="4"/></g>
  <text x="155" y="30" text-anchor="middle" font-size="10" fill="currentColor">Λ(x) evaluated at each field element — zeros (dots) mark error positions</text>
</svg>
<figcaption>Chien search sweeps the whole field; wherever Λ(x) evaluates to zero, that position holds an error.</figcaption>
</figure>

## How it works

The error-locator polynomial Λ(x) has the property that its roots are the *inverses* of the
field elements that mark the error positions. Because a finite field has only a fixed number
of nonzero elements — every one a power α⁰, α¹, α², … of a primitive element α — you can find
all the roots by exhaustive substitution: plug in each α^i in turn and check whether Λ(α^i)
comes out to zero.

Chien's contribution is making that sweep cheap. Instead of recomputing every power of every
term from scratch, it keeps a running set of registers, one per polynomial coefficient. To
move from testing α^i to testing α^(i+1), each register is multiplied by a fixed constant
(α raised to that term's degree), and the values are summed. A zero sum means α^i is a root
and therefore position *i* is in error:

- Set up one accumulator per coefficient of Λ(x), scaled by increasing powers of α.
- For each candidate position, add the accumulators; if the total is zero, record the
  position.
- Advance every accumulator by one multiplication and repeat across the whole codeword.

The cost is one field multiply-and-add per coefficient per position, giving simple,
regular, highly parallelisable hardware — which is why Chien search is the standard root
finder in RS/BCH decoder chips even though it is, at heart, brute-force evaluation.

## In practice

Chien search sits in the middle of the classic three-stage algebraic decoder:
Berlekamp–Massey (or extended Euclid) builds Λ(x); **Chien search finds its roots to give
the error positions**; then the [Forney algorithm](/reference/forney-algorithm/) computes
the error magnitudes at exactly those positions. For a code that corrects *t* errors, the
degree of Λ(x) tells you how many roots to expect; if Chien search finds fewer distinct
roots in the valid range than the degree, the received word contained more errors than the
code can handle, and the decoder can flag an uncorrectable block rather than emit a wrong
correction. That built-in failure detection is one reason the exhaustive sweep is valued over
cleverer but less transparent root finders for short codes.

## Relevance to SDR

The short Reed–Solomon and BCH codes used in digital radio signalling — for example the RS
protection on parts of [P25](/reference/p25-phase-1/) and the BCH-guarded sync/status words
in several trunking formats — are decoded with exactly this position-then-value pipeline.
Chien search is an internal step of the [forward error correction](/reference/forward-error-correction/)
stage, invisible in the traffic itself. GopherTrunk validates and corrects these protected
fields as part of decoding; Chien search names the general-purpose method for the
root-finding half of that block-code math, whether realised as a full field sweep or, for
the smallest codes, a lookup.

## Sources

[^wiki]: [Chien search](https://en.wikipedia.org/wiki/Chien_search) — Wikipedia, for the field-sweep root finding of the error-locator polynomial.
[^chien]: [Cyclic decoding procedures for BCH codes](https://ieeexplore.ieee.org/document/1053907) — R. T. Chien, IEEE Trans. Information Theory (1964), the original search procedure.
