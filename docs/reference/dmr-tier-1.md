---
slug: dmr-tier-1
title: DMR Tier I
entry_type: protocol
category: land-mobile-trunking
description: DMR Tier I is the licence-free tier of the ETSI DMR standard, intended for low-power consumer and light commercial use without an individual licence.
keywords: DMR Tier I, DMR Tier 1, licence-free, PMR446, dPMR, consumer radio
aka: [DMR Tier I, DMR Tier 1]
autolink: true
infobox:
  - { label: Type, value: Digital land-mobile radio }
  - { label: Part of, value: DMR (ETSI) }
  - { label: Licensing, value: Licence-free / low power }
  - { label: Access, value: TDMA-capable (often single-slot use) }
  - { label: Channel spacing, value: 12.5 kHz }
  - { label: Modulation, value: 4FSK }
  - { label: Vocoder, value: AMBE+2 }
  - { label: GopherTrunk support, value: As conventional DMR }
see_also: [dmr, dmr-tier-2, dmr-tier-3, frequency-shift-keying, etsi]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
---

**DMR Tier I** is the **licence-free** tier of the [DMR](/reference/dmr/) standard,
defined by [ETSI](/reference/etsi/) for low-power, short-range consumer and light
commercial use without an individual user licence.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 360 110" role="img" aria-label="A single licence-free DMR Tier I channel at low power." xmlns="http://www.w3.org/2000/svg">
  <rect x="60" y="40" width="240" height="34" rx="5" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.2"/>
  <text x="180" y="61" text-anchor="middle" font-size="9" fill="currentColor">fixed low-power channel</text>
  <text x="180" y="96" text-anchor="middle" font-size="9" fill="currentColor">licence-free, no trunking</text>
</svg>
<figcaption>DMR Tier I uses fixed, low-power licence-free channels with no system planning or trunking.</figcaption>
</figure>

## Overview

Tier I targets the same role as licence-free analog radios, but digital. It uses
fixed low power and a small set of designated channels, so no system planning or
trunking is involved. Equipment is simple and inexpensive.

## Technical characteristics

| Property | Value |
|----------|-------|
| Licensing | Licence-free, limited power |
| Access | Conventional (TDMA-capable hardware) |
| Channel | 12.5 kHz |
| Modulation | [4FSK](/reference/frequency-shift-keying/) |
| Vocoder | [AMBE+2](/reference/ambe-plus-2/) |

## History

Tier I was specified alongside the other tiers in ETSI's DMR documents to cover the
unlicensed consumer segment.[^wiki]

## Deployment

Found in inexpensive consumer handhelds in regions that allocate licence-free DMR
channels; less common than Tier II in North America.

## Decoding it with GopherTrunk

Tier I traffic is conventional DMR and decodes like any [Tier II](/reference/dmr-tier-2/)
conventional channel once tuned. See [Status](/status.html).

## Sources

[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, for the ETSI DMR tier structure, including the licence-free Tier I.
