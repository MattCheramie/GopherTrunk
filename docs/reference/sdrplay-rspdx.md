---
slug: sdrplay-rspdx
title: SDRplay RSPdx
entry_type: hardware
category: sdr-devices
description: SDRplay RSPdx (and the current RSPdx-R2 revision) is a 14-bit, 1 kHz–2 GHz receive-only SDR with extensive preselection, three antenna ports, and an HDR mode for high-dynamic-range LF/MW/HF reception.
keywords: SDRplay RSPdx, RSPdx-R2, RSPdx R2, HDR mode, high dynamic range SDR, 14-bit receiver, three antenna ports, wideband receiver, SoapySDR
aka: [RSPdx, RSPdx-R2, SDRplay RSPdx, SDRplay RSPdx-R2]
autolink: true
infobox:
  - { label: Type, value: Receive-only SDR }
  - { label: Vendor/Chip, value: "SDRplay, Mirics chipset" }
  - { label: ADC, value: 14-bit (HDR mode) }
  - { label: Range, value: 1 kHz – 2 GHz }
  - { label: Bandwidth, value: up to ~10 MHz }
  - { label: TX, value: No }
  - { label: Current model, value: "RSPdx-R2 (2023 refresh)" }
  - { label: Typical price, value: ~US$250 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B0821NMGVP?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [sdrplay-rsp1a, sdrplay-rspduo, software-defined-radio, soapysdr, rf-filter, dynamic-range]
affiliate: true
cite_urls:
  - https://en.wikipedia.org/wiki/Software-defined_radio
  - https://www.sdrplay.com/rspdx/
---

**SDRplay RSPdx** is a top-of-line receive-only
[software-defined radio](/reference/software-defined-radio/) that covers **1 kHz to 2 GHz**
with a **14-bit** [ADC](/reference/analog-to-digital-converter/), three selectable antenna
ports, and an extensive bank of front-end filters.[^wiki] It extends the
[RSP1A](/reference/sdrplay-rsp1a/) with more preselection and a dedicated **HDR
(high-dynamic-range) mode** that sharpens performance below about 2 MHz.[^sdrplay]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A frequency coverage bar for the SDRplay RSPdx from 1 kilohertz to 2 gigahertz, with a highlighted high-dynamic-range mode region below about 2 megahertz at the low end." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="80" x2="430" y2="80" stroke="currentColor" stroke-opacity="0.4"/>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="30" y="96">0</text><text x="163" y="96">2 GHz</text><text x="296" y="96">4 GHz</text><text x="430" y="96">6 GHz</text></g>
  <rect x="30" y="50" width="133" height="20" rx="3" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.3"/>
  <rect x="30" y="50" width="10" height="20" rx="2" fill="currentColor" fill-opacity="0.6"/>
  <text x="230" y="38" text-anchor="middle" font-size="10" fill="currentColor">RSPdx (1 kHz–2 GHz); dark band = HDR mode (LF/MW/HF)</text>
</svg>
<figcaption>The RSPdx spans VLF to UHF; below ~2 MHz its HDR mode trades span for cleaner dynamic range.</figcaption>
</figure>

## Overview

The RSPdx is aimed at listeners who need the best possible front end in a single-tuner RSP:
DXers chasing weak signals under strong broadcast neighbours, and monitors working crowded
bands where [intermodulation](/reference/intermodulation/) and overload are the real enemy.
It keeps the RSP family's continuous 1 kHz–2 GHz tuning and up to ~10 MHz of visible
[bandwidth](/reference/bandwidth/), but wraps the Mirics tuner in a much larger set of
switchable [preselection](/reference/rf-filter/) filters and adds three antenna inputs
(two SMA plus a dedicated HF/BNC path) so different antennas can serve different bands
without a patch panel.

## What it is

Like the rest of the line, the RSPdx is a Mirics-based receiver that streams raw
[IQ](/reference/iq-data/) over USB while the host does all
[demodulation](/reference/demodulation/) and decoding. Its defining feature is **HDR
mode**, available on the lower bands (LF, MW, and the HF broadcast/amateur segments). In
HDR mode the receiver reconfigures its signal path to widen usable
[dynamic range](/reference/dynamic-range/) and improve
[spurious-free dynamic range](/reference/spurious-free-dynamic-range/) at the cost of
instantaneous span — a deliberate trade that pays off when a nearby MW blowtorch would
otherwise swamp a distant HF signal. Above those bands the RSPdx behaves as a wideband
receiver with the fuller filter bank switched in as you tune.

## Variants

The **RSPdx-R2** is the current-production revision (2023) and the one you will buy new
today — it supersedes the original RSPdx with a refreshed front end and an upgraded
processor for improved performance, while keeping the same three antenna ports, 14-bit
converter, HDR mode, interface, and software. Treat "RSPdx" and "RSPdx-R2" as the same
device for GopherTrunk purposes; prefer the R2 when buying. The line contrasts with:

- **[RSP1A](/reference/sdrplay-rsp1a/)** — the entry model: one antenna port, fewer filters,
  no HDR mode.
- **[RSPduo](/reference/sdrplay-rspduo/)** — dual-tuner, trading the RSPdx's rich single
  front end for two independent 2 MHz receivers usable in diversity.

All models share SDRplay's proprietary API/service and its
[SoapySDR](/reference/soapysdr/) module, so applications see them through a common
interface.

## Relevance to SDR

The RSPdx is most valued below UHF, where its filtering and HDR mode let it pull weak
signals out from under strong ones — a scenario RTL-SDR-class hardware handles poorly. For
VHF/UHF trunking it works like any wideband RSP: capture a system's control and voice
channels in one ~10 MHz window and channelise on the host.

As with every RSP, GopherTrunk has no native RSPdx backend — the device requires SDRplay's
closed API/service, so any use goes through a SoapySDR bridge rather than GopherTrunk's
direct USB drivers for RTL-SDR, HackRF, and Airspy. The RF hardware is more than capable;
the practical question is driver integration, so verify current project support before
relying on it.

## Sources

[^wiki]: [Software-defined radio](https://en.wikipedia.org/wiki/Software-defined_radio) — Wikipedia, background on receiver-class SDRs including the SDRplay RSP line.
[^sdrplay]: [RSPdx](https://www.sdrplay.com/rspdx/) — SDRplay, official product page covering the 14-bit converter, filter banks, three antenna ports, and HDR mode.
