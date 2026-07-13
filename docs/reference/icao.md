---
slug: icao
title: ICAO
entry_type: organization
category: organizations
description: ICAO, the International Civil Aviation Organization, is the UN agency that standardises aviation, including the ADS-B surveillance system and Mode S transponders.
keywords: ICAO, International Civil Aviation Organization, ADS-B, Mode S, 24-bit address, aviation standards, Annex 10, SARPs
aka: [ICAO, International Civil Aviation Organization]
autolink: true
infobox:
  - { label: Type, value: UN specialised agency }
  - { label: Focus, value: Civil aviation standards }
  - { label: Standards, value: ADS-B, Mode S }
see_also: [ads-b, mode-s, compact-position-reporting, rtca, eurocae, itu]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/rf-sdr/other-signals/ }
cite_urls:
  - https://www.icao.int/
  - https://en.wikipedia.org/wiki/International_Civil_Aviation_Organization
---

**ICAO** (the **International Civil Aviation Organization**) is the United Nations agency
that standardises international civil aviation, including the
**[ADS-B](/reference/ads-b/)** surveillance system and [Mode S](/reference/mode-s/)
transponders.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="ICAO sets aviation Standards and Recommended Practices that regional bodies detail into transponder requirements." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="42" width="100" height="32" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="70" y="55">ICAO</text><text x="70" y="67" font-size="7.5">Annex 10</text>
    <rect x="170" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="225" y="55">RTCA / EUROCAE</text><text x="225" y="67" font-size="7.5">MOPS detail</text>
    <rect x="330" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="385" y="55">Mode S / ADS-B</text><text x="385" y="67" font-size="7.5">at 1090 MHz</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="120" y1="58" x2="169" y2="58" marker-end="url(#rel_icao)"/><line x1="280" y1="58" x2="329" y2="58" marker-end="url(#rel_icao)"/></g>
    <text x="230" y="96" font-size="8" fill="currentColor" fill-opacity="0.85">global SARPs → regional MOPS → interoperable avionics</text>
  </g>
  <defs><marker id="rel_icao" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>ICAO sets international civil-aviation standards; regional bodies turn them into the transponder requirements behind ADS-B.</figcaption>
</figure>

## Overview

ICAO was established by the Chicago Convention of 1944 and is the UN specialised agency for
civil aviation, with more than 190 member states and headquarters in Montréal. It sets
worldwide **Standards and Recommended Practices** (SARPs), published as Annexes to the
convention, that make international air travel safe and interoperable: everything from runway
markings and airspace procedures to the radio systems aircraft use to be seen and to
communicate. The technical radio standards live chiefly in **Annex 10 (Aeronautical
Telecommunications)**, which defines the secondary surveillance radar transponder modes and the
data formats that ride on them.

For an SDR listener the most important of these is the **[Mode S](/reference/mode-s/)**
transponder and the **[ADS-B](/reference/ads-b/)** (Automatic Dependent Surveillance –
Broadcast) system built on top of it. ICAO assigns every aircraft a unique **24-bit address**
(the ICAO hex code) that identifies it in every Mode S and ADS-B message, and it defines the
1090 MHz Extended Squitter format in which aircraft broadcast their identity, position,
altitude, and velocity. Position is encoded using
[Compact Position Reporting](/reference/compact-position-reporting/), a clever scheme ICAO
specifies to squeeze a global latitude/longitude into few bits. ICAO sets the *what* — the
message formats, the address allocations, the required performance — while the detailed
*how* (the Minimum Operational Performance Standards that avionics must pass) is written by
regional bodies: [RTCA](/reference/rtca/)'s DO-260 series in the United States and its
harmonised counterpart, [EUROCAE](/reference/eurocae/)'s ED-102, in Europe. The radio spectrum
those systems occupy is allocated internationally through the [ITU](/reference/itu/).

## Relevance to SDR

ICAO standards define the 1090 MHz signals an SDR receives when tracking aircraft, and that
makes aircraft tracking one of the most accessible and rewarding SDR activities: because ADS-B
is an open, unencrypted broadcast standard designed for interoperability, any receiver can
decode it. When a tool like `dump1090` or a similar decoder shows you an aircraft's flight,
altitude, and position on a map, it is parsing exactly the Annex 10 Extended Squitter format,
resolving the ICAO 24-bit address, and running the Compact Position Reporting algorithm ICAO
specified. The global uniqueness of the 24-bit address is what lets independent ground receivers
worldwide — and crowdsourced networks — stitch together a coherent picture of air traffic.

Aircraft surveillance sits outside GopherTrunk's land-mobile trunking focus, so GopherTrunk
does not itself decode ADS-B; dedicated tools handle 1090 MHz. But the reference stands as
context for the wider RF landscape an SDR user explores, and it explains why aviation signals
are so consistent across the world: a single UN body harmonised them. See the
[other signals you'll meet](/learn/rf-sdr/other-signals/) lesson for where ADS-B fits among the
non-trunking signals you can receive.

## Sources

[^home]: [International Civil Aviation Organization](https://www.icao.int/) — ICAO's official site, for the SARPs and Annex 10 standards behind Mode S and ADS-B.
[^wiki]: [International Civil Aviation Organization](https://en.wikipedia.org/wiki/International_Civil_Aviation_Organization) — Wikipedia, for ICAO's history, structure, and standards role.
