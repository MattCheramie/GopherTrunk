---
slug: arm-architecture
title: ARM architecture
entry_type: concept
category: hw-foundations
description: ARM is a family of RISC instruction set architectures licensed by Arm Holdings, prized for power efficiency and dominant in phones, tablets, embedded devices, and increasingly laptops and servers.
keywords: ARM architecture, RISC, Arm Holdings, AArch64, ARM64, low power, mobile, embedded, Apple Silicon, SoC
aka: [ARM, AArch64]
autolink: true
infobox:
  - { label: Type, value: ISA (RISC) }
  - { label: Licensed by, value: Arm Holdings }
  - { label: 64-bit, value: "AArch64 (ARMv8, 2011)" }
  - { label: Known for, value: Power efficiency }
  - { label: Dominates, value: Mobile & embedded }
see_also: [instruction-set-architecture, x86, risc-v, central-processing-unit, raspberry-pi, system-on-a-chip]
cite_urls:
  - https://en.wikipedia.org/wiki/ARM_architecture_family
---

**ARM** is a family of RISC [instruction set architectures](/reference/instruction-set-architecture/) licensed by Arm Holdings, known above all for power efficiency and its reach across phones, embedded devices, and — increasingly — laptops and servers.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="Arm Holdings designs the ARM instruction set and core blueprints, then licenses them to many chip makers who each integrate ARM cores into their own systems-on-a-chip for phones, single-board computers, laptops, and servers." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="160" y="14" width="140" height="34" rx="4" fill="currentColor" fill-opacity="0.12"/>
    <rect x="20" y="88" width="90" height="30" rx="4"/>
    <rect x="130" y="88" width="90" height="30" rx="4"/>
    <rect x="240" y="88" width="90" height="30" rx="4"/>
    <rect x="350" y="88" width="90" height="30" rx="4"/>
    <line x1="230" y1="48" x2="65" y2="88"/>
    <line x1="230" y1="48" x2="175" y2="88"/>
    <line x1="230" y1="48" x2="285" y2="88"/>
    <line x1="230" y1="48" x2="395" y2="88"/>
    <rect x="20" y="140" width="420" height="26" rx="4" stroke-dasharray="3 3" stroke-opacity="0.7"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="230" y="28" font-size="9" font-weight="600">Arm Holdings</text>
    <text x="230" y="40" font-size="8" fill-opacity="0.85">designs ISA + core IP</text>
    <text x="65" y="100" font-size="8">Qualcomm</text>
    <text x="65" y="111" font-size="7.5" fill-opacity="0.8">phone SoC</text>
    <text x="175" y="100" font-size="8">Apple</text>
    <text x="175" y="111" font-size="7.5" fill-opacity="0.8">M-series</text>
    <text x="285" y="100" font-size="8">Broadcom</text>
    <text x="285" y="111" font-size="7.5" fill-opacity="0.8">Raspberry Pi</text>
    <text x="395" y="100" font-size="8">Ampere</text>
    <text x="395" y="111" font-size="7.5" fill-opacity="0.8">server CPU</text>
    <text x="230" y="157" font-size="8">licensees integrate ARM cores into their own silicon</text>
  </g>
</svg>
<figcaption>Arm Holdings sells the architecture and core blueprints but builds no chips itself; dozens of licensees drop ARM cores into their own systems-on-a-chip, which is how one efficient instruction set ended up in nearly every phone as well as single-board computers, Apple laptops, and cloud servers.</figcaption>
</figure>

## Overview

ARM follows the RISC philosophy of a small set of simple, fixed-length instructions, which keeps decode logic modest and power draw low. Arm Holdings designs the architecture and reference core blueprints but does not manufacture chips itself; instead it *licenses* them — either a ready-made core design or an architecture licence to build a custom implementation — to companies that integrate ARM cores into their own [systems-on-a-chip](/reference/system-on-a-chip/).

The current 64-bit variant, introduced with ARMv8 in 2011, is called **AArch64** (or ARM64). It runs alongside the older 32-bit AArch32 mode on most application-class cores. This licensing model, combined with a strong performance-per-watt story, put ARM cores in nearly every smartphone, in vast numbers of embedded and IoT devices, in Apple's desktop and laptop Macs, and in a growing share of the server and cloud market.

## How it compares

The clearest way to place ARM is against the incumbent [x86](/reference/x86/) and the open [RISC-V](/reference/risc-v/):

| Trait | ARM | x86 | RISC-V |
|-------|-----|-----|--------|
| Style | RISC | CISC | RISC |
| Licensing | Paid licence from Arm | Intel/AMD only | Open, royalty-free |
| Instruction length | Fixed (mostly) | Variable | Fixed |
| Traditional stronghold | Mobile & embedded | PC & server | Research & embedded |
| 64-bit name | AArch64 | x86-64 | RV64 |

The fixed-length, load/store design shared by ARM and RISC-V is what makes their decoders simpler than x86's variable-length instructions — one reason the RISC families started in low-power niches, though ARM's reach now extends well up into high-performance computing.

## Where it fits

ARM's efficiency made it the default for battery-powered and embedded computing, the opposite end of the spectrum from where x86 grew up. The [Raspberry Pi](/reference/raspberry-pi/) and most single-board computers use ARM [CPUs](/reference/central-processing-unit/), which is exactly why a low-power GopherTrunk capture node sitting by the antenna usually runs on ARM rather than x86 — Go cross-compiles the same decoder source to an AArch64 target with no code changes.

## Sources

[^wiki]: [ARM architecture family](https://en.wikipedia.org/wiki/ARM_architecture_family) — Wikipedia, on ARM, its RISC design, the AArch64 64-bit mode, and Arm Holdings' licensing model.
