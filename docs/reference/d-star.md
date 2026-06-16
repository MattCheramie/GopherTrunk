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
