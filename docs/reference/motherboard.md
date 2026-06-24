---
slug: motherboard
title: Motherboard
entry_type: hardware
category: hw-foundations
description: A motherboard is the main printed circuit board of a computer, holding the CPU socket, memory slots, chipset, and expansion buses that connect every component together.
keywords: motherboard, mainboard, logic board, PCB, CPU socket, chipset, expansion slots, form factor
aka: [Mainboard, Logic board]
infobox:
  - { label: Type, value: Main circuit board }
  - { label: Holds, value: CPU, RAM, chipset, buses }
  - { label: Key buses, value: PCIe, USB, SATA }
  - { label: Form factors, value: ATX, microATX, Mini-ITX }
see_also: [central-processing-unit, chipset, pci-express, system-bus, power-supply-unit, integrated-circuit]
cite_urls:
  - https://en.wikipedia.org/wiki/Motherboard
---

A **motherboard** (or mainboard) is the main printed circuit board of a computer — the backbone that holds the [CPU](/reference/central-processing-unit/), memory, and [chipset](/reference/chipset/) and wires every other part together.[^wiki]

## Overview

The board provides a socket for the CPU, slots for [RAM](/reference/random-access-memory/), and connectors and [expansion buses](/reference/system-bus/) such as [PCIe](/reference/pci-express/), [USB](/reference/usb/), and SATA. Its [chipset](/reference/chipset/) and traces route signals between the processor, memory, [storage](/reference/data-storage/), and [I/O](/reference/input-output/). Firmware ([BIOS/UEFI](/reference/bios-uefi/)) lives in a chip on the board and brings the system up at power-on. Physical size and mounting follow a *form factor* — common ones are ATX, microATX, and Mini-ITX.

## Where it fits

Everything plugs into the motherboard, so it sets what a machine can hold: how many memory channels, how many PCIe lanes, which CPU generation. Power arrives from the [PSU](/reference/power-supply-unit/) through dedicated connectors. A small-form-factor board — like the one inside a [Raspberry Pi](/reference/raspberry-pi/) acting as a GopherTrunk capture node — folds the chipset and I/O onto a single tiny PCB, but the role is the same: be the board that everything else connects to.

## Sources

[^wiki]: [Motherboard](https://en.wikipedia.org/wiki/Motherboard) — Wikipedia, on the main board of a computer and what it carries.
