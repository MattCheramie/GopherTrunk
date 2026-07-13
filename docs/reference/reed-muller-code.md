---
slug: reed-muller-code
title: Reed–Muller code
entry_type: algorithm
category: error-correction
description: A Reed–Muller code is a family of linear block codes RM(r,m) built from Boolean monomials of degree ≤ r; RM(1,m) is biorthogonal and Hadamard-related, famously used on Mariner deep-space imaging.
keywords: Reed-Muller code, RM code, RM(r,m), biorthogonal code, majority-logic decoding, Reed decoding, Plotkin construction, Mariner 9, deep space, polar code, Hadamard
aka: [Reed–Muller code, Reed-Muller code, RM code]
autolink: true
infobox:
  - { label: Type, value: Linear block code family }
  - { label: Parameters, value: order r, length 2^m }
  - { label: Notable use, value: Mariner deep-space imaging }
see_also: [hadamard-code, forward-error-correction, bch-code, reed-solomon-code, hamming-code, polar-code]
cite_urls:
  - https://en.wikipedia.org/wiki/Reed%E2%80%93Muller_code
  - https://en.wikipedia.org/wiki/Mariner_9
---

A **Reed–Muller code**, written **RM(r, m)**, is a family of linear
[block error-correction codes](/reference/forward-error-correction/) parameterised by
an *order* `r` and a *length exponent* `m`: the codewords are the value tables of all
Boolean functions in `m` variables of algebraic degree at most `r`.[^wiki] The family
spans a full range from high-rate, low-protection codes to very-low-rate, very-strong
codes as `r` moves from `m` down to `0`, which is what makes it a useful teaching and
engineering "dial" between rate and distance.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="The recursive Plotkin construction: two shorter Reed-Muller codewords u and v combine into a double-length codeword formed as u concatenated with u XOR v." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="rmar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <rect x="30" y="30" width="90" height="26"/><rect x="30" y="90" width="90" height="26"/>
    <rect x="250" y="60" width="90" height="26"/><rect x="345" y="60" width="90" height="26"/>
  </g>
  <text x="75" y="47" text-anchor="middle" font-size="11" fill="currentColor">u  (len n)</text>
  <text x="75" y="107" text-anchor="middle" font-size="11" fill="currentColor">v  (len n)</text>
  <text x="295" y="77" text-anchor="middle" font-size="11" fill="currentColor">u</text>
  <text x="390" y="77" text-anchor="middle" font-size="11" fill="currentColor">u ⊕ v</text>
  <line x1="120" y1="43" x2="245" y2="66" stroke="currentColor" stroke-width="1" marker-end="url(#rmar)"/>
  <line x1="120" y1="103" x2="340" y2="76" stroke="currentColor" stroke-width="1" marker-end="url(#rmar)"/>
  <text x="342" y="118" text-anchor="middle" font-size="9" fill="currentColor" fill-opacity="0.85">codeword of length 2n</text>
  <line x1="250" y1="100" x2="435" y2="100" stroke="currentColor" stroke-width="0.7" stroke-dasharray="3 3"/>
</svg>
<figcaption>The Plotkin "(u | u ⊕ v)" construction builds RM(r, m) recursively from RM(r, m−1) and RM(r−1, m−1), doubling the length each step.</figcaption>
</figure>

## How it works

RM(r, m) has block length `n = 2^m`. Its generator rows are the evaluation vectors of
the monomials `1, x₁, …, xₘ, x₁x₂, …` up to degree `r`, so the message length is the
number of such monomials and the **minimum distance is `2^(m−r)`**. Two boundary
cases anchor the family:

- **RM(0, m)** is the repetition code — one bit sent `2^m` times, maximum protection,
  minimum rate.
- **RM(m, m)** is the whole space — no redundancy at all.
- **RM(m−1, m)** is a single-parity-check code, and **RM(1, m)** is the
  **biorthogonal code**, closely tied to the [Hadamard code](/reference/hadamard-code/):
  its non-constant codewords are the rows of a Hadamard matrix and their complements,
  so it has huge minimum distance `2^(m−1)` at a very low rate.

Codes double in length through the **Plotkin `(u | u ⊕ v)` construction** shown above,
which builds RM(r, m) from RM(r, m−1) and RM(r−1, m−1). Classic decoding uses
**Reed's majority-logic algorithm**: each message coefficient is estimated by a
majority vote over many independent parity checks, so the code corrects up to
`2^(m−r−1) − 1` errors with simple, hardware-friendly logic. RM(1, m) in particular
can be decoded by a **fast Hadamard transform**, correlating the received word against
all codewords at once in `O(n log n)`.

## Variants

Beyond the binary codes, **generalised Reed–Muller codes** extend the idea to
non-binary alphabets, and **punctured/shortened** RM codes trim the length to fit a
frame. The family also connects forward in time: **polar codes**
([polar-code](/reference/polar-code/)), the 5G control-channel code, are built on the
same `2^m × 2^m` transform matrix as Reed–Muller and differ only in *which* rows are
frozen — RM keeps the highest-weight rows, polar keeps the most-reliable ones under a
specific channel. This kinship is why RM is often taught as the "ancestor" of polar
coding.

## In practice

Reed–Muller's headline appearance is **deep-space imaging**: NASA's **Mariner 9**
(1971–72) and Mariner 6/7 sent their black-and-white Mars pictures protected by the
RM(1, 5) biorthogonal code — a 6-bit pixel value became a 32-bit codeword with
minimum distance 16, decoded by a Green Machine hardware Hadamard transform. The very
low rate was affordable because bandwidth was cheap relative to the enormous cost of a
lost frame across interplanetary distances.

## Relevance to SDR

Reed–Muller is more common in space, storage and standards lineage than in the VHF/UHF
land-mobile systems **GopherTrunk** decodes, so GT does not run an RM decoder in its
chain — those links lean on [BCH](/reference/bch-code/),
[Golay](/reference/golay-code/) and [Reed–Solomon](/reference/reed-solomon-code/)
codes instead. RM still matters to the SDR reader on two fronts: it is the
low-rate/high-gain code you reach for on a deeply buried channel (deep space, beacon
telemetry), and its transform structure is the direct conceptual root of the
[polar codes](/reference/polar-code/) carrying 5G NR control information — the same
Hadamard-style butterfly a software radio would compute to decode either one.

## Sources

[^wiki]: [Reed–Muller code](https://en.wikipedia.org/wiki/Reed%E2%80%93Muller_code) — Wikipedia, for the RM(r, m) definition, minimum distance 2^(m−r), the Plotkin construction, Reed majority-logic decoding, the biorthogonal RM(1, m)–Hadamard link, and the Mariner and polar-code connections.
