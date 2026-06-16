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
  - { title: "SDR hardware — RTL-SDR, HackRF, Airspy", url: /learn/sdr-hardware/ }
external:
  - { title: "SoapySDR (project)", url: https://github.com/pothosware/SoapySDR }
---

**SoapySDR** is a vendor-neutral hardware-abstraction library that exposes a **common
API** across many [software-defined radios](/reference/software-defined-radio/), so
applications can support diverse devices without per-vendor code.

## Overview

Its **SoapyRemote** companion streams [IQ](/reference/iq-data/) over a network, letting a
radio on one machine serve an application on another — useful for placing the dongle at
the antenna.

## Relevance to SDR

SoapySDR-style remoting (alongside [rtl_tcp](/reference/rtl-tcp/)) lets GopherTrunk use
radios that live on a separate host.
