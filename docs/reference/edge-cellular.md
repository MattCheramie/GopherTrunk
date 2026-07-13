---
slug: edge-cellular
title: EDGE (2.75G)
entry_type: protocol
category: cellular
description: "EDGE is the 2.75G upgrade to GSM/GPRS that adds 8PSK modulation to roughly triple packet-data throughput on existing carriers."
keywords: EDGE, Enhanced Data rates for GSM Evolution, EGPRS, 2.75G, 8PSK, GSM, GPRS, 3GPP, cellular data
aka: [EDGE, EGPRS, Enhanced Data rates for GSM Evolution, 2.75G]
autolink: true
infobox:
  - { label: Type, value: "Cellular packet data (2.75G)" }
  - { label: Standards body, value: 3GPP }
  - { label: Introduced, value: "2003" }
  - { label: Access, value: "Shared TDMA slots (packet)" }
  - { label: Channel spacing, value: 200 kHz (on GSM carriers) }
  - { label: Modulation, value: "GMSK and 8PSK (adaptive)" }
  - { label: GopherTrunk support, value: Not decoded (out of scope) }
see_also: [gsm, gprs, 8psk, tdma, 3gpp]
cite_urls:
  - https://en.wikipedia.org/wiki/Enhanced_Data_rates_for_GSM_Evolution
  - https://www.3gpp.org/technologies/geran
---

**EDGE** (Enhanced Data rates for GSM Evolution), also called EGPRS, is the 2.75G
enhancement of [GSM](/reference/gsm/) and [GPRS](/reference/gprs/). By adding
[8PSK](/reference/8psk/) modulation alongside the original GMSK, EDGE packs three bits
into each symbol instead of one, roughly tripling per-slot throughput on the very same
200 kHz carriers.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="An 8PSK constellation of eight equally spaced points on a circle, contrasted with GSM's simpler binary GMSK, showing that EDGE carries three bits per symbol." xmlns="http://www.w3.org/2000/svg">
  <circle cx="150" cy="80" r="52" fill="none" stroke="currentColor" stroke-opacity="0.35"/>
  <g fill="currentColor">
    <circle cx="202" cy="80" r="3.5"/>
    <circle cx="187" cy="43" r="3.5"/>
    <circle cx="150" cy="28" r="3.5"/>
    <circle cx="113" cy="43" r="3.5"/>
    <circle cx="98" cy="80" r="3.5"/>
    <circle cx="113" cy="117" r="3.5"/>
    <circle cx="150" cy="132" r="3.5"/>
    <circle cx="187" cy="117" r="3.5"/>
  </g>
  <line x1="90" y1="80" x2="210" y2="80" stroke="currentColor" stroke-opacity="0.3"/>
  <line x1="150" y1="20" x2="150" y2="140" stroke="currentColor" stroke-opacity="0.3"/>
  <text x="150" y="158" text-anchor="middle" font-size="9" fill="currentColor">8PSK: 8 phases = 3 bits / symbol</text>
  <g stroke="currentColor" stroke-width="1.2" fill="none"><path d="M300 60 h110" marker-end="url(#edgear)"/></g>
  <text x="355" y="52" text-anchor="middle" font-size="9" fill="currentColor">≈3× the GPRS bit rate</text>
  <text x="355" y="90" text-anchor="middle" font-size="8" fill="currentColor">same 200 kHz carrier,</text>
  <text x="355" y="102" text-anchor="middle" font-size="8" fill="currentColor">same TDMA slot timing</text>
  <defs><marker id="edgear" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>EDGE swaps GSM's binary GMSK for 8PSK, carrying three bits per symbol to lift data rates without new spectrum.</figcaption>
</figure>

## Overview

EDGE keeps the [GSM](/reference/gsm/) frame structure, 200 kHz carriers, and
[TDMA](/reference/tdma/) slot timing untouched — it changes only how bits are carried
inside each burst. When the radio link is clean, the transmitter uses
[8PSK](/reference/8psk/) to send three bits per symbol; when conditions worsen it falls
back toward robust GMSK. Nine modulation-and-coding schemes (MCS-1 to MCS-9) let the
network adapt continuously, and incremental redundancy re-uses failed transmissions
instead of discarding them.

## Technical characteristics

| Property | Value |
|----------|-------|
| Generation | 2.75G |
| Modulation | GMSK and 8PSK, link-adaptive |
| Coding schemes | MCS-1 to MCS-9 |
| Peak per slot | ≈59.2 kbit/s (MCS-9) |
| Peak multislot | ≈236–473 kbit/s (Evolved EDGE) |
| Symbol rate | 270.833 ksym/s (as GSM) |
| Retransmission | Incremental redundancy (hybrid ARQ) |

Because the symbol rate and channel plan are unchanged, an operator upgrades to EDGE
mostly through base-station software and modem support in handsets.

## History

EDGE was standardised by [3GPP](/reference/3gpp/) and reached commercial networks
around 2003, filling the gap before 3G coverage was widespread. A later Evolved EDGE
release added dual-carrier operation, higher-order coding, and downlink diversity to
push peak rates further, though few networks fully deployed it.

## Deployment

EDGE gave carriers a cheap route to "3G-like" data on 2G infrastructure and became a
near-universal fallback in GSM networks; many rural and roaming areas relied on it for
years. It shares spectrum and core nodes with [GPRS](/reference/gprs/), so the two
coexist on the same cells. As operators refarm 2G spectrum for LTE and 5G, EDGE is
being retired alongside GSM.

## Decoding it with GopherTrunk

GopherTrunk decodes land-mobile and utility signals; **cellular data such as EDGE is
out of scope and is not decoded.** Its traffic is private, authenticated, and usually
ciphered on operator-licensed carriers. EDGE is documented here to complete the
[GSM](/reference/gsm/) evolution story and to illustrate an adaptive
[8PSK](/reference/8psk/) air interface.

## Sources

[^wiki]: [Enhanced Data rates for GSM Evolution](https://en.wikipedia.org/wiki/Enhanced_Data_rates_for_GSM_Evolution) — Wikipedia, for the 2.75G EDGE/EGPRS enhancement, its adaptive 8PSK modulation, and the MCS coding schemes.
