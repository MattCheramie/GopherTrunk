---
slug: smartnet-smartzone
title: SmartNet / SmartZone
entry_type: protocol
category: land-mobile-trunking
description: SmartNet and SmartZone are Motorola's analog trunking systems that use a dedicated 3600-baud control channel to assign FM voice channels across single or multiple sites.
keywords: SmartNet, SmartZone, Motorola trunking, Type II, 3600 baud control channel, analog trunking, OmniLink, wide-area, land-mobile
aka: [SmartNet, SmartZone, "SmartNet II", OmniLink]
autolink: true
infobox:
  - { label: Type, value: Analog trunked land-mobile radio }
  - { label: Standards body, value: "Motorola (proprietary)" }
  - { label: Introduced, value: "1980s" }
  - { label: Access, value: "FDMA (channel per call)" }
  - { label: Channel spacing, value: 25 / 12.5 kHz }
  - { label: Modulation, value: "Analog FM voice; 3600 bps control" }
  - { label: Vocoder, value: "None (analog voice)" }
  - { label: GopherTrunk support, value: See Status }
see_also: [motorola-type-ii, control-channel, trunked-radio, simulcast, frequency-modulation, channel-grant]
cite_urls:
  - https://en.wikipedia.org/wiki/Motorola_Type_II
  - https://en.wikipedia.org/wiki/Trunked_radio_system
---

**SmartNet** and its multi-site successor **SmartZone** are Motorola's family of
**analog trunked-radio** systems. They carry ordinary
[FM](/reference/frequency-modulation/) voice on the traffic channels but coordinate
access with a **dedicated 3600-baud digital [control channel](/reference/control-channel/)**
that continuously assigns callers to free frequencies — the hallmark of
[Motorola Type II](/reference/motorola-type-ii/) trunking.[^type2][^trunk]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A dedicated control channel streams 3600-baud data that grants callers onto separate analog FM voice channels." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="sn_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" stroke-width="1.1" font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="30" y="55" width="120" height="40" fill="currentColor" fill-opacity="0.22"/><text x="90" y="79">control</text><text x="90" y="48">3600 bps data</text>
    <rect x="250" y="20" width="120" height="30" fill="none"/><text x="310" y="39">voice call A</text>
    <rect x="250" y="60" width="120" height="30" fill="none"/><text x="310" y="79">voice call B</text>
    <rect x="250" y="100" width="120" height="30" fill="none"/><text x="310" y="119">voice call C</text>
  </g>
  <g stroke="currentColor" stroke-width="1"><line x1="150" y1="70" x2="250" y2="35" marker-end="url(#sn_ar)"/><line x1="150" y1="75" x2="250" y2="75" marker-end="url(#sn_ar)"/><line x1="150" y1="80" x2="250" y2="115" marker-end="url(#sn_ar)"/></g>
  <text x="200" y="147" text-anchor="middle" font-size="8" fill="currentColor">control channel grants each call a free FM voice frequency</text>
</svg>
<figcaption>SmartNet/SmartZone keep one channel as a data control channel that hands callers onto separate analog FM voice channels.</figcaption>
</figure>

## Overview

SmartNet is Motorola's classic single-site trunking product; SmartZone extends it to
wide-area operation by networking multiple sites so a subscriber can roam and be tracked
across a region, and OmniLink networks several SmartZone systems together. All of them
share the same air interface: a fast digital control channel plus analog FM voice.
Systems are commonly identified by their **Type II** signalling, which uses fleet/subfleet
talkgroup addressing rather than the older fixed Type I fleetmaps.

## Technical characteristics

| Property | Value |
|----------|-------|
| Control channel | Dedicated, 3600 bps digital |
| Voice | Analog FM |
| Addressing | Type II talkgroups (some legacy Type I) |
| Channel | 25 kHz (also 12.5 kHz refarmed) |
| Bands | VHF, UHF, 700/800/900 MHz |
| Multi-site | SmartZone / OmniLink networking |
| Simulcast | Common on wide-area sites |

Because voice is analog, anyone monitoring a granted frequency hears clear audio unless
the system adds separate voice inversion or DES/DVP scrambling. Many wide-area SmartZone
systems use [simulcast](/reference/simulcast/), transmitting the same channel from
several towers on one frequency.

## History

Motorola introduced SmartNet in the 1980s as an evolution of its earlier trunking, with
SmartZone and OmniLink following to serve statewide and metropolitan agencies through the
1990s. These systems carried a large share of US public-safety and business trunked
traffic before P25 digital migration, and many remained in service well into the 2000s
and beyond.

## Deployment

SmartNet, SmartZone, and OmniLink systems were deployed extensively by police, fire,
utilities, transit, and large enterprises across North America. Although many agencies
have migrated to P25, a substantial number of analog Motorola trunked systems remain on
the air, especially for business and secondary users.

## Decoding it with GopherTrunk

GopherTrunk can demodulate the 3600-baud control channel and follow Type II grants to
tune the analog FM voice channels; the analog voice itself is then straightforward FM
audio. Multi-site roaming, banding plans, and simulcast handling vary by system, so the
precise level of SmartNet/SmartZone tracking is best confirmed on the
[Status](/status.html) page. Voice-inversion or DES scrambling is not defeated — GopherTrunk
handles clear and known-key traffic only.

## Sources

[^type2]: [Motorola Type II](https://en.wikipedia.org/wiki/Motorola_Type_II) — Wikipedia, for the Type II trunking signalling, the 3600-baud control channel, and SmartNet/SmartZone product naming.
[^trunk]: [Trunked radio system](https://en.wikipedia.org/wiki/Trunked_radio_system) — Wikipedia, for the control-channel-plus-voice-channel trunking model these systems implement.
