---
slug: volatile-memory
title: Volatile vs non-volatile memory
entry_type: concept
category: hw-storage
description: Volatile memory loses its contents when power is removed, while non-volatile memory retains them; the distinction explains why computers separate fast working memory from persistent storage.
keywords: volatile memory, non-volatile memory, RAM, ROM, flash, persistence, data retention, DRAM
aka: [non-volatile memory]
infobox:
  - { label: Volatile, value: Loses data without power (RAM) }
  - { label: Non-volatile, value: Retains data (flash, ROM, disk) }
  - { label: Volatile trait, value: Fast, used as working memory }
  - { label: Non-volatile trait, value: Persistent storage }
  - { label: Key question, value: "Does it survive power-off?" }
see_also: [random-access-memory, read-only-memory, flash-memory, memory-hierarchy, data-storage, cache-memory]
cite_urls:
  - https://en.wikipedia.org/wiki/Volatile_memory
  - https://en.wikipedia.org/wiki/Non-volatile_memory
---

**Volatile memory** loses its contents the moment power is removed, while **non-volatile memory** keeps them — a distinction that shapes how every computer divides fast working memory from lasting storage.[^vol][^nonvol]

## Overview

[RAM](/reference/random-access-memory/) is the archetypal volatile memory: it is fast and freely rewritable, which makes it ideal as the place a running program keeps its data, but everything in it vanishes at power-off. Non-volatile memory — [flash](/reference/flash-memory/), [ROM](/reference/read-only-memory/), hard disks, tape — gives up some speed or flexibility in exchange for retaining data without power. [Cache memory](/reference/cache-memory/) is also volatile. The trade-off is fundamental: the fastest memories tend to be volatile, and persistence tends to cost speed or write endurance.

## Where it fits

This split is why a computer cannot simply use one kind of memory for everything. Volatile RAM holds the working state of the [operating system](/reference/operating-system/) and applications; non-volatile storage holds files and programs between sessions, which is why work must be *saved* to disk to survive a reboot. The [memory hierarchy](/reference/memory-hierarchy/) arranges these layers by speed and cost. For GopherTrunk, decoded frames live briefly in volatile RAM as they are processed, then are written to non-volatile storage so the log survives a restart or power cut at the antenna site.

## Sources

[^vol]: [Volatile memory](https://en.wikipedia.org/wiki/Volatile_memory) — Wikipedia, on memory that requires power to retain data.
[^nonvol]: [Non-volatile memory](https://en.wikipedia.org/wiki/Non-volatile_memory) — Wikipedia, on memory that retains data without power.
