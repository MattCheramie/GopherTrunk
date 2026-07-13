---
slug: cdma
title: Code-division multiple access (CDMA)
entry_type: algorithm
category: spread-spectrum
description: CDMA lets many users share the same frequency band simultaneously by assigning each a distinct spreading code, separating them by correlation rather than by time or frequency slots.
keywords: code-division multiple access, CDMA, spreading code, IS-95, cdmaOne, UMTS, WCDMA, near-far problem, power control, Walsh codes, PN codes, GPS
aka: [CDMA, code-division multiple access]
autolink: true
infobox:
  - { label: Type, value: Multiple-access scheme }
  - { label: Separates users by, value: Orthogonal / PN codes }
  - { label: Used by, value: IS-95, UMTS, GPS }
see_also: [direct-sequence-spread-spectrum, gold-code, maximal-length-sequence, tdma, fdma, hadamard-code]
cite_urls:
  - https://en.wikipedia.org/wiki/Code-division_multiple_access
  - https://en.wikipedia.org/wiki/IS-95
---

**Code-division multiple access (CDMA)** lets many transmitters occupy the *same* frequency
band at the *same* time by giving each one a distinct spreading **code**; a receiver recovers
one user by correlating against that user's code and treating everyone else as noise.[^wiki]
Where [FDMA](/reference/fdma/) divides users by frequency and [TDMA](/reference/tdma/) by time
slot, CDMA divides them by code — all users overlap fully in time and frequency and are
pulled apart by [correlation](/reference/matched-filter/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Three users' data streams each multiplied by a different code, summed into one shared channel, then recovered individually by correlating with the matching code." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="cdmaar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="45" y="35">A × cₐ</text>
    <text x="45" y="75">B × c_b</text>
    <text x="45" y="115">C × c_c</text>
    <text x="230" y="20">shared band (Σ)</text>
    <text x="415" y="35">·cₐ → A</text>
    <text x="415" y="75">·c_b → B</text>
    <text x="415" y="115">·c_c → C</text>
  </g>
  <g stroke="currentColor" stroke-width="1.1" fill="none" marker-end="url(#cdmaar)">
    <path d="M75 32 C 150 32 160 70 205 70"/>
    <path d="M75 72 H205"/>
    <path d="M75 112 C 150 112 160 74 205 74"/>
    <path d="M255 70 C 300 70 300 35 375 35"/>
    <path d="M255 72 H375"/>
    <path d="M255 74 C 300 74 300 112 375 112"/>
  </g>
  <rect x="205" y="55" width="50" height="30" fill="currentColor" fill-opacity="0.2" stroke="currentColor"/>
</svg>
<figcaption>Every CDMA user shares one band; each is coded on transmit and pulled back out by correlating against its own code on receive.</figcaption>
</figure>

## How it works

CDMA is built on [direct-sequence spread spectrum](/reference/direct-sequence-spread-spectrum/):
each user's low-rate data is multiplied by a high-rate code sequence before transmission. The
codes are chosen so that a user's own code correlates strongly with itself but weakly with
every other user's code. At the receiver, multiplying the composite signal by one code
despreads that user's data back to full amplitude while the other users — uncorrelated —
stay spread out and contribute only a small amount of noise-like interference.

Two code roles appear together in practice:

- **Orthogonal codes** ([Walsh/Hadamard](/reference/hadamard-code/)) give *zero*
  cross-correlation when users are perfectly time-aligned — ideal for the synchronous
  downlink from one base station.
- **PN codes** ([m-sequences](/reference/maximal-length-sequence/) and
  [Gold codes](/reference/gold-code/)) give low but non-zero cross-correlation for
  *un*synchronized links (the uplink, or GPS satellites), where perfect orthogonality is
  impossible.

## In practice

The defining engineering problem is the **near-far problem**: a handset close to the base
station arrives far stronger than a distant one, and since separation relies on
finite-quality code correlation, the strong signal's residual interference can bury the weak
one. CDMA systems solve this with fast, tight **power control** — commanding every handset to
transmit just enough power to arrive at roughly equal strength (IS-95 adjusts power ~800
times per second). Capacity is *soft*: adding users gradually raises everyone's noise floor
rather than exhausting a fixed pool of slots, so the system degrades gracefully. A **RAKE
receiver** exploits [multipath](/reference/multipath-propagation/) by correlating several
delayed copies of the code and combining them, turning echoes into diversity gain.

## Relevance to SDR

CDMA defined a generation of cellular: **IS-95 / cdmaOne** and **CDMA2000**, and the wideband
CDMA air interface of **UMTS/3G**. **GPS** is a CDMA system too — every satellite shares
1575.42 MHz and is distinguished only by its [Gold code](/reference/gold-code/). Modern
[LTE](/reference/ofdm/) and 5G NR moved to OFDMA, but CDMA remains foundational and still
lives inside GNSS receivers.

GopherTrunk's target land-mobile trunking systems are FDMA and TDMA, not CDMA, so the scanner
does not implement a despreading/correlating receiver. CDMA is documented here to place the
code-division idea alongside the time- and frequency-division schemes the scanner actually
follows, and as the direct parent of the spreading-code families used in GNSS.

## Sources

[^wiki]: [Code-division multiple access](https://en.wikipedia.org/wiki/Code-division_multiple_access) — Wikipedia, for the code-separation principle, near-far problem, and power control.
