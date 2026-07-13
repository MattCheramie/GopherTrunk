---
slug: rtca
title: RTCA
entry_type: organization
category: organizations
description: RTCA is a US standards body for aviation electronics; its DO-260 series defines the ADS-B performance requirements that transponders must meet.
keywords: RTCA, DO-260, DO-282, ADS-B standards, aviation, MOPS, avionics, UAT, Radio Technical Commission for Aeronautics
aka: [RTCA]
autolink: true
infobox:
  - { label: Type, value: Standards organization (US) }
  - { label: Focus, value: Aviation electronics }
  - { label: Standards, value: DO-260 (ADS-B), DO-282 (UAT) }
see_also: [ads-b, mode-s, icao, eurocae, uat-978, compact-position-reporting]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/rf-sdr/other-signals/ }
cite_urls:
  - https://www.rtca.org/
  - https://en.wikipedia.org/wiki/RTCA
---

**RTCA** is a United States standards organization for **aviation electronics**. Its
**DO-260** series of Minimum Operational Performance Standards (MOPS) defines the
[ADS-B](/reference/ads-b/) requirements that aircraft transponders must meet, alongside
the international standards from [ICAO](/reference/icao/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="RTCA publishing the DO-260 and DO-282 standards that turn ICAO requirements into testable transponder behaviour for ADS-B." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="42" width="110" height="32" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="75" y="55">RTCA</text><text x="75" y="67" font-size="7.5">US committees</text>
    <rect x="175" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="230" y="55">DO-260 (1090)</text><text x="230" y="67" font-size="7.5">DO-282 (UAT)</text>
    <rect x="330" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="385" y="55">ADS-B</text><text x="385" y="67" font-size="7.5">transponders</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="130" y1="58" x2="174" y2="58" marker-end="url(#or_rtc)"/><line x1="285" y1="58" x2="329" y2="58" marker-end="url(#or_rtc)"/></g>
    <text x="230" y="96" font-size="8" fill="currentColor" fill-opacity="0.85">harmonised with EUROCAE's ED-102 for use in Europe</text>
  </g>
  <defs><marker id="or_rtc" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>RTCA's DO-260 (and DO-282 for UAT) MOPS define how ADS-B transponders must perform, harmonised with EUROCAE's ED-102.</figcaption>
</figure>

## Overview

RTCA — historically the Radio Technical Commission for Aeronautics — is a US private,
not-for-profit association that develops consensus technical standards for aviation, working
closely with the Federal Aviation Administration, which frequently adopts RTCA documents by
reference in its regulations. Its output takes the form of **DO-** documents, most importantly
the **Minimum Operational Performance Standards (MOPS)** that specify how a piece of avionics
must behave and perform to be certified. Where [ICAO](/reference/icao/) sets the high-level,
worldwide *what* — the message formats and required capabilities in Annex 10 — RTCA writes the
detailed, testable *how* for the US market.

For surveillance, two RTCA documents matter to anyone receiving aircraft signals. **DO-260**
(with revisions DO-260A and DO-260B) is the MOPS for **1090 MHz Extended Squitter**
[ADS-B](/reference/ads-b/), riding on [Mode S](/reference/mode-s/); the revision letter denotes
successive versions of the message set and integrity reporting, and equipment or decoders often
state which version they implement. **DO-282** is the parallel MOPS for the alternative
**Universal Access Transceiver** ([UAT-978](/reference/uat-978/)) ADS-B link used at 978 MHz by
general aviation in the United States. RTCA's standards are deliberately harmonised with their
European counterparts from [EUROCAE](/reference/eurocae/) — DO-260B corresponds to ED-102A — so
that avionics work interchangeably on both sides of the Atlantic, which is essential for
international flight.

## Relevance to SDR

RTCA's DO-260B is the document most often cited in ADS-B decoder documentation, because it is
the precise definition of the bit-level message formats a receiver must parse: the Extended
Squitter types, the position and velocity encodings (including
[Compact Position Reporting](/reference/compact-position-reporting/)), and the integrity fields.
When an open-source decoder note says it "implements DO-260B", it means it handles that version's
full message set. For the 978 MHz UAT link common on US light aircraft, DO-282 plays the same
role, which is why UAT decoders and 1090 MHz decoders are effectively separate implementations
of two different RTCA standards.

Aircraft surveillance is outside GopherTrunk's land-mobile trunking scope, so GopherTrunk does
not decode ADS-B itself — dedicated 1090 MHz and 978 MHz tools do that — but RTCA is worth
knowing as the authority that pins down exactly what those signals contain. See the
[other signals you'll meet](/learn/rf-sdr/other-signals/) lesson for where ADS-B sits among the
wider set of receivable signals.

## Sources

[^home]: [RTCA](https://www.rtca.org/) — RTCA's official site, developer of the DO-260 (ADS-B) and DO-282 (UAT) Minimum Operational Performance Standards.
[^wiki]: [RTCA](https://en.wikipedia.org/wiki/RTCA) — Wikipedia, on RTCA and its DO-260 MOPS defining ADS-B performance requirements.
