---
slug: mode-s
title: Mode S
entry_type: protocol
category: aviation-marine
description: Mode S is the 1090 MHz selective-addressing aviation transponder protocol underlying ADS-B, carrying a 24-bit aircraft address and CRC-protected data frames.
keywords: Mode S, Mode-S, 1090 MHz, transponder, secondary surveillance radar, ICAO 24-bit address, squitter, extended squitter, DF17, ADS-B, TCAS, CRC-24, interrogation reply
aka: [Mode S, Mode-S, "mode select"]
autolink: true
infobox:
  - { label: Type, value: Aviation transponder / SSR reply protocol }
  - { label: Standards body, value: ICAO Annex 10 / RTCA / EUROCAE }
  - { label: Frequency, value: 1030 MHz up / 1090 MHz down }
  - { label: Access, value: Selective address + all-call }
  - { label: Modulation, value: PPM, 1 Mbps (DPSK on 1030 uplink) }
  - { label: Frame, value: 56- or 112-bit, CRC-24 }
  - { label: GopherTrunk support, value: Decoded (via ADS-B path) }
see_also: [ads-b, mode-ac, tcas, multilateration, compact-position-reporting, uat-978, cyclic-redundancy-check, icao]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/rf-sdr/other-signals/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Secondary_surveillance_radar#Mode_S
  - https://mode-s.org/decode/
external:
  - { title: "GopherTrunk ADS-B decoder", url: /adsb.html }
---

