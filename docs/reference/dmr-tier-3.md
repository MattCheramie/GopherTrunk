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
  - { title: "What is trunked radio?", url: /learn/what-is-trunking/ }
  - { title: "The digital protocol landscape", url: /learn/protocol-landscape/ }
external:
  - { title: "Digital mobile radio (Wikipedia)", url: https://en.wikipedia.org/wiki/Digital_mobile_radio }
---

**DMR Tier III** is the **trunked** tier of the [DMR](/reference/dmr/) standard. It
adds a [control channel](/reference/control-channel/) and trunking signalling so many
[talkgroups](/reference/talkgroup/) can share a pool of two-slot
[TDMA](/reference/tdma/) channels, assigned on demand.

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
