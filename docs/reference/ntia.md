---
slug: ntia
title: National Telecommunications and Information Administration (NTIA)
entry_type: organization
category: organizations
description: "The NTIA is the US executive-branch agency that manages federal government use of the radio spectrum, the counterpart to the FCC's authority over non-federal users."
keywords: NTIA, National Telecommunications and Information Administration, federal spectrum, US government, Commerce Department, IRAC, spectrum management
aka: [NTIA, National Telecommunications and Information Administration]
autolink: true
infobox:
  - { label: Type, value: US federal agency (Dept. of Commerce) }
  - { label: Established, value: "1978" }
  - { label: Role, value: Manages federal government spectrum use }
see_also: [fcc, frequency-bands, itu, itu-r]
cite_urls:
  - https://www.ntia.gov/
  - https://en.wikipedia.org/wiki/National_Telecommunications_and_Information_Administration
---

The **National Telecommunications and Information Administration** (**NTIA**) is the
United States executive-branch agency that **manages the federal government's use of the
radio [spectrum](/reference/frequency-bands/)**.[^wiki] Part of the Department of Commerce,
it is the counterpart to the [FCC](/reference/fcc/): where the FCC licenses commercial,
amateur, and other non-federal users, the NTIA allocates and assigns frequencies to federal
agencies such as the military, aviation authorities, and public-safety and scientific
bodies.[^home]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 108" role="img" aria-label="US spectrum management split between the NTIA for federal users and the FCC for non-federal users." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="170" y="10" width="120" height="26" rx="5" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.2"/><text x="230" y="27">US spectrum</text>
    <rect x="30" y="60" width="150" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="105" y="79">NTIA — federal</text>
    <rect x="280" y="60" width="150" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="355" y="79">FCC — non-federal</text>
    <g stroke="currentColor" stroke-width="1.1">
      <line x1="205" y1="36" x2="115" y2="59" marker-end="url(#ar_ntia)"/>
      <line x1="255" y1="36" x2="345" y2="59" marker-end="url(#ar_ntia)"/>
    </g>
  </g>
  <defs><marker id="ar_ntia" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>In the US, federal spectrum is managed by the NTIA and non-federal spectrum by the FCC.</figcaption>
</figure>

## Overview

The NTIA was created in 1978 by consolidating earlier telecommunications-policy offices, and
it serves as the President's principal adviser on telecommunications and information policy.
Its spectrum role is exercised largely through the **Office of Spectrum Management**, which
maintains the assignments for the roughly sixty federal agencies that use radio. Much of the
coordination work happens in the **Interdepartment Radio Advisory Committee** (IRAC), where
federal users negotiate their shared use of bands.

Because the United States has this split system, the national allocation table is a joint
product: the NTIA's federal allocations and the FCC's non-federal allocations are reconciled
into a single chart, and many bands are shared between the two on a coordinated basis. The
NTIA also runs research at its Institute for Telecommunication Sciences, administers grant
programmes for broadband and public-safety communications, and — with the FCC and the State
Department — helps form the US position taken into [ITU](/reference/itu/) forums and the
treaty conferences run by the [ITU-R](/reference/itu-r/) sector.

## Relevance to SDR

For anyone scanning in the United States, the NTIA explains a large share of what appears on
the waterfall. Federal law-enforcement, military, aeronautical, and land-mobile systems all
operate under NTIA assignments rather than FCC licences, so a band that looks empty in the
FCC's public licence database may be heavily used by federal agencies coordinated through the
NTIA. Recognising whether a band is federal (NTIA) or non-federal (FCC) is a useful first
step in identifying an unknown signal.

The NTIA does not affect how a receiver decodes anything, so GopherTrunk implements none of
its rules. But its allocations shape where US federal trunking and other systems live, which
is directly relevant to deciding where to point a scanner. As always, GopherTrunk decodes
only what it lawfully receives, and users are responsible for the rules that apply to
federal and non-federal bands in their jurisdiction.

## Sources

[^home]: [NTIA](https://www.ntia.gov/) — the agency's official site, for its spectrum-management role and federal allocation responsibilities.
[^wiki]: [National Telecommunications and Information Administration](https://en.wikipedia.org/wiki/National_Telecommunications_and_Information_Administration) — Wikipedia, for the NTIA's history, structure, and division of spectrum authority with the FCC.
