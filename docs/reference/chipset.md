---
slug: chipset
title: Chipset
entry_type: hardware
category: hw-foundations
description: A chipset is the set of chips on a motherboard that manages data flow between the CPU, memory, and peripherals, defining much of a platform's connectivity and features.
keywords: chipset, northbridge, southbridge, PCH, platform, motherboard, I/O hub
infobox:
  - { label: Type, value: Support chips on the board }
  - { label: Manages, value: CPU–memory–I/O data flow }
  - { label: Sets, value: Ports, lanes, features }
  - { label: Modern form, value: Single hub (PCH) }
see_also: [motherboard, central-processing-unit, pci-express, usb, system-bus, bios-uefi]
cite_urls:
  - https://en.wikipedia.org/wiki/Chipset
---

A **chipset** is the set of support chips on a [motherboard](/reference/motherboard/) that manages the flow of data between the [CPU](/reference/central-processing-unit/), memory, and peripherals.[^wiki]

## Overview

Historically a PC chipset split into a *northbridge* (fast links to memory and graphics) and a *southbridge* (slower I/O like storage, USB, and audio). As CPUs absorbed the memory controller and graphics link onto the die, the northbridge's job moved into the processor, leaving a single I/O hub — Intel calls it the Platform Controller Hub (PCH). Today the chipset mainly provides the extra [PCIe](/reference/pci-express/) lanes, [USB](/reference/usb/) ports, SATA connectors, and other I/O that a platform exposes.

## Where it fits

The chipset, paired with a given CPU socket, defines much of what a board can do: how many devices it supports, which features are enabled, and how peripherals reach the processor over the [system bus](/reference/system-bus/). It works alongside the board's [BIOS/UEFI](/reference/bios-uefi/) firmware at startup. On a single-board computer the chipset's functions are folded into the [system-on-a-chip](/reference/system-on-a-chip/), so a GopherTrunk capture node has no separate chipset — but the connectivity it provides is the same idea.

## Sources

[^wiki]: [Chipset](https://en.wikipedia.org/wiki/Chipset) — Wikipedia, on motherboard chipsets, northbridge/southbridge, and the modern PCH.
