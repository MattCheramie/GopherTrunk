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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A dongle on a remote computer streaming IQ over the network to GopherTrunk on another machine." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <path d="M40 60 v-22 m-7 0 l7 -10 l7 10" fill="none" stroke="currentColor" stroke-width="1.6"/>
    <rect x="70" y="44" width="92" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="116" y="58">Pi + dongle</text><text x="116" y="69" font-size="7.5">rtl_tcp server</text>
    <rect x="300" y="44" width="110" height="30" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="355" y="63">GopherTrunk</text>
    <line x1="162" y1="59" x2="299" y2="59" stroke="currentColor" stroke-width="1.2" stroke-dasharray="4 3" marker-end="url(#rtar)"/><text x="230" y="51">IQ over network</text>
    <line x1="52" y1="59" x2="69" y2="59" stroke="currentColor" stroke-width="1.1"/>
  </g>
  <defs><marker id="rtar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>rtl_tcp streams IQ over the network, so the dongle can sit at the antenna while decoding runs elsewhere.</figcaption>
</figure>

## Overview

A common setup runs rtl_tcp on a Raspberry Pi right at the [antenna](/reference/antenna/)
— keeping the coax run short — while the decoder runs on a more capable host elsewhere.

## Relevance to SDR

GopherTrunk can attach to an rtl_tcp (or [SoapyRemote](/reference/soapysdr/)) source,
treating a remote dongle like a local one.
