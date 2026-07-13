---
slug: gold-code
title: Gold code
entry_type: algorithm
category: spread-spectrum
description: A Gold code is the XOR of two carefully chosen maximal-length sequences, producing large families of codes with low, bounded cross-correlation ideal for multi-user spread spectrum; used in GPS C/A codes and CDMA.
keywords: Gold code, Gold sequence, preferred pair, m-sequence, cross-correlation, GPS C/A code, CDMA, spreading code, LFSR, PRN
aka: [Gold code, Gold sequence]
autolink: true
infobox:
  - { label: Type, value: PN spreading-code family }
  - { label: Built from, value: XOR of two m-sequences }
  - { label: Used by, value: GPS C/A codes, CDMA }
see_also: [maximal-length-sequence, linear-feedback-shift-register, direct-sequence-spread-spectrum, cdma, barker-code, gps-receiver]
cite_urls:
  - https://en.wikipedia.org/wiki/Gold_code
  - https://www.gps.gov/technical/icwg/
---

**A Gold code** is formed by XOR-ing two [maximal-length sequences](/reference/maximal-length-sequence/)
of the same length that are chosen as a "preferred pair," yielding a *large family* of codes
whose pairwise **cross-correlation** is low and tightly bounded.[^wiki] That property is
exactly what a multi-user [spread-spectrum](/reference/direct-sequence-spread-spectrum/)
system needs: each user (or satellite) gets its own Gold code, and the receiver separates
them by correlation with minimal mutual interference.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Two LFSR-generated m-sequences fed into an exclusive-OR gate produce a Gold code; shifting the second sequence by different amounts yields the whole family of codes." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="goldar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" fill="none" stroke-width="1.1">
    <rect x="35" y="25" width="110" height="26"/><rect x="35" y="90" width="110" height="26"/>
    <circle cx="230" cy="70" r="18"/>
    <rect x="330" y="57" width="100" height="26"/>
  </g>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="90" y="42">m-sequence 1 (LFSR)</text>
    <text x="90" y="107">m-sequence 2 (LFSR)</text>
    <text x="230" y="74">XOR</text>
    <text x="380" y="74">Gold code</text>
  </g>
  <g stroke="currentColor" stroke-width="1.1" marker-end="url(#goldar)">
    <path d="M145 38 C 190 38 195 62 211 64"/>
    <path d="M145 103 C 190 103 195 78 211 76"/>
    <path d="M248 70 H328"/>
  </g>
</svg>
<figcaption>A Gold code is the XOR of two preferred m-sequences; varying the relative phase of the second sequence generates the whole low-cross-correlation family.</figcaption>
</figure>

## How it works

Take two [LFSR](/reference/linear-feedback-shift-register/)-generated
[m-sequences](/reference/maximal-length-sequence/) of length N = 2ⁿ−1, chosen so their
cross-correlation takes only three values — a **preferred pair**. XOR the two sequences
together. By shifting the second sequence by 0, 1, 2, … N−1 chips relative to the first
before combining, you get N distinct Gold codes, and adding the two parent m-sequences
themselves gives a family of **N + 2** codes total.

The payoff is bounded cross-correlation. Any two codes in the family correlate to at most a
small "three-valued" set of magnitudes — {−1, −t(n), t(n)−2} where t(n) ≈ 2^((n+2)/2) — far
better and, crucially, more *uniform* than picking arbitrary m-sequences, whose mutual
correlation can spike badly. Individual Gold codes are *not* maximal-length and have slightly
worse **autocorrelation** sidelobes than a pure m-sequence, so there is a small trade of
peak sharpness for a much larger, better-behaved family. For spreading many simultaneous
users that trade is overwhelmingly worth it.

## Relevance to SDR

The canonical use is **GPS**: every satellite transmits on the same 1575.42 MHz L1 carrier
and is distinguished only by its 1023-chip Gold code (the PRN or C/A code), letting a
[GPS receiver](/reference/gps-receiver/) — a [CDMA](/reference/cdma/) receiver in disguise —
acquire and separate satellites by correlation. Gold codes also serve as scrambling and
spreading codes in cellular CDMA and other multi-access links. They sit in the same code
family toolbox as [Barker codes](/reference/barker-code/) (short sync words) and Kasami codes
(another low-cross-correlation family).

GopherTrunk's land-mobile trunking targets do not use Gold codes, and the scanner implements
no Gold-code correlator. The entry is here to explain how GNSS and CDMA build large sets of
near-orthogonal spreading codes from the same [LFSR](/reference/linear-feedback-shift-register/)
machinery that also produces the [scrambling](/reference/scrambling/) sequences seen in
digital voice protocols.

## Sources

[^wiki]: [Gold code](https://en.wikipedia.org/wiki/Gold_code) — Wikipedia, for the preferred-pair XOR construction, family size, and the three-valued cross-correlation bound.
