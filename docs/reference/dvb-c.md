---
slug: dvb-c
title: DVB-C / DVB-C2
entry_type: protocol
category: broadcast
description: "DVB-C is the European standard for digital cable television, delivering MPEG video over a coaxial network using high-order QAM in a fixed channel raster."
keywords: DVB-C, DVB-C2, digital cable television, QAM, 256-QAM, ETSI, EN 300 429, cable TV, MPEG transport stream, HFC network
aka: [DVB-C, DVB-C2]
autolink: true
infobox:
  - { label: Type, value: Digital cable television }
  - { label: Standards body, value: "ETSI (DVB Project)" }
  - { label: Introduced, value: "1994 (C), 2009 (C2)" }
  - { label: Access, value: Broadcast over cable (HFC) }
  - { label: Channel spacing, value: "6 / 7 / 8 MHz" }
  - { label: Modulation, value: "16- to 256-QAM (C); OFDM+QAM (C2)" }
  - { label: GopherTrunk support, value: Not decoded (out of scope) }
see_also: [quadrature-amplitude-modulation, dvb-t, dvb-s, reed-solomon-code]
cite_urls:
  - https://en.wikipedia.org/wiki/DVB-C
  - https://www.etsi.org/deliver/etsi_en/300400_300499/300429/
---

**DVB-C** (Digital Video Broadcasting — Cable) is the [ETSI](/reference/etsi/)
standard for digital television carried over hybrid fibre-coaxial cable networks.[^wiki]
Because a cable plant is a benign, shielded channel with a high signal-to-noise
ratio and little multipath, DVB-C dispenses with the heavy protection of its
terrestrial and satellite siblings and uses dense
[quadrature-amplitude modulation](/reference/quadrature-amplitude-modulation/) — up
to 256-QAM — on a single carrier to maximise bits per hertz.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A 64-QAM constellation grid of evenly spaced points, showing how DVB-C packs many bits per symbol on a clean cable channel." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="80" x2="250" y2="80" stroke="currentColor" stroke-opacity="0.35"/>
  <line x1="140" y1="15" x2="140" y2="145" stroke="currentColor" stroke-opacity="0.35"/>
  <g fill="currentColor">
    <circle cx="65" cy="35" r="3"/><circle cx="90" cy="35" r="3"/><circle cx="115" cy="35" r="3"/><circle cx="140" cy="35" r="3"/><circle cx="165" cy="35" r="3"/><circle cx="190" cy="35" r="3"/><circle cx="215" cy="35" r="3"/>
    <circle cx="65" cy="57" r="3"/><circle cx="90" cy="57" r="3"/><circle cx="115" cy="57" r="3"/><circle cx="140" cy="57" r="3"/><circle cx="165" cy="57" r="3"/><circle cx="190" cy="57" r="3"/><circle cx="215" cy="57" r="3"/>
    <circle cx="65" cy="80" r="3"/><circle cx="90" cy="80" r="3"/><circle cx="115" cy="80" r="3"/><circle cx="165" cy="80" r="3"/><circle cx="190" cy="80" r="3"/><circle cx="215" cy="80" r="3"/>
    <circle cx="65" cy="103" r="3"/><circle cx="90" cy="103" r="3"/><circle cx="115" cy="103" r="3"/><circle cx="140" cy="103" r="3"/><circle cx="165" cy="103" r="3"/><circle cx="190" cy="103" r="3"/><circle cx="215" cy="103" r="3"/>
    <circle cx="65" cy="125" r="3"/><circle cx="90" cy="125" r="3"/><circle cx="115" cy="125" r="3"/><circle cx="140" cy="125" r="3"/><circle cx="165" cy="125" r="3"/><circle cx="190" cy="125" r="3"/><circle cx="215" cy="125" r="3"/>
  </g>
  <text x="140" y="158" text-anchor="middle" font-size="9" fill="currentColor">dense QAM grid — up to 8 bits/symbol (256-QAM)</text>
  <text x="360" y="70" text-anchor="middle" font-size="9" fill="currentColor">clean coax channel →</text>
  <text x="360" y="86" text-anchor="middle" font-size="9" fill="currentColor">high SNR, no multipath</text>
</svg>
<figcaption>On a shielded cable plant DVB-C can use very dense QAM, trading robustness for spectral efficiency the air-interface variants cannot risk.</figcaption>
</figure>

## Overview

DVB-C fits into the same 6, 7, or 8 MHz channel raster as analog cable, so operators
could overlay digital services on existing plant. It uses a single-carrier QAM
waveform — 16-, 32-, 64-, 128-, or 256-QAM — with only a light forward-error-correction
layer: an outer [Reed–Solomon](/reference/reed-solomon-code/) code plus byte
interleaving, and no inner convolutional code, because the cable channel rarely needs
one. That simplicity yields the highest payload of the DVB family for a given
bandwidth: a 256-QAM, 8 MHz channel carries roughly 51 Mbit/s.

## Technical characteristics

| Property | Value |
|----------|-------|
| Modulation | 16/32/64/128/256-QAM (single carrier) |
| FEC | [Reed–Solomon](/reference/reed-solomon-code/) (204,188) + interleaving |
| Roll-off | 15% root-raised-cosine |
| Symbol rate | ≈ 6.9 Msym/s (typical, 8 MHz) |
| Payload | MPEG-2 transport stream |
| Peak bitrate | ≈ 51 Mbit/s (256-QAM, 8 MHz) |
| C2 waveform | OFDM with QAM subcarriers, LDPC/BCH FEC |

## History

DVB-C was published as ETSI EN 300 429 in 1994, alongside the satellite and
terrestrial standards, and became the basis of digital cable worldwide.[^etsi] The
DVB-C2 extension (EN 302 769, 2009) reworked the physical layer around OFDM with QAM
subcarriers and modern LDPC/BCH coding, adding up to 4096-QAM and lifting capacity by
roughly 30%, though C2 has seen limited deployment as operators moved payload to
DOCSIS and IP delivery.

## Deployment

DVB-C is widely deployed by cable operators across Europe and parts of Asia. Because
it shares the QAM technique used by the North American cable and DOCSIS systems, the
underlying receiver design is closely related, differing mainly in channel width and
FEC framing. Set-top boxes tune a QAM channel, recover the transport stream, and
demultiplex the selected service.

## Decoding it with GopherTrunk

DVB-C is a wired, in-cable signal and is entirely outside GopherTrunk's scope, which
targets over-the-air land-mobile trunking. Even for enthusiasts it is less accessible
than the broadcast variants, since it exists only inside a coaxial network rather than
radiating over the air. For the on-air members of the family, see
[DVB-T](/reference/dvb-t/) and [DVB-S](/reference/dvb-s/).

## Sources

[^wiki]: [DVB-C](https://en.wikipedia.org/wiki/DVB-C) — Wikipedia, for the cable DVB standard, its single-carrier QAM waveform, and the DVB-C2 successor.
[^etsi]: [EN 300 429 (DVB-C)](https://www.etsi.org/deliver/etsi_en/300400_300499/300429/) — ETSI, the primary standard defining DVB-C framing, QAM modulation, and Reed–Solomon coding.
