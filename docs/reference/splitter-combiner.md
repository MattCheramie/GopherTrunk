---
slug: splitter-combiner
title: Splitter / combiner
entry_type: hardware
category: rf-front-end
description: "A splitter divides one RF signal into several ports (or a combiner does the reverse); the Wilkinson design adds a resistor to keep the outputs isolated and matched."
keywords: splitter, combiner, power divider, RF splitter, two-way splitter, Wilkinson divider, resistive splitter, reactive splitter, port isolation, insertion loss
aka: [splitter, combiner, power divider, power splitter]
autolink: true
infobox:
  - { label: Type, value: "Passive power-division network" }
  - { label: Function, value: "Divide or combine RF power" }
  - { label: Common types, value: "Resistive, reactive, Wilkinson" }
  - { label: 2-way loss, value: "~3 dB split + insertion loss" }
see_also: [directional-coupler, rf-filter, attenuator, antenna-diversity, return-loss]
cite_urls:
  - https://en.wikipedia.org/wiki/Power_dividers_and_directional_couplers
  - https://en.wikipedia.org/wiki/Wilkinson_power_divider
---

A **splitter** divides one RF signal into two or more output ports; a **combiner** is the same
device run in reverse, merging several signals onto one port.[^wiki] Splitting power *N* ways
means each output is inherently weaker — a two-way even split loses about **3 dB** to each port
before any real-world loss — so the design's job is to divide cleanly while keeping the ports
matched and, ideally, isolated from one another.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A two-way power splitter taking one input and dividing it into two equal outputs each about 3 dB down, with an isolation resistor between the outputs as in a Wilkinson divider." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <line x1="40" y1="75" x2="150" y2="75"/>
    <line x1="150" y1="75" x2="230" y2="45"/>
    <line x1="150" y1="75" x2="230" y2="105"/>
    <line x1="230" y1="45" x2="360" y2="45"/>
    <line x1="230" y1="105" x2="360" y2="105"/>
    <path d="M300 45 L300 62 L306 66 L294 72 L306 78 L294 84 L300 88 L300 105" stroke-width="1.1"/>
  </g>
  <circle cx="150" cy="75" r="3" fill="currentColor"/>
  <g font-size="9" fill="currentColor">
    <text x="60" y="67">in</text>
    <text x="368" y="49">out 1 (−3 dB)</text>
    <text x="368" y="109">out 2 (−3 dB)</text>
    <text x="315" y="78" font-size="8">iso R</text>
  </g>
</svg>
<figcaption>A two-way splitter divides the input into two equal outputs about 3 dB down; a Wilkinson resistor between them provides port isolation.</figcaption>
</figure>

## Overview

The core specifications of a divider are its **split ratio** (usually equal), **insertion loss**
above the theoretical split, **isolation** between output ports, and **return loss** at each
port. The unavoidable ~3 dB per port of a two-way split is not a defect but the arithmetic of
sharing power; a good divider adds only a small fraction of a dB of real loss on top and keeps
every port well matched.

## How it works

There are three common approaches, trading simplicity for performance:

- **Resistive** — a small network of resistors. Broadband and dead simple, but lossy (each
  output of a resistive two-way is ~6 dB down, not 3) and the ports are not isolated.
- **Reactive (transformer/transmission-line)** — uses transformers or quarter-wave lines to
  divide with near-ideal ~3 dB loss, but with limited isolation between outputs.
- **Wilkinson divider** — the workhorse. Two quarter-wave lines plus an **isolation resistor**
  between the outputs. The resistor dissipates no power when the load is balanced, yet provides
  good **isolation** and matches all ports. This is why it dominates RF splitter design.

Run any of these backwards and it becomes a **combiner**. Combining is subtler than splitting:
the sources must be in phase for their power to add efficiently, and the isolation resistor in a
Wilkinson absorbs the difference when they are not, protecting the sources from each other.

## Relevance to SDR

Splitters are everywhere in multi-receiver setups: feeding one antenna to several SDRs, tapping a
signal for a spectrum monitor, or distributing a common reference. A closely related device, the
**[directional coupler](/reference/directional-coupler/)**, is essentially an *unequal,
direction-selective* divider used for sampling rather than even division. In receiving arrays,
splitters and combiners route signals for [antenna diversity](/reference/antenna-diversity/) and
phased feeds. Remember the 3 dB-per-port cost: splitting one antenna to several receivers lowers
the signal at each, which a low-noise [preamplifier](/reference/preamplifier/) ahead of the
splitter can offset.

**GopherTrunk** is receive-only software and includes no splitter or combiner. The device is
purely part of the RF plumbing ahead of the SDR — relevant to GT users who want to share one
antenna among multiple receivers or SDR instances, where the split loss and port isolation
directly affect the quality of each receiver's I/Q stream.

## Sources

[^wiki]: [Power dividers and directional couplers](https://en.wikipedia.org/wiki/Power_dividers_and_directional_couplers) — Wikipedia, on resistive, reactive, and Wilkinson power dividers, insertion loss, and isolation.
