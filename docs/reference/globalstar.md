---
slug: globalstar
title: Globalstar
entry_type: protocol
category: satellite-gnss
description: "Globalstar is a low-earth-orbit satellite constellation providing voice and low-rate data using a bent-pipe CDMA air interface with an L-band uplink and S-band downlink."
keywords: Globalstar, LEO satellite, satellite phone, CDMA, bent-pipe, SPOT, simplex data, L-band, S-band, asset tracking, satellite messaging
aka: [Globalstar]
autolink: true
infobox:
  - { label: Type, value: LEO satellite comms }
  - { label: Standards body, value: Globalstar (Qualcomm CDMA) }
  - { label: Introduced, value: "2000" }
  - { label: Access, value: CDMA (bent-pipe) }
  - { label: Channel spacing, value: "1.6 GHz up / 2.4 GHz down" }
  - { label: Modulation, value: QPSK spread-spectrum }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [iridium, orbcomm, inmarsat, cdma, qpsk, frequency-bands]
cite_urls:
  - https://en.wikipedia.org/wiki/Globalstar
  - https://en.wikipedia.org/wiki/Code-division_multiple_access
---

**Globalstar** is a [low-earth-orbit](/reference/frequency-bands/) satellite
constellation offering satellite-telephone voice, low-rate data, and one-way asset
tracking. It uses a **bent-pipe** [CDMA](/reference/cdma/) air interface: the satellites
carry no onboard processing and simply relay signals between a user handset and a
ground gateway, with an [L-band](/reference/frequency-bands/) uplink near 1.6 GHz and an
S-band downlink near 2.4 GHz.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A Globalstar low-earth-orbit satellite relays a handset signal straight through to a ground gateway without onboard processing." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="gstar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="20" y1="140" x2="440" y2="140" stroke="currentColor" stroke-opacity="0.4"/>
  <circle cx="230" cy="35" r="9" fill="currentColor" fill-opacity="0.3" stroke="currentColor"/>
  <text x="230" y="22" text-anchor="middle" font-size="9" fill="currentColor">LEO satellite (~1414 km) — bent pipe</text>
  <g font-size="9" fill="currentColor" text-anchor="middle"><text x="70" y="158">handset (1.6 GHz)</text><text x="390" y="158">gateway (2.4 GHz)</text></g>
  <line x1="75" y1="128" x2="222" y2="44" stroke="currentColor" marker-end="url(#gstar)"/>
  <line x1="238" y1="44" x2="385" y2="128" stroke="currentColor" marker-end="url(#gstar)"/>
</svg>
<figcaption>Globalstar satellites are bent pipes: each relays the uplink straight down to a gateway, which does the switching on the ground.</figcaption>
</figure>

## Overview

Globalstar was conceived in the 1990s as a lower-cost rival to
[Iridium](/reference/iridium/), trading Iridium's complex satellite-to-satellite
crosslinks for simple bent-pipe repeaters and a dense network of ground gateways. The
constellation of roughly 48 satellites flies in inclined orbits near 1414 km. Because a
call must see both a satellite and a gateway at once, coverage is strong over populated
land and coastal waters but absent over mid-ocean and the poles.[^wiki]

## Technical characteristics

| Property | Value |
|----------|-------|
| Orbit | LEO, ~1414 km, ~52° inclination |
| Satellites | ~48 (plus spares) |
| Access | Qualcomm CDMA spread spectrum |
| Uplink | L-band, ~1.61–1.62 GHz |
| Downlink | S-band, ~2.48–2.50 GHz |
| Modulation | [QPSK](/reference/qpsk/), direct-sequence spread |
| Services | Voice, packet data, SPOT/simplex tracking |

The CDMA waveform is borrowed from terrestrial IS-95 cellular: each user's data is spread
by a pseudo-random code across a wide channel, so many users share the same frequency and
are separated by their codes rather than by time or channel.[^cdma]

## History

Service launched commercially around 2000. A first-generation constellation suffered
amplifier degradation in the late 2000s that hurt two-way voice, largely resolved by a
second-generation replacement fleet. Alongside phones, Globalstar built a large business
in one-way **simplex** data: the consumer SPOT messengers and industrial asset trackers
transmit short bursts up to the satellites for relay to a gateway. In 2022 Apple
contracted Globalstar's spectrum and network for the iPhone's Emergency SOS via
satellite.[^wiki]

## Deployment

Globalstar serves maritime, expedition, and remote-industrial users needing voice and
tracking beyond cellular coverage, and its simplex link carries huge volumes of
low-cost location beacons. The S-band simplex downlink has attracted amateur
[SDR](/reference/software-defined-radio/) experimenters, but the CDMA spreading, the
proprietary framing, and the 2.4 GHz band make it a far harder target than the VHF data
of [Orbcomm](/reference/orbcomm/) or the L-band voice of [Iridium](/reference/iridium/).

## Decoding it with GopherTrunk

GopherTrunk does not decode Globalstar. It is a terrestrial VHF/UHF land-mobile trunking
scanner, and Globalstar's spread-spectrum CDMA physical layer, S-band frequencies, and
satellite framing share nothing with the C4FM/QPSK land-mobile protocols GopherTrunk
implements. It is listed here as the CDMA, bent-pipe member of the LEO family alongside
[Iridium](/reference/iridium/) and [Orbcomm](/reference/orbcomm/).

## Sources

[^wiki]: [Globalstar](https://en.wikipedia.org/wiki/Globalstar) — Wikipedia, for the constellation design, bent-pipe architecture, L/S-band plan, and simplex data services.
[^cdma]: [Code-division multiple access](https://en.wikipedia.org/wiki/Code-division_multiple_access) — Wikipedia, for the spread-spectrum access method Globalstar borrows from IS-95.
