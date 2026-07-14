---
slug: soapysdr
title: SoapySDR
entry_type: technology
category: sdr-frameworks
description: SoapySDR is a vendor-neutral hardware abstraction library that provides a common API to many software-defined radios, including networked access via SoapyRemote.
keywords: SoapySDR, SDR abstraction, SoapyRemote, hardware API, device support, Pothosware, driver plugin, gr-soapy
aka: [SoapySDR]
autolink: true
infobox:
  - { label: Type, value: SDR hardware abstraction layer }
  - { label: Provides, value: Common API across many SDRs }
  - { label: Remote, value: SoapyRemote (network IQ) }
see_also: [rtl-tcp, gnuradio, gqrx, software-defined-radio, rtl-sdr, airspy]
related_lessons:
  - { title: "SDR hardware — RTL-SDR, HackRF, Airspy", url: /learn/rf-sdr/sdr-hardware/ }
cite_urls:
  - https://github.com/pothosware/SoapySDR
  - https://github.com/pothosware/SoapySDR/wiki
---

**SoapySDR** is a vendor-neutral hardware-abstraction library that exposes a **common
API** across many [software-defined radios](/reference/software-defined-radio/), so
applications can support diverse devices without per-vendor code.[^proj] Instead of an app
linking directly to librtlsdr, libhackrf, libairspy and a dozen others, it links once to
SoapySDR and lets a **driver plugin** translate the calls for whatever hardware is plugged
in.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="Applications calling a common SoapySDR layer that drives several different SDR devices through per-device driver plugin modules." xmlns="http://www.w3.org/2000/svg">
  <rect x="150" y="14" width="160" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="230" y="31" text-anchor="middle" font-size="9" fill="currentColor">applications (GQRX, GNU Radio…)</text>
  <rect x="120" y="52" width="220" height="26" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="230" y="69" text-anchor="middle" font-size="9" fill="currentColor">SoapySDR (common API)</text>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="60" y="92" width="80" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="100" y="109">RTL-SDR</text>
    <rect x="190" y="92" width="80" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="230" y="109">HackRF</text>
    <rect x="320" y="92" width="80" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="360" y="109">Airspy</text>
  </g>
  <g stroke="currentColor" stroke-width="1.1"><line x1="230" y1="40" x2="230" y2="52"/><line x1="100" y1="78" x2="100" y2="92"/><line x1="230" y1="78" x2="230" y2="92"/><line x1="360" y1="78" x2="360" y2="92"/></g>
</svg>
<figcaption>SoapySDR is a vendor-neutral abstraction layer: one API that drives many different SDR devices through per-device plugins.</figcaption>
</figure>

## How it works

An application asks SoapySDR to *enumerate* devices, then opens one by a set of key/value
arguments (`driver=rtlsdr`, `driver=hackrf`, and so on). The library loads the matching
plugin module and hands back a device object with a uniform set of methods: set the tune
frequency, set the [sample rate](/reference/sample-rate/), set gains, and stream
[IQ](/reference/iq-data/) buffers. Every device — however different its native SDK — presents
the same calls, so the application code is identical whether the samples come from a
$25 dongle or a [USRP](/reference/usrp-ettus/). Capabilities the hardware lacks (a bias tee,
a second channel, transmit) are discoverable through the API rather than assumed, so an app
can adapt gracefully.

The companion **SoapyRemote** plugin makes a network-attached radio look local. A
`SoapySDRServer` process runs on the machine with the dongle; on the client, opening
`driver=remote` transparently forwards every API call and streams the IQ back over TCP/UDP.
From the application's point of view nothing changed — it is still talking to a normal
SoapySDR device — but the radio is on another host. This is the key advantage over a
single-device protocol like [rtl_tcp](/reference/rtl-tcp/): SoapyRemote works for *any*
SoapySDR-supported radio, not just RTL dongles.

## In practice

SoapySDR is the glue underneath a large slice of the open SDR ecosystem.
[GNU Radio](/reference/gnuradio/) ships a `gr-soapy` block, [GQRX](/reference/gqrx/),
CubicSDR, and SDRangel can all source from it, and Linux distributions package the drivers
so one `SoapySDRUtil --find` lists every attached radio.[^wiki] Because it is a thin abstraction
rather than a heavyweight framework, the overhead is negligible; the practical cost is one
more layer that must have a working driver plugin for your specific device.

## Relevance to SDR

For SDR users, SoapySDR's value is portability: write or configure once, run against many
radios, and place the radio anywhere on the network via SoapyRemote. GopherTrunk benefits
from the same idea — SoapySDR-style remoting (alongside [rtl_tcp](/reference/rtl-tcp/)) lets
it use radios that live on a separate host, so the dongle can sit at the antenna over a short
[coax](/reference/coaxial-cable/) run while decoding happens on a more capable machine.

## Sources

[^proj]: [SoapySDR](https://github.com/pothosware/SoapySDR) — the project repository, documenting the vendor-neutral SDR abstraction API and SoapyRemote.
[^wiki]: [SoapySDR wiki](https://github.com/pothosware/SoapySDR/wiki) — Pothosware, the driver list and SoapyRemote documentation describing per-device plugins and networked streaming.
