---
slug: ais
title: Automatic Identification System (AIS)
entry_type: protocol
category: aviation-marine
description: AIS (Automatic Identification System) is a maritime VHF data system in which ships broadcast identity, position, course, and speed using GMSK in a self-organising TDMA scheme.
keywords: AIS, Automatic Identification System, marine tracking, GMSK, SOTDMA, 161.975 162.025, AIS channel A B, vessel position, MMSI, NMEA, AIVDM, satellite AIS
aka: [AIS]
autolink: true
infobox:
  - { label: Type, value: Maritime data broadcast }
  - { label: Standards body, value: ITU-R M.1371 / IMO / IEC }
  - { label: Band, value: VHF marine (161.975 / 162.025 MHz) }
  - { label: Access, value: Self-organising TDMA (SOTDMA) }
  - { label: Modulation, value: GMSK, 9600 bps }
  - { label: Content, value: MMSI, position, course, speed }
  - { label: GopherTrunk support, value: Decoded }
see_also: [ads-b, dsc, marine-vhf, epirb-406, gmsk, tdma, itu, cyclic-redundancy-check]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/rf-sdr/other-signals/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Automatic_identification_system
  - https://www.itu.int/rec/R-REC-M.1371
external:
  - { title: "GopherTrunk AIS decoder", url: /ais.html }
---

**AIS** (**Automatic Identification System**) is a maritime safety system in which
ships and shore stations broadcast their **identity, position, course, and speed** on
VHF marine frequencies. It uses [GMSK](/reference/gmsk/) modulation in a
self-organising [TDMA](/reference/tdma/) scheme so that many vessels share just two
channels without a central controller. AIS is to shipping what
[ADS-B](/reference/ads-b/) is to aviation — a continuous, receivable, unencrypted
position broadcast that underpins collision avoidance and traffic monitoring.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="Multiple ships each transmitting a short position burst in its own assigned time slot on a shared marine VHF channel, so bursts do not collide." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="82" x2="430" y2="82" stroke="currentColor" stroke-opacity="0.4"/>
  <g stroke="currentColor" stroke-opacity="0.25" stroke-width="0.8"><line x1="95" y1="45" x2="95" y2="82"/><line x1="165" y1="45" x2="165" y2="82"/><line x1="255" y1="45" x2="255" y2="82"/><line x1="345" y1="45" x2="345" y2="82"/></g>
  <g fill="currentColor" fill-opacity="0.3" stroke="currentColor" stroke-width="1"><rect x="50" y="52" width="22" height="30"/><rect x="120" y="52" width="22" height="30"/><rect x="210" y="52" width="22" height="30"/><rect x="300" y="52" width="22" height="30"/><rect x="370" y="52" width="22" height="30"/></g>
  <text x="230" y="102" text-anchor="middle" font-size="8.5" fill="currentColor">~162 MHz · GMSK · 2250 slots/min · self-organising (SOTDMA)</text>
  <text x="230" y="30" text-anchor="middle" font-size="9" fill="currentColor">each ship claims a time slot for its position burst</text>
</svg>
<figcaption>AIS ships broadcast short position bursts in self-organising time slots on shared marine VHF channels, so transmissions interleave without a controller.</figcaption>
</figure>

## Overview

Each AIS station transmits position reports keyed to its **MMSI** (Maritime Mobile
Service Identity) on **161.975 MHz** (AIS 1 / channel A) and **162.025 MHz** (AIS 2 /
channel B), alternating between the two for diversity. Reports are carried in
HDLC-style packets with bit-stuffing, a training preamble, a start/stop flag, and a
[CRC-16](/reference/cyclic-redundancy-check/); a shore receiver frames the GMSK bit
stream and emits the familiar `!AIVDM` NMEA sentences. There are 27 defined message
types: dynamic position reports (types 1–3), the base-station report (type 4), the
static/voyage report giving name, dimensions, and destination (type 5), safety and
binary messages, and the Class B reports used by smaller craft. The result is a live
map of nearby vessel traffic.

Two equipment classes exist. **Class A**, mandatory on larger SOLAS vessels, uses
**SOTDMA** (Self-Organising TDMA): each transmitter listens, learns which of the 2250
slots per minute per channel are free, and reserves future slots, so the network
schedules itself with no master. Reporting rate scales with speed and manoeuvre — every
2–10 s for a moving ship, every 3 minutes at anchor. **Class B**, for leisure and small
commercial craft, transmits at lower power and less often, typically using CSTDMA
(carrier-sense TDMA) that simply checks a slot is clear before using it.

## Technical characteristics

| Property | Value |
|----------|-------|
| Band | VHF marine, 161.975 / 162.025 MHz |
| Channel spacing | 25 kHz |
| Access | SOTDMA (Class A) / CSTDMA (Class B) |
| Modulation | GMSK, BT 0.4 |
| Bit rate | 9600 bps |
| Framing | HDLC with bit-stuffing, CRC-16 |
| Slots | 2250 per minute per channel |
| Identity | 9-digit MMSI |

## History

AIS was developed in the 1990s and standardised by the [ITU](/reference/itu/) in
Recommendation ITU-R M.1371, with IEC test standards and IMO carriage requirements. The
IMO SOLAS convention mandated Class A AIS on most vessels over 300 gross tons on
international voyages and all passenger ships, phased in from 2002 to 2004, to reduce
collisions and give coastal authorities a real-time traffic picture. Class B was added
later to bring smaller vessels into the picture affordably.[^wiki][^itu]

## Deployment

AIS is now ubiquitous in commercial shipping, port operations, vessel-traffic services
(VTS), and search-and-rescue. Satellite AIS receivers extend coverage over open ocean
where shore stations cannot reach, feeding global ship-tracking services. On the safety
side it works alongside [DSC](/reference/dsc/) distress calling and
[EPIRB 406](/reference/epirb-406/) beacons, and AIS-SART transmitters broadcast a
man-overboard or lifeboat position using the same air interface. As with ADS-B, the
open unencrypted format created a large hobbyist reception community.

## Decoding it with GopherTrunk

GopherTrunk tunes the two AIS channels, demodulates the 9600 bps GMSK bursts,
performs HDLC de-framing (flag detection and bit de-stuffing), validates the
CRC-16, and decodes the message payload into position reports and vessel data. Because
AIS is unencrypted and the modulation is straightforward, it is one of the marine
protocols GopherTrunk handles end to end, alongside its
[marine VHF](/reference/marine-vhf/) and DSC support. See the
[AIS decoder](/ais.html) page for the live plot.

## Sources

[^wiki]: [Automatic identification system](https://en.wikipedia.org/wiki/Automatic_identification_system) — Wikipedia, for the maritime VHF AIS system, its GMSK/SOTDMA air interface, Class A/B equipment, message types, and ITU/IMO standardisation.
[^itu]: [Recommendation ITU-R M.1371](https://www.itu.int/rec/R-REC-M.1371) — International Telecommunication Union, the primary technical standard defining the AIS physical layer (GMSK, 9600 bps), SOTDMA slot structure, and message formats.
