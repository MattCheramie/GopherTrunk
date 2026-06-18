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
  - { title: "Digital modulation & constellations", url: /learn/rf-sdr/digital-modulation/ }
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
external:
  - { title: "Project 25 (Wikipedia)", url: https://en.wikipedia.org/wiki/Project_25 }
---

**P25 Phase 1** is the first-generation air interface of [Project 25](/reference/project-25/),
using **[FDMA](/reference/fdma/)** — one conversation per 12.5 kHz channel — with
[C4FM](/reference/c4fm/) modulation and the [IMBE](/reference/imbe/) vocoder.

<figure class="figure" markdown="0">
<svg viewBox="0 0 360 150" role="img" aria-label="Stacked 12.5 kHz FDMA channels each carrying one P25 Phase 1 call." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="135" x2="40" y2="20" stroke="currentColor" stroke-opacity="0.4" marker-end="url(#fa_p25-phase-1)"/>
  <text x="22" y="80" font-size="9" fill="currentColor" transform="rotate(-90 22 80)">frequency</text>
  <g fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1">
    <rect x="50" y="28" width="260" height="22"/><rect x="50" y="60" width="260" height="22"/><rect x="50" y="92" width="260" height="22"/>
  </g>
  <g font-size="8.5" fill="currentColor"><text x="180" y="43" text-anchor="middle">one call per channel (12.5 kHz)</text><text x="180" y="75" text-anchor="middle">one call per channel</text><text x="180" y="107" text-anchor="middle">one call per channel</text></g>
  <defs><marker id="fa_p25-phase-1" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>P25 Phase 1 is FDMA: each call occupies its own 12.5 kHz channel.</figcaption>
</figure>

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
