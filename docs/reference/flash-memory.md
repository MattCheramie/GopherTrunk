---
slug: flash-memory
title: Flash memory
entry_type: hardware
category: hw-storage
description: Flash memory is non-volatile solid-state storage that holds data without power by trapping charge in transistor cells, erasable and rewritable in blocks.
keywords: flash memory, NAND, NOR, floating gate, non-volatile, erase block, wear leveling, SLC, MLC, TLC
infobox:
  - { label: Type, value: Non-volatile solid-state memory }
  - { label: Cell, value: Floating-gate transistor }
  - { label: Variants, value: NAND, NOR }
  - { label: Erased in, value: Blocks }
  - { label: Wears out, value: Limited erase cycles }
see_also: [solid-state-drive, sd-card, emmc, read-only-memory, volatile-memory, nvme]
cite_urls:
  - https://en.wikipedia.org/wiki/Flash_memory
---

**Flash memory** is non-volatile solid-state storage that retains data without power by trapping electric charge in floating-gate transistor cells.[^wiki]

## Overview

Each cell holds one or more bits as the presence or absence of trapped charge; reading senses that charge, while writing and erasing move it through the insulating layer. Flash comes in two main families: **NOR**, which allows fast random reads and is used to store firmware, and **NAND**, which is denser and cheaper per bit and underlies most mass storage. Cells can be erased only a block at a time and endure a finite number of erase cycles, so controllers use *wear leveling* to spread writes evenly. Packing more bits per cell (SLC, MLC, TLC, QLC) raises capacity at the cost of endurance and speed.

## Where it fits

Flash is the medium behind nearly all modern removable and embedded storage: the [SSD](/reference/solid-state-drive/), the [SD card](/reference/sd-card/), [eMMC](/reference/emmc/) chips on single-board computers, and USB sticks. Unlike [volatile](/reference/volatile-memory/) [RAM](/reference/random-access-memory/), it keeps its contents when powered off, and unlike masked [ROM](/reference/read-only-memory/) it can be rewritten in the field — which is how firmware updates work. A GopherTrunk capture node on a [Raspberry Pi](/reference/raspberry-pi/) typically boots and logs to flash-based storage.

## Sources

[^wiki]: [Flash memory](https://en.wikipedia.org/wiki/Flash_memory) — Wikipedia, on NAND/NOR flash, floating-gate cells, and wear leveling.
