---
slug: ltr
title: LTR (Logic Trunked Radio)
entry_type: protocol
category: land-mobile-trunking
description: LTR (Logic Trunked Radio) is a simple distributed trunking protocol by E.F. Johnson with no dedicated control channel — signalling is embedded subaudibly on each channel.
keywords: LTR, Logic Trunked Radio, E.F. Johnson, distributed trunking, subaudible signalling, business radio, LTR-Net, PassPort, home repeater, 300 baud
aka: [LTR, Logic Trunked Radio]
autolink: true
infobox:
  - { label: Type, value: Analog trunked radio }
  - { label: Developer, value: E.F. Johnson }
  - { label: Access, value: FDMA, distributed (no dedicated control channel) }
  - { label: Signalling, value: Subaudible LCN data on each channel }
  - { label: Voice, value: Analog FM }
  - { label: GopherTrunk support, value: See Status }
see_also: [fdma, trunked-radio, control-channel, motorola-type-ii, edacs, mpt-1327]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Logic_Trunked_Radio
  - https://wiki.radioreference.com/index.php/Logic_Trunked_Radio_(LTR)
---

**LTR** (**Logic Trunked Radio**) is a simple, low-cost trunking protocol from
**E.F. Johnson**. Unlike systems with a dedicated [control channel](/reference/control-channel/),
LTR is **distributed**: the trunking data rides **subaudibly on each voice channel**, so
every repeater carries its own low-speed signalling rather than one channel coordinating the
rest.[^wiki] This makes LTR cheap to build but distinctive to monitor, since there is no
single channel to watch for system activity.

<figure class="figure" markdown="0">
<svg viewBox="0 0 380 130" role="img" aria-label="Several LTR channels, each carrying analog voice plus its own embedded subaudible signalling, with no dedicated control channel." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1" fill="none"><rect x="40" y="35" width="300" height="22"/><rect x="40" y="63" width="300" height="22"/><rect x="40" y="91" width="300" height="22"/></g>
  <g font-size="8" fill="currentColor"><text x="50" y="50">voice + subaudible data</text><text x="50" y="78">voice + subaudible data</text><text x="50" y="106">voice + subaudible data</text></g>
  <text x="190" y="128" text-anchor="middle" font-size="8" fill="currentColor">distributed — no dedicated control channel</text>
</svg>
<figcaption>LTR is distributed trunking: each channel carries its own subaudible signalling, with no separate control channel.</figcaption>
</figure>

## Overview

Because there is no separate control channel, an LTR radio follows calls by reading a
low-rate data burst embedded **below the audio passband** (a subaudible ~300 bit/s stream)
on each channel. That data identifies the **home repeater** (the channel a given group is
assigned to), the **logical channel number** in use, and a **group/goto** field telling
radios which channel to move to for the next transmission. Radios in a group idle on their
home repeater's frequency; when a call starts, the subaudible data steers them to the active
channel and back. This decentralised design means each repeater in a shared system operates
semi-independently, which keeps hardware cheap and avoids a single point of failure, at the
cost of the self-describing convenience a dedicated control channel provides. Later variants
extended the idea: **LTR-Net** adds networking and richer features, and **PassPort**
(from Trident/SmartTrunk lineage) layers on wide-area roaming and registration.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | [FDMA](/reference/fdma/), distributed |
| Signalling | Subaudible (~300 bit/s) data burst on every channel |
| Data fields | Home repeater, logical channel number, group/goto |
| Voice | Analog FM |
| Variants | Standard LTR, LTR-Net, PassPort |
| Bands | VHF, UHF, 800/900 MHz |

## History

LTR was introduced by **E.F. Johnson** and became a mainstay of low-cost business and SMR
(Specialised Mobile Radio) trunking.[^wiki][^rr] Its distributed philosophy set it apart from
the control-channel systems of [Motorola](/reference/motorola-type-ii/) and
[EDACS](/reference/edacs/), targeting operators who wanted trunking economy over feature
richness. Over time the family grew to include LTR-Net and the PassPort roaming variant, and
LTR-style subaudible signalling influenced other economy trunking schemes.

## Deployment

LTR is common in **commercial and business shared systems** — taxi and delivery fleets,
utilities, tow companies, and community-repeater operators — especially across North America.
Many of these systems are still on the air because they are inexpensive to maintain and meet
modest fleet needs without a migration to digital.

## Decoding it with GopherTrunk

Monitoring LTR is unusual because there is no control channel to lock to: a decoder must
demodulate the **subaudible data on each voice channel** to learn the home-repeater and
goto information, then piece together group activity across channels. Voice is plain analog
FM, so no vocoder is required — the challenge is recovering the low-rate embedded signalling
cleanly. See the [Status](/status.html) page for GopherTrunk's handling of LTR's subaudible
signalling and any supported variants.

## Sources

[^wiki]: [Logic Trunked Radio](https://en.wikipedia.org/wiki/Logic_Trunked_Radio) — Wikipedia, for the E.F. Johnson LTR distributed trunking scheme and its subaudible per-channel signalling.
[^rr]: [Logic Trunked Radio (LTR)](https://wiki.radioreference.com/index.php/Logic_Trunked_Radio_(LTR)) — RadioReference Wiki, for home-repeater/goto signalling, logical channel numbering, and the LTR-Net and PassPort variants.
