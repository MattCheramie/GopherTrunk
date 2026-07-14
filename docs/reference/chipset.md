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

<figure class="figure" markdown="0">
<svg viewBox="0 0 400 232" role="img" aria-label="The CPU, now holding the memory and graphics controllers that were once the northbridge, links directly to RAM and the x16 GPU, and connects down over a DMI link to the chipset or Platform Controller Hub, which fans out to USB, SATA, and PCIe peripherals." xmlns="http://www.w3.org/2000/svg">
  <rect x="60" y="24" width="150" height="54" rx="6" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.3"/>
  <text x="135" y="46" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">CPU</text>
  <text x="135" y="62" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.85">on-die memory + graphics</text>
  <g fill="currentColor" font-size="8" text-anchor="middle">
    <rect x="248" y="24" width="86" height="24" rx="4" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1"/><text x="291" y="40">RAM</text>
    <rect x="248" y="54" width="86" height="24" rx="4" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1"/><text x="291" y="70">GPU · PCIe x16</text>
  </g>
  <g stroke="currentColor" stroke-width="1.1" stroke-opacity="0.6" fill="none">
    <line x1="210" y1="40" x2="248" y2="36"/><line x1="210" y1="60" x2="248" y2="66"/>
  </g>
  <text x="291" y="94" text-anchor="middle" font-size="7" fill="currentColor" fill-opacity="0.5">(was the northbridge)</text>
  <line x1="135" y1="78" x2="135" y2="112" stroke="currentColor" stroke-width="1.4" fill="none" marker-end="url(#cs_ar)"/>
  <text x="152" y="98" font-size="7.5" fill="currentColor" fill-opacity="0.85">DMI</text>
  <rect x="60" y="112" width="150" height="46" rx="6" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.3"/>
  <text x="135" y="132" text-anchor="middle" font-size="9.5" fill="currentColor" font-weight="600">chipset · PCH</text>
  <text x="135" y="147" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.85">I/O hub</text>
  <g fill="currentColor" font-size="8" text-anchor="middle">
    <rect x="24" y="188" width="84" height="30" rx="4" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1"/><text x="66" y="207">USB</text>
    <rect x="118" y="188" width="84" height="30" rx="4" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1"/><text x="160" y="207">SATA</text>
    <rect x="212" y="188" width="110" height="30" rx="4" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1"/><text x="267" y="207">PCIe lanes</text>
  </g>
  <g stroke="currentColor" stroke-width="1.1" stroke-opacity="0.6" fill="none">
    <line x1="120" y1="158" x2="66" y2="188"/><line x1="135" y1="158" x2="160" y2="188"/><line x1="150" y1="158" x2="267" y2="188"/>
  </g>
  <defs><marker id="cs_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Modern PCs have one chipset chip — Intel's Platform Controller Hub — bridging the CPU to slower I/O: USB, SATA, and spare PCIe lanes. The old northbridge that once handled memory and graphics has moved onto the CPU die, so the RAM and the x16 GPU link now attach straight to the processor.</figcaption>
</figure>

## Overview

Historically a PC chipset split into a *northbridge* (fast links to memory and graphics) and a *southbridge* (slower I/O like storage, USB, and audio). As CPUs absorbed the memory controller and graphics link onto the die, the northbridge's job moved into the processor, leaving a single I/O hub — Intel calls it the Platform Controller Hub (PCH). Today the chipset mainly provides the extra [PCIe](/reference/pci-express/) lanes, [USB](/reference/usb/) ports, SATA connectors, and other I/O that a platform exposes.

## Where it fits

The chipset, paired with a given CPU socket, defines much of what a board can do: how many devices it supports, which features are enabled, and how peripherals reach the processor over the [system bus](/reference/system-bus/). It works alongside the board's [BIOS/UEFI](/reference/bios-uefi/) firmware at startup. On a single-board computer the chipset's functions are folded into the [system-on-a-chip](/reference/system-on-a-chip/), so a GopherTrunk capture node has no separate chipset — but the connectivity it provides is the same idea.

## Sources

[^wiki]: [Chipset](https://en.wikipedia.org/wiki/Chipset) — Wikipedia, on motherboard chipsets, northbridge/southbridge, and the modern PCH.
