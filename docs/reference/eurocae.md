---
slug: eurocae
title: EUROCAE
entry_type: organization
category: organizations
description: EUROCAE is the European standards body for aviation electronics; its ED-102 series mirrors RTCA's DO-260 ADS-B requirements for use in Europe.
keywords: EUROCAE, ED-102, ADS-B standards, Europe, aviation, avionics, MOPS, European Organisation for Civil Aviation Equipment
aka: [EUROCAE]
autolink: true
infobox:
  - { label: Type, value: Standards organization (Europe) }
  - { label: Focus, value: Aviation electronics }
  - { label: Standards, value: ED-102 (ADS-B) }
see_also: [ads-b, mode-s, rtca, icao, compact-position-reporting, itu]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/rf-sdr/other-signals/ }
cite_urls:
  - https://www.eurocae.net/
  - https://en.wikipedia.org/wiki/EUROCAE
---

**EUROCAE** (the European Organisation for Civil Aviation Equipment) is the European
standards body for **aviation electronics**. Its **ED-102** series mirrors
[RTCA](/reference/rtca/)'s DO-260 [ADS-B](/reference/ads-b/) requirements for use in
Europe, under the umbrella of [ICAO](/reference/icao/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="EUROCAE publishing the ED-102 standard, the European counterpart to RTCA's DO-260 for ADS-B, harmonised for interoperability." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="42" width="110" height="32" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="75" y="55">EUROCAE</text><text x="75" y="67" font-size="7.5">Europe</text>
    <rect x="175" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="230" y="55">ED-102A ≈</text><text x="230" y="67" font-size="7.5">RTCA DO-260B</text>
    <rect x="330" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="385" y="55">ADS-B (Europe)</text><text x="385" y="67" font-size="7.5">1090 MHz ES</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="130" y1="58" x2="174" y2="58" marker-end="url(#or_euc)"/><line x1="285" y1="58" x2="329" y2="58" marker-end="url(#or_euc)"/></g>
    <text x="230" y="96" font-size="8" fill="currentColor" fill-opacity="0.85">harmonised with US MOPS → one decoder works on both</text>
  </g>
  <defs><marker id="or_euc" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>EUROCAE's ED-102 is the European counterpart to RTCA's DO-260 for ADS-B, deliberately harmonised so avionics interoperate.</figcaption>
</figure>

## Overview

EUROCAE, founded in 1963 and based in the Paris area, is a non-profit membership organisation
that develops technical standards for civil-aviation equipment in Europe. It plays the same role
on the European side that [RTCA](/reference/rtca/) plays in the United States, and the two work
in deliberate partnership: many of their documents are developed jointly so that avionics
certified to one apply to the other. Its standards are published as **ED-** documents, and,
like RTCA's DO- series, they are frequently referenced by the certification authorities — in
Europe, the European Union Aviation Safety Agency (EASA) — giving them regulatory weight.

For surveillance, EUROCAE's key document is **ED-102**, the European Minimum Operational
Performance Standards for 1090 MHz Extended Squitter [ADS-B](/reference/ads-b/) riding on
[Mode S](/reference/mode-s/). **ED-102A** is harmonised with RTCA's **DO-260B**: the two describe
the same message formats, the same position encoding using
[Compact Position Reporting](/reference/compact-position-reporting/), and the same integrity
reporting, so an aircraft equipped to one standard is understood by ground systems built to the
other. That harmonisation is not an accident but the whole point — international aviation
requires that a transponder work identically over Europe and North America. The high-level
requirements both bodies elaborate come from [ICAO](/reference/icao/)'s Annex 10, and the radio
spectrum they occupy is allocated internationally by the [ITU](/reference/itu/).

## Relevance to SDR

For someone decoding aircraft signals, EUROCAE matters because ADS-B documentation on both
continents often cites *both* ED-102A and DO-260B, and understanding that they are effectively
the same specification removes confusion. A decoder built to DO-260B will correctly parse the
Extended Squitter messages from a European aircraft certified to ED-102A, because the two are
harmonised down to the bit layout — the position, velocity, and identity fields are identical.
In practice this means the SDR aircraft-tracking ecosystem is genuinely global: the same
open-source decoder and the same crowdsourced networks work over Frankfurt and over Chicago.

Aircraft surveillance falls outside GopherTrunk's land-mobile trunking focus, so GopherTrunk
does not decode ADS-B; that is the domain of dedicated 1090 MHz tools. EUROCAE is included here
as the European half of the standards story that makes those signals uniform worldwide. See the
[other signals you'll meet](/learn/rf-sdr/other-signals/) lesson for where ADS-B fits among the
non-trunking signals an SDR can receive.

## Sources

[^home]: [EUROCAE](https://www.eurocae.net/) — EUROCAE's official site, the European developer of the ED-102 ADS-B standards, harmonised with RTCA's DO-260.
[^wiki]: [EUROCAE](https://en.wikipedia.org/wiki/EUROCAE) — Wikipedia, on EUROCAE and its ED-102 ADS-B standards, the European counterpart to RTCA's DO-260.
