---
slug: wspr
title: WSPR
entry_type: protocol
category: amateur-digital
description: "WSPR (Weak Signal Propagation Reporter) is an amateur-radio beacon mode using low-rate 4-FSK and convolutional coding to probe HF propagation paths with milliwatt-level signals."
keywords: WSPR, whisper, Weak Signal Propagation Reporter, beacon, 4-FSK, MFSK, WSJT-X, propagation, WSPRnet, K1JT, amateur radio
aka: [WSPR]
autolink: true
infobox:
  - { label: Type, value: Propagation-beacon amateur mode }
  - { label: Developed by, value: Joe Taylor (K1JT) }
  - { label: Introduced, value: 2008 }
  - { label: Modulation, value: 4-FSK, 1.4648 baud }
  - { label: Timing, value: ~111 s transmission, 2-min slots }
  - { label: FEC, value: K=32, rate-1/2 convolutional }
  - { label: GopherTrunk support, value: Not decoded (use WSJT-X) }
see_also: [ft8, joe-taylor, m-ary-fsk, convolutional-code]
cite_urls:
  - https://en.wikipedia.org/wiki/WSPR_(amateur_radio_software)
  - https://www.physics.princeton.edu/pulsar/k1jt/WSPR_3.0_User.pdf
---

**WSPR** (**Weak Signal Propagation Reporter**, pronounced "whisper") is an
amateur-radio **beacon mode** built not to hold conversations but to map which radio
paths are open. Stations transmit a tiny fixed message — callsign, grid locator, and
transmit power — using very slow 4-tone FSK, and receivers worldwide upload every decode
to a central database, turning the amateur bands into a live propagation sensor.[^wiki]
Because the mode sacrifices data rate for sensitivity, WSPR signals of a few hundred
milliwatts are routinely decoded across oceans.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A WSPR beacon transmits a callsign, grid locator, and power level, which distant receivers decode and upload to a global propagation database." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="wsprar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="25" y="45" width="95" height="34" fill="currentColor" fill-opacity="0.15" stroke="currentColor"/>
  <text x="72" y="60" font-size="8.5" fill="currentColor" text-anchor="middle">beacon TX</text>
  <text x="72" y="72" font-size="7.5" fill="currentColor" text-anchor="middle">call·grid·dBm</text>
  <line x1="120" y1="62" x2="210" y2="62" stroke="currentColor" marker-end="url(#wsprar)"/>
  <text x="165" y="55" font-size="7.5" fill="currentColor" text-anchor="middle">HF path</text>
  <rect x="210" y="45" width="90" height="34" fill="none" stroke="currentColor"/>
  <text x="255" y="65" font-size="8.5" fill="currentColor" text-anchor="middle">RX / decode</text>
  <line x1="300" y1="62" x2="380" y2="62" stroke="currentColor" marker-end="url(#wsprar)"/>
  <rect x="380" y="45" width="70" height="34" fill="currentColor" fill-opacity="0.15" stroke="currentColor"/>
  <text x="415" y="65" font-size="8" fill="currentColor" text-anchor="middle">WSPRnet</text>
  <text x="230" y="103" font-size="8" fill="currentColor" text-anchor="middle">4-FSK · 1.46 baud · rate-1/2 K=32 convolutional code</text>
</svg>
<figcaption>WSPR beacons a callsign, grid, and power; distant receivers decode and log spots to a global database, revealing open propagation paths.</figcaption>
</figure>

## Overview

A WSPR transmission is entirely automated. On even two-minute boundaries a station keys up
for about 111 seconds, sending a 50-bit message spread across 162 slow tones. Receivers
running the same software decode any signals in a 200 Hz window, extract the reported
power, estimate the received SNR, and upload the result — with timestamps — to WSPRnet.
The aggregate is a continuously updated worldwide map of band openings.

## Technical characteristics

| Property | Value |
|----------|-------|
| Modulation | 4-FSK, continuous-phase, ~1.46 Hz tone spacing |
| Symbol rate | 1.4648 baud (162 symbols) |
| Occupied bandwidth | ~6 Hz per signal |
| Slot length | 2 min (110.6 s of tones) |
| Message payload | 50 bits (28-bit call, 15-bit grid, 7-bit power) |
| FEC | Constraint-length 32, rate-1/2 [convolutional code](/reference/convolutional-code/) |
| Threshold | ≈ −28 dB SNR (2.5 kHz reference) |

## History

WSPR was released in 2008 by [Joe Taylor](/reference/joe-taylor/) (K1JT) as part of the
WSJT family of weak-signal programs. Its convolutional-coded, ultra-narrow waveform
pushed decoding several dB below what conversational modes could reach, and the paired
WSPRnet reporting site made it a standard tool for propagation research. Much of the
coding philosophy later informed conversational modes like [FT8](/reference/ft8/).[^man]

## Deployment

WSPR runs on the HF amateur bands (with dedicated 200 Hz windows such as 14.0956 MHz on
20 m) and increasingly on LF/MF and VHF. Purpose-built low-power beacons and networked
receivers make it a widely cited source of real-time ionospheric-propagation data.

## Decoding it with GopherTrunk

GopherTrunk does not decode WSPR — WSPR is an HF weak-signal beacon mode outside the scope
of a trunked land-mobile scanner. It is received with an SSB-capable receiver or SDR
feeding audio into **WSJT-X** or the dedicated WSPR software, with the PC clock locked to
UTC; a stable, frequency-accurate front end matters because the signal is only a few hertz
wide.

## Sources

[^wiki]: [WSPR (amateur radio software)](https://en.wikipedia.org/wiki/WSPR_(amateur_radio_software)) — Wikipedia, for the beacon concept, 4-FSK waveform, message contents, and the WSPRnet reporting network.
[^man]: [WSPR User's Guide](https://www.physics.princeton.edu/pulsar/k1jt/WSPR_3.0_User.pdf) — K1JT, the authoritative description of the WSPR protocol, timing, message coding, and operating procedure.
