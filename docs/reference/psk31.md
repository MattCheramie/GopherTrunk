---
slug: psk31
title: PSK31
entry_type: protocol
category: amateur-digital
description: "PSK31 is a narrowband amateur-radio keyboard chat mode using differential BPSK at 31.25 baud with a variable-length Varicode alphabet for efficient live text."
keywords: PSK31, phase shift keying, BPSK, differential BPSK, Varicode, 31.25 baud, narrowband, keyboard chat, Peter Martinez, G3PLX, amateur radio
aka: [PSK31]
autolink: true
infobox:
  - { label: Type, value: Narrowband keyboard chat mode }
  - { label: Created by, value: Peter Martinez (G3PLX) }
  - { label: Introduced, value: 1998 }
  - { label: Modulation, value: Differential BPSK, 31.25 baud }
  - { label: Alphabet, value: Varicode (variable-length) }
  - { label: Bandwidth, value: ~31 Hz per signal }
  - { label: GopherTrunk support, value: Not decoded (use fldigi) }
see_also: [bpsk, differential-decoding, phase-shift-keying, rtty]
cite_urls:
  - https://en.wikipedia.org/wiki/PSK31
  - http://www.arrl.org/psk31-spec
---

**PSK31** is a **narrowband amateur-radio keyboard chat mode** that carries live,
conversational text in a signal only about 31 Hz wide. It sends data as
[differential BPSK](/reference/differential-decoding/) at 31.25 baud and encodes text with
a variable-length **Varicode** alphabet — short bit patterns for common letters, longer
ones for rare characters — so ordinary English flows at a comfortable typing speed.[^wiki]
Introduced in 1998, PSK31 revitalized HF digital operation by pairing sound-card decoding
with a waveform far narrower and more sensitive than [RTTY](/reference/rtty/), and it
remains a staple of the HF digital sub-bands.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="PSK31 conveys bits as the presence or absence of a 180-degree carrier phase reversal, shown as a BPSK constellation with two points and the reversal between them." xmlns="http://www.w3.org/2000/svg">
  <line x1="150" y1="20" x2="150" y2="120" stroke="currentColor" stroke-width="0.6"/>
  <line x1="80" y1="70" x2="220" y2="70" stroke="currentColor" stroke-width="0.6"/>
  <circle cx="95" cy="70" r="5" fill="currentColor"/><circle cx="205" cy="70" r="5" fill="currentColor"/>
  <path d="M100 70 A55 55 0 0 1 200 70" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="3 3"/>
  <text x="95" y="92" font-size="8" fill="currentColor" text-anchor="middle">phase 180°</text>
  <text x="205" y="92" font-size="8" fill="currentColor" text-anchor="middle">phase 0°</text>
  <text x="150" y="35" font-size="8" fill="currentColor" text-anchor="middle">180° reversal = bit</text>
  <g font-size="8.5" fill="currentColor"><text x="280" y="55">"0" → phase reversal</text><text x="280" y="75">"1" → no reversal</text><text x="280" y="98">31.25 baud · Varicode text</text></g>
</svg>
<figcaption>PSK31 differentially encodes each bit as the presence (0) or absence (1) of a 180° carrier phase reversal, spelling out Varicode-encoded text at 31.25 baud.</figcaption>
</figure>

## Overview

PSK31 is built for casual real-time chat between two or more stations. Because it is
differential — each bit is decided by comparing one symbol's phase to the previous one —
the receiver does not need an absolute phase reference, only to detect whether the carrier
flipped 180°. Idle time is filled with a steady stream of phase reversals, which also gives
the decoder a clean timing signal to lock onto. Raised-cosine pulse shaping keeps the
occupied bandwidth tight and the spectrum clean.

## Variants

A phase-only signal is vulnerable at very low SNR, so **QPSK31** adds a convolutional code
across a four-phase constellation for error correction at the cost of a more demanding
receiver. Faster relatives (PSK63, PSK125) simply scale the symbol rate up for better
throughput on stronger paths, and multi-carrier variants stack several PSK streams side by
side.

## Technical characteristics

| Property | Value |
|----------|-------|
| Modulation | Differential [BPSK](/reference/bpsk/) (QPSK variant optional) |
| Symbol rate | 31.25 baud |
| Occupied bandwidth | ~31 Hz per signal |
| Text coding | Varicode (variable-length, self-synchronizing) |
| Pulse shaping | Raised-cosine |
| Error control | None (BPSK); convolutional code (QPSK variant) |

## History

PSK31 was developed by Peter Martinez (G3PLX) and released in 1998. It arrived just as
personal computers with sound cards made software modems practical, and its combination of
tiny bandwidth, good sensitivity, and free decoding software made it an immediate hit,
displacing much everyday RTTY chat on HF.[^spec] The Varicode alphabet — inspired by the
efficiency ideas behind Morse code — was a key part of that success.

## Deployment

PSK31 lives on the HF amateur bands, clustered in well-known digital watering holes such as
14.070 MHz on 20 m, where many narrow signals pack into a single waterfall display. It is
used for conversational contacts, DXing, and casual nets worldwide.

## Decoding it with GopherTrunk

GopherTrunk does not decode PSK31 — it is a trunked land-mobile scanner, not an HF
keyboard-mode decoder. PSK31 is received with an SSB receiver or SDR feeding audio into a
multimode program such as **fldigi**, Digital Master, or MultiPSK, which recovers the
differential phase and decodes Varicode. Its extreme narrowness means the front end mainly
needs frequency stability and a clean audio slice.

## Sources

[^wiki]: [PSK31](https://en.wikipedia.org/wiki/PSK31) — Wikipedia, for the differential BPSK waveform, 31.25-baud rate, Varicode alphabet, and the mode's origin and use.
[^spec]: [PSK31 Specification](http://www.arrl.org/psk31-spec) — ARRL / G3PLX, the authoritative description of PSK31's modulation, Varicode coding, and idle-signaling behavior.
