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

## Overview

PCIe replaced older shared parallel buses with scalable *point-to-point* links built from one or more *lanes*, each a pair of differential wires in each direction. A slot is described by its lane count — x1, x4, x8, x16 — and each new generation roughly doubles per-lane bandwidth. Unlike a classic [system bus](/reference/system-bus/), devices do not share one set of wires; each gets a dedicated link to a switch, so adding cards does not slow the others.

## What it's for

A [GPU](/reference/graphics-processing-unit/) typically takes a x16 slot, while NVMe [SSDs](/reference/data-storage/), network cards, and capture cards use narrower links. The slots and lanes live on the [motherboard](/reference/motherboard/), routed through the chipset. In a high-throughput SDR rig, a PCIe slot is where you would add a fast capture card or an accelerator to feed wideband samples into the decode pipeline without starving the bus.

## Sources

[^wiki]: [PCI Express](https://en.wikipedia.org/wiki/PCI_Express) — Wikipedia, on the PCIe serial expansion standard, lanes, and generations.
