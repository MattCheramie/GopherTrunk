---
slug: banana-pi
title: Banana Pi
entry_type: hardware
category: hw-sbc
description: Banana Pi is a brand of single-board computers in the Raspberry Pi mould, notable early on for onboard SATA and gigabit Ethernet, and later for a wide range of ARM and RISC-V boards.
keywords: Banana Pi, SATA SBC, Allwinner, gigabit Ethernet, Raspberry Pi alternative, RISC-V board, ARM single-board computer
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

**Banana Pi** is a brand of [single-board computers](/reference/single-board-computer/) in the [Raspberry Pi](/reference/raspberry-pi/) mould, an early Pi alternative that stood out for onboard SATA and gigabit Ethernet.[^wiki]

## Overview

The first Banana Pi boards matched the Pi's size and 40-pin [GPIO](/reference/gpio/) header but added a SATA port and gigabit networking, making them attractive for small file servers. The range has since grown to cover many ARM SoCs and, more recently, [RISC-V](/reference/risc-v/) boards. As with other clones, Linux support varies by model and is generally less polished than the Raspberry Pi's.

## Where it fits

Banana Pi sits among the Pi alternatives — [Orange Pi](/reference/orange-pi/), [ODROID](/reference/odroid/), [Rock Pi](/reference/rock-pi/) — and is worth a look when you want a disk and wired networking on one cheap board. For a GopherTrunk setup that both captures and serves decoded data, the built-in SATA can host the storage without a USB adapter hanging off the board.

## Sources

[^wiki]: [Banana Pi](https://en.wikipedia.org/wiki/Banana_Pi) — Wikipedia, on the Banana Pi single-board computers.
