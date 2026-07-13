---
slug: dmr-association
title: DMR Association
entry_type: organization
category: organizations
description: "The DMR Association is the industry group that promotes the ETSI DMR digital radio standard and certifies interoperability between different manufacturers' equipment."
keywords: DMR Association, Digital Mobile Radio, interoperability, ETSI DMR, IOP certification, land mobile radio, two-way radio
aka: [DMR Association]
autolink: true
infobox:
  - { label: Type, value: Industry trade association }
  - { label: Focus, value: DMR interoperability and promotion }
  - { label: Standard, value: "ETSI TS 102 361 (DMR)" }
see_also: [dmr, dmr-tier-2, dmr-tier-1, dmr-tier-3, etsi]
cite_urls:
  - https://www.dmrassociation.org/
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
---

The **DMR Association** is the **industry body that promotes the Digital Mobile Radio
standard and certifies interoperability between different vendors' equipment**.[^wiki] Its
central purpose is to make sure that [DMR](/reference/dmr/) radios and infrastructure built by
competing manufacturers actually work together, by running a formal interoperability
(IOP) testing and certification programme against the published ETSI DMR standard.[^home]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 116" role="img" aria-label="The DMR Association certifying interoperability between DMR equipment from different manufacturers against the ETSI standard." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="165" y="10" width="130" height="28" rx="5" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.2"/><text x="230" y="28">DMR Association</text>
    <text x="230" y="52" font-size="8">IOP certification vs ETSI standard</text>
    <rect x="30" y="70" width="100" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="80" y="89">Vendor A radio</text>
    <rect x="180" y="70" width="100" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="230" y="89">Vendor B radio</text>
    <rect x="330" y="70" width="100" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="380" y="89">Vendor C infra</text>
    <g stroke="currentColor" stroke-width="1.1">
      <line x1="130" y1="85" x2="179" y2="85" marker-end="url(#ar_dmra)"/>
      <line x1="280" y1="85" x2="329" y2="85" marker-end="url(#ar_dmra)"/>
      <line x1="230" y1="38" x2="230" y2="69" marker-end="url(#ar_dmra)"/>
    </g>
  </g>
  <defs><marker id="ar_dmra" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The DMR Association's interoperability certification lets radios and infrastructure from different vendors work together.</figcaption>
</figure>

## Overview

DMR is an open digital land-mobile-radio standard defined by [ETSI](/reference/etsi/) (the
TS 102 361 series). Because it is an open standard rather than one vendor's proprietary
system, many manufacturers build DMR equipment — and the risk with any multi-vendor standard
is that small differences in interpretation prevent one company's radios from working on
another's system. The DMR Association exists to close that gap. It is a membership group of
manufacturers and other stakeholders that maintains an **interoperability process**: vendors
submit their equipment for testing against defined feature profiles, and products that pass
are listed as IOP-certified, giving buyers confidence they can mix equipment.

The association also acts as the standard's public advocate — explaining the DMR
[tiers](/reference/dmr-tier-2/), publishing guidance, and promoting DMR against competing
digital land-mobile technologies. It is careful to distinguish its promotional and
certification role from the standards-writing role, which belongs to ETSI: ETSI publishes the
normative specification, and the DMR Association builds the conformance and interoperability
programme on top of it. Its work centres on the conventional and trunked professional tiers
of DMR rather than the amateur community's separate, informally coordinated networks.

## Relevance to SDR

DMR is one of the most widely deployed digital voice modes an SDR user will encounter, used
across business, utility, and public-service fleets worldwide. The interoperability the DMR
Association certifies is part of why DMR looks consistent on the air: the two-slot
[TDMA](/reference/dmr-tier-2/) structure, the 12.5 kHz channel with a 4-level FSK modulation,
and the burst framing behave the same regardless of which vendor built the transmitter, which
is exactly what makes DMR practical to decode with a general receiver.

GopherTrunk decodes DMR — its clear (unencrypted) traffic across the conventional and Tier
II/III trunked tiers is squarely within scope — so the standard the DMR Association promotes
is directly relevant to what GopherTrunk does. The association itself writes no code and sets
no signal-processing rules; its value to a decoder author is that its interoperability regime
keeps real-world DMR signals close to the ETSI specification GopherTrunk implements against.
As always, GopherTrunk decodes clear traffic, not encrypted transmissions.

## Sources

[^home]: [DMR Association](https://www.dmrassociation.org/) — the association's official site, for its interoperability certification programme and DMR tier guidance.
[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, for the DMR standard and the association's promotional and interoperability role.
