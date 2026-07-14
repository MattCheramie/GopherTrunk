---
slug: pci-express
title: PCI Express (PCIe)
entry_type: hardware
category: hw-foundations
description: PCI Express is the high-speed serial expansion standard that connects GPUs, SSDs, and other add-in cards to a computer over scalable point-to-point lanes.
keywords: PCI Express, PCIe, expansion slot, lanes, serial bus, add-in card, NVMe, x16
aka: [PCIe, PCI-E]
infobox:
  - { label: Type, value: Serial expansion bus }
  - { label: Topology, value: Point-to-point lanes }
  - { label: Widths, value: x1, x4, x8, x16 }
  - { label: Connects, value: GPUs, SSDs, NICs }
see_also: [system-bus, motherboard, graphics-processing-unit, usb, chipset, data-storage]
cite_urls:
  - https://en.wikipedia.org/wiki/PCI_Express
---

**PCI Express** (**PCIe**) is the high-speed serial expansion standard that connects add-in cards — graphics, storage, networking — to a computer's [chipset](/reference/chipset/) and CPU.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 236" role="img" aria-label="The top half shows an old shared parallel bus where three devices tap one common set of wires and contend for its bandwidth; the bottom half shows PCI Express giving each device — a GPU, an SSD, and a NIC — its own dedicated point-to-point link of x16, x4, or x1 lanes to a root or switch." xmlns="http://www.w3.org/2000/svg">
  <text x="150" y="20" text-anchor="middle" font-size="9" fill="currentColor" font-weight="600">shared parallel bus (old)</text>
  <g fill="currentColor" font-size="8" text-anchor="middle">
    <rect x="52" y="30" width="64" height="26" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="84" y="47">device</text>
    <rect x="162" y="30" width="64" height="26" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="194" y="47">device</text>
    <rect x="272" y="30" width="64" height="26" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="304" y="47">device</text>
  </g>
  <line x1="52" y1="74" x2="352" y2="74" stroke="currentColor" stroke-width="3"/>
  <g stroke="currentColor" stroke-width="1" stroke-opacity="0.6"><line x1="84" y1="56" x2="84" y2="74"/><line x1="194" y1="56" x2="194" y2="74"/><line x1="304" y1="56" x2="304" y2="74"/></g>
  <text x="204" y="92" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.85">one set of wires — devices contend for bandwidth</text>
  <line x1="20" y1="108" x2="440" y2="108" stroke="currentColor" stroke-width="1" stroke-opacity="0.3" stroke-dasharray="5 3"/>
  <text x="160" y="126" text-anchor="middle" font-size="9" fill="currentColor" font-weight="600">PCI Express — point-to-point</text>
  <rect x="40" y="146" width="100" height="60" rx="6" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.3"/>
  <text x="90" y="172" text-anchor="middle" font-size="9" fill="currentColor" font-weight="600">root /</text>
  <text x="90" y="186" text-anchor="middle" font-size="9" fill="currentColor" font-weight="600">switch</text>
  <g fill="currentColor" font-size="8.5" text-anchor="middle">
    <rect x="316" y="138" width="110" height="26" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="371" y="155">GPU</text>
    <rect x="316" y="172" width="110" height="26" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="371" y="189">SSD</text>
    <rect x="316" y="206" width="110" height="26" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="371" y="223">NIC</text>
  </g>
  <g stroke="currentColor" stroke-width="1.4" fill="none">
    <line x1="140" y1="164" x2="316" y2="151" marker-end="url(#pcie_ar)"/>
    <line x1="140" y1="176" x2="316" y2="185" marker-end="url(#pcie_ar)"/>
    <line x1="140" y1="188" x2="316" y2="219" marker-end="url(#pcie_ar)"/>
  </g>
  <g fill="currentColor" font-size="7.5" text-anchor="middle">
    <text x="238" y="150">x16</text><text x="238" y="176">x4</text><text x="238" y="212">x1</text>
  </g>
  <defs><marker id="pcie_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The old parallel bus wired every device to one shared set of lines, so they competed for bandwidth. PCIe instead gives each device its own point-to-point link to a switch or the CPU's root complex, built from 1, 4, or 16 lanes (x1/x4/x16). Adding a card no longer slows the others.</figcaption>
</figure>

## Overview

PCIe replaced older shared parallel buses with scalable *point-to-point* links built from one or more *lanes*, each a pair of differential wires in each direction. A slot is described by its lane count — x1, x4, x8, x16 — and each new generation roughly doubles per-lane bandwidth. Unlike a classic [system bus](/reference/system-bus/), devices do not share one set of wires; each gets a dedicated link to a switch, so adding cards does not slow the others.

## What it's for

A [GPU](/reference/graphics-processing-unit/) typically takes a x16 slot, while NVMe [SSDs](/reference/data-storage/), network cards, and capture cards use narrower links. The slots and lanes live on the [motherboard](/reference/motherboard/), routed through the chipset. In a high-throughput SDR rig, a PCIe slot is where you would add a fast capture card or an accelerator to feed wideband samples into the decode pipeline without starving the bus.

## Sources

[^wiki]: [PCI Express](https://en.wikipedia.org/wiki/PCI_Express) — Wikipedia, on the PCIe serial expansion standard, lanes, and generations.
