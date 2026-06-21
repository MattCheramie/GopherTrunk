---
slug: ltr
title: LTR (Logic Trunked Radio)
entry_type: protocol
category: protocols
description: LTR (Logic Trunked Radio) is a simple distributed trunking protocol by E.F. Johnson with no dedicated control channel — signalling is embedded subaudibly on each channel.
keywords: LTR, Logic Trunked Radio, E.F. Johnson, distributed trunking, subaudible signalling, business radio
aka: [LTR, Logic Trunked Radio]
autolink: true
infobox:
  - { label: Type, value: Analog trunked radio }
  - { label: Developer, value: E.F. Johnson }
  - { label: Access, value: FDMA, distributed (no dedicated control channel) }
  - { label: Signalling, value: Subaudible LCN data on each channel }
  - { label: Voice, value: Analog FM }
  - { label: GopherTrunk support, value: See Status }
see_also: [trunked-radio, control-channel, motorola-type-ii, edacs, mpt-1327]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Logic_Trunked_Radio
---

**LTR** (**Logic Trunked Radio**) is a simple, low-cost trunking protocol from
**E.F. Johnson**. Unlike systems with a dedicated [control channel](/reference/control-channel/),
LTR is **distributed**: trunking data rides **subaudibly on each voice channel**, so
every channel carries its own low-speed signalling.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 380 130" role="img" aria-label="Several LTR channels, each carrying analog voice plus its own embedded subaudible signalling — no dedicated control channel." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1" fill="none"><rect x="40" y="35" width="300" height="22"/><rect x="40" y="63" width="300" height="22"/><rect x="40" y="91" width="300" height="22"/></g>
  <g font-size="8" fill="currentColor"><text x="50" y="50">voice + subaudible data</text><text x="50" y="78">voice + subaudible data</text><text x="50" y="106">voice + subaudible data</text></g>
  <text x="190" y="128" text-anchor="middle" font-size="8" fill="currentColor">distributed — no dedicated control channel</text>
</svg>
<figcaption>LTR is distributed trunking: each channel carries its own subaudible signalling, with no separate control channel.</figcaption>
</figure>

## Overview

Because there is no separate control channel, radios monitor the embedded data
(logical channel numbers and home-repeater info) to follow calls. This makes LTR
cheap to deploy but trickier to monitor than control-channel systems.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | [FDMA](/reference/fdma/), distributed |
| Signalling | Subaudible data on each channel |
| Voice | Analog FM |

## History

Introduced by E.F. Johnson and widely used for business/SMR trunking; variants
include LTR-Net and PassPort.[^wiki]

## Deployment

Common in commercial/business shared systems (taxi, delivery, utilities), especially
in North America.

## Decoding it with GopherTrunk

See [Status](/status.html) for GopherTrunk's handling of LTR's subaudible signalling.

## Sources

[^wiki]: [Logic Trunked Radio](https://en.wikipedia.org/wiki/Logic_Trunked_Radio) — Wikipedia, for the E.F. Johnson LTR distributed trunking scheme and its subaudible per-channel signalling.
