---
slug: 5g-nr
title: 5G NR (New Radio)
entry_type: protocol
category: cellular
description: 5G NR is the 3GPP fifth-generation air interface using scalable OFDM numerologies, LDPC and polar coding, massive MIMO, and mmWave bands for high-throughput, low-latency service.
keywords: 5G NR, New Radio, 3GPP, OFDM numerology, flexible subcarrier spacing, LDPC code, polar code, massive MIMO, beamforming, mmWave, FR1, FR2, sub-6, standalone, non-standalone
aka: [5G NR, New Radio, 5G]
autolink: true
infobox:
  - { label: Type, value: 5G cellular air interface }
  - { label: Standards body, value: "3GPP (Release 15, 2018)" }
  - { label: Introduced, value: "2019 (commercial)" }
  - { label: Access, value: "OFDMA (CP-OFDM), scalable numerology" }
  - { label: Bands, value: "FR1 sub-6 GHz, FR2 mmWave 24–52 GHz" }
  - { label: Coding, value: "LDPC (data), polar (control)" }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [ofdm, ldpc-code, polar-code, mimo, 3gpp, lte, beamforming, base-station-enodeb-gnodeb]
cite_urls:
  - https://en.wikipedia.org/wiki/5G_NR
  - https://www.3gpp.org/technologies/5g-system-overview
---

**5G NR (New Radio)** is the [3GPP](/reference/3gpp/) fifth-generation radio-access
technology, built on a flexible [OFDM](/reference/ofdm/) framework with **scalable
numerologies**, [LDPC](/reference/ldpc-code/) coding for data and
[polar coding](/reference/polar-code/) for control, and heavy reliance on massive
[MIMO](/reference/mimo/) and [beamforming](/reference/beamforming/).[^wiki] It spans
two frequency ranges — sub-6 GHz (FR1) and millimetre-wave (FR2) — to serve
high-throughput, low-latency, and massive-device use cases from a single framework.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="5G NR scalable numerology: three subcarrier-spacing options of 15, 30, and 60 kilohertz produce progressively shorter OFDM symbol slots, trading frequency width for shorter latency, feeding an LDPC-coded data path and a beamformed massive-MIMO antenna." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="nrar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <text x="120" y="18" text-anchor="middle" font-size="9" fill="currentColor">scalable numerology (µ)</text>
  <g stroke="currentColor" stroke-width="0.8" font-size="8" fill="currentColor">
    <rect x="30" y="28" width="180" height="20" fill="currentColor" fill-opacity="0.16"/><text x="120" y="42" text-anchor="middle">15 kHz · 1 ms slot</text>
    <rect x="30" y="54" width="90" height="20" fill="currentColor" fill-opacity="0.22"/><text x="75" y="68" text-anchor="middle">30 kHz</text>
    <rect x="30" y="80" width="45" height="20" fill="currentColor" fill-opacity="0.3"/><text x="52" y="94" text-anchor="middle">60</text>
  </g>
  <text x="120" y="120" text-anchor="middle" font-size="8" fill="currentColor">wider spacing → shorter slot → lower latency</text>
  <rect x="250" y="40" width="60" height="26" fill="none" stroke="currentColor"/><text x="280" y="57" text-anchor="middle" font-size="8" fill="currentColor">LDPC</text>
  <line x1="310" y1="53" x2="345" y2="53" stroke="currentColor" marker-end="url(#nrar)"/>
  <g stroke="currentColor"><line x1="360" y1="30" x2="360" y2="90"/><line x1="360" y1="40" x2="415" y2="25"/><line x1="360" y1="55" x2="420" y2="55"/><line x1="360" y1="70" x2="415" y2="85"/></g>
  <text x="385" y="108" text-anchor="middle" font-size="8" fill="currentColor">massive MIMO beam</text>
</svg>
<figcaption>5G NR scales OFDM subcarrier spacing (15/30/60 kHz and higher) to trade bandwidth for latency, pairs LDPC-coded data with beamformed massive-MIMO arrays.</figcaption>
</figure>

## Overview

5G NR keeps OFDM but makes its parameters configurable: subcarrier spacing can be 15,
30, 60, 120, or 240 kHz, letting an operator pick short slots for low latency at high
frequencies or narrow spacing for coverage at low ones. It runs in **non-standalone**
mode anchored to an [LTE](/reference/lte/) core, or **standalone** with a native 5G
core. FR2 mmWave carriers offer huge bandwidth but short range, so they lean hard on
[beamforming](/reference/beamforming/) to steer energy toward each user.

## Technical characteristics

| Property | Value |
|----------|-------|
| Waveform | CP-OFDM (downlink and uplink; optional DFT-s-OFDM uplink) |
| Numerology | Subcarrier spacing 15–240 kHz (µ = 0–4) |
| Frequency ranges | FR1: 410 MHz–7.125 GHz; FR2: 24.25–52.6 GHz |
| Channel bandwidth | Up to 100 MHz (FR1), 400 MHz (FR2) |
| Data coding | [LDPC](/reference/ldpc-code/) |
| Control coding | [Polar code](/reference/polar-code/) |
| Modulation | QPSK to 256-QAM |
| Multi-antenna | Massive [MIMO](/reference/mimo/), analog/digital beamforming |

Choosing LDPC for high-throughput data (efficient at long block lengths) and polar
codes for short, reliable control messages was a defining decision of NR's Release 15
design.

## History

3GPP finalised the first NR specification in **Release 15 (2018)**, with commercial
launches beginning in 2019.[^3gpp] Later releases added reduced-capability (RedCap)
devices, non-terrestrial (satellite) access, and sidelink, broadening NR beyond
handset broadband.

## Deployment

5G NR rolled out worldwide from 2019 onward, most commonly as non-standalone sub-6 GHz
service overlaid on existing LTE, with mmWave used for dense hotspots and fixed
wireless access. It coexists with and aggregates alongside [LTE](/reference/lte/) at
most operators.

## Decoding it with GopherTrunk

**GopherTrunk does not decode 5G NR.** NR is a licensed, wideband, beamformed cellular
air interface whose bandwidth, adaptive numerology, and encryption place it entirely
outside the scope of a narrowband land-mobile trunking scanner. It appears in this
guide to show where OFDM, LDPC, and polar coding meet in a real modern system, and why
those algorithms matter to the broader RF world even though GopherTrunk's decode chain
targets single-carrier C4FM/QPSK trunking instead.

## Sources

[^wiki]: [5G NR](https://en.wikipedia.org/wiki/5G_NR) — Wikipedia, for the scalable OFDM numerology, LDPC/polar coding split, and FR1/FR2 band structure.
[^3gpp]: [5G system overview](https://www.3gpp.org/technologies/5g-system-overview) — 3GPP, for the Release 15 timeline and standalone/non-standalone architecture.
