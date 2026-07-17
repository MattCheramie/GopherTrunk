---
slug: banana-pi
title: Banana Pi
entry_type: hardware
category: hw-sbc
description: Banana Pi is a brand of single-board computers in the Raspberry Pi mould, notable early on for onboard SATA and gigabit Ethernet, and later for a wide range of ARM and RISC-V boards.
keywords: Banana Pi, SATA SBC, Allwinner, gigabit Ethernet, Raspberry Pi alternative, RISC-V board, ARM single-board computer, onboard storage
infobox:
  - { label: Type, value: Single-board computer }
  - { label: CPU, value: ARM (Allwinner / others), some RISC-V }
  - { label: Runs, value: Linux / Android }
  - { label: Noted for, value: Onboard SATA, gigabit Ethernet }
  - { label: Typical price, value: ~$25 – $100 }
see_also: [raspberry-pi, single-board-computer, orange-pi, odroid, rock-pi, risc-v]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Banana_Pi
---

**Banana Pi** is a brand of [single-board computers](/reference/single-board-computer/) in the [Raspberry Pi](/reference/raspberry-pi/) mould — an early Pi alternative that stood out for onboard SATA and gigabit Ethernet.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="Top view of a Banana Pi board. Along one edge sit a gigabit Ethernet jack and USB ports; a SATA data connector and its power header sit on an adjacent edge; a 40-pin GPIO header runs along the top; the ARM system-on-chip sits in the centre. The onboard SATA and gigabit Ethernet are the features that distinguished early Banana Pi boards." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="40" y="24" width="380" height="120" rx="6" fill-opacity="0.05" fill="currentColor"/>
    <rect x="196" y="66" width="64" height="46" rx="3" fill-opacity="0.14" fill="currentColor"/>
    <g stroke-width="1.1">
      <rect x="60" y="120" width="220" height="12" rx="2"/>
      <line x1="66" y1="120" x2="66" y2="132"/><line x1="86" y1="120" x2="86" y2="132"/>
      <line x1="106" y1="120" x2="106" y2="132"/><line x1="126" y1="120" x2="126" y2="132"/>
      <line x1="146" y1="120" x2="146" y2="132"/><line x1="166" y1="120" x2="166" y2="132"/>
      <line x1="186" y1="120" x2="186" y2="132"/><line x1="206" y1="120" x2="206" y2="132"/>
      <line x1="226" y1="120" x2="226" y2="132"/><line x1="246" y1="120" x2="246" y2="132"/>
    </g>
    <rect x="384" y="34" width="30" height="24" rx="2" fill-opacity="0.12" fill="currentColor"/>
    <rect x="384" y="66" width="30" height="18" rx="2" fill-opacity="0.12" fill="currentColor"/>
    <rect x="384" y="90" width="30" height="18" rx="2" fill-opacity="0.12" fill="currentColor"/>
    <rect x="46" y="34" width="14" height="34" rx="2" fill-opacity="0.18" fill="currentColor"/>
    <rect x="46" y="76" width="10" height="20" rx="2" fill-opacity="0.12" fill="currentColor"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8.5">
    <text x="228" y="92" text-anchor="middle" font-size="9" font-weight="600">ARM SoC</text>
    <text x="170" y="146" text-anchor="middle">40-pin GPIO</text>
    <text x="399" y="30" text-anchor="middle">GbE</text>
    <text x="399" y="79" text-anchor="middle">USB</text>
    <text x="399" y="103" text-anchor="middle">USB</text>
    <text x="53" y="30" text-anchor="middle" font-weight="600">SATA</text>
    <text x="51" y="107" text-anchor="middle">pwr</text>
  </g>
</svg>
<figcaption>The early Banana Pi's calling cards were a real SATA data port (plus its power header) and a gigabit Ethernet jack on a Pi-sized board — enough to build a small file server or storage-backed node without a USB disk adapter.</figcaption>
</figure>

## Overview

The first Banana Pi boards matched the Pi's size and 40-pin [GPIO](/reference/gpio/) header but added a SATA port and gigabit networking, making them attractive for small file servers and network-attached storage. Where a stock Pi of the era had to hang a disk off USB, a Banana Pi could drive a 2.5-inch drive natively, and its faster Ethernet moved data off the board without bottlenecking.

The range has since grown well beyond that first board to cover many ARM system-on-chips and, more recently, [RISC-V](/reference/risc-v/) boards — an unusually wide catalogue for a single brand. As with other Pi clones, though, Linux support varies model to model and is generally less polished than the Raspberry Pi's, so a given board's usefulness depends heavily on how mature its vendor image and kernel are.

## The family

Banana Pi is best understood as one of several Pi-alternative brands, each with a slightly different pitch:

| Brand | Maker | Typical SoC | Distinctive trait |
|-------|-------|-------------|-------------------|
| Banana Pi | SinoVoip | Allwinner / others, some RISC-V | Onboard SATA + gigabit Ethernet |
| Orange Pi | Shenzhen Xunlong | Allwinner / Rockchip | Low price, many variants |
| Rock Pi | Radxa | Rockchip (RK3588) | NVMe, fast I/O |
| ODROID | Hardkernel | Amlogic | Performance per board |

## Where it fits

Banana Pi sits among the Pi alternatives — [Orange Pi](/reference/orange-pi/), [ODROID](/reference/odroid/), [Rock Pi](/reference/rock-pi/) — and is worth a look when you want a disk and wired networking on one cheap board. For a GopherTrunk setup that both captures and serves decoded data, the built-in SATA can host the storage locally without a USB adapter hanging off the board, and gigabit Ethernet keeps recorded IQ or a growing call archive moving to a collector without saturating the link.

## Sources

[^wiki]: [Banana Pi](https://en.wikipedia.org/wiki/Banana_Pi) — Wikipedia, on the Banana Pi single-board computers.
