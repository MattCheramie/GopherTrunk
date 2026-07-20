---
slug: nrsc
title: National Radio Systems Committee (NRSC)
entry_type: organization
category: organizations
description: The NRSC, the National Radio Systems Committee, is a joint NAB/CTA body that sets US terrestrial broadcast standards, including NRSC-5 for HD Radio and the RBDS variant of RDS.
keywords: NRSC, National Radio Systems Committee, NRSC-5, HD Radio, IBOC, NRSC-4, RBDS, RDS, NAB, CTA, broadcast standards
aka: [NRSC, National Radio Systems Committee]
autolink: true
infobox:
  - { label: Type, value: US broadcast standards committee }
  - { label: Focus, value: Terrestrial AM/FM broadcasting }
  - { label: Sponsors, value: "NAB and CTA" }
  - { label: Standards, value: "NRSC-5, NRSC-4, RBDS" }
see_also: [hd-radio, rds, broadcast-am, broadcast-fm, fcc]
cite_urls:
  - https://www.nrscstandards.org/
  - https://en.wikipedia.org/wiki/National_Radio_Systems_Committee
---

**NRSC** (the **National Radio Systems Committee**) is a joint committee of the US National
Association of Broadcasters (NAB) and the Consumer Technology Association (CTA) that sets
standards for terrestrial radio broadcasting — including **[HD Radio](/reference/hd-radio/)**
and the **[RBDS](/reference/rds/)** data system on
**[FM](/reference/broadcast-fm/)** and **[AM](/reference/broadcast-am/)**.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="The NRSC, sponsored by NAB and CTA, publishes three broadcast standards: NRSC-5 HD Radio, NRSC-4 the AM emission mask, and RBDS." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="55" width="100" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="70" y="70">NRSC</text><text x="70" y="82" font-size="7.5">NAB + CTA</text>
    <rect x="300" y="12" width="150" height="28" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="375" y="30" font-size="8">NRSC-5 · HD Radio (IBOC)</text>
    <rect x="300" y="58" width="150" height="28" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="375" y="76" font-size="8">NRSC-4 · AM mask</text>
    <rect x="300" y="104" width="150" height="28" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="375" y="122" font-size="8">RBDS · US RDS</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="120" y1="66" x2="299" y2="26" marker-end="url(#rel_nrsc)"/><line x1="120" y1="72" x2="299" y2="72" marker-end="url(#rel_nrsc)"/><line x1="120" y1="78" x2="299" y2="118" marker-end="url(#rel_nrsc)"/></g>
  </g>
  <defs><marker id="rel_nrsc" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The NRSC's standards define US digital radio (NRSC-5), the AM emission mask (NRSC-4), and the RBDS data layer on FM and AM.</figcaption>
</figure>

## Overview

The NRSC is a technical standards-setting body jointly sponsored by the **National Association
of Broadcasters (NAB)** and the **Consumer Technology Association (CTA)**. It brings
broadcasters and equipment makers together to write voluntary standards that keep US
terrestrial AM and FM radio — and the receivers built for it — interoperable. Its standards are
frequently referenced by the [FCC](/reference/fcc/) in rulemaking, giving them practical force
even though the committee itself is an industry body rather than a regulator.

Three of its standards matter most to a radio listener. **NRSC-5** defines
**[HD Radio](/reference/hd-radio/)**, the **in-band on-channel (IBOC)** system that lets a
station broadcast digital audio and data in the sidebands of its existing analog AM or FM
channel, so a single assignment carries both analog and digital services. **NRSC-4** specifies
the **AM emission mask** — the limits on an AM station's occupied bandwidth and spectral
splatter that keep adjacent channels clean. And the committee standardised **RBDS** (the Radio
Broadcast Data System), the US variant of Europe's **[RDS](/reference/rds/)**, which carries
station name, program type, song text, and traffic flags in a low-rate subcarrier on
**[FM](/reference/broadcast-fm/)**. Together these define how US **[AM](/reference/broadcast-am/)**
and FM broadcasting delivers digital audio and data alongside the legacy analog signal.

## Relevance to SDR

The NRSC's standards describe signals an SDR user routinely encounters across the broadcast
band. The RBDS/RDS subcarrier on an FM station is openly decodable — many SDR programs display
the station name and now-playing text pulled straight from that data group — and the structure
they parse is the one the NRSC standardised. HD Radio's IBOC sidebands are visible on a
waterfall as the digital "shoulders" flanking an analog FM or AM carrier, and open decoders
exist that recover the NRSC-5 digital audio and program-service data. Knowing which standard
defines a given feature helps a listener interpret what appears on the spectrum.

Broadcast radio sits outside GopherTrunk's land-mobile trunking focus, so GopherTrunk does not
itself decode HD Radio or RBDS; dedicated tools handle those. The reference stands as context
for the wider RF landscape an SDR user explores, and it identifies the body that makes US AM
and FM digital and data services consistent from one station and receiver to the next.

## Sources

[^home]: [NRSC Standards](https://www.nrscstandards.org/) — the NRSC's official standards site, for NRSC-5, NRSC-4, and RBDS documents.
[^wiki]: [National Radio Systems Committee](https://en.wikipedia.org/wiki/National_Radio_Systems_Committee) — Wikipedia, for the committee's sponsors, history, and standards.
