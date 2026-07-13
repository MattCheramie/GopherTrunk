---
slug: packet-radio
title: Packet radio
entry_type: protocol
category: paging-data
description: Packet radio is amateur digital data over radio using the AX.25 protocol, typically AFSK at 1200 bps or direct GFSK at 9600 bps for messaging, APRS, and BBS access.
keywords: packet radio, AX.25, AFSK, GFSK, 1200 baud, 9600 baud, TNC, amateur radio, APRS, Bell 202, BBS, digipeater
aka: ["packet radio", "AX.25 packet"]
autolink: true
infobox:
  - { label: Type, value: Amateur digital data protocol }
  - { label: Standards body, value: "TAPR / ARRL (AX.25)" }
  - { label: Introduced, value: "1980s" }
  - { label: Access, value: "CSMA (shared channel)" }
  - { label: Channel spacing, value: 12.5 / 25 kHz }
  - { label: Modulation, value: "AFSK 1200 bps / GFSK 9600 bps" }
  - { label: Link layer, value: AX.25 }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [ax25, afsk, kiss-tnc, direwolf, gfsk, aprs]
cite_urls:
  - https://en.wikipedia.org/wiki/Packet_radio
  - https://en.wikipedia.org/wiki/AX.25
---

**Packet radio** is the amateur-radio method of sending digital data over the air using
the **[AX.25](/reference/ax25/)** link-layer protocol. Most commonly it uses
**[AFSK](/reference/afsk/)** at 1200 bps (Bell 202 tones through an FM radio) or direct
**[GFSK](/reference/gfsk/)** at 9600 bps, framing data into addressed, error-checked
packets that digipeaters can relay across a network.[^wiki][^ax25]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="An AX.25 packet frame: flag, source and destination callsigns, control and PID fields, information payload, frame-check sequence, and closing flag." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1" font-size="7.5" fill="currentColor" text-anchor="middle">
    <rect x="20" y="45" width="34" height="30" fill="currentColor" fill-opacity="0.22"/><text x="37" y="63">flag</text>
    <rect x="54" y="45" width="60" height="30" fill="none"/><text x="84" y="63">dest</text>
    <rect x="114" y="45" width="60" height="30" fill="none"/><text x="144" y="63">source</text>
    <rect x="174" y="45" width="40" height="30" fill="currentColor" fill-opacity="0.12"/><text x="194" y="63">ctrl</text>
    <rect x="214" y="45" width="34" height="30" fill="currentColor" fill-opacity="0.12"/><text x="231" y="63">PID</text>
    <rect x="248" y="45" width="110" height="30" fill="none"/><text x="303" y="63">information</text>
    <rect x="358" y="45" width="44" height="30" fill="currentColor" fill-opacity="0.12"/><text x="380" y="63">FCS</text>
    <rect x="402" y="45" width="34" height="30" fill="currentColor" fill-opacity="0.22"/><text x="419" y="63">flag</text>
  </g>
  <text x="228" y="98" text-anchor="middle" font-size="8.5" fill="currentColor">AX.25 frame · addressed, CRC-protected · AFSK 1200 or GFSK 9600 bps</text>
</svg>
<figcaption>Packet radio wraps data in AX.25 frames carrying callsign addresses and a frame-check sequence between HDLC flags.</figcaption>
</figure>

## Overview

Packet radio borrows the HDLC framing of X.25 and adapts it, as AX.25, to carry amateur
callsigns as source and destination addresses. A [terminal node
controller](/reference/kiss-tnc/) (TNC) handles the modem and framing, presenting the
computer with clean packets. Channel access is unslotted CSMA: stations listen before
transmitting and back off on collision. Digipeaters extend range by decoding a packet and
re-transmitting it toward its destination.

## Technical characteristics

| Property | Value |
|----------|-------|
| Link layer | AX.25 (HDLC-derived) |
| Modulation (VHF) | AFSK 1200 bps, Bell 202 tones |
| Modulation (higher rate) | GFSK 9600 bps (G3RUH) |
| Error check | 16-bit CRC frame-check sequence |
| Access | CSMA, carrier-sense with back-off |
| Addressing | Amateur callsigns + SSID |

The 1200 bps AFSK mode passes audio tones through an ordinary FM transceiver, so any
voice radio can send packet. The 9600 bps mode instead feeds the modulator directly,
needing a radio with a flat baseband ("9600 ready") port.

## History

Amateur packet radio took off in the 1980s after TAPR produced affordable TNC designs and
the AX.25 specification was standardised. It powered bulletin-board systems, keyboard-to-
keyboard chat, and store-and-forward mail networks before the internet, and later became
the transport for [APRS](/reference/aprs/) position and telemetry reporting.

## Deployment

Packet radio remains active on VHF/UHF amateur bands, most visibly as the AX.25 layer
under APRS on 144.390 MHz (North America) and regional equivalents, plus Winlink email
gateways and emergency-communications nets.

## Decoding it with GopherTrunk

Packet radio is **not decoded** by GopherTrunk, whose scope is land-mobile trunking and a
few paging codes. AX.25 packet is well served by dedicated software such as
[Direwolf](/reference/direwolf/), a software TNC that demodulates AFSK/GFSK and emits
[KISS](/reference/kiss-tnc/) frames. It is documented here to place the amateur data mode
in the broader signal landscape.

## Sources

[^wiki]: [Packet radio](https://en.wikipedia.org/wiki/Packet_radio) — Wikipedia, for the amateur packet-radio system, its AFSK 1200 / GFSK 9600 modes, and the TNC/digipeater architecture.
[^ax25]: [AX.25](https://en.wikipedia.org/wiki/AX.25) — Wikipedia, for the AX.25 link-layer framing, callsign addressing, and frame-check sequence.
