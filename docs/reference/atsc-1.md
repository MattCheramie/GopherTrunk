---
slug: atsc-1
title: ATSC 1.0
entry_type: protocol
category: broadcast
description: "ATSC 1.0 is the North American digital terrestrial television standard, carrying MPEG-2 video over a single-carrier 8VSB waveform with trellis-coded modulation and Reed-Solomon FEC."
keywords: ATSC, ATSC 1.0, 8VSB, digital terrestrial television, DTV, trellis-coded modulation, Reed-Solomon, MPEG-2, A/53, vestigial sideband, North American digital TV
aka: [ATSC 1.0, ATSC, 8VSB DTV]
autolink: true
infobox:
  - { label: Type, value: Digital terrestrial television }
  - { label: Standards body, value: "ATSC (A/53)" }
  - { label: Introduced, value: "1996 (US adoption)" }
  - { label: Access, value: Broadcast (one-to-many) }
  - { label: Channel spacing, value: "6 MHz" }
  - { label: Modulation, value: "8VSB (single carrier)" }
  - { label: GopherTrunk support, value: Not decoded (out of scope) }
see_also: [vestigial-sideband, trellis-coded-modulation, reed-solomon-code, atsc-3, dvb-t]
cite_urls:
  - https://en.wikipedia.org/wiki/ATSC_standards
  - https://www.atsc.org/atsc-documents/type/a53-atsc-digital-television-standard/
---

**ATSC 1.0** is the first-generation digital terrestrial television standard of the
Advanced Television Systems Committee, used for over-the-air TV in the United States,
Canada, Mexico, and South Korea.[^wiki] Unlike the multi-carrier COFDM systems used
in Europe, ATSC 1.0 transmits an MPEG-2 transport stream on a **single carrier** using
eight-level [vestigial sideband](/reference/vestigial-sideband/) modulation (8VSB),
protected by [trellis-coded modulation](/reference/trellis-coded-modulation/) and a
[Reed–Solomon](/reference/reed-solomon-code/) outer code.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="An eight-level pulse-amplitude signal with a pilot tone at the band edge, illustrating the 8VSB waveform used by ATSC 1.0." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="80" x2="300" y2="80" stroke="currentColor" stroke-opacity="0.4"/>
  <g stroke="currentColor" stroke-width="1.3" stroke-dasharray="2 4" stroke-opacity="0.5">
    <line x1="30" y1="30" x2="300" y2="30"/><line x1="30" y1="44" x2="300" y2="44"/><line x1="30" y1="58" x2="300" y2="58"/><line x1="30" y1="72" x2="300" y2="72"/><line x1="30" y1="88" x2="300" y2="88"/><line x1="30" y1="102" x2="300" y2="102"/><line x1="30" y1="116" x2="300" y2="116"/><line x1="30" y1="130" x2="300" y2="130"/>
  </g>
  <polyline points="40,44 65,116 90,30 115,88 140,58 165,130 190,72 215,44 240,102 265,58 290,116" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <text x="165" y="147" text-anchor="middle" font-size="9" fill="currentColor">8 amplitude levels = 3 bits/symbol</text>
  <line x1="335" y1="130" x2="335" y2="40" stroke="currentColor" stroke-width="2"/>
  <line x1="335" y1="40" x2="430" y2="40" stroke="currentColor" stroke-opacity="0.5"/>
  <line x1="335" y1="130" x2="430" y2="130" stroke="currentColor" stroke-opacity="0.4"/>
  <text x="380" y="34" text-anchor="middle" font-size="8" fill="currentColor">pilot</text>
  <text x="385" y="122" text-anchor="middle" font-size="8" fill="currentColor">6 MHz VSB channel</text>
</svg>
<figcaption>ATSC 1.0 sends a single-carrier eight-level VSB signal with a small in-band pilot to aid carrier recovery.</figcaption>
</figure>

## Overview

8VSB maps three bits to each of eight amplitude levels at 10.76 million symbols per
second, filling a 6 MHz channel and delivering about 19.4 Mbit/s of payload — enough
for one HDTV program or several standard-definition subchannels. A small pilot tone
near the lower band edge gives the receiver a reference for carrier recovery. The
choice of a single-carrier waveform, rather than OFDM, was deliberate: it offers a
higher tolerance to certain impulsive noise but leaves the receiver's adaptive
equalizer to fight the multipath that COFDM handles with a guard interval — the source
of ATSC's early reputation for difficult indoor reception.

## Technical characteristics

| Property | Value |
|----------|-------|
| Modulation | 8VSB (8-level vestigial sideband) |
| Symbol rate | 10.76 Msym/s |
| Inner code | 2/3-rate [trellis-coded modulation](/reference/trellis-coded-modulation/) |
| Outer code | [Reed–Solomon](/reference/reed-solomon-code/) (207,187) |
| Interleaving | 52-segment convolutional |
| Payload | 19.39 Mbit/s MPEG-2 transport stream |
| Video | MPEG-2 (H.264 for mobile ATSC-M/H) |

## History

The ATSC standard A/53 was completed in 1995 and adopted by the US FCC in 1996;
regular broadcasts began in 1998, and the US analog shutdown followed in 2009.[^atsc]
8VSB was selected over a COFDM proposal after a contested comparison, a decision that
shaped a decade of receiver-equalizer research aimed at closing its multipath gap
against DVB-T.

## Deployment

ATSC 1.0 is the terrestrial DTV system of the United States, Canada, Mexico, South
Korea, and a few other countries. It is now being overlaid and gradually succeeded by
[ATSC 3.0](/reference/atsc-3/), an OFDM-based next-generation system, though the two
are not backward compatible and 1.0 broadcasts continue during the transition.

## Decoding it with GopherTrunk

GopherTrunk does **not** decode ATSC 1.0 — MPEG-2 television is far outside its
land-mobile trunking focus. The 6 MHz 8VSB signal can be captured with a wideband
[software-defined radio](/reference/software-defined-radio/) and demodulated in
purpose-built tools, but the channel is too wide for a narrowband
[RTL-SDR](/reference/rtl-sdr/) to take in at once. The modern successor is documented
under [ATSC 3.0](/reference/atsc-3/).

## Sources

[^wiki]: [ATSC standards](https://en.wikipedia.org/wiki/ATSC_standards) — Wikipedia, for the ATSC 1.0 system, its 8VSB waveform, trellis coding, and Reed–Solomon FEC.
[^atsc]: [A/53 ATSC Digital Television Standard](https://www.atsc.org/atsc-documents/type/a53-atsc-digital-television-standard/) — ATSC, the primary standard defining ATSC 1.0 modulation and channel coding.
