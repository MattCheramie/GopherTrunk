---
slug: solid-state-drive
title: Solid-state drive (SSD)
entry_type: hardware
category: hw-storage
description: A solid-state drive stores data in non-volatile flash memory with no moving parts, giving much faster access and better durability than a spinning hard disk.
keywords: solid-state drive, SSD, flash storage, NAND, SATA SSD, NVMe SSD, wear leveling, TBW
aka: [SSD]
infobox:
  - { label: Type, value: Flash non-volatile storage }
  - { label: Medium, value: NAND flash memory }
  - { label: Interfaces, value: SATA, NVMe (PCIe) }
  - { label: Moving parts, value: None }
  - { label: Strength, value: Fast random access }
see_also: [hard-disk-drive, flash-memory, nvme, data-storage, memory-hierarchy, file-system]
cite_urls:
  - https://en.wikipedia.org/wiki/Solid-state_drive
---

A **solid-state drive (SSD)** is a storage device that keeps data in non-volatile [flash memory](/reference/flash-memory/) with no moving parts, delivering far faster access than a mechanical disk.[^wiki]

## Overview

Where a [hard disk drive](/reference/hard-disk-drive/) seeks a head across spinning platters, an SSD reads any block electronically, so random access is orders of magnitude quicker and there is no mechanical latency or noise. A controller manages the underlying NAND, spreading writes across cells (wear leveling) because flash cells endure a limited number of erase cycles. SSDs connect over the older SATA interface or, for much higher throughput, over [NVMe](/reference/nvme/) on a [PCI Express](/reference/pci-express/) link.

## Where it fits

In the [memory hierarchy](/reference/memory-hierarchy/) an SSD sits between [RAM](/reference/random-access-memory/) and slow bulk disk: not as fast as memory, but vastly faster than an HDD for the random reads a [file system](/reference/file-system/) and database generate. For GopherTrunk, an SSD is the natural home for the active database of decoded calls and recent recordings — its quick random writes keep up with a busy control channel — while cheaper [HDD](/reference/hard-disk-drive/) capacity holds the long-term archive.

## Sources

[^wiki]: [Solid-state drive](https://en.wikipedia.org/wiki/Solid-state_drive) — Wikipedia, on flash-based drives and how they differ from mechanical disks.
