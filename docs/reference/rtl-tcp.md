---
slug: rtl-tcp
title: rtl_tcp
entry_type: technology
category: sdr-data-streaming
description: rtl_tcp is a small server that streams raw IQ samples from an RTL-SDR over a TCP network connection, allowing the dongle to be used remotely.
keywords: rtl_tcp, network SDR, remote RTL-SDR, IQ over TCP, Raspberry Pi SDR, Osmocom, rtl-sdr tools, remote dongle
aka: [rtl_tcp]
autolink: true
infobox:
  - { label: Type, value: Network IQ server }
  - { label: Streams, value: Raw IQ from RTL-SDR over TCP }
  - { label: Use, value: Dongle at antenna, decode elsewhere }
see_also: [rtl-sdr, soapysdr, gqrx, gnuradio, iq-data, software-defined-radio]
related_lessons:
  - { title: "SDR hardware — RTL-SDR, HackRF, Airspy", url: /learn/rf-sdr/sdr-hardware/ }
cite_urls:
  - https://osmocom.org/projects/rtl-sdr/wiki
  - https://en.wikipedia.org/wiki/Rtl-sdr
---

**rtl_tcp** is a small server that streams raw [IQ](/reference/iq-data/) samples from an
[RTL-SDR](/reference/rtl-sdr/) over a **TCP** connection, letting the dongle be used from
another machine on the network.[^osmo] It is part of the Osmocom rtl-sdr toolset and is the
simplest, oldest way to put physical distance between the radio and the software decoding it.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A dongle on a remote Raspberry Pi streaming IQ over the network to GopherTrunk on another machine." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <path d="M40 60 v-22 m-7 0 l7 -10 l7 10" fill="none" stroke="currentColor" stroke-width="1.6"/>
    <rect x="70" y="44" width="92" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="116" y="58">Pi + dongle</text><text x="116" y="69" font-size="7.5">rtl_tcp server</text>
    <rect x="300" y="44" width="110" height="30" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="355" y="63">GopherTrunk</text>
    <line x1="162" y1="59" x2="299" y2="59" stroke="currentColor" stroke-width="1.2" stroke-dasharray="4 3" marker-end="url(#rtar)"/><text x="230" y="51">IQ over TCP</text>
    <line x1="52" y1="59" x2="69" y2="59" stroke="currentColor" stroke-width="1.1"/>
  </g>
  <defs><marker id="rtar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>rtl_tcp streams IQ over the network, so the dongle can sit at the antenna while decoding runs elsewhere.</figcaption>
</figure>

## How it works

You launch `rtl_tcp -a <address> -p <port>` on the host that has the dongle. It opens the
RTL-SDR, sets a default centre frequency and [sample rate](/reference/sample-rate/), and
listens for a TCP client. When a client connects, rtl_tcp reads 8-bit unsigned I and Q bytes
straight off the RTL2832U and pushes them down the socket as a continuous stream — no
compression, no framing beyond the raw byte pairs. In the reverse direction the client sends
short **command messages** (a one-byte command plus a 32-bit value) to retune, change the
sample rate, or adjust gain, so the remote radio remains fully controllable. The wire format
is deliberately trivial, which is why almost every SDR application — including
[GQRX](/reference/gqrx/), SDR#, and [GNU Radio](/reference/gnuradio/) via `gr-osmosdr` —
speaks it.

Two consequences follow from streaming *raw* 8-bit IQ. First, the bandwidth is fixed by the
sample rate: 2.4 MS/s of complex 8-bit samples is about 38 Mbit/s on the wire, so a fast
LAN is fine but a constrained link is not. Second, there is no buffering intelligence — a
network hiccup shows up as dropped samples, which a decoder sees as a momentary loss of
lock. For this reason rtl_tcp is at its best over wired Ethernet or solid Wi-Fi on the same
network segment.

## Variants and alternatives

- **rtl_tcp** — RTL-SDR only, tiny and universal; the default when the radio is a dongle.
- **[SoapyRemote](/reference/soapysdr/)** — the general-purpose successor: works for any
  SoapySDR-supported radio (HackRF, Airspy, SDRplay), not just RTL, and exposes the full
  device API rather than rtl_tcp's fixed command set.
- **spyserver** — Airspy's optimised network protocol, which can send a decimated slice
  rather than the whole band to save bandwidth.
- Vendor tools such as **rtl_sdr** (file capture) and **rtl_fm** (built-in FM demod) share
  the same library but run locally rather than over the network.

## In practice

The classic deployment runs rtl_tcp on a Raspberry Pi mounted right at the
[antenna](/reference/antenna/), keeping the [coax](/reference/coaxial-cable/) run short and
its loss low, while the CPU-heavy decoding runs on a desktop or server elsewhere in the
building. This is often a better signal path than a long feedline to a centrally located
radio, because feedline loss ahead of the SDR is unrecoverable whereas a short digital link
after it is essentially lossless.

## Relevance to SDR

rtl_tcp is a cornerstone of remote and headless SDR setups and one of the first tools most
RTL-SDR users encounter. GopherTrunk can attach to an rtl_tcp (or
[SoapyRemote](/reference/soapysdr/)) source, treating a remote dongle like a local one — so
the antenna, LNA, and dongle can live at the mast while GopherTrunk decodes the trunking
traffic on a machine with the horsepower to run its DSP chain.

## Sources

[^osmo]: [rtl-sdr (osmocom)](https://osmocom.org/projects/rtl-sdr/wiki) — the Osmocom rtl-sdr project wiki, home of rtl_tcp and the RTL-SDR driver tools.
[^rtl]: [RTL-SDR](https://en.wikipedia.org/wiki/Rtl-sdr) — Wikipedia, on the RTL2832U-based dongles and the Osmocom software stack that rtl_tcp belongs to.
