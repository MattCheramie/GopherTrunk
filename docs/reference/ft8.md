---
slug: ft8
title: FT8
entry_type: protocol
category: amateur-digital
description: "FT8 is a weak-signal amateur-radio digital mode using 8-FSK in 15-second transmit/receive slots with LDPC forward error correction to complete contacts below the noise floor."
keywords: FT8, weak signal, 8-FSK, MFSK, WSJT-X, LDPC, 15 second, Franke Taylor, amateur radio digital mode, HF propagation
aka: [FT8]
autolink: true
infobox:
  - { label: Type, value: Weak-signal amateur digital mode }
  - { label: Developed by, value: Steven Franke (K9AN) & Joe Taylor (K1JT) }
  - { label: Introduced, value: 2017 (WSJT-X) }
  - { label: Modulation, value: 8-FSK, 6.25 baud }
  - { label: Timing, value: 15-second T/R sequences }
  - { label: FEC, value: LDPC(174,91) + 14-bit CRC }
  - { label: GopherTrunk support, value: Not decoded (use WSJT-X) }
see_also: [m-ary-fsk, ldpc-code, wspr, joe-taylor, ft4, jt65]
cite_urls:
  - https://en.wikipedia.org/wiki/FT8
  - https://wsjt.sourceforge.io/FT4_FT8_QEX.pdf
---

**FT8** is a **weak-signal amateur-radio digital mode** that completes minimal contacts —
callsigns, grid locators, and signal reports — at signal levels well below what the ear
can hear. It uses [8-ary FSK](/reference/m-ary-fsk/) in tightly synchronized
15-second slots and an [LDPC](/reference/ldpc-code/) error-correcting code, so a decode
can succeed at signal-to-noise ratios around −21 dB in a 2.5 kHz reference bandwidth.[^wiki]
Named for its designers **F**ranke and **T**aylor and its **8** tones, FT8 became the most
popular mode on the amateur HF bands within a few years of its 2017 release.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="FT8 transmits one of eight FSK tones every 0.16 seconds inside repeating 15-second time slots, decoded by LDPC forward error correction." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="ft8ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="30" y1="120" x2="440" y2="120" stroke="currentColor" stroke-width="1" marker-end="url(#ft8ar)"/>
  <line x1="30" y1="30" x2="30" y2="120" stroke="currentColor" stroke-width="1" marker-end="url(#ft8ar)"/>
  <text x="450" y="123" font-size="8" fill="currentColor" text-anchor="end">time</text>
  <text x="20" y="34" font-size="8" fill="currentColor" text-anchor="end">tone</text>
  <g stroke="currentColor" stroke-width="2" fill="none">
    <path d="M40 100 H70 M70 50 H100 M100 80 H130 M130 40 H160 M160 90 H190 M190 60 H220 M220 100 H250 M250 70 H280"/>
  </g>
  <g stroke="currentColor" stroke-width="0.6" stroke-dasharray="2 3" opacity="0.5"><line x1="70" y1="50" x2="70" y2="100"/><line x1="100" y1="50" x2="100" y2="80"/><line x1="130" y1="40" x2="130" y2="80"/><line x1="160" y1="40" x2="160" y2="90"/><line x1="190" y1="60" x2="190" y2="90"/><line x1="220" y1="60" x2="220" y2="100"/><line x1="250" y1="70" x2="250" y2="100"/></g>
  <text x="160" y="140" font-size="8.5" fill="currentColor" text-anchor="middle">8-FSK symbols · 6.25 baud · 79 symbols per 15 s slot</text>
</svg>
<figcaption>FT8 sends 79 tones drawn from an 8-tone alphabet in each 15-second slot; LDPC coding recovers the 77-bit message below the noise floor.</figcaption>
</figure>

## Overview

An FT8 exchange is deliberately spartan. Stations alternate transmit and receive on
15-second boundaries locked to UTC, so accurate clock synchronization (usually via NTP)
is essential. A full QSO — call, grid, signal report, roger-report, and 73 — takes about
a minute. Because every station in a passband transmits in the same slots, a single
receiver decodes dozens of overlapping signals at once across the audio spectrum.

## Technical characteristics

| Property | Value |
|----------|-------|
| Modulation | 8-FSK, Gaussian-smoothed, 6.25 Hz tone spacing |
| Symbol rate | 6.25 baud (79 symbols per transmission) |
| Occupied bandwidth | ~50 Hz per signal |
| Slot length | 15 s (12.64 s of tones) |
| Message payload | 77 bits |
| FEC | [LDPC](/reference/ldpc-code/)(174,91) with 14-bit CRC |
| Sync | three 7×7 Costas arrays |
| Threshold | ≈ −21 dB SNR (2.5 kHz reference) |

## History

FT8 was introduced in mid-2017 by Steven Franke (K9AN) and Nobel laureate
[Joe Taylor](/reference/joe-taylor/) (K1JT) as part of the free WSJT-X software suite.
It combined the sub-noise sensitivity of earlier modes like [JT65](/reference/jt65/) with
a far faster 15-second cadence, and the modern LDPC code replaced the older
Reed-Solomon approach. Its speed and sensitivity made it dominant on the HF bands almost
immediately.[^qex]

## Deployment

FT8 runs worldwide on the HF amateur bands (notably 20 m at 14.074 MHz) and on VHF/UHF for
weak-signal work. Decodes are widely uploaded to spotting networks such as PSK Reporter,
producing real-time global propagation maps. A faster sibling, [FT4](/reference/ft4/),
uses 7.5-second slots for contesting.

## Decoding it with GopherTrunk

GopherTrunk does not decode FT8 — it is a trunked-radio scanner focused on land-mobile
systems (P25, DMR, NXDN, TETRA and similar), not HF weak-signal modes. FT8 is received
with a general-coverage SSB receiver or SDR feeding audio into **WSJT-X**, the reference
decoder, or compatible software such as JTDX. Any SDR that can deliver a clean 2–3 kHz
USB audio slice on the right band and dial frequency, with an accurately set clock, is a
suitable front end.

## Sources

[^wiki]: [FT8](https://en.wikipedia.org/wiki/FT8) — Wikipedia, for the 8-FSK air interface, 15-second timing, LDPC coding, and the mode's origin and popularity.
[^qex]: [The FT4 and FT8 Communication Protocols](https://wsjt.sourceforge.io/FT4_FT8_QEX.pdf) — Franke, Somerville & Taylor, QEX, the authoritative description of the FT8 waveform, message format, and coding.
