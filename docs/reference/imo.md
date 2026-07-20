---
slug: imo
title: International Maritime Organization (IMO)
entry_type: organization
category: organizations
description: The IMO, the International Maritime Organization, is the UN agency for maritime safety that adopts SOLAS and, through it, mandates the GMDSS radio systems including AIS and DSC.
keywords: IMO, International Maritime Organization, SOLAS, GMDSS, AIS, DSC, EPIRB, NAVTEX, MMSI, maritime safety
aka: [IMO, International Maritime Organization]
autolink: true
infobox:
  - { label: Type, value: UN specialised agency }
  - { label: Focus, value: Maritime safety, pollution prevention }
  - { label: Founded, value: 1948 }
  - { label: Standards, value: SOLAS, GMDSS }
see_also: [ais, dsc, navtex, epirb-406, solas, cospas-sarsat, itu]
cite_urls:
  - https://www.imo.org/
  - https://en.wikipedia.org/wiki/International_Maritime_Organization
---

**IMO** (the **International Maritime Organization**) is the United Nations agency for
maritime safety and the prevention of marine pollution. It adopts the
**[SOLAS](/reference/solas/)** convention and, through it, mandates the radio systems that
keep ships safe — including **[AIS](/reference/ais/)** position reporting and
**[DSC](/reference/dsc/)** digital distress alerting.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="The IMO adopts the SOLAS convention, whose GMDSS chapter mandates four radio systems: AIS, DSC, EPIRB, and NAVTEX." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="15" y="58" width="90" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="60" y="79">IMO</text>
    <rect x="150" y="58" width="100" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="200" y="73">SOLAS</text><text x="200" y="85" font-size="7.5">GMDSS</text>
    <rect x="315" y="10" width="130" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="380" y="27">AIS · position</text>
    <rect x="315" y="44" width="130" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="380" y="61">DSC · distress</text>
    <rect x="315" y="78" width="130" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="380" y="95">EPIRB · beacon</text>
    <rect x="315" y="112" width="130" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="380" y="129">NAVTEX · safety</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="105" y1="75" x2="149" y2="75" marker-end="url(#rel_imo)"/><line x1="250" y1="75" x2="314" y2="23" marker-end="url(#rel_imo)"/><line x1="250" y1="75" x2="314" y2="57" marker-end="url(#rel_imo)"/><line x1="250" y1="75" x2="314" y2="91" marker-end="url(#rel_imo)"/><line x1="250" y1="75" x2="314" y2="125" marker-end="url(#rel_imo)"/></g>
  </g>
  <defs><marker id="rel_imo" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The IMO adopts SOLAS, whose GMDSS provisions mandate the maritime radio systems an SDR listener meets on the VHF and HF bands.</figcaption>
</figure>

## Overview

The IMO was established in 1948 (as the Inter-Governmental Maritime Consultative Organization,
renamed in 1982) and is the UN specialised agency responsible for the safety and security of
shipping and the prevention of pollution by ships. It works largely by adopting international
conventions that member states then bring into national law. Its flagship instrument is the
**[SOLAS](/reference/solas/)** convention (Safety of Life at Sea), whose radio chapter defines
the **Global Maritime Distress and Safety System (GMDSS)** — the mandatory suite of automated
alerting and safety-communication equipment carried by SOLAS-class ships.

Several of the GMDSS components are radio systems an SDR user will recognise.
**[DSC](/reference/dsc/)** (Digital Selective Calling) sends digital distress and calling
alerts on dedicated channels; **[NAVTEX](/reference/navtex/)** broadcasts navigational and
meteorological safety messages as narrow-band direct printing on 518 kHz;
**[EPIRB](/reference/epirb-406/)** beacons transmit distress alerts on 406 MHz to the
[Cospas-Sarsat](/reference/cospas-sarsat/) satellite system; and **[AIS](/reference/ais/)**
(Automatic Identification System) continuously reports each ship's identity, position, course,
and speed on VHF. The IMO also issues the permanent **IMO ship identification number**, and
together with the [ITU](/reference/itu/) it underpins the **MMSI** (Maritime Mobile Service
Identity) numbers that identify vessels in DSC and AIS traffic. The ITU allocates the spectrum
and specifies the technical radio characteristics; the IMO mandates that ships carry and use
the equipment.

## Relevance to SDR

The IMO's mandates are why coastal VHF is full of decodable maritime data. Because AIS is an
open, unencrypted broadcast on 161.975 and 162.025 MHz, a modest SDR and an AIS decoder can
plot ships in real time, and the same receivers feed crowdsourced vessel-tracking networks.
NAVTEX on 518 kHz and DSC on the VHF and HF distress channels are likewise receivable and
decodable with common SDR tools, letting a listener read the safety-broadcast and alerting
layer the IMO built into GMDSS.

Maritime signals sit outside GopherTrunk's land-mobile trunking focus, so GopherTrunk does not
itself decode AIS or NAVTEX; dedicated tools handle those. The reference stands as context for
the wider RF landscape an SDR user explores, and it explains why maritime radio is so uniform
worldwide: a single UN body, through SOLAS, harmonised it.

## Sources

[^home]: [International Maritime Organization](https://www.imo.org/) — the IMO's official site, for SOLAS, the GMDSS, and the maritime identity schemes.
[^wiki]: [International Maritime Organization](https://en.wikipedia.org/wiki/International_Maritime_Organization) — Wikipedia, for the IMO's history, structure, and conventions.
