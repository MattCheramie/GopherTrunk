---
slug: plutosdr
title: ADALM-Pluto (PlutoSDR)
entry_type: hardware
category: sdr-devices
description: ADALM-Pluto (PlutoSDR) is Analog Devices' low-cost learning SDR, an AD9363-based transmit-and-receive radio covering 325 MHz–3.8 GHz with 12-bit sampling.
keywords: ADALM-Pluto, PlutoSDR, Pluto SDR, AD9363, AD9364, Analog Devices, learning SDR, transceiver, 12-bit, full duplex, Zynq
aka: [PlutoSDR, ADALM-Pluto, Pluto]
autolink: true
infobox:
  - { label: Type, value: Learning SDR transceiver }
  - { label: Vendor/Chip, value: "Analog Devices, AD9363" }
  - { label: ADC, value: 12-bit }
  - { label: Range, value: 325 MHz – 3.8 GHz }
  - { label: Bandwidth, value: up to ~20 MHz }
  - { label: TX, value: Yes (full-duplex) }
  - { label: Typical price, value: ~US$230 }
see_also: [software-defined-radio, iq-data, mimo, gnuradio, hackrf]
cite_urls:
  - https://en.wikipedia.org/wiki/ADALM-PLUTO
  - https://www.analog.com/en/resources/evaluation-hardware-and-software/evaluation-boards-kits/adalm-pluto.html
---

**ADALM-Pluto** (often **PlutoSDR**) is Analog Devices' low-cost learning
[software-defined radio](/reference/software-defined-radio/): a pocket-sized,
transmit-and-receive radio built on the **AD9363** transceiver, covering **325 MHz to
3.8 GHz** with **12-bit** [ADC](/reference/analog-to-digital-converter/) sampling.[^wiki] It
was designed as a teaching tool — an active learning module — pairing capable RF with
Analog Devices' documentation and courseware.[^adi]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A frequency coverage bar for the ADALM-Pluto from about 325 megahertz to 3.8 gigahertz on an axis from 0 to 6 gigahertz, marked as a transmit-and-receive learning radio." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="430" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="30" y="86">0</text><text x="163" y="86">2 GHz</text><text x="296" y="86">4 GHz</text><text x="430" y="86">6 GHz</text></g>
  <rect x="52" y="40" width="200" height="20" rx="3" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.3"/>
  <text x="230" y="28" text-anchor="middle" font-size="10" fill="currentColor">ADALM-Pluto (325 MHz–3.8 GHz) TX + RX</text>
</svg>
<figcaption>Pluto is a compact full-duplex transceiver spanning UHF through low microwave, aimed at learners.</figcaption>
</figure>

## Overview

The Pluto's job is to make hands-on RF affordable: for the price of a mid-range dongle you
get a full transmit-and-receive radio you can build modulators, demodulators, and links
around. It streams [IQ](/reference/iq-data/) over USB while the host runs the signal
processing, and it integrates cleanly with MATLAB/Simulink, [GNU Radio](/reference/gnuradio/),
and Python via `libiio`/`pyadi-iio`. Because it transmits, learners can generate a signal
and receive it on the same device — closing the loop that a receive-only SDR cannot.

## What it is

Inside is a Xilinx **Zynq** SoC (an ARM processor beside an FPGA fabric) paired with the
AD9363 [zero-IF](/reference/zero-if/) transceiver. The Zynq runs embedded Linux and Analog
Devices' **IIO** framework, which exposes the radio to the host over USB (and optionally
Ethernet-over-USB). Instantaneous [bandwidth](/reference/bandwidth/) is up to about 20 MHz.
A well-known detail: the AD9363's stated 325 MHz–3.8 GHz range is a specification limit, not
a hard wall — the closely related AD9364 covers 70 MHz–6 GHz, and a widely documented
firmware tweak lets many Pluto units tune the wider AD9364 range and enable a second
channel. Treat that as unspecified/unguaranteed operation rather than a rated feature.

## Variants

The Pluto is a single product rather than a family, but a couple of distinctions matter:

- **Rev B vs Rev C/D** — later hardware revisions changed the RF connectors (from SMA to
  U.FL on some revs) and internal details; software is common across them.
- **AD9363 vs "unlocked" AD9364 behaviour** — the extended tuning range and dual channels
  are the unofficial modification noted above, not a factory configuration.

Its natural peers are other affordable transceivers such as the
[HackRF One](/reference/hackrf/) and [LimeSDR](/reference/limesdr/) Mini; the Pluto's
differentiator is its explicit learning focus and tight MATLAB/Python tooling.

## Relevance to SDR

The Pluto is squarely an education and prototyping device: teaching digital communications,
building modem experiments, testing a receiver against a self-generated signal, and general
UHF/microwave tinkering. Its full-duplex path and courseware make it a common classroom SDR.

For trunking reception it is a poor fit and not the intended use: its lower frequency limit
(~325 MHz on stock firmware) misses VHF systems entirely, its dynamic range is ordinary, and
transmit is irrelevant to scanning. GopherTrunk offers native USB backends for RTL-SDR,
HackRF, and Airspy, not for the Pluto's IIO stack, so any use would rely on an external
bridge. It belongs in the learning and RF-development category rather than the scanning
toolbox.

## Sources

[^wiki]: [ADALM-PLUTO](https://en.wikipedia.org/wiki/ADALM-PLUTO) — Wikipedia, on the PlutoSDR, its AD9363 transceiver, Zynq SoC, and the AD9364 frequency-range modification.
[^adi]: [ADALM-PLUTO](https://www.analog.com/en/resources/evaluation-hardware-and-software/evaluation-boards-kits/adalm-pluto.html) — Analog Devices, official product/learning page with tuning range, bandwidth, and tooling.
