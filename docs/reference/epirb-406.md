---
slug: epirb-406
title: EPIRB (406 MHz)
entry_type: protocol
category: aviation-marine
description: A 406 MHz EPIRB is a maritime distress beacon that transmits a coded digital burst to the Cospas-Sarsat satellite system to alert rescuers and identify a vessel in an emergency.
keywords: EPIRB, 406 MHz, emergency position indicating radio beacon, Cospas-Sarsat, distress beacon, 121.5 MHz homing, GPS EPIRB, hex ID, maritime distress, GMDSS, SART
aka: [EPIRB, 406 MHz EPIRB]
autolink: true
infobox:
  - { label: Type, value: Maritime distress beacon }
  - { label: Standards body, value: Cospas-Sarsat / ITU }
  - { label: Introduced, value: 406 MHz digital from 1980s–90s }
  - { label: Frequency, value: 406.0–406.1 MHz (+ 121.5 MHz homer) }
  - { label: Modulation, value: Phase-modulated digital burst }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [cospas-sarsat, dsc, emergency-call, phase-shift-keying]
cite_urls:
  - https://en.wikipedia.org/wiki/Emergency_position-indicating_radiobeacon
  - https://cospas-sarsat.int/
---

**An EPIRB (Emergency Position-Indicating Radio Beacon) operating on 406 MHz** is a
maritime distress transmitter that, once activated, sends a **coded digital burst to the
Cospas-Sarsat satellite constellation** to summon rescue and identify the vessel in
trouble. The burst carries a registered beacon identity and, in GPS-equipped units, a
position, so a rescue coordination centre knows **who** is in distress and **where**
almost immediately.[^wiki] A 406 MHz EPIRB is the satellite backbone of maritime distress
alerting, working alongside [DSC](/reference/dsc/) and other GMDSS elements.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="An activated EPIRB on a life raft transmitting a coded 406 megahertz burst up to a satellite, which relays the beacon identity and position to a rescue center." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="epirbar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <path d="M30 120 q30 -14 60 0" fill="none" stroke="currentColor"/>
  <rect x="52" y="96" width="8" height="20" fill="currentColor"/>
  <text x="56" y="135" text-anchor="middle" font-size="8" fill="currentColor">EPIRB</text>
  <circle cx="235" cy="30" r="9" fill="none" stroke="currentColor"/>
  <text x="235" y="18" text-anchor="middle" font-size="8" fill="currentColor">satellite</text>
  <line x1="62" y1="96" x2="226" y2="36" stroke="currentColor" marker-end="url(#epirbar)"/>
  <text x="120" y="66" font-size="7.5" fill="currentColor">406 MHz coded burst</text>
  <rect x="360" y="95" width="80" height="30" rx="3" fill="none" stroke="currentColor"/>
  <text x="400" y="114" text-anchor="middle" font-size="8" fill="currentColor">rescue centre</text>
  <line x1="244" y1="36" x2="374" y2="94" stroke="currentColor" marker-end="url(#epirbar)"/>
  <text x="300" y="62" font-size="7.5" fill="currentColor">ID + position relayed</text>
</svg>
<figcaption>An activated 406 MHz EPIRB sends a coded burst to Cospas-Sarsat satellites, which relay the beacon identity and position to a rescue coordination centre.</figcaption>
</figure>

## Overview

Each 406 MHz beacon is programmed with a unique **15-character hexadecimal identity** tied
to a national registration database, which links the beacon to a specific vessel, owner,
and emergency contacts. When activated — manually or automatically on immersion — the
EPIRB transmits a short digital message roughly every 50 seconds. Modern beacons embed an
internal GNSS receiver so the message includes a precise position; even without it,
satellites can locate older beacons by Doppler processing. A low-power **121.5 MHz**
signal is also radiated so rescuers can home in over the final distance.

## Technical characteristics

| Property | Value |
|----------|-------|
| Alert frequency | 406.0–406.1 MHz |
| Homing frequency | 121.5 MHz |
| Message | 15-hex-character beacon ID, optional encoded position |
| Modulation | Phase-modulated (biphase) digital burst |
| Repetition | ~50 s |
| System | [Cospas-Sarsat](/reference/cospas-sarsat/) satellites |

## History

The analog 121.5 MHz distress beacon came first but suffered false alerts and poor
location accuracy. The digital 406 MHz beacon, with its registered identity and stronger
coding, was introduced through the Cospas-Sarsat programme; satellite processing of the
121.5 MHz channel was discontinued in 2009, leaving 406 MHz as the sole satellite-alerting
frequency. EPIRBs are the maritime member of a family that also includes aviation ELTs
and personal PLBs.

## Deployment

Required on SOLAS vessels and carried voluntarily on many smaller craft, EPIRBs are a
mandated part of GMDSS. The same 406 MHz infrastructure serves aviation and land beacons,
and it interlocks with an [emergency-call](/reference/emergency-call/) response chain of
rescue coordination centres.

## Decoding it with GopherTrunk

GopherTrunk does **not** decode 406 MHz EPIRB bursts. It is a land-mobile trunking
scanner, and EPIRB is a specialised safety-of-life signal that should never be transmitted
except in genuine distress; casual decoding is out of GT's scope and, for live beacons, a
matter for the official Cospas-Sarsat system rather than hobby receivers.

## Sources

[^wiki]: [Emergency position-indicating radiobeacon](https://en.wikipedia.org/wiki/Emergency_position-indicating_radiobeacon) — Wikipedia, for the 406 MHz EPIRB distress function, Cospas-Sarsat satellite alerting, registered beacon identity, GPS position encoding, and the 121.5 MHz homing signal.
