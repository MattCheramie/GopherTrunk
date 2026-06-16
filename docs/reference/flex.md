---
slug: flex
title: FLEX
entry_type: protocol
category: protocols
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
  - { title: "Other signals you'll meet", url: /learn/other-signals/ }
external:
  - { title: "FLEX (Wikipedia)", url: https://en.wikipedia.org/wiki/FLEX_(protocol) }
---

**FLEX** is a high-speed one-way **paging** protocol developed by **Motorola** to
succeed [POCSAG](/reference/pocsag/). It uses 2- or 4-level
[FSK](/reference/frequency-shift-keying/) at up to 6400 bps, with robust
synchronisation and [interleaving](/reference/interleaving/) that make it resilient on
wide-area simulcast networks.

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
deployed by commercial paging carriers.

## Deployment

Commercial wide-area paging, healthcare, and emergency notification networks.

## Decoding it with GopherTrunk

FLEX shares the FSK paging family with POCSAG; see [Status](/status.html) for current
coverage and the [POCSAG decoder](/pocsag.html) page for the paging pipeline.
