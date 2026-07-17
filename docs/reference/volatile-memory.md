---
slug: volatile-memory
title: Volatile vs non-volatile memory
entry_type: concept
category: hw-storage
description: Volatile memory loses its contents when power is removed, while non-volatile memory retains them; the distinction explains why computers separate fast working memory from persistent storage.
keywords: volatile memory, non-volatile memory, RAM, ROM, flash, persistence, data retention, DRAM, power-off
aka: [non-volatile memory]
infobox:
  - { label: Volatile, value: Loses data without power (RAM) }
  - { label: Non-volatile, value: Retains data (flash, ROM, disk) }
  - { label: Volatile trait, value: Fast, used as working memory }
  - { label: Non-volatile trait, value: Persistent storage }
  - { label: Key question, value: "Does it survive power-off?" }
see_also: [random-access-memory, read-only-memory, flash-memory, memory-hierarchy, data-storage, cache-memory, solid-state-drive]
cite_urls:
  - https://en.wikipedia.org/wiki/Volatile_memory
  - https://en.wikipedia.org/wiki/Non-volatile_memory
---

**Volatile memory** loses its contents the moment power is removed, while **non-volatile memory** keeps them — a distinction that shapes how every computer divides fast working memory from lasting storage.[^vol][^nonvol]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 168" role="img" aria-label="Two panels showing what survives power-off. With power on, both a RAM chip and a flash chip hold their data. After power is removed, the RAM chip is blank because it is volatile, while the flash chip still holds the same data because it is non-volatile." xmlns="http://www.w3.org/2000/svg">
  <text x="115" y="22" font-size="9" text-anchor="middle" fill="currentColor" font-weight="600">power ON</text>
  <text x="345" y="22" font-size="9" text-anchor="middle" fill="currentColor" font-weight="600">power OFF</text>
  <line x1="230" y1="30" x2="230" y2="150" stroke="currentColor" stroke-width="0.8" stroke-dasharray="3 3" stroke-opacity="0.5"/>
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <rect x="40" y="40" width="150" height="42" rx="3" fill="currentColor" fill-opacity="0.05"/>
    <rect x="40" y="98" width="150" height="42" rx="3" fill="currentColor" fill-opacity="0.05"/>
    <rect x="270" y="40" width="150" height="42" rx="3" fill="currentColor" fill-opacity="0.05"/>
    <rect x="270" y="98" width="150" height="42" rx="3" fill="currentColor" fill-opacity="0.05"/>
  </g>
  <g fill="currentColor" stroke="none" font-family="ui-monospace, monospace" font-size="9">
    <text x="52" y="66">RAM  1011 0110</text>
    <text x="52" y="124">flash 1011 0110</text>
    <text x="282" y="66" fill-opacity="0.35">RAM  &#8212;&#8212;&#8212;&#8212; &#8212;&#8212;&#8212;&#8212;</text>
    <text x="282" y="124">flash 1011 0110</text>
  </g>
  <g fill="currentColor" stroke="none" font-size="7.5">
    <text x="115" y="94" text-anchor="middle" fill-opacity="0.85">volatile: needs power to hold bits</text>
    <text x="115" y="152" text-anchor="middle" fill-opacity="0.85">non-volatile: bits stay put</text>
    <text x="345" y="94" text-anchor="middle" fill-opacity="0.85">contents lost</text>
    <text x="345" y="152" text-anchor="middle" fill-opacity="0.85">contents retained</text>
  </g>
</svg>
<figcaption>Cut the power and volatile RAM forgets everything, while non-volatile flash keeps the same bits — the single test that sorts working memory from persistent storage.</figcaption>
</figure>

## Overview

[RAM](/reference/random-access-memory/) is the archetypal volatile memory: it is fast and freely rewritable, which makes it ideal as the place a running program keeps its data, but everything in it vanishes at power-off. Dynamic RAM even needs constant refreshing to hold its charge while powered. Non-volatile memory — [flash](/reference/flash-memory/), [ROM](/reference/read-only-memory/), hard disks, tape — gives up some speed or flexibility in exchange for retaining data without power.

[Cache memory](/reference/cache-memory/) is also volatile, as are the CPU's registers. The trade-off is fundamental: the fastest memories tend to be volatile, and persistence tends to cost speed or write endurance. That is why systems layer them rather than picking one.

## Volatile vs non-volatile

The dividing line is simple, but it drives where each technology is used:

| Trait | Volatile | Non-volatile |
|-------|----------|--------------|
| Survives power-off | No | Yes |
| Examples | DRAM, SRAM, cache | Flash, ROM, disk, tape |
| Typical speed | Fastest | Slower |
| Typical role | Working memory | Persistent storage |
| Rewrites | Effectively unlimited | Often limited (flash wear) |

## Where it fits

This split is why a computer cannot simply use one kind of memory for everything. Volatile RAM holds the working state of the [operating system](/reference/operating-system/) and applications; non-volatile storage holds files and programs between sessions, which is why work must be *saved* to disk to survive a reboot. The [memory hierarchy](/reference/memory-hierarchy/) arranges these layers by speed and cost. For GopherTrunk, decoded frames live briefly in volatile RAM as they are processed, then are written to non-volatile storage so the log survives a restart or power cut at the antenna site.

## Sources

[^vol]: [Volatile memory](https://en.wikipedia.org/wiki/Volatile_memory) — Wikipedia, on memory that requires power to retain data.
[^nonvol]: [Non-volatile memory](https://en.wikipedia.org/wiki/Non-volatile_memory) — Wikipedia, on memory that retains data without power.
