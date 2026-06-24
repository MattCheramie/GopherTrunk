---
slug: bios-uefi
title: BIOS & UEFI
entry_type: concept
category: hw-foundations
description: BIOS and its modern successor UEFI are the firmware that initializes a computer's hardware at power-on and hands control to the operating system's bootloader.
keywords: BIOS, UEFI, firmware, boot, POST, bootloader, secure boot, motherboard firmware
aka: [BIOS, UEFI]
autolink: true
infobox:
  - { label: Type, value: Platform firmware }
  - { label: Runs, value: At power-on, before OS }
  - { label: Does, value: Init hardware, start boot }
  - { label: Successor, value: UEFI replaces legacy BIOS }
see_also: [motherboard, operating-system, chipset, central-processing-unit, firmware, data-storage]
cite_urls:
  - https://en.wikipedia.org/wiki/BIOS
  - https://en.wikipedia.org/wiki/UEFI
---

**BIOS** (Basic Input/Output System) and its modern successor **UEFI** (Unified Extensible Firmware Interface) are the [firmware](/reference/firmware/) that brings a computer's hardware up at power-on and hands control to the operating system.[^bios][^uefi]

## Overview

Stored in a chip on the [motherboard](/reference/motherboard/), this firmware is the first code a [CPU](/reference/central-processing-unit/) runs when power is applied. It performs the power-on self-test (POST), initializes the [chipset](/reference/chipset/), memory, and basic devices, then locates a bootable [storage](/reference/data-storage/) device and starts its *bootloader*, which in turn loads the [operating system](/reference/operating-system/). Legacy BIOS dates to the early PC; UEFI replaced it with a more capable design supporting large disks, faster boot, and features like Secure Boot.

## Where it fits

BIOS/UEFI is the layer between bare hardware and software — without it the machine would not know how to start. It is configurable (boot order, clocks, virtualization toggles) through a setup screen, and it can be updated as *flashing* firmware. On small SDR capture boards the role is filled by an equivalent bootloader baked into the platform; conceptually the job is the same: wake the hardware, then start the OS that runs GopherTrunk.

## Sources

[^bios]: [BIOS](https://en.wikipedia.org/wiki/BIOS) — Wikipedia, on legacy PC firmware and POST.
[^uefi]: [UEFI](https://en.wikipedia.org/wiki/UEFI) — Wikipedia, on the modern UEFI firmware interface.
