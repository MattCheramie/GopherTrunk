---
slug: single-frequency-network
title: Single-frequency network (SFN)
entry_type: technology
category: broadcast
description: A single-frequency network has many transmitters broadcast the identical signal on the same frequency, tightly time-synchronised so OFDM's guard interval turns the overlapping copies into constructive multipath; used by DAB, DVB-T, and ATSC 3.0.
keywords: SFN, single-frequency network, multi-frequency network, MFN, OFDM, guard interval, GPS synchronisation, DAB, DVB-T, ISDB-T, ATSC 3.0, simulcast
aka: [SFN, Single-frequency network]
autolink: true
infobox:
  - { label: Type, value: Broadcast transmitter network }
  - { label: Key enabler, value: "OFDM guard interval" }
  - { label: Synchronisation, value: GPS-disciplined timing }
  - { label: Used by, value: "DAB, DVB-T/T2, ISDB-T, ATSC 3.0" }
see_also: [ofdm, dab, dvb-t, atsc-3, isdb-t, simulcast, guard-band, multipath-propagation]
cite_urls:
  - https://en.wikipedia.org/wiki/Single-frequency_network
  - https://en.wikipedia.org/wiki/Orthogonal_frequency-division_multiplexing
---

A **single-frequency network** (**SFN**) has multiple transmitters broadcast the
*identical* signal on the **same frequency**, tightly synchronised in time so their
overlapping coverage reinforces rather than clashes.[^wiki] It is made possible by
[OFDM](/reference/ofdm/): the modulation's guard interval absorbs the delayed copies
arriving from neighbouring transmitters as constructive
[multipath](/reference/multipath-propagation/) instead of interference.

<figure class="figure" markdown="0">
<svg viewBox="0 0 420 220" role="img" aria-label="Three transmitter towers, all labelled frequency f-zero, with overlapping coverage circles and a receiver sitting in the region where all three overlap." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.2" fill="none" stroke-opacity="0.45">
    <circle cx="120" cy="90" r="80"/><circle cx="290" cy="90" r="80"/><circle cx="205" cy="160" r="80"/>
  </g>
  <g stroke="currentColor" stroke-width="1.4" fill="currentColor">
    <g><line x1="120" y1="90" x2="120" y2="55"/><path d="M120 55 L112 40 M120 55 L128 40" fill="none"/><circle cx="120" cy="90" r="3"/></g>
    <g><line x1="290" y1="90" x2="290" y2="55"/><path d="M290 55 L282 40 M290 55 L298 40" fill="none"/><circle cx="290" cy="90" r="3"/></g>
    <g><line x1="205" y1="160" x2="205" y2="125"/><path d="M205 125 L197 110 M205 125 L213 110" fill="none"/><circle cx="205" cy="160" r="3"/></g>
  </g>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="120" y="36">f₀</text><text x="290" y="36">f₀</text><text x="205" y="106">f₀</text>
  </g>
  <g><path d="M205 120 l-7 -12 h14 z" fill="currentColor"/><circle cx="205" cy="112" r="4" fill="none" stroke="currentColor" stroke-width="1.3"/></g>
  <text x="205" y="200" text-anchor="middle" font-size="9" fill="currentColor">receiver in the overlap hears all three as constructive multipath</text>
</svg>
<figcaption>Every tower radiates the identical signal on the same frequency f₀; a receiver in the overlap combines the time-aligned copies as reinforcement, not interference.</figcaption>
</figure>

## How it works

In an ordinary network, two transmitters on the same channel would jam each other wherever their
coverage met. OFDM changes the arithmetic. It divides the signal across many narrow subcarriers
and prepends each symbol with a **guard interval** — a short copy of the symbol's tail. Any echo
that arrives within that guard window, whether it is a reflection off terrain or a copy from a
second transmitter, lands inside the same symbol period and adds coherently at the receiver.
Provided every transmitter radiates bit-for-bit the same waveform at the same instant, a distant
transmitter is indistinguishable from a natural echo, and the receiver treats the sum as
constructive [multipath](/reference/multipath-propagation/).

Two conditions make this work. The transmitters must be **frequency- and time-locked**, which in
practice means GPS-disciplined references so their clocks and symbol timing agree to a fraction
of the guard interval. And the network's inter-transmitter spacing must keep differential delays
within that guard window — a longer guard interval permits wider spacing at the cost of some
throughput. Get either wrong and the delayed copies fall outside the guard window and become
inter-symbol interference instead of reinforcement.

## In practice

SFNs save spectrum. A **multi-frequency network (MFN)** gives each transmitter its own channel to
avoid mutual interference, consuming several channels to cover a region; an SFN covers the same
region on a single channel, freeing the rest for other services. Digital broadcasting adopted the
technique widely: **[DAB](/reference/dab/)** digital radio, **[DVB-T](/reference/dvb-t/)** and
DVB-T2 television, Japan's **[ISDB-T](/reference/isdb-t/)**, and
**[ATSC 3.0](/reference/atsc-3/)** all run SFNs, and HD Radio uses on-channel boosters on the same
principle. The SFN is the digital-broadcast cousin of analog land-mobile
[simulcast](/reference/simulcast/), which likewise keys several transmitters together on one
frequency — but where analog simulcast must carefully manage overlap distortion, OFDM's guard
interval handles it by design.

## Relevance to SDR

An SDR receiving a digital-broadcast multiplex in an SFN region is, without any special handling,
demodulating the coherent sum of several transmitters — one reason OFDM broadcasts stay robust
indoors and in cluttered terrain. Understanding SFNs explains why a strong digital signal can
originate from more than one tower at once, and why signal-quality metrics behave differently from
a single-transmitter link. SFN broadcasting sits outside GopherTrunk's land-mobile trunking scope,
but it is documented here as the OFDM network architecture behind the digital-radio and television
signals an SDR user encounters alongside trunked voice.

## Sources

[^wiki]: [Single-frequency network](https://en.wikipedia.org/wiki/Single-frequency_network) — Wikipedia, for the identical-signal same-frequency design, GPS synchronisation, and the SFN-versus-MFN spectrum trade.
[^ofdm]: [Orthogonal frequency-division multiplexing](https://en.wikipedia.org/wiki/Orthogonal_frequency-division_multiplexing) — Wikipedia, for the guard interval that lets an OFDM receiver absorb delayed copies as constructive multipath.
