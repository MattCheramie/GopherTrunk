---
slug: capacity-plus
title: Capacity Plus
entry_type: protocol
category: land-mobile-trunking
description: Capacity Plus is Motorola's proprietary DMR trunking mode that pools channels and rotates the control signalling, adding trunked capacity to conventional DMR.
keywords: Capacity Plus, Cap Plus, Capacity Max, Motorola DMR trunking, MOTOTRBO, rest channel, single-site trunking, CSBK, color code
aka: [Capacity Plus, "Cap Plus", "Capacity Max"]
autolink: true
infobox:
  - { label: Type, value: Proprietary DMR trunking }
  - { label: Vendor, value: Motorola (MOTOTRBO) }
  - { label: Based on, value: DMR two-slot TDMA }
  - { label: Access, value: Pooled channels, rotating control (rest channel) }
  - { label: Channel spacing, value: 12.5 kHz }
  - { label: Signalling, value: CSBK (Motorola grant formats) }
  - { label: Vocoder, value: AMBE+2 }
  - { label: GopherTrunk support, value: Decoded (vendor-aware) }
see_also: [dmr, dmr-tier-3, connect-plus, rest-channel, csbk, color-code, four-fsk, trunked-radio]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
  - https://www.dmrassociation.org/
---

**Capacity Plus** is Motorola's proprietary **DMR trunking** mode (part of the
MOTOTRBO family) that pools several channels and **rotates the control signalling**
among them (see [rest channel](/reference/rest-channel/)), giving conventional
[DMR](/reference/dmr/) trunked capacity without a separate, permanently dedicated
control channel. Capacity Max is the larger multi-site successor, and Hytera's
[Connect Plus](/reference/connect-plus/) is a competing vendor equivalent.[^wiki][^dmra]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A pool of DMR channels with control rotating among them, several carrying two TDMA voice slots." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1" font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="40" y="40" width="90" height="40" fill="currentColor" fill-opacity="0.2"/><line x1="85" y1="40" x2="85" y2="80"/><text x="85" y="95">rest/control</text>
    <rect x="150" y="40" width="90" height="40" fill="none"/><line x1="195" y1="40" x2="195" y2="80"/><text x="195" y="95">voice</text>
    <rect x="260" y="40" width="90" height="40" fill="none"/><line x1="305" y1="40" x2="305" y2="80"/><text x="305" y="95">voice</text>
    <rect x="370" y="40" width="70" height="40" fill="none"/><line x1="405" y1="40" x2="405" y2="80"/><text x="405" y="95">voice</text>
  </g>
  <text x="230" y="22" text-anchor="middle" font-size="8.5" fill="currentColor">pooled two-slot DMR channels; control rotates</text>
</svg>
<figcaption>Capacity Plus pools two-slot DMR channels and moves control among them rather than dedicating one.</figcaption>
</figure>

## Overview

Standard [DMR Tier III](/reference/dmr-tier-3/) trunking permanently assigns one
channel as the [control channel](/reference/control-channel/). Capacity Plus instead
treats a group of channels as a shared pool and lets whichever channel is currently
idle carry the trunking signalling — the [rest channel](/reference/rest-channel/).
When a call is set up, radios move to a granted timeslot and the rest/control
function may hop to a different idle channel. This squeezes more usable capacity out
of a small channel count and avoids "wasting" a full channel on control alone, at the
cost of being a proprietary, Motorola-specific scheme rather than the open Tier III
standard.

## Technical characteristics

| Property | Value |
|----------|-------|
| Air interface | [DMR](/reference/dmr/) two-slot [TDMA](/reference/tdma/), [4FSK](/reference/four-fsk/) |
| Trunking | Proprietary Motorola (Capacity Plus Single-Site / Multi-Site) |
| Control | Rotating [rest channel](/reference/rest-channel/), not a fixed control channel |
| Signalling | [CSBK](/reference/csbk/) with Motorola grant formats |
| Vocoder | [AMBE+2](/reference/ambe-plus-2/) |
| Squelch / identity | [color code](/reference/color-code/), talkgroup, source ID |

Because the control signalling moves, a scanner cannot simply park on one control
frequency; it has to track the rest channel as it hops and then follow the granted
timeslot for voice.

## History

Capacity Plus grew out of Motorola's MOTOTRBO product line as an entry-level trunking
option for customers who wanted more capacity than conventional
[Tier II](/reference/dmr-tier-2/) but did not need full multi-site
[Tier III](/reference/dmr-tier-3/). Capacity Max later extended the concept to larger
multi-site networks with centralised control.[^wiki][^dmra]

## Deployment

Common on business, industrial, and campus MOTOTRBO systems where a handful of
channels must serve many users cost-effectively. It competes with open Tier III and
with Hytera's [Connect Plus](/reference/connect-plus/) in the same market segment.

## Decoding it with GopherTrunk

Following Capacity Plus means tracking the rotating rest/control channel as it hops,
reading the Motorola-format [CSBK](/reference/csbk/) grants, then decoding the granted
[timeslot](/reference/tdma/) — GopherTrunk's DMR support is vendor-aware of these
Motorola grant formats and renders the [AMBE+2](/reference/ambe-plus-2/) voice. See
[Status](/status.html).

## Sources

[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, for DMR and Motorola's proprietary Capacity Plus trunking built on the DMR air interface.
[^dmra]: [DMR Association](https://www.dmrassociation.org/) — the DMR manufacturer association, for the DMR air-interface basis on which vendor trunking modes such as Capacity Plus are layered.
