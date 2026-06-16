---
slug: dmr
title: Digital Mobile Radio (DMR)
entry_type: protocol
category: protocols
description: Digital Mobile Radio (DMR) is an ETSI open standard for digital two-way radio using two-slot TDMA in a 12.5 kHz channel with the AMBE+2 vocoder, defined in three tiers.
keywords: DMR, Digital Mobile Radio, ETSI, two-slot TDMA, AMBE+2, Tier I II III, MOTOTRBO
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
see_also: [dmr-tier-1, dmr-tier-2, dmr-tier-3, ambe-plus-2, tdma, etsi, control-channel]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/protocol-landscape/ }
  - { title: "What is trunked radio?", url: /learn/what-is-trunking/ }
external:
  - { title: "Digital mobile radio (Wikipedia)", url: https://en.wikipedia.org/wiki/Digital_mobile_radio }
---

**Digital Mobile Radio** (**DMR**) is an open digital two-way-radio standard from
[ETSI](/reference/etsi/), widely used in business, commercial, and amateur radio. It
places **two timeslots** ([TDMA](/reference/tdma/)) in a 12.5 kHz channel using
[4FSK](/reference/frequency-shift-keying/) modulation and the
[AMBE+2](/reference/ambe-plus-2/) vocoder.

## Overview

DMR's low cost and ETSI openness made it ubiquitous outside public safety. It is
defined in three tiers of increasing capability:
[Tier I](/reference/dmr-tier-1/) (licence-free), [Tier II](/reference/dmr-tier-2/)
(licensed conventional), and [Tier III](/reference/dmr-tier-3/) (trunked with a
[control channel](/reference/control-channel/)). Motorola's MOTOTRBO is the
best-known commercial implementation.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | Two-slot TDMA |
| Channel | 12.5 kHz |
| Modulation | 4FSK, 4800 [baud](/reference/symbol-rate/) (9600 bps) |
| Vocoder | AMBE+2 |
| Error correction | Golay, Hamming, BPTC, Reed–Solomon (by burst) |

Two-slot TDMA effectively yields 6.25 kHz-equivalent efficiency, letting two calls
share one frequency.

## History

ETSI published the DMR standard in 2005, with commercial radios following soon after.
Amateur DMR grew rapidly in the 2010s via networked repeaters and hotspots.

## Deployment

DMR dominates commercial/business radio and is popular with amateurs. Trunked
deployments use Tier III; most conventional business and ham systems are Tier II.

## Decoding it with GopherTrunk

GopherTrunk decodes both DMR timeslots, follows Tier III trunking, and renders AMBE+2
audio. Optional known-key [DMR encryption](/dmr-encryption.html) handling is
documented separately. See [Status](/status.html).
