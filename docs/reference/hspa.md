---
slug: hspa
title: HSPA (3.5G)
entry_type: protocol
category: cellular
description: "HSPA is the 3.5G upgrade to UMTS/W-CDMA that adds fast shared channels, higher-order QAM, and hybrid ARQ to boost mobile broadband speeds."
keywords: HSPA, HSDPA, HSUPA, HSPA+, 3.5G, UMTS, W-CDMA, 3GPP, 16-QAM, 64-QAM, MIMO, mobile broadband, cellular
aka: [HSPA, HSDPA, HSUPA, HSPA+, High Speed Packet Access, 3.5G]
autolink: true
infobox:
  - { label: Type, value: "Cellular mobile broadband (3.5G)" }
  - { label: Standards body, value: 3GPP }
  - { label: Introduced, value: "2005" }
  - { label: Access, value: "CDMA (on W-CDMA carriers)" }
  - { label: Channel spacing, value: 5 MHz }
  - { label: Modulation, value: "QPSK, 16-QAM, 64-QAM" }
  - { label: GopherTrunk support, value: Not decoded (out of scope) }
see_also: [umts-wcdma, cdma, 3gpp, quadrature-amplitude-modulation, mimo]
cite_urls:
  - https://en.wikipedia.org/wiki/High_Speed_Packet_Access
  - https://www.3gpp.org/technologies/hspa
---

**HSPA** (High Speed Packet Access) is the 3.5G family of enhancements to
[UMTS/W-CDMA](/reference/umts-wcdma/), combining HSDPA on the downlink and HSUPA on the
uplink. It layers fast, shared, scheduled channels over the existing 5 MHz
[CDMA](/reference/cdma/) carrier and adds higher-order
[QAM](/reference/quadrature-amplitude-modulation/) plus hybrid ARQ to deliver true
mobile-broadband speeds.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A base station scheduler dividing a shared high-speed downlink channel among users in short 2 ms intervals according to their radio conditions." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="40" width="70" height="46" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1"/>
  <text x="65" y="66" text-anchor="middle" font-size="8.5" fill="currentColor">scheduler</text>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <rect x="150" y="24" width="34" height="24"/>
    <rect x="184" y="24" width="34" height="24" fill="currentColor" fill-opacity="0.25"/>
    <rect x="218" y="24" width="34" height="24"/>
    <rect x="252" y="24" width="34" height="24" fill="currentColor" fill-opacity="0.25"/>
    <rect x="286" y="24" width="34" height="24"/>
    <rect x="320" y="24" width="34" height="24" fill="currentColor" fill-opacity="0.25"/>
  </g>
  <path d="M100 63 L146 44" stroke="currentColor" stroke-width="1.1" fill="none" marker-end="url(#hspaar)"/>
  <text x="252" y="18" text-anchor="middle" font-size="8" fill="currentColor">2 ms transmission-time intervals → best-placed user served each slot</text>
  <text x="252" y="70" text-anchor="middle" font-size="8.5" fill="currentColor">adaptive QPSK / 16-QAM / 64-QAM + hybrid ARQ retransmission</text>
  <text x="252" y="112" text-anchor="middle" font-size="9" fill="currentColor">shared high-speed channel over one 5 MHz W-CDMA carrier</text>
  <defs><marker id="hspaar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>HSPA adds a fast shared channel with a base-station scheduler that serves the best-placed user in each 2 ms interval, adapting modulation to radio conditions.</figcaption>
</figure>

## Overview

Plain [UMTS](/reference/umts-wcdma/) gave every bearer a fixed dedicated channel, which
was inefficient for the bursty traffic of web and app use. HSPA introduces a high-speed
shared channel that a fast scheduler in the base station reallocates every 2 ms to
whichever user currently has the best signal. It couples this with adaptive modulation
(up to 64-[QAM](/reference/quadrature-amplitude-modulation/)), hybrid ARQ that combines
retransmissions rather than discarding them, and later [MIMO](/reference/mimo/) — all
on the same 5 MHz W-CDMA carrier.

## Technical characteristics

| Property | Value |
|----------|-------|
| Generation | 3.5G |
| Components | HSDPA (down), HSUPA (up), HSPA+ (evolved) |
| Carrier spacing | 5 MHz (as W-CDMA) |
| Modulation | QPSK, 16-QAM, 64-QAM (down); QPSK, 16-QAM (up) |
| Scheduling | 2 ms TTI, fast link adaptation |
| Retransmission | Hybrid ARQ with soft combining |
| Peak (HSPA+) | Tens of Mbit/s with MIMO / dual-carrier |

Key rate gains come from moving scheduling and retransmission control from the core
network down into the base station, cutting latency and reacting to fast fading.

## History

[3GPP](/reference/3gpp/) introduced HSDPA in Release 5 (2002 spec, networks from about
2005) and HSUPA in Release 6, together branded HSPA. Later releases added HSPA+ (also
called Evolved HSPA), bringing 64-QAM, [MIMO](/reference/mimo/), and multi-carrier
operation that kept 3G competitive as [LTE](/reference/lte/) rolled out.

## Deployment

HSPA became the workhorse of 3G mobile broadband worldwide, delivered through USB
modems and smartphones alike, and HSPA+ let operators advertise "4G-like" speeds on
upgraded 3G networks. It runs on the same [UMTS](/reference/umts-wcdma/) carriers and
core, so activation was largely a software and scheduler upgrade. HSPA is now being
retired as carriers shut down 3G in favour of LTE and 5G.

## Decoding it with GopherTrunk

GopherTrunk scans trunked land-mobile and utility signals; **cellular broadband such as
HSPA is out of scope and is not decoded.** It carries private, authenticated, ciphered
subscriber data on licensed [W-CDMA](/reference/umts-wcdma/) spectrum. HSPA is included
for reference as the high-speed evolution of 3G.

## Sources

[^wiki]: [High Speed Packet Access](https://en.wikipedia.org/wiki/High_Speed_Packet_Access) — Wikipedia, for the 3.5G HSPA family, its HSDPA/HSUPA components, fast shared-channel scheduling, and higher-order QAM.
