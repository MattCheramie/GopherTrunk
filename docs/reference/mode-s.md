---
slug: mode-s
title: Mode S
entry_type: protocol
category: protocols
description: Mode S is the 1090 MHz selective-addressing aviation transponder protocol underlying ADS-B, carrying a 24-bit aircraft address and CRC-protected data frames.
keywords: Mode S, Mode-S, 1090 MHz, transponder, ICAO 24-bit address, squitter, ADS-B, CRC-24
aka: [Mode S, Mode-S]
autolink: true
see_also: [ads-b, compact-position-reporting, cyclic-redundancy-check, icao]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/other-signals/ }
external:
  - { title: "Secondary surveillance radar — Mode S (Wikipedia)", url: https://en.wikipedia.org/wiki/Secondary_surveillance_radar#Mode_S }
  - { title: "GopherTrunk ADS-B decoder", url: /adsb.html }
---

**Mode S** (mode select) is the selective-addressing aviation transponder protocol on
**1090 MHz** that carries each aircraft's unique **24-bit ICAO address** and forms the
foundation of [ADS-B](/reference/ads-b/). Messages are 56- or 112-bit frames protected
by a [CRC-24](/reference/cyclic-redundancy-check/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A Mode S frame: a short pulse preamble followed by a data block of address and payload ending in a CRC." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.2"><rect x="30" y="45" width="14" height="26" fill="currentColor" fill-opacity="0.4"/><rect x="50" y="45" width="14" height="26" fill="currentColor" fill-opacity="0.4"/><rect x="74" y="45" width="14" height="26" fill="currentColor" fill-opacity="0.4"/></g>
  <text x="58" y="86" text-anchor="middle" font-size="8" fill="currentColor">preamble</text>
  <rect x="110" y="45" width="120" height="26" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="170" y="62" text-anchor="middle" font-size="8.5" fill="currentColor">ICAO addr + data</text>
  <rect x="230" y="45" width="180" height="26" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/><text x="320" y="62" text-anchor="middle" font-size="8.5" fill="currentColor">payload</text>
  <text x="230" y="98" text-anchor="middle" font-size="8" fill="currentColor">56- or 112-bit frame · CRC-24 · 1090 MHz PPM</text>
</svg>
<figcaption>A Mode S frame: pulse preamble, then a CRC-protected data block keyed by the aircraft's 24-bit address.</figcaption>
</figure>

## Overview

ADS-B is carried in *extended squitter* (DF17/18) Mode S frames whose payload includes
identity, position, and velocity. GopherTrunk decodes these (via a BEAST upstream or
native demod) into tracked aircraft — see the [ADS-B](/adsb.html) page.
