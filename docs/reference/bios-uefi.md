---
slug: bios-uefi
title: BIOS & UEFI
entry_type: concept
category: hw-foundations
description: BIOS and its modern successor UEFI are the firmware that initializes a computer's hardware at power-on, runs the power-on self-test, and hands control to the operating system's bootloader.
keywords: BIOS, UEFI, firmware, boot, POST, power-on self-test, bootloader, secure boot, GPT, motherboard firmware
aka: [BIOS, UEFI]
autolink: true
infobox:
  - { label: Type, value: Platform firmware }
  - { label: Runs, value: At power-on, before OS }
  - { label: Does, value: Init hardware, POST, start boot }
  - { label: Stored in, value: Motherboard flash chip }
  - { label: Successor, value: UEFI replaces legacy BIOS }
see_also: [motherboard, operating-system, firmware, bootloader, chipset, data-storage]
cite_urls:
  - https://en.wikipedia.org/wiki/BIOS
  - https://en.wikipedia.org/wiki/UEFI
---

**BIOS** (Basic Input/Output System) and its modern successor **UEFI** (Unified Extensible Firmware Interface) are the [firmware](/reference/firmware/) that brings a computer's hardware up at power-on and hands control to the operating system.[^bios][^uefi]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="The boot sequence from left to right: power applied, the CPU runs firmware from the motherboard flash chip, the firmware performs a power-on self-test, initializes the chipset and memory, finds a bootable storage device, and launches its bootloader, which loads the operating system." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <rect x="8" y="46" width="66" height="40" rx="4"/>
    <rect x="90" y="46" width="72" height="40" rx="4" fill="currentColor" fill-opacity="0.12"/>
    <rect x="178" y="46" width="72" height="40" rx="4" fill="currentColor" fill-opacity="0.12"/>
    <rect x="266" y="46" width="72" height="40" rx="4"/>
    <rect x="354" y="46" width="98" height="40" rx="4"/>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="currentColor">
    <path d="M74 66 h12 m-4 -3 l4 3 l-4 3" fill="none"/>
    <path d="M162 66 h12 m-4 -3 l4 3 l-4 3" fill="none"/>
    <path d="M250 66 h12 m-4 -3 l4 3 l-4 3" fill="none"/>
    <path d="M338 66 h12 m-4 -3 l4 3 l-4 3" fill="none"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="41" y="63" font-size="8.5">Power</text>
    <text x="41" y="74" font-size="8.5">on</text>
    <text x="126" y="60" font-size="8.5" font-weight="600">Firmware</text>
    <text x="126" y="71" font-size="7.5" fill-opacity="0.85">POST</text>
    <text x="126" y="80" font-size="7.5" fill-opacity="0.85">self-test</text>
    <text x="214" y="60" font-size="8.5" font-weight="600">Init</text>
    <text x="214" y="71" font-size="7.5" fill-opacity="0.85">chipset,</text>
    <text x="214" y="80" font-size="7.5" fill-opacity="0.85">RAM, devices</text>
    <text x="302" y="63" font-size="8.5">Find boot</text>
    <text x="302" y="74" font-size="8.5">device</text>
    <text x="403" y="60" font-size="8.5">Bootloader</text>
    <text x="403" y="71" font-size="7.5" fill-opacity="0.85">loads the OS</text>
    <text x="230" y="118" font-size="8" fill-opacity="0.9">legacy BIOS &#8594; UEFI: large disks (GPT), fast boot, Secure Boot</text>
  </g>
</svg>
<figcaption>From the instant power is applied, the firmware self-tests the hardware, brings up the chipset and memory, locates a bootable drive, and starts its bootloader — the same job whether the firmware is legacy BIOS or the more capable UEFI.</figcaption>
</figure>

## Overview

Stored in a flash chip on the [motherboard](/reference/motherboard/), this firmware is the very first code a [CPU](/reference/central-processing-unit/) runs when power is applied. It performs the *power-on self-test* (POST), a quick sanity check of memory and core devices, then initializes the [chipset](/reference/chipset/), RAM, and basic peripherals so the machine is in a known, working state.

Once the platform is up, the firmware consults its configured boot order, locates a bootable [storage](/reference/data-storage/) device, and starts that device's *[bootloader](/reference/bootloader/)*, which in turn loads the [operating system](/reference/operating-system/). The firmware also exposes a *setup screen* for changing boot order, clocks, and virtualization toggles, and it can itself be updated — a process called *flashing*.

## BIOS versus UEFI

Legacy BIOS dates to the original IBM PC of the early 1980s and carried real limitations by modern standards. UEFI replaced it with a cleaner, more capable design:

| Aspect | Legacy BIOS | UEFI |
|--------|-------------|------|
| Era | Early 1980s | 2000s onward |
| Disk scheme | MBR (≤ 2 TB) | GPT (very large disks) |
| Mode | 16-bit real mode | 32-/64-bit |
| Boot integrity | None | Secure Boot |
| Interface | Text menus | Drivers, shell, richer UI |

Most systems shipped in the last decade use UEFI, often with a *compatibility support module* (CSM) that emulates legacy BIOS for older operating systems.

## Where it fits

BIOS/UEFI is the indispensable layer between bare hardware and software — without it a machine has no idea how to start. On small SDR capture boards the role is filled by an equivalent [bootloader](/reference/bootloader/) or ROM baked into the platform rather than a PC-style setup screen, but the job is identical: wake the hardware, then start the OS that runs the GopherTrunk daemon.

## Sources

[^bios]: [BIOS](https://en.wikipedia.org/wiki/BIOS) — Wikipedia, on legacy PC firmware, POST, and the boot handoff.
[^uefi]: [UEFI](https://en.wikipedia.org/wiki/UEFI) — Wikipedia, on the modern UEFI firmware interface, GPT, and Secure Boot.
