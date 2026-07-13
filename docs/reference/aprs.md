---
slug: aprs
title: APRS
entry_type: protocol
category: paging-data
description: APRS (Automatic Packet Reporting System) is an amateur-radio data network for real-time position, weather, telemetry, and messaging, carried over AX.25 packet using AFSK.
keywords: APRS, Automatic Packet Reporting System, amateur radio, 144.390, AX.25, AFSK, Bob Bruninga, position reporting
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
see_also: [ax25, afsk, mueller-muller-timing-recovery, cyclic-redundancy-check]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/rf-sdr/other-signals/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Automatic_Packet_Reporting_System
external:
  - { title: "GopherTrunk APRS decoder", url: /aprs.html }
---

**APRS** (**Automatic Packet Reporting System**) is an amateur-radio data network for
real-time tactical information — **position, weather, telemetry, and messaging**. It
is carried as [AX.25](/reference/ax25/) packet frames using
[AFSK](/reference/afsk/), most commonly on 144.390 MHz in North America.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="An APRS packet carrying a callsign and position payload over AX.25 on 144.39 MHz." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1"><rect x="30" y="40" width="90" height="28" fill="currentColor" fill-opacity="0.12"/><rect x="120" y="40" width="220" height="28" fill="currentColor" fill-opacity="0.22"/><rect x="340" y="40" width="90" height="28" fill="none"/></g>
  <g font-size="8.5" fill="currentColor" text-anchor="middle"><text x="75" y="58">callsign</text><text x="230" y="58">position / weather / message</text><text x="385" y="58">FCS</text></g>
  <text x="230" y="88" text-anchor="middle" font-size="8" fill="currentColor">AFSK over AX.25 · 144.39 MHz (NA)</text>
</svg>
<figcaption>APRS carries position and telemetry in AX.25 packets, usually via 1200-baud AFSK on 144.39 MHz.</figcaption>
</figure>

## Overview

Stations beacon their position (often from GPS), and digipeaters relay packets so
local activity propagates across a region; internet gateways (igates) bridge RF to the
global APRS-IS network. The result is a live map of amateur stations, weather sensors,
and mobile trackers.

## Technical characteristics

| Property | Value |
|----------|-------|
| Link layer | AX.25 UI frames |
| Modulation | 1200 bps AFSK (Bell 202 tones) |
| Integrity | FCS / CRC-16 |
| Content | Position, weather, telemetry, messages |

## History

Created by Bob Bruninga (WB4APR) in the 1980s–90s; it remains one of amateur radio's
most active data modes.[^wiki]

## Deployment

Amateur radio worldwide via beacons, digipeaters, igates, and the APRS-IS network.

## Decoding it with GopherTrunk

GopherTrunk demodulates the AFSK, recovers AX.25 frames, and decodes APRS payloads.
See the [APRS / AX.25 decoder](/aprs.html) page.

## Sources

[^wiki]: [Automatic Packet Reporting System](https://en.wikipedia.org/wiki/Automatic_Packet_Reporting_System) — Wikipedia, for the APRS data network, its AX.25/AFSK link layer, payload types, and origins.
