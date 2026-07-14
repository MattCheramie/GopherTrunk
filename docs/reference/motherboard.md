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

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 284" role="img" aria-label="A top-down block diagram of a motherboard PCB showing the CPU socket and RAM slots grouped at the top, the chipset below them, and PCIe expansion slots, USB, and SATA connectors around the edges, all joined by traces on the board." xmlns="http://www.w3.org/2000/svg">
  <rect x="12" y="12" width="416" height="260" rx="10" fill="currentColor" fill-opacity="0.04" stroke="currentColor" stroke-width="1.5"/>
  <text x="26" y="30" font-size="8" fill="currentColor" fill-opacity="0.7">motherboard · top view</text>
  <rect x="44" y="44" width="120" height="96" rx="6" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/>
  <text x="104" y="88" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">CPU</text>
  <text x="104" y="102" text-anchor="middle" font-size="8" fill="currentColor">socket</text>
  <text x="224" y="40" text-anchor="middle" font-size="8" fill="currentColor" font-weight="600">RAM slots</text>
  <g fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1">
    <rect x="196" y="46" width="9" height="94" rx="2"/><rect x="212" y="46" width="9" height="94" rx="2"/><rect x="228" y="46" width="9" height="94" rx="2"/><rect x="244" y="46" width="9" height="94" rx="2"/>
  </g>
  <rect x="64" y="176" width="100" height="60" rx="6" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.3"/>
  <text x="114" y="205" text-anchor="middle" font-size="9" fill="currentColor" font-weight="600">chipset</text>
  <text x="114" y="219" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.85">PCH</text>
  <rect x="196" y="196" width="196" height="14" rx="2" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1"/><text x="294" y="207" text-anchor="middle" font-size="8" fill="currentColor">PCIe x16</text>
  <rect x="196" y="224" width="140" height="14" rx="2" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/><text x="266" y="235" text-anchor="middle" font-size="8" fill="currentColor">PCIe x1</text>
  <g fill="currentColor" font-size="7.5" text-anchor="middle">
    <rect x="196" y="248" width="42" height="14" rx="2" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1"/><text x="217" y="258">USB</text>
    <rect x="244" y="248" width="42" height="14" rx="2" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1"/><text x="265" y="258">USB</text>
  </g>
  <text x="408" y="144" text-anchor="middle" font-size="7.5" fill="currentColor" font-weight="600">SATA</text>
  <g fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1">
    <rect x="392" y="150" width="26" height="12" rx="2"/><rect x="392" y="166" width="26" height="12" rx="2"/><rect x="392" y="182" width="26" height="12" rx="2"/>
  </g>
  <g stroke="currentColor" stroke-width="1" stroke-opacity="0.45" fill="none">
    <line x1="164" y1="70" x2="196" y2="70"/><line x1="164" y1="96" x2="196" y2="96"/>
    <line x1="104" y1="140" x2="104" y2="176"/>
    <line x1="164" y1="200" x2="196" y2="203"/><line x1="164" y1="216" x2="196" y2="231"/>
    <line x1="164" y1="188" x2="392" y2="160"/>
    <line x1="140" y1="236" x2="217" y2="248"/>
  </g>
</svg>
<figcaption>Seen from above, the motherboard is the backbone everything plugs into: the CPU socket and RAM slots sit close together up top, the chipset routes slower I/O, and the PCIe, USB, and SATA connectors fan out to expansion cards and drives. Its copper traces wire all of them together on one PCB.</figcaption>
</figure>

## Overview

The board provides a socket for the CPU, slots for [RAM](/reference/random-access-memory/), and connectors and [expansion buses](/reference/system-bus/) such as [PCIe](/reference/pci-express/), [USB](/reference/usb/), and SATA. Its [chipset](/reference/chipset/) and traces route signals between the processor, memory, [storage](/reference/data-storage/), and [I/O](/reference/input-output/). Firmware ([BIOS/UEFI](/reference/bios-uefi/)) lives in a chip on the board and brings the system up at power-on. Physical size and mounting follow a *form factor* — common ones are ATX, microATX, and Mini-ITX.

## Where it fits

Everything plugs into the motherboard, so it sets what a machine can hold: how many memory channels, how many PCIe lanes, which CPU generation. Power arrives from the [PSU](/reference/power-supply-unit/) through dedicated connectors. A small-form-factor board — like the one inside a [Raspberry Pi](/reference/raspberry-pi/) acting as a GopherTrunk capture node — folds the chipset and I/O onto a single tiny PCB, but the role is the same: be the board that everything else connects to.

## Sources

[^wiki]: [Motherboard](https://en.wikipedia.org/wiki/Motherboard) — Wikipedia, on the main board of a computer and what it carries.
