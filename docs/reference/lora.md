---
slug: lora
title: LoRa
entry_type: protocol
category: wireless-data-iot
description: LoRa is a low-power, long-range modulation that encodes data as chirps (frequency sweeps), giving excellent sensitivity for IoT telemetry in ISM bands.
keywords: LoRa, chirp spread spectrum, CSS, LoRaWAN, IoT, long range, ISM band, dechirp, spreading factor, Semtech, 433 868 915 MHz, Meshtastic
aka: [LoRa, "chirp spread spectrum"]
autolink: true
infobox:
  - { label: Type, value: Low-power wide-area PHY (IoT) }
  - { label: Standards body, value: Semtech (proprietary PHY) / LoRa Alliance }
  - { label: Band, value: Sub-GHz ISM (433 / 868 / 915 MHz) + 2.4 GHz }
  - { label: Access, value: ALOHA (LoRaWAN MAC above the PHY) }
  - { label: Modulation, value: Chirp spread spectrum (CSS) }
  - { label: Spreading factors, value: SF7–SF12 }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [lorawan, meshtastic, lora-alliance, direct-sequence-spread-spectrum, frequency-shift-keying, bandwidth, signal-to-noise-ratio, internet-of-things]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/rf-sdr/other-signals/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/LoRa
  - https://lora-developers.semtech.com/documentation/tech-papers-and-guides/lora-and-lorawan/
---

**LoRa** (from "long range") is a low-power, long-range **modulation** that encodes
data as **chirps** — frequency sweeps that ramp linearly across the channel. Its
**chirp spread spectrum** (CSS) waveform spreads each symbol over a wide bandwidth,
giving LoRa exceptional sensitivity: it can be demodulated several dB *below* the
[noise floor](/reference/noise-floor/), which is what makes it popular for low-rate IoT
telemetry over kilometres in unlicensed ISM bands.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A spectrogram showing several up-chirps (rising frequency sweeps) that wrap around the channel edge, the time-frequency signature of LoRa symbols." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="20" width="380" height="80" fill="none" stroke="currentColor" stroke-opacity="0.4"/>
  <g stroke="currentColor" stroke-width="1.8" fill="none"><path d="M50 95 L110 25"/><path d="M120 95 L165 45 M172 100 L180 95"/><path d="M172 100 L200 78"/><path d="M210 95 L270 25"/><path d="M280 95 L340 25"/><path d="M350 95 L390 40"/></g>
  <text x="30" y="60" font-size="8" fill="currentColor" transform="rotate(-90 30 60)">freq</text>
  <text x="230" y="118" text-anchor="middle" font-size="8.5" fill="currentColor">time → · each ramp is one chirp symbol; its start offset encodes the value</text>
</svg>
<figcaption>LoRa encodes symbols as rising frequency chirps whose cyclic start offset carries the data; the diagonal sweeps are its waterfall signature.</figcaption>
</figure>

## Overview

A LoRa symbol is an **up-chirp** that sweeps the full channel bandwidth; the *value* of
the symbol is set by where in the sweep the chirp starts and wraps around the band edge
(a cyclic time shift). A receiver **de-chirps** by multiplying the incoming signal with a
locally generated **down-chirp**, which collapses each chirp to a constant tone. An
[FFT](/reference/fast-fourier-transform/) then reads off the tone's frequency bin, and
that bin index *is* the symbol. Because de-chirping concentrates a wideband chirp into a
single narrow bin, it delivers large processing gain — the mechanism behind LoRa's
sub-noise sensitivity, conceptually related to
[direct-sequence spread spectrum](/reference/direct-sequence-spread-spectrum/) but using
a chirp instead of a pseudorandom code.

Two parameters trade range against data rate. **Bandwidth** (typically 125, 250, or 500
kHz) sets the chirp's sweep width. The **spreading factor** SF7–SF12 sets the chirp
length: each step up doubles the symbol duration (2^SF chips per symbol), adding roughly
2.5 dB of link budget but halving the bit rate. High SF reaches farthest and slowest;
low SF is faster and shorter-range. Payloads are protected by a header CRC and forward
error correction (a configurable coding rate), with Gray-mapping and interleaving applied
to the symbols.

## Technical characteristics

| Property | Value |
|----------|-------|
| Modulation | Chirp spread spectrum (CSS), up/down-chirps |
| Bands | 433 / 868 / 915 MHz sub-GHz ISM (also 2.4 GHz) |
| Bandwidth | 125 / 250 / 500 kHz (sub-GHz) |
| Spreading factor | SF7–SF12 (2^SF chips/symbol) |
| Bit rate | ~0.25–50 kbps depending on SF and BW |
| Coding | Configurable FEC (4/5–4/8), CRC, interleaving |
| Sensitivity | down to about −137 dBm at SF12 |

## History

LoRa's chirp-modulation PHY was developed by the French startup Cycleo and acquired by
**Semtech** in 2012, which keeps the physical layer proprietary and sells the radio
chips. To standardise the network layer on top, industry members formed the **LoRa
Alliance** in 2015, which publishes the open **[LoRaWAN](/reference/lorawan/)**
specification — the MAC, addressing, and encryption that turn raw LoRa links into a
low-power wide-area network. The PHY and the network layer are thus governed
separately.[^wiki][^semtech]

## Deployment

LoRa underlies a large share of long-range IoT: utility metering, agricultural and
environmental sensors, asset tracking, and building automation, typically via LoRaWAN
star-of-stars networks with battery devices reporting to gateways. Beyond LoRaWAN, the
same PHY carries other protocols — most visibly **[Meshtastic](/reference/meshtastic/)**,
an open off-grid mesh-messaging project popular with hobbyists — and point-to-point
telemetry links. It competes with other LPWAN technologies such as NB-IoT and Sigfox.

## Decoding it with GopherTrunk

GopherTrunk does **not** decode LoRa. It targets trunked land-mobile voice and
control systems (P25, DMR, NXDN, TETRA and the like), and LoRa's chirp PHY and IoT MAC
sit outside that scope; recovering it needs a de-chirp/FFT demodulator and, for LoRaWAN,
the device session keys. LoRa is still a common and easily recognised sight on the
[waterfall](/reference/spectrogram/) in the 433/868/915 MHz bands — the diagonal chirp
ramps are unmistakable — so it is worth knowing even though GopherTrunk leaves it to
dedicated tools.

## Sources

[^wiki]: [LoRa](https://en.wikipedia.org/wiki/LoRa) — Wikipedia, for the chirp-spread-spectrum modulation, spreading factors and bandwidth trade-offs, ISM-band use, and the Semtech/LoRa Alliance split between PHY and LoRaWAN.
[^semtech]: [LoRa and LoRaWAN technical papers and guides](https://lora-developers.semtech.com/documentation/tech-papers-and-guides/lora-and-lorawan/) — Semtech, the chip vendor's primary documentation for the CSS physical layer, de-chirp reception, link-budget/spreading-factor behaviour, and sensitivity figures.
