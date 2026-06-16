---
slug: aprs
title: APRS
entry_type: protocol
category: protocols
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
  - { title: "Other signals you'll meet", url: /learn/other-signals/ }
external:
  - { title: "Automatic Packet Reporting System (Wikipedia)", url: https://en.wikipedia.org/wiki/Automatic_Packet_Reporting_System }
  - { title: "GopherTrunk APRS decoder", url: /aprs.html }
---

**APRS** (**Automatic Packet Reporting System**) is an amateur-radio data network for
real-time tactical information — **position, weather, telemetry, and messaging**. It
is carried as [AX.25](/reference/ax25/) packet frames using
[AFSK](/reference/afsk/), most commonly on 144.390 MHz in North America.

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
most active data modes.

## Deployment

Amateur radio worldwide via beacons, digipeaters, igates, and the APRS-IS network.

## Decoding it with GopherTrunk

GopherTrunk demodulates the AFSK, recovers AX.25 frames, and decodes APRS payloads.
See the [APRS / AX.25 decoder](/aprs.html) page.
