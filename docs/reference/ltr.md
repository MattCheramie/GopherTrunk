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
  - { title: "The digital protocol landscape", url: /learn/protocol-landscape/ }
external:
  - { title: "Logic Trunked Radio (Wikipedia)", url: https://en.wikipedia.org/wiki/Logic_Trunked_Radio }
---

**LTR** (**Logic Trunked Radio**) is a simple, low-cost trunking protocol from
**E.F. Johnson**. Unlike systems with a dedicated [control channel](/reference/control-channel/),
LTR is **distributed**: trunking data rides **subaudibly on each voice channel**, so
every channel carries its own low-speed signalling.

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
include LTR-Net and PassPort.

## Deployment

Common in commercial/business shared systems (taxi, delivery, utilities), especially
in North America.

## Decoding it with GopherTrunk

See [Status](/status.html) for GopherTrunk's handling of LTR's subaudible signalling.
