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

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 234" role="img" aria-label="Top: one flash cell in cross-section — a control gate over an isolated floating gate that traps charge, above a channel with source and drain. The trapped charge is the stored bit. Bottom: a grid of cells, with a dashed outline marking one erase block, because flash is written a page at a time but erased a whole block at a time." xmlns="http://www.w3.org/2000/svg">
  <text x="140" y="18" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">One flash cell</text>
  <g fill="currentColor" font-size="8.5">
    <rect x="160" y="28" width="100" height="18" rx="2" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.1"/>
    <text x="210" y="40" text-anchor="middle" font-size="8">control gate</text>
    <rect x="160" y="52" width="100" height="16" rx="2" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1.1" stroke-dasharray="3 2"/>
    <circle cx="185" cy="60" r="1.7" fill="currentColor"/><circle cx="210" cy="60" r="1.7" fill="currentColor"/><circle cx="235" cy="60" r="1.7" fill="currentColor"/>
    <rect x="140" y="76" width="140" height="24" fill="currentColor" fill-opacity="0.06" stroke="currentColor" stroke-width="1.1"/>
    <rect x="140" y="76" width="26" height="24" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1"/>
    <rect x="254" y="76" width="26" height="24" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1"/>
    <text x="153" y="91" text-anchor="middle" font-size="7.5">S</text>
    <text x="267" y="91" text-anchor="middle" font-size="7.5">D</text>
    <text x="210" y="91" text-anchor="middle" font-size="7">channel</text>
    <text x="288" y="60" text-anchor="start" font-size="8">floating gate</text>
    <text x="288" y="71" text-anchor="start" font-size="7.5" fill-opacity="0.85">traps charge = the bit</text>
  </g>
  <line x1="280" y1="60" x2="286" y2="60" stroke="currentColor" stroke-width="0.9" stroke-opacity="0.6"/>
  <text x="220" y="130" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">Cells erase a block at a time</text>
  <g fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="0.8">
    <rect x="90" y="150" width="26" height="14" rx="2"/><rect x="118" y="150" width="26" height="14" rx="2"/><rect x="146" y="150" width="26" height="14" rx="2"/><rect x="174" y="150" width="26" height="14" rx="2"/><rect x="202" y="150" width="26" height="14" rx="2"/><rect x="230" y="150" width="26" height="14" rx="2"/><rect x="258" y="150" width="26" height="14" rx="2"/><rect x="286" y="150" width="26" height="14" rx="2"/><rect x="314" y="150" width="26" height="14" rx="2"/><rect x="342" y="150" width="26" height="14" rx="2"/>
    <rect x="90" y="168" width="26" height="14" rx="2"/><rect x="118" y="168" width="26" height="14" rx="2"/><rect x="146" y="168" width="26" height="14" rx="2"/><rect x="174" y="168" width="26" height="14" rx="2"/><rect x="202" y="168" width="26" height="14" rx="2"/><rect x="230" y="168" width="26" height="14" rx="2"/><rect x="258" y="168" width="26" height="14" rx="2"/><rect x="286" y="168" width="26" height="14" rx="2"/><rect x="314" y="168" width="26" height="14" rx="2"/><rect x="342" y="168" width="26" height="14" rx="2"/>
    <rect x="90" y="186" width="26" height="14" rx="2"/><rect x="118" y="186" width="26" height="14" rx="2"/><rect x="146" y="186" width="26" height="14" rx="2"/><rect x="174" y="186" width="26" height="14" rx="2"/><rect x="202" y="186" width="26" height="14" rx="2"/><rect x="230" y="186" width="26" height="14" rx="2"/><rect x="258" y="186" width="26" height="14" rx="2"/><rect x="286" y="186" width="26" height="14" rx="2"/><rect x="314" y="186" width="26" height="14" rx="2"/><rect x="342" y="186" width="26" height="14" rx="2"/>
  </g>
  <rect x="86" y="146" width="146" height="58" rx="3" fill="none" stroke="currentColor" stroke-width="1.3" stroke-dasharray="4 3"/>
  <text x="159" y="220" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.9">one erase block</text>
  <text x="312" y="220" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.85">wear leveling spreads writes</text>
</svg>
<figcaption>Each cell stores a bit (or several) as charge trapped on an isolated floating gate — it stays put with the power off. Cells can be written a page at a time but only erased a whole block at a time, and each block wears out after many cycles, so controllers spread writes with wear leveling.</figcaption>
</figure>

## Overview

Each cell holds one or more bits as the presence or absence of trapped charge; reading senses that charge, while writing and erasing move it through the insulating layer. Flash comes in two main families: **NOR**, which allows fast random reads and is used to store firmware, and **NAND**, which is denser and cheaper per bit and underlies most mass storage. Cells can be erased only a block at a time and endure a finite number of erase cycles, so controllers use *wear leveling* to spread writes evenly. Packing more bits per cell (SLC, MLC, TLC, QLC) raises capacity at the cost of endurance and speed.

## Where it fits

Flash is the medium behind nearly all modern removable and embedded storage: the [SSD](/reference/solid-state-drive/), the [SD card](/reference/sd-card/), [eMMC](/reference/emmc/) chips on single-board computers, and USB sticks. Unlike [volatile](/reference/volatile-memory/) [RAM](/reference/random-access-memory/), it keeps its contents when powered off, and unlike masked [ROM](/reference/read-only-memory/) it can be rewritten in the field — which is how firmware updates work. A GopherTrunk capture node on a [Raspberry Pi](/reference/raspberry-pi/) typically boots and logs to flash-based storage.

## Sources

[^wiki]: [Flash memory](https://en.wikipedia.org/wiki/Flash_memory) — Wikipedia, on NAND/NOR flash, floating-gate cells, and wear leveling.
