---
slug: rfss
title: RF Subsystem (RFSS)
entry_type: term
category: trunked-radio
description: "An RFSS is an 8-bit P25 identifier for an RF Subsystem, a cluster of trunking sites under one controller; with the WACN and System ID it completes a network's address."
keywords: RFSS, RF subsystem, 8-bit, P25 site cluster, sub-system, WACN, system id, site controller, zone, trunking
aka: [RFSS, RF subsystem]
autolink: true
infobox:
  - { label: Type, value: P25 subsystem identifier }
  - { label: Size, value: 8 bits (0x00–0xFF) }
  - { label: Groups, value: A cluster of trunking sites }
see_also: [wacn, system-id, trunking-site, multisite-trunking, project-25, roaming]
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
---

An **RFSS (RF Subsystem)** is an 8-bit code identifying a cluster of
[trunking sites](/reference/trunking-site/) that operate under one controller within a
[P25](/reference/project-25/) system — the lowest of the three levels of the P25
identity hierarchy.[^wiki] It sits beneath the [System ID](/reference/system-id/), which
sits beneath the [WACN](/reference/wacn/); together the triple
`WACN + System ID + RFSS` uniquely names one operator's subsystem, and each RFSS then
contains its individually numbered sites.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="One RFSS drawn as a controller connected to several numbered trunking sites, itself nested inside a System ID and WACN." xmlns="http://www.w3.org/2000/svg">
  <text x="30" y="24" font-size="9" fill="currentColor">WACN › System ID › RFSS 1</text>
  <rect x="180" y="34" width="100" height="28" rx="6" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/>
  <text x="230" y="52" text-anchor="middle" font-size="9" fill="currentColor">RFSS controller</text>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <rect x="40" y="110" width="90" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="85" y="129">Site 1</text>
    <rect x="185" y="110" width="90" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="230" y="129">Site 2</text>
    <rect x="330" y="110" width="90" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="375" y="129">Site 3</text>
  </g>
  <g stroke="currentColor" stroke-width="1" stroke-dasharray="4 3">
    <line x1="210" y1="62" x2="90" y2="108"/><line x1="230" y1="62" x2="230" y2="108"/><line x1="250" y1="62" x2="372" y2="108"/>
  </g>
</svg>
<figcaption>An RFSS groups a set of numbered sites under one controller; it is the innermost field of the P25 identity triple.</figcaption>
</figure>

## How it works

The RFSS number is broadcast on the [control channel](/reference/control-channel/) in
the network-status messages, alongside the WACN and System ID, so a radio or monitor
learns the full identity of the subsystem it is hearing. Within an RFSS each site has
its own site number, and the sites are linked so that talkgroups can be active across
several of them at once — the [multisite](/reference/multisite-trunking/) behaviour that
lets radios [roam](/reference/roaming/) seamlessly. A large network may divide into
multiple RFSSs (for example by geography or agency), each with its own controller but
sharing the parent System ID and WACN.

Because the RFSS field is only 8 bits, values repeat across different systems; it is
meaningful only *inside* its parent WACN and System ID. Small systems often have a
single RFSS numbered 1, so in practice the RFSS mainly matters on large statewide
networks that split into several subsystems.

## In practice

- The RFSS is written as a two-hex-digit or decimal value (8 bits, 0–255); databases
  list it after the WACN and System ID and before the individual site number.
- Statewide systems commonly carve their coverage into several RFSSs — for example by
  region or by owning agency — each with its own controller but one shared System ID.
- A radio uses the site number *within* an RFSS, plus the RFSS itself, to know exactly
  which cell it is on and which neighbours it can roam to.

## Relevance to SDR

For a monitor the RFSS completes the picture begun by the WACN and System ID: it tells
you not just which network but which subsystem — and by extension which region or agency
— a control channel serves. **GopherTrunk** decodes the RFSS from the P25 control
channel and reports it with the WACN and System ID, so each tracked control channel
carries its full P25 identity. As with the other identity fields, this is descriptive
metadata rather than a security control.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on the P25 system-identity hierarchy including the RF Subsystem.
