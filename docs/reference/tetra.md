---
slug: tetra
title: TETRA
entry_type: protocol
category: protocols
description: TETRA (Terrestrial Trunked Radio) is an ETSI digital trunked-radio standard using four-slot TDMA and π/4-DQPSK, widely used by public safety and transport outside North America.
keywords: TETRA, Terrestrial Trunked Radio, ETSI, four-slot TDMA, pi/4-DQPSK, ACELP, public safety Europe
aka: [TETRA, Terrestrial Trunked Radio]
autolink: true
infobox:
  - { label: Type, value: Digital trunked radio }
  - { label: Standards body, value: ETSI }
  - { label: Introduced, value: "1995" }
  - { label: Access, value: TDMA (4 slots) }
  - { label: Channel spacing, value: 25 kHz }
  - { label: Modulation, value: π/4-DQPSK }
  - { label: Vocoder, value: ACELP }
  - { label: GopherTrunk support, value: See Status }
see_also: [trunked-radio, tdma, phase-shift-keying, control-channel, etsi]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/protocol-landscape/ }
  - { title: "What is trunked radio?", url: /learn/what-is-trunking/ }
external:
  - { title: "Terrestrial Trunked Radio (Wikipedia)", url: https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio }
---

**TETRA** (**Terrestrial Trunked Radio**) is an [ETSI](/reference/etsi/) digital
[trunked-radio](/reference/trunked-radio/) standard built for public-safety and
professional users, especially in Europe and much of the world outside North America.
It uses **four-slot [TDMA](/reference/tdma/)** and π/4-DQPSK
([phase-shift keying](/reference/phase-shift-keying/)).

## Overview

TETRA is a complete system standard — not just an air interface — with rich features:
group and individual calls, direct mode, packet data, and strong security. Four
timeslots per 25 kHz carrier give high capacity.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | TDMA, 4 slots |
| Channel | 25 kHz |
| Modulation | π/4-DQPSK (18 kbps gross) |
| Vocoder | ACELP |

## History

Standardised by ETSI from the mid-1990s; adopted broadly by European emergency
services, transport, and military/government users.

## Deployment

National public-safety networks across Europe, the Middle East, Asia, and elsewhere,
plus transport and utilities. Rare in North America, where [P25](/reference/project-25/)
dominates public safety.

## Decoding it with GopherTrunk

TETRA uses a distinct modulation and vocoder; consult the [Status](/status.html) page
for GopherTrunk's current TETRA coverage.
