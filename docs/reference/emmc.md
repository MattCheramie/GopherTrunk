---
slug: emmc
title: eMMC
entry_type: hardware
category: hw-storage
description: eMMC is flash storage with a controller in a single BGA chip soldered onto a board, giving embedded devices and cheaper computers built-in storage without a removable card.
keywords: eMMC, embedded MultiMediaCard, embedded flash, soldered storage, NAND, MMC, JEDEC, BGA, single-board computer
aka: [embedded MultiMediaCard]
autolink: true
infobox:
  - { label: Type, value: Embedded flash storage }
  - { label: Medium, value: NAND flash + controller }
  - { label: Mounting, value: Soldered to board (BGA) }
  - { label: Standard, value: JEDEC eMMC }
  - { label: Common use, value: Phones, tablets, SBCs }
see_also: [flash-memory, sd-card, solid-state-drive, jedec, nvme, data-storage, single-board-computer]
cite_urls:
  - https://en.wikipedia.org/wiki/MultiMediaCard#eMMC
---

**eMMC (embedded MultiMediaCard)** is a single package that combines [flash memory](/reference/flash-memory/) and its controller, soldered directly onto a device's board as built-in storage.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Cross-section of an eMMC package. A single chip package holds a stack of NAND flash dies above a small controller die that handles wear leveling; the package sits on a row of solder balls that fix it permanently to the circuit board below." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="120" y="24" width="220" height="66" rx="4" fill-opacity="0.06" fill="currentColor"/>
    <rect x="140" y="34" width="180" height="16" rx="2" fill-opacity="0.14" fill="currentColor"/>
    <rect x="146" y="40" width="180" height="16" rx="2" fill-opacity="0.14" fill="currentColor"/>
    <rect x="152" y="46" width="180" height="16" rx="2" fill-opacity="0.14" fill="currentColor"/>
    <rect x="150" y="68" width="120" height="14" rx="2" fill-opacity="0.22" fill="currentColor"/>
    <line x1="90" y1="112" x2="440" y2="112" stroke-width="1.6"/>
  </g>
  <g fill="currentColor" stroke="currentColor" stroke-width="0.6">
    <circle cx="150" cy="100" r="4" fill-opacity="0.5"/>
    <circle cx="180" cy="100" r="4" fill-opacity="0.5"/>
    <circle cx="210" cy="100" r="4" fill-opacity="0.5"/>
    <circle cx="240" cy="100" r="4" fill-opacity="0.5"/>
    <circle cx="270" cy="100" r="4" fill-opacity="0.5"/>
    <circle cx="300" cy="100" r="4" fill-opacity="0.5"/>
  </g>
  <g fill="currentColor" stroke="none">
    <text x="352" y="44" font-size="8.5">NAND flash</text>
    <text x="352" y="54" font-size="8.5">dies (stacked)</text>
    <text x="210" y="78" font-size="8.5" text-anchor="middle" font-weight="600">controller</text>
    <text x="316" y="103" font-size="8" text-anchor="start">solder balls (BGA)</text>
    <text x="90" y="126" font-size="8.5">circuit board — permanently mounted</text>
  </g>
</svg>
<figcaption>An eMMC package stacks NAND flash dies over a controller that hides wear leveling, and a grid of solder balls fixes the whole thing to the board — one chip, no connector, no removable card.</figcaption>
</figure>

## Overview

Like an [SD card](/reference/sd-card/), eMMC bundles NAND flash with a controller that handles wear leveling, bad-block management, and error correction, then presents a plain block device to the host. The difference is packaging: eMMC is a ball-grid-array (BGA) chip reflow-soldered onto the board rather than a card in a slot. The interface and command set are standardised by [JEDEC](/reference/jedec/), evolving from the older MMC standard into successive eMMC revisions.

Because it is fixed in place, eMMC is mechanically reliable and compact, with no contacts to corrode or work loose. Capacities are modest — typically a handful to tens of gigabytes — and sustained throughput sits well below a modern [NVMe](/reference/nvme/) [SSD](/reference/solid-state-drive/), roughly on par with a good SD card. It is the built-in storage in most phones, tablets, and the cheaper tiers of single-board computers.

## How it compares

eMMC occupies the middle of the flash storage range — sturdier and usually faster than a card, far cheaper and slower than an NVMe drive:

| Trait | SD card | eMMC | NVMe SSD |
|-------|---------|------|----------|
| Mounting | Removable slot | Soldered (BGA) | M.2 / slot |
| Interface | SD bus | eMMC (JEDEC) | PCIe |
| Typical speed | ~10–100 MB/s | ~100–300 MB/s | 1000+ MB/s |
| Capacity | Up to ~1 TB | ~4–256 GB | Up to multi-TB |
| Removable | Yes | No | Usually |

## Where it fits

eMMC is the practical default for embedded gear that needs dependable built-in storage without the cost or size of a discrete drive. Some [single-board computers](/reference/single-board-computer/) offer an eMMC module as a sturdier alternative to booting from microSD. For a GopherTrunk node that runs unattended near an antenna, soldered or socketed eMMC avoids the wear and connection issues that plague constantly written SD cards, while staying cheaper and lower-power than fitting a full SSD.

## Sources

[^wiki]: [MultiMediaCard — eMMC](https://en.wikipedia.org/wiki/MultiMediaCard#eMMC) — Wikipedia, on embedded MMC flash storage and its JEDEC standardisation.
