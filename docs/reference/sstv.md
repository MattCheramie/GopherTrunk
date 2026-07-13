---
slug: sstv
title: Slow-Scan Television (SSTV)
entry_type: protocol
category: amateur-digital
description: "SSTV (slow-scan television) sends still images over voice-bandwidth radio by mapping pixel brightness and color to an audio subcarrier that frequency-modulates the RF carrier."
keywords: SSTV, slow-scan television, Robot 36, Scottie, Martin, PD120, VIS code, FM subcarrier image, amateur radio image transmission, 14.230 MHz
aka: [SSTV, Slow-Scan TV]
autolink: true
infobox:
  - { label: Type, value: Analog image mode }
  - { label: Standards body, value: Amateur convention (no formal standard) }
  - { label: Introduced, value: 1950s–1960s }
  - { label: Access, value: Simplex voice channel }
  - { label: Channel spacing, value: ~2.5 kHz audio (SSB/FM voice) }
  - { label: Modulation, value: FM audio subcarrier (1500–2300 Hz) }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [frequency-modulation, subcarrier, single-sideband, rtty, morse-code]
cite_urls:
  - https://en.wikipedia.org/wiki/Slow-scan_television
---

**SSTV** (**slow-scan television**) is an amateur-radio mode that sends still images
across an ordinary voice channel by encoding each pixel as an audio tone whose
[frequency](/reference/frequency-modulation/) represents brightness. That audio
[subcarrier](/reference/subcarrier/) then modulates the RF carrier — via SSB on HF or
FM on VHF — so a single frame trickles through in seconds to a couple of minutes rather
than the 30 frames per second of broadcast television.[^wiki] The "slow scan" name
captures exactly that trade: image bandwidth is squeezed down to the ~2.5 kHz a voice
radio already passes.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="SSTV maps line-by-line pixel brightness to audio tone frequency between 1500 and 2300 hertz, with a 1200 hertz sync pulse starting each scan line." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="sstvar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="30" y1="20" x2="30" y2="120" stroke="currentColor" stroke-width="1"/>
  <line x1="30" y1="120" x2="440" y2="120" stroke="currentColor" stroke-width="1" marker-end="url(#sstvar)"/>
  <text x="18" y="24" font-size="8" fill="currentColor" text-anchor="end">2300 Hz</text>
  <text x="18" y="123" font-size="8" fill="currentColor" text-anchor="end">1200 Hz</text>
  <text x="235" y="140" font-size="8.5" fill="currentColor" text-anchor="middle">time along one scan line</text>
  <path d="M40 115 L40 60 L60 60 L60 40 L90 40 L90 80 L130 80 L130 55 L175 55 L175 90 L215 90" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <line x1="40" y1="120" x2="40" y2="110" stroke="currentColor" stroke-width="2"/>
  <text x="120" y="30" font-size="8.5" fill="currentColor" text-anchor="middle">tone = pixel brightness</text>
  <text x="42" y="132" font-size="7.5" fill="currentColor">sync</text>
</svg>
<figcaption>Each scan line begins with a 1200 Hz sync pulse; brightness then rides as a tone swept between roughly 1500 and 2300 Hz.</figcaption>
</figure>

## Overview

An SSTV transmission is a sequence of horizontal scan lines. Before the image, a short
**VIS (Vertical Interval Signaling) code** — a digital header of FSK tones — announces
which mode follows, so the receiver knows the line count, timing, and color order. Each
line then starts with a 1200 Hz **sync pulse**, after which the luminance (and, in color
modes, the separate red/green/blue or Y/R-Y/B-Y components) is sent as a tone gliding
between about 1500 Hz (black) and 2300 Hz (white). The receiver measures instantaneous
frequency, paints one pixel per time step, and assembles the picture line by line.

## Technical characteristics

| Property | Value |
|----------|-------|
| Signal | Audio tone 1500–2300 Hz + 1200 Hz sync |
| Header | VIS code (FSK) identifies the mode |
| Common modes | Robot 36, Scottie 1/2/DX, Martin 1/2, PD90/120/180 |
| Frame time | ~8 s to ~4 min depending on mode |
| RF carrier | SSB on HF, FM on VHF/UHF |
| Popular frequency | 14.230 MHz (20 m), 144.500 MHz FM |

## History

SSTV grew out of 1950s experiments by Copthorne Macdonald, who demonstrated that a
narrow-band, long-persistence image could fit a voice channel. Early systems used
monochrome and slow electromechanical or long-persistence CRT displays; the arrival of
frame stores and, later, PC sound-card software turned it into an accessible color mode.
The International Space Station periodically runs SSTV image events on VHF, drawing large
numbers of casual receivers.[^wiki]

## Deployment

SSTV is purely an amateur and experimental mode. It appears on HF phone segments (notably
14.230 and 14.233 MHz), on VHF FM simplex, and from the ISS. Because it is analog and
tolerant, even a weak or noisy copy still yields a recognizable, if speckled, picture.

## Decoding it with GopherTrunk

GopherTrunk targets digital land-mobile trunking and a handful of data modes; **it does
not decode SSTV**. SSTV is an analog image mode outside GopherTrunk's decode chain, and
it is well served by dedicated tools (MMSSTV, QSSTV, RX-SSTV). The relevant GopherTrunk
building blocks — [FM demodulation](/reference/frequency-modulation/) and
[subcarrier](/reference/subcarrier/) handling — are the same primitives such decoders
rely on, but no SSTV frame decoder ships in GopherTrunk.

## Sources

[^wiki]: [Slow-scan television](https://en.wikipedia.org/wiki/Slow-scan_television) — Wikipedia, for SSTV's tone-to-brightness encoding, VIS headers, mode families, and history.
