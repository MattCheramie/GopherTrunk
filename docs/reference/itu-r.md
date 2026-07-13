---
slug: itu-r
title: ITU Radiocommunication Sector (ITU-R)
entry_type: organization
category: organizations
description: "ITU-R is the radiocommunication sector of the ITU that manages the global radio spectrum, publishes the Radio Regulations, and runs the World Radiocommunication Conferences."
keywords: ITU-R, ITU Radiocommunication Sector, Radio Regulations, WRC, spectrum management, satellite orbits, radiocommunication
aka: [ITU-R, ITU Radiocommunication Sector]
autolink: true
infobox:
  - { label: Type, value: Sector of the ITU }
  - { label: Established, value: "1992 (from the CCIR)" }
  - { label: Role, value: Global spectrum management, Radio Regulations }
see_also: [itu, wrc, frequency-bands, fcc, ofcom, cept]
cite_urls:
  - https://www.itu.int/en/ITU-R/
  - https://en.wikipedia.org/wiki/ITU-R
---

The **ITU Radiocommunication Sector** (**ITU-R**) is the arm of the
[International Telecommunication Union](/reference/itu/) responsible for **managing the
global radio-frequency [spectrum](/reference/frequency-bands/) and satellite orbits**.[^wiki]
It maintains the international **Radio Regulations** — the binding treaty that allocates
frequency bands to services worldwide — and organises the conferences at which that treaty
is revised.[^home]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 118" role="img" aria-label="ITU-R producing the Radio Regulations and running the World Radiocommunication Conference, which national regulators implement." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="175" y="12" width="110" height="28" rx="5" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.2"/><text x="230" y="30">ITU-R</text>
    <rect x="40" y="58" width="120" height="28" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="100" y="76">WRC (treaty)</text>
    <rect x="300" y="58" width="120" height="28" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="360" y="76">Radio Regulations</text>
    <rect x="150" y="94" width="160" height="20" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="230" y="108" font-size="8">national regulators (FCC, Ofcom, …)</text>
    <g stroke="currentColor" stroke-width="1.1">
      <line x1="200" y1="40" x2="120" y2="57" marker-end="url(#ar_itur)"/>
      <line x1="260" y1="40" x2="345" y2="57" marker-end="url(#ar_itur)"/>
      <line x1="120" y1="86" x2="215" y2="94" marker-end="url(#ar_itur)"/>
      <line x1="345" y1="86" x2="250" y2="94" marker-end="url(#ar_itur)"/>
    </g>
  </g>
  <defs><marker id="ar_itur" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>ITU-R runs the WRC and maintains the Radio Regulations that national regulators then implement.</figcaption>
</figure>

## Overview

ITU-R is one of the three sectors of the ITU (the others handle standardisation and
development). It was created in 1992 when the ITU reorganised, succeeding the earlier
International Radio Consultative Committee (the CCIR). Its core products are the **Radio
Regulations**, the **table of frequency allocations** that divides the world into three
regions and assigns each band to services such as broadcasting, mobile, fixed, maritime,
aeronautical, radionavigation, amateur, and space research, and a large library of
**ITU-R Recommendations** giving detailed technical parameters for those services.

Much of ITU-R's day-to-day work runs through study groups of experts who develop the
technical basis for decisions, and through the **Radiocommunication Bureau**, which records
frequency assignments and coordinates satellite filings so that networks in geostationary
and other orbits do not interfere. The politically decisive changes, though, happen at the
[World Radiocommunication Conference](/reference/wrc/) — the treaty-making meeting ITU-R
convenes, ordinarily every three to four years, to revise the Radio Regulations. Between
conferences, ITU-R study groups prepare the technical studies that delegates negotiate over.

## Relevance to SDR

Everything you can tune with a software-defined radio sits somewhere in the ITU-R allocation
table. When a specific service — ADS-B at 1090 MHz, marine AIS at 162 MHz, a public-safety
trunking band — appears in the same place across countries, that consistency traces back to
ITU-R allocations that national regulators like the [FCC](/reference/fcc/), the UK's
[Ofcom](/reference/ofcom/), and regional bodies such as [CEPT](/reference/cept/) implement
domestically. Knowing which service ITU-R has allocated a band to is often the fastest way to
guess what an unidentified signal is.

ITU-R does not touch a receiver's decode chain, so GopherTrunk does not implement any of its
regulations directly. But the allocation framework it maintains is the reason a given band
plan is worth pointing a scanner at, and its Recommendations are frequently the primary
reference for the technical parameters of the systems GopherTrunk decodes.

## Sources

[^home]: [ITU Radiocommunication Sector](https://www.itu.int/en/ITU-R/) — the sector's official site, for the Radio Regulations, allocation tables, and conference information.
[^wiki]: [ITU-R](https://en.wikipedia.org/wiki/ITU-R) — Wikipedia, for the sector's history, structure, and role in global spectrum management.
