---
slug: usrp-b210
title: USRP B210 (and AD9361 clones)
entry_type: hardware
category: sdr-devices
description: "The Ettus USRP B210 is a wideband AD9361-based SDR transceiver covering 70 MHz–6 GHz with up to 56 MHz of instantaneous bandwidth and 2x2 MIMO; low-cost functional clones of the same design are widely sold and behave nearly identically."
keywords: USRP B210, Ettus B210, AD9361 SDR, B210 clone, wideband transceiver, 70 MHz 6 GHz, 56 MHz bandwidth, 2x2 MIMO, UHD, best value wideband SDR, TX RX SDR
aka: [USRP B210, B210, AD9361 SDR]
autolink: true
infobox:
  - { label: Type, value: Wideband SDR transceiver (RX + TX) }
  - { label: Vendor/Chip, value: "Ettus (NI); Analog Devices AD9361 + Xilinx Spartan-6" }
  - { label: Range, value: "70 MHz – 6 GHz" }
  - { label: Bandwidth, value: "up to 56 MHz (30.72 MS/s stable over USB 3.0)" }
  - { label: ADC, value: "12-bit" }
  - { label: MIMO, value: "2x2 (two coherent RX + TX)" }
  - { label: TX, value: "Yes (full-duplex)" }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://opensourcesdrlab.com/products/b210-ad9361\" rel=\"nofollow noopener\">Buy a B210 clone &rarr;</a>" }
see_also: [usrp-ettus, limesdr, plutosdr, bladerf, hackrf, dynamic-range, software-defined-radio]
related_lessons:
  - { title: "SDR hardware — RTL-SDR, HackRF, Airspy", url: /learn/rf-sdr/sdr-hardware/ }
cite_urls:
  - https://www.ettus.com/all-products/ub210-kit/
  - https://en.wikipedia.org/wiki/Universal_Software_Radio_Peripheral
---

The **USRP B210** is a wideband [SDR](/reference/software-defined-radio/) transceiver from
[Ettus Research](/reference/usrp-ettus/) built around the Analog Devices **AD9361** RFIC.
It covers **70 MHz – 6 GHz** continuous, with up to **56 MHz** of instantaneous
[bandwidth](/reference/bandwidth/), two coherent receive and two transmit channels (2x2
MIMO), and full-duplex TX/RX over USB 3.0.[^ettus] Because the AD9361 does most of the
work, functionally-identical **low-cost clones** of the same reference design are widely
sold — and for receive-only monitoring they behave essentially the same as the original.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A frequency coverage bar for USRP B210 (70 MHz–6 GHz) on an axis from about 0 to 6 gigahertz." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="430" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="30" y="86">0</text><text x="163" y="86">2 GHz</text><text x="296" y="86">4 GHz</text><text x="430" y="86">6 GHz</text></g>
  <rect x="35" y="40" width="391" height="20" rx="3" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.3"/>
  <text x="230" y="28" text-anchor="middle" font-size="10" fill="currentColor">USRP B210 (70 MHz–6 GHz) coverage</text>
</svg>
<figcaption>The B210's AD9361 front end spans 70 MHz–6 GHz with wide instantaneous bandwidth.</figcaption>
</figure>

## Overview

Where an [Airspy](/reference/airspy/) or [HydraSDR](/reference/hydrasdr/) is a tuned
receive dongle, the B210 is a full **transceiver**: two RX and two TX chains, wide
capture, and the [UHD](/reference/usrp-ettus/) driver stack. For a scanning workload its
draw is the **wide instantaneous bandwidth** — comfortably enough to hold an entire
trunked system's control and voice channels inside one capture — and the coherent second
receiver, useful for direction finding or diversity. The trade-off versus a purpose-built
receive dongle is cost, power, and a heavier software stack.

## How it works

The AD9361 is a complete 2x2 zero-IF transceiver on a chip: programmable synthesizers,
mixers, baseband filters, and 12-bit converters, all tuned by the host. A Xilinx Spartan-6
FPGA packetizes the streams to the host over USB 3.0. The sustained rate over USB tops out
around **30.72 MS/s** (≈56 MHz with the AD9361's filters), well beyond what a single
trunked system needs. Being zero-IF, it carries the usual DC-offset and I/Q-imbalance
behaviour of that architecture — GopherTrunk's DDC keeps the wanted channel off the DC
spike, and a [broadcast-FM notch filter](/reference/fm-broadcast-filter/) still pays off
in a strong-signal metro.

## Variants and clones

- **Ettus USRP B210** — the original (NI/Ettus), the reference against which the clones are
  measured; the **B200** is the single-channel sibling.
- **AD9361 "B210" clones** — several vendors sell boards built to the B210 reference design
  (e.g. via [opensourcesdrlab](https://opensourcesdrlab.com/products/b210-ad9361)). In
  practice they are a strong value: for receive monitoring the difference from an original
  is negligible, at a fraction of the price. Firmware/UHD compatibility varies by vendor —
  confirm the seller supports the current UHD before buying.

Against a [LimeSDR](/reference/limesdr/) or [BladeRF](/reference/bladerf/) the B210 is the
most mature of the AD9361/wideband-transceiver class in software support (UHD is
ubiquitous); against an [ADALM-Pluto](/reference/plutosdr/) it adds a second coherent
channel and much wider bandwidth.

## Relevance to SDR

GopherTrunk drives USRP hardware through UHD (see [USRP (Ettus)](/reference/usrp-ettus/)).
The B210's wide capture suits following a whole trunked system — or several control
channels — on one device via the `role: wideband` path, and its receive quality is well
above a dongle's. As always it is a receiver for monitoring here: it decodes clear and
scrambled traffic, never keyed encryption, and its transmit capability is irrelevant to
scanning.

## Where to buy

The original is sold by **[Ettus/NI](https://www.ettus.com/all-products/ub210-kit/)**
(around $1,500). Functionally-equivalent AD9361 clones are far cheaper and widely
available — e.g.
**[opensourcesdrlab B210 (AD9361)](https://opensourcesdrlab.com/products/b210-ad9361)** —
and on eBay/AliExpress. Confirm current UHD support with the seller.

## Sources

[^ettus]: [Ettus USRP B210](https://www.ettus.com/all-products/ub210-kit/) — Ettus Research (NI), on the B210's AD9361 front end, 70 MHz–6 GHz range, up to 56 MHz bandwidth, 2x2 MIMO and USB 3.0 streaming.
