---
slug: duplexer
title: Duplexer
entry_type: hardware
category: rf-front-end
description: "A duplexer lets a transmitter and receiver share one antenna by separating their frequencies with sharp filters, giving deep isolation so the TX does not swamp the RX."
keywords: duplexer, cavity duplexer, repeater duplexer, TX RX isolation, notch duplexer, bandpass duplexer, band-pass band-reject, frequency separation, cavity filter
aka: [duplexer, cavity duplexer]
autolink: true
infobox:
  - { label: Type, value: "Frequency-selective RF network" }
  - { label: Function, value: "Share one antenna: TX and RX" }
  - { label: Built from, value: "Cavity / notch filters" }
  - { label: Key spec, value: "TX-to-RX isolation" }
see_also: [cavity-filter, diplexer, circulator, rf-filter, guard-band]
cite_urls:
  - https://en.wikipedia.org/wiki/Duplexer
  - https://en.wikipedia.org/wiki/Cavity_filter
---

A **duplexer** lets a transmitter and receiver operate on a **single**
[antenna](/reference/antenna/) at the same time by separating their two frequencies with sharp
filters.[^wiki] Its whole purpose is **isolation**: preventing the powerful transmit signal from
reaching — and desensitising or damaging — the receiver only a small frequency offset away. It
is the defining component of a repeater.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A duplexer with three ports: the antenna connects through two filter banks, one passing the transmit frequency and rejecting the receive frequency, the other doing the reverse, so TX and RX share one antenna with deep isolation." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="90" y="30" width="70" height="34" rx="4"/>
    <rect x="90" y="100" width="70" height="34" rx="4"/>
    <line x1="160" y1="47" x2="250" y2="82"/>
    <line x1="160" y1="117" x2="250" y2="82"/>
    <line x1="250" y1="82" x2="420" y2="82"/>
    <line x1="30" y1="47" x2="90" y2="47"/>
    <line x1="30" y1="117" x2="90" y2="117"/>
  </g>
  <g font-size="9" fill="currentColor">
    <text x="20" y="40" text-anchor="end">TX</text>
    <text x="20" y="120" text-anchor="end">RX</text>
    <text x="125" y="51" text-anchor="middle" font-size="8">pass TX,</text><text x="125" y="60" text-anchor="middle" font-size="8">reject RX</text>
    <text x="125" y="121" text-anchor="middle" font-size="8">pass RX,</text><text x="125" y="130" text-anchor="middle" font-size="8">reject TX</text>
    <text x="380" y="74">antenna</text>
  </g>
</svg>
<figcaption>Two sharp filter banks join at the antenna: one passes the transmit frequency, the other the receive frequency, giving deep TX-to-RX isolation.</figcaption>
</figure>

## Overview

A repeater listens on one frequency and re-transmits on another, offset by a fixed amount
(the *split*). Both must use the same antenna, yet the transmitter may put out tens of watts
while the receiver is trying to hear microwatts on a frequency perhaps only a few hundred kHz
away. Bridging that enormous ratio — often needing 80–100 dB of TX-to-RX isolation — is exactly
what a duplexer does, and why it is built from very selective filters.

## How it works

A duplexer is a three-port network: antenna, transmit, receive. Between the antenna and each
radio sits a filter tuned so that the wanted frequency passes with low loss while the *other*
frequency is deeply rejected. Two common styles:

- **Band-pass / band-reject (notch)** — each leg has a sharp notch at the opposite frequency,
  so the TX leg notches out the receive frequency and vice versa. Compact and efficient for
  moderate splits.
- **Band-pass** — each leg passes only its own frequency; more insertion loss but broader
  rejection, useful in RF-dense sites.

The filters are almost always resonant **[cavity filters](/reference/cavity-filter/)** — tuned
metal cans with very high [Q](/reference/q-factor/) — because only high-Q resonators can be that
selective at high power with low loss. The achievable isolation depends on how far apart the two
frequencies are: a wider split (or [guard band](/reference/guard-band/)) makes the filtering
job easier.

## Relevance to SDR

Duplexers are ubiquitous in land-mobile radio: every VHF/UHF repeater, and the base stations of
the trunked systems GopherTrunk targets — [P25](/reference/p25-phase-1/),
[DMR](/reference/dmr/), and similar — use one to share the tower antenna between transmit and
receive. A duplexer separates TX and RX by **frequency**, which is a different approach from a
[circulator](/reference/circulator/) (separation by direction) and closely related to a
[diplexer](/reference/diplexer/) (which splits two *bands* rather than a TX/RX pair on one
service).

**GopherTrunk** is a receive-only decoder and neither contains nor needs a duplexer. The device
matters to GT only as part of the infrastructure it monitors: a mistuned or degraded duplexer at
a site raises the transmitter's noise and can desensitise nearby receivers, degrading the
signals GT decodes.

## Sources

[^wiki]: [Duplexer](https://en.wikipedia.org/wiki/Duplexer) — Wikipedia, on shared-antenna TX/RX operation, isolation, and cavity-filter duplexers.
