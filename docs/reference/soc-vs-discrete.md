---
slug: soc-vs-discrete
title: SoC vs discrete
entry_type: concept
category: hw-accelerators
description: SoC vs discrete is the design trade-off between integrating processors and accelerators onto one system-on-a-chip with shared memory versus using separate dedicated chips, each with its own memory, connected on a board.
keywords: SoC, system on a chip, discrete, integrated GPU, dedicated GPU, shared memory, PCI Express, integration, accelerator, trade-off, edge, data center
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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 168" role="img" aria-label="On the left, a single system-on-a-chip die holds the CPU, GPU, NPU, and modem sharing one memory pool. On the right, separate CPU and GPU chips each have their own memory and connect over a PCI Express bus on a board." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor">
    <text x="105" y="20" text-anchor="middle" font-size="9" stroke="none" font-weight="600">SoC (integrated)</text>
    <rect x="20" y="30" width="170" height="86" rx="6" fill-opacity="0.06" stroke-width="1.5"/>
    <rect x="32" y="42" width="52" height="26" rx="3" fill-opacity="0.16" stroke-width="1"/>
    <text x="58" y="59" text-anchor="middle" font-size="7.5" stroke="none">CPU</text>
    <rect x="92" y="42" width="52" height="26" rx="3" fill-opacity="0.16" stroke-width="1"/>
    <text x="118" y="59" text-anchor="middle" font-size="7.5" stroke="none">GPU</text>
    <rect x="32" y="74" width="52" height="26" rx="3" fill-opacity="0.16" stroke-width="1"/>
    <text x="58" y="91" text-anchor="middle" font-size="7.5" stroke="none">NPU</text>
    <rect x="92" y="74" width="52" height="26" rx="3" fill-opacity="0.16" stroke-width="1"/>
    <text x="118" y="91" text-anchor="middle" font-size="7.5" stroke="none">modem</text>
    <rect x="150" y="42" width="30" height="58" rx="3" fill-opacity="0.10" stroke-width="1"/>
    <text x="165" y="86" text-anchor="middle" font-size="7" stroke="none" transform="rotate(-90 165 74)">shared RAM</text>
    <text x="105" y="130" text-anchor="middle" font-size="7.5" stroke="none" fill-opacity="0.9">one die &#183; low power &#183; small</text>
  </g>
  <g stroke="currentColor" fill="currentColor">
    <text x="352" y="20" text-anchor="middle" font-size="9" stroke="none" font-weight="600">Discrete (separate)</text>
    <rect x="256" y="30" width="192" height="86" rx="6" fill-opacity="0.03" stroke-width="1.4" stroke-dasharray="4 3"/>
    <rect x="268" y="46" width="60" height="30" rx="3" fill-opacity="0.16" stroke-width="1"/>
    <text x="298" y="65" text-anchor="middle" font-size="7.5" stroke="none">CPU</text>
    <rect x="268" y="82" width="60" height="18" rx="2" fill-opacity="0.08" stroke-width="0.9"/>
    <text x="298" y="94" text-anchor="middle" font-size="6.5" stroke="none">RAM</text>
    <rect x="376" y="46" width="60" height="30" rx="3" fill-opacity="0.16" stroke-width="1"/>
    <text x="406" y="65" text-anchor="middle" font-size="7.5" stroke="none">GPU</text>
    <rect x="376" y="82" width="60" height="18" rx="2" fill-opacity="0.08" stroke-width="0.9"/>
    <text x="406" y="94" text-anchor="middle" font-size="6.5" stroke="none">VRAM</text>
    <line x1="328" y1="61" x2="376" y2="61" stroke-width="1.3" fill="none"/>
    <text x="352" y="56" text-anchor="middle" font-size="6.5" stroke="none" fill-opacity="0.9">PCIe</text>
    <text x="352" y="130" text-anchor="middle" font-size="7.5" stroke="none" fill-opacity="0.9">own memory each &#183; peak performance</text>
  </g>
  <text x="230" y="158" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.9">integration cuts the distance data travels; separation lets each part grow as large as its own package</text>
</svg>
<figcaption>An SoC packs the CPU, GPU, NPU, and modem onto one die sharing a single memory pool; a discrete design gives each chip its own memory and connects them over a bus like PCI Express — trading power and size against peak performance.</figcaption>
</figure>

## Overview

An integrated approach puts the CPU, [GPU](/reference/graphics-processing-unit/), [NPU](/reference/neural-processing-unit/), memory controller, and I/O on one die that shares a single pool of memory. A discrete approach gives an accelerator its own chip and its own high-bandwidth memory, connected over a bus such as PCI Express. Integration shortens the distance data has to travel, cutting power and latency; separation lets each part be made as large and fast as its own package allows.

Shared memory is the pivotal difference. On an SoC the CPU and its accelerators read the same RAM, so passing data between them can be nearly free — no copy across a bus. A discrete accelerator has its own dedicated memory that is faster and larger than what an SoC can share, but every hand-off means copying data across the interconnect, which costs both time and energy.

## Trade-offs

The two designs optimise for opposite ends of the power-and-performance curve, and the right pick follows the device it lives in:

| Aspect | SoC (integrated) | Discrete (separate) |
|--------|------------------|---------------------|
| Memory | Shared pool | Own dedicated memory |
| Power &amp; size | Low, compact | High, bulky |
| Peak performance | Limited by die area | As large as its package |
| Data movement | Nearly free on-die | Copy across the bus |
| Upgradability | Fixed | Swap the card |
| Typical home | Phones, SBCs, edge | Data centers, gaming PCs |

## Where it fits

The SoC wins on power, physical size, and cost — which is why phones, single-board computers, and edge devices use integrated [accelerators](/reference/ai-accelerator/). The discrete part wins on peak performance and upgradability, which is why data-center training rigs and gaming PCs bolt on standalone GPUs with dedicated memory. The choice mirrors the broader [hardware-acceleration](/reference/hardware-acceleration/) question of where compute should live. For a GopherTrunk capture node, an integrated SoC board by the antenna is the natural pick: low power and small, with the radio front end doing the heavy RF work, while a rack-mounted decode server can justify a discrete GPU for many-channel processing.

## Sources

[^wiki]: [System on a chip](https://en.wikipedia.org/wiki/System_on_a_chip) — Wikipedia, on integrating components onto a single chip versus discrete designs and shared versus dedicated memory.
