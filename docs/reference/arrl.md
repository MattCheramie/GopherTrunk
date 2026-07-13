---
slug: arrl
title: ARRL (American Radio Relay League)
entry_type: organization
category: organizations
description: "The ARRL is the national association for amateur radio in the United States, representing hams before the FCC and publishing standards and operating resources."
keywords: ARRL, American Radio Relay League, amateur radio, ham radio, national association, IARU, band plans, QST
aka: [ARRL, American Radio Relay League]
autolink: true
infobox:
  - { label: Type, value: National membership association }
  - { label: Founded, value: "1914" }
  - { label: Region, value: United States }
see_also: [ft8, iaru, rsgb, wspr, js8call]
cite_urls:
  - https://www.arrl.org/
  - https://en.wikipedia.org/wiki/American_Radio_Relay_League
---

**The ARRL** (the **American Radio Relay League**) is the national membership association
for amateur ("ham") radio operators in the United States, representing their interests
before the [FCC](/reference/fcc/) and internationally, and serving as a hub for training,
standards, and operating resources.[^home] Founded in 1914, it is one of the oldest
continuously operating radio organizations in the world and the US member society of the
[IARU](/reference/iaru/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="The ARRL represents US amateur operators before the FCC and coordinates internationally through the IARU." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="arrl_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <rect x="180" y="42" width="100" height="34" rx="5" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.2"/><text x="230" y="63">ARRL</text>
    <rect x="20" y="44" width="110" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1"/><text x="75" y="60">US amateurs</text><text x="75" y="70" font-size="7.5">members</text>
    <rect x="330" y="10" width="110" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="385" y="27">FCC (regulator)</text>
    <rect x="330" y="72" width="110" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="385" y="89">IARU (global)</text>
    <g stroke="currentColor" stroke-width="1"><line x1="130" y1="59" x2="178" y2="59" marker-end="url(#arrl_ar)"/><line x1="280" y1="52" x2="329" y2="30" marker-end="url(#arrl_ar)"/><line x1="280" y1="66" x2="329" y2="82" marker-end="url(#arrl_ar)"/></g>
  </g>
</svg>
<figcaption>The ARRL represents its US amateur members to the FCC and to the world through the IARU.</figcaption>
</figure>

## Overview

The ARRL was created in 1914 by Hiram Percy Maxim to organize the relay of messages across
the country by a network of amateur stations — the "relay" in its name — at a time when a
single station could not span the continent. As amateur radio matured, the league's mission
broadened into advocacy, education, emergency communications, and technical publishing. It
is a non-profit headquartered in Newington, Connecticut, governed by an elected board and
supported by its members.

Much of what the ARRL does touches the technical fabric of amateur radio. It publishes the
long-running *QST* magazine and reference works such as the *ARRL Handbook* and *ARRL
Antenna Book*, maintains the widely followed US **band plans** that partition each amateur
allocation into segments for different modes, operates the W1AW headquarters station,
sponsors contests and awards, and runs the Amateur Radio Emergency Service (ARES). Through
the IARU it participates in the international coordination of amateur spectrum ahead of
World Radiocommunication Conferences. It also administers the US Volunteer Examiner
Coordinator program that many operators pass through to earn their licenses.

## Relevance to SDR

Amateur radio and software-defined radio are deeply intertwined, and the ARRL sits at the
center of the US amateur community that drives much SDR experimentation. The modern digital
modes that dominate the HF bands — [FT8](/reference/ft8/), [WSPR](/reference/wspr/), and
[JS8Call](/reference/js8call/) — are typically run through an SDR or a sound-card interface,
and the band-plan segments the ARRL coordinates are where an operator points a receiver to
find them. The league's publications are a standard entry point for newcomers learning
propagation, antennas, and modulation, all of which underpin SDR reception.

GopherTrunk is not an amateur-radio application and the ARRL does not author any protocol it
decodes; the connection is contextual rather than technical. That said, the amateur bands
are a rich playground for the same DSP GopherTrunk uses elsewhere, and an SDR tuned to an
amateur allocation coordinated under ARRL band plans is one of the most common ways people
first encounter the demodulation and decoding ideas this guide describes.

## Sources

[^home]: [ARRL](https://www.arrl.org/) — the league's official site, for US amateur band plans, licensing resources, and publications.
[^wiki]: [American Radio Relay League](https://en.wikipedia.org/wiki/American_Radio_Relay_League) — Wikipedia, for the ARRL's history, structure, and role in amateur radio.
