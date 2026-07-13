---
slug: wrc
title: World Radiocommunication Conference (WRC)
entry_type: organization
category: organizations
description: "The WRC is the ITU treaty conference, held every few years, that revises the international Radio Regulations governing how the radio spectrum is allocated worldwide."
keywords: WRC, World Radiocommunication Conference, Radio Regulations, ITU, spectrum allocation, treaty, WARC, agenda item
aka: [WRC, World Radiocommunication Conference, WARC]
autolink: true
infobox:
  - { label: Type, value: ITU treaty conference }
  - { label: Frequency, value: Every 3–4 years }
  - { label: Outcome, value: Revised Radio Regulations }
see_also: [itu-r, itu, frequency-bands, iaru, fcc]
cite_urls:
  - https://www.itu.int/en/ITU-R/conferences/wrc/
  - https://en.wikipedia.org/wiki/World_Radiocommunication_Conference
---

The **World Radiocommunication Conference** (**WRC**) is the treaty-making conference of the
[ITU](/reference/itu/) at which the world's governments **revise the international Radio
Regulations that govern how the radio [spectrum](/reference/frequency-bands/) is allocated**.[^wiki]
Convened by the [ITU-R](/reference/itu-r/) sector roughly every three to four years, a WRC
brings together delegations from ITU member states to negotiate changes to frequency
allocations and the technical rules that go with them; its decisions carry the force of an
international treaty.[^home]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 112" role="img" aria-label="A World Radiocommunication Conference cycle: agenda items are studied, negotiated at the conference, and adopted as revised Radio Regulations." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="18" y="46" width="110" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="73" y="65">agenda items</text>
    <rect x="175" y="46" width="110" height="30" rx="5" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.2"/><text x="230" y="65">WRC</text>
    <rect x="332" y="46" width="110" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="387" y="61">revised Radio</text><text x="387" y="72" font-size="8">Regulations</text>
    <g stroke="currentColor" stroke-width="1.1">
      <line x1="128" y1="61" x2="174" y2="61" marker-end="url(#ar_wrc)"/>
      <line x1="285" y1="61" x2="331" y2="61" marker-end="url(#ar_wrc)"/>
    </g>
    <text x="230" y="26" font-size="8.5">study cycle → conference → treaty</text>
  </g>
  <defs><marker id="ar_wrc" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Each WRC turns years of technical study on set agenda items into a revised Radio Regulations treaty.</figcaption>
</figure>

## Overview

A WRC does not start from a blank page. Each conference works to a fixed **agenda** of items
agreed at the previous one, so the intervening years are spent on preparatory technical study
inside the [ITU-R](/reference/itu-r/) study groups. Regional groups — the Americas, Europe,
Africa, the Arab states, and Asia-Pacific — coordinate common positions in advance, and
national administrations bring negotiating briefs shaped by their regulators and industries.
At the conference itself, delegates work through the agenda in committees, resolve competing
demands for the same bands, and adopt the results as revisions to the Radio Regulations, the
binding treaty that then governs spectrum use until the next WRC.

The conference has a long lineage. Earlier meetings were called World Administrative Radio
Conferences (WARCs) — the 1979 WARC, for instance, created the amateur "WARC bands" still in
use today — before the series was renamed WRC in the 1990s reorganisation of the ITU. Typical
modern agenda items include finding new spectrum for mobile broadband, adjusting satellite and
Earth-observation allocations, and protecting incumbent services such as aeronautical and
radioastronomy from new interference.

## Relevance to SDR

The WRC is where the map of the spectrum is redrawn, so its outcomes eventually reach every
SDR listener. A new mobile allocation, a reshuffled satellite band, or a change to an amateur
segment all begin as WRC decisions before national regulators like the [FCC](/reference/fcc/)
implement them domestically. Advocacy groups such as the [IARU](/reference/iaru/) attend
specifically to defend allocations that hobbyists care about, which is why following WRC
agendas gives a preview of how the bands you scan may shift over the coming years.

A WRC has nothing to do with a receiver's signal processing, so GopherTrunk implements none
of its provisions. Its relevance is entirely contextual: the treaty it produces is the
top-level reason a given service occupies a given band worldwide, and therefore why a
particular band plan is worth decoding at all.

## Sources

[^home]: [World Radiocommunication Conferences](https://www.itu.int/en/ITU-R/conferences/wrc/) — the ITU's official WRC page, for conference agendas, outcomes, and the Radio Regulations.
[^wiki]: [World Radiocommunication Conference](https://en.wikipedia.org/wiki/World_Radiocommunication_Conference) — Wikipedia, for the history of the conference series and its treaty role.
