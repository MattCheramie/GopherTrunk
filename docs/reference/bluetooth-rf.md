---
slug: bluetooth-rf
title: Bluetooth (RF layer)
entry_type: protocol
category: wireless-data-iot
description: "Bluetooth's RF layer is a 2.4 GHz frequency-hopping spread-spectrum air interface using GFSK across 79 (Classic) or 40 (LE) channels for short-range personal-area links."
keywords: Bluetooth, Bluetooth RF, Bluetooth Classic, BR/EDR, FHSS, frequency hopping, GFSK, 2.4 GHz ISM, adaptive frequency hopping, personal area network, piconet
aka: [Bluetooth Classic PHY, BR/EDR, Bluetooth radio, EDR, "Enhanced Data Rate"]
autolink: true
infobox:
  - { label: Type, value: Short-range wireless PHY }
  - { label: Standards body, value: Bluetooth SIG }
  - { label: Introduced, value: "1998–1999" }
  - { label: Access, value: "FHSS, 1600 hops/s" }
  - { label: Channel spacing, value: 1 MHz (79 channels) }
  - { label: Modulation, value: "GFSK (π/4-DQPSK, 8DPSK in EDR)" }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [bluetooth, bluetooth-le, frequency-hopping-spread-spectrum, gfsk, bluetooth-sig, internet-of-things]
cite_urls:
  - https://en.wikipedia.org/wiki/Bluetooth
  - https://en.wikipedia.org/wiki/Frequency-hopping_spread_spectrum
---

**Bluetooth (RF layer)** is the short-range radio physical layer that carries
Bluetooth personal-area links over the 2.4 GHz ISM band using
[frequency-hopping spread spectrum](/reference/frequency-hopping-spread-spectrum/) and
[GFSK](/reference/gfsk/) modulation.[^wiki] Standardised by the
[Bluetooth SIG](/reference/bluetooth-sig/), it hops rapidly across the band to dodge
interference and eavesdroppers; the everyday concept is covered at
[Bluetooth](/reference/bluetooth/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Bluetooth hops its carrier frequency 1600 times per second across many 1 MHz channels in the 2.4 GHz band, shown as a staircase of packets landing on different frequencies over time." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="btrfar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="40" y1="120" x2="440" y2="120" stroke="currentColor" stroke-opacity="0.5" marker-end="url(#btrfar)"/>
  <line x1="40" y1="120" x2="40" y2="20" stroke="currentColor" stroke-opacity="0.5" marker-end="url(#btrfar)"/>
  <text x="435" y="137" text-anchor="end" font-size="9" fill="currentColor">time →</text>
  <text x="16" y="30" font-size="9" fill="currentColor">f</text>
  <g fill="currentColor" fill-opacity="0.35" stroke="currentColor" stroke-width="0.8">
    <rect x="55" y="90" width="34" height="12"/><rect x="95" y="40" width="34" height="12"/><rect x="135" y="70" width="34" height="12"/><rect x="175" y="30" width="34" height="12"/><rect x="215" y="100" width="34" height="12"/><rect x="255" y="55" width="34" height="12"/><rect x="295" y="80" width="34" height="12"/><rect x="335" y="45" width="34" height="12"/><rect x="375" y="95" width="34" height="12"/>
  </g>
  <text x="230" y="14" text-anchor="middle" font-size="10" fill="currentColor">1600 hops/s across 79 × 1 MHz channels</text>
</svg>
<figcaption>Bluetooth Classic hops its carrier 1600 times per second across 79 one-megahertz channels, spreading each packet over a different frequency.</figcaption>
</figure>

## Overview

Bluetooth Classic (BR/EDR) divides the 2.402–2.480 GHz band into 79 channels spaced
1 MHz apart and hops between them 1600 times per second in a pseudo-random sequence
shared by the connected devices, or *piconet*. Each 625 µs slot carries a packet on a
new frequency, so narrowband interferers only spoil the occasional hop. **Adaptive
frequency hopping** (AFH) further improves coexistence by skipping channels known to be
busy with Wi-Fi. The basic rate uses GFSK at 1 Mbit/s; **Enhanced Data Rate** (EDR)
switches the packet payload to π/4-DQPSK (2 Mbit/s) or 8DPSK (3 Mbit/s).

## Technical characteristics

| Property | Value |
|----------|-------|
| Band | 2.402–2.480 GHz ISM |
| Channels | 79 × 1 MHz (Classic) |
| Hop rate | 1600 hops/s |
| Modulation | GFSK (BR); π/4-DQPSK, 8DPSK (EDR) |
| Symbol rate | 1 Msym/s |
| Bit rate | 1 / 2 / 3 Mbit/s |
| Coexistence | Adaptive frequency hopping |

The spread-spectrum design trades peak throughput for resilience in the shared ISM
band. A companion low-power variant, [Bluetooth LE](/reference/bluetooth-le/), uses a
different, simpler channel plan aimed at battery devices.

## History

Ericsson began the work in 1994; the Bluetooth SIG published version 1.0 in 1999. EDR
arrived with version 2.0 (2004), and Bluetooth 4.0 (2010) introduced the separate Low
Energy layer. The name and the runic logo reference the 10th-century Danish king Harald
"Bluetooth" Gormsson.

## Deployment

Bluetooth Classic dominates wireless audio (headsets, speakers, car kits) and legacy
peripherals. Newer sensor, wearable, and [IoT](/reference/internet-of-things/) devices
increasingly favour the LE variant for its lower power draw.

## Decoding it with GopherTrunk

GopherTrunk does **not** decode Bluetooth. Following a frequency-hopping link requires
tracking 1600 pseudo-random hops per second across 80 MHz of spectrum with knowledge of
the piconet's hop sequence and clock — a very different problem from GopherTrunk's
fixed-channel land-mobile decoders. To GopherTrunk, Bluetooth is simply another
occupant of the noisy 2.4 GHz band that an SDR operator learns to recognise.

## Sources

[^wiki]: [Bluetooth](https://en.wikipedia.org/wiki/Bluetooth) — Wikipedia, on the Bluetooth radio layer, its 2.4 GHz frequency-hopping GFSK air interface, EDR modulations, and the SIG.
