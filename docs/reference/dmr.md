---
slug: dmr
title: Digital Mobile Radio (DMR)
entry_type: protocol
category: land-mobile-trunking
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
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
---

**Digital Mobile Radio** (**DMR**) is an open digital two-way-radio standard from
[ETSI](/reference/etsi/), widely used in business, commercial, and amateur radio. It
places **two timeslots** ([TDMA](/reference/tdma/)) in a 12.5 kHz channel using
[4FSK](/reference/frequency-shift-keying/) modulation and the
[AMBE+2](/reference/ambe-plus-2/) vocoder.[^wiki]

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

ETSI published the DMR standard in 2005, with commercial radios following soon after.[^wiki]
Amateur DMR grew rapidly in the 2010s via networked repeaters and hotspots.

## Deployment

DMR dominates commercial/business radio and is popular with amateurs. Trunked
deployments use Tier III; most conventional business and ham systems are Tier II.

## Decoding it with GopherTrunk

GopherTrunk decodes both DMR timeslots, follows Tier III trunking, and renders AMBE+2
audio. Optional known-key [DMR encryption](/dmr-encryption.html) handling is
documented separately. See [Status](/status.html).

## Sources

[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, for the ETSI DMR standard, its two-slot TDMA air interface, the tier structure, and the AMBE+2 vocoder.
