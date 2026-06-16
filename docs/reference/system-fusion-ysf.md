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
  - { title: "The digital protocol landscape", url: /learn/protocol-landscape/ }
external:
  - { title: "System Fusion (Wikipedia)", url: https://en.wikipedia.org/wiki/System_Fusion }
---

**System Fusion** (**Yaesu System Fusion**, **YSF**) is Yaesu's amateur digital-voice
system, using [C4FM](/reference/c4fm/) modulation and an [AMBE](/reference/ambe/)-family
[vocoder](/reference/vocoder/). It is notable for automatically mixing digital and
analog users on the same repeater.

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
