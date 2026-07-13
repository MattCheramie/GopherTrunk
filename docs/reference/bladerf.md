---
slug: bladerf
title: BladeRF
entry_type: hardware
category: sdr-devices
description: "BladeRF is Nuand's line of USB 3.0 full-duplex transceiver SDRs built around an on-board FPGA, spanning HF/VHF up to 6 GHz."
keywords: BladeRF, bladeRF 2.0 micro, Nuand, FPGA SDR, LMS6002D, AD9361, Cyclone IV, Cyclone V, full duplex transceiver, USB 3.0 SDR, xA4, xA9
aka: [BladeRF, blade RF, bladeRF 2.0 micro]
autolink: true
infobox:
  - { label: Type, value: USB full-duplex transceiver SDR }
  - { label: Vendor, value: Nuand }
  - { label: On-board, value: Intel/Altera FPGA }
  - { label: ADC, value: 12-bit }
  - { label: Range, value: "47 MHz – 6 GHz (2.0 micro)" }
  - { label: Bandwidth, value: up to ~56 MHz }
  - { label: TX, value: Yes (full duplex) }
  - { label: Typical price, value: "$480 – $720" }
see_also: [software-defined-radio, field-programmable-gate-array, hackrf, usrp-ettus, limesdr, rtl-sdr]
cite_urls:
  - https://en.wikipedia.org/wiki/BladeRF
  - https://www.nuand.com/bladerf-2-0-micro/
---

**BladeRF** is a line of USB 3.0
[software-defined radio](/reference/software-defined-radio/) transceivers from the
company **Nuand**, distinguished from cheaper receivers by an on-board
[FPGA](/reference/field-programmable-gate-array/) and a **full-duplex** transmit and
receive path.[^wiki] Where an [RTL-SDR](/reference/rtl-sdr/) is a receive-only dongle,
a BladeRF is a two-way radio platform that can both listen and transmit while running
custom real-time DSP in fabric before samples ever reach the host.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="Block diagram of a BladeRF: an RF transceiver chip for transmit and receive connects to an on-board FPGA, which streams samples over USB 3.0 to a computer." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="brar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="34" y="40">RX ant</text>
    <text x="34" y="90">TX ant</text>
    <rect x="80" y="34" width="110" height="52" rx="4" fill="none" stroke="currentColor" stroke-opacity="0.7"/>
    <text x="135" y="56">RF transceiver</text><text x="135" y="70" font-size="7">AD9361 / LMS6002D</text>
    <rect x="222" y="34" width="96" height="52" rx="4" fill="none" stroke="currentColor" stroke-opacity="0.7"/>
    <text x="270" y="56">FPGA</text><text x="270" y="70" font-size="7">Cyclone IV / V</text>
    <text x="400" y="62">host (USB 3.0)</text>
  </g>
  <g stroke="currentColor" stroke-opacity="0.6" fill="none">
    <line x1="60" y1="40" x2="78" y2="45" marker-end="url(#brar)"/>
    <line x1="78" y1="80" x2="60" y2="86" marker-end="url(#brar)"/>
    <line x1="190" y1="60" x2="220" y2="60" marker-end="url(#brar)"/>
    <line x1="318" y1="60" x2="352" y2="60" marker-end="url(#brar)"/>
  </g>
</svg>
<figcaption>An RF transceiver front end feeds an on-board FPGA that can process samples before they cross USB 3.0 to the host.</figcaption>
</figure>

## Overview

Nuand launched the original BladeRF in 2013 through a Kickstarter campaign, positioning
it between the low-cost hobbyist dongles and the far pricier
[USRP](/reference/usrp-ettus/) research platforms. Two things set the family apart: a
genuine **transmit** capability with independent RX and TX chains (true full duplex),
and a user-programmable **FPGA** that lets developers offload filtering, channelisation,
or an entire modem into hardware. That FPGA is the defining feature — it is why the
BladeRF appears in GNU Radio flowgraphs, cellular-network experiments, and custom
protocol work rather than just spectrum monitoring.

## Variants

Two hardware generations exist, with meaningfully different coverage:

- **BladeRF x40 / x115 (2013).** The original board pairs a Lime Microsystems
  **LMS6002D** RF transceiver with an Intel/Altera **Cyclone IV** FPGA (40k or 115k
  logic elements — the number in the model name). It tunes roughly **300 MHz – 3.8 GHz**
  with up to ~28 MHz of usable [bandwidth](/reference/bandwidth/) and 12-bit converters.
- **BladeRF 2.0 micro xA4 / xA5 / xA9 (2018).** The current generation moves to the
  Analog Devices **AD9361** transceiver (the same silicon behind many mid-range SDRs)
  and a **Cyclone V** FPGA. It extends coverage to about **47 MHz – 6 GHz**, widens the
  channel to as much as **~56 MHz**, and adds a 2×2 MIMO option, bias-tee outputs, and an
  expansion header. The suffix again denotes FPGA size (xA4 = 49k, xA9 = 301k logic
  elements).

Both generations use **12-bit** analog-to-digital and digital-to-analog converters — a
large dynamic-range improvement over the 8-bit [RTL2832U](/reference/rtl2832u/) — and
connect over USB 3.0 (SuperSpeed) to sustain the high sample rates that wide bandwidth
demands.

## In practice

The FPGA and the transmit path put the BladeRF in a different class of use than a
scanner dongle. Typical projects include running a small GSM or LTE base station in a
lab, building custom modems, passive radar, and any workflow where DSP must happen at
line rate before the USB bus becomes a bottleneck. The trade-off is cost and complexity:
at **$480–$720** it is many times the price of an RTL-SDR, transmitting legally requires
appropriate licensing and filtering, and getting the most from the board means writing
or loading FPGA images. For pure reception it is often overkill compared with a
purpose-built receiver.

## Relevance to GopherTrunk

GopherTrunk is a **receive-only** trunking decoder, so it uses none of the BladeRF's
transmit capability and does not load custom FPGA images. Where a BladeRF can help is as
a plain wideband **IQ source** through a SoapySDR-style interface: its 12-bit front end
and wide capture bandwidth can channelise several control channels at once, much like an
[Airspy](/reference/airspy/) does in GopherTrunk's wideband role. For most users a
BladeRF is more radio than trunk-tracking needs — an RTL-SDR pool or an Airspy is the
cheaper, better-matched tool — but if you already own one, it is a capable capture front
end. See the [hardware guide](/hardware.html) for GopherTrunk's tested devices.

## Sources

[^wiki]: [BladeRF](https://en.wikipedia.org/wiki/BladeRF) — Wikipedia, on Nuand's BladeRF hardware, FPGA, transceivers and coverage.
[^nuand]: [bladeRF 2.0 micro](https://www.nuand.com/bladerf-2-0-micro/) — Nuand, product specifications for the AD9361-based 2.0 micro generation.
