---
slug: libiio
title: libiio
entry_type: technology
category: sdr-frameworks
description: "libiio is Analog Devices' cross-platform C library for the Linux Industrial I/O framework, used to stream IQ and configure devices such as the PlutoSDR over USB, network, or local backends."
keywords: libiio, Industrial I/O, IIO, Analog Devices, PlutoSDR, AD9361, iio_context, buffer streaming, IIO backend, network backend, gr-iio
aka: [libiio, IIO library]
autolink: true
infobox:
  - { label: Type, value: IIO device access library (C) }
  - { label: Vendor, value: Analog Devices }
  - { label: Common use, value: PlutoSDR (AD9361) streaming }
see_also: [plutosdr, soapysdr, gnuradio, iq-data, usrp-ettus, sample-rate]
cite_urls:
  - https://github.com/analogdevicesinc/libiio
  - https://wiki.analog.com/resources/tools-software/linux-software/libiio
---

**libiio** is a cross-platform C library from Analog Devices that provides a uniform way to
access devices built on the Linux **Industrial I/O (IIO)** framework — the kernel subsystem for
ADCs, DACs, and RF transceivers.[^lib] In the software-defined-radio world its best-known job is
driving the **[PlutoSDR](/reference/plutosdr/)** and other radios built on Analog Devices'
AD9361/AD936x transceiver: libiio is how a host program tunes them, sets gains and
[sample rate](/reference/sample-rate/), and streams [IQ](/reference/iq-data/) buffers in and out.
It abstracts *where* the device lives, so the same code works whether the radio is local, on USB,
or across a network.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A host application using libiio to reach an IIO device through interchangeable backends — local, USB, or network — to configure channels and stream IQ buffers." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="iioar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="12" y="52" width="92" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="58" y="66" text-anchor="middle" font-size="8" fill="currentColor">host app</text>
  <text x="58" y="77" text-anchor="middle" font-size="7" fill="currentColor">(iio_context)</text>
  <rect x="132" y="52" width="80" height="34" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/>
  <text x="172" y="72" text-anchor="middle" font-size="8" fill="currentColor">libiio</text>
  <g font-size="7.5" fill="currentColor" text-anchor="middle">
    <rect x="244" y="16" width="70" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="279" y="31">local</text>
    <rect x="244" y="57" width="70" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="279" y="72">USB</text>
    <rect x="244" y="98" width="70" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="279" y="113">network</text>
  </g>
  <rect x="360" y="52" width="88" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="404" y="66" text-anchor="middle" font-size="8" fill="currentColor">IIO device</text>
  <text x="404" y="77" text-anchor="middle" font-size="7" fill="currentColor">(AD9361)</text>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <line x1="104" y1="69" x2="130" y2="69" marker-end="url(#iioar)"/>
    <line x1="212" y1="66" x2="243" y2="30" marker-end="url(#iioar)"/>
    <line x1="212" y1="69" x2="243" y2="69" marker-end="url(#iioar)"/>
    <line x1="212" y1="72" x2="243" y2="108" marker-end="url(#iioar)"/>
    <line x1="314" y1="69" x2="359" y2="69" marker-end="url(#iioar)"/>
  </g>
  <text x="230" y="134" text-anchor="middle" font-size="9" fill="currentColor">same API, interchangeable backends — the radio can be local or remote</text>
</svg>
<figcaption>libiio reaches an IIO device through interchangeable backends (local, USB, network) so a PlutoSDR looks the same to the application whether it is on the local bus or across the LAN.</figcaption>
</figure>

## How it works

libiio models hardware as a tree. The root is an **`iio_context`**, opened against a backend and
a URI (for example `usb:`, `ip:192.168.2.1`, or a local context on the device itself). A context
contains **devices**, each device has **channels** (an AD9361 has receive and transmit I and Q
channels plus control channels), and each channel exposes **attributes** — tunable settings such
as center frequency, RF bandwidth, sampling frequency, and gain, read and written as name/value
pairs. This self-describing structure means an application can enumerate exactly what a device
offers rather than hard-coding it.

Sample transfer uses **buffers**. The program marks the channels it wants, creates a buffer of a
chosen size, then repeatedly refills it (receive) or pushes it (transmit); libiio moves the raw
sample blocks across the active backend and provides helpers to step through the interleaved I/Q
data with the correct format and scaling. Two properties make it well suited to SDR:

- **Backend transparency** — the identical API works locally and remotely. Because the network
  backend forwards contexts over IP, a PlutoSDR plugged into one machine can be streamed by a
  program on another with only a URI change, similar in spirit to
  [SoapyRemote](/reference/soapysdr/).
- **Device-agnostic control** — attributes are generic, so the same code drives any AD936x-based
  board, and non-radio IIO sensors, through one interface.

Language bindings (C, C++, Python, C#) and command-line tools (`iio_info`, `iio_readdev`) sit on
top of the same core.

## Relevance to SDR

libiio is the native access path for the Analog Devices SDR family — most visibly the
[PlutoSDR](/reference/plutosdr/), a popular low-cost learning and experimentation radio, and the
larger FMComms and ADALM boards built on the same transceivers. It plugs into the wider ecosystem
through wrappers: [GNU Radio](/reference/gnuradio/) reaches these radios via the `gr-iio` blocks,
and [SoapySDR](/reference/soapysdr/) offers a Pluto/libiio driver so the device also appears in
any Soapy-based application. Its network backend is especially handy for the Pluto, whose common
form factor is a USB-attached or Ethernet-gadget device that developers frequently drive from a
separate host.

**GopherTrunk** does not use libiio and does not currently target the PlutoSDR; its front-end
support centers on RTL-SDR, Airspy, and network IQ sources reached through GopherTrunk's own Go
device layer, keeping it a single dependency-free static binary. libiio is relevant here as the
reference model of a well-structured device-access library — a self-describing tree of
devices/channels/attributes, buffer-based streaming, and backend-transparent local-or-remote
access — the same set of concerns any SDR front-end abstraction, GopherTrunk's included, has to
address.

## Sources

[^lib]: [libiio](https://github.com/analogdevicesinc/libiio) — Analog Devices, documenting the IIO context/device/channel/attribute model, buffer-based sample streaming, the local/USB/network backends, and language bindings.
