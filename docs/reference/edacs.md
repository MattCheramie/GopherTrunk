---
slug: edacs
title: EDACS
entry_type: protocol
category: protocols
description: EDACS (Enhanced Digital Access Communications System) is a trunked-radio system from GE/Ericsson/M-A-COM using a dedicated control channel, with analog and ProVoice digital variants.
keywords: EDACS, Ericsson, GE, M/A-COM, trunked radio, ProVoice, control channel, legacy
aka: [EDACS]
autolink: true
infobox:
  - { label: Type, value: Trunked radio (analog & digital) }
  - { label: Developer, value: GE / Ericsson / M-A-COM }
  - { label: Era, value: 1980s–2000s }
  - { label: Access, value: FDMA with dedicated control channel }
  - { label: Voice, value: Analog FM or ProVoice (AMBE) }
  - { label: GopherTrunk support, value: See Status }
see_also: [trunked-radio, control-channel, motorola-type-ii, ltr, mpt-1327]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Enhanced_Digital_Access_Communications_System
---

**EDACS** (**Enhanced Digital Access Communications System**) is a
[trunked-radio](/reference/trunked-radio/) system developed by GE/Ericsson (later
M-A-COM). It uses a **dedicated [control channel](/reference/control-channel/)** that
continuously coordinates the system, with analog FM voice or the digital **ProVoice**
([AMBE](/reference/ambe/)) option.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 380 140" role="img" aria-label="EDACS dedicated control channel assigning analog or ProVoice channels." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="20" width="300" height="26" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="190" y="37" text-anchor="middle" font-size="9" fill="currentColor">dedicated control channel</text>
  <g stroke="currentColor" stroke-width="1.1" fill="none"><rect x="40" y="80" width="90" height="34"/><rect x="150" y="80" width="90" height="34"/><rect x="260" y="80" width="80" height="34" fill="currentColor" fill-opacity="0.18"/></g>
  <text x="190" y="130" text-anchor="middle" font-size="8.5" fill="currentColor">analog FM voice channels (assigned on demand)</text>
  <line x1="190" y1="46" x2="300" y2="78" stroke="currentColor" stroke-dasharray="3 3" marker-end="url(#lg_edacs)"/>
  <defs><marker id="lg_edacs" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>EDACS uses a dedicated control channel to assign analog FM (or digital ProVoice) channels.</figcaption>
</figure>

## Overview

EDACS is known for fast call setup via its always-on control channel and tightly
specified channel numbering. Systems can be wide-area (multi-site) and were a primary
competitor to [Motorola Type II](/reference/motorola-type-ii/).

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | [FDMA](/reference/fdma/) |
| Control channel | Dedicated, continuous |
| Voice | Analog FM or ProVoice (digital) |

## History

Deployed from the 1980s by GE/Ericsson and M-A-COM for public-safety and utility
fleets; gradually displaced by [P25](/reference/project-25/).[^wiki]

## Deployment

Legacy public-safety, utility, and transportation systems, some still operating.

## Decoding it with GopherTrunk

See the [Status](/status.html) page for GopherTrunk's EDACS coverage (control-channel
following and supported voice modes).

## Sources

[^wiki]: [Enhanced Digital Access Communications System](https://en.wikipedia.org/wiki/Enhanced_Digital_Access_Communications_System) — Wikipedia, for the GE/Ericsson/M-A-COM EDACS trunking system, its dedicated control channel, and ProVoice digital option.
