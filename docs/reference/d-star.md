---
slug: d-star
title: D-STAR
entry_type: protocol
category: protocols
description: D-STAR (Digital Smart Technologies for Amateur Radio) is an amateur digital-voice and data standard developed by the JARL and popularised by Icom, using GMSK and an AMBE-family vocoder.
keywords: D-STAR, amateur digital voice, JARL, Icom, GMSK, AMBE, DV, reflectors
aka: [D-STAR, DSTAR]
autolink: true
infobox:
  - { label: Type, value: Amateur digital voice + data }
  - { label: Developer, value: JARL (popularised by Icom) }
  - { label: Access, value: FDMA }
  - { label: Channel spacing, value: 6.25 kHz (DV) }
  - { label: Modulation, value: GMSK (4800 bps) }
  - { label: Vocoder, value: AMBE (DV mode) }
  - { label: GopherTrunk support, value: See Status }
see_also: [system-fusion-ysf, m17, gmsk, ambe, vocoder]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/protocol-landscape/ }
external:
  - { title: "D-STAR (Wikipedia)", url: https://en.wikipedia.org/wiki/D-STAR }
---

**D-STAR** (**Digital Smart Technologies for Amateur Radio**) is an amateur-radio
digital voice and data standard developed by the Japan Amateur Radio League (JARL)
and widely implemented by **Icom**. It uses [GMSK](/reference/gmsk/) modulation and an
[AMBE](/reference/ambe/)-family [vocoder](/reference/vocoder/) for its digital-voice
(DV) mode.

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 110" role="img" aria-label="D-STAR digital voice from radio to repeater to internet gateways." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="30" y="44" width="70" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="65" y="63">radio</text>
    <rect x="150" y="44" width="80" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="190" y="63">repeater</text>
    <rect x="290" y="44" width="120" height="30" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="350" y="58">internet</text><text x="350" y="69" font-size="8">reflectors/gateways</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="100" y1="59" x2="149" y2="59" marker-end="url(#am_d-star)"/><line x1="230" y1="59" x2="289" y2="59" marker-end="url(#am_d-star)"/></g>
    <text x="65" y="30" font-size="8">GMSK · AMBE</text>
  </g>
  <defs><marker id="am_d-star" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>D-STAR carries digital voice and data and links repeaters worldwide via internet reflectors.</figcaption>
</figure>

## Overview

D-STAR carries digital voice plus a low-rate data stream, and is known for its
internet-linked **reflectors** and gateways that connect repeaters worldwide. DV mode
fits in a narrow 6.25 kHz channel.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | [FDMA](/reference/fdma/) |
| Modulation | GMSK, 4800 bps |
| Vocoder | AMBE (DV) |
| Data | Low-rate data alongside voice |

## History

Specified by the JARL around 2001 and brought to market by Icom; one of the earliest
mainstream amateur digital-voice systems.

## Deployment

Amateur radio worldwide, via DV repeaters, hotspots, and networked reflectors.

## Decoding it with GopherTrunk

See [Status](/status.html) for GopherTrunk's D-STAR link-layer and voice coverage.
