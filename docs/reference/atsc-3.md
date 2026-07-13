---
slug: atsc-3
title: ATSC 3.0 (NextGen TV)
entry_type: protocol
category: broadcast
description: "ATSC 3.0, marketed as NextGen TV, is the IP-based North American digital television standard using an OFDM physical layer with LDPC coding to deliver HEVC video over the air."
keywords: ATSC 3.0, NextGen TV, digital terrestrial television, OFDM, LDPC, HEVC, ROUTE, MMT, IP delivery, A/322, bootstrap, North American digital TV
aka: [ATSC 3.0, NextGen TV]
autolink: true
infobox:
  - { label: Type, value: Digital terrestrial television (IP-based) }
  - { label: Standards body, value: "ATSC (A/300 suite)" }
  - { label: Introduced, value: "2017 (published), 2020 (US launch)" }
  - { label: Access, value: Broadcast (one-to-many), IP transport }
  - { label: Channel spacing, value: "6 MHz" }
  - { label: Modulation, value: "OFDM, QPSK–4096-QAM subcarriers" }
  - { label: GopherTrunk support, value: Not decoded (out of scope) }
see_also: [ofdm, ldpc-code, atsc-1, dvb-t]
cite_urls:
  - https://en.wikipedia.org/wiki/ATSC_3.0
  - https://www.atsc.org/atsc-30-standard/
---

**ATSC 3.0**, marketed in the United States as **NextGen TV**, is the
next-generation North American digital terrestrial television standard.[^wiki] It
breaks compatibility with [ATSC 1.0](/reference/atsc-1/) to adopt an
[OFDM](/reference/ofdm/) physical layer with modern
[LDPC](/reference/ldpc-code/) forward error correction, and it carries everything —
video, audio, data, and even interactive apps — as **IP packets** rather than an MPEG
transport stream, unifying broadcast and broadband delivery.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A layered stack showing ATSC 3.0 delivering IP packets over ROUTE or MMT transport on an OFDM physical layer with LDPC coding." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1" font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="120" y="20" width="220" height="24" fill="currentColor" fill-opacity="0.05"/><text x="230" y="36">HEVC video · AC-4 / MPEG-H audio</text>
    <rect x="120" y="48" width="220" height="24" fill="currentColor" fill-opacity="0.1"/><text x="230" y="64">IP packets (broadcast + broadband)</text>
    <rect x="120" y="76" width="220" height="24" fill="currentColor" fill-opacity="0.15"/><text x="230" y="92">ROUTE / DASH · MMT transport</text>
    <rect x="120" y="104" width="220" height="30" fill="currentColor" fill-opacity="0.22"/><text x="230" y="123">OFDM physical layer · LDPC + BCH FEC</text>
  </g>
  <line x1="230" y1="134" x2="230" y2="150" stroke="currentColor" marker-end="url(#atsc3ar)"/>
  <text x="230" y="147" text-anchor="middle" font-size="8" fill="currentColor"> </text>
  <text x="70" y="122" text-anchor="middle" font-size="8" fill="currentColor">6 MHz air</text>
  <defs><marker id="atsc3ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>ATSC 3.0 stacks IP-based media over ROUTE or MMT transport on a flexible OFDM/LDPC physical layer.</figcaption>
</figure>

## Overview

ATSC 3.0 is defined as a suite of documents (the A/300 series) rather than a single
spec, reflecting its modular design. The physical layer (A/322) is a highly
configurable OFDM waveform: subcarrier constellations from QPSK up to 4096-QAM,
several LDPC code rates, and multiple pilot and guard-interval patterns let a
broadcaster trade robustness against capacity — even splitting one 6 MHz channel into
layered pipes, a rugged one for mobile reception and a high-capacity one for fixed
4K. Every emission begins with a fixed **bootstrap** signal that a receiver can detect
regardless of the service configuration that follows.

## Technical characteristics

| Property | Value |
|----------|-------|
| Waveform | OFDM (bootstrap + configurable payload) |
| Subcarrier modulation | QPSK, 16- to 4096-QAM (non-uniform constellations) |
| Inner FEC | [LDPC](/reference/ldpc-code/) (multiple rates) |
| Outer FEC | BCH / CRC |
| Transport | IP via ROUTE/DASH and MMT |
| Video / audio | HEVC (H.265); Dolby AC-4 / MPEG-H |
| Channel | 6 MHz (US); layered/robust modes |

## History

ATSC finalised the ATSC 3.0 standards in 2017, and the first US commercial NextGen TV
stations launched in 2020.[^atsc] South Korea deployed ATSC 3.0 ahead of the US, using
it for the 2018 Winter Olympics. Because it is not backward compatible with ATSC 1.0,
US broadcasters run the two in parallel, sharing spectrum during a lengthy voluntary
transition.

## Deployment

ATSC 3.0 is rolling out across US markets and is deployed in South Korea. Its IP
foundation enables features beyond ATSC 1.0: 4K HDR video, immersive audio, targeted
advertising, datacasting, and hooks for future services such as broadcast-assisted
positioning. Adoption depends on receiver penetration and on stations completing the
1.0-to-3.0 spectrum shuffle.

## Decoding it with GopherTrunk

GopherTrunk does **not** decode ATSC 3.0; IP-based HEVC television is well outside its
land-mobile trunking scope. As with the other digital-TV systems, the RF can be
captured by a wideband [software-defined radio](/reference/software-defined-radio/)
and processed in dedicated OFDM tools, but the full 6 MHz channel exceeds a
narrowband dongle's bandwidth. The first-generation system it replaces is documented
under [ATSC 1.0](/reference/atsc-1/).

## Sources

[^wiki]: [ATSC 3.0](https://en.wikipedia.org/wiki/ATSC_3.0) — Wikipedia, for the NextGen TV system, its OFDM physical layer, LDPC coding, and IP transport.
[^atsc]: [ATSC 3.0 standard](https://www.atsc.org/atsc-30-standard/) — ATSC, the primary document suite defining the ATSC 3.0 physical layer and delivery stack.
