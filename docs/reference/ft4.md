---
slug: ft4
title: FT4
entry_type: protocol
category: amateur-digital
description: "FT4 is a fast weak-signal amateur-radio digital mode, a 4-GFSK contesting cousin of FT8 that uses 7.5-second transmit/receive slots and LDPC forward error correction."
keywords: FT4, weak signal, 4-GFSK, MFSK, WSJT-X, LDPC, contesting, 7.5 second, amateur radio digital mode, FT8
aka: [FT4]
autolink: true
infobox:
  - { label: Type, value: Weak-signal amateur digital mode }
  - { label: Developed by, value: Steven Franke (K9AN) & Joe Taylor (K1JT) }
  - { label: Introduced, value: 2019 (WSJT-X) }
  - { label: Modulation, value: 4-GFSK, 20.83 baud }
  - { label: Timing, value: 7.5-second T/R sequences }
  - { label: FEC, value: LDPC(174,91) + 14-bit CRC }
  - { label: GopherTrunk support, value: Not decoded (use WSJT-X) }
see_also: [ft8, m-ary-fsk, ldpc-code, joe-taylor]
cite_urls:
  - https://en.wikipedia.org/wiki/FT4
  - https://wsjt.sourceforge.io/FT4_FT8_QEX.pdf
---

**FT4** is a **fast weak-signal amateur-radio digital mode** built for contesting — a
quicker sibling of [FT8](/reference/ft8/) that trades a little sensitivity for roughly
double the rate. It uses 4-tone GFSK in 7.5-second transmit/receive slots and carries the
same 77-bit messages protected by the same [LDPC](/reference/ldpc-code/) code, so a full
contact takes under half a minute.[^wiki] FT4 shares FT8's message structure and much of
its decoder, but is optimized for the rapid-fire exchanges of a radio contest.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="FT4 uses four FSK tones at 20.83 baud in 7.5-second slots, about twice as fast as FT8's 15-second slots." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1"><line x1="30" y1="55" x2="440" y2="55"/><line x1="30" y1="105" x2="440" y2="105"/></g>
  <g stroke="currentColor" stroke-width="1.2" opacity="0.7"><line x1="30" y1="40" x2="30" y2="70"/><line x1="235" y1="40" x2="235" y2="70"/><line x1="440" y1="40" x2="440" y2="70"/></g>
  <g stroke="currentColor" stroke-width="1.2" opacity="0.7"><line x1="30" y1="90" x2="30" y2="120"/><line x1="132" y1="90" x2="132" y2="120"/><line x1="235" y1="90" x2="235" y2="120"/><line x1="338" y1="90" x2="338" y2="120"/><line x1="440" y1="90" x2="440" y2="120"/></g>
  <text x="30" y="30" font-size="8.5" fill="currentColor">FT8 — 15 s slots</text>
  <text x="30" y="82" font-size="8.5" fill="currentColor">FT4 — 7.5 s slots (≈2× faster)</text>
</svg>
<figcaption>FT4 halves FT8's slot length to 7.5 seconds, roughly doubling contact rate at the cost of a few dB of sensitivity.</figcaption>
</figure>

## Overview

FT4 was designed specifically to answer FT8's main weakness for contesters: at one QSO
per minute, FT8 is too slow to run a busy contest exchange. FT4 keeps the sub-noise
philosophy — synchronized time slots, a fixed 77-bit message, and strong forward error
correction — but shortens the cycle and widens each signal slightly to move data faster.

## Technical characteristics

| Property | Value |
|----------|-------|
| Modulation | 4-GFSK, ~23.4 Hz tone spacing |
| Symbol rate | 20.83 baud (105 symbols per transmission) |
| Occupied bandwidth | ~90 Hz per signal |
| Slot length | 7.5 s (4.48 s of tones) |
| Message payload | 77 bits (same as FT8) |
| FEC | [LDPC](/reference/ldpc-code/)(174,91) with 14-bit CRC |
| Sync | 4×4 Costas-style arrays + ramp symbols |
| Threshold | ≈ −17.5 dB SNR (2.5 kHz reference) |

## History

FT4 was released in 2019 by Steven Franke (K9AN) and [Joe Taylor](/reference/joe-taylor/)
(K1JT) as an experimental contesting mode in WSJT-X, after FT8's runaway popularity made
clear that a faster variant was wanted for high-rate operating.[^qex] It reuses FT8's
source-encoding and LDPC coding, differing mainly in modulation and timing.

## Deployment

FT4 is used on the HF amateur bands during contests, on designated sub-band frequencies
(for example 14.080 MHz on 20 m). Outside contest weekends it sees far less traffic than
[FT8](/reference/ft8/), which remains the everyday weak-signal workhorse.

## Decoding it with GopherTrunk

GopherTrunk does not decode FT4; it is a trunked land-mobile scanner, not an HF
weak-signal decoder. FT4 is received the same way as FT8 — an SSB receiver or SDR feeding
a clean USB audio slice into **WSJT-X** (or JTDX), with the PC clock synchronized to UTC.

## Sources

[^wiki]: [FT4](https://en.wikipedia.org/wiki/FT4) — Wikipedia, for FT4's role as a fast contesting mode, its 4-GFSK modulation, 7.5-second timing, and relationship to FT8.
[^qex]: [The FT4 and FT8 Communication Protocols](https://wsjt.sourceforge.io/FT4_FT8_QEX.pdf) — Franke, Somerville & Taylor, QEX, the primary reference for the FT4 waveform, timing, and message coding.