**Mode S** (**mode select**) is the selective-addressing aviation transponder protocol
on **1090 MHz** that carries each aircraft's unique **24-bit ICAO address** and forms
the foundation of [ADS-B](/reference/ads-b/). It is the modern layer of secondary
surveillance radar (SSR): a ground interrogator can address one specific aircraft
instead of prompting every transponder in the beam, cutting the mutual interference
("garbling") that plagues the older [Mode A/C](/reference/mode-ac/) scheme. Messages are
56- or 112-bit frames protected by a [CRC-24](/reference/cyclic-redundancy-check/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A Mode S frame: a short pulse preamble followed by a data block of downlink format, ICAO address, and payload, ending in a 24-bit CRC." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.2"><rect x="30" y="45" width="10" height="26" fill="currentColor" fill-opacity="0.4"/><rect x="44" y="45" width="10" height="26" fill="currentColor" fill-opacity="0.4"/><rect x="64" y="45" width="10" height="26" fill="currentColor" fill-opacity="0.4"/><rect x="78" y="45" width="10" height="26" fill="currentColor" fill-opacity="0.4"/></g>
  <text x="58" y="86" text-anchor="middle" font-size="8" fill="currentColor">preamble</text>
  <rect x="110" y="45" width="60" height="26" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="140" y="62" text-anchor="middle" font-size="8" fill="currentColor">DF / addr</text>
  <rect x="170" y="45" width="150" height="26" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/><text x="245" y="62" text-anchor="middle" font-size="8.5" fill="currentColor">payload / ADS-B</text>
  <rect x="320" y="45" width="90" height="26" fill="none" stroke="currentColor" stroke-width="1.2" stroke-dasharray="3 2"/><text x="365" y="62" text-anchor="middle" font-size="8.5" fill="currentColor">CRC-24</text>
  <text x="230" y="100" text-anchor="middle" font-size="8" fill="currentColor">56- or 112-bit frame · CRC-24 · 1090 MHz · 1 Mbps PPM</text>
</svg>
<figcaption>A Mode S frame: pulse preamble, then a CRC-protected data block keyed by the aircraft's 24-bit address.</figcaption>
</figure>

## Overview

A Mode S transaction is defined by the interrogator on **1030 MHz** and the reply on
**1090 MHz**. The downlink reply begins with a fixed four-pulse **preamble** and then a
data block whose first five bits are the **downlink format** (DF) that names the message
type. Short 56-bit replies cover altitude (DF4/DF20), identity (DF5/DF21), and
all-call/acquisition (DF11); long 112-bit replies carry the *Comm-B* and *Extended
Squitter* content. The waveform is the same **pulse-position modulation at 1 Mbps** used
by ADS-B — each 1 µs bit places its pulse in the first or second half-microsecond. Every
frame ends with a **24-bit CRC** whose parity is *overlaid* with the aircraft address on
addressed replies, so a receiver both checks integrity and recovers the address in one
step; on Extended Squitter the address is sent in the clear and the CRC stands alone.[^decode]

Squitters are unsolicited transmissions a transponder emits without being interrogated.
The *short* acquisition squitter (DF11) announces the 24-bit address roughly once a
second; the *[Extended Squitter](/reference/ads-b/)* (DF17/DF18) carries the ADS-B
position, velocity, and identification payloads. This is the key point for SDR users:
ADS-B is not a separate radio system but a specific set of Mode S message types.

## Technical characteristics

| Property | Value |
|----------|-------|
| Uplink | 1030 MHz, DPSK interrogation |
| Downlink | 1090 MHz, PPM reply, 1 Mbps |
| Frames | 56-bit (short) / 112-bit (long) |
| Address | 24-bit ICAO aircraft address |
| Format field | 5-bit downlink format (DF) |
| Integrity | CRC-24, address-overlaid on addressed replies |
| Key formats | DF4/5 (altitude/ID), DF11 (all-call), DF17/18 (ADS-B) |

## History

Mode S was developed from the 1970s–80s to overcome the capacity and interference
limits of Mode A/C, in which every transponder in an interrogator's beam replies at
once. By assigning each airframe a permanent 24-bit address, Mode S lets a ground
station poll aircraft individually and exchange data-link messages (Comm-A/Comm-B). It
was standardised in [ICAO](/reference/icao/) Annex 10 Volume IV, with RTCA and EUROCAE
MOPS elaborating the transponder and Extended Squitter behaviour. Mode S also underpins
airborne collision avoidance: [TCAS](/reference/tcas/) II coordinates escape manoeuvres
between aircraft over the Mode S link. The Extended Squitter layer was then repurposed
to carry ADS-B, making Mode S the substrate for modern broadcast surveillance.[^wiki]

## Deployment

Mode S transponders are mandatory for most powered aircraft in controlled airspace
worldwide, and the elementary/enhanced surveillance (ELS/EHS) Comm-B registers report
extra parameters such as selected altitude and heading to air-traffic control.
Ground-based [multilateration](/reference/multilateration/) systems time the arrival of
the same Mode S reply at several receivers to locate aircraft that emit only Mode S and
not ADS-B position, and the same technique underlies volunteer "MLAT" networks.

## Decoding it with GopherTrunk

GopherTrunk decodes Mode S as part of its 1090 MHz [ADS-B](/reference/ads-b/) path: it
correlates the four-pulse preamble, slices the PPM bits, reads the downlink format,
verifies (or address-recovers from) the [CRC-24](/reference/cyclic-redundancy-check/),
and hands DF17/DF18 payloads to the ADS-B state machine. Short frames (DF4/5/11) are
recognised for address and altitude even when they carry no position. GopherTrunk can
also accept Mode S frames from an upstream feeder in the BEAST binary format. Interactive
interrogation, TCAS coordination, and the 1030 MHz uplink are outside its scope — it is a
passive receiver. See the [ADS-B decoder](/adsb.html) page.

## Sources

[^wiki]: [Secondary surveillance radar — Mode S](https://en.wikipedia.org/wiki/Secondary_surveillance_radar#Mode_S) — Wikipedia, for the Mode S selective-addressing transponder protocol, 1030/1090 MHz link, 24-bit ICAO address, downlink formats, frame sizes, and CRC.
[^decode]: [The 1090 MHz Riddle — mode-s.org](https://mode-s.org/decode/) — Junzi Sun, TU Delft, an open technical reference for Mode S downlink formats, CRC-24 with address overlay, and Extended Squitter decoding, corroborating the frame details here.
