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
  - { title: "The digital protocol landscape", url: /learn/protocol-landscape/ }
external:
  - { title: "Enhanced Digital Access Communications System (Wikipedia)", url: https://en.wikipedia.org/wiki/Enhanced_Digital_Access_Communications_System }
---

**EDACS** (**Enhanced Digital Access Communications System**) is a
[trunked-radio](/reference/trunked-radio/) system developed by GE/Ericsson (later
M-A-COM). It uses a **dedicated [control channel](/reference/control-channel/)** that
continuously coordinates the system, with analog FM voice or the digital **ProVoice**
([AMBE](/reference/ambe/)) option.

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
fleets; gradually displaced by [P25](/reference/project-25/).

## Deployment

Legacy public-safety, utility, and transportation systems, some still operating.

## Decoding it with GopherTrunk

See the [Status](/status.html) page for GopherTrunk's EDACS coverage (control-channel
following and supported voice modes).
