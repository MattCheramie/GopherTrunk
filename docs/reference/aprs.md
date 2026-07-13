---
slug: aprs
title: APRS
entry_type: protocol
category: paging-data
description: "APRS (Automatic Packet Reporting System) is an amateur-radio data network for real-time position, weather, telemetry, and messaging, carried over AX.25 packet using AFSK."
keywords: APRS, Automatic Packet Reporting System, amateur radio, 144.390, AX.25, AFSK, Bell 202, Bob Bruninga, position reporting, digipeater, igate, APRS-IS
aka: [APRS]
autolink: true
infobox:
  - { label: Type, value: Amateur data network }
  - { label: Created by, value: Bob Bruninga (WB4APR) }
  - { label: Band, value: 144.390 MHz (North America) }
  - { label: Link layer, value: AX.25 (UI frames) }
  - { label: Modulation, value: 1200 bps AFSK (Bell 202) }
  - { label: Content, value: Position, weather, telemetry, messages }
  - { label: GopherTrunk support, value: Decoded }
see_also: [ax25, afsk, packet-radio, direwolf, mueller-muller-timing-recovery, cyclic-redundancy-check]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/rf-sdr/other-signals/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Automatic_Packet_Reporting_System
  - http://www.aprs.org/doc/APRS101.PDF
external:
  - { title: "GopherTrunk APRS decoder", url: /aprs.html }
---

**APRS** (**Automatic Packet Reporting System**) is an amateur-radio data network for
real-time tactical information — **position, weather, telemetry, and messaging**. It
is carried as [AX.25](/reference/ax25/) packet frames using
[AFSK](/reference/afsk/), most commonly on 144.390 MHz in North America.[^wiki][^spec]
Unlike traditional point-to-point [packet radio](/reference/packet-radio/), APRS is a
**connectionless broadcast** system: stations transmit short beacons that any receiver
can hear, and a mesh of relays spreads them, producing a live shared map of a region.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="An APRS packet carrying a callsign and position payload over AX.25 on 144.39 MHz, ending in a frame-check sequence." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1"><rect x="30" y="40" width="90" height="28" fill="currentColor" fill-opacity="0.12"/><rect x="120" y="40" width="220" height="28" fill="currentColor" fill-opacity="0.22"/><rect x="340" y="40" width="90" height="28" fill="none"/></g>
  <g font-size="8.5" fill="currentColor" text-anchor="middle"><text x="75" y="58">callsign</text><text x="230" y="58">position / weather / message</text><text x="385" y="58">FCS</text></g>
  <text x="230" y="88" text-anchor="middle" font-size="8" fill="currentColor">AFSK over AX.25 · 144.39 MHz (NA)</text>
</svg>
<figcaption>APRS carries position and telemetry in AX.25 UI packets, usually via 1200-baud Bell 202 AFSK on 144.39 MHz.</figcaption>
</figure>

## Overview

An APRS station beacons its own state — typically a GPS position, but also weather,
telemetry, objects, and status text — in an [AX.25](/reference/ax25/) unnumbered
information (UI) frame. **Digipeaters** hear these frames and rebroadcast them,
following a path written into the frame (the well-known **WIDE1-1, WIDE2-1** aliases)
so a beacon propagates a controlled number of hops across a region without looping
forever. **Internet gateways** (igates) forward received frames into **APRS-IS**, the
global TCP/IP backbone, where they feed live maps and databases. The information itself
lives in the packet's payload as a compact human-readable/position-encoded text format
defined by the APRS specification.

## Technical characteristics

| Property | Value |
|----------|-------|
| Link layer | [AX.25](/reference/ax25/) UI (unnumbered information) frames |
| Modulation | 1200 bps [AFSK](/reference/afsk/), Bell 202 (1200/2200 Hz tones) |
| Band/frequency | 144.390 MHz (NA), 144.800 MHz (Europe), others |
| Integrity | 16-bit FCS ([CRC](/reference/cyclic-redundancy-check/)) |
| Relaying | Digipeaters (WIDEn-N paths), igates → APRS-IS |
| Content | Position, weather, telemetry, objects, messages, status |

## How it works

At the physical layer APRS is **Bell 202 AFSK**: a 1200 Hz tone for one bit value and
a 2200 Hz tone for the other, transmitted as audio through an ordinary FM voice radio.
The two audio tones ride the FM carrier, so any FM receiver — or an SDR FM demodulator
— recovers the tones, and a decoder then distinguishes them by frequency to rebuild the
bit stream. Bits are [NRZI](/reference/nrzi/)-encoded (a *change* of tone signals a 0),
which removes the need to know absolute tone polarity. Above that sits the AX.25 frame:
flag-delimited, bit-stuffed, and checked by a 16-bit
[FCS](/reference/cyclic-redundancy-check/). A common software path is a **soft** TNC
such as [Direwolf](/reference/direwolf/), which does the AFSK demodulation and
[KISS](/reference/kiss-tnc/) framing in software rather than in a hardware modem;
GopherTrunk performs the equivalent DSP internally.

## History

APRS was created by **Bob Bruninga (WB4APR)** starting in the 1980s and refined through
the 1990s, growing out of his interest in real-time tactical mapping for amateur
events and emergencies.[^wiki] The design's insistence on *broadcast* beacons rather
than connected links — a departure from the packet-radio norm of the era — is what let
it scale into a live situational picture. It remains one of amateur radio's most active
data modes decades later.

## Deployment

APRS is used worldwide for position tracking (vehicles, hikers, balloons, and
satellites), automated weather stations, event and emergency communications, and
station-to-station messaging. The RF network of digipeaters and igates feeds APRS-IS,
which in turn drives public tracking sites. It coexists with, and is built on top of,
general-purpose [packet radio](/reference/packet-radio/) infrastructure.

## Decoding it with GopherTrunk

APRS is fully in scope for GopherTrunk and requires no proprietary components. The
decoder FM-demodulates the channel, recovers the Bell 202 tones with timing recovery,
reverses the NRZI and bit-stuffing, validates the AX.25 FCS, and parses the APRS
payload into position, weather, telemetry, and message records. See the
[APRS / AX.25 decoder](/aprs.html) page.

## Sources

[^wiki]: [Automatic Packet Reporting System](https://en.wikipedia.org/wiki/Automatic_Packet_Reporting_System) — Wikipedia, for the APRS data network, its AX.25/AFSK link layer, digipeater/igate relaying, payload types, and origins.
[^spec]: [APRS Protocol Reference (APRS101)](http://www.aprs.org/doc/APRS101.PDF) — the APRS Working Group specification, for the packet payload formats, position encoding, and message types.
