---
slug: apco-international
title: APCO International
entry_type: organization
category: organizations
description: APCO International is the association of public-safety communications professionals that, with the TIA, drove the creation of the Project 25 (P25) standards.
keywords: APCO, APCO International, public safety communications, P25, Project 25, APCO-25, dispatchers, 9-1-1
aka: [APCO, APCO International]
autolink: true
infobox:
  - { label: Type, value: Professional association }
  - { label: Focus, value: Public-safety communications }
  - { label: Role, value: Co-driver of P25 (APCO-25) }
see_also: [project-25, tia, trunked-radio, fcc, talkgroup, control-channel]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
cite_urls:
  - https://www.apcointl.org/
  - https://en.wikipedia.org/wiki/Association_of_Public-Safety_Communications_Officials-International
---

**APCO International** (the Association of Public-Safety Communications Officials) is the
professional association for public-safety communications.[^home] With the
[TIA](/reference/tia/) it drove the creation of [Project 25](/reference/project-25/),
sometimes called **APCO-25**.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="APCO gathers public-safety user requirements that the TIA turns into the P25 standard." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="42" width="100" height="32" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="70" y="55">APCO</text><text x="70" y="67" font-size="7.5">(users)</text>
    <rect x="170" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="225" y="55">requirements →</text><text x="225" y="67" font-size="7.5">TIA-102 standard</text>
    <rect x="330" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="385" y="55">P25 radios</text><text x="385" y="67" font-size="7.5">interoperable</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="120" y1="58" x2="169" y2="58" marker-end="url(#rel_apco-international)"/><line x1="280" y1="58" x2="329" y2="58" marker-end="url(#rel_apco-international)"/></g>
    <text x="230" y="96" font-size="8" fill="currentColor" fill-opacity="0.85">dispatchers · 9-1-1 centres · radio managers set the needs</text>
  </g>
  <defs><marker id="rel_apco-international" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>APCO International represents public-safety communications users and drove the requirements that became P25.</figcaption>
</figure>

## Overview

APCO was founded in 1935 and is the oldest and largest professional organisation for public-safety
communications, with tens of thousands of members: 9-1-1 telecommunicators (dispatchers), radio
system managers, and the agencies that operate police, fire, and emergency-medical networks. Its
work spans training and certification for dispatchers, operational standards, frequency
coordination, and advocacy before regulators such as the [FCC](/reference/fcc/). But its most
lasting technical legacy is [Project 25](/reference/project-25/).

In the late 1980s, US public-safety agencies faced a problem: proprietary analog and early
digital radios could not interoperate across agencies or jurisdictions, a dangerous gap during
multi-agency incidents. APCO — representing the *users* — set out the requirements for an open,
multi-vendor digital standard: it had to interoperate across manufacturers, work in existing
12.5 kHz channels, support both [conventional](/reference/conventional-radio/) and
[trunked](/reference/trunked-radio/) operation, and provide graceful backward compatibility.
Those requirements were handed to the [TIA](/reference/tia/), which did the formal engineering
and published them as the TIA-102 series. This division of labour — APCO defines *what
public-safety needs*, the TIA specifies *how to build it* — is why the standard is called both
"P25" and "APCO-25". APCO continues to represent user interests as the standard evolves through
[P25 Phase 1](/reference/p25-phase-1/) and [P25 Phase 2](/reference/p25-phase-2/), and it runs
the widely used frequency-coordination and interoperability programmes that shape how real
systems are deployed, from [talkgroup](/reference/talkgroup/) planning to shared regional
networks.

## Relevance to SDR

APCO's role explains both the naming and the character of the systems an SDR user is most
likely to encounter in North America. "APCO-25" and "P25" refer to the same standard; the APCO
half of the name reflects that the requirements came from public-safety users, which is why P25
is built around features like fast [channel grants](/reference/channel-grant/), emergency
signalling, and interoperability talkgroups rather than consumer convenience. When you scan a
metropolitan area and find a [P25 control channel](/reference/control-channel/) coordinating
police, fire, and EMS on shared infrastructure, you are seeing the direct result of the
interoperability goal APCO championed.

APCO does not publish the bit-level air interface — that is the [TIA](/reference/tia/)'s
TIA-102 documents, which is what a decoder actually implements — so its relevance to GopherTrunk
is contextual rather than technical: it tells you *why* the P25 systems GopherTrunk decodes look
the way they do. GopherTrunk decodes clear and scrambled P25 traffic; it cannot recover the
keyed [encryption](/reference/encryption/) that many public-safety agencies now enable on
sensitive talkgroups. For a grounding in how these shared systems work, see the
[what is trunked radio?](/learn/rf-sdr/what-is-trunking/) lesson.

## Sources

[^home]: [APCO International](https://www.apcointl.org/) — the association's official site, representing public-safety communications professionals and its role in P25.
[^wiki]: [Association of Public-Safety Communications Officials-International](https://en.wikipedia.org/wiki/Association_of_Public-Safety_Communications_Officials-International) — Wikipedia, for APCO's history and its role in the creation of P25.
