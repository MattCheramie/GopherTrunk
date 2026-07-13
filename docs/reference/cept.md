---
slug: cept
title: European Conference of Postal and Telecommunications Administrations (CEPT)
entry_type: organization
category: organizations
description: "CEPT is the European body that coordinates postal and telecommunications policy, harmonising radio spectrum use across member countries."
keywords: CEPT, European Conference of Postal and Telecommunications Administrations, ECC, spectrum harmonisation, Europe, telecommunications policy
aka: [CEPT, European Conference of Postal and Telecommunications Administrations]
autolink: true
infobox:
  - { label: Type, value: Intergovernmental organisation }
  - { label: Founded, value: "1959" }
  - { label: Region, value: Europe (48 member countries) }
see_also: [etsi, ofcom, itu, itu-r, frequency-bands]
cite_urls:
  - https://www.cept.org/
  - https://en.wikipedia.org/wiki/European_Conference_of_Postal_and_Telecommunications_Administrations
---

The **European Conference of Postal and Telecommunications Administrations** (**CEPT**)
is the **intergovernmental body through which European national regulators coordinate
postal and telecommunications policy**, most importantly the harmonised use of the radio
[spectrum](/reference/frequency-bands/).[^wiki] Founded in 1959, CEPT gives the
regulators of nearly fifty countries a forum to agree common frequency plans and licensing
approaches so that equipment and services work consistently across borders.[^home]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="CEPT coordinating multiple European national regulators into a common harmonised frequency plan." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="180" y="12" width="100" height="30" rx="5" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.2"/><text x="230" y="31">CEPT / ECC</text>
    <rect x="20" y="84" width="90" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="65" y="101">Regulator A</text>
    <rect x="130" y="84" width="90" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="175" y="101">Regulator B</text>
    <rect x="240" y="84" width="90" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="285" y="101">Regulator C</text>
    <rect x="350" y="84" width="90" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="395" y="101">Regulator D</text>
    <g stroke="currentColor" stroke-width="1.1">
      <line x1="230" y1="42" x2="65" y2="83" marker-end="url(#ar_cept)"/>
      <line x1="230" y1="42" x2="175" y2="83" marker-end="url(#ar_cept)"/>
      <line x1="230" y1="42" x2="285" y2="83" marker-end="url(#ar_cept)"/>
      <line x1="230" y1="42" x2="395" y2="83" marker-end="url(#ar_cept)"/>
    </g>
  </g>
  <defs><marker id="ar_cept" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>CEPT harmonises spectrum decisions across its member regulators so plans align Europe-wide.</figcaption>
</figure>

## Overview

CEPT works through several permanent committees. The **Electronic Communications Committee**
(ECC) handles radio spectrum and does the technical heavy lifting: it issues ECC Decisions
and Recommendations, publishes the European Common Allocation table, and manages harmonised
band plans for services from broadcasting to short-range devices. A parallel committee deals
with postal matters, and a third coordinates the European positions taken into global forums.
CEPT decisions are not binding treaties in themselves, but members implement them nationally,
which is why a licence-free band or a public-safety allocation tends to look the same from
one European country to the next.

Historically CEPT also incubated technical standards work: the effort that became the GSM
mobile standard began inside CEPT before being handed to a dedicated standards body. That
spin-off created [ETSI](/reference/etsi/) in 1988, and the two organisations remain closely
linked — CEPT sets the regulatory and spectrum framework while ETSI writes the detailed
technical standards that operate within it. CEPT also coordinates the common European brief
carried into [ITU](/reference/itu/) meetings and the treaty-level conferences run by the
[ITU-R](/reference/itu-r/) sector.

## Relevance to SDR

For anyone scanning in Europe, CEPT is the reason the bands behave predictably. When you
tune a European PMR446 walkie-talkie channel, a harmonised trunking allocation, or an
ISM/short-range-device band, the plan you are listening across was almost certainly set by
a CEPT/ECC decision that national regulators such as the UK's [Ofcom](/reference/ofcom/)
then adopted. That harmonisation is what lets a single band plan and a single receiver
configuration make sense across the continent, rather than needing per-country retuning.

CEPT allocations shape where land-mobile trunking, amateur, maritime, and aviation signals
live, and therefore where GopherTrunk users point their radios. GopherTrunk itself does not
encode any regulatory data; it simply decodes whatever it receives. But understanding that a
European deployment sits in a CEPT-harmonised band helps explain why the same
[frequency plans](/reference/frequency-bands/) recur across countries and where to expect a
given service.

## Sources

[^home]: [CEPT](https://www.cept.org/) — the organisation's official site, for its committee structure, ECC decisions, and harmonised band plans.
[^wiki]: [European Conference of Postal and Telecommunications Administrations](https://en.wikipedia.org/wiki/European_Conference_of_Postal_and_Telecommunications_Administrations) — Wikipedia, for CEPT's history, membership, and role in European spectrum harmonisation.
