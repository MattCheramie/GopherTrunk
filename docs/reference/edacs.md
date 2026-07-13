---
slug: edacs
title: EDACS
entry_type: protocol
category: land-mobile-trunking
description: EDACS (Enhanced Digital Access Communications System) is a trunked-radio system from GE/Ericsson/M-A-COM using a dedicated control channel, with analog and ProVoice digital variants.
keywords: EDACS, Ericsson, GE, M/A-COM, trunked radio, ProVoice, control channel, legacy, 9600 baud, GE-Star, Aegis, wide-area
aka: [EDACS]
autolink: true
infobox:
  - { label: Type, value: Trunked radio (analog & digital) }
  - { label: Developer, value: GE / Ericsson / M-A-COM }
  - { label: Era, value: 1980s–2000s }
  - { label: Access, value: FDMA with dedicated control channel }
  - { label: Control channel, value: 9600 bps digital }
  - { label: Voice, value: Analog FM or ProVoice (AMBE) }
  - { label: GopherTrunk support, value: See Status }
see_also: [provoice, ambe, fdma, trunked-radio, control-channel, motorola-type-ii, ltr, mpt-1327]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Enhanced_Digital_Access_Communications_System
  - https://wiki.radioreference.com/index.php/EDACS
---

**EDACS** (**Enhanced Digital Access Communications System**) is a
[trunked-radio](/reference/trunked-radio/) system developed by **GE**, later **Ericsson**,
and finally **M-A-COM**. It uses a **dedicated [control channel](/reference/control-channel/)**
that continuously coordinates the system, carrying analog FM voice or the digital
**[ProVoice](/reference/provoice/)** option, which encodes speech with an
[AMBE](/reference/ambe/)-family [vocoder](/reference/vocoder/).[^wiki] EDACS was the main
competitor to [Motorola Type II](/reference/motorola-type-ii/) in the North American
public-safety and utility market.

<figure class="figure" markdown="0">
<svg viewBox="0 0 380 140" role="img" aria-label="An EDACS dedicated control channel assigning analog FM or digital ProVoice voice channels." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="20" width="300" height="26" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="190" y="37" text-anchor="middle" font-size="9" fill="currentColor">dedicated control channel (9600 bps)</text>
  <g stroke="currentColor" stroke-width="1.1" fill="none"><rect x="40" y="80" width="90" height="34"/><rect x="150" y="80" width="90" height="34"/><rect x="260" y="80" width="80" height="34" fill="currentColor" fill-opacity="0.18"/></g>
  <text x="190" y="130" text-anchor="middle" font-size="8.5" fill="currentColor">analog FM or ProVoice channels (assigned on demand)</text>
  <line x1="190" y1="46" x2="300" y2="78" stroke="currentColor" stroke-dasharray="3 3" marker-end="url(#lg_edacs)"/>
  <defs><marker id="lg_edacs" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>EDACS uses a dedicated, always-on control channel to assign analog FM or digital ProVoice voice channels.</figcaption>
</figure>

## Overview

EDACS is known for **fast call setup** delivered by its always-on control channel and its
tightly specified **logical channel numbering (LCN)**, which maps physical frequencies to
fixed channel numbers so radios can retune quickly. When a user keys up, the control
channel broadcasts a [channel grant](/reference/channel-grant/) naming the LCN for that
[talkgroup](/reference/talkgroup/), and radios move to the corresponding frequency. Systems
can be single-site or **wide-area** (multi-site, networked), and voice may be analog FM or,
on ProVoice-equipped systems, digital AMBE. Because the LCN-to-frequency mapping is
system-specific and not broadcast in a self-describing way, correctly monitoring an EDACS
system historically required knowing its channel plan — a notable contrast with
self-describing control channels. Variants and marketing names in this family include GE-Star,
Aegis (an early digital-voice mode), and the later ProVoice.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | [FDMA](/reference/fdma/) with a dedicated control channel |
| Control channel | Continuous 9600 bit/s digital signalling |
| Channel numbering | Logical Channel Numbers (LCN) mapped to frequencies |
| Voice | Analog FM or [ProVoice](/reference/provoice/) ([AMBE](/reference/ambe/) digital) |
| Topology | Single-site or wide-area (networked multi-site) |
| Bands | VHF, UHF, 800/900 MHz |

## History

EDACS was **deployed from the 1980s** by GE, whose two-way radio business passed to
Ericsson and then to M-A-COM (later part of Harris/L3Harris).[^wiki][^rr] It was engineered
as a high-reliability trunking system for public safety and utilities, emphasising fast
access and robust wide-area operation. Digital voice arrived first as **Aegis** and later
as **ProVoice**, layering AMBE-coded speech onto the existing control architecture. As the
industry standardised on [P25](/reference/project-25/), EDACS was gradually displaced,
though its lineage continued in the vendor's later product families.

## Deployment

EDACS ran many **legacy public-safety, utility, and transportation** systems, especially in
North America, and some remain in service today. Utility companies in particular valued its
wide-area reliability. Most large public-safety users have since migrated to P25, but the
installed base and its distinctive LCN scheme keep EDACS relevant to scanner and SDR users
monitoring older infrastructure.

## Decoding it with GopherTrunk

Following an EDACS system means locking to the **9600 bit/s control channel**, decoding its
signalling, and mapping grants through the LCN plan to the correct voice frequency — a
process that benefits from a known channel list. Analog FM voice is directly playable; the
[ProVoice](/reference/provoice/) digital option needs an AMBE-family vocoder and has
historically been gated by proprietary licensing. Keyed encryption is out of scope for
GopherTrunk's receiver-only design. See the [Status](/status.html) page for GopherTrunk's
current EDACS control-channel following and supported voice modes.

## Sources

[^wiki]: [Enhanced Digital Access Communications System](https://en.wikipedia.org/wiki/Enhanced_Digital_Access_Communications_System) — Wikipedia, for the GE/Ericsson/M-A-COM EDACS trunking system, its dedicated control channel, and the ProVoice digital option.
[^rr]: [EDACS](https://wiki.radioreference.com/index.php/EDACS) — RadioReference Wiki, for logical channel numbering, wide-area operation, and the Aegis/ProVoice digital-voice history.
