---
slug: libairspy
title: libairspy
entry_type: technology
category: sdr-frameworks
description: "libairspy is the host-side C library for Airspy R2 and Mini receivers, streaming 12-bit-derived IQ or real samples through an asynchronous callback API."
keywords: libairspy, Airspy, Airspy R2, Airspy Mini, host library, airspy_rx, libusb, packed samples, IQ streaming
aka: [libairspy, Airspy library]
autolink: true
infobox:
  - { label: Type, value: SDR host driver library }
  - { label: Idea, value: Host API for Airspy R2/Mini receivers }
  - { label: API, value: Async RX callback over libusb }
see_also: [airspy, callback-vs-stream-api, software-defined-radio, soapysdr, sample-format, iq-data]
cite_urls:
  - https://github.com/airspy/airspyone_host
  - https://airspy.com/
---

**libairspy** is the host-side C library that drives the
[Airspy](/reference/airspy/) R2 and Airspy Mini receivers, streaming their high-dynamic-range
[IQ](/reference/iq-data/) over USB.[^air] It is the driver layer beneath every Airspy
application, and its design reflects the Airspy's positioning as a step up in ADC quality
from the RTL-SDR while keeping the same simple callback-based host interface.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 145" role="img" aria-label="libairspy receives packed 12-bit samples over USB from the Airspy, optionally unpacks and converts them to float or 16-bit IQ, and delivers them to the application through a registered RX callback." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="airar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="20" y="55" width="90" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="65" y="74" text-anchor="middle" font-size="9" fill="currentColor">Airspy</text>
  <rect x="150" y="55" width="150" height="30" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="225" y="70" text-anchor="middle" font-size="9" fill="currentColor">libairspy</text>
  <text x="225" y="80" text-anchor="middle" font-size="7" fill="currentColor">unpack + convert</text>
  <rect x="340" y="55" width="100" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="390" y="74" text-anchor="middle" font-size="9" fill="currentColor">application</text>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <line x1="110" y1="70" x2="150" y2="70" marker-end="url(#airar)"/>
    <line x1="300" y1="70" x2="340" y2="70" marker-end="url(#airar)"/>
  </g>
  <text x="130" y="50" font-size="7" fill="currentColor" text-anchor="middle">packed 12-bit</text>
  <text x="320" y="50" font-size="7" fill="currentColor" text-anchor="middle">float / int16 IQ</text>
  <text x="225" y="120" font-size="8" fill="currentColor" text-anchor="middle">RX callback delivers one buffer per USB transfer</text>
</svg>
<figcaption>libairspy unpacks the Airspy's packed 12-bit samples on the host and converts them to the requested format, delivering IQ (or real) buffers to the application via an RX callback.</figcaption>
</figure>

## How it works

An application calls `airspy_open`, then configures tuning and rate through
`airspy_set_freq`, `airspy_set_samplerate` (the R2 offers 10 and 2.5 MS/s; the Mini
10/6/3 MS/s), and a gain interface that can be driven either as linearized "sensitivity"
values or as the individual LNA, mixer, and VGA stages. The Airspy's real ADC samples at up
to 20 MS/s and the front end mixes to a low-IF; the R2 then performs an internal
[quadrature](/reference/quadrature-demodulation/)/decimation step, so the host can request
either **real** samples or **complex IQ**. To save USB bandwidth the device sends samples
**packed** at 12 bits, and libairspy unpacks them and, if asked, converts to `float32` or
`int16` — meaningful [sample-format](/reference/sample-format/) work done on the host rather
than the wire.

Delivery uses an **asynchronous callback**: `airspy_start_rx` registers a function that
libairspy invokes with an `airspy_transfer` struct — a pointer, a sample count, and the
format — once per filled `libusb` buffer. As with its sibling libraries this is a
control-inverting [callback API](/reference/callback-vs-stream-api/): the library owns the
streaming loop and pushes data, and the callback must consume each buffer quickly enough to
avoid dropped samples. `airspy_stop_rx` and `airspy_close` tear the session down.

## In practice

The library ships with tools such as `airspy_rx` (record IQ or real samples to a file),
`airspy_info`, and calibration/SPI-flash utilities. It is wrapped by
[gr-osmosdr](/reference/gr-osmosdr/) for GNU Radio and by a `SoapyAirspy` plugin for
[SoapySDR](/reference/soapysdr/), so most applications use the Airspy through those common
layers rather than calling libairspy directly. A separate `libairspyhf` library serves the
Airspy HF+ family — different hardware, different API — so the two should not be confused.

## Relevance to SDR

libairspy is how software reaches the Airspy's main selling point: a genuinely higher-quality
front end (12-bit ADC, better dynamic range and out-of-band rejection than an 8-bit dongle)
at 10 MS/s of usable bandwidth. For an SDR developer it is also a clean example of a host
library that does real DSP-adjacent work — bit-unpacking and format conversion — on the CPU
before handing samples up, which shapes how much processing the application still has to do.

GopherTrunk works with Airspy captures, and the Airspy is directly relevant to GT's
[documented DSP notes](/reference/software-defined-radio/): several trunking captures GT has
analyzed came from an Airspy running at its native 10 MS/s. As a pure-Go application GT does
not link the C libairspy; it ingests IQ over network transports or from recorded files, then
runs a rate-invariant decode chain that normalizes any capture rate to the per-protocol
channel rate. The buffers it receives are exactly what an `airspy_start_rx` callback would
deliver.

## Sources

[^air]: [airspyone_host repository](https://github.com/airspy/airspyone_host) — Airspy, the source for libairspy, the RX callback API, packed-sample handling, and the airspy_rx tool.
