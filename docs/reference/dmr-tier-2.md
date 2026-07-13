---
slug: dmr-tier-2
title: DMR Tier II
entry_type: protocol
category: land-mobile-trunking
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
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
---

**DMR Tier II** is the **licensed, conventional** tier of the [DMR](/reference/dmr/)
standard. It is non-trunked — each repeater pair uses fixed frequencies — but carries
**two voice timeslots** per 12.5 kHz channel via [TDMA](/reference/tdma/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 380 140" role="img" aria-label="Two TDMA slots in a conventional DMR Tier II channel." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="105" x2="360" y2="105" stroke="currentColor" stroke-opacity="0.4" marker-end="url(#ta_dmr2)"/>
  <text x="195" y="128" text-anchor="middle" font-size="9" fill="currentColor">time → · one 12.5 kHz channel, 2 slots</text>
  <g stroke="currentColor" stroke-width="1.1"><rect x="40" y="40" width="52" height="50" fill="currentColor" fill-opacity="0.22"/><rect x="92" y="40" width="52" height="50" fill="none"/><rect x="144" y="40" width="52" height="50" fill="currentColor" fill-opacity="0.22"/><rect x="196" y="40" width="52" height="50" fill="none"/><rect x="248" y="40" width="52" height="50" fill="currentColor" fill-opacity="0.22"/><rect x="300" y="40" width="52" height="50" fill="none"/></g><g font-size="9" fill="currentColor" text-anchor="middle"><text x="66" y="69">1</text><text x="118" y="69">2</text><text x="170" y="69">1</text><text x="222" y="69">2</text><text x="274" y="69">1</text><text x="326" y="69">2</text></g>
  <defs><marker id="ta_dmr2" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>DMR Tier II is conventional two-slot TDMA — two calls per fixed channel.</figcaption>
</figure>

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
MOTOTRBO line from the late 2000s.[^wiki]

## Deployment

Extremely common in business, utility, and amateur radio. Amateur networks bridge
Tier II repeaters and hotspots over the internet.

## Decoding it with GopherTrunk

GopherTrunk decodes both timeslots of a Tier II channel and renders AMBE+2 audio.
Color Code, time slot, and talkgroup are read off the air per call — you only
configure each repeater's frequency. A single dongle channelizes a cluster of
repeaters at once; to cover many repeaters spread across a wide band, add one
`role: wideband` dongle per ~2 MHz cluster, all pointed at the same system. The
[`dmr-tier2-multi-repeater`](https://github.com/MattCheramie/GopherTrunk/tree/main/samples/dmr-tier2-multi-repeater)
sample is a worked "a bunch of repeaters, different CCs and slots" config; for a
single carrier see
[`dmr-simplex`](https://github.com/MattCheramie/GopherTrunk/tree/main/samples/dmr-simplex).
For trunked DMR, see [Tier III](/reference/dmr-tier-3/). See [Status](/status.html).

## Sources

[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, for the ETSI DMR tiers, including the licensed conventional Tier II and its two-slot TDMA.
