---
slug: motorola-type-ii
title: Motorola Type II
entry_type: protocol
category: protocols
description: Motorola Type II is a classic analog trunked-radio system using a digital control channel to assign analog FM voice channels, widely deployed before the move to P25.
keywords: Motorola Type II, SmartNet, SmartZone, analog trunking, control channel, fleet, public safety legacy
aka: [Motorola Type II, SmartNet, SmartZone]
autolink: true
infobox:
  - { label: Type, value: Analog trunked radio (digital control) }
  - { label: Developer, value: Motorola }
  - { label: Era, value: 1980s–2000s }
  - { label: Access, value: FDMA with digital control channel }
  - { label: Voice, value: Analog FM }
  - { label: Control channel, value: 3600 bps digital }
  - { label: GopherTrunk support, value: Decoded }
see_also: [trunked-radio, control-channel, edacs, ltr, talkgroup, radio-id]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/what-is-trunking/ }
  - { title: "The digital protocol landscape", url: /learn/protocol-landscape/ }
external:
  - { title: "Motorola Type II (Wikipedia)", url: https://en.wikipedia.org/wiki/Motorola_Type_II }
---

**Motorola Type II** is a classic **analog trunked-radio** family (SmartNet /
SmartZone) that pairs a **digital [control channel](/reference/control-channel/)**
with **analog FM** voice channels. It was the dominant trunking technology for
public-safety and business fleets before the migration to digital
[P25](/reference/project-25/).

## Overview

The control channel (a 3600 bps data stream) issues [channel grants](/reference/channel-grant/),
pointing radios in a [talkgroup](/reference/talkgroup/) to an analog voice frequency.
This makes it a [trunked](/reference/trunked-radio/) system even though the voice
itself is analog FM. Each transmitting radio carries a [radio ID](/reference/radio-id/).

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | [FDMA](/reference/fdma/) |
| Control channel | 3600 bps digital |
| Voice | Analog FM |
| IDs | Talkgroups and radio IDs in control data |

## History

Introduced by Motorola in the 1980s (SmartNet), later extended with multi-site
SmartZone, and ubiquitous through the 1990s–2000s.

## Deployment

Many legacy public-safety and commercial systems; steadily replaced by P25, though
some remain in service.

## Decoding it with GopherTrunk

GopherTrunk decodes the Type II control channel and follows grants to the analog
voice channels. See [Status](/status.html).
