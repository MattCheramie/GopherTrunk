---
slug: fcc
title: Federal Communications Commission (FCC)
entry_type: organization
category: organizations
description: The FCC is the United States communications regulator — allocating spectrum, licensing users, and setting the technical rules that shaped digital radio.
keywords: FCC, Federal Communications Commission, spectrum, licensing, narrowbanding, US regulator, Part 90, Part 15, Part 97
aka: [FCC, Federal Communications Commission]
autolink: true
infobox:
  - { label: Type, value: US government regulator }
  - { label: Founded, value: "1934" }
  - { label: Role, value: US spectrum allocation and licensing }
see_also: [itu, ntia, ofcom, tia, apco-international, arrl, frequency-bands]
related_lessons:
  - { title: "Frequency, bands & the spectrum", url: /learn/rf-sdr/frequency-and-spectrum/ }
  - { title: "Legal & ethical monitoring", url: /learn/rf-sdr/legal-ethical/ }
cite_urls:
  - https://www.fcc.gov/
  - https://en.wikipedia.org/wiki/Federal_Communications_Commission
---

The **Federal Communications Commission** (**FCC**) is the United States regulator of
interstate radio, television, wire, satellite, and cable.[^wiki] It allocates US spectrum,
licenses users, and sets the technical rules that non-federal transmitters must obey.[^home]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="The FCC dividing the US radio spectrum into allocated and licensed service blocks under the ITU framework." xmlns="http://www.w3.org/2000/svg">
  <text x="230" y="20" text-anchor="middle" font-size="9" fill="currentColor">US spectrum allocations &amp; licensing (Title 47 CFR)</text>
  <line x1="30" y1="72" x2="430" y2="72" stroke="currentColor" stroke-opacity="0.4"/>
  <g stroke="currentColor" stroke-width="1.1">
    <rect x="40" y="54" width="70" height="18" fill="currentColor" fill-opacity="0.12"/><rect x="110" y="54" width="60" height="18" fill="currentColor" fill-opacity="0.22"/><rect x="170" y="54" width="80" height="18" fill="currentColor" fill-opacity="0.12"/><rect x="250" y="54" width="60" height="18" fill="currentColor" fill-opacity="0.22"/><rect x="310" y="54" width="110" height="18" fill="currentColor" fill-opacity="0.12"/>
  </g>
  <g font-size="7.5" fill="currentColor" text-anchor="middle">
    <text x="75" y="66">Part 90</text><text x="140" y="66">Part 15</text><text x="210" y="66">Part 97</text><text x="280" y="66">Part 22</text><text x="365" y="66">Part 73</text>
  </g>
  <text x="230" y="98" text-anchor="middle" font-size="8.5" fill="currentColor">within ITU allocations · FCC (civilian) alongside NTIA (federal)</text>
  <text x="230" y="112" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.8">licenses public-safety, business, broadcast &amp; amateur users</text>
</svg>
<figcaption>The FCC allocates and licenses non-federal radio spectrum in the United States, within the ITU's global framework.</figcaption>
</figure>

## Overview

Created by the Communications Act of 1934, the FCC is an independent US government agency
overseen by Congress and run by five commissioners. It regulates all *non-federal* use of the
radio spectrum — commercial, public-safety, amateur, and consumer — while federal government
use (military, and much of aviation and law enforcement) is coordinated separately by the
[NTIA](/reference/ntia/). The two share the spectrum under a national plan that itself sits
inside the international allocations set by the [ITU](/reference/itu/). The FCC's technical
rules live in Title 47 of the Code of Federal Regulations, whose parts every SDR user
eventually meets: **Part 15** governs unlicensed devices (Wi-Fi, Bluetooth, ISM-band gadgets),
**Part 90** covers private land-mobile radio (the world of public-safety
[trunked-radio](/reference/trunked-radio/), business, and [P25](/reference/project-25/)
systems), **Part 97** covers the amateur service, and **Part 22/24** cover cellular and PCS.

Beyond allocating bands, the FCC certifies equipment, issues licences, and sets emission
masks and power limits. One of its most consequential land-mobile actions was the
**narrowbanding** mandate, which required VHF/UHF business and public-safety licensees to move
from 25 kHz to 12.5 kHz channel spacing (or equivalent efficiency). That efficiency pressure
was a strong tailwind for digital systems: 12.5 kHz [DMR](/reference/dmr/) and
[NXDN](/reference/nxdn/), and the 6.25 kHz-equivalent efficiency of [P25 Phase 2](/reference/p25-phase-2/)
[TDMA](/reference/tdma/), all fit the narrowband world the FCC pushed toward. The commission
works with US standards bodies and user groups — [TIA](/reference/tia/) writes the P25
standards, [APCO International](/reference/apco-international/) represents public-safety users,
and the [ARRL](/reference/arrl/) represents amateur operators — but the FCC itself sets the
binding rules.

## Relevance to SDR

Almost everything an SDR user in the United States tunes to is shaped by an FCC decision: the
channel plan, the modulation efficiency, the power level, and whether a band is licensed or
open. Knowing which Part governs a service tells you what to expect — a Part 90 public-safety
channel is likely narrowband digital voice, while a Part 15 device in the 902–928 MHz ISM band
might be a [LoRa](/reference/lora/) sensor or a cordless phone. The narrowbanding history
explains why so much of the modern land-mobile spectrum you scan is digital rather than analog
FM.

For reception specifically, the FCC's rules matter in a different way: in the US, the
Communications Act generally permits *listening* to most radio transmissions but restricts
*divulging or using* the contents of certain communications, and intercepting some services
(notably cellular) is explicitly barred. What you may lawfully receive and what you may do with
it varies by jurisdiction, so the FCC is the reference point for US users the way
[Ofcom](/reference/ofcom/) is for the UK. GopherTrunk is a receiver and decoder and implements
no FCC logic itself, but the systems it decodes exist in the shape the FCC's band plans and
narrowbanding rules gave them — see the [legal &amp; ethical monitoring](/learn/rf-sdr/legal-ethical/)
lesson before you tune.

## Sources

[^home]: [Federal Communications Commission](https://www.fcc.gov/) — the FCC's official site, for US spectrum allocation, licensing, equipment certification, and the Part rules.
[^wiki]: [Federal Communications Commission](https://en.wikipedia.org/wiki/Federal_Communications_Commission) — Wikipedia, for the agency's history, structure, and regulatory remit.
