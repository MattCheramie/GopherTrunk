---
slug: soc-vs-discrete
title: SoC vs discrete
entry_type: concept
category: hw-accelerators
description: SoC vs discrete is the design trade-off between integrating processors and accelerators onto one system-on-a-chip versus using separate dedicated chips connected on a board.
keywords: SoC, system on a chip, discrete, integrated GPU, dedicated GPU, integration, accelerator, trade-off, edge, data center
infobox:
  - { label: Type, value: Design trade-off }
  - { label: SoC, value: One chip, shared memory, low power }
  - { label: Discrete, value: Separate chips, own memory, max performance }
  - { label: SoC wins, value: Power, size, cost, edge devices }
  - { label: Discrete wins, value: Peak performance, upgradability }
see_also: [system-on-a-chip, graphics-processing-unit, hardware-acceleration, ai-accelerator, neural-processing-unit, edge-ai]
cite_urls:
  - https://en.wikipedia.org/wiki/System_on_a_chip
---

**SoC vs discrete** is the design trade-off between integrating the processor and its accelerators onto a single [system-on-a-chip](/reference/system-on-a-chip/) versus using separate, dedicated chips wired together on a board.[^wiki]

## Overview

An integrated approach puts the CPU, [GPU](/reference/graphics-processing-unit/), [NPU](/reference/neural-processing-unit/), memory controller, and I/O on one die that shares a single pool of memory. A discrete approach gives an accelerator its own chip and its own high-bandwidth memory, connected over a bus such as PCI Express. Integration shortens the distance data has to travel, cutting power and latency; separation lets each part be made as large and fast as its own package allows.

## Trade-offs

The SoC wins on power, physical size, and cost — which is why phones, single-board computers, and edge devices use integrated [accelerators](/reference/ai-accelerator/). The discrete part wins on peak performance and upgradability, which is why data-center training rigs and gaming PCs bolt on standalone GPUs with dedicated memory. The choice mirrors the broader [hardware-acceleration](/reference/hardware-acceleration/) question of where compute should live. For a GopherTrunk capture node, an integrated SoC board by the antenna is the natural pick: low power and small, with the radio front end doing the heavy RF work.

## Sources

[^wiki]: [System on a chip](https://en.wikipedia.org/wiki/System_on_a_chip) — Wikipedia, on integrating components onto a single chip versus discrete designs.
