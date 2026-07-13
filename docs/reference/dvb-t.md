---
slug: dvb-t
title: DVB-T / DVB-T2
entry_type: protocol
category: broadcast
description: "DVB-T and DVB-T2 are the European standards for digital terrestrial television, carrying MPEG/H.264/HEVC video over a COFDM waveform with QAM subcarriers."
keywords: DVB-T, DVB-T2, digital terrestrial television, DTT, COFDM, OFDM, QAM, ETSI, EN 300 744, EN 302 755, terrestrial TV, MPEG transport stream
aka: [DVB-T, DVB-T2, DTT]
autolink: true
infobox:
  - { label: Type, value: Digital terrestrial television }
  - { label: Standards body, value: "ETSI (DVB Project)" }
  - { label: Introduced, value: "1997 (T), 2009 (T2)" }
  - { label: Access, value: Broadcast (one-to-many) }
  - { label: Channel spacing, value: "6 / 7 / 8 MHz" }
  - { label: Modulation, value: "COFDM, QPSK–256-QAM subcarriers" }
  - { label: GopherTrunk support, value: Not decoded (out of scope) }
see_also: [ofdm, quadrature-amplitude-modulation, reed-solomon-code, ldpc-code, dvb-s, dvb-c]
cite_urls:
  - https://en.wikipedia.org/wiki/DVB-T
  - https://www.etsi.org/deliver/etsi_en/300700_300799/300744/
---

