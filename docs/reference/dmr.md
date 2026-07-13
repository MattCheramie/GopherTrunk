---
slug: dmr
title: Digital Mobile Radio (DMR)
entry_type: protocol
category: land-mobile-trunking
description: Digital Mobile Radio (DMR) is an ETSI open standard for digital two-way radio using two-slot TDMA in a 12.5 kHz channel with the AMBE+2 vocoder, defined in three tiers.
keywords: DMR, Digital Mobile Radio, ETSI, TS 102 361, two-slot TDMA, 4FSK, AMBE+2, Tier I II III, MOTOTRBO, color code, CSBK, Capacity Plus
aka: [DMR, Digital Mobile Radio]
autolink: true
infobox:
  - { label: Type, value: Digital land-mobile radio }
  - { label: Standards body, value: ETSI }
  - { label: Introduced, value: "2005" }
  - { label: Access, value: TDMA (2 slots) }
  - { label: Channel spacing, value: 12.5 kHz }
  - { label: Modulation, value: 4FSK (4800 baud, 9600 bps) }
  - { label: Vocoder, value: AMBE+2 }
  - { label: Tiers, value: I (licence-free), II (conventional), III (trunked) }
  - { label: GopherTrunk support, value: Decoded }
see_also: [dmr-tier-1, dmr-tier-2, dmr-tier-3, four-fsk, ambe-plus-2, color-code, capacity-plus, connect-plus, tdma, etsi, control-channel]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
  - https://www.dmrassociation.org/
---

**Digital Mobile Radio** (**DMR**) is an open digital two-way-radio standard from
[ETSI](/reference/etsi/) (published as TS 102 361), widely used in business,
commercial, utility, and amateur radio. It places **two timeslots**
([TDMA](/reference/tdma/)) in a 12.5 kHz channel using
[4FSK](/reference/four-fsk/) modulation at 4800 baud and the
[AMBE+2](/reference/ambe-plus-2/) vocoder.[^wiki][^dmra]

<figure class="figure" markdown="0">
<svg viewBox="0 0 380 140" role="img" aria-label="Two TDMA slots in a 12.5 kHz DMR channel." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="105" x2="360" y2="105" stroke="currentColor" stroke-opacity="0.4" marker-end="url(#ta_dmr)"/>
  <text x="195" y="128" text-anchor="middle" font-size="9" fill="currentColor">time → · one 12.5 kHz channel, 2 slots</text>
  <g stroke="currentColor" stroke-width="1.1"><rect x="40" y="40" width="52" height="50" fill="currentColor" fill-opacity="0.22"/><rect x="92" y="40" width="52" height="50" fill="none"/><rect x="144" y="40" width="52" height="50" fill="currentColor" fill-opacity="0.22"/><rect x="196" y="40" width="52" height="50" fill="none"/><rect x="248" y="40" width="52" height="50" fill="currentColor" fill-opacity="0.22"/><rect x="300" y="40" width="52" height="50" fill="none"/></g><g font-size="9" fill="currentColor" text-anchor="middle"><text x="66" y="69">1</text><text x="118" y="69">2</text><text x="170" y="69">1</text><text x="222" y="69">2</text><text x="274" y="69">1</text><text x="326" y="69">2</text></g>
  <defs><marker id="ta_dmr" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>DMR divides each 12.5 kHz channel into two TDMA timeslots.</figcaption>
</figure>

## Overview

DMR's low cost and ETSI openness made it ubiquitous outside public safety. It is
defined in three tiers of increasing capability:
[Tier I](/reference/dmr-tier-1/) (licence-free), [Tier II](/reference/dmr-tier-2/)
(licensed conventional), and [Tier III](/reference/dmr-tier-3/) (trunked with a
[control channel](/reference/control-channel/)). Each carrier is shared by two
independent conversations in alternating 30 ms bursts, and every call is tagged with
a [color code](/reference/color-code/) (0–15) that acts as a digital squelch so that
co-channel systems do not interfere. Motorola's MOTOTRBO is the best-known commercial
implementation, and vendors layer proprietary trunking such as
[Capacity Plus](/reference/capacity-plus/) and Hytera's
[Connect Plus](/reference/connect-plus/) on top of the DMR air interface.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | Two-slot TDMA |
| Channel | 12.5 kHz |
| Modulation | [4FSK](/reference/four-fsk/), 4800 [baud](/reference/symbol-rate/) (9600 bps) |
| Vocoder | [AMBE+2](/reference/ambe-plus-2/) (half-rate) |
| Squelch / identity | [color code](/reference/color-code/) (4-bit), slot, talkgroup, source ID |
| Error correction | Golay, Hamming, [BPTC](/reference/bptc/), Reed–Solomon (by burst) |
| Trunking signalling | CSBK (control signalling block), Tier III |

Two-slot TDMA effectively yields 6.25 kHz-equivalent efficiency, letting two calls
share one frequency. The 4FSK symbol stream carries a repeating burst structure:
each timeslot alternates voice frames with embedded signalling (color code, link
control, and — on trunked systems — CSBK messages), so the identity of a call can be
read directly off the air without a separate database lookup.

## History

ETSI published the DMR standard (TS 102 361, parts 1–4) in 2005, with commercial
radios following soon after; the [DMR Association](/reference/dmr-association/) was
formed by manufacturers to promote interoperability and run compliance testing.[^wiki][^dmra]
Amateur DMR grew rapidly in the 2010s via internet-linked repeaters and low-cost
personal hotspots that bridge a handheld to worldwide talkgroup networks.

## Deployment

DMR dominates commercial and business radio worldwide and is popular with amateurs.
Trunked deployments use [Tier III](/reference/dmr-tier-3/) (or a vendor mode such as
[Capacity Plus](/reference/capacity-plus/)); most conventional business and ham
systems are [Tier II](/reference/dmr-tier-2/). Because the standard is open and the
radios are inexpensive, DMR often competes directly with [P25](/reference/project-25/)
and [TETRA](/reference/tetra/) on cost.

## Decoding it with GopherTrunk

GopherTrunk decodes both DMR timeslots, reads the [color code](/reference/color-code/)
and per-burst link control, follows Tier III (and vendor Capacity Plus / Connect
Plus) trunking signalling, and renders [AMBE+2](/reference/ambe-plus-2/) audio. A
single dongle channelizes a cluster of nearby repeaters at once. Optional known-key
[DMR encryption](/dmr-encryption.html) handling is documented separately; keyed
proprietary encryption is not defeated. See [Status](/status.html).

## Sources

[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, for the ETSI DMR standard, its two-slot TDMA air interface, the tier structure, the color code, and the AMBE+2 vocoder.
[^dmra]: [DMR Association](https://www.dmrassociation.org/) — the manufacturer association behind DMR, for the TS 102 361 standard basis, interoperability process, and tier definitions.
