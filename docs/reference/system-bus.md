---
slug: system-bus
title: System bus
entry_type: concept
category: hw-foundations
description: A system bus is the shared set of conductors that carries data, addresses, and control signals between a computer's CPU, memory, and I/O devices.
keywords: system bus, data bus, address bus, control bus, front-side bus, interconnect, bus width
infobox:
  - { label: Type, value: Shared interconnect }
  - { label: Carries, value: Data, address, control }
  - { label: Width, value: Bits transferred at once }
  - { label: Modern form, value: Point-to-point links }
see_also: [central-processing-unit, random-access-memory, pci-express, motherboard, von-neumann-architecture, input-output]
cite_urls:
  - https://en.wikipedia.org/wiki/Bus_(computing)
---

A **system bus** is the shared set of electrical conductors that moves data, addresses, and control signals between a computer's [CPU](/reference/central-processing-unit/), [memory](/reference/random-access-memory/), and [I/O](/reference/input-output/) devices.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 214" role="img" aria-label="A system bus drawn as three parallel shared lines — an address bus, a data bus, and a control bus — with the CPU tapping them from above and RAM and I/O devices tapping the same lines from below, so all three components exchange signals over the one interconnect." xmlns="http://www.w3.org/2000/svg">
  <rect x="44" y="22" width="120" height="32" rx="5" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.3"/>
  <text x="104" y="42" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">CPU</text>
  <line x1="24" y1="68" x2="384" y2="68" stroke="currentColor" stroke-width="1.6"/>
  <line x1="24" y1="92" x2="384" y2="92" stroke="currentColor" stroke-width="1.6"/>
  <line x1="24" y1="116" x2="384" y2="116" stroke="currentColor" stroke-width="1.6"/>
  <text x="390" y="71" font-size="8.5" fill="currentColor" font-weight="600">address bus</text>
  <text x="390" y="95" font-size="8.5" fill="currentColor" font-weight="600">data bus</text>
  <text x="390" y="119" font-size="8.5" fill="currentColor" font-weight="600">control bus</text>
  <rect x="60" y="150" width="108" height="42" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/>
  <text x="114" y="176" text-anchor="middle" font-size="9" fill="currentColor">RAM</text>
  <rect x="240" y="150" width="124" height="42" rx="5" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.2"/>
  <text x="302" y="176" text-anchor="middle" font-size="9" fill="currentColor">I/O devices</text>
  <g stroke="currentColor" stroke-width="1" stroke-opacity="0.6">
    <line x1="72" y1="54" x2="72" y2="68"/><line x1="104" y1="54" x2="104" y2="92"/><line x1="136" y1="54" x2="136" y2="116"/>
    <line x1="88" y1="150" x2="88" y2="68"/><line x1="114" y1="150" x2="114" y2="92"/><line x1="140" y1="150" x2="140" y2="116"/>
    <line x1="266" y1="150" x2="266" y2="68"/><line x1="292" y1="150" x2="292" y2="92"/><line x1="318" y1="150" x2="318" y2="116"/>
  </g>
  <g fill="currentColor"><circle cx="72" cy="68" r="2"/><circle cx="104" cy="92" r="2"/><circle cx="136" cy="116" r="2"/><circle cx="88" cy="68" r="2"/><circle cx="114" cy="92" r="2"/><circle cx="140" cy="116" r="2"/><circle cx="266" cy="68" r="2"/><circle cx="292" cy="92" r="2"/><circle cx="318" cy="116" r="2"/></g>
  <text x="205" y="208" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">where a value goes · what the value is · when to move it</text>
</svg>
<figcaption>A classic system bus is really three buses working together: the address bus names a location, the data bus carries the value, and the control bus says read or write and when. CPU, RAM, and I/O all hang off the same shared lines — which is why heavy traffic makes the bus a bottleneck, and why modern machines split it into point-to-point links.</figcaption>
</figure>

## Overview

Classically a bus has three parts: a *data bus* (the values being moved), an *address bus* (which memory location or device they go to), and a *control bus* (timing and read/write signals). The bus *width* — how many bits cross at once — and its clock set the bandwidth. This shared design comes straight from the [von Neumann architecture](/reference/von-neumann-architecture/), where one path connects the processor to a common memory and devices.

## Where it fits

Because the components on the [motherboard](/reference/motherboard/) cannot talk directly, the bus is the meeting point, and a shared bus can become a bottleneck when many devices compete for it. Modern systems mostly replace one wide shared bus with many fast *point-to-point* links — [PCIe](/reference/pci-express/) being the dominant example — but the concept is unchanged: a defined interconnect that lets the parts of a computer exchange data in step.

## Sources

[^wiki]: [Bus (computing)](https://en.wikipedia.org/wiki/Bus_(computing)) — Wikipedia, on data, address, and control buses.
