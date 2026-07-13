---
slug: hfdl
title: HFDL
entry_type: protocol
category: aviation-marine
description: "HFDL (High Frequency Data Link) is a long-range aviation datalink using PSK on HF shortwave, relaying ACARS traffic via sky-wave propagation over oceans and polar routes."
keywords: HFDL, High Frequency Data Link, HF ACARS, shortwave datalink, PSK, sky-wave, oceanic, polar, ARINC 635, ground station network, aviation HF
aka: [HFDL, HF Data Link]
autolink: true
infobox:
  - { label: Type, value: HF long-range aviation datalink }
  - { label: Standards body, value: "ARINC 635 / ICAO" }
  - { label: Access, value: "TDMA (32 s slot map)" }
  - { label: Band, value: "HF, 2.6–22 MHz" }
  - { label: Modulation, value: "PSK, 300–1800 bps" }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [phase-shift-keying, sky-wave, acars, ionospheric-propagation, ads-c, shortwave-broadcast]
cite_urls:
  - https://en.wikipedia.org/wiki/High_Frequency_Data_Link
  - https://en.wikipedia.org/wiki/Aircraft_Communications_Addressing_and_Reporting_System
---

**HFDL** (**High Frequency Data Link**) is a long-range aviation datalink that carries
[ACARS](/reference/acars/) traffic over **HF shortwave** using
[phase-shift keying](/reference/phase-shift-keying/), reaching aircraft thousands of
kilometres from any ground station by [sky-wave](/reference/sky-wave/)
[ionospheric propagation](/reference/ionospheric-propagation/).[^wiki] Where VHF
datalink is line-of-sight and satcom needs a satellite in view, HFDL exploits the
ionosphere's ability to bend HF signals around the curvature of the Earth, making it the
economical datalink for **oceanic and polar** routes.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="An aircraft's HF signal reflects off the ionosphere and travels over the horizon to a distant ground station, illustrating sky-wave propagation of the HFDL datalink." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="hfdlar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <path d="M20 130 Q230 120 440 130" stroke="currentColor" stroke-width="1.2" fill="none"/>
  <text x="30" y="145" font-size="7.5" fill="currentColor">Earth</text>
  <path d="M20 40 Q230 20 440 40" stroke="currentColor" stroke-width="1" stroke-dasharray="4 3" fill="none"/>
  <text x="230" y="18" text-anchor="middle" font-size="7.5" fill="currentColor">ionosphere</text>
  <path d="M60 118 l25 6 l-6 -12 m6 12 l10 -8" stroke="currentColor" stroke-width="1.4" fill="none"/>
  <text x="70" y="112" font-size="7.5" fill="currentColor">aircraft</text>
  <path d="M85 115 L225 45" stroke="currentColor" stroke-width="1.2" marker-end="url(#hfdlar)"/>
  <path d="M235 45 L390 118" stroke="currentColor" stroke-width="1.2" marker-end="url(#hfdlar)"/>
  <path d="M385 118 v-16 m-6 0 l6 -8 l6 8" stroke="currentColor" stroke-width="1.4" fill="none"/>
  <text x="385" y="132" text-anchor="middle" font-size="7.5" fill="currentColor">HF ground station</text>
  <text x="230" y="72" text-anchor="middle" font-size="8" fill="currentColor">PSK · 2.6–22 MHz · sky-wave</text>
</svg>
<figcaption>HFDL relies on sky-wave: the aircraft's HF signal reflects off the ionosphere to reach a ground station far over the horizon.</figcaption>
</figure>

## Overview

HFDL is run as a single global network: a handful of HF ground stations, each on
several frequencies across the HF spectrum, together provide near-worldwide coverage.
Aircraft log onto whichever station and frequency propagate best at the current time of
day, and the network hands messages to and from the same [ACARS](/reference/acars/)
back-end that VHF and satcom use. Because HF conditions swing with the ionosphere,
ground stations broadcast a squitter advertising which frequencies are usable.

## Technical characteristics

| Property | Value |
|----------|-------|
| Band | HF, roughly 2.6–22 MHz (upper sideband) |
| Modulation | 2/4/8-PSK, single tone |
| Data rates | 300, 600, 1200, and 1800 bps (condition-adaptive) |
| Channel bandwidth | ~3 kHz (voice-channel compatible) |
| Access | TDMA within a 32-second frame (slot map) |
| Ground stations | Global network on shared frequencies |
| Payload | ACARS (incl. ADS-C, position, ops) |
| Standard | ARINC 635 |

The modulation is single-tone PSK whose order and rate adapt to propagation — from
robust BPSK at 300 bps in poor conditions up to 8-PSK at 1800 bps when the channel is
strong. A 32-second TDMA frame divided into slots lets many aircraft share one ground
station's frequency, coordinated by the station's slot assignments.

## History

HFDL was standardised in ARINC 635 and entered service in the late 1990s as a low-cost
alternative to HF voice position reporting and to satcom, particularly for operators
crossing oceanic and polar airspace where VHF cannot reach. Its ground network grew to
give effectively global coverage.

## Deployment

HFDL is widely fitted to long-haul aircraft and is popular with shortwave listeners
because, like VHF [ACARS](/reference/acars/), it is unencrypted and decodable with a
capable HF SDR and software. It commonly carries [ADS-C](/reference/ads-c/) position
contracts over regions with no other datalink coverage, complementing satcom.

## Decoding it with GopherTrunk

**Not decoded.** HFDL is an HF datalink, outside GopherTrunk's land-mobile VHF/UHF and
1090 MHz [ADS-B](/reference/ads-b/) scope, and it requires an HF-capable receiver GT
does not target. Enthusiasts decode HFDL with dedicated HF tooling; this page is
included for honest context on the aviation datalink family alongside
[ACARS](/reference/acars/) and [VDL Mode 2](/reference/vdl-mode-2/), and to point at the
[sky-wave](/reference/sky-wave/) propagation that makes it work.

## Sources

[^wiki]: [High Frequency Data Link](https://en.wikipedia.org/wiki/High_Frequency_Data_Link) — Wikipedia, for HFDL's PSK HF physical layer, adaptive 300–1800 bps rates, 32-second TDMA slot map, global ground-station network, and its role carrying ACARS over oceanic and polar routes.
