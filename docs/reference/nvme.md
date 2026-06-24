---
slug: nvme
title: NVMe
entry_type: concept
category: hw-storage
description: NVMe is a high-speed protocol for accessing solid-state storage directly over a PCI Express link, replacing older disk interfaces designed for slow mechanical drives.
keywords: NVMe, Non-Volatile Memory Express, PCIe storage, M.2, NVMe SSD, queue depth, storage protocol
aka: [Non-Volatile Memory Express]
autolink: true
infobox:
  - { label: Type, value: Storage interface protocol }
  - { label: Runs over, value: PCI Express }
  - { label: Replaces, value: AHCI / SATA for SSDs }
  - { label: Form factors, value: M.2, U.2, add-in card }
  - { label: Strength, value: Low latency, deep parallelism }
see_also: [solid-state-drive, flash-memory, pci-express, hard-disk-drive, data-storage, memory-hierarchy]
cite_urls:
  - https://en.wikipedia.org/wiki/NVM_Express
---

**NVMe (Non-Volatile Memory Express)** is a protocol for talking to fast solid-state storage directly over a [PCI Express](/reference/pci-express/) link, designed from the start for flash rather than spinning disks.[^wiki]

## Overview

Older interfaces such as SATA used the AHCI command set, which assumed the slow seek times of a [hard disk drive](/reference/hard-disk-drive/) and offered a single shallow command queue. NVMe instead exposes many deep queues that a [flash](/reference/flash-memory/)-based [SSD](/reference/solid-state-drive/) can service in parallel, with far lower per-command overhead. Drives commonly appear as an M.2 stick plugged straight into the board's PCIe lanes, so the [CPU](/reference/central-processing-unit/) reaches the flash without a legacy disk controller in the path.

## What it's for

NVMe exists to stop the interface from being the bottleneck once the medium is flash. A SATA SSD tops out around 550 MB/s; an NVMe drive on a few PCIe lanes runs several times that, with much lower latency under load. For a GopherTrunk node decoding many talkgroups at once, an NVMe SSD ensures that writing decoded frames and IQ snapshots never stalls the pipeline — though for a simple capture node a plain SD card or SATA SSD is usually plenty.

## Sources

[^wiki]: [NVM Express](https://en.wikipedia.org/wiki/NVM_Express) — Wikipedia, on the NVMe storage protocol and its advantages over AHCI/SATA.
