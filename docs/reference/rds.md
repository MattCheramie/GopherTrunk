---
slug: rds
title: Radio Data System (RDS / RBDS)
entry_type: protocol
category: broadcast
description: "RDS is the low-rate digital data protocol on FM broadcast's 57 kHz subcarrier, carrying station identity (PI/PS), radio text, and traffic flags."
keywords: RDS, Radio Data System, RBDS, 57 kHz subcarrier, PI code, PS name, RadioText, RT, traffic announcement, TMC, FM data, CENELEC EN 50067
aka: [RDS, RBDS, Radio Data System]
autolink: true
infobox:
  - { label: Type, value: FM broadcast data protocol }
  - { label: Standards body, value: "CENELEC / IEC (RBDS: NRSC)" }
  - { label: Introduced, value: "1984" }
  - { label: Access, value: Continuous data on a subcarrier }
  - { label: Channel spacing, value: "57 kHz subcarrier within FM MPX" }
  - { label: Modulation, value: DSB-SC amplitude modulation, 1187.5 bps }
  - { label: Vocoder, value: None (data only) }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [broadcast-fm, subcarrier, pre-emphasis-de-emphasis, frequency-modulation, software-defined-radio]
cite_urls:
  - https://en.wikipedia.org/wiki/Radio_Data_System
  - https://en.wikipedia.org/wiki/RBDS
---

## Overview

**Radio Data System** (**RDS**) is the digital sideband protocol that lets an
[FM broadcast](/reference/broadcast-fm/) station send small amounts of data alongside its
audio, most visibly the scrolling station name and song title on a car radio.[^wiki] It
rides on a 57 kHz [subcarrier](/reference/subcarrier/) — the third harmonic of the FM
stereo pilot — carrying just 1187.5 bits per second. The North American variant, **RBDS**,
is functionally compatible with minor differences in program-type tables. RDS is what
turns an FM tuner from a dial into a device that knows what it is listening to.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="The RDS 57 kHz subcarrier sitting above the FM stereo multiplex, expanded to show a 104-bit group split into four blocks each carrying a checkword." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="rdsa" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <text x="70" y="30" text-anchor="middle" font-size="8" fill="currentColor">FM MPX</text>
  <rect x="30" y="40" width="60" height="30" fill="currentColor" fill-opacity="0.15" stroke="currentColor"/>
  <text x="60" y="60" text-anchor="middle" font-size="7" fill="currentColor">audio</text>
  <rect x="100" y="40" width="34" height="30" fill="currentColor" fill-opacity="0.4" stroke="currentColor"/>
  <text x="117" y="34" text-anchor="middle" font-size="7" fill="currentColor">57k</text>
  <line x1="134" y1="55" x2="180" y2="55" stroke="currentColor" marker-end="url(#rdsa)"/>
  <g font-size="7" fill="currentColor" text-anchor="middle">
    <rect x="190" y="45" width="55" height="26" fill="none" stroke="currentColor"/><text x="217" y="61">Block A</text>
    <rect x="245" y="45" width="55" height="26" fill="none" stroke="currentColor"/><text x="272" y="61">Block B</text>
    <rect x="300" y="45" width="55" height="26" fill="none" stroke="currentColor"/><text x="327" y="61">Block C</text>
    <rect x="355" y="45" width="55" height="26" fill="none" stroke="currentColor"/><text x="382" y="61">Block D</text>
  </g>
  <text x="300" y="95" text-anchor="middle" font-size="8" fill="currentColor">one 104-bit group = 4 × (16 data + 10 checkword) bits</text>
</svg>
<figcaption>RDS data on the 57 kHz subcarrier is framed into 104-bit groups of four blocks, each with an error-correcting checkword.</figcaption>
</figure>

## Technical characteristics

| Property | Value |
|----------|-------|
| Subcarrier | 57 kHz, DSB-SC, phase-locked to the 19 kHz pilot |
| Data rate | 1187.5 bps (pilot ÷ 48) |
| Line coding | Differential, then biphase (Manchester-style) |
| Frame | 104-bit group = 4 blocks × (16 data + 10 check) |
| Error control | Shortened cyclic code with offset words |
| Key fields | PI (identity), PS (8-char name), RT (64-char text), PTY, TA/TP, CT (clock), AF |

## History

RDS grew out of European broadcasting research in the late 1970s and was standardised by
CENELEC in 1984 (EN 50067, later IEC 62106). The United States adopted the compatible RBDS
standard through the NRSC in 1992. Later additions layered richer services onto the same
groups, including the Traffic Message Channel (TMC) for navigation systems and RDS2, which
adds extra subcarriers for higher throughput and even small images.

## Deployment

RDS is near-universal on FM broadcasting worldwide, driving the station-name and
radio-text displays on virtually every modern receiver, plus automatic frequency
following (AF), traffic-announcement interrupts (TA/TP), and clock setting (CT). It is a
data-only protocol — there is no voice component — and it degrades gracefully, with the
checkword-protected groups simply being discarded when reception is poor.

## Decoding it with GopherTrunk

**GopherTrunk** does not decode RDS. GopherTrunk is a trunked land-mobile scanner for
systems such as P25, DMR, NXDN, and TETRA; RDS belongs to the FM broadcast world, which is
out of its scope. RDS is straightforward for general-purpose SDR software to decode —
demodulate the 57 kHz [subcarrier](/reference/subcarrier/), recover the biphase clock,
differentially decode, and check each block against its offset word — and several
open-source tools (e.g. redsea) do exactly this from an FM receiver's baseband.

## Sources

[^wiki]: [Radio Data System](https://en.wikipedia.org/wiki/Radio_Data_System) — Wikipedia, for the 57 kHz subcarrier, 1187.5 bps rate, the 104-bit group/block/checkword structure, and the PI/PS/RT/TMC data fields.
