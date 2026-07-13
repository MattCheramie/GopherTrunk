---
slug: diplexer
title: Diplexer
entry_type: hardware
category: rf-front-end
description: "A diplexer is a three-port filter network that splits or combines two frequency bands on one common port, letting two radios share one antenna or feedline."
keywords: diplexer, frequency splitting, band splitting, high pass low pass, crossover network, dual-band antenna sharing, combiner, common port, triplexer
aka: [diplexer]
autolink: true
infobox:
  - { label: Type, value: "Frequency-selective RF network" }
  - { label: Ports, value: "Three (common + two bands)" }
  - { label: Function, value: "Split/combine two bands on one line" }
  - { label: Key spec, value: "Insertion loss, band isolation" }
see_also: [duplexer, rf-filter, splitter-combiner, guard-band, cavity-filter]
cite_urls:
  - https://en.wikipedia.org/wiki/Diplexer
  - https://en.wikipedia.org/wiki/Multiplexer
---

A **diplexer** is a three-port filter network that **splits or combines two frequency bands**
on a single common port.[^wiki] It lets two radios operating in different bands share one
[antenna](/reference/antenna/) or feedline: energy in band A flows between the common port and
the band-A port, energy in band B flows between the common port and the band-B port, and the
two bands stay isolated from each other.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A diplexer with a common antenna port branching into a low-pass filter to a low-band port and a high-pass filter to a high-band port, separating two frequency bands." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <line x1="40" y1="75" x2="130" y2="75"/>
    <line x1="130" y1="75" x2="180" y2="40"/>
    <line x1="130" y1="75" x2="180" y2="110"/>
    <rect x="180" y="28" width="66" height="26" rx="4"/>
    <rect x="180" y="98" width="66" height="26" rx="4"/>
    <line x1="246" y1="41" x2="340" y2="41"/>
    <line x1="246" y1="111" x2="340" y2="111"/>
  </g>
  <circle cx="130" cy="75" r="3" fill="currentColor"/>
  <g font-size="9" fill="currentColor">
    <text x="55" y="67">antenna</text>
    <text x="213" y="45" text-anchor="middle" font-size="8">low-pass</text>
    <text x="213" y="115" text-anchor="middle" font-size="8">high-pass</text>
    <text x="348" y="45">low band</text>
    <text x="348" y="115">high band</text>
  </g>
</svg>
<figcaption>A diplexer routes each band through its own filter, so two radios in different bands can share one common antenna port.</figcaption>
</figure>

## Overview

The defining feature of a diplexer is that it separates by **frequency band**, and — unlike a
switch — it does so passively and simultaneously, so both bands are live at once. A classic
example is a dual-band VHF/UHF station: a diplexer joins a 2 m and a 70 cm radio to one
dual-band antenna, each seeing only its own band. It is essentially an RF crossover network,
the radio-frequency analogue of the crossover that splits woofer and tweeter in a loudspeaker.

## How it works

A diplexer is built from two complementary **[filters](/reference/rf-filter/)** meeting at the
common port. In the simplest two-band form, a **low-pass** filter carries the lower band and a
**high-pass** filter the upper band; their crossover is placed in the [guard band](/reference/guard-band/)
between them. Each filter presents a high impedance (an open) to the other's band, so the two
paths do not load one another and energy is steered to the correct port. For bands that are
close together, the simple high/low-pass pair is replaced by two **band-pass** sections — often
sharp [cavity filters](/reference/cavity-filter/) — to get enough separation. A **triplexer**
extends the idea to three bands.

Key specifications are **insertion loss** in each path (kept low so little signal is wasted)
and **isolation** between the two band ports (how well band A is kept out of the band-B radio).

## Diplexer vs duplexer

The names are often confused. A **[duplexer](/reference/duplexer/)** separates the *transmit and
receive frequencies of one service* — a closely spaced pair on the same band — to share an
antenna between a TX and an RX. A **diplexer** separates *two different frequency bands*, each of
which may itself carry both TX and RX. In short: a duplexer splits a TX/RX pair; a diplexer
splits two bands.

## Relevance to SDR

Diplexers are common in multi-band installations and are handy for SDR users who want to feed a
single wideband antenna and feedline to more than one receiver or to combine antennas optimised
for different bands. A diplexer can also serve as a crude preselector, keeping a strong
out-of-band signal (say broadcast FM) out of the path feeding a VHF/UHF receiver, reducing
overload and [intermodulation](/reference/intermodulation/).

**GopherTrunk** is software and contains no diplexer; the component is purely part of the
antenna and feed system upstream of the SDR. It is relevant to GT users only as a way to share
or clean up the RF feeding the receiver that produces GT's I/Q stream.

## Sources

[^wiki]: [Diplexer](https://en.wikipedia.org/wiki/Diplexer) — Wikipedia, on three-port band-splitting networks and the distinction from duplexers.
