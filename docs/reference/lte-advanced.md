---
slug: lte-advanced
title: LTE-Advanced (LTE-A)
entry_type: protocol
category: cellular
description: LTE-Advanced is the 3GPP Release 10 enhancement of LTE that reaches true 4G IMT-Advanced rates through carrier aggregation, higher-order MIMO, and coordinated multipoint.
keywords: LTE-Advanced, LTE-A, LTE Advanced Pro, 3GPP Release 10, carrier aggregation, MIMO, CoMP, relay nodes, IMT-Advanced, 4G, gigabit LTE
aka: [LTE-Advanced, LTE-A, LTE Advanced Pro]
autolink: true
infobox:
  - { label: Type, value: 4G cellular (IMT-Advanced) }
  - { label: Standards body, value: "3GPP (Release 10, 2011)" }
  - { label: Introduced, value: "2013 (commercial)" }
  - { label: Access, value: "OFDMA / SC-FDMA with carrier aggregation" }
  - { label: Aggregate bandwidth, value: "Up to 100 MHz (5 × 20 MHz)" }
  - { label: Multi-antenna, value: "Up to 8×8 downlink MIMO" }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [lte, mimo, ofdma, 5g-nr, 3gpp, carrier-to-noise-ratio]
cite_urls:
  - https://en.wikipedia.org/wiki/LTE_Advanced
  - https://www.3gpp.org/technologies/keywords-acronyms/97-lte-advanced
---

**LTE-Advanced (LTE-A)** is the [3GPP](/reference/3gpp/) Release 10 evolution of
[LTE](/reference/lte/) that was the first version to satisfy the ITU's IMT-Advanced
definition of true 4G.[^wiki] It reaches gigabit-class peak rates chiefly through
**carrier aggregation** — bonding several LTE carriers into one wider effective
channel — combined with higher-order [MIMO](/reference/mimo/) and coordinated
transmission between cells.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="Three separate 20 MHz LTE component carriers, some contiguous and some in different bands, are combined by carrier aggregation into a single 60 MHz aggregate pipe delivered to one handset." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="ltaar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" stroke-width="1">
    <rect x="30" y="30" width="70" height="34" fill="currentColor" fill-opacity="0.2"/>
    <rect x="110" y="30" width="70" height="34" fill="currentColor" fill-opacity="0.2"/>
    <rect x="230" y="30" width="70" height="34" fill="currentColor" fill-opacity="0.2"/>
  </g>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <text x="65" y="51">CC1 20 MHz</text><text x="145" y="51">CC2 20 MHz</text><text x="265" y="51">CC3 20 MHz</text>
    <text x="145" y="20">Band A (contiguous)</text><text x="265" y="20">Band B</text>
  </g>
  <line x1="165" y1="70" x2="220" y2="110" stroke="currentColor" stroke-width="0.9" marker-end="url(#ltaar)"/>
  <line x1="265" y1="70" x2="230" y2="110" stroke="currentColor" stroke-width="0.9" marker-end="url(#ltaar)"/>
  <rect x="120" y="112" width="220" height="30" fill="currentColor" fill-opacity="0.32" stroke="currentColor"/>
  <text x="230" y="131" text-anchor="middle" font-size="9" fill="currentColor">aggregated 60 MHz to one UE</text>
</svg>
<figcaption>Carrier aggregation bonds multiple LTE component carriers — even in different bands — into one wider pipe for a single handset.</figcaption>
</figure>

## Overview

Where baseline LTE topped out at a 20 MHz carrier, LTE-Advanced lets a handset receive
and transmit on several **component carriers** at once, contiguous or scattered across
bands, for up to 100 MHz of aggregate bandwidth in Release 10. It reuses the same
[OFDMA](/reference/ofdma/) waveform and resource-block grid as LTE, so it is an
enhancement layered onto existing networks rather than a new air interface.

## Technical characteristics

| Property | Value |
|----------|-------|
| Carrier aggregation | Up to 5 × 20 MHz component carriers (100 MHz) |
| Downlink MIMO | Up to 8×8 spatial layers |
| Uplink MIMO | Up to 4×4 |
| CoMP | Coordinated multipoint transmission/reception |
| Relay nodes | Wireless backhaul relays for coverage infill |
| Heterogeneous networks | Macro + small-cell interference coordination (eICIC) |

Later Release 13/14 enhancements, marketed as **LTE Advanced Pro** or "gigabit LTE,"
pushed aggregation to more carriers and added licensed-assisted access (LAA) into
unlicensed spectrum.

## History

3GPP completed Release 10 in **2011**, with commercial LTE-Advanced networks appearing
from about 2013.[^3gpp] It served as the practical bridge to [5G NR](/reference/5g-nr/):
many features first proven in LTE-A — dense carrier aggregation, massive MIMO, and
tight small-cell coordination — were carried forward into the 5G design.

## Deployment

Nearly all mature LTE operators enabled carrier aggregation, so most "4G" service in
the late 2010s was in practice LTE-Advanced. It remains a heavily used capacity and
coverage layer, frequently aggregated together with 5G carriers in non-standalone
deployments.

## Decoding it with GopherTrunk

**GopherTrunk does not decode LTE-Advanced.** As with [LTE](/reference/lte/), this is a
licensed wideband cellular air interface far outside the scope of a narrowband
land-mobile trunking scanner, and carrier aggregation only widens the required capture
bandwidth. It is catalogued here to explain how 4G reached gigabit rates in the same
spectral neighbourhood a scanner monitors.

## Sources

[^wiki]: [LTE Advanced](https://en.wikipedia.org/wiki/LTE_Advanced) — Wikipedia, for carrier aggregation, higher-order MIMO, and the IMT-Advanced qualification.
[^3gpp]: [LTE-Advanced](https://www.3gpp.org/technologies/keywords-acronyms/97-lte-advanced) — 3GPP, for the Release 10 timeline and feature set.
