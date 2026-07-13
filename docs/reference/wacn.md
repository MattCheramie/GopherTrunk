---
slug: wacn
title: Wide Area Communications Network (WACN)
entry_type: term
category: trunked-radio
description: "A WACN is a 20-bit P25 identifier naming a wide-area network; combined with the System ID and RFSS it uniquely identifies which trunked system a radio is on."
keywords: WACN, wide area communications network, 20-bit, P25 network ID, system identity, WACN system ID, RFSS, roaming, P25 registration
aka: [WACN, wide area communications network]
autolink: true
infobox:
  - { label: Type, value: P25 network identifier }
  - { label: Size, value: 20 bits (0x00000–0xFFFFF) }
  - { label: With, value: System ID + RFSS = full identity }
see_also: [system-id, rfss, project-25, network-access-code, roaming, radio-id]
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
---

A **Wide Area Communications Network (WACN)** is a 20-bit code that names a P25
wide-area network — the top level of the [P25](/reference/project-25/) identity
hierarchy.[^wiki] On its own a WACN is not enough to pinpoint a system; combined with
the [System ID](/reference/system-id/) and the [RFSS](/reference/rfss/) it forms the
globally unique address of the trunked system a radio is registered on. The WACN space
is administered so that operators receive blocks that do not collide, much as IP address
blocks are allocated.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A nested hierarchy: a WACN contains one or more System IDs, each of which contains one or more RFSSs, each of which contains sites." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="20" width="420" height="120" rx="7" fill="currentColor" fill-opacity="0.06" stroke="currentColor" stroke-width="1.3"/>
  <text x="30" y="36" font-size="9.5" fill="currentColor">WACN 0xBEE00 (20 bits)</text>
  <rect x="34" y="44" width="392" height="84" rx="6" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1.1"/>
  <text x="44" y="60" font-size="9" fill="currentColor">System ID 0x2A7 (12 bits)</text>
  <rect x="48" y="68" width="180" height="52" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/>
  <text x="138" y="86" text-anchor="middle" font-size="8.5" fill="currentColor">RFSS 1 (8 bits)</text>
  <text x="138" y="102" text-anchor="middle" font-size="8" fill="currentColor">sites 1…n</text>
  <rect x="240" y="68" width="180" height="52" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/>
  <text x="330" y="86" text-anchor="middle" font-size="8.5" fill="currentColor">RFSS 2 (8 bits)</text>
  <text x="330" y="102" text-anchor="middle" font-size="8" fill="currentColor">sites 1…n</text>
</svg>
<figcaption>P25 identity nests: a WACN contains System IDs, which contain RFSSs, which contain sites — together a globally unique address.</figcaption>
</figure>

## How it works

The WACN, System ID, and RFSS are broadcast on the [control channel](/reference/control-channel/)
in the network-status and identifier messages, so any radio — or monitor — can learn
exactly which system it is hearing without prior configuration. A subscriber radio uses
this triple during [registration](/reference/registration/) and
[roaming](/reference/roaming/): as it moves between sites it checks whether a candidate
site advertises its home WACN and System ID before affiliating, and treats a foreign
WACN as another operator's network.

The 20-bit WACN allows roughly a million distinct wide-area networks. It sits above the
12-bit System ID (which distinguishes systems within one operator's WACN) and the 8-bit
RFSS (which distinguishes subsystems within a system). Note this identity chain is
separate from the fast per-frame [Network Access Code](/reference/network-access-code/):
the NAC filters individual frames, while the WACN/System ID/RFSS triple is the
registered network identity.

## In practice

- The WACN is written as a five-hex-digit value (20 bits), e.g. `0xBEE00`; public
  databases list it together with the System ID as a system's fingerprint.
- Some large shared networks — statewide interoperability systems and the like — span
  many operators under one coordinated WACN, so the WACN can group agencies that share
  infrastructure.
- A radio treats a site advertising a foreign WACN as another network and will not
  register there, which is how roaming stays contained to the home system.

## Relevance to SDR

The WACN is one of the most useful things a monitor can extract, because it and the
System ID let a scanner recognise a system across sites and match it to public
databases of known networks. **GopherTrunk** decodes the WACN from the P25 control
channel and reports it alongside the System ID and RFSS, which is how it labels which
network a given control channel belongs to. Like the other identity fields it is
descriptive data, not a security mechanism — reading the WACN does not defeat
encryption on voice traffic.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on the P25 system-identity hierarchy including the WACN.
