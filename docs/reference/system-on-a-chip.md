---
slug: system-on-a-chip
title: System on a chip (SoC)
entry_type: hardware
category: hw-mobile
description: A system on a chip (SoC) integrates a computer's major subsystems — CPU, GPU, memory controller, radios, and accelerators — onto a single piece of silicon, the building block of nearly every phone and small device.
keywords: system on a chip, SoC, application processor, integrated CPU GPU, mobile chip, Snapdragon, Apple Silicon, SoC vs discrete
aka: [SoC]
autolink: true
infobox:
  - { label: Type, value: Integrated circuit }
  - { label: Integrates, value: CPU, GPU, memory & I/O controllers }
  - { label: Common in, value: Phones, tablets, SBCs }
  - { label: Examples, value: Snapdragon, Apple Silicon, Broadcom }
see_also: [mobile-operating-system, arm-architecture, central-processing-unit, soc-vs-discrete, integrated-circuit, cellular-modem]
cite_urls:
  - https://en.wikipedia.org/wiki/System_on_a_chip
---

A **system on a chip (SoC)** is an integrated circuit that combines a computer's major subsystems — processor cores, graphics, memory and I/O controllers, and often radios and accelerators — onto a single die.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 236" role="img" aria-label="A single system-on-chip die containing labelled blocks — CPU cores, GPU, and NPU across the top, and a memory controller, cellular modem, and GPS below — all wired to a shared on-chip interconnect running across the middle." xmlns="http://www.w3.org/2000/svg">
  <text x="230" y="18" text-anchor="middle" font-size="11" fill="currentColor" font-weight="600">One SoC die</text>
  <rect x="15" y="30" width="430" height="196" rx="10" fill="currentColor" fill-opacity="0.04" stroke="currentColor" stroke-width="1.5"/>
  <g font-size="10" fill="currentColor" text-anchor="middle">
    <rect x="34" y="46" width="116" height="40" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/>
    <text x="92" y="63">CPU cores</text>
    <text x="92" y="77" font-size="8">general compute</text>
    <rect x="172" y="46" width="100" height="40" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/>
    <text x="222" y="63">GPU</text>
    <text x="222" y="77" font-size="8">graphics</text>
    <rect x="294" y="46" width="120" height="40" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/>
    <text x="354" y="63">NPU</text>
    <text x="354" y="77" font-size="8">on-device ML</text>
    <rect x="34" y="112" width="380" height="20" rx="4" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.2"/>
    <text x="224" y="126" font-size="9" font-weight="600">on-chip interconnect</text>
    <rect x="34" y="158" width="120" height="40" rx="5" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.2"/>
    <text x="94" y="175">memory ctrl</text>
    <text x="94" y="189" font-size="8">to RAM</text>
    <rect x="176" y="158" width="110" height="40" rx="5" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.2"/>
    <text x="231" y="175">cellular modem</text>
    <text x="231" y="189" font-size="8">radio</text>
    <rect x="308" y="158" width="106" height="40" rx="5" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.2"/>
    <text x="361" y="175">GPS</text>
    <text x="361" y="189" font-size="8">location</text>
  </g>
  <g stroke="currentColor" stroke-width="1" stroke-opacity="0.55">
    <line x1="92" y1="86" x2="92" y2="112"/>
    <line x1="222" y1="86" x2="222" y2="112"/>
    <line x1="354" y1="86" x2="354" y2="112"/>
    <line x1="94" y1="132" x2="94" y2="158"/>
    <line x1="231" y1="132" x2="231" y2="158"/>
    <line x1="361" y1="132" x2="361" y2="158"/>
  </g>
</svg>
<figcaption>An SoC pulls the parts a desktop spreads across a motherboard onto one die: compute blocks (CPU, GPU, NPU) and I/O blocks (memory, modem, GPS) all share a single on-chip interconnect. Shorter traces mean smaller, cheaper, and lower-power — at the cost of upgradability.</figcaption>
</figure>

## Overview

Where a desktop spreads its parts across a [motherboard](/reference/motherboard/), an SoC packs them into one chip: one or more [CPU](/reference/central-processing-unit/) cores, a [GPU](/reference/graphics-processing-unit/), a memory controller, and blocks such as a [cellular modem](/reference/cellular-modem/), [GPS receiver](/reference/gps-receiver/), and an [NPU](/reference/neural-processing-unit/) for on-device machine learning. Most are built on the [Arm architecture](/reference/arm-architecture/). Familiar examples include Qualcomm's Snapdragon, Apple Silicon, and the Broadcom parts inside the [Raspberry Pi](/reference/raspberry-pi/).

## Where it fits

Integration is what makes a [smartphone](/reference/smartphone/) small, cheap, and power-efficient: shorter traces and shared silicon cut size and energy use, at the cost of the upgradability you get from [discrete](/reference/soc-vs-discrete/) parts. The same logic puts SoCs in tablets, single-board computers, and embedded gear. A capture node running GopherTrunk on a Pi leans on its Broadcom SoC for everything but the radio front end, which still needs an external SDR dongle.

## Sources

[^wiki]: [System on a chip](https://en.wikipedia.org/wiki/System_on_a_chip) — Wikipedia, on SoC integration and uses.
