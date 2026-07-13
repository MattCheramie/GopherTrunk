---
slug: iaru
title: International Amateur Radio Union (IARU)
entry_type: organization
category: organizations
description: "The IARU is the global federation of national amateur radio societies that defends amateur spectrum allocations at the ITU and coordinates band plans worldwide."
keywords: IARU, International Amateur Radio Union, amateur radio, ham radio, band plan, spectrum defence, amateur allocations
aka: [IARU, International Amateur Radio Union]
autolink: true
infobox:
  - { label: Type, value: Federation of national societies }
  - { label: Founded, value: "1925" }
  - { label: Role, value: Amateur spectrum advocacy, band plans }
see_also: [arrl, rsgb, itu, itu-r, wrc, frequency-bands]
cite_urls:
  - https://www.iaru.org/
  - https://en.wikipedia.org/wiki/International_Amateur_Radio_Union
---

The **International Amateur Radio Union** (**IARU**) is the **worldwide federation of
national amateur radio societies**, formed in 1925 to represent the interests of licensed
amateurs in international spectrum matters.[^wiki] Its central mission is to defend and
expand the amateur and amateur-satellite service allocations in the radio
[spectrum](/reference/frequency-bands/) and to coordinate voluntary band plans so that
amateurs around the world use their shared frequencies compatibly.[^home]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="National amateur radio societies federating into the IARU, which represents them at the ITU." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="14" width="90" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="65" y="31">ARRL</text>
    <rect x="130" y="14" width="90" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="175" y="31">RSGB</text>
    <rect x="240" y="14" width="90" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="285" y="31">other societies</text>
    <rect x="175" y="58" width="110" height="28" rx="5" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.2"/><text x="230" y="76">IARU</text>
    <rect x="175" y="100" width="110" height="14" rx="3" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="230" y="110" font-size="8">ITU / WRC</text>
    <g stroke="currentColor" stroke-width="1.1">
      <line x1="65" y1="40" x2="215" y2="58" marker-end="url(#ar_iaru)"/>
      <line x1="175" y1="40" x2="228" y2="57" marker-end="url(#ar_iaru)"/>
      <line x1="285" y1="40" x2="245" y2="58" marker-end="url(#ar_iaru)"/>
      <line x1="230" y1="86" x2="230" y2="99" marker-end="url(#ar_iaru)"/>
    </g>
  </g>
  <defs><marker id="ar_iaru" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>National societies federate into the IARU, which represents amateur radio at the ITU and its conferences.</figcaption>
</figure>

## Overview

The IARU is organised around the three [ITU](/reference/itu/) radio regions: Region 1
(Europe, Africa, the Middle East, and northern Asia), Region 2 (the Americas), and Region 3
(Asia-Pacific). Member societies — the [ARRL](/reference/arrl/) in the United States, the
[RSGB](/reference/rsgb/) in the United Kingdom, and roughly 160 others — join through their
regional organisation, and the IARU maintains a small international secretariat and elected
officers to coordinate policy. It is a non-governmental body: it holds no regulatory power
of its own, but it is the recognised voice of the amateur service in the international arena.

Its most visible ongoing work is spectrum defence. Amateur bands sit alongside commercial,
military, and scientific users who periodically seek to reallocate them, so the IARU sends
observers and technical experts to prepare for and attend the treaty conferences where those
decisions are made. It also publishes region-wide band plans — voluntary agreements dividing
each amateur band into segments for CW, digital modes, SSB, and so on — and runs practical
services such as the international beacon network and coordination of amateur satellites.

## Relevance to SDR

Many software-defined-radio hobbyists are also licensed amateurs, and the amateur bands are
among the most rewarding to explore with an SDR: they carry a huge diversity of modes, from
CW and SSB voice to FT8, packet, and amateur digital voice. The reason those bands exist and
remain available is in large part the IARU's advocacy at the [ITU-R](/reference/itu-r/)
sector and at each [World Radiocommunication Conference](/reference/wrc/), where amateur
allocations are periodically re-examined.

For a receiving tool like GopherTrunk, the IARU is context rather than a decoded system:
its band plans tell you which slices of an amateur band to expect a given signal in, and its
allocations are why amateur segments recur in the same places across regions. GopherTrunk
focuses on land-mobile trunking rather than amateur weak-signal modes, but the same tuning
principles apply — knowing the band plan tells you where to look.

## Sources

[^home]: [IARU](https://www.iaru.org/) — the union's official site, for its regional structure, band plans, and spectrum-defence activities.
[^wiki]: [International Amateur Radio Union](https://en.wikipedia.org/wiki/International_Amateur_Radio_Union) — Wikipedia, for the IARU's history, membership, and role in international amateur spectrum matters.
