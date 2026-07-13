---
slug: provoice
title: ProVoice
entry_type: protocol
category: land-mobile-trunking
description: ProVoice is the digital-voice option for EDACS trunked systems, using an AMBE-family vocoder over the otherwise analog-trunking EDACS air interface.
keywords: ProVoice, EDACS digital voice, M/A-COM, Ericsson, AMBE, digital trunking, Aegis, proprietary vocoder
aka: [ProVoice]
autolink: true
infobox:
  - { label: Type, value: Digital voice mode for EDACS trunking }
  - { label: Developer, value: Ericsson / M-A-COM }
  - { label: Runs on, value: EDACS control-channel trunking }
  - { label: Voice, value: AMBE-family vocoder }
  - { label: Access, value: FDMA (EDACS control channel) }
  - { label: GopherTrunk support, value: See Status }
see_also: [edacs, ambe, fdma, vocoder, trunked-radio, control-channel]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Enhanced_Digital_Access_Communications_System
  - https://wiki.radioreference.com/index.php/EDACS
---

**ProVoice** is the **digital-voice option for [EDACS](/reference/edacs/)** trunked systems
(GE/Ericsson, later M-A-COM). It replaces EDACS's analog FM voice with a digital
[AMBE](/reference/ambe/)-family [vocoder](/reference/vocoder/) while keeping the same EDACS
control-channel signalling and trunking architecture underneath.[^wiki] In other words,
ProVoice is not a separate trunking system — it is the payload on the voice channel that an
EDACS control channel grants, swapping analog audio for coded digital speech.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="An EDACS control channel assigning a voice channel that carries digital ProVoice AMBE frames instead of analog FM." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="24" width="400" height="22" rx="4" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.1"/><text x="230" y="39" text-anchor="middle" font-size="8.5" fill="currentColor">EDACS control channel</text>
  <g stroke="currentColor" stroke-width="1.1" fill="none"><rect x="120" y="64" width="60" height="26"/><rect x="180" y="64" width="60" height="26"/><rect x="240" y="64" width="60" height="26"/><rect x="300" y="64" width="60" height="26"/></g>
  <text x="240" y="105" text-anchor="middle" font-size="8" fill="currentColor">ProVoice = digital AMBE frames (vs analog FM)</text>
  <line x1="230" y1="46" x2="240" y2="62" stroke="currentColor" stroke-dasharray="3 3"/>
</svg>
<figcaption>ProVoice carries digital AMBE voice over EDACS, in place of the system's analog FM voice channels.</figcaption>
</figure>

## Overview

On a ProVoice system, the [EDACS](/reference/edacs/) control channel behaves exactly as it
does for analog voice: it broadcasts [channel grants](/reference/channel-grant/) that steer
radios in a [talkgroup](/reference/talkgroup/) to a logical channel number. What changes is
the modulated voice channel itself — instead of analog FM audio, it carries digital speech
frames produced by an **AMBE-family vocoder** (the same lineage of Multi-Band Excitation
coders used by P25 and DMR, though ProVoice's framing is its own). This gave EDACS operators
the benefits of digital voice — cleaner audio at the noise floor and optional encryption —
without abandoning their existing trunking infrastructure. ProVoice was the successor to
EDACS's earlier **Aegis** digital-voice mode.

## Technical characteristics

| Property | Value |
|----------|-------|
| Trunking layer | [EDACS](/reference/edacs/) control channel ([FDMA](/reference/fdma/)) |
| Voice | Digital, [AMBE](/reference/ambe/)-family [vocoder](/reference/vocoder/) |
| Predecessor mode | Aegis (earlier EDACS digital voice) |
| Signalling | Unchanged from analog EDACS (talkgroups, radio IDs, LCN) |
| Encryption | Optional (system-dependent) |

## History

ProVoice was introduced by **Ericsson/M-A-COM** as the digital-voice evolution of EDACS,
following the earlier Aegis mode, to keep the platform competitive with digital
[P25](/reference/project-25/) while protecting operators' investment in EDACS trunking.[^wiki][^rr]
Its vocoder and framing are proprietary, which — combined with EDACS's own decline — kept
ProVoice a comparatively niche digital-voice format, largely confined to the installed EDACS
base rather than winning new deployments.

## Deployment

ProVoice appears on **legacy EDACS public-safety and utility systems** that upgraded to
digital voice, primarily in North America. As those systems migrate to P25, ProVoice usage
continues to shrink, but some networks remain on the air and are of interest to scanner and
SDR users monitoring older infrastructure.

## Decoding it with GopherTrunk

The **EDACS trunking layer is followed the same way regardless of whether the voice is analog
or ProVoice** — the decoder tracks the control channel and channel grants identically. The
difference is only in the voice payload: ProVoice needs an AMBE-family vocoder, and its
proprietary format has historically required licensed components, so digital-voice recovery is
gated accordingly. Keyed encryption is out of scope for GopherTrunk's receiver-only design.
See the [Status](/status.html) page for the current state of EDACS control-channel following
and ProVoice voice support.

## Sources

[^wiki]: [Enhanced Digital Access Communications System](https://en.wikipedia.org/wiki/Enhanced_Digital_Access_Communications_System) — Wikipedia, for EDACS and its ProVoice digital-voice option using an AMBE-family vocoder.
[^rr]: [EDACS](https://wiki.radioreference.com/index.php/EDACS) — RadioReference Wiki, for the Aegis and ProVoice digital-voice modes and their relationship to the EDACS control channel.
