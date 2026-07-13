---
slug: nfc-rf
title: NFC (RF layer)
entry_type: protocol
category: wireless-data-iot
description: "NFC's RF layer is a 13.56 MHz inductively coupled near-field interface where a reader powers and talks to a passive tag through a shared magnetic field using load modulation."
keywords: NFC, near-field communication, 13.56 MHz, inductive coupling, load modulation, ASK, ISO 14443, ISO 15693, ISO 18092, RFID, contactless, magnetic coupling, HF RFID
aka: [NFC RF, NFC air interface, NFC physical layer]
autolink: true
infobox:
  - { label: Type, value: Inductive near-field interface }
  - { label: Standards body, value: "ISO/IEC, NFC Forum" }
  - { label: Introduced, value: "2003–2004" }
  - { label: Access, value: Reader-driven, half-duplex }
  - { label: Frequency, value: 13.56 MHz HF }
  - { label: Modulation, value: "ASK / load modulation, subcarrier" }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [rfid-rf, near-field-communication, amplitude-shift-keying, near-field-far-field, frequency-bands, subcarrier]
cite_urls:
  - https://en.wikipedia.org/wiki/Near-field_communication
  - https://en.wikipedia.org/wiki/ISO/IEC_14443
---

**NFC (RF layer)** is the short-range physical interface behind contactless cards and
tap-to-pay, operating at 13.56 MHz through *inductive coupling* rather than a radiating
wave.[^wiki] A reader's coil and a tag's coil act like a loosely coupled transformer in
the antenna's [near field](/reference/near-field-far-field/): the reader supplies energy
and a clock, and the tag replies by modulating the load it presents. This page covers the
radio interface; for the everyday concept see
[near-field communication](/reference/near-field-communication/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="An NFC reader coil couples magnetically to a passive tag coil at close range; the reader's 13.56 MHz field powers the tag, and the tag answers by load modulation that the reader senses back through the shared field." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none">
    <ellipse cx="110" cy="75" rx="30" ry="45" stroke-width="1.5"/>
    <ellipse cx="118" cy="75" rx="22" ry="34" stroke-width="1"/>
    <ellipse cx="330" cy="75" rx="26" ry="40" stroke-width="1.5"/>
    <ellipse cx="322" cy="75" rx="18" ry="30" stroke-width="1"/>
  </g>
  <g stroke="currentColor" stroke-width="1" stroke-opacity="0.55" fill="none">
    <path d="M150 55 Q225 40 290 55"/><path d="M150 75 Q225 62 300 75"/><path d="M150 95 Q225 110 290 95"/>
  </g>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="110" y="135">reader coil</text>
    <text x="330" y="135">tag coil</text>
    <text x="225" y="30">13.56 MHz magnetic field</text>
    <text x="225" y="128">power + clock →   ← load modulation</text>
  </g>
</svg>
<figcaption>An NFC reader powers a passive tag through a shared 13.56 MHz magnetic field; the tag answers by varying the load it presents on that field.</figcaption>
</figure>

## Overview

NFC is high-frequency (HF) [RFID](/reference/rfid-rf/) with an added peer-to-peer mode.
The reader drives an unmodulated (or ASK-modulated) 13.56 MHz carrier through its coil.
A nearby passive tag rectifies that field to power its chip, and to answer it switches an
extra load on its own coil on and off. That switching reflects back as tiny amplitude
changes on the reader's carrier — often carried on a subcarrier offset (typically
847 kHz) — which the reader detects. Because coupling falls off very steeply with
distance, range is only a few centimetres; short range is a security feature, making a tap
deliberate and hard to intercept from afar. The reader sends data by
[amplitude-shift keying](/reference/amplitude-shift-keying/) the field.

## Technical characteristics

| Property | Value |
|----------|-------|
| Frequency | 13.56 MHz (HF ISM) |
| Coupling | Magnetic (inductive) near field |
| Range | ~1–4 cm |
| Reader → tag | ASK (100% or 10%) |
| Tag → reader | Load modulation on ~847 kHz subcarrier |
| Data rate | 106 / 212 / 424 kbit/s (typical) |
| Base standards | ISO/IEC 14443, 15693, 18092 |

Modes include card emulation (a phone acting as a contactless card), reader/writer (a
phone scanning a tag), and peer-to-peer. The relevant standards — ISO/IEC 14443 for
proximity cards and 18092 for NFCIP — define framing and bit rates on top of this radio.

## History

NFC grew out of HF RFID work by Sony (FeliCa) and NXP (Mifare), formalised around
2003–2004 and promoted by the NFC Forum. Its breakout came with smartphone wallets and
transit systems in the 2010s.

## Deployment

NFC is everywhere contactless: payment cards and phone wallets, transit fare cards, access
badges, and product tags. It complements longer-range radios in a phone, handling the
intentional "touch here" interactions rather than networked data.

## Decoding it with GopherTrunk

GopherTrunk does **not** decode NFC. Its 13.56 MHz inductive, near-field interface is a
fundamentally different regime from the far-field VHF/UHF signals GopherTrunk targets, and
it is not radiated in the usual sense — you cannot receive it at range with an ordinary
antenna. NFC analysis uses dedicated readers or purpose-built HF probes, well outside
GopherTrunk's land-mobile and aeronautical scope.

## Sources

[^wiki]: [Near-field communication](https://en.wikipedia.org/wiki/Near-field_communication) — Wikipedia, on the NFC air interface, 13.56 MHz inductive coupling, load modulation, and the ISO/IEC standards; see also [ISO/IEC 14443](https://en.wikipedia.org/wiki/ISO/IEC_14443) for the proximity-card layer.
