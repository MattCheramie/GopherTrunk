---
slug: iden
title: Integrated Digital Enhanced Network (iDEN)
entry_type: protocol
category: land-mobile-trunking
description: iDEN is Motorola's proprietary TDMA land-mobile and cellular technology using M16QAM modulation to combine dispatch, telephone, and data on 25 kHz channels.
keywords: iDEN, Integrated Digital Enhanced Network, Motorola, M16QAM, TDMA, push-to-talk, Nextel, Direct Connect, VSELP, 25 kHz
aka: [iDEN, "Integrated Digital Enhanced Network"]
autolink: true
infobox:
  - { label: Type, value: Proprietary digital trunked + cellular }
  - { label: Standards body, value: "Motorola (proprietary)" }
  - { label: Introduced, value: "1994" }
  - { label: Access, value: TDMA (6 slots per carrier) }
  - { label: Channel spacing, value: 25 kHz }
  - { label: Modulation, value: M16QAM (16-QAM, 64 kbps) }
  - { label: Vocoder, value: "VSELP (later AMBE/EVRC variants)" }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [tdma, quadrature-amplitude-modulation, motorola-type-ii, gsm, vocoder, trunked-radio]
cite_urls:
  - https://en.wikipedia.org/wiki/Integrated_Digital_Enhanced_Network
  - https://en.wikipedia.org/wiki/Quadrature_amplitude_modulation
---

**iDEN** (**Integrated Digital Enhanced Network**) is Motorola's proprietary digital
radio technology that combined trunked push-to-talk dispatch, cellular telephony, short
messaging, and data on a single network. It uses **[TDMA](/reference/tdma/)** with six
time slots per carrier and a proprietary **M16QAM** —
[16-QAM](/reference/quadrature-amplitude-modulation/) — modulation to pack roughly
64 kbps into a 25 kHz channel.[^wiki][^qam]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="iDEN uses a 16-point QAM constellation and divides each 25 kHz carrier into six TDMA slots." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="0.8" stroke-opacity="0.4"><line x1="30" y1="85" x2="180" y2="85"/><line x1="105" y1="15" x2="105" y2="155"/></g>
  <g fill="currentColor">
    <circle cx="60" cy="40" r="2.5"/><circle cx="90" cy="40" r="2.5"/><circle cx="120" cy="40" r="2.5"/><circle cx="150" cy="40" r="2.5"/>
    <circle cx="60" cy="70" r="2.5"/><circle cx="90" cy="70" r="2.5"/><circle cx="120" cy="70" r="2.5"/><circle cx="150" cy="70" r="2.5"/>
    <circle cx="60" cy="100" r="2.5"/><circle cx="90" cy="100" r="2.5"/><circle cx="120" cy="100" r="2.5"/><circle cx="150" cy="100" r="2.5"/>
    <circle cx="60" cy="130" r="2.5"/><circle cx="90" cy="130" r="2.5"/><circle cx="120" cy="130" r="2.5"/><circle cx="150" cy="130" r="2.5"/>
  </g>
  <text x="105" y="168" text-anchor="middle" font-size="8.5" fill="currentColor">16-QAM: 4 bits per symbol</text>
  <g stroke="currentColor" stroke-width="1.1" font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="250" y="60" width="30" height="40" fill="currentColor" fill-opacity="0.22"/><rect x="280" y="60" width="30" height="40" fill="none"/><rect x="310" y="60" width="30" height="40" fill="currentColor" fill-opacity="0.22"/><rect x="340" y="60" width="30" height="40" fill="none"/><rect x="370" y="60" width="30" height="40" fill="currentColor" fill-opacity="0.22"/><rect x="400" y="60" width="30" height="40" fill="none"/>
  </g>
  <text x="325" y="118" text-anchor="middle" font-size="8" fill="currentColor">6 TDMA slots per 25 kHz carrier</text>
  <text x="325" y="45" text-anchor="middle" font-size="8" fill="currentColor">time →</text>
</svg>
<figcaption>iDEN packs four bits per symbol with 16-QAM and time-shares each 25 kHz carrier among six users.</figcaption>
</figure>

## Overview

iDEN was engineered to give a single subscriber unit both cellular-phone service and
fast, Nextel-style "Direct Connect" walkie-talkie dispatch. Its use of a
higher-order quadrature-amplitude modulation was unusual for land-mobile radio, where
constant-envelope schemes dominate: M16QAM trades some resilience to noise and amplifier
non-linearity for a much higher bit rate, which iDEN needed to carry digitised voice for
six simultaneous users plus signalling in one 25 kHz slot.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | TDMA, 6 slots per carrier |
| Channel | 25 kHz |
| Modulation | M16QAM (16-QAM), 4 bits/symbol |
| Gross rate | ~64 kbps per carrier |
| Vocoder | VSELP originally; later AMBE-family half-rate |
| Bands | 800 MHz (SMR), 900 MHz, 1.5 GHz |
| Services | Dispatch PTT, interconnect, SMS, packet data |

Because 16-QAM carries information in both amplitude and phase, iDEN receivers need a
linear front end and accurate equalisation — a sharper contrast with the frequency-shift
and constant-envelope waveforms used by most other PMR systems.

## History

Motorola launched iDEN in 1994, and it became the backbone of Nextel Communications in
the United States, prized for near-instant nationwide push-to-talk. Networks also ran in
Latin America, the Middle East, and Asia. After Sprint acquired Nextel, the US iDEN
network was progressively shut down (completed 2013) in favour of CDMA and LTE, though
some international iDEN networks persisted longer.

## Deployment

At its peak iDEN served tens of millions of subscribers, dominated by fleet, construction,
and field-service users who valued its dispatch feature. Today most iDEN networks have
been decommissioned, making live iDEN signals uncommon compared with the [Motorola
Type II](/reference/motorola-type-ii/) analog trunking and P25 systems that remain widespread.

## Decoding it with GopherTrunk

iDEN is **not decoded** by GopherTrunk. Its proprietary M16QAM physical layer, six-slot
framing, and closed vocoder fall outside GopherTrunk's C4FM/π-4-DQPSK decode chain, and
its cellular-style signalling is a different problem class from scanner trunk-following.
It is included in this field guide for historical and identification context. GopherTrunk
decodes clear and known-key traffic only, and does not target iDEN.

## Sources

[^wiki]: [Integrated Digital Enhanced Network](https://en.wikipedia.org/wiki/Integrated_Digital_Enhanced_Network) — Wikipedia, for the iDEN architecture, six-slot TDMA, M16QAM modulation, and its Nextel dispatch heritage.
[^qam]: [Quadrature amplitude modulation](https://en.wikipedia.org/wiki/Quadrature_amplitude_modulation) — Wikipedia, for 16-QAM as the modulation carrying four bits per symbol in amplitude and phase.
