---
slug: memory-hierarchy
title: Memory hierarchy
entry_type: concept
category: hw-storage
description: The memory hierarchy arranges a computer's storage in layers trading speed against capacity and cost, from fast registers and cache down to RAM, flash, disk, and tape.
keywords: memory hierarchy, cache, RAM, registers, storage tiers, latency, locality, speed vs capacity
infobox:
  - { label: Type, value: Storage organisation principle }
  - { label: Top (fast/small), value: Registers, cache }
  - { label: Middle, value: RAM, SSD }
  - { label: Bottom (slow/large), value: HDD, tape }
  - { label: Trade-off, value: Speed vs capacity vs cost }
see_also: [cache-memory, random-access-memory, solid-state-drive, hard-disk-drive, volatile-memory, data-storage]
related_lessons:
  - { title: "Building blocks", url: /learn/intro-hardware/building-blocks/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Memory_hierarchy
---

The **memory hierarchy** is the layered arrangement of a computer's storage, ordered so that each level trades speed for capacity and cost against the one below it.[^wiki]

## Overview

At the top sit the [CPU](/reference/central-processing-unit/) registers and [cache memory](/reference/cache-memory/): tiny, blisteringly fast, and expensive per byte. Below them is main memory, [RAM](/reference/random-access-memory/), which is larger but slower. Further down comes persistent storage — a [solid-state drive](/reference/solid-state-drive/), then a [hard disk drive](/reference/hard-disk-drive/), and finally archival [magnetic tape](/reference/magnetic-tape/) — each step larger and cheaper but slower to reach. The hierarchy works because programs exhibit *locality*: they tend to reuse the same data and nearby data, so keeping hot items in the fast upper levels gives most of the speed of fast memory at most of the cost of slow storage.

## Where it fits

The hierarchy is the unifying framework for the whole storage-and-memory category: every device here is really a point on this speed-versus-capacity curve, with the [volatile/non-volatile](/reference/volatile-memory/) split cutting across it. The [operating system](/reference/operating-system/) and hardware constantly shuttle data between levels — caching disk blocks in RAM, paging RAM to disk. GopherTrunk benefits the same way: hot decode state stays in RAM and cache, the working call database lives on fast [SSD](/reference/solid-state-drive/), and bulk captures sink to cheap disk or tape.

## Sources

[^wiki]: [Memory hierarchy](https://en.wikipedia.org/wiki/Memory_hierarchy) — Wikipedia, on the layered organisation of computer storage by speed and cost.
