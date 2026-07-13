---
slug: etsi
title: ETSI
entry_type: organization
category: organizations
description: ETSI, the European Telecommunications Standards Institute, is the standards body behind DMR, dPMR, and TETRA, among many telecommunications standards.
keywords: ETSI, European Telecommunications Standards Institute, DMR, dPMR, TETRA, GSM, 3GPP, standards, Sophia Antipolis
aka: [ETSI, European Telecommunications Standards Institute]
autolink: true
infobox:
  - { label: Type, value: Standards organization }
  - { label: Region, value: Europe (global influence) }
  - { label: Standards, value: DMR, dPMR, TETRA, GSM }
see_also: [dmr, dpmr, tetra, 3gpp, cept, dmr-association]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
cite_urls:
  - https://www.etsi.org/
  - https://en.wikipedia.org/wiki/ETSI
---

**ETSI** (the **European Telecommunications Standards Institute**) is an independent,
not-for-profit standards organization whose work is used worldwide.[^home] In land-mobile radio
it authored [DMR](/reference/dmr/), [dPMR](/reference/dpmr/), and
[TETRA](/reference/tetra/), and it was the original home of the GSM cellular standard.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="ETSI publishes open telecommunications standards that many vendors implement into interoperable radio systems." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="42" width="100" height="32" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="70" y="61">ETSI</text>
    <rect x="170" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="225" y="61">open standard</text>
    <rect x="330" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="385" y="55">DMR · TETRA</text><text x="385" y="67">dPMR · GSM</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="120" y1="58" x2="169" y2="58" marker-end="url(#rel_etsi)"/><line x1="280" y1="58" x2="329" y2="58" marker-end="url(#rel_etsi)"/></g>
    <text x="230" y="96" font-size="8" fill="currentColor" fill-opacity="0.85">many vendors → interoperable, low-cost equipment</text>
  </g>
  <defs><marker id="rel_etsi" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>ETSI is the European standards body behind DMR, TETRA, dPMR, and (originally) GSM; open specs let many vendors build interoperable gear.</figcaption>
</figure>

## Overview

Founded in 1988 and based in Sophia Antipolis, France, ETSI is officially recognised by the
European Union as a European Standards Organisation, but its membership and influence are
global — hundreds of manufacturers, network operators, regulators, and research bodies from
dozens of countries take part. ETSI produces standards across the whole telecommunications
field: fixed and mobile networks, broadcasting, the Internet, and land-mobile radio. It has a
history of spinning off or co-founding partnerships for its biggest work: GSM began as an ETSI
project before the cellular effort moved into the **[3GPP](/reference/3gpp/)** partnership that
ETSI co-founded and still hosts, carrying the line through UMTS, [LTE](/reference/lte/), and
[5G NR](/reference/5g-nr/).

In the professional and amateur land-mobile world, ETSI's most visible standards are the ones
GopherTrunk cares about. **[DMR](/reference/dmr/)** (Digital Mobile Radio, ETSI TS 102 361) is
a two-slot [TDMA](/reference/tdma/) system spanning simple licensed-channel use
([Tier 1](/reference/dmr-tier-1/) and [Tier 2](/reference/dmr-tier-2/)) up to full
[trunking](/reference/dmr-tier-3/); its openness and 12.5 kHz efficiency made it the dominant
low-cost digital land-mobile standard, with day-to-day interoperability testing coordinated by
the industry [DMR Association](/reference/dmr-association/). **[dPMR](/reference/dpmr/)** is an
[FDMA](/reference/fdma/) 6.25 kHz alternative, and **[TETRA](/reference/tetra/)** (Terrestrial
Trunked Radio) is the four-slot TDMA standard used by many European public-safety and transport
networks. ETSI standards are developed by consensus among members and then published openly,
which — unlike a single-vendor design — lets many manufacturers build compatible radios.
Spectrum and licensing conditions for these systems in Europe are coordinated through
[CEPT](/reference/cept/) and administered by national regulators such as
[Ofcom](/reference/ofcom/), all within the [ITU](/reference/itu/) framework.

## Relevance to SDR

Several of the digital systems GopherTrunk decodes are ETSI standards, and it is precisely
their openness that makes third-party decoders feasible. Because DMR, dPMR, and TETRA are
published specifications rather than proprietary secrets, a decoder author can read exactly how
the [frame synchronisation](/reference/frame-synchronization/), slot structure,
[FEC](/reference/forward-error-correction/), and signalling bursts are laid out and implement
them faithfully. That is a sharp contrast with fully proprietary systems, where the air
interface must be reverse-engineered. The one part an open standard cannot make free is the
[vocoder](/reference/vocoder/): DMR uses [AMBE+2](/reference/ambe-plus-2/) from
[DVSI](/reference/dvsi/) and TETRA uses ACELP, both patent-encumbered, which is a separate
licensing question from the ETSI air-interface spec itself.

For an SDR user, ETSI's role explains a lot about what you hear on the band. The proliferation
of cheap DMR handhelds, the four-slot structure you see when you decode TETRA, and the
Europe-first flavour of these systems all trace back to ETSI's consensus process and 12.5 kHz
efficiency goals. GopherTrunk implements the ETSI air interfaces for the systems it supports;
consult its status page for exactly which tiers and features are decoded, and see the
[digital protocol landscape](/learn/rf-sdr/protocol-landscape/) lesson for how these standards
sit alongside the North American [P25](/reference/project-25/) family.

## Sources

[^home]: [ETSI](https://www.etsi.org/) — the institute's official site, where its telecommunications standards (including DMR, dPMR, and TETRA) are published.
[^wiki]: [ETSI](https://en.wikipedia.org/wiki/ETSI) — Wikipedia, for ETSI's history, its role in GSM and 3GPP, and its DMR, dPMR, and TETRA standards.
