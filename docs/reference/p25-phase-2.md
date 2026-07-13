---
slug: p25-phase-2
title: P25 Phase 2
entry_type: protocol
category: land-mobile-trunking
description: P25 Phase 2 is the TDMA air interface of Project 25, placing two voice timeslots in a 12.5 kHz channel using H-DQPSK/H-CPM and the AMBE+2 vocoder for doubled spectrum efficiency.
keywords: P25 Phase 2, TDMA, AMBE+2, H-DQPSK, H-CPM, two-slot, spectrum efficiency, public safety, 6.25 kHz, TIA-102, control channel
aka: [P25 Phase 2, P25 Phase II, Phase 2 P25]
autolink: true
infobox:
  - { label: Type, value: Digital land-mobile radio }
  - { label: Part of, value: Project 25 }
  - { label: Access, value: TDMA (2 slots) }
  - { label: Channel spacing, value: 12.5 kHz (6.25 kHz equivalent) }
  - { label: Modulation, value: H-DQPSK / H-CPM }
  - { label: Vocoder, value: AMBE+2 }
  - { label: GopherTrunk support, value: Decoded }
see_also: [project-25, p25-phase-1, p25-cai, ambe-plus-2, c4fm, cqpsk, network-access-code, tsbk, tdma, control-channel]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
  - { title: "Analog vs. digital voice", url: /learn/rf-sdr/digital-voice/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
  - https://www.cisa.gov/safecom/p25
---

**P25 Phase 2** is the second-generation air interface of [Project 25](/reference/project-25/),
using **two-slot [TDMA](/reference/tdma/)** to carry two simultaneous voice
conversations in a single 12.5 kHz channel — effectively 6.25 kHz per call.[^wiki]
It reuses Phase 1's [Common Air Interface](/reference/p25-cai/) addressing and
signalling but replaces the FDMA voice channel with a time-shared one.

<figure class="figure" markdown="0">
<svg viewBox="0 0 380 140" role="img" aria-label="Two TDMA slots in a 12.5 kHz channel for P25 Phase 2." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="105" x2="360" y2="105" stroke="currentColor" stroke-opacity="0.4" marker-end="url(#ta_p2)"/>
  <text x="195" y="128" text-anchor="middle" font-size="9" fill="currentColor">time → · one 12.5 kHz channel, 2 slots</text>
  <g stroke="currentColor" stroke-width="1.1"><rect x="40" y="40" width="52" height="50" fill="currentColor" fill-opacity="0.22"/><rect x="92" y="40" width="52" height="50" fill="none"/><rect x="144" y="40" width="52" height="50" fill="currentColor" fill-opacity="0.22"/><rect x="196" y="40" width="52" height="50" fill="none"/><rect x="248" y="40" width="52" height="50" fill="currentColor" fill-opacity="0.22"/><rect x="300" y="40" width="52" height="50" fill="none"/></g><g font-size="9" fill="currentColor" text-anchor="middle"><text x="66" y="69">1</text><text x="118" y="69">2</text><text x="170" y="69">1</text><text x="222" y="69">2</text><text x="274" y="69">1</text><text x="326" y="69">2</text></g>
  <defs><marker id="ta_p2" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>P25 Phase 2 uses two-slot TDMA, fitting two calls in one 12.5 kHz channel.</figcaption>
</figure>

## Overview

Phase 2 was introduced to meet FCC spectrum-efficiency goals. Where
[Phase 1](/reference/p25-phase-1/) gives each call its own frequency, Phase 2 divides
a traffic channel into two repeating timeslots, doubling voice capacity in the same
bandwidth. It uses the more efficient half-rate [AMBE+2](/reference/ambe-plus-2/)
vocoder and a phase-shift modulation family: H-DQPSK on the outbound (base-to-mobile)
link and H-CPM on the inbound (mobile-to-base) link. The two are chosen so that a
mobile's transmitter can stay simple and constant-envelope while the base station
uses the more linear form. Crucially, Phase 2 is a *traffic-channel* technology: a
system almost always keeps a [Phase 1](/reference/p25-phase-1/) control channel to
run the trunking.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | TDMA, 2 slots |
| Channel | 12.5 kHz (6.25 kHz equivalent capacity) |
| Modulation | H-DQPSK (outbound) / H-CPM (inbound) |
| Symbol rate | 6000 symbols/s per channel |
| Vocoder | [AMBE+2](/reference/ambe-plus-2/) (half-rate, ~3.6 kbps incl. FEC) |
| Control channel | Usually a [C4FM](/reference/c4fm/) Phase 1 [TSBK](/reference/tsbk/) control channel |
| Addressing | shared with Phase 1 ([NAC](/reference/network-access-code/), talkgroup, radio ID) |

A common deployment keeps a [C4FM](/reference/c4fm/) Phase 1
[control channel](/reference/control-channel/) — carrying [TSBK](/reference/tsbk/)
grants and the [WACN](/reference/wacn/) / [System ID](/reference/system-id/) /
[RFSS](/reference/rfss/) identity — while voice traffic rides Phase 2 TDMA slots. The
grant tells a listening radio (or scanner) not just which frequency but which of the
two timeslots to decode.

## History

Phase 2 was standardised by the [TIA](/reference/tia/) within the TIA-102 suite to
follow narrowbanding and 6.25 kHz-equivalent spectrum-efficiency mandates, and has
been deployed on large metropolitan and statewide systems since the early 2010s.[^wiki][^cisa]
The switch to AMBE+2 (from Phase 1's IMBE) was necessary because two calls now share
the bit budget of one, so each needs a lower-rate vocoder.

## Deployment

Phase 2 is widely used by busy urban and countywide public-safety systems where
channel capacity is at a premium, frequently mixed with Phase 1 on the same network:
Phase 1 control channel and mutual-aid channels, Phase 2 for the bulk of routine
voice traffic. As with Phase 1, much of this traffic is AES-encrypted on many modern
systems.

## Decoding it with GopherTrunk

GopherTrunk follows the (Phase 1) control channel, reads the
[TSBK](/reference/tsbk/) [channel grant](/reference/channel-grant/), tunes to the
assigned Phase 2 traffic channel *and slot*, and decodes the
[AMBE+2](/reference/ambe-plus-2/) voice. Because two calls interleave in time on one
frequency, the decoder demultiplexes the bursts by slot before running the vocoder.
Clear and known-key calls produce audio; keyed-encrypted calls are logged with their
metadata only. See [Status](/status.html).

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, for the P25 Phase 2 two-slot TDMA air interface, H-DQPSK/H-CPM modulation, the AMBE+2 vocoder, and TIA standardisation.
[^cisa]: [P25 (Project 25)](https://www.cisa.gov/safecom/p25) — CISA SAFECOM, for the TIA-102 suite and the 6.25 kHz-equivalent spectrum-efficiency mandate that motivated Phase 2.
