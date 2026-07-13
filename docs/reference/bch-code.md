---
slug: bch-code
title: BCH code
entry_type: algorithm
category: error-correction
description: BCH codes are cyclic block error-correction codes built over a Galois field that can be designed to correct a chosen number of bit errors; POCSAG, FLEX, and DSC use BCH codes.
keywords: BCH code, Bose Chaudhuri Hocquenghem, cyclic code, designed distance, Galois field, GF(2^m), generator polynomial, syndrome, Berlekamp-Massey, POCSAG, DSC
aka: [BCH code]
autolink: true
infobox:
  - { label: Type, value: Cyclic block code }
  - { label: Built over, value: Galois field GF(2^m) }
  - { label: Designed for, value: A target error-correction capability }
see_also: [reed-solomon-code, cyclic-redundancy-check, berlekamp-massey-algorithm, hamming-code, forward-error-correction, pocsag]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/rf-sdr/other-signals/ }
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/BCH_code
  - https://en.wikipedia.org/wiki/Cyclic_code
---

**BCH codes** (Bose–Chaudhuri–Hocquenghem) are a class of **cyclic** block
[error-correction](/reference/forward-error-correction/) codes that can be constructed to
correct a chosen number of bit errors.[^wiki] Their defining feature is *designed distance*:
the code builder picks how many errors t must be correctable, and the algebra then dictates
how many parity bits are required — a flexibility that makes BCH a common choice for
paging and signalling formats.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="A BCH codeword shown as a block of message bits followed by parity-check bits." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="40" width="230" height="30" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="155" y="59" text-anchor="middle" font-size="9" fill="currentColor">message bits (k)</text>
  <rect x="270" y="40" width="150" height="30" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.2"/><text x="345" y="59" text-anchor="middle" font-size="9" fill="currentColor">parity (n−k)</text>
  <text x="230" y="90" text-anchor="middle" font-size="9" fill="currentColor">e.g. BCH(31,21) in POCSAG</text>
</svg>
<figcaption>BCH codes append algebraically-computed parity bits that correct multiple bit errors per codeword.</figcaption>
</figure>

## How it works

A BCH code is **cyclic**, meaning any cyclic shift of a codeword is also a codeword; this
lets both encoding and syndrome computation be done with simple shift-register polynomial
arithmetic, the same machinery behind the [CRC](/reference/cyclic-redundancy-check/). The
code is defined over the Galois field GF(2^m): to build a t-error-correcting code you form a
generator polynomial whose roots include 2t consecutive powers of a primitive element, so
every valid codeword is divisible by it. On reception the decoder evaluates the received
polynomial at those roots to produce **syndromes**; if they are all zero the word is clean,
and otherwise it solves for an *error-locator polynomial* — typically with the
[Berlekamp–Massey algorithm](/reference/berlekamp-massey-algorithm/) — whose roots reveal
which bit positions to flip. Because BCH is binary, once the positions are known the values
are simply inverted, so no separate error-magnitude step is needed.

## Variants

The binary [Hamming codes](/reference/hamming-code/) are the special single-error-correcting
case of BCH. Extending the same construction from bits to multi-bit symbols yields the
[Reed–Solomon codes](/reference/reed-solomon-code/), which are exactly the non-binary BCH
codes and share the identical syndrome/locator decoding path — Reed–Solomon just adds a
Forney error-value stage because its symbols carry magnitude as well as position. Between
these extremes, primitive and shortened BCH codes give designers a fine-grained way to trade
parity overhead for correction strength.

## Relevance to SDR

BCH coding protects the short, critical words in several signalling formats GopherTrunk
meets: [POCSAG](/reference/pocsag/) paging uses a BCH(31,21) code on each codeword,
[DSC](/reference/dsc/) marine calling uses a small BCH code, and FLEX paging layers BCH with
interleaving. Recovering these words even when a few bits are corrupted is what lets the
decoder read paging and control traffic reliably on a weak signal.

## Sources

[^wiki]: [BCH code](https://en.wikipedia.org/wiki/BCH_code) — Wikipedia, for the cyclic code family, Galois-field construction, and designed error-correction capability. See also [Cyclic code](https://en.wikipedia.org/wiki/Cyclic_code) for the shift-register polynomial structure.
