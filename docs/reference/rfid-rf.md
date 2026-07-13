---
slug: rfid-rf
title: RFID (RF layer)
entry_type: protocol
category: wireless-data-iot
description: "RFID is contactless identification where a reader powers a tag over LF, HF, or UHF radio; the tag replies by load modulation (near field) or backscatter (far field)."
keywords: RFID, radio-frequency identification, LF 125 kHz, HF 13.56 MHz, UHF 860-960 MHz, backscatter, load modulation, EPC Gen2, ISO 14443, ISO 15693, passive tag
aka: [RFID, "radio-frequency identification"]
autolink: true
infobox:
  - { label: Type, value: Contactless identification radio }
  - { label: Bands, value: "LF 125/134 kHz, HF 13.56 MHz, UHF 860–960 MHz" }
  - { label: Coupling, value: "Inductive (LF/HF) or backscatter (UHF)" }
  - { label: Tag power, value: Passive (reader-powered), semi-passive, or active }
  - { label: Standards, value: "ISO 14443/15693, ISO 18000, EPC Gen2" }
  - { label: GopherTrunk support, value: Not decoded (out of scope) }
see_also: [nfc-rf, on-off-keying, amplitude-shift-keying, frequency-bands, internet-of-things]
cite_urls:
  - https://en.wikipedia.org/wiki/Radio-frequency_identification
  - https://en.wikipedia.org/wiki/Backscatter#Radio_frequency
---

**RFID** (radio-frequency identification) is contactless identification in which a
**reader** energizes a nearby **tag** over radio and the tag replies with its stored ID.[^wiki]
The defining feature at the RF layer is that most tags are **passive**: they carry no
battery and instead harvest power from the reader's field, then answer either by *load
modulation* in the near field or by *backscatter* in the far field. RFID is the workhorse of
access badges, inventory, toll tags, and animal microchips, and it is the family
[NFC](/reference/nfc-rf/) grew out of.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A reader emits a continuous carrier that powers a passive tag; the tag replies by switching its antenna load, reflecting a modulated backscatter signal back to the reader." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="rf_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="30" y="55" width="70" height="40" fill="currentColor" fill-opacity="0.15" stroke="currentColor"/><text x="65" y="112">reader</text>
    <rect x="360" y="55" width="70" height="40" fill="none" stroke="currentColor"/><text x="395" y="112">tag (passive)</text>
  </g>
  <g stroke="currentColor" fill="none">
    <path d="M105 65 L355 65" marker-end="url(#rf_ar)"/>
    <path d="M355 88 L105 88" stroke-dasharray="4 3" marker-end="url(#rf_ar)"/>
  </g>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <text x="230" y="58">carrier powers the tag →</text>
    <text x="230" y="101">← backscatter / load-modulated reply (ID)</text>
  </g>
</svg>
<figcaption>The reader supplies a continuous carrier that both powers a passive tag and carries its reply back as a load-modulated or backscattered signal.</figcaption>
</figure>

## Overview

RFID spans three broad band regimes with genuinely different physics. **LF** and **HF**
tags sit in the reader's *near field* and couple inductively, like loosely linked
transformer windings; the tag replies by varying the load on its coil, which the reader
senses as a small change in its own field. **UHF** tags sit in the *far field* and reply by
**backscatter** — switching the impedance of their antenna so they reflect more or less of
the reader's incident wave, encoding data in that modulated reflection.

## Technical characteristics

| Band | Frequency | Coupling | Typical range | Examples |
|------|-----------|----------|---------------|----------|
| LF | 125 / 134 kHz | Inductive (near field) | ~cm | Animal ID, access fobs |
| HF | 13.56 MHz | Inductive (near field) | ~cm–1 m | Smart cards, NFC, ISO 14443/15693 |
| UHF | 860–960 MHz | Backscatter (far field) | ~1–10 m | EPC Gen2 inventory, toll tags |

Readers commonly encode commands with amplitude/[on-off keying](/reference/on-off-keying/)
so the tag keeps receiving power during the modulation, while tag replies use subcarrier
[ASK](/reference/amplitude-shift-keying/) or phase changes on the backscattered signal.

## History

Passive backscatter identification traces back to World War II IFF and to Harry Stockman's
1948 work on communication by reflected power. Commercial RFID grew through the late 20th
century, and the EPCglobal Gen2 UHF standard (later ISO 18000-63) standardized supply-chain
tagging in the 2000s.[^bs]

## Deployment

RFID is everywhere unglamorous: access control, library and retail inventory, passports and
transit cards (HF), electronic toll collection and warehouse tracking (UHF), and pet
microchips (LF). It underpins much of the physical [Internet of Things](/reference/internet-of-things/),
with [NFC](/reference/nfc-rf/) as its consumer-facing HF subset.

## Decoding it with GopherTrunk

RFID is out of scope for GopherTrunk, which decodes trunked land-mobile voice, not
short-range identification. LF/HF RFID lives near the reader's coil and is not a
free-space signal a scanner tunes across a band; UHF backscatter is dominated by the
reader's own carrier and needs a reader-style transceiver, not a passive receiver.
GopherTrunk implements none of the RFID air interfaces.

## Sources

[^wiki]: [Radio-frequency identification](https://en.wikipedia.org/wiki/Radio-frequency_identification) — Wikipedia, for the LF/HF/UHF band structure, passive tags, and coupling methods.
[^bs]: [Backscatter — radio frequency](https://en.wikipedia.org/wiki/Backscatter#Radio_frequency) — Wikipedia, for the reflected-power reply mechanism used by UHF tags.
