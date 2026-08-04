---
slug: usrp-ettus
title: USRP (Ettus Research)
entry_type: hardware
category: sdr-devices
description: USRP is Ettus Research's family of high-performance transmit-and-receive software-defined radios (B200/B210, N-series, X-series) driven by the open UHD stack for research and industry.
keywords: USRP, Ettus Research, UHD, B200, B210, N210, N310, X310, software radio, research SDR, GNU Radio, full duplex, MIMO
aka: [USRP, Ettus USRP, Universal Software Radio Peripheral]
autolink: true
infobox:
  - { label: Type, value: SDR transceiver family }
  - { label: Vendor/Chip, value: "Ettus Research (NI), AD936x" }
  - { label: ADC, value: 12–16-bit (model-dependent) }
  - { label: Range, value: DC – 6 GHz (typical) }
  - { label: Bandwidth, value: up to ~160 MHz (model-dependent) }
  - { label: TX, value: Yes (full-duplex) }
  - { label: Typical price, value: "~US$1,000 and up" }
see_also: [software-defined-radio, gnuradio, mimo, field-programmable-gate-array, soapysdr]
cite_urls:
  - https://en.wikipedia.org/wiki/Universal_Software_Radio_Peripheral
  - https://www.ettus.com/
---

**USRP** (Universal Software Radio Peripheral) is Ettus Research's long-running family of
high-performance transmit-and-receive
[software-defined radios](/reference/software-defined-radio/), spanning the compact
**B200/B210** through the networked **N-series** and the flagship **X-series**.[^wiki] All
are driven by the open-source **UHD** (USRP Hardware Driver) and are a de-facto standard for
SDR research, teaching, and industrial prototyping.[^ettus]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="Three USRP tiers arranged by capability: B-series bus-powered devices, N-series networked radios, and X-series high-bandwidth platforms, all driven by the UHD driver." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="24" y="30" width="120" height="40" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="84" y="46">B200 / B210</text><text x="84" y="60" font-size="7">USB, bus-powered</text>
    <rect x="170" y="30" width="120" height="40" rx="4" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.2"/><text x="230" y="46">N-series</text><text x="230" y="60" font-size="7">Ethernet networked</text>
    <rect x="316" y="30" width="120" height="40" rx="4" fill="currentColor" fill-opacity="0.3" stroke="currentColor" stroke-width="1.2"/><text x="376" y="46">X-series</text><text x="376" y="60" font-size="7">wide BW, 10 GbE</text>
    <text x="230" y="98" font-size="8">all share the UHD driver and GNU Radio support</text>
  </g>
</svg>
<figcaption>The USRP range scales from bus-powered B-series to networked N- and high-bandwidth X-series, unified by UHD.</figcaption>
</figure>

## Overview

Where a dongle is a fixed receiver, a USRP is a configurable radio platform. Most models
pair a wideband RF front end with a large [FPGA](/reference/field-programmable-gate-array/)
that can run custom sample processing on the device itself, then move [IQ](/reference/iq-data/)
to the host over USB 3.0 (B-series) or Ethernet up to 10 GbE (N- and X-series). They
transmit and receive full-duplex, most support [MIMO](/reference/mimo/) with two or more
channels, and many accept an external 10 MHz reference and PPS so several units can be
locked together for [phased-array](/reference/phased-array-antenna/) and coherent work.
That capability comes at a price well above hobby SDRs — hundreds to many thousands of
dollars.

## What it is

The modern B- and E-series use Analog Devices **AD936x**
[zero-IF](/reference/zero-if/) transceivers; larger models use daughterboard front ends
that swap to cover different bands. Coverage is broadly DC to 6 GHz depending on the front
end, with instantaneous [bandwidth](/reference/bandwidth/) from tens of MHz on a B200 up to
around 160 MHz on high-end X-series boards, and [ADC](/reference/analog-to-digital-converter/)
resolution of 12 to 16 bits by model.

The software story is UHD: a single cross-platform driver and API that every USRP speaks,
tightly integrated with [GNU Radio](/reference/gnuradio/) (whose `gr-uhd` blocks are the
usual way to build flowgraphs) and reachable through [SoapySDR](/reference/soapysdr/) as
well. That common driver — not any one board — is the real ecosystem.

## Variants

- **[B200 / B210](/reference/usrp-b210/)** — bus-powered USB 3.0, single (B200) or dual
  (B210) channel, AD9361-class front end; the affordable research entry, and the model with
  widely-sold low-cost clones.
- **N-series (N200/N210, N3xx)** — Ethernet-connected, daughterboard or integrated front
  ends, suited to fixed installations and multi-unit synchronisation.
- **X-series (X300/X310, X4xx)** — the widest bandwidth and highest channel counts, 10 GbE
  connectivity, aimed at demanding research and defence applications.
- **E-series** — embedded/stand-alone units with an on-board processor for field use.

## Relevance to SDR

USRPs underpin a huge share of published SDR and wireless research: cellular
([LTE](/reference/lte/), [5G NR](/reference/5g-nr/)) testbeds, spectrum sensing,
direction finding, radar, and signals experiments where reproducibility and precise timing
matter. Their synchronisation and MIMO features enable coherent multi-channel systems no
single dongle can build.

For plain trunking reception a USRP is far more radio than the job needs — transmit, MIMO,
and FPGA processing are surplus to receive-only scanning, and the cost is high. GopherTrunk
ships native USB backends for RTL-SDR, HackRF, and Airspy, not a UHD backend, so a USRP
would be used through a SoapySDR bridge if at all. It is best understood as a
research-grade instrument whose scanning use is incidental to its role as a general RF
development platform.

## Sources

[^wiki]: [Universal Software Radio Peripheral](https://en.wikipedia.org/wiki/Universal_Software_Radio_Peripheral) — Wikipedia, on the USRP families, UHD, and their research use.
[^ettus]: [Ettus Research](https://www.ettus.com/) — Ettus Research (an NI company), official site with per-model specifications for the B-, N-, and X-series.
