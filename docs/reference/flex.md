---
slug: flex
title: FLEX
entry_type: protocol
category: paging-data
description: "FLEX is a high-speed one-way paging protocol developed by Motorola, using 4-level FSK at up to 6400 bps with strong synchronisation and error correction for reliable wide-area paging."
keywords: FLEX paging, Motorola FLEX, 4-FSK, high-speed pager, 1600 3200 6400 bps, simulcast paging, ReFLEX, frame phase, time synchronised
aka: [FLEX, ReFLEX]
autolink: true
infobox:
  - { label: Type, value: One-way paging protocol }
  - { label: Developer, value: Motorola }
  - { label: Modulation, value: 2- or 4-level FSK }
  - { label: Bit rates, value: 1600 / 3200 / 6400 bps }
  - { label: Error correction, value: BCH + interleaving }
  - { label: GopherTrunk support, value: See Status }
see_also: [pocsag, ermes, four-fsk, frequency-shift-keying, bch-code, interleaving]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/rf-sdr/other-signals/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/FLEX_(protocol)
  - https://www.sigidwiki.com/wiki/FLEX
---

**FLEX** is a high-speed one-way **paging** protocol developed by **Motorola** to
succeed [POCSAG](/reference/pocsag/). It uses 2- or 4-level
[FSK](/reference/frequency-shift-keying/) at up to 6400 bps, with a rigid
time-synchronised frame structure and heavy [interleaving](/reference/interleaving/)
that make it resilient on wide-area **simulcast** networks where many transmitters key
up on the same frequency at once.[^wiki][^sigid] Where POCSAG is asynchronous — a page
can start at any time — FLEX is a scheduled system with a global time base, which is
what lets it pack far more capacity onto a channel.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="A FLEX frame with a sync header and time-multiplexed data blocks, running at higher rates than POCSAG." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1">
    <rect x="30" y="40" width="60" height="28" fill="currentColor" fill-opacity="0.12"/><rect x="90" y="40" width="85" height="28" fill="none"/><rect x="175" y="40" width="85" height="28" fill="none"/><rect x="260" y="40" width="85" height="28" fill="none"/><rect x="345" y="40" width="85" height="28" fill="none"/>
  </g>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="60" y="58">sync</text><text x="132" y="58">block</text><text x="217" y="58">block</text><text x="302" y="58">block</text><text x="387" y="58">block</text></g>
  <text x="230" y="88" text-anchor="middle" font-size="8" fill="currentColor">multi-level FSK · up to 6400 bps</text>
</svg>
<figcaption>FLEX is a higher-rate paging protocol using time-synchronised frames and multi-level FSK, one of 128 frames per 4-minute cycle.</figcaption>
</figure>

## Overview

FLEX divides time into a strict hierarchy. The largest unit is a **cycle** of
1.875 seconds… assembled into a **4-minute** super-structure of **128 frames**
(numbered 0–127), each frame carrying a **sync** section followed by 11 data
**blocks**. Every frame's sync also announces the **speed** (1600, 3200, or 6400 bps)
and **level** (2-FSK or 4-FSK) the data blocks that follow will use, so the same
channel can shift rate frame by frame. A pager is assigned to particular frames and
only has to wake for those — the same battery-saving idea as POCSAG's frame slots, but
scaled up and locked to a network-wide clock. Because the schedule is deterministic,
many simulcast transmitters can radiate the identical waveform in the identical time
slot, extending coverage without collisions.

## Technical characteristics

| Property | Value |
|----------|-------|
| Modulation | 2-FSK or 4-FSK (level signalled per frame) |
| Bit rates | 1600, 3200, 6400 bps |
| Frame structure | 128 frames per 4-minute cycle; sync + 11 blocks |
| Coding | [BCH(31,21)](/reference/bch-code/) + bit interleaving |
| Timing | Network-wide time base for simulcast |
| Variants | FLEX (one-way), ReFLEX (two-way), InFLEXion (voice) |

## How it works

The 4-FSK option is what gives FLEX its top speed: four frequency deviations encode
two bits per symbol, doubling throughput over 2-FSK at the same symbol rate — the same
[4-level FSK](/reference/four-fsk/) idea used by land-mobile C4FM systems, applied to
paging. Robustness comes from two layers: [BCH](/reference/bch-code/) coding corrects
bit errors inside each codeword exactly as in POCSAG, and **block interleaving** then
spreads the bits of each codeword across time so a fade or a simulcast-overlap glitch
damages many codewords slightly (correctable) rather than one codeword catastrophically
(uncorrectable). The precise frame timing means a receiver, once locked to the sync,
knows exactly where every field sits and can predict the next frame's rate before it
arrives.

## History

Motorola introduced FLEX in the early-to-mid 1990s as paging traffic outgrew POCSAG's
capacity, and it was widely licensed to paging carriers worldwide.[^wiki] Two notable
relatives extended the family: **ReFLEX** added a return channel for two-way paging
and confirmation (used by interactive pagers and some telemetry), and **InFLEXion**
attempted voice paging. As cellular text messaging displaced consumer paging, FLEX
persisted mainly in high-reliability wide-area systems.

## Deployment

FLEX is used by commercial wide-area paging carriers, hospital and healthcare
networks, and emergency-notification and utility systems where a fast, simulcast,
one-way channel is valued. Its scheduled structure makes it well suited to the
regional simulcast systems that still carry critical alerting today, and it coexists
with legacy POCSAG and, in Europe, [ERMES](/reference/ermes/) infrastructure.

## Decoding it with GopherTrunk

FLEX belongs to the same FSK paging family GopherTrunk targets with POCSAG: the front
end FM-demodulates the channel and slices the 2-/4-FSK symbols, and the decoder then
has to acquire FLEX's frame sync, follow the per-frame speed/level signalling, and
de-interleave and BCH-correct the blocks. The added work over POCSAG is the rigid
frame/cycle timing and the multi-rate handling rather than any proprietary barrier;
FLEX content is not encrypted at the protocol level. See [Status](/status.html) for
current coverage and the [POCSAG decoder](/pocsag.html) page for the shared paging
pipeline.

## Sources

[^wiki]: [FLEX](https://en.wikipedia.org/wiki/FLEX_(protocol)) — Wikipedia, for Motorola's high-speed paging protocol, its multi-level FSK rates, frame/cycle timing, and the ReFLEX/InFLEXion variants.
[^sigid]: [FLEX](https://www.sigidwiki.com/wiki/FLEX) — Signal Identification Guide, for the on-air characteristics, bit rates, and framing of the FLEX paging signal.
