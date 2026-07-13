---
slug: flex
title: FLEX
entry_type: protocol
category: paging-data
description: FLEX is a high-speed one-way paging protocol developed by Motorola, using 4-level FSK at up to 6400 bps with strong synchronisation and error correction for reliable wide-area paging.
keywords: FLEX paging, Motorola FLEX, 4-FSK, high-speed pager, 1600 3200 6400 bps, simulcast paging
aka: [FLEX]
autolink: true
infobox:
  - { label: Type, value: One-way paging protocol }
  - { label: Developer, value: Motorola }
  - { label: Modulation, value: 2- or 4-level FSK }
  - { label: Bit rates, value: 1600 / 3200 / 6400 bps }
  - { label: Error correction, value: BCH + interleaving }
  - { label: GopherTrunk support, value: See Status }
see_also: [pocsag, frequency-shift-keying, bch-code, interleaving]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/rf-sdr/other-signals/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/FLEX_(protocol)
---

**FLEX** is a high-speed one-way **paging** protocol developed by **Motorola** to
succeed [POCSAG](/reference/pocsag/). It uses 2- or 4-level
[FSK](/reference/frequency-shift-keying/) at up to 6400 bps, with robust
synchronisation and [interleaving](/reference/interleaving/) that make it resilient on
wide-area simulcast networks.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="A FLEX frame with sync and time-multiplexed blocks, at higher rates than POCSAG." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1">
    <rect x="30" y="40" width="60" height="28" fill="currentColor" fill-opacity="0.12"/><rect x="90" y="40" width="85" height="28" fill="none"/><rect x="175" y="40" width="85" height="28" fill="none"/><rect x="260" y="40" width="85" height="28" fill="none"/><rect x="345" y="40" width="85" height="28" fill="none"/>
  </g>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="60" y="58">sync</text><text x="132" y="58">block</text><text x="217" y="58">block</text><text x="302" y="58">block</text><text x="387" y="58">block</text></g>
  <text x="230" y="88" text-anchor="middle" font-size="8" fill="currentColor">multi-level FSK · up to 6400 bps</text>
</svg>
<figcaption>FLEX is a higher-rate paging protocol using time-multiplexed frames and multi-level FSK.</figcaption>
</figure>

## Overview

FLEX organises traffic into precisely timed frames and phases, allowing high capacity
and good battery life for pagers. Its time-synchronised structure suits large
simulcast paging systems covering whole regions.

## Technical characteristics

| Property | Value |
|----------|-------|
| Modulation | 2-FSK or 4-FSK |
| Bit rates | 1600, 3200, 6400 bps |
| Coding | BCH + bit interleaving |
| Framing | Time-synchronised frames/phases |

## History

Introduced by Motorola in the 1990s as paging demand outgrew POCSAG's capacity; widely
deployed by commercial paging carriers.[^wiki]

## Deployment

Commercial wide-area paging, healthcare, and emergency notification networks.

## Decoding it with GopherTrunk

FLEX shares the FSK paging family with POCSAG; see [Status](/status.html) for current
coverage and the [POCSAG decoder](/pocsag.html) page for the paging pipeline.

## Sources

[^wiki]: [FLEX](https://en.wikipedia.org/wiki/FLEX_(protocol)) — Wikipedia, for Motorola's high-speed paging protocol, its multi-level FSK rates, and time-synchronised framing.
