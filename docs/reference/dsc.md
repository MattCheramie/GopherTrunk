---
slug: dsc
title: Digital Selective Calling (DSC)
entry_type: protocol
category: aviation-marine
description: DSC (Digital Selective Calling) is a maritime calling and distress protocol sending FSK bursts on VHF channel 70 and HF to address stations or signal emergencies.
keywords: DSC, Digital Selective Calling, GMDSS, VHF channel 70, distress alert, MMSI, FFSK, ITU-R M.493, maritime safety, coast station
aka: [DSC]
autolink: true
infobox:
  - { label: Type, value: Maritime calling / distress alerting }
  - { label: Standards body, value: ITU-R M.493 (GMDSS) }
  - { label: Band, value: VHF Ch 70 (156.525 MHz) + MF/HF }
  - { label: Modulation, value: FFSK data burst (1300/2100 Hz) }
  - { label: Addressing, value: MMSI (selective calling) }
  - { label: Error correction, value: Symbol repetition + parity }
  - { label: GopherTrunk support, value: Decoded }
see_also: [ais, marine-vhf, epirb-406, ffsk, frequency-shift-keying, afsk, itu]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/rf-sdr/other-signals/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_selective_calling
  - https://www.itu.int/rec/R-REC-M.493
external:
  - { title: "GopherTrunk DSC decoder", url: /dsc.html }
---

**DSC** (**Digital Selective Calling**) is a maritime protocol for **calling specific
stations and broadcasting distress alerts**. Part of the Global Maritime Distress and
Safety System (GMDSS), it sends short [FSK](/reference/frequency-shift-keying/) data
bursts on **VHF channel 70** (156.525 MHz) and on MF/HF distress frequencies. Rather
than keep a crew listening on a voice channel, DSC lets a radio automatically send or
receive a digitally addressed call — including a one-button distress alert carrying the
vessel's identity and position — and only then draw attention to it.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A Digital Selective Calling burst laid out in fields: a dot pattern for synchronisation, a format specifier and sender MMSI, the call category or distress nature, and an error-check code." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1"><rect x="40" y="40" width="70" height="28" fill="currentColor" fill-opacity="0.12"/><rect x="110" y="40" width="120" height="28" fill="currentColor" fill-opacity="0.22"/><rect x="230" y="40" width="120" height="28" fill="none"/><rect x="350" y="40" width="70" height="28" fill="none"/></g>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="75" y="58">dot / phasing</text><text x="170" y="58">format / MMSI</text><text x="290" y="58">distress / call</text><text x="385" y="58">ECC</text></g>
  <text x="230" y="88" text-anchor="middle" font-size="8" fill="currentColor">FFSK 1200 baud · VHF Ch 70 (and MF/HF) · each symbol sent twice</text>
</svg>
<figcaption>DSC sends short FFSK bursts on VHF channel 70 for distress and routine calling, carrying the sender's MMSI; time-diversity repetition guards each symbol.</figcaption>
</figure>

## Overview

A DSC message carries the sender's and, for selective calls, the recipient's **MMSI**,
a **category** (routine, safety, urgency, distress), a message content or *nature of
distress* field, an optional **position and time**, and a means of continuing the
exchange (e.g. the working channel to switch to). A **distress alert** automatically
conveys the vessel identity and, if a GPS receiver is interfaced, its position and the
nature of the emergency, then repeats until acknowledged by a coast station. DSC
complements [AIS](/reference/ais/) on the safety side and is the alerting layer that
tells stations *when* to listen on the [marine VHF](/reference/marine-vhf/) voice
channels.

Physically, VHF DSC uses **fast FSK ([FFSK](/reference/ffsk/))** at 1200 baud with mark
and space tones of 1300 and 2100 Hz, sent as a phase-continuous audio-frequency shift.
MF/HF DSC uses narrow-shift FSK at 100 baud. The protocol has no cyclic redundancy
check; instead it relies on **time-diversity error control** — every character is
transmitted twice, once in a forward-position stream and again, delayed, in a
receive-position stream, and each 10-bit symbol carries a parity count of its four zero
bits. A decoder can therefore correct or flag most single-burst errors by comparing the
two copies.

## Technical characteristics

| Property | Value |
|----------|-------|
| Band | VHF Ch 70 (156.525 MHz) + MF/HF DSC frequencies |
| Modulation | FFSK (VHF, 1200 baud) / narrow FSK (MF/HF, 100 baud) |
| Symbol | 10 bits: 7 data + 3-bit zero-count parity |
| Error control | Symbol repetition (time diversity) + parity |
| Addressing | 9-digit MMSI, plus group and all-ships calls |
| Categories | Routine, safety, urgency, distress |

## History

DSC was standardised by the [ITU](/reference/itu/) in Recommendation ITU-R M.493 and
adopted as the calling and distress-alerting core of the GMDSS, which the IMO phased in
through the 1990s (fully effective from 1 February 1999). It replaced the older manual
watch on distress voice frequencies and, on MF, the human-monitored 2182 kHz and Morse
500 kHz watches, with automated digital alerting that guarantees a distress call reaches
a coast station and other nearby vessels.[^wiki][^itu]

## Deployment

DSC-capable radios are mandatory on SOLAS vessels and standard on modern recreational
VHF sets, where a dedicated red distress button triggers the alert. Coast stations,
rescue coordination centres, and other ships maintain an automatic watch on channel 70,
so the system works alongside [EPIRB 406](/reference/epirb-406/) beacons and AIS-SART to
form the GMDSS alerting chain. Because the bursts are short and unencrypted, DSC traffic
is readily received and logged by shore-based SDR listeners.

## Decoding it with GopherTrunk

GopherTrunk tunes channel 70 (or an MF/HF DSC frequency), demodulates the FFSK to a
symbol stream, aligns to the dot/phasing pattern, applies the symbol-repetition and
zero-count error control, and decodes the format, MMSI, category, and position fields
into a readable DSC message. Because DSC is unencrypted and the modulation is simple, it
is one of the maritime protocols GopherTrunk decodes end to end, alongside
[AIS](/reference/ais/). See the [DSC decoder](/dsc.html) page.

## Sources

[^wiki]: [Digital selective calling](https://en.wikipedia.org/wiki/Digital_selective_calling) — Wikipedia, for the maritime DSC protocol within GMDSS, VHF channel 70 signalling, MMSI addressing, symbol repetition, and distress alerting.
[^itu]: [Recommendation ITU-R M.493](https://www.itu.int/rec/R-REC-M.493) — International Telecommunication Union, the primary standard defining the DSC message format, symbol structure, error control, and call categories.
