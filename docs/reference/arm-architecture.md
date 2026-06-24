---
slug: arm-architecture
title: ARM architecture
entry_type: concept
category: hw-foundations
description: ARM is a family of RISC instruction set architectures licensed by Arm Holdings, prized for power efficiency and dominant in phones, tablets, and embedded devices.
keywords: ARM architecture, RISC, Arm Holdings, AArch64, low power, mobile, embedded, Apple Silicon
aka: [ARM, AArch64]
autolink: true
infobox:
  - { label: Type, value: ISA (RISC) }
  - { label: Licensed by, value: Arm Holdings }
  - { label: Known for, value: Power efficiency }
  - { label: Dominates, value: Mobile & embedded }
see_also: [instruction-set-architecture, x86, risc-v, central-processing-unit, raspberry-pi, semiconductor]
cite_urls:
  - https://en.wikipedia.org/wiki/ARM_architecture_family
---

**ARM** is a family of RISC [instruction set architectures](/reference/instruction-set-architecture/) licensed by Arm Holdings, known above all for power efficiency.[^wiki]

## Overview

ARM follows the RISC philosophy of a small set of simple, fixed-length instructions, which keeps chips lean and power-thrifty. Arm Holdings designs the architecture and core blueprints but does not make chips itself; instead it *licenses* them to companies that build their own [SoCs](/reference/system-on-a-chip/) around ARM cores. The 64-bit variant is AArch64. This licensing model put ARM cores in nearly every smartphone, in countless embedded devices, and increasingly in laptops and servers.

## Where it fits

ARM's efficiency made it the default for battery-powered and embedded computing, the opposite end of the spectrum from where [x86](/reference/x86/) grew up; [RISC-V](/reference/risc-v/) is a newer open RISC rival. The [Raspberry Pi](/reference/raspberry-pi/) and most single-board computers use ARM [CPUs](/reference/central-processing-unit/), which is exactly why a low-power GopherTrunk capture node by the antenna usually runs on ARM rather than x86.

## Sources

[^wiki]: [ARM architecture family](https://en.wikipedia.org/wiki/ARM_architecture_family) — Wikipedia, on ARM, its RISC design, and licensing model.