**DVB-T** (Digital Video Broadcasting — Terrestrial) and its successor **DVB-T2**
are the [ETSI](/reference/etsi/) standards for over-the-air digital television used
across Europe, much of Asia, Africa, and Australia.[^wiki] They carry compressed
video and audio in an MPEG-2 transport stream over a **coded orthogonal
frequency-division multiplexing** ([COFDM](/reference/ofdm/)) waveform, spreading the
signal across thousands of [QAM](/reference/quadrature-amplitude-modulation/)
subcarriers so that a single 8 MHz channel survives the multipath of rooftop and
indoor reception.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A DVB-T channel spectrum showing thousands of closely spaced OFDM subcarriers filling one 8 MHz television channel." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="115" x2="440" y2="115" stroke="currentColor" stroke-opacity="0.5" marker-end="url(#dvbtar)"/>
  <line x1="30" y1="115" x2="30" y2="25" stroke="currentColor" stroke-opacity="0.5"/>
  <g stroke="currentColor" stroke-width="1.4">
    <line x1="60" y1="115" x2="60" y2="45"/><line x1="72" y1="115" x2="72" y2="42"/><line x1="84" y1="115" x2="84" y2="46"/><line x1="96" y1="115" x2="96" y2="41"/><line x1="108" y1="115" x2="108" y2="44"/><line x1="120" y1="115" x2="120" y2="43"/><line x1="132" y1="115" x2="132" y2="45"/><line x1="144" y1="115" x2="144" y2="42"/><line x1="156" y1="115" x2="156" y2="46"/><line x1="168" y1="115" x2="168" y2="43"/><line x1="180" y1="115" x2="180" y2="44"/><line x1="192" y1="115" x2="192" y2="42"/><line x1="204" y1="115" x2="204" y2="45"/><line x1="216" y1="115" x2="216" y2="43"/><line x1="228" y1="115" x2="228" y2="46"/><line x1="240" y1="115" x2="240" y2="41"/><line x1="252" y1="115" x2="252" y2="44"/><line x1="264" y1="115" x2="264" y2="43"/><line x1="276" y1="115" x2="276" y2="45"/><line x1="288" y1="115" x2="288" y2="42"/><line x1="300" y1="115" x2="300" y2="46"/><line x1="312" y1="115" x2="312" y2="43"/><line x1="324" y1="115" x2="324" y2="44"/><line x1="336" y1="115" x2="336" y2="42"/><line x1="348" y1="115" x2="348" y2="45"/><line x1="360" y1="115" x2="360" y2="43"/><line x1="372" y1="115" x2="372" y2="46"/><line x1="384" y1="115" x2="384" y2="44"/>
  </g>
  <line x1="54" y1="30" x2="390" y2="30" stroke="currentColor" stroke-dasharray="3 3" stroke-opacity="0.6"/>
  <text x="222" y="22" text-anchor="middle" font-size="9" fill="currentColor">≈ 8 MHz channel · up to 32 k QAM subcarriers</text>
  <text x="235" y="138" text-anchor="middle" font-size="9" fill="currentColor">frequency →</text>
  <defs><marker id="dvbtar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>DVB-T fills a single TV channel with thousands of orthogonal QAM subcarriers, trading raw speed for multipath immunity.</figcaption>
</figure>

## Overview

DVB-T packs digital television into the 6, 7, or 8 MHz raster once used by a single
analog channel. Rather than one high-rate carrier, it distributes the payload over
1,705 subcarriers ("2K mode") or 6,817 ("8K mode"); DVB-T2 extends this to 32K.
Each subcarrier is modulated with QPSK, 16-QAM, or 64-QAM (T2 adds 256-QAM). A
**guard interval** — a cyclic repetition prepended to every OFDM symbol — absorbs
echoes from reflections and from other transmitters in a single-frequency network,
which is what lets terrestrial TV work with a simple indoor antenna.

## Technical characteristics

| Property | Value |
|----------|-------|
| Waveform | COFDM (OFDM + FEC) |
| Subcarriers | 2K / 8K (DVB-T); 1K–32K (DVB-T2) |
| Subcarrier modulation | QPSK, 16-QAM, 64-QAM; +256-QAM in T2 |
| Inner code | Convolutional (T), [LDPC](/reference/ldpc-code/) (T2) |
| Outer code | [Reed–Solomon](/reference/reed-solomon-code/) (T); BCH (T2) |
| Guard interval | 1/4, 1/8, 1/16, 1/32 of symbol |
| Payload | MPEG-2 transport stream (MPEG-2, H.264, HEVC video) |
| Peak bitrate | ≈ 31 Mbit/s (T), ≈ 50 Mbit/s (T2, 8 MHz) |

## History

The DVB Project published DVB-T as ETSI EN 300 744 in 1997, and the first services
launched in the UK in 1998.[^etsi] It pairs concatenated forward error correction —
an outer Reed–Solomon code over an inner convolutional code — with COFDM. DVB-T2
(EN 302 755, 2009) replaced that stack with a far stronger BCH-plus-LDPC scheme
borrowed from the DVB-S2 satellite standard, raising capacity by roughly 30–50% and
enabling terrestrial HD and UHD.

## Deployment

DVB-T and DVB-T2 are the dominant terrestrial systems outside the Americas, Japan,
and China. Most European countries have migrated, or are migrating, to DVB-T2 with
HEVC to reclaim spectrum for mobile broadband. Transmitters are often arranged in
single-frequency networks, where many sites share one channel and the guard interval
turns overlapping signals into constructive multipath rather than interference.

## Decoding it with GopherTrunk

GopherTrunk is a land-mobile trunking scanner and does **not** demodulate DVB-T; MPEG
transport-stream video is outside its scope. The RF signal is, however, perfectly
receivable with the same [software-defined radio](/reference/software-defined-radio/)
hardware: an 8 MHz DVB-T channel exceeds the ~2.4 MHz bandwidth of an
[RTL-SDR](/reference/rtl-sdr/) — indeed the RTL2832U demodulator was designed for
DVB-T before hobbyists repurposed it — but a wider receiver such as an
[Airspy](/reference/airspy/) or [HackRF](/reference/hackrf/) can capture the full
channel for analysis in dedicated tools. Related terrestrial and satellite variants
are covered under [DVB-C](/reference/dvb-c/) and [DVB-S](/reference/dvb-s/).

## Sources

[^wiki]: [DVB-T](https://en.wikipedia.org/wiki/DVB-T) — Wikipedia, for the terrestrial DVB standard, its COFDM waveform, subcarrier modes, and QAM constellations.
[^etsi]: [EN 300 744 (DVB-T framing and modulation)](https://www.etsi.org/deliver/etsi_en/300700_300799/300744/) — ETSI, the primary standard defining DVB-T channel coding and OFDM modulation.
