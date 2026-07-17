---
slug: solid-state-drive
title: Solid-state drive (SSD)
entry_type: hardware
category: hw-storage
description: A solid-state drive stores data in non-volatile flash memory with no moving parts, giving much faster access and better durability than a spinning hard disk.
keywords: solid-state drive, SSD, flash storage, NAND, SATA SSD, NVMe SSD, wear leveling, TBW, random access
aka: [SSD]
infobox:
  - { label: Type, value: Flash non-volatile storage }
  - { label: Medium, value: NAND flash memory }
  - { label: Interfaces, value: SATA, NVMe (PCIe) }
  - { label: Moving parts, value: None }
  - { label: Strength, value: Fast random access }
see_also: [hard-disk-drive, flash-memory, nvme, data-storage, memory-hierarchy, file-system, emmc]
cite_urls:
  - https://en.wikipedia.org/wiki/Solid-state_drive
---

A **solid-state drive (SSD)** is a storage device that keeps data in non-volatile [flash memory](/reference/flash-memory/) with no moving parts, delivering far faster access than a mechanical disk.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="Side-by-side comparison of an SSD and a hard disk drive. The SSD is drawn as a controller chip wired to a bank of NAND flash chips with no moving parts, so any block is reached electronically. The hard disk is drawn as a spinning platter with a seek arm that must physically swing to the right track, making random access much slower." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <rect x="24" y="30" width="196" height="96" rx="4" fill="currentColor" fill-opacity="0.04"/>
    <rect x="40" y="70" width="30" height="22" rx="2" fill="currentColor" fill-opacity="0.22" stroke="none"/>
    <rect x="96" y="46" width="26" height="18" rx="2" fill="currentColor" fill-opacity="0.15" stroke="none"/>
    <rect x="130" y="46" width="26" height="18" rx="2" fill="currentColor" fill-opacity="0.15" stroke="none"/>
    <rect x="164" y="46" width="26" height="18" rx="2" fill="currentColor" fill-opacity="0.15" stroke="none"/>
    <rect x="96" y="94" width="26" height="18" rx="2" fill="currentColor" fill-opacity="0.15" stroke="none"/>
    <rect x="130" y="94" width="26" height="18" rx="2" fill="currentColor" fill-opacity="0.15" stroke="none"/>
    <rect x="164" y="94" width="26" height="18" rx="2" fill="currentColor" fill-opacity="0.15" stroke="none"/>
    <path d="M70 74 H96 M70 82 H96 M70 90 H130 M122 55 H96 M122 103 H96" stroke-width="0.8"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8">
    <text x="55" y="84" text-anchor="middle" font-size="7.5">ctrl</text>
    <text x="122" y="22" text-anchor="middle" font-weight="600">SSD &#8212; NAND flash chips</text>
    <text x="122" y="142" text-anchor="middle" font-size="7.5" fill-opacity="0.9">any block reached electronically &#183; no moving parts</text>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <rect x="240" y="30" width="196" height="96" rx="4" fill="currentColor" fill-opacity="0.04"/>
    <circle cx="320" cy="78" r="34" fill="currentColor" fill-opacity="0.06"/>
    <circle cx="320" cy="78" r="6" fill="currentColor" fill-opacity="0.3" stroke="none"/>
    <line x1="410" y1="44" x2="332" y2="72" stroke-width="1.4"/>
    <circle cx="410" cy="44" r="3" fill="currentColor" stroke="none"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8">
    <text x="338" y="22" text-anchor="middle" font-weight="600">HDD &#8212; spinning platter</text>
    <text x="338" y="142" text-anchor="middle" font-size="7.5" fill-opacity="0.9">seek arm must swing to the track &#183; slower random access</text>
  </g>
</svg>
<figcaption>An SSD is a controller wired to NAND flash chips, so any block is reached electronically; a hard disk must physically swing a seek arm to the right track on a spinning platter, which is why the SSD wins decisively on random access.</figcaption>
</figure>

## Overview

Where a [hard disk drive](/reference/hard-disk-drive/) seeks a head across spinning platters, an SSD reads any block electronically, so random access is orders of magnitude quicker and there is no mechanical latency, noise, or vibration sensitivity. A controller manages the underlying NAND, spreading writes across cells (wear leveling) because each flash cell endures only a limited number of erase cycles, and remapping cells that wear out. Endurance is often quoted as *TBW* — total bytes that can be written over the drive's life.

SSDs connect over the older SATA interface, which caps around 550 MB/s, or over [NVMe](/reference/nvme/) on a [PCI Express](/reference/pci-express/) link for several times that throughput and much lower latency. Either way the medium is the same NAND flash; the interface just sets how fast the host can reach it.

## SSD vs HDD

The two fill the same role but trade off very differently:

| Trait | SSD | HDD |
|-------|-----|-----|
| Moving parts | None | Platters + heads |
| Random access | Very fast | Slow (seek + rotate) |
| Sequential speed | High | Moderate |
| Cost per TB | Higher | Lower |
| Durability | Shock-resistant | Sensitive to shock |
| Wear limit | Erase cycles (TBW) | Mechanical wear |

## Where it fits

In the [memory hierarchy](/reference/memory-hierarchy/) an SSD sits between [RAM](/reference/random-access-memory/) and slow bulk disk: not as fast as memory, but vastly faster than an HDD for the random reads a [file system](/reference/file-system/) and database generate. For GopherTrunk, an SSD is the natural home for the active database of decoded calls and recent recordings — its quick random writes keep up with a busy control channel — while cheaper [HDD](/reference/hard-disk-drive/) capacity holds the long-term archive.

## Sources

[^wiki]: [Solid-state drive](https://en.wikipedia.org/wiki/Solid-state_drive) — Wikipedia, on flash-based drives and how they differ from mechanical disks.
