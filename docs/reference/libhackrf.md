---
slug: libhackrf
title: libhackrf
entry_type: technology
category: sdr-frameworks
description: "libhackrf is the host-side C library for Great Scott Gadgets' HackRF One, streaming 8-bit IQ for receive and transmit through an asynchronous callback API."
keywords: libhackrf, HackRF, HackRF One, Great Scott Gadgets, host library, hackrf_transfer, half-duplex transceiver, USB SDR, libusb
aka: [libhackrf, HackRF library]
autolink: true
infobox:
  - { label: Type, value: SDR host driver library }
  - { label: Idea, value: Host API for the HackRF One half-duplex transceiver }
  - { label: API, value: Async TX/RX callback over libusb }
see_also: [hackrf, callback-vs-stream-api, software-defined-radio, soapysdr, iq-data, sample-format]
cite_urls:
  - https://github.com/greatscottgadgets/hackrf
  - https://hackrf.readthedocs.io/
---

**libhackrf** is the host-side C library that drives the
[HackRF One](/reference/hackrf/), the wide-band half-duplex transceiver from Great Scott
Gadgets, streaming signed 8-bit [IQ](/reference/iq-data/) samples between the computer and
the radio over USB.[^gsg] It is the thin driver layer every HackRF application ultimately
calls, whether to receive, transmit, or reflash the device firmware.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="libhackrf mediates between application code and the HackRF One over USB, running an RX callback that receives IQ buffers or a TX callback that supplies them, since the device is half-duplex." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="hrfar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="20" y="55" width="110" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="75" y="74" text-anchor="middle" font-size="9" fill="currentColor">application</text>
  <rect x="175" y="55" width="110" height="30" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="230" y="74" text-anchor="middle" font-size="9" fill="currentColor">libhackrf</text>
  <rect x="330" y="55" width="110" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="385" y="74" text-anchor="middle" font-size="9" fill="currentColor">HackRF One</text>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <line x1="130" y1="65" x2="175" y2="65" marker-end="url(#hrfar)"/>
    <line x1="285" y1="65" x2="330" y2="65" marker-end="url(#hrfar)"/>
  </g>
  <text x="152" y="45" font-size="7" fill="currentColor" text-anchor="middle">RX or TX callback</text>
  <text x="307" y="45" font-size="7" fill="currentColor" text-anchor="middle">USB (libusb)</text>
  <text x="230" y="110" font-size="8" fill="currentColor" text-anchor="middle">half-duplex: receive or transmit, not both</text>
</svg>
<figcaption>libhackrf exposes the HackRF One through an asynchronous callback: one buffer of 8-bit IQ per transfer, in the receive direction or the transmit direction, since the hardware is half-duplex.</figcaption>
</figure>

## How it works

After `hackrf_init` and `hackrf_open`, an application configures the radio through a
uniform setter interface: `hackrf_set_freq` (1 MHz–6 GHz), `hackrf_set_sample_rate` (up to
20 MS/s), and a three-stage gain chain — the RF amp (a fixed ~11 dB LNA), the IF/LNA gain,
and the baseband VGA gain — plus the antenna-port bias tee. The MAX2837/MAX5864 analog
front end delivers signed 8-bit I and Q, a wider dynamic range than the RTL-SDR's unsigned
bytes but still only 8 bits, which is the HackRF's main [sample-format](/reference/sample-format/)
trade-off.

Streaming uses an **asynchronous callback**. For receive, `hackrf_start_rx` takes a
callback that libhackrf invokes with each filled `libusb` transfer buffer; for transmit,
`hackrf_start_tx` invokes a callback that the application *fills*. This inverts control in
the classic [callback-versus-stream](/reference/callback-vs-stream-api/) way: the library
drives the clock and the app must service each buffer promptly or drop samples. Crucially,
the HackRF is **half-duplex** — it can receive or transmit but not simultaneously — so an
application runs one direction at a time and calls `hackrf_stop_rx`/`hackrf_stop_tx` to
switch.

## In practice

The library ships with command-line tools built on top of it: `hackrf_transfer` (record
IQ to a file or replay it out the air), `hackrf_info`, `hackrf_sweep` (a fast spectrum
sweep across the whole 6 GHz range), and `hackrf_spiflash`/`hackrf_cpldjtag` for firmware
updates. Higher layers wrap it rather than reimplement it: [gr-osmosdr](/reference/gr-osmosdr/)
provides a GNU Radio source/sink, and [SoapySDR](/reference/soapysdr/) exposes a
`SoapyHackRF` plugin so cross-vendor applications reach the device through a common API.
Because it uses `libusb`, the same library works on Linux, macOS, and Windows.

## Relevance to SDR

libhackrf matters because the HackRF One is the accessible entry point to *transmit*-capable
SDR and to very wide instantaneous bandwidth (20 MHz), and this library is how software
reaches those capabilities. For anyone writing SDR software it is a compact example of a
bidirectional host driver: the same asynchronous buffer contract serves both the RX and TX
data paths, with the half-duplex constraint enforced by the API rather than assumed.

GopherTrunk is a receive-only trunking decoder and has no transmit path, so it uses only the
RX side of hardware like this. As a pure-Go program it does not link the C libhackrf directly;
it ingests IQ over network transports or from recorded [IQ files](/reference/iq-data/). The
producer/consumer shape is identical to libhackrf's RX callback, though — buffers of 8-bit IQ
arrive, GT converts them to complex baseband and runs its decode chain — so the wideband
capture a HackRF provides is exactly the kind of input GT's channelizer is built to split into
multiple decoded channels.

## Sources

[^gsg]: [HackRF repository and documentation](https://github.com/greatscottgadgets/hackrf) — Great Scott Gadgets, the source for libhackrf, the host API, and the hackrf_transfer/hackrf_sweep tools.
