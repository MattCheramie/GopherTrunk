---
slug: system-fusion-ysf
title: System Fusion (YSF)
entry_type: protocol
category: protocols
description: System Fusion (Yaesu System Fusion, YSF) is Yaesu's amateur C4FM digital-voice system, supporting digital and analog modes with an AMBE-family vocoder and internet-linked rooms.
keywords: System Fusion, YSF, Yaesu, C4FM, amateur digital voice, Fusion, Wires-X, AMBE
aka: [System Fusion, Yaesu System Fusion, YSF, Fusion]
autolink: true
infobox:
  - { label: Type, value: Amateur digital voice }
  - { label: Developer, value: Yaesu }
  - { label: Access, value: FDMA }
  - { label: Modulation, value: C4FM (4-level FSK) }
  - { label: Vocoder, value: AMBE family }
  - { label: GopherTrunk support, value: See Status }
see_also: [d-star, m17, c4fm, ambe, vocoder]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
external:
  - { title: "System Fusion (Wikipedia)", url: https://en.wikipedia.org/wiki/System_Fusion }
---

**System Fusion** (**Yaesu System Fusion**, **YSF**) is Yaesu's amateur digital-voice
system, using [C4FM](/reference/c4fm/) modulation and an [AMBE](/reference/ambe/)-family
[vocoder](/reference/vocoder/). It is notable for automatically mixing digital and
analog users on the same repeater.

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 110" role="img" aria-label="System Fusion digital voice linked over WIRES-X rooms." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="30" y="44" width="70" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="65" y="63">radio</text>
    <rect x="150" y="44" width="80" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="190" y="63">repeater</text>
    <rect x="290" y="44" width="120" height="30" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="350" y="58">internet</text><text x="350" y="69" font-size="8">WIRES-X rooms</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="100" y1="59" x2="149" y2="59" marker-end="url(#am_system-fusion-ysf)"/><line x1="230" y1="59" x2="289" y2="59" marker-end="url(#am_system-fusion-ysf)"/></g>
    <text x="65" y="30" font-size="8">C4FM</text>
  </g>
  <defs><marker id="am_system-fusion-ysf" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>System Fusion (YSF) uses C4FM and can bridge analog and digital users, linked via WIRES-X.</figcaption>
</figure>

## Overview

Fusion repeaters can operate in modes that bridge analog FM and C4FM digital,
smoothing the transition for clubs. The WIRES-X network links Fusion repeaters and
"rooms" over the internet.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | [FDMA](/reference/fdma/) |
| Modulation | C4FM |
| Vocoder | AMBE family |
| Modes | Digital voice, data, analog FM mix |

## History

Introduced by Yaesu in the mid-2010s as its entry into amateur digital voice,
competing with [D-STAR](/reference/d-star/) and amateur [DMR](/reference/dmr/).

## Deployment

Amateur radio, via Fusion repeaters, hotspots, and WIRES-X rooms.

## Decoding it with GopherTrunk

YSF shares C4FM modulation with [P25 Phase 1](/reference/p25-phase-1/); see
[Status](/status.html) for GopherTrunk's coverage.
