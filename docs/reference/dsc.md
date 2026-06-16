---
slug: dsc
title: Digital Selective Calling (DSC)
entry_type: protocol
category: protocols
description: DSC (Digital Selective Calling) is a maritime calling and distress-alerting protocol, sending FSK data bursts on VHF channel 70 and HF to address specific stations or signal emergencies.
keywords: DSC, Digital Selective Calling, GMDSS, VHF channel 70, distress alert, MMSI, FFSK, maritime
aka: [DSC]
autolink: true
infobox:
  - { label: Type, value: Maritime calling / distress alerting }
  - { label: Standards body, value: ITU (GMDSS) }
  - { label: Band, value: VHF Ch 70 (156.525 MHz) + HF }
  - { label: Modulation, value: FSK data burst }
  - { label: Addressing, value: MMSI (selective calling) }
  - { label: Error correction, value: Symbol repetition + parity }
  - { label: GopherTrunk support, value: Decoded }
see_also: [ais, ffsk, frequency-shift-keying, itu]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/other-signals/ }
external:
  - { title: "Digital selective calling (Wikipedia)", url: https://en.wikipedia.org/wiki/Digital_selective_calling }
  - { title: "GopherTrunk DSC decoder", url: /dsc.html }
---

**DSC** (**Digital Selective Calling**) is a maritime protocol for **calling specific
stations and broadcasting distress alerts**. Part of the Global Maritime Distress and
Safety System (GMDSS), it sends short [FSK](/reference/frequency-shift-keying/) data
bursts on **VHF channel 70** (156.525 MHz) and on HF.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="A short Digital Selective Calling burst carrying a call type and the sender's maritime identity." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1"><rect x="40" y="40" width="70" height="28" fill="currentColor" fill-opacity="0.12"/><rect x="110" y="40" width="120" height="28" fill="currentColor" fill-opacity="0.22"/><rect x="230" y="40" width="120" height="28" fill="none"/><rect x="350" y="40" width="70" height="28" fill="none"/></g>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="75" y="58">dotting</text><text x="170" y="58">format / MMSI</text><text x="290" y="58">distress / call</text><text x="385" y="58">ECC</text></g>
  <text x="230" y="88" text-anchor="middle" font-size="8" fill="currentColor">FFSK · VHF Ch 70 (and HF)</text>
</svg>
<figcaption>DSC sends short FFSK bursts on VHF channel 70 for distress and routine calling, carrying the sender's MMSI.</figcaption>
</figure>

## Overview

A DSC message carries the sender's and (for selective calls) recipient's MMSI, a
category (routine, safety, urgency, distress), and optional position. A distress alert
automatically conveys identity and, if interfaced, GPS position. DSC complements
[AIS](/reference/ais/) on the safety side.

## Technical characteristics

| Property | Value |
|----------|-------|
| Band | VHF Ch 70 + HF DSC frequencies |
| Modulation | FSK burst |
| Addressing | MMSI |
| Error control | Symbol repetition + parity |

## History

Standardised by the [ITU](/reference/itu/) and mandated within GMDSS from the 1990s to
modernise distress alerting beyond voice calling.

## Deployment

SOLAS and recreational vessels, coast stations, and rescue coordination centres.

## Decoding it with GopherTrunk

GopherTrunk demodulates the FSK, applies the symbol-repetition error control, and
decodes DSC messages. See the [DSC decoder](/dsc.html) page.
