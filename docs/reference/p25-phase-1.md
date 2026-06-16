---
slug: p25-phase-1
title: P25 Phase 1
entry_type: protocol
category: protocols
description: P25 Phase 1 is the FDMA air interface of Project 25, using C4FM modulation at 4800 baud and the IMBE vocoder in 12.5 kHz channels for North American public-safety radio.
keywords: P25 Phase 1, C4FM, IMBE, FDMA, 9600 bps, public safety, trunking
aka: [P25 Phase 1, P25 Phase I, Phase 1 P25]
autolink: true
infobox:
  - { label: Type, value: Digital land-mobile radio }
  - { label: Part of, value: Project 25 }
  - { label: Access, value: FDMA }
  - { label: Channel spacing, value: 12.5 kHz }
  - { label: Modulation, value: C4FM (4-level FSK) / CQPSK }
  - { label: Symbol rate, value: 4800 baud (9600 bps) }
  - { label: Vocoder, value: IMBE }
  - { label: GopherTrunk support, value: Decoded }
see_also: [project-25, p25-phase-2, c4fm, imbe, fdma, control-channel]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/digital-modulation/ }
  - { title: "What is trunked radio?", url: /learn/what-is-trunking/ }
external:
  - { title: "Project 25 (Wikipedia)", url: https://en.wikipedia.org/wiki/Project_25 }
---

**P25 Phase 1** is the first-generation air interface of [Project 25](/reference/project-25/),
using **[FDMA](/reference/fdma/)** — one conversation per 12.5 kHz channel — with
[C4FM](/reference/c4fm/) modulation and the [IMBE](/reference/imbe/) vocoder.

## Overview

In Phase 1 each call occupies its own frequency. A trunked Phase 1 system uses a
dedicated [control channel](/reference/control-channel/) to assign callers to voice
channels; conventional Phase 1 simply transmits on a fixed frequency. It is the most
widely deployed P25 variant and the baseline that Phase 2 builds on.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | FDMA |
| Channel | 12.5 kHz |
| Modulation | C4FM (4-level FSK); CQPSK on the linear path |
| Symbol rate | 4800 [baud](/reference/symbol-rate/) → 9600 bps |
| Vocoder | IMBE (7.2 kbps incl. FEC) |
| Error correction | Golay, Hamming, Reed–Solomon, trellis (by field) |

C4FM and CQPSK are designed to be detected by the same receiver, so a single
demodulator can handle both transmit paths.

## History

Phase 1 documents were published by the [TIA](/reference/tia/) starting in the
mid-1990s and saw broad public-safety adoption through the 2000s as agencies migrated
from analog and proprietary systems.

## Deployment

Phase 1 underpins many statewide and municipal public-safety networks across North
America, often alongside Phase 2 voice channels on the same system.

## Decoding it with GopherTrunk

GopherTrunk demodulates the C4FM symbols, recovers the IMBE frames, and synthesises
audio. The [constellation](/reference/constellation-diagram/) and
[eye diagram](/reference/eye-diagram/) views help confirm a clean lock. See
[Status](/status.html) for details.
