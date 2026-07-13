---
slug: vdl-mode-2
title: VDL Mode 2
entry_type: protocol
category: aviation-marine
description: "VDL Mode 2 is a VHF aircraft datalink using D8PSK at 31.5 kbps to carry ACARS-over-AVLC and ATN traffic, the higher-rate successor to plain VHF ACARS."
keywords: VDL Mode 2, VDL2, VHF Data Link Mode 2, D8PSK, 31500 bps, AVLC, ACARS over AVLC, ATN, CSMA, 136.975 MHz, aviation datalink
aka: [VDL Mode 2, VDL2, VDLM2]
autolink: true
infobox:
  - { label: Type, value: VHF aviation datalink }
  - { label: Standards body, value: "ICAO Annex 10 / ARINC 631" }
  - { label: Access, value: "CSMA (carrier-sense)" }
  - { label: Channel spacing, value: 25 kHz }
  - { label: Modulation, value: "D8PSK, 31.5 kbps" }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [8psk, acars, phase-shift-keying, differential-decoding, hfdl]
cite_urls:
  - https://en.wikipedia.org/wiki/VHF_Data_Link
  - https://en.wikipedia.org/wiki/Aircraft_Communications_Addressing_and_Reporting_System
---

**VDL Mode 2** (**VHF Data Link Mode 2**) is a digital aircraft datalink that carries
[ACARS](/reference/acars/) and ATN traffic over VHF using **differential 8-PSK
([D8PSK](/reference/8psk/))** at **31,500 bps** — roughly thirteen times the throughput
of plain 2400 bps VHF ACARS.[^wiki] It is the mainstream high-rate VHF bearer for
airline datalink today, moving the same operational and FANS messages far faster than
the legacy [MSK](/reference/minimum-shift-keying/) channel.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="An eight-point differential 8-PSK constellation on the left feeds a VHF link at 31.5 kilobits per second carrying ACARS-over-AVLC frames to a ground station." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="vdlar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <circle cx="80" cy="70" r="40" fill="none" stroke="currentColor" stroke-width="1" stroke-opacity="0.5"/>
  <g fill="currentColor">
    <circle cx="120" cy="70" r="2.5"/><circle cx="108" cy="98" r="2.5"/><circle cx="80" cy="110" r="2.5"/><circle cx="52" cy="98" r="2.5"/><circle cx="40" cy="70" r="2.5"/><circle cx="52" cy="42" r="2.5"/><circle cx="80" cy="30" r="2.5"/><circle cx="108" cy="42" r="2.5"/>
  </g>
  <text x="80" y="135" text-anchor="middle" font-size="8" fill="currentColor">D8PSK · 3 bits/symbol</text>
  <path d="M130 70 h90" stroke="currentColor" stroke-width="1.2" marker-end="url(#vdlar)"/>
  <text x="175" y="62" text-anchor="middle" font-size="7.5" fill="currentColor">31.5 kbps VHF</text>
  <rect x="225" y="52" width="90" height="34" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/>
  <text x="270" y="72" text-anchor="middle" font-size="8" fill="currentColor">AVLC frames</text>
  <path d="M315 69 h90" stroke="currentColor" stroke-width="1.2" marker-end="url(#vdlar)"/>
  <text x="360" y="61" text-anchor="middle" font-size="7.5" fill="currentColor">ground station</text>
  <text x="230" y="120" text-anchor="middle" font-size="8" fill="currentColor">CSMA access · 25 kHz channels · 136.975 MHz CSC</text>
</svg>
<figcaption>VDL Mode 2 keys three bits per symbol with differential 8-PSK at 31.5 kbps, framing data in AVLC over carrier-sense VHF channels.</figcaption>
</figure>

## Overview

VDL Mode 2 keeps ACARS's message applications but replaces the physical and link layers.
The physical layer is [differentially encoded](/reference/differential-decoding/) 8-PSK
at 10,500 symbols/s (31.5 kbps); the link layer is AVLC (Aviation VHF Link Control), an
HDLC-derived framing with addressing and acknowledgements. Media access is CSMA —
stations listen before transmitting and back off on a busy channel, sharing each 25 kHz
channel among many aircraft and ground stations.

## Technical characteristics

| Property | Value |
|----------|-------|
| Band | VHF aeronautical (Common Signalling Channel 136.975 MHz) |
| Channel spacing | 25 kHz |
| Modulation | Differential 8-PSK (D8PSK) |
| Symbol rate | 10,500 symbols/s |
| Bit rate | 31,500 bps |
| Link layer | AVLC (HDLC-derived) |
| Media access | CSMA (carrier-sense multiple access) |
| Payloads | ACARS-over-AVLC (AOA) and ATN/OSI |

The move to 8-PSK packs three bits into every symbol, and differential encoding removes
the need to resolve absolute carrier phase — the receiver only tracks phase *changes*,
which simplifies demodulation on a fading VHF channel. AVLC's acknowledgements make the
link reliable, unlike the fire-and-forget style of some legacy ACARS blocks.

## History

VDL Mode 2 was standardised by ICAO in Annex 10 and the AEEC in ARINC 631 to relieve
saturation of the 2400 bps ACARS channels as datalink use grew. Deployment accelerated
through the 2000s and 2010s as datalink service providers rolled out ground networks,
first for ACARS-over-AVLC and later for ATN Baseline traffic.

## Deployment

VDL Mode 2 is now the primary VHF datalink for most airline aircraft, operated by ARINC
and SITA ground networks worldwide, with 136.975 MHz reserved as the common signalling
channel. It coexists with legacy VHF [ACARS](/reference/acars/) and with the HF and
satcom bearers such as [HFDL](/reference/hfdl/) used over oceans.

## Decoding it with GopherTrunk

**Not decoded.** VDL Mode 2 is an aviation VHF datalink outside GopherTrunk's
land-mobile trunking and 1090 MHz [ADS-B](/reference/ads-b/) focus. Its D8PSK signal is
receivable with a modest SDR and open decoders, but GopherTrunk does not implement the
D8PSK/AVLC chain. This page frames it honestly alongside its predecessor
[ACARS](/reference/acars/) and the underlying modulation [8-PSK](/reference/8psk/).

## Sources

[^wiki]: [VHF Data Link](https://en.wikipedia.org/wiki/VHF_Data_Link) — Wikipedia, for VDL Mode 2's differential 8-PSK physical layer, 31.5 kbps rate, AVLC link layer, CSMA access, the 136.975 MHz common signalling channel, and its ACARS/ATN payloads.
