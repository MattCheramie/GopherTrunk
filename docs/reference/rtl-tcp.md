---
slug: rtl-tcp
title: rtl_tcp
entry_type: technology
category: hardware
description: rtl_tcp is a small server that streams raw IQ samples from an RTL-SDR over a TCP network connection, allowing the dongle to be used remotely.
keywords: rtl_tcp, network SDR, remote RTL-SDR, IQ over TCP, Raspberry Pi SDR
aka: [rtl_tcp]
autolink: true
infobox:
  - { label: Type, value: Network IQ server }
  - { label: Streams, value: Raw IQ from RTL-SDR over TCP }
  - { label: Use, value: Dongle at antenna, decode elsewhere }
see_also: [rtl-sdr, soapysdr, iq-data, software-defined-radio]
related_lessons:
  - { title: "SDR hardware — RTL-SDR, HackRF, Airspy", url: /learn/sdr-hardware/ }
external:
  - { title: "rtl-sdr (osmocom)", url: https://osmocom.org/projects/rtl-sdr/wiki }
---

**rtl_tcp** is a small server that streams raw [IQ](/reference/iq-data/) samples from an
[RTL-SDR](/reference/rtl-sdr/) over a **TCP** connection, letting the dongle be used from
another machine on the network.

## Overview

A common setup runs rtl_tcp on a Raspberry Pi right at the [antenna](/reference/antenna/)
— keeping the coax run short — while the decoder runs on a more capable host elsewhere.

## Relevance to SDR

GopherTrunk can attach to an rtl_tcp (or [SoapyRemote](/reference/soapysdr/)) source,
treating a remote dongle like a local one.
