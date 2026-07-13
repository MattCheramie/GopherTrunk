---
slug: dvb-s
title: DVB-S / DVB-S2
entry_type: protocol
category: broadcast
description: "DVB-S and DVB-S2 are the European standards for digital satellite television, carrying MPEG/HEVC video over QPSK or 8PSK with concatenated FEC and, in S2, LDPC coding."
keywords: DVB-S, DVB-S2, DVB-S2X, digital satellite television, QPSK, 8PSK, LDPC, Reed-Solomon, ETSI, EN 300 421, EN 302 307, MPEG transport stream, Ku band
aka: [DVB-S, DVB-S2, DVB-S2X]
autolink: true
infobox:
  - { label: Type, value: Digital satellite television }
  - { label: Standards body, value: "ETSI (DVB Project)" }
  - { label: Introduced, value: "1994 (S), 2005 (S2)" }
  - { label: Access, value: Broadcast (one-to-many) }
  - { label: Channel spacing, value: "typically 27–36 MHz transponder" }
  - { label: Modulation, value: "QPSK, 8PSK (S2 adds 16/32-APSK)" }
  - { label: GopherTrunk support, value: Not decoded (out of scope) }
see_also: [qpsk, 8psk, ldpc-code, reed-solomon-code, dvb-t, dvb-c]
cite_urls:
  - https://en.wikipedia.org/wiki/DVB-S2
  - https://www.etsi.org/deliver/etsi_en/302300_302399/302307/
---

**DVB-S** (Digital Video Broadcasting — Satellite) and its successor **DVB-S2** are
the [ETSI](/reference/etsi/) standards for digital television delivered from
geostationary satellites.[^wiki] Because a satellite transponder is power-limited but
bandwidth-rich, DVB-S uses constant-envelope phase modulation —
[QPSK](/reference/qpsk/), or [8PSK](/reference/8psk/) and higher APSK orders in
DVB-S2 — wrapped in strong forward error correction so the link closes with a small
dish under rain fade.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A QPSK constellation of four points on a circle feeding a satellite downlink to a dish, illustrating DVB-S phase modulation." xmlns="http://www.w3.org/2000/svg">
  <circle cx="90" cy="85" r="45" fill="none" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="35" y1="85" x2="145" y2="85" stroke="currentColor" stroke-opacity="0.35"/>
  <line x1="90" y1="30" x2="90" y2="140" stroke="currentColor" stroke-opacity="0.35"/>
  <g fill="currentColor"><circle cx="122" cy="53" r="4"/><circle cx="58" cy="53" r="4"/><circle cx="58" cy="117" r="4"/><circle cx="122" cy="117" r="4"/></g>
  <text x="90" y="160" text-anchor="middle" font-size="9" fill="currentColor">QPSK — 4 phases, 2 bits/symbol</text>
  <path d="M175 85 L250 85" stroke="currentColor" stroke-opacity="0.7" marker-end="url(#dvbsar)"/>
  <text x="212" y="78" text-anchor="middle" font-size="8" fill="currentColor">downlink</text>
  <circle cx="320" cy="55" r="12" fill="none" stroke="currentColor"/><text x="320" y="40" text-anchor="middle" font-size="8" fill="currentColor">satellite</text>
  <path d="M320 67 L360 120" stroke="currentColor" stroke-dasharray="3 3" stroke-opacity="0.6"/>
  <path d="M345 135 A28 28 0 0 1 385 120 L365 108 A20 20 0 0 0 350 128 Z" fill="currentColor" fill-opacity="0.25" stroke="currentColor"/>
  <text x="372" y="150" text-anchor="middle" font-size="8" fill="currentColor">dish</text>
  <defs><marker id="dvbsar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>DVB-S carries data as satellite-friendly phase modulation; DVB-S2 adds LDPC coding to squeeze near-Shannon capacity from each transponder.</figcaption>
</figure>

## Overview

A DVB-S signal occupies a whole satellite transponder — commonly 27 to 36 MHz — and
runs at a symbol rate of tens of millions of symbols per second. The original DVB-S
used only QPSK with a concatenated code: an outer
[Reed–Solomon](/reference/reed-solomon-code/) block code protecting an inner
punctured convolutional code. DVB-S2 replaced that with a **BCH-plus-LDPC** stack and
added 8PSK, 16-APSK, and 32-APSK, together with adaptive coding and modulation, so
the transmitter can pick the most efficient constellation the current link margin
allows.

## Technical characteristics

| Property | Value |
|----------|-------|
| Modulation | QPSK (S); +8PSK/16-APSK/32-APSK (S2) |
| Inner FEC | Convolutional (S); [LDPC](/reference/ldpc-code/) (S2) |
| Outer FEC | [Reed–Solomon](/reference/reed-solomon-code/) (S); BCH (S2) |
| Symbol rate | ≈ 1–45 Msym/s (transponder-dependent) |
| Roll-off | 35% (S); 35/25/20% (S2); down to 5% (S2X) |
| Payload | MPEG transport stream / generic stream (S2) |
| Efficiency gain | S2 ≈ 30% more capacity than S |

## History

The DVB Project standardised DVB-S as ETSI EN 300 421 in 1994; it became the
workhorse of satellite TV worldwide. DVB-S2 followed in 2005 as EN 302 307, one of
the first mass-market systems to adopt LDPC codes, bringing performance within a
fraction of a decibel of the [Shannon limit](/reference/shannon-capacity/).[^etsi]
The DVB-S2X extension (2014) added finer code rates, lower roll-off, and higher-order
constellations for professional and high-throughput links.

## Deployment

DVB-S/S2 dominates direct-to-home satellite platforms across Europe, Asia, Africa,
the Middle East, and Latin America, and underpins satellite news gathering and
broadcast contribution feeds. Ku-band and Ka-band downlinks are received with dishes
from 45 cm upward, the size set by the frequency, transponder power, and required
rain-fade margin.

## Decoding it with GopherTrunk

GopherTrunk does **not** decode DVB-S; it targets terrestrial land-mobile trunking,
not MPEG satellite video. The downlink is nonetheless a favourite of SDR hobbyists:
after a low-noise block downconverter shifts the Ku-band signal to an L-band IF, a
wideband receiver plus dedicated software can lock a transponder and even display its
QPSK/8PSK constellation. The terrestrial and cable siblings are covered under
[DVB-T](/reference/dvb-t/) and [DVB-C](/reference/dvb-c/).

## Sources

[^wiki]: [DVB-S2](https://en.wikipedia.org/wiki/DVB-S2) — Wikipedia, for the satellite DVB standards, their phase-modulation constellations, and the move to LDPC coding.
[^etsi]: [EN 302 307 (DVB-S2)](https://www.etsi.org/deliver/etsi_en/302300_302399/302307/) — ETSI, the primary standard defining DVB-S2 channel coding, LDPC/BCH FEC, and modulation.
