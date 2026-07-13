---
slug: lte
title: LTE (Long-Term Evolution)
entry_type: protocol
category: cellular
description: LTE is the 3GPP 4G mobile-broadband standard using OFDMA downlink and SC-FDMA uplink over scalable resource blocks, with turbo-coded transport and an all-IP packet core.
keywords: LTE, Long-Term Evolution, 4G, 3GPP, OFDMA, SC-FDMA, resource block, turbo code, eNodeB, EPC, VoLTE, carrier aggregation, E-UTRA
aka: [LTE, Long-Term Evolution, 4G LTE, E-UTRA]
autolink: true
infobox:
  - { label: Type, value: 4G cellular broadband }
  - { label: Standards body, value: "3GPP (Release 8, 2008)" }
  - { label: Introduced, value: "2009 (commercial)" }
  - { label: Access, value: "OFDMA down / SC-FDMA up" }
  - { label: Channel spacing, value: "1.4–20 MHz (scalable)" }
  - { label: Modulation, value: "QPSK, 16/64/256-QAM" }
  - { label: Coding, value: Turbo code (data), convolutional (control) }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [ofdm, ofdma, turbo-code, 3gpp, lte-advanced, 5g-nr, mimo, volte, base-station-enodeb-gnodeb]
cite_urls:
  - https://en.wikipedia.org/wiki/LTE_(telecommunication)
  - https://www.3gpp.org/technologies/keywords-acronyms/98-lte
---

**LTE (Long-Term Evolution)** is the [3GPP](/reference/3gpp/) fourth-generation
mobile-broadband standard, built on [OFDMA](/reference/ofdma/) in the downlink and
[SC-FDMA](/reference/ofdma/) in the uplink, carrying user data over a grid of
**resource blocks** protected by [turbo coding](/reference/turbo-code/).[^wiki] It
replaced the circuit-switched core of earlier cellular generations with an all-IP
packet architecture, delivering the high-throughput mobile internet that smartphones
depend on.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="An LTE time-frequency resource grid: the vertical axis is frequency divided into subcarriers grouped into resource blocks, the horizontal axis is time divided into slots and symbols, with individual resource elements shaded." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="ltear" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="45" y1="150" x2="45" y2="20" stroke="currentColor" stroke-opacity="0.6" marker-end="url(#ltear)"/>
  <line x1="45" y1="150" x2="430" y2="150" stroke="currentColor" stroke-opacity="0.6" marker-end="url(#ltear)"/>
  <text x="18" y="88" text-anchor="middle" font-size="9" fill="currentColor" transform="rotate(-90 18 88)">frequency →</text>
  <text x="235" y="170" text-anchor="middle" font-size="9" fill="currentColor">time (slots / OFDM symbols) →</text>
  <g stroke="currentColor" stroke-width="0.7" stroke-opacity="0.5">
    <rect x="55" y="30" width="84" height="110" fill="currentColor" fill-opacity="0.14"/>
    <rect x="139" y="30" width="84" height="110" fill="none"/>
    <rect x="223" y="30" width="84" height="110" fill="currentColor" fill-opacity="0.14"/>
    <rect x="307" y="30" width="84" height="110" fill="none"/>
  </g>
  <g stroke="currentColor" stroke-width="0.4" stroke-opacity="0.35">
    <line x1="55" y1="52" x2="391" y2="52"/><line x1="55" y1="74" x2="391" y2="74"/><line x1="55" y1="96" x2="391" y2="96"/><line x1="55" y1="118" x2="391" y2="118"/>
    <line x1="83" y1="30" x2="83" y2="140"/><line x1="111" y1="30" x2="111" y2="140"/>
  </g>
  <rect x="55" y="52" width="28" height="22" fill="currentColor" fill-opacity="0.5"/>
  <text x="97" y="24" text-anchor="middle" font-size="8" fill="currentColor">1 resource block = 12 subcarriers × 1 slot</text>
</svg>
<figcaption>LTE maps user data onto a time-frequency grid; the smallest schedulable unit is a resource block of 12 subcarriers over one 0.5 ms slot.</figcaption>
</figure>

## Overview

LTE, standardised as **E-UTRA** (Evolved UMTS Terrestrial Radio Access), was designed
for low latency, high peak rates, and flat, IP-based signalling. The radio access
network consists of [base stations called eNodeBs](/reference/base-station-enodeb-gnodeb/)
connected to an Evolved Packet Core (EPC). Unlike the digital land-mobile systems
GopherTrunk targets, LTE is a wideband, scheduled, multi-user system: a central
scheduler assigns resource blocks to handsets millisecond by millisecond.

## Technical characteristics

| Property | Value |
|----------|-------|
| Downlink access | [OFDMA](/reference/ofdma/) |
| Uplink access | SC-FDMA (single-carrier FDMA, low PAPR) |
| Channel bandwidth | 1.4, 3, 5, 10, 15, 20 MHz |
| Subcarrier spacing | 15 kHz |
| Resource block | 12 subcarriers × 0.5 ms slot |
| Modulation | QPSK, 16-QAM, 64-QAM (256-QAM in later releases) |
| Channel coding | [Turbo code](/reference/turbo-code/) for data; tail-biting convolutional for control |
| Duplexing | FDD and TDD variants |
| Multi-antenna | [MIMO](/reference/mimo/) up to 4×4 in Release 8 |

The SC-FDMA uplink is essentially OFDMA with a DFT precoding stage, which lowers the
peak-to-average power ratio so handset power amplifiers run more efficiently.

## History

3GPP froze LTE in **Release 8 (2008)**, with the first commercial networks launching
in 2009–2010.[^3gpp] Marketed as "4G," early LTE did not initially meet the ITU
IMT-Advanced bar; that gap was closed by [LTE-Advanced](/reference/lte-advanced/) in
Release 10. Successive releases added carrier aggregation, higher-order MIMO, and the
machine-type variants that seeded the transition toward [5G NR](/reference/5g-nr/).

## Deployment

LTE became the dominant global mobile-broadband technology of the 2010s and remains
widely deployed as the coverage and voice layer beneath 5G. Voice initially fell back
to legacy circuit-switched networks until [VoLTE](/reference/volte/) carried calls as
packets over the LTE bearer itself.

## Decoding it with GopherTrunk

**GopherTrunk does not decode LTE.** LTE is a licensed, scheduled, wideband cellular
system whose air interface is out of scope for a land-mobile trunking scanner
([P25](/reference/p25-phase-1/), [DMR](/reference/dmr/), [NXDN](/reference/nxdn/),
[TETRA](/reference/tetra/)), and its user-plane traffic is encrypted end to end.
Recovering LTE requires a 20 MHz-capable front end plus full OFDMA channel estimation,
scheduling recovery, and turbo decoding — well beyond GopherTrunk's narrowband
single-carrier decode chain. It is documented here for context on the wider RF
landscape a scanner shares the spectrum with.

## Sources

[^wiki]: [LTE (telecommunication)](https://en.wikipedia.org/wiki/LTE_(telecommunication)) — Wikipedia, for the OFDMA/SC-FDMA air interface, resource-block structure, and all-IP architecture.
[^3gpp]: [LTE](https://www.3gpp.org/technologies/keywords-acronyms/98-lte) — 3GPP, for the Release 8 standardisation timeline and design goals.
