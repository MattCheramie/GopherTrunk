---
slug: libbladerf
title: libbladeRF
entry_type: technology
category: sdr-frameworks
description: "libbladeRF is Nuand's host-side C library for the bladeRF full-duplex transceiver, offering both synchronous and asynchronous IQ streaming and FPGA control."
keywords: libbladeRF, bladeRF, Nuand, host library, full-duplex transceiver, sync interface, async interface, FPGA, LMS6002D, AD9361
aka: [libbladeRF, bladeRF library]
autolink: true
infobox:
  - { label: Type, value: SDR host driver library }
  - { label: Idea, value: Host API for the Nuand bladeRF full-duplex transceiver }
  - { label: API, value: Sync (blocking) + async (callback) streaming }
see_also: [bladerf, software-defined-radio, callback-vs-stream-api, soapysdr, field-programmable-gate-array, iq-data]
cite_urls:
  - https://github.com/Nuand/bladeRF
  - https://www.nuand.com/libbladeRF-doc/
---

**libbladeRF** is the host-side C library that drives Nuand's
[bladeRF](/reference/bladerf/) family of full-duplex transceivers, giving applications
control of tuning, gain, the on-board [FPGA](/reference/field-programmable-gate-array/),
and both simultaneous transmit and receive [IQ](/reference/iq-data/) streams.[^nuand] It is
a notably richer host library than the receive-only dongle drivers, matching the bladeRF's
position as a lab-grade transceiver.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="libbladeRF connects an application to the bladeRF over USB 3.0, carrying two simultaneous full-duplex IQ streams, one receive and one transmit, plus control and FPGA configuration." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="bldar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="20" y="55" width="110" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="75" y="76" text-anchor="middle" font-size="9" fill="currentColor">application</text>
  <rect x="175" y="55" width="110" height="34" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="230" y="76" text-anchor="middle" font-size="9" fill="currentColor">libbladeRF</text>
  <rect x="330" y="45" width="110" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="385" y="64" text-anchor="middle" font-size="9" fill="currentColor">bladeRF + FPGA</text>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <line x1="130" y1="66" x2="175" y2="66" marker-end="url(#bldar)"/>
    <line x1="285" y1="60" x2="330" y2="60" marker-end="url(#bldar)"/>
  </g>
  <text x="230" y="118" font-size="8" fill="currentColor" text-anchor="middle">RX stream ⇅ TX stream (full-duplex) + control</text>
  <text x="307" y="42" font-size="7" fill="currentColor" text-anchor="middle">USB 3.0</text>
</svg>
<figcaption>libbladeRF carries two simultaneous IQ streams — receive and transmit — over USB 3.0, plus control and FPGA configuration, reflecting the bladeRF's full-duplex design.</figcaption>
</figure>

## How it works

An application opens a device with `bladerf_open`, optionally loads an FPGA bitstream with
`bladerf_load_fpga`, and configures per-channel parameters through `bladerf_set_frequency`,
`bladerf_set_sample_rate`, `bladerf_set_bandwidth`, and `bladerf_set_gain`. The RF transceiver
underneath is an LMS6002D (bladeRF x40/x115) or an AD9361 (bladeRF 2.0 micro), and libbladeRF
abstracts their register maps behind that uniform interface. Samples are 12-bit-native but
carried in 16-bit `int16` IQ pairs (an `SC16Q11` fixed-point format), and the library also
exposes the FPGA's expansion I/O, triggers, and the VCTCXO calibration.

libbladeRF deliberately offers **two streaming interfaces**, illustrating the classic
[stream-versus-callback](/reference/callback-vs-stream-api/) choice within one API. The
**synchronous** interface (`bladerf_sync_rx`/`bladerf_sync_tx`) is a blocking pull model:
the application calls to receive or transmit a block and the library manages buffering
internally — simple to write against. The **asynchronous** interface registers callbacks and
runs a buffer pool the application services as transfers complete — lower latency and
finer control, but more work. Because the hardware is **full-duplex** over USB 3.0, RX and TX
streams can run at once, so the async path commonly drives both directions concurrently.

## In practice

The library ships with the `bladeRF-cli` control/scripting tool and utilities for loading
FPGA images and flashing firmware. It is wrapped by [gr-osmosdr](/reference/gr-osmosdr/) and
by a `SoapyBladeRF` plugin for [SoapySDR](/reference/soapysdr/), so cross-vendor applications
can reach the device through those common layers. The FPGA is a first-class part of the
workflow: users can load Nuand's stock images or their own gateware to offload DSP (channel
filtering, decimation, custom triggers) before samples ever cross USB.

## Relevance to SDR

libbladeRF is the gateway to full-duplex, transmit-capable, FPGA-accelerated SDR at up to
61 MS/s (2.0 micro), which puts it in a different class from receive-only dongles and closer
to the [USRP](/reference/software-defined-radio/) end of the market. For a developer it is an
instructive host library precisely because it exposes both streaming philosophies and because
so much of the signal path can be pushed into the FPGA — a reminder that where DSP runs
(device, host, or somewhere between) is a design decision, not a given.

GopherTrunk is a receive-only, pure-Go decoder with no FPGA offload and no transmit path, so
it uses only a fraction of what a transceiver like the bladeRF offers. It does not link the C
libbladeRF; it takes IQ over network transports or from recorded [IQ files](/reference/iq-data/)
and does all of its channelization and demodulation in Go on the host CPU. The bladeRF's wide
capture bandwidth is nonetheless well matched to GT's multi-channel channelizer, which is built
to split a wideband stream into many simultaneously decoded trunking channels.

## Sources

[^nuand]: [bladeRF repository and libbladeRF documentation](https://github.com/Nuand/bladeRF) — Nuand, the source for libbladeRF, its synchronous and asynchronous streaming interfaces, and FPGA/host control.
