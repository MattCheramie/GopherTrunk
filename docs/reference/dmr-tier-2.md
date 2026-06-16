---
slug: dmr-tier-2
title: DMR Tier II
entry_type: protocol
category: protocols
description: DMR Tier II is the licensed conventional tier of the ETSI DMR standard, using two-slot TDMA in 12.5 kHz channels — the most common commercial and amateur DMR mode.
keywords: DMR Tier II, DMR Tier 2, conventional DMR, MOTOTRBO, two-slot TDMA, AMBE+2
aka: [DMR Tier II, DMR Tier 2]
autolink: true
infobox:
  - { label: Type, value: Digital land-mobile radio }
  - { label: Part of, value: DMR (ETSI) }
  - { label: Licensing, value: Licensed conventional }
  - { label: Access, value: Two-slot TDMA }
  - { label: Channel spacing, value: 12.5 kHz }
  - { label: Modulation, value: 4FSK (9600 bps) }
  - { label: Vocoder, value: AMBE+2 }
  - { label: GopherTrunk support, value: Decoded (both slots) }
see_also: [dmr, dmr-tier-1, dmr-tier-3, tdma, ambe-plus-2]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/protocol-landscape/ }
external:
  - { title: "Digital mobile radio (Wikipedia)", url: https://en.wikipedia.org/wiki/Digital_mobile_radio }
---

**DMR Tier II** is the **licensed, conventional** tier of the [DMR](/reference/dmr/)
standard. It is non-trunked — each repeater pair uses fixed frequencies — but carries
**two voice timeslots** per 12.5 kHz channel via [TDMA](/reference/tdma/).

## Overview

Tier II is the workhorse of commercial DMR and the basis of amateur DMR repeaters.
Because it is conventional, you tune directly to a known frequency rather than
following a [control channel](/reference/control-channel/); the two timeslots still
allow two simultaneous conversations.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | Two-slot TDMA |
| Channel | 12.5 kHz |
| Modulation | [4FSK](/reference/frequency-shift-keying/), 9600 bps |
| Vocoder | [AMBE+2](/reference/ambe-plus-2/) |

## History

Tier II was the first widely commercialised DMR tier, popularised by Motorola's
MOTOTRBO line from the late 2000s.

## Deployment

Extremely common in business, utility, and amateur radio. Amateur networks bridge
Tier II repeaters and hotspots over the internet.

## Decoding it with GopherTrunk

GopherTrunk decodes both timeslots of a Tier II channel and renders AMBE+2 audio. For
trunked DMR, see [Tier III](/reference/dmr-tier-3/). See [Status](/status.html).
