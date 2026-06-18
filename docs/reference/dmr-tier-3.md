---
slug: dmr-tier-3
title: DMR Tier III
entry_type: protocol
category: protocols
description: DMR Tier III is the trunked tier of the ETSI DMR standard, adding a control channel and signalling (CSBK) so many talkgroups share a pool of two-slot TDMA channels.
keywords: DMR Tier III, DMR Tier 3, trunked DMR, Capacity Plus, control channel, CSBK, talkgroup
aka: [DMR Tier III, DMR Tier 3]
autolink: true
infobox:
  - { label: Type, value: Digital trunked radio }
  - { label: Part of, value: DMR (ETSI) }
  - { label: Licensing, value: Licensed trunked }
  - { label: Access, value: Two-slot TDMA + control channel }
  - { label: Channel spacing, value: 12.5 kHz }
  - { label: Signalling, value: CSBK (control signalling block) }
  - { label: Vocoder, value: AMBE+2 }
  - { label: GopherTrunk support, value: Decoded }
see_also: [dmr, dmr-tier-2, trunked-radio, control-channel, talkgroup, channel-grant, tdma]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
external:
  - { title: "Digital mobile radio (Wikipedia)", url: https://en.wikipedia.org/wiki/Digital_mobile_radio }
---

**DMR Tier III** is the **trunked** tier of the [DMR](/reference/dmr/) standard. It
adds a [control channel](/reference/control-channel/) and trunking signalling so many
[talkgroups](/reference/talkgroup/) can share a pool of two-slot
[TDMA](/reference/tdma/) channels, assigned on demand.

<figure class="figure" markdown="0">
<svg viewBox="0 0 380 150" role="img" aria-label="A DMR Tier III control channel assigning two-slot TDMA traffic channels from a pool." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="20" width="300" height="26" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="190" y="37" text-anchor="middle" font-size="9" fill="currentColor">control channel (CSBK)</text>
  <g stroke="currentColor" stroke-width="1.1"><rect x="40" y="80" width="90" height="40" fill="none"/><line x1="85" y1="80" x2="85" y2="120"/><rect x="150" y="80" width="90" height="40" fill="none"/><line x1="195" y1="80" x2="195" y2="120"/><rect x="260" y="80" width="80" height="40" fill="currentColor" fill-opacity="0.18"/><line x1="300" y1="80" x2="300" y2="120"/></g>
  <text x="190" y="138" text-anchor="middle" font-size="8.5" fill="currentColor">two-slot TDMA traffic pool, assigned on demand</text>
  <line x1="190" y1="46" x2="300" y2="78" stroke="currentColor" stroke-dasharray="3 3" marker-end="url(#t3ar)"/>
  <defs><marker id="t3ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>DMR Tier III adds a control channel that assigns two-slot TDMA traffic channels — trunked DMR.</figcaption>
</figure>

## Overview

Where [Tier II](/reference/dmr-tier-2/) is conventional, Tier III is a full
[trunked-radio](/reference/trunked-radio/) system. Radios register
([affiliate](/reference/affiliation/)) and request calls over the control channel,
which issues [channel grants](/reference/channel-grant/) pointing them to a traffic
channel and slot. (Motorola's proprietary Capacity Plus offers similar trunking
outside the strict Tier III standard.)

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | Two-slot TDMA + dedicated control channel |
| Signalling | CSBK control messages |
| Channel | 12.5 kHz |
| Vocoder | [AMBE+2](/reference/ambe-plus-2/) |

## History

Tier III trunking was standardised by [ETSI](/reference/etsi/) to give DMR a
multi-site, high-capacity option competing with P25 and TETRA in the commercial
market.

## Deployment

Used by larger commercial, utility, and transport operators needing trunked capacity
at lower cost than [TETRA](/reference/tetra/) or [P25](/reference/project-25/).

## Decoding it with GopherTrunk

GopherTrunk locks the Tier III control channel, follows CSBK channel grants to the
assigned channel/slot, and decodes the voice. See [Status](/status.html).
