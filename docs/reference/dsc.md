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
