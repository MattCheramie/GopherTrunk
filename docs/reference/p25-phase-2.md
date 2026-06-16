---
slug: p25-phase-2
title: P25 Phase 2
entry_type: protocol
category: protocols
description: P25 Phase 2 is the TDMA air interface of Project 25, placing two voice timeslots in a 12.5 kHz channel using H-DQPSK/H-CPM and the AMBE+2 vocoder for doubled spectrum efficiency.
keywords: P25 Phase 2, TDMA, AMBE+2, H-DQPSK, two-slot, spectrum efficiency, public safety
aka: [P25 Phase 2, P25 Phase II, Phase 2 P25]
autolink: true
infobox:
  - { label: Type, value: Digital land-mobile radio }
  - { label: Part of, value: Project 25 }
  - { label: Access, value: TDMA (2 slots) }
  - { label: Channel spacing, value: 12.5 kHz (6.25 kHz equivalent) }
  - { label: Modulation, value: H-DQPSK / H-CPM }
  - { label: Vocoder, value: AMBE+2 }
  - { label: GopherTrunk support, value: Decoded }
see_also: [project-25, p25-phase-1, ambe-plus-2, tdma, control-channel]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/protocol-landscape/ }
  - { title: "Analog vs. digital voice", url: /learn/digital-voice/ }
external:
  - { title: "Project 25 (Wikipedia)", url: https://en.wikipedia.org/wiki/Project_25 }
---

**P25 Phase 2** is the second-generation air interface of [Project 25](/reference/project-25/),
using **two-slot [TDMA](/reference/tdma/)** to carry two simultaneous voice
conversations in a single 12.5 kHz channel — effectively 6.25 kHz per call.

## Overview

Phase 2 was introduced to meet spectrum-efficiency goals. Where
[Phase 1](/reference/p25-phase-1/) gives each call its own frequency, Phase 2 divides
a traffic channel into two repeating timeslots, doubling capacity. It uses the more
efficient [AMBE+2](/reference/ambe-plus-2/) vocoder and a phase-shift modulation
(H-DQPSK on the outbound link, H-CPM on the inbound).

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | TDMA, 2 slots |
| Channel | 12.5 kHz (6.25 kHz equivalent capacity) |
| Modulation | H-DQPSK (outbound) / H-CPM (inbound) |
| Vocoder | AMBE+2 (half-rate) |
| Control channel | Usually a C4FM Phase 1 control channel |

A common deployment keeps a [C4FM](/reference/c4fm/) Phase 1
[control channel](/reference/control-channel/) while voice traffic uses Phase 2 TDMA
slots.

## History

Phase 2 was standardised by the [TIA](/reference/tia/) to follow narrowbanding and
spectrum-efficiency mandates, and has been deployed on large metropolitan and
statewide systems since the early 2010s.

## Deployment

Phase 2 is widely used by busy urban public-safety systems where channel capacity is
at a premium, frequently mixed with Phase 1 on the same network.

## Decoding it with GopherTrunk

GopherTrunk follows the control channel, tunes to the assigned Phase 2 traffic
channel and slot, and decodes the AMBE+2 voice. See [Status](/status.html).
