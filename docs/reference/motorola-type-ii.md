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
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
external:
  - { title: "Motorola Type II (Wikipedia)", url: https://en.wikipedia.org/wiki/Motorola_Type_II }
---

**Motorola Type II** is a classic **analog trunked-radio** family (SmartNet /
SmartZone) that pairs a **digital [control channel](/reference/control-channel/)**
with **analog FM** voice channels. It was the dominant trunking technology for
public-safety and business fleets before the migration to digital
[P25](/reference/project-25/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 380 140" role="img" aria-label="Motorola Type II digital control channel assigning analog voice channels." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="20" width="300" height="26" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="190" y="37" text-anchor="middle" font-size="9" fill="currentColor">digital control channel (3600 bps)</text>
  <g stroke="currentColor" stroke-width="1.1" fill="none"><rect x="40" y="80" width="90" height="34"/><rect x="150" y="80" width="90" height="34"/><rect x="260" y="80" width="80" height="34" fill="currentColor" fill-opacity="0.18"/></g>
  <text x="190" y="130" text-anchor="middle" font-size="8.5" fill="currentColor">analog FM voice channels (assigned on demand)</text>
  <line x1="190" y1="46" x2="300" y2="78" stroke="currentColor" stroke-dasharray="3 3" marker-end="url(#lg_motorola-type-ii)"/>
  <defs><marker id="lg_motorola-type-ii" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Motorola Type II pairs a digital control channel with analog FM voice channels.</figcaption>
</figure>

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
