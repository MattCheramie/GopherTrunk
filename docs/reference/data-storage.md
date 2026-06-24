---
slug: data-storage
title: Storage (SSD, HDD, flash)
entry_type: hardware
category: hw-foundations
description: Storage is where a computer keeps data long-term so it survives a power cycle, using technologies such as hard drives, solid-state drives, and flash memory.
keywords: storage, SSD, HDD, hard drive, flash memory, SD card, non-volatile, permanent storage
aka: [storage, data storage]
infobox:
  - { label: Type, value: Long-term storage }
  - { label: Survives power loss, value: Yes (non-volatile) }
  - { label: Forms, value: HDD, SSD, flash }
  - { label: Speed, value: Slower than RAM }
see_also: [computer-hardware, random-access-memory, central-processing-unit, single-board-computer, microcontroller]
related_lessons:
  - { title: "The building blocks", url: /learn/intro-hardware/building-blocks/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Computer_data_storage
---

**Storage** is where a computer keeps data long-term, so that files and programs survive a power cycle.[^wiki] It is the permanent counterpart to [RAM](/reference/random-access-memory/).

## Overview
Storage comes in a few common forms:

- **HDD** (hard disk drive): data on spinning magnetic platters — cheap and high-capacity, but slower and mechanically fragile.
- **SSD** (solid-state drive): flash memory with no moving parts — faster, quieter, and more rugged.
- **Flash memory**: the non-volatile technology inside SSDs, SD cards, and the on-board storage of small devices.

All of these are **non-volatile** — they hold their contents with the power off, unlike RAM. The trade-off is speed: even fast storage is much slower than RAM, which is why the [CPU](/reference/central-processing-unit/) does its active work in memory and reads from or writes to storage only when it must.

## Where it fits
Storage is one of the four building blocks of [computer hardware](/reference/computer-hardware/). On a [single-board computer](/reference/single-board-computer/) it is often a microSD card; on a [microcontroller](/reference/microcontroller/) it may be a small block of on-chip flash; on a [server](/reference/server/) it can be racks of SSDs.

## Sources
[^wiki]: [Computer data storage](https://en.wikipedia.org/wiki/Computer_data_storage) — Wikipedia, on non-volatile storage and HDD/SSD/flash technologies.
