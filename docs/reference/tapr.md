---
slug: tapr
title: Tucson Amateur Packet Radio (TAPR)
entry_type: organization
category: organizations
description: TAPR, Tucson Amateur Packet Radio, is a US amateur-radio non-profit that pioneered digital communications, building the early TNCs behind AX.25 packet radio and co-defining the KISS protocol.
keywords: TAPR, Tucson Amateur Packet Radio, TNC, TNC-2, AX.25, KISS, packet radio, open hardware, DCC, amateur radio
aka: [TAPR, Tucson Amateur Packet Radio]
autolink: true
infobox:
  - { label: Type, value: Amateur-radio non-profit R&D }
  - { label: Focus, value: Amateur digital communications }
  - { label: Founded, value: 1982 }
  - { label: Standards, value: "TNC hardware, KISS" }
see_also: [packet-radio, ax25, kiss-tnc, arrl, aprs]
cite_urls:
  - https://tapr.org/
  - https://en.wikipedia.org/wiki/Tucson_Amateur_Packet_Radio
---

**TAPR** (**Tucson Amateur Packet Radio**) is a US amateur-radio non-profit that pioneered
amateur digital communications. It developed the early terminal node controllers that put
**[AX.25](/reference/ax25/)** **[packet radio](/reference/packet-radio/)** on the air and
co-defined the **[KISS](/reference/kiss-tnc/)** host-to-TNC protocol.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A TAPR milestone timeline from the TNC-1 and TNC-2 controllers, through the KISS protocol, to later open hardware and software-defined radio projects." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="60" x2="430" y2="60" stroke="currentColor" stroke-width="1.2"/>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <circle cx="70" cy="60" r="4" fill="currentColor"/><text x="70" y="44">TNC-1 / TNC-2</text><text x="70" y="82" font-size="7.5">AX.25 on air</text>
    <circle cx="200" cy="60" r="4" fill="currentColor"/><text x="200" y="44">KISS</text><text x="200" y="82" font-size="7.5">host ↔ TNC</text>
    <circle cx="330" cy="60" r="4" fill="currentColor"/><text x="330" y="44">open hardware</text><text x="330" y="82" font-size="7.5">GPSDO · SDR</text>
    <text x="405" y="52" font-size="9">→</text>
  </g>
</svg>
<figcaption>From the first packet TNCs to open hardware, TAPR has driven amateur digital-radio experimentation for four decades.</figcaption>
</figure>

## Overview

TAPR was founded in 1982 in Tucson, Arizona, and quickly became the focal point of the amateur
packet-radio movement. Its defining early achievement was the **terminal node controller**
(TNC) — the modem-and-protocol box that sits between a radio and a computer. The
**TAPR TNC-1** and, especially, the widely cloned **TNC-2** made **[packet
radio](/reference/packet-radio/)** affordable and interoperable, letting hams exchange data
over VHF using the **[AX.25](/reference/ax25/)** link-layer protocol. TAPR later helped define
**[KISS](/reference/kiss-tnc/)** ("Keep It Simple, Stupid"), the minimal framing protocol that
lets a host computer drive a TNC directly, moving the protocol intelligence into software — a
design that endures in modern soundcard and SDR-based packet setups.

Beyond packet, TAPR has long championed **open hardware** and experimentation. Over the years
its projects and kits have spanned GPS-disciplined oscillators for precise frequency and
timing references, digital signal processing, and software-defined radio, and it developed an
early open-hardware licence to keep such designs freely shareable. TAPR also co-hosts, with the
**[ARRL](/reference/arrl/)**, the annual **Digital Communications Conference (DCC)**, a long-
running venue for papers and demonstrations on amateur digital modes. That community and its
tooling fed directly into modern systems such as **[APRS](/reference/aprs/)** (the Automatic
Packet Reporting System), which rides on the same AX.25 packet foundation TAPR helped
establish.

## Relevance to SDR

TAPR's legacy is everywhere a hobbyist decodes amateur data. The AX.25 framing and KISS
interface it helped standardise are exactly what a modern SDR-plus-software packet station
speaks: a soundcard modem or SDR replaces the old hardware TNC, but the on-air protocol and the
KISS link to the host are unchanged, which is why decades-old packet gear and new SDR setups
interoperate. APRS position and telemetry beacons, widely received with cheap SDRs, are built
on that same lineage. TAPR's open-hardware ethos also seeded much of the culture — shared
designs, GPSDO references, and community conferences — that the wider SDR world now takes for
granted.

Amateur packet sits outside GopherTrunk's land-mobile trunking focus, so GopherTrunk does not
itself decode AX.25 or APRS; dedicated tools handle those. The reference stands as context for
the wider RF landscape an SDR user explores, and it credits the organisation whose early work
made amateur digital radio open and interoperable in the first place.

## Sources

[^home]: [Tucson Amateur Packet Radio](https://tapr.org/) — TAPR's official site, for its projects, kits, open-hardware work, and the DCC.
[^wiki]: [Tucson Amateur Packet Radio](https://en.wikipedia.org/wiki/Tucson_Amateur_Packet_Radio) — Wikipedia, for TAPR's history, the TNC-2, and its role in packet radio.
