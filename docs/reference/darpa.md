---
slug: darpa
title: DARPA (Defense Advanced Research Projects Agency)
entry_type: organization
category: organizations
description: "DARPA is the US Defense Department's advanced research agency, whose funded work seeded spread-spectrum radio, GPS, packet networking, and the early internet."
keywords: DARPA, Defense Advanced Research Projects Agency, ARPA, spread spectrum, GPS, packet radio, ARPANET, defense research
aka: [DARPA, ARPA, Defense Advanced Research Projects Agency]
autolink: true
infobox:
  - { label: Type, value: US government R&D agency }
  - { label: Founded, value: "1958" }
  - { label: Parent, value: US Department of Defense }
see_also: [direct-sequence-spread-spectrum, frequency-hopping-spread-spectrum, gps-gnss, packet-radio, internet-of-things]
cite_urls:
  - https://www.darpa.mil/
  - https://en.wikipedia.org/wiki/DARPA
---

**DARPA** (the **Defense Advanced Research Projects Agency**) is the research and
development arm of the United States Department of Defense, chartered to fund high-risk,
high-payoff technology that may not have an immediate military customer.[^home] Created in
1958, it is best known in radio for seeding foundational work on spread-spectrum
communications, satellite navigation, and packet networking — the last of which grew into
the internet.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 118" role="img" aria-label="DARPA-funded research seeded spread spectrum, GPS, and packet radio, which matured into widely used civilian technologies." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="darpa_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <rect x="20" y="44" width="96" height="34" rx="5" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.2"/><text x="68" y="65">DARPA</text>
    <rect x="200" y="8" width="150" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="275" y="24">Spread spectrum</text>
    <rect x="200" y="49" width="150" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="275" y="65">GPS / navigation</text>
    <rect x="200" y="90" width="150" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="275" y="105">Packet radio → internet</text>
    <g stroke="currentColor" stroke-width="1"><line x1="116" y1="55" x2="198" y2="22" marker-end="url(#darpa_ar)"/><line x1="116" y1="61" x2="198" y2="61" marker-end="url(#darpa_ar)"/><line x1="116" y1="67" x2="198" y2="100" marker-end="url(#darpa_ar)"/></g>
  </g>
</svg>
<figcaption>DARPA-funded programs seeded spread spectrum, GPS, and packet radio, all now everyday technologies.</figcaption>
</figure>

## Overview

DARPA was established in 1958 as ARPA, the Advanced Research Projects Agency, in the wake of
the Soviet launch of Sputnik, with a mandate to keep the US ahead in strategic technology
and to prevent technological surprise. It has been renamed between ARPA and DARPA several
times over its history. Its operating model is distinctive: a lean agency of temporary
program managers, each running focused programs for a few years and funding work at
universities, national labs, and industry rather than performing research in-house. This
model has produced an outsized record of breakthroughs well beyond radio, including stealth
aircraft, autonomous vehicles (via the DARPA Grand Challenges), and early machine-learning
and materials work.

For radio specifically, DARPA-supported programs advanced anti-jam and low-probability-of-
intercept communications built on [spread-spectrum](/reference/direct-sequence-spread-spectrum/)
techniques, funded early satellite-navigation research that fed into the military GPS program,
and — through the ARPANET and the **packet radio** projects of the 1970s — pioneered the idea
of routing digital packets over shared wireless links, a direct ancestor of both the internet
and modern mobile data.

## Relevance to SDR

Several technologies at the heart of software-defined radio trace part of their lineage to
DARPA-funded research. [Direct-sequence](/reference/direct-sequence-spread-spectrum/) and
[frequency-hopping](/reference/frequency-hopping-spread-spectrum/) spread spectrum — the
basis of GPS ranging codes, CDMA cellular, and much military communications — were matured
in the defense research environment DARPA helped drive. [GPS](/reference/gps-gnss/) itself,
now the reference clock and position source behind countless SDR applications, grew from
that same strategic-navigation lineage. And the packet-radio work prefigured the
software-defined, networked radios that are commonplace today.

GopherTrunk implements none of DARPA's programs directly — it is a receive-only
land-mobile scanner — but it lives downstream of these ideas: it relies on spread-spectrum-
derived timing when disciplined to GPS, and it processes the digital land-mobile protocols
that inherited packet-oriented signalling. DARPA is included here as the origin point for
several RF concepts the rest of this guide treats as building blocks.

## Sources

[^home]: [DARPA](https://www.darpa.mil/) — the agency's official site, for its mission, program model, and research areas.
[^wiki]: [DARPA](https://en.wikipedia.org/wiki/DARPA) — Wikipedia, for the agency's history and its role in spread spectrum, GPS, and packet networking.
