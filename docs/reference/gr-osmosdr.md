---
slug: gr-osmosdr
title: gr-osmosdr
entry_type: technology
category: sdr-frameworks
description: "gr-osmosdr is a GNU Radio source and sink block that gives one common interface to many SDR front ends — RTL-SDR, HackRF, Airspy, bladeRF, USRP, and more — via a device-string argument."
keywords: gr-osmosdr, gnuradio osmosdr source, osmocom, SDR source block, RTL-SDR HackRF Airspy bladeRF, device argument string, gr-soapy, Osmocom SDR
aka: [gr-osmosdr, osmosdr source]
autolink: true
infobox:
  - { label: Type, value: GNU Radio source/sink block }
  - { label: Idea, value: One block, many SDR front ends }
  - { label: Origin, value: Osmocom project }
see_also: [gnuradio, soapysdr, rtl-sdr, hackrf, airspy, bladerf]
cite_urls:
  - https://osmocom.org/projects/gr-osmosdr/wiki
  - https://github.com/osmocom/gr-osmosdr
---

**gr-osmosdr** is a [GNU Radio](/reference/gnuradio/) block — an **Osmocom Source** and
matching sink — that presents a single, uniform interface to a wide range of
[software-defined-radio](/reference/software-defined-radio/) front ends.[^osmo] Instead of
using a different, device-specific source block for each radio, a flowgraph drops in one
`osmosdr source` and selects the hardware with a short **device argument string** such as
`rtl=0`, `hackrf=0`, or `airspy=0`. From the [GNU Radio Companion](/reference/gnuradio-companion/)
canvas it makes almost any supported dongle look the same, which is why it became the standard
starting block for practical SDR flowgraphs.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A single gr-osmosdr source block in a GNU Radio flowgraph selecting one of several supported radios by a device-argument string, then feeding IQ samples into the rest of the flowgraph." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="osar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="7.5" fill="currentColor" text-anchor="middle">
    <rect x="12" y="16" width="72" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="48" y="30">RTL-SDR</text>
    <rect x="12" y="46" width="72" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="48" y="60">HackRF</text>
    <rect x="12" y="76" width="72" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="48" y="90">Airspy</text>
    <rect x="12" y="106" width="72" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="48" y="120">bladeRF</text>
  </g>
  <rect x="150" y="52" width="120" height="40" rx="6" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/>
  <text x="210" y="70" text-anchor="middle" font-size="8" fill="currentColor">osmosdr source</text>
  <text x="210" y="82" text-anchor="middle" font-size="7" fill="currentColor">args: "rtl=0"</text>
  <rect x="330" y="52" width="118" height="40" rx="6" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="389" y="76" text-anchor="middle" font-size="8" fill="currentColor">rest of flowgraph</text>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <line x1="84" y1="27" x2="149" y2="60" marker-end="url(#osar)"/>
    <line x1="84" y1="57" x2="149" y2="66" marker-end="url(#osar)"/>
    <line x1="84" y1="87" x2="149" y2="78" marker-end="url(#osar)"/>
    <line x1="84" y1="117" x2="149" y2="84" marker-end="url(#osar)"/>
    <line x1="270" y1="72" x2="328" y2="72" marker-end="url(#osar)"/>
  </g>
  <text x="240" y="132" text-anchor="middle" font-size="9" fill="currentColor">one block, chosen by device string, streams IQ into the graph</text>
</svg>
<figcaption>gr-osmosdr abstracts many front ends behind one GNU Radio block: a device-argument string picks the radio, and the same IQ stream flows into the rest of the flowgraph.</figcaption>
</figure>

## How it works

gr-osmosdr is a thin dispatch layer over a set of per-device back ends. When a flowgraph opens
the source, the block parses the device-argument string, loads the matching back end, and
configures the hardware through it. The back ends wrap the native libraries —
`librtlsdr` for [RTL-SDR](/reference/rtl-sdr/), `libhackrf` for [HackRF](/reference/hackrf/),
`libairspy` for [Airspy](/reference/airspy/), `libbladeRF` for [bladeRF](/reference/bladerf/),
UHD for USRPs, and others — plus a [SoapySDR](/reference/soapysdr/) back end that in turn reaches
any Soapy-supported device. Once open, the block exposes a common set of GNU Radio parameters:

- **Tuning and rate** — center frequency, [sample rate](/reference/sample-rate/), and frequency
  correction, applied uniformly regardless of the underlying radio.
- **Gain** — a normalized overall gain plus the named gain stages (LNA, mixer, VGA/IF) a device
  happens to expose.
- **Optional features** — bias-tee power, antenna selection, and bandwidth, surfaced where the
  hardware supports them.

The block then streams complex [IQ](/reference/iq-data/) samples into the flowgraph like any
other source. Multiple devices can be opened at once, and a matching sink provides the transmit
direction for radios that transmit. In effect gr-osmosdr does for [GNU Radio](/reference/gnuradio/)
flowgraphs what [SoapySDR](/reference/soapysdr/) does at the library level: collapse many
device-specific SDKs into one interface — and indeed newer setups often let gr-osmosdr reach
hardware *through* SoapySDR.

## Relevance to SDR

For years gr-osmosdr was the practical answer to "how do I get samples from my dongle into a GNU
Radio flowgraph," and it remains extremely common in tutorials, published flowgraphs, and shipped
tools. Because a flowgraph written against it is hardware-agnostic, the same `.grc` file runs on
whatever radio a user owns simply by changing the device string — a major reason GNU Radio
examples are so portable across the RTL-SDR, HackRF, and Airspy that dominate the hobby. It sits
alongside `gr-soapy` as the two standard ways a [GNU Radio Companion](/reference/gnuradio-companion/)
graph talks to real hardware.

**GopherTrunk** does not use gr-osmosdr or GNU Radio at all. GopherTrunk is a pure-Go trunking
scanner that talks to its supported front ends (RTL-SDR, Airspy, and network IQ sources) through
its own Go device layer, shipping as a single static binary with no GNU Radio runtime. The design
intent is the same as gr-osmosdr's — one uniform interface over several dongles so the decode path
does not care which radio is attached — but GopherTrunk implements that abstraction itself in Go
rather than borrowing GNU Radio's block.

## Sources

[^osmo]: [gr-osmosdr](https://osmocom.org/projects/gr-osmosdr/wiki) — Osmocom project wiki, documenting the common source/sink block, device-argument strings, the per-device back ends (RTL-SDR, HackRF, Airspy, bladeRF, UHD, SoapySDR), and the exposed tuning/gain parameters.
