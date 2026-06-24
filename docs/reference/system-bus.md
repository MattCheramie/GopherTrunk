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

## Overview

Classically a bus has three parts: a *data bus* (the values being moved), an *address bus* (which memory location or device they go to), and a *control bus* (timing and read/write signals). The bus *width* — how many bits cross at once — and its clock set the bandwidth. This shared design comes straight from the [von Neumann architecture](/reference/von-neumann-architecture/), where one path connects the processor to a common memory and devices.

## Where it fits

Because the components on the [motherboard](/reference/motherboard/) cannot talk directly, the bus is the meeting point, and a shared bus can become a bottleneck when many devices compete for it. Modern systems mostly replace one wide shared bus with many fast *point-to-point* links — [PCIe](/reference/pci-express/) being the dominant example — but the concept is unchanged: a defined interconnect that lets the parts of a computer exchange data in step.

## Sources

[^wiki]: [Bus (computing)](https://en.wikipedia.org/wiki/Bus_(computing)) — Wikipedia, on data, address, and control buses.
