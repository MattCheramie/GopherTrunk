---
slug: glonass
title: GLONASS
entry_type: protocol
category: satellite-gnss
description: "GLONASS is Russia's satellite navigation system, historically distinguished by giving each satellite its own L-band frequency (FDMA) rather than a shared CDMA channel."
keywords: GLONASS, Globalnaya Navigatsionnaya Sputnikovaya Sistema, Russia satellite navigation, FDMA, GNSS, L1OF, Roscosmos, satnav
aka: [GLONASS]
autolink: true
infobox:
  - { label: Type, value: Satellite navigation (GNSS) }
  - { label: Operator, value: Roscosmos / Russian Space Forces }
  - { label: Introduced, value: "1982 (global service 2011)" }
  - { label: Access, value: "FDMA (legacy), CDMA (modern)" }
  - { label: Band, value: "L1 ~1602 MHz, L2 ~1246 MHz" }
  - { label: Modulation, value: BPSK-spread DSSS }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [gnss, fdma, gps-gnss, doppler-shift, multilateration]
cite_urls:
  - https://en.wikipedia.org/wiki/GLONASS
  - https://www.glonass-iac.ru/en/
---

**GLONASS** (from the Russian *Globalnaya Navigatsionnaya Sputnikovaya Sistema*,
"Global Navigation Satellite System") is Russia's satellite navigation system and one
of the four global members of the [GNSS](/reference/gnss/) family.[^wiki] Its defining
technical feature is historical: where [GPS](/reference/gps-gnss/) separates
satellites by unique codes on one frequency, legacy GLONASS gives **each satellite its
own frequency** — a textbook use of [FDMA](/reference/fdma/) in space.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="GLONASS assigns each satellite a distinct frequency channel across the L1 band, in contrast to code-division sharing." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="120" x2="440" y2="120" stroke="currentColor" stroke-opacity="0.5"/>
  <text x="235" y="145" text-anchor="middle" font-size="10" fill="currentColor">frequency → (each satellite on its own channel)</text>
  <g fill="currentColor" fill-opacity="0.3" stroke="currentColor">
    <rect x="50" y="60" width="34" height="60"/>
    <rect x="120" y="45" width="34" height="75"/>
    <rect x="190" y="70" width="34" height="50"/>
    <rect x="260" y="52" width="34" height="68"/>
    <rect x="330" y="66" width="34" height="54"/>
  </g>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="67" y="52">k=-2</text><text x="137" y="37">k=-1</text><text x="207" y="62">k=0</text><text x="277" y="44">k=+1</text><text x="347" y="58">k=+2</text>
  </g>
</svg>
<figcaption>Legacy GLONASS separates satellites by frequency channel k, each 562.5 kHz apart, rather than by code.</figcaption>
</figure>

## Overview

GLONASS is a constellation of 24 satellites in medium Earth orbit, in three
orbital planes at about 19,100 km — slightly lower than GPS. Its high orbital
inclination (about 64.8°) was chosen for good coverage at high latitudes, making it
particularly effective across Russia's northern territory. Like every
[GNSS](/reference/gnss/), it works by one-way ranging: satellites broadcast time and
orbit data, and a receiver measures pseudoranges to four or more of them and
[multilaterates](/reference/multilateration/) its position and clock error.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access (legacy) | [FDMA](/reference/fdma/): channel spacing 562.5 kHz on L1 |
| Access (modern) | CDMA on new L3/L1OC signals |
| Carrier | L1 ≈ 1602 MHz, L2 ≈ 1246 MHz |
| Satellites | 24 in three MEO planes, ~19,100 km |
| Inclination | 64.8° (high-latitude coverage) |
| Modulation | BPSK direct-sequence spread spectrum |

The FDMA scheme means antipodal satellites (never simultaneously visible) can reuse
the same channel number, so a modest set of frequencies covers the whole
constellation. The trade-off is that a receiver's front end must span a wider band and
handle per-channel [Doppler shift](/reference/doppler-shift/) offsets, which is one
reason modern GLONASS signals are migrating toward CDMA for easier interoperability
with GPS, Galileo, and BeiDou.

## History

The Soviet Union began launching GLONASS satellites in 1982 and declared the system
operational in 1993, but the constellation decayed after the Soviet collapse and did
not return to full global coverage until 2011 following a sustained modernization
program. Newer GLONASS-K satellites add CDMA signals alongside the legacy FDMA ones.

## Deployment

GLONASS is used worldwide, almost always in combination with other constellations:
most modern smartphone and survey receivers track GPS, GLONASS, Galileo, and BeiDou
together for faster, more robust fixes. Adding GLONASS to a multi-constellation
receiver noticeably improves availability in urban canyons and at high latitudes.

## Decoding it with GopherTrunk

GopherTrunk does **not** decode GLONASS. As a VHF/UHF land-mobile trunking scanner it
has no L-band satellite navigation path. GLONASS is receivable with a general-purpose
software-defined radio and an active L-band antenna — its FDMA structure actually makes
the individual carriers easier to see on a spectrum display than code-multiplexed GPS —
but recovering a position fix still requires wideband capture and dedicated correlation
software. For GopherTrunk it is out of scope; any GNSS involvement is limited to
external timing hardware.

## Sources

[^wiki]: [GLONASS](https://en.wikipedia.org/wiki/GLONASS) — Wikipedia, for GLONASS history, the FDMA channelization and 562.5 kHz spacing, the high-inclination orbit, and the migration toward CDMA signals.
