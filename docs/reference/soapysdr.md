---
slug: soapysdr
title: SoapySDR
entry_type: technology
category: hardware
description: SoapySDR is a vendor-neutral hardware abstraction library that provides a common API to many software-defined radios, including networked access via SoapyRemote.
keywords: SoapySDR, SDR abstraction, SoapyRemote, hardware API, device support
aka: [SoapySDR]
autolink: true
infobox:
  - { label: Type, value: SDR hardware abstraction layer }
  - { label: Provides, value: Common API across many SDRs }
  - { label: Remote, value: SoapyRemote (network IQ) }
see_also: [rtl-tcp, software-defined-radio, rtl-sdr, airspy]
related_lessons:
  - { title: "SDR hardware — RTL-SDR, HackRF, Airspy", url: /learn/rf-sdr/sdr-hardware/ }
external:
  - { title: "SoapySDR (project)", url: https://github.com/pothosware/SoapySDR }
---

**SoapySDR** is a vendor-neutral hardware-abstraction library that exposes a **common
API** across many [software-defined radios](/reference/software-defined-radio/), so
applications can support diverse devices without per-vendor code.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="Applications calling a common SoapySDR layer that drives several different SDR devices through plugin modules." xmlns="http://www.w3.org/2000/svg">
  <rect x="150" y="14" width="160" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="230" y="31" text-anchor="middle" font-size="9" fill="currentColor">applications</text>
  <rect x="120" y="52" width="220" height="26" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="230" y="69" text-anchor="middle" font-size="9" fill="currentColor">SoapySDR (common API)</text>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="60" y="92" width="80" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="100" y="109">RTL-SDR</text>
    <rect x="190" y="92" width="80" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="230" y="109">HackRF</text>
    <rect x="320" y="92" width="80" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="360" y="109">Airspy</text>
  </g>
  <g stroke="currentColor" stroke-width="1.1"><line x1="230" y1="40" x2="230" y2="52"/><line x1="100" y1="78" x2="100" y2="92"/><line x1="230" y1="78" x2="230" y2="92"/><line x1="360" y1="78" x2="360" y2="92"/></g>
</svg>
<figcaption>SoapySDR is a vendor-neutral abstraction layer: one API that drives many different SDR devices.</figcaption>
</figure>

## Overview

Its **SoapyRemote** companion streams [IQ](/reference/iq-data/) over a network, letting a
radio on one machine serve an application on another — useful for placing the dongle at
the antenna.

## Relevance to SDR

SoapySDR-style remoting (alongside [rtl_tcp](/reference/rtl-tcp/)) lets GopherTrunk use
radios that live on a separate host.
