---
slug: nvme
title: NVMe
entry_type: concept
category: hw-storage
description: NVMe is a high-speed protocol for accessing solid-state storage directly over a PCI Express link, replacing older disk interfaces designed for slow mechanical drives.
keywords: NVMe, Non-Volatile Memory Express, PCIe storage, M.2, NVMe SSD, queue depth, storage protocol, AHCI, SATA
aka: [Non-Volatile Memory Express]
autolink: true
infobox:
  - { label: Type, value: Storage interface protocol }
  - { label: Runs over, value: PCI Express }
  - { label: Replaces, value: AHCI / SATA for SSDs }
  - { label: Form factors, value: M.2, U.2, add-in card }
  - { label: Strength, value: Low latency, deep parallelism }
see_also: [solid-state-drive, flash-memory, pci-express, hard-disk-drive, data-storage, memory-hierarchy, emmc]
cite_urls:
  - https://en.wikipedia.org/wiki/NVM_Express
---

**NVMe (Non-Volatile Memory Express)** is a protocol for talking to fast solid-state storage directly over a [PCI Express](/reference/pci-express/) link, designed from the start for flash rather than spinning disks.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Bar chart comparing peak sequential bandwidth of three storage interfaces. SATA III tops out near 550 megabytes per second, NVMe over four lanes of PCIe generation 3 reaches roughly 3500, and NVMe over four lanes of PCIe generation 4 reaches roughly 7000, drawn to scale as horizontal bars." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor">
    <line x1="118" y1="20" x2="118" y2="122" stroke-width="1.2"/>
    <rect x="118" y="30" width="31" height="18" fill-opacity="0.2" stroke="none"/>
    <rect x="118" y="66" width="200" height="18" fill-opacity="0.32" stroke="none"/>
    <rect x="118" y="102" width="400" height="18" fill-opacity="0.5" stroke="none"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8.5">
    <text x="112" y="43" text-anchor="end">SATA III</text>
    <text x="112" y="79" text-anchor="end">NVMe PCIe 3.0 x4</text>
    <text x="112" y="115" text-anchor="end">NVMe PCIe 4.0 x4</text>
  </g>
  <g fill="currentColor" stroke="none" font-size="8" font-family="ui-monospace, monospace">
    <text x="154" y="43">~550 MB/s</text>
    <text x="323" y="79">~3500 MB/s</text>
    <text x="405" y="115" text-anchor="end" fill-opacity="0.85">~7000 MB/s</text>
  </g>
  <text x="118" y="140" font-size="8" fill="currentColor" fill-opacity="0.9">peak sequential read — bars drawn to scale; the interface, not the flash, is the ceiling</text>
</svg>
<figcaption>SATA was built for mechanical disks and caps near 550 MB/s; NVMe rides PCI Express lanes directly, so bandwidth scales with lane count and generation instead of a legacy disk bus.</figcaption>
</figure>

## Overview

Older interfaces such as SATA used the AHCI command set, which assumed the slow seek times of a [hard disk drive](/reference/hard-disk-drive/) and offered a single shallow command queue. NVMe instead exposes thousands of deep queues that a [flash](/reference/flash-memory/)-based [SSD](/reference/solid-state-drive/) can service in parallel, with far lower per-command overhead and no translation to a disk-era protocol. Drives commonly appear as an M.2 stick plugged straight into the board's PCIe lanes, so the [CPU](/reference/central-processing-unit/) reaches the flash without a legacy disk controller in the path.

Because it rides PCI Express, NVMe's ceiling rises with the link: a four-lane PCIe 3.0 slot delivers several gigabytes per second, and PCIe 4.0 and 5.0 roughly double it each generation. Latency drops too, since a command need not wait behind others in a one-deep queue. That parallelism is exactly what a busy SSD wants, since its many NAND chips can be read and written at once.

## SATA vs NVMe

The two interfaces target the same flash but differ in everything around it:

| Aspect | SATA (AHCI) | NVMe |
|--------|-------------|------|
| Designed for | Mechanical disks | Flash / solid state |
| Physical link | SATA cable | PCI Express lanes |
| Command queues | 1 queue, 32 deep | 64K queues, 64K deep |
| Peak bandwidth | ~550 MB/s | Several GB/s and up |
| Latency overhead | Higher (legacy stack) | Lower |

## Where it fits

NVMe exists to stop the interface from being the bottleneck once the medium is flash. For a GopherTrunk node decoding many talkgroups at once, an NVMe SSD ensures that writing decoded frames and high-rate IQ snapshots never stalls the pipeline — sustained multi-gigabyte-per-second capture files land without back-pressure. For a simple single-channel capture node, though, a plain SD card, [eMMC](/reference/emmc/), or SATA SSD is usually plenty, and NVMe is worth the cost mainly when the write rate genuinely climbs.

## Sources

[^wiki]: [NVM Express](https://en.wikipedia.org/wiki/NVM_Express) — Wikipedia, on the NVMe storage protocol and its advantages over AHCI/SATA.
