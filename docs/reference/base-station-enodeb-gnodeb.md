---
slug: base-station-enodeb-gnodeb
title: Base station (eNodeB / gNodeB)
entry_type: term
category: cellular
description: A cellular base station is the fixed radio node that links handsets to the core network; in LTE it is the eNodeB and in 5G NR the gNodeB, typically split into sectors per cell site.
keywords: base station, cell site, cell tower, eNodeB, eNB, gNodeB, gNB, NodeB, BTS, sector, cell, RRH, baseband unit, backhaul, macro cell, small cell, RAN
aka: [base station, eNodeB, eNB, gNodeB, gNB, cell site]
autolink: true
infobox:
  - { label: Role, value: Fixed radio node between handsets and core network }
  - { label: LTE name, value: "eNodeB (evolved Node B)" }
  - { label: 5G NR name, value: "gNodeB (next-generation Node B)" }
see_also: [lte, 5g-nr, mimo, beamforming, antenna, trunking-site]
cite_urls:
  - https://en.wikipedia.org/wiki/Base_station
  - https://en.wikipedia.org/wiki/ENodeB
---

A **base station** is the fixed radio node that connects mobile handsets to the
operator's core network over the air interface. In [LTE](/reference/lte/) it is called
the **eNodeB (evolved Node B)**, and in [5G NR](/reference/5g-nr/) the **gNodeB
(next-generation Node B)**; both terms name the same functional thing — the tower-side
radio that a cell site presents to phones.[^wiki][^enb] A single physical site is
usually divided into **sectors**, each a separate cell facing a different direction.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A cell site mast with three sector antennas facing 120 degrees apart, each covering a wedge-shaped cell, connected down to a baseband unit that links via backhaul to the core network." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="bsar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="150" y1="30" x2="150" y2="120" stroke="currentColor" stroke-width="1.6"/>
  <g stroke="currentColor" fill="currentColor" fill-opacity="0.12">
    <path d="M150 60 L110 30 L150 20 Z"/><path d="M150 60 L190 30 L150 20 Z"/>
    <path d="M150 90 L110 120 L100 90 Z"/>
  </g>
  <text x="150" y="138" text-anchor="middle" font-size="8" fill="currentColor">mast + 3 sector antennas</text>
  <rect x="240" y="55" width="70" height="34" fill="currentColor" fill-opacity="0.14" stroke="currentColor"/>
  <text x="275" y="72" text-anchor="middle" font-size="8" fill="currentColor">baseband</text><text x="275" y="82" text-anchor="middle" font-size="7" fill="currentColor">eNB / gNB</text>
  <rect x="370" y="55" width="70" height="34" fill="none" stroke="currentColor"/>
  <text x="405" y="76" text-anchor="middle" font-size="8" fill="currentColor">core network</text>
  <line x1="165" y1="72" x2="238" y2="72" stroke="currentColor" marker-end="url(#bsar)"/>
  <line x1="310" y1="72" x2="368" y2="72" stroke="currentColor" marker-end="url(#bsar)"/>
  <text x="340" y="52" text-anchor="middle" font-size="7" fill="currentColor">backhaul</text>
</svg>
<figcaption>A cell site divides into sectors, each a cell served by directional antennas; the base station's baseband unit links those radios via backhaul to the core network.</figcaption>
</figure>

## How it works

Physically, a base station combines antennas (usually mounted high for line of sight),
radio units that up- and down-convert between baseband and RF, and a **baseband unit**
that runs the modem and scheduler. In modern deployments the radio is often a
**remote radio head** at the antenna, connected by fibre to a baseband unit at the base
of the tower or pooled in a central office (a "cloud RAN" split). The base station
schedules the shared air interface, assigning time-frequency resources to each handset,
manages [handovers](/reference/cellular-handover/) as users move, and increasingly steers
energy with massive [MIMO](/reference/mimo/) and [beamforming](/reference/beamforming/),
especially in 5G.

A **cell** is the coverage area of one sector on one carrier; a typical macro site hosts
three sectors at 120° spacing, and dense areas add small cells. Naming has tracked the
generations: **BTS** in GSM, **NodeB** in 3G, **eNodeB** in 4G LTE, and **gNodeB** in
5G NR — the "e" and "g" prefixes marking each generation's evolution of the same node.

## Relevance to SDR

The base station is the counterpart, in cellular, of the [trunking site](/reference/trunking-site/)
in land-mobile radio: a fixed, sectored radio node that hands mobiles between coverage
areas and coordinates access to shared spectrum. That parallel is the useful one for SDR
readers — the sectored-site, neighbour-list, handover architecture GopherTrunk follows on
[P25](/reference/p25-phase-1/) and [DMR](/reference/dmr/) trunked networks is
conceptually the same as a cellular eNodeB/gNodeB, just narrowband and (often) in the
clear.

**GopherTrunk does not decode cellular base stations.** eNodeB and gNodeB air interfaces
are wideband, licensed, and encrypted, and lie outside the scope of a land-mobile
trunking scanner. The term is documented here so the cellular network structure a scanner
shares the spectrum with is clear, and to draw the analogy to the trunking sites
GopherTrunk does follow.

## Sources

[^wiki]: [Base station](https://en.wikipedia.org/wiki/Base_station) — Wikipedia, for the general definition of a base station and cell-site sectorisation.
[^enb]: [eNodeB](https://en.wikipedia.org/wiki/ENodeB) — Wikipedia, for the LTE eNodeB and its relationship to the NodeB/gNodeB lineage.
