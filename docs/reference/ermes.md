---
slug: ermes
title: European Radio Message System (ERMES)
entry_type: protocol
category: paging-data
description: ERMES is a pan-European high-speed paging standard using 4-FSK modulation at 6250 bps across frequency-agile channels, designed by ETSI to unify European paging.
keywords: ERMES, European Radio Message System, ETSI, paging, 4-FSK, 6250 bps, 169 MHz, frequency agile, alphanumeric paging
aka: [ERMES, "European Radio Message System"]
autolink: true
infobox:
  - { label: Type, value: One-way high-speed paging protocol }
  - { label: Standards body, value: ETSI }
  - { label: Introduced, value: "1990s" }
  - { label: Access, value: "FDMA/TDMA, frequency-agile" }
  - { label: Channel spacing, value: 25 kHz (169 MHz band) }
  - { label: Modulation, value: 4-FSK at 6250 bps }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [four-fsk, flex, pocsag, frequency-shift-keying, etsi, frame-synchronization]
cite_urls:
  - https://en.wikipedia.org/wiki/ERMES
  - https://en.wikipedia.org/wiki/Radio_paging
---

**ERMES** (the **European Radio Message System**) is a high-speed
[paging](/reference/pocsag/) standard developed by [ETSI](/reference/etsi/) to give
Europe a single, roaming-capable paging technology. It transmits at 6250 bps using
**[4-FSK](/reference/four-fsk/)** — four-level frequency-shift keying — and is
frequency-agile, letting a pager hop across a block of channels to follow its
messages.[^wiki][^pg]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="ERMES pagers roam across a block of adjacent 25 kHz channels, following a frequency-agile 4-FSK data stream." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="er_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="30" y1="100" x2="440" y2="100" stroke="currentColor" stroke-opacity="0.4" marker-end="url(#er_ar)"/>
  <text x="235" y="122" text-anchor="middle" font-size="9" fill="currentColor">frequency → · 16 channels in the 169 MHz band</text>
  <g stroke="currentColor" stroke-width="1.1">
    <rect x="45" y="55" width="45" height="40" fill="currentColor" fill-opacity="0.22"/>
    <rect x="95" y="55" width="45" height="40" fill="none"/>
    <rect x="145" y="55" width="45" height="40" fill="currentColor" fill-opacity="0.22"/>
    <rect x="195" y="55" width="45" height="40" fill="none"/>
    <rect x="245" y="55" width="45" height="40" fill="currentColor" fill-opacity="0.22"/>
  </g>
  <path d="M67 55 C 90 30, 140 30, 167 55" fill="none" stroke="currentColor" stroke-width="1" marker-end="url(#er_ar)"/>
  <text x="150" y="30" text-anchor="middle" font-size="8" fill="currentColor">pager follows the data as it hops channels</text>
  <text x="235" y="47" text-anchor="middle" font-size="8" fill="currentColor">4-FSK · 6250 bps</text>
</svg>
<figcaption>ERMES is frequency-agile: a pager retunes across a block of 25 kHz channels to keep receiving its 4-FSK data stream.</figcaption>
</figure>

## Overview

ERMES organises transmissions into a repeating cycle of sequences and batches so that a
pager knows exactly when and where its address will appear, allowing it to sleep between
its slots for long battery life. The four-level FSK doubles the bits-per-symbol over the
two-level FSK of [POCSAG](/reference/pocsag/), while forward error correction and
interleaving protect the data against fading. The frequency-agile design spreads one
logical service across up to sixteen 25 kHz channels in the 169.4–169.8 MHz band, so
capacity scales and pagers can follow traffic across the block.

## Technical characteristics

| Property | Value |
|----------|-------|
| Modulation | 4-FSK (four-level) |
| Bit rate | 6250 bps |
| Band | 169.4–169.8 MHz |
| Channels | Up to 16 × 25 kHz, frequency-agile |
| Structure | Slotted sequences and batches |
| Coding | Forward error correction + interleaving |

The slotted, frequency-agile structure is what distinguishes ERMES from the simpler
asynchronous POCSAG format and lets it reach much higher throughput.

## History

ETSI developed ERMES in the early 1990s as its attempt to standardise European paging
the way GSM standardised cellular, competing with the American [FLEX](/reference/flex/)
high-speed paging family and the ubiquitous POCSAG. Though technically capable, it saw
limited uptake, and a common European roaming paging market never fully materialised.

## Deployment

A handful of ERMES services operated across Europe in the 1990s and 2000s before mobile
phones and SMS eclipsed dedicated paging. Most have since closed, making live ERMES rare
compared with POCSAG, which persists in hospital and emergency use.

## Decoding it with GopherTrunk

GopherTrunk does **not decode** ERMES. Its paging support centres on the common codes
POCSAG and FLEX; ERMES is documented here for identification and to round out the paging
landscape. GopherTrunk decodes clear traffic only, and ERMES is not currently a supported
target.

## Sources

[^wiki]: [ERMES](https://en.wikipedia.org/wiki/ERMES) — Wikipedia, for the ETSI European Radio Message System, its 4-FSK modulation at 6250 bps, and its frequency-agile 169 MHz operation.
[^pg]: [Radio paging](https://en.wikipedia.org/wiki/Radio_paging) — Wikipedia, for the place of ERMES among paging standards alongside POCSAG and FLEX.
